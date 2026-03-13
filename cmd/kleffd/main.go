package main

import (
	"log"

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
	
	// Next we would initialize the "adapters/out/runtime" based on cfg.RuntimeMode...
}
