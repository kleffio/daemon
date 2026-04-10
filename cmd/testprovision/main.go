package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	k8sadapter "github.com/kleffio/kleff-daemon/internal/adapters/out/runtime/kubernetes"
	"github.com/kleffio/kleff-daemon/internal/application/ports"
)

// testprovision provisions a PaperMC Minecraft server via the Kubernetes adapter
// and prints the result. It expects kubectl proxy running on localhost:8888.
//
// Usage (on the cluster node):
//
//	kubectl proxy --port=8888 &
//	./testprovision
//
// To clean up afterwards:
//
//	./testprovision cleanup
func main() {
	const (
		proxyURL    = "http://localhost:8888"
		namespace   = "default"
		nodeID      = "test-node"
		serverID    = "kleff-test-papermc"
		ownerID     = "test-owner"
		blueprintID = "minecraft-vanilla"
	)

	adapter, err := k8sadapter.New(proxyURL, namespace, nodeID)
	if err != nil {
		log.Fatalf("failed to create kubernetes adapter: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	// Cleanup mode — remove the GameServer if it exists.
	if len(os.Args) > 1 && os.Args[1] == "cleanup" {
		fmt.Printf("Removing %s...\n", serverID)
		if err := adapter.Remove(ctx, serverID); err != nil {
			log.Fatalf("remove failed: %v", err)
		}
		fmt.Println("Removed.")
		return
	}

	// Build the WorkloadSpec from the minecraft-papermc blueprint/construct.
	spec := ports.WorkloadSpec{
		OwnerID:     ownerID,
		ServerID:    serverID,
		BlueprintID: blueprintID,
		Image:       "itzg/minecraft-server",

		// From blueprint.json resources
		MemoryBytes:   2048 * 1024 * 1024, // 2048 MB
		CPUMillicores: 500,                // 0.5 vCPU (test only)

		// From construct.json env + user config defaults
		EnvOverrides: map[string]string{
			"EULA":        "TRUE",
			"TYPE":        "VANILLA",
			"VERSION":     "LATEST",
			"DIFFICULTY":  "normal",
			"MAX_PLAYERS": "20",
			"MEMORY":      "2G",
		},

		// From construct.json ports
		PortRequirements: []ports.PortRequirement{
			{TargetPort: 25565, Protocol: "tcp"},
			{TargetPort: 25565, Protocol: "udp"},
			{TargetPort: 25575, Protocol: "tcp"}, // RCON
		},

		// From construct.json runtime_hints
		RuntimeHints: ports.RuntimeHints{
			KubernetesStrategy: "agones",
			ExposeUDP:          true,
		},
	}

	fmt.Printf("Provisioning %s (%s) via Agones...\n", serverID, spec.Image)
	fmt.Printf("  Memory: %d MB  CPU: %dm\n", spec.MemoryBytes/1024/1024, spec.CPUMillicores)
	fmt.Printf("  Ports: 25565/tcp, 25565/udp, 25575/tcp\n")
	fmt.Println("  Waiting for GameServer to become Ready (up to 10 min)...")

	start := time.Now()
	server, err := adapter.Deploy(ctx, spec)
	if err != nil {
		log.Fatalf("deploy failed: %v", err)
	}

	fmt.Printf("\nServer is ready! (took %s)\n", time.Since(start).Round(time.Second))
	fmt.Printf("  RuntimeRef : %s\n", server.RuntimeRef)
	fmt.Printf("  State      : %s\n", server.State)
	fmt.Printf("  NodeID     : %s\n", server.Labels.NodeID)
	fmt.Printf("  ServerID   : %s\n", server.Labels.ServerID)

	endpoint, err := adapter.Endpoint(ctx, serverID)
	if err != nil {
		fmt.Printf("  Endpoint   : (could not resolve: %v)\n", err)
	} else {
		fmt.Printf("  Endpoint   : %s\n", endpoint)
	}

	fmt.Printf("\nTo clean up: ./testprovision cleanup\n")
}
