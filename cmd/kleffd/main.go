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

	// Initialize the structural logger
	baseLogger := logging.NewSlogAdapter()
	
	// Globally inject the node ID into all subsequent logs via With()
	daemonLog := baseLogger.With(ports.LogKeyNodeID, cfg.NodeID)

	daemonLog.Info("Daemon starting", "runtime_mode", cfg.RuntimeMode)

	// Initialize the Database
	sqliteDB, err := db.InitDB(cfg.DatabasePath)
	if err != nil {
		daemonLog.Error("Failed to initialize database", err, "path", cfg.DatabasePath)
		os.Exit(1)
	}
	defer sqliteDB.Close()
	daemonLog.Info("Database initialized successfully", "path", cfg.DatabasePath)
	
	// Next we would initialize the "adapters/out/runtime" based on cfg.RuntimeMode...
}
