package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/kleffio/kleff-daemon/internal/adapters/out/db"
	"github.com/kleffio/kleff-daemon/internal/adapters/out/observability/logging"
	queueadapter "github.com/kleffio/kleff-daemon/internal/adapters/out/queue"
	memrepo "github.com/kleffio/kleff-daemon/internal/adapters/out/repository/memory"
	dockeradapter "github.com/kleffio/kleff-daemon/internal/adapters/out/runtime/docker"
	k8sadapter "github.com/kleffio/kleff-daemon/internal/adapters/out/runtime/kubernetes"
	"github.com/kleffio/kleff-daemon/internal/app/config"
	"github.com/kleffio/kleff-daemon/internal/application/ports"
	"github.com/kleffio/kleff-daemon/internal/workers"
	"github.com/kleffio/kleff-daemon/internal/workers/jobs"
	"k8s.io/client-go/rest"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to start daemon: %v", err)
	}

	baseLogger := logging.NewSlogAdapter()
	daemonLog := baseLogger.With(ports.LogKeyNodeID, cfg.NodeID)

	// --- Runtime adapter: auto-detect Docker vs Kubernetes ---
	runtime, err := detectRuntime(cfg, daemonLog)
	if err != nil {
		daemonLog.Error("Failed to initialize runtime adapter", err)
		os.Exit(1)
	}

	// --- Database ---
	sqliteDB, err := db.InitDB(cfg.DatabasePath)
	if err != nil {
		daemonLog.Error("Failed to initialize database", err, "path", cfg.DatabasePath)
		os.Exit(1)
	}
	defer sqliteDB.Close()
	daemonLog.Info("Database initialized", "path", cfg.DatabasePath)

	// --- Queue ---
	var q ports.Queue
	switch cfg.QueueBackend {
	case config.QueueBackendRedis:
		rq, err := queueadapter.NewRedisQueue(cfg.RedisURL, cfg.RedisPassword, cfg.RedisTLS)
		if err != nil {
			daemonLog.Error("Failed to initialize Redis queue", err)
			os.Exit(1)
		}
		q = rq
		daemonLog.Info("Queue backend: Redis", "url", cfg.RedisURL)
	default:
		q = queueadapter.NewMemoryQueue()
		daemonLog.Info("Queue backend: in-memory")
	}

	// --- Repository ---
	repo := memrepo.NewServerRepository()

	// --- Dispatcher + workers ---
	dispatcher := workers.NewDispatcher(q, 4)
	dispatcher.Register(jobs.JobTypeServerProvision, workers.NewProvisionWorker(runtime, repo, daemonLog).Handle)
	dispatcher.Register(jobs.JobTypeServerStart, workers.NewStartWorker(runtime, repo, daemonLog).Handle)
	dispatcher.Register(jobs.JobTypeServerStop, workers.NewStopWorker(runtime, repo, daemonLog).Handle)
	dispatcher.Register(jobs.JobTypeServerDelete, workers.NewDeleteWorker(runtime, repo, daemonLog).Handle)
	dispatcher.Register(jobs.JobTypeServerRestart, workers.NewRestartWorker(runtime, repo, daemonLog).Handle)

	daemonLog.Info("Daemon started", "node_id", cfg.NodeID, "grpc_port", cfg.GRPCPort)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	dispatcher.Run(ctx)
	daemonLog.Info("Daemon shutdown complete")
}

func detectRuntime(cfg *config.Config, logger ports.Logger) (ports.RuntimeAdapter, error) {
	// Explicit kubeconfig → always Kubernetes.
	if cfg.Kubeconfig != "" {
		adapter, err := k8sadapter.New(cfg.Kubeconfig, cfg.KubeNamespace, cfg.NodeID)
		if err != nil {
			return nil, err
		}
		logger.Info("Runtime: Kubernetes", "kubeconfig", cfg.Kubeconfig)
		return adapter, nil
	}

	// In-cluster environment detected → Kubernetes.
	if _, err := rest.InClusterConfig(); err == nil {
		adapter, err := k8sadapter.New("", cfg.KubeNamespace, cfg.NodeID)
		if err != nil {
			return nil, err
		}
		logger.Info("Runtime: Kubernetes (in-cluster)")
		return adapter, nil
	}

	// Fall back to Docker.
	adapter, err := dockeradapter.New(cfg.NodeID)
	if err != nil {
		return nil, err
	}
	logger.Info("Runtime: Docker")
	return adapter, nil
}
