package main

import (
	"fmt"
	"log"
	
	"github.com/kleffio/gameserver-daemon/internal/app/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to start daemon: %v", err)
	}

	fmt.Printf("Daemon starting!\n")
	fmt.Printf("Node ID: %s\n", cfg.NodeID)
	fmt.Printf("Runtime Mode: %s\n", cfg.RuntimeMode)
	
	// Next we would initialize the "adapters/out/runtime" based on cfg.RuntimeMode...
}
