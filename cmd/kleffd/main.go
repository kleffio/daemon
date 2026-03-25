package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/kleffio/gameserver-daemon/internal/adapters/out/db"
	"github.com/kleffio/gameserver-daemon/internal/adapters/out/observability/logging"
	"github.com/kleffio/gameserver-daemon/internal/adapters/out/queue"
	"github.com/kleffio/gameserver-daemon/internal/adapters/out/repository/memory"
	k8sruntime "github.com/kleffio/gameserver-daemon/internal/adapters/out/runtime/kubernetes"
	"github.com/kleffio/gameserver-daemon/internal/app/config"
	"github.com/kleffio/gameserver-daemon/internal/application/ports"
	"github.com/kleffio/gameserver-daemon/internal/workers"
	"github.com/kleffio/gameserver-daemon/internal/workers/jobs"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to start daemon: %v", err)
	}

	baseLogger := logging.NewSlogAdapter()
	daemonLog := baseLogger.With(ports.LogKeyNodeID, cfg.NodeID)

	daemonLog.Info("Daemon starting", "runtime_mode", cfg.RuntimeMode, "queue_backend", cfg.QueueBackend)

	sqliteDB, err := db.InitDB(cfg.DatabasePath)
	if err != nil {
		daemonLog.Error("Failed to initialize database", err, "path", cfg.DatabasePath)
		os.Exit(1)
	}
	defer sqliteDB.Close()
	daemonLog.Info("Database initialized", "path", cfg.DatabasePath)

	// ── Queue ────────────────────────────────────────────────────────────────
	var q ports.Queue
	switch cfg.QueueBackend {
	case config.QueueBackendRedis:
		rq, err := queue.NewRedisQueue(cfg.RedisURL, cfg.RedisPassword, cfg.RedisTLS)
		if err != nil {
			daemonLog.Error("Failed to connect to Redis", err, "url", cfg.RedisURL)
			os.Exit(1)
		}
		daemonLog.Info("Connected to Redis queue", "url", cfg.RedisURL)
		q = rq
	default:
		daemonLog.Info("Using in-memory queue")
		q = queue.NewMemoryQueue()
	}

	// ── Container runtime ────────────────────────────────────────────────────
	var runtime ports.ContainerRuntime
	switch cfg.RuntimeMode {
	case config.RuntimeModeKubernetes:
		rt, err := k8sruntime.New(cfg.KubeConfig, cfg.KubeNS, cfg.NodeID)
		if err != nil {
			daemonLog.Error("Failed to initialize Kubernetes runtime", err)
			os.Exit(1)
		}
		daemonLog.Info("Kubernetes runtime initialized", "namespace", cfg.KubeNS)
		runtime = rt
	default:
		daemonLog.Error("Unsupported runtime mode", nil, "mode", cfg.RuntimeMode)
		os.Exit(1)
	}

	// ── Repository ───────────────────────────────────────────────────────────
	// TODO: replace with a SQLite-backed repository once the adapter is implemented.
	repo := memory.NewServerRepository()

	// ── Workers + Dispatcher ─────────────────────────────────────────────────
	dispatcher := workers.NewDispatcher(q, 4)
	dispatcher.Register(jobs.JobTypeServerProvision, workers.NewProvisionWorker(runtime, repo, daemonLog).Handle)
	dispatcher.Register(jobs.JobTypeServerStart, workers.NewStartWorker(runtime, repo, daemonLog).Handle)
	dispatcher.Register(jobs.JobTypeServerStop, workers.NewStopWorker(runtime, repo, daemonLog).Handle)
	dispatcher.Register(jobs.JobTypeServerDelete, workers.NewDeleteWorker(runtime, repo, daemonLog).Handle)
	dispatcher.Register(jobs.JobTypeServerRestart, workers.NewRestartWorker(runtime, repo, daemonLog).Handle)

	// ── Graceful shutdown ────────────────────────────────────────────────────
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		daemonLog.Info("Shutdown signal received", "signal", sig.String())
		cancel()
	}()

	daemonLog.Info("Dispatcher running", "concurrency", 4)
	dispatcher.Run(ctx)
	daemonLog.Info("Daemon stopped")
}
