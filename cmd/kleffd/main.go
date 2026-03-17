package main

import (
	"log"
	"os"

	"github.com/kleffio/gameserver-daemon/internal/adapters/out/db"
	"github.com/kleffio/gameserver-daemon/internal/adapters/out/observability/logging"
	"github.com/kleffio/gameserver-daemon/internal/app/config"
	"github.com/kleffio/gameserver-daemon/internal/application/ports"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to start daemon: %v", err)
	}

	baseLogger := logging.NewSlogAdapter()
	
	daemonLog := baseLogger.With(ports.LogKeyNodeID, cfg.NodeID)

	daemonLog.Info("Daemon starting", "runtime_mode", cfg.RuntimeMode)

	sqliteDB, err := db.InitDB(cfg.DatabasePath)
	if err != nil {
		daemonLog.Error("Failed to initialize database", err, "path", cfg.DatabasePath)
		os.Exit(1)
	}
	defer sqliteDB.Close()
	daemonLog.Info("Database initialized successfully", "path", cfg.DatabasePath)
	
}
