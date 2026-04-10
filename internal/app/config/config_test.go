package config

import (
	"os"
	"testing"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

func resetViperAndFlags() {
	viper.Reset()
	pflag.CommandLine = pflag.NewFlagSet(os.Args[0], pflag.ExitOnError)
}

// setRequiredKleffVars sets the required KLEFF_* vars that every test must supply
// unless it is specifically testing their absence.
func setRequiredKleffVars(t *testing.T) {
	t.Helper()
	os.Setenv("KLEFF_PLATFORM_URL", "http://platform.test")
	os.Setenv("KLEFF_SHARED_SECRET", "test-secret")
	t.Cleanup(func() {
		os.Unsetenv("KLEFF_PLATFORM_URL")
		os.Unsetenv("KLEFF_SHARED_SECRET")
	})
}

func TestConfigLoadsDefaults(t *testing.T) {
	resetViperAndFlags()

	os.Setenv("KLEFF_NODE_ID", "default-node")
	defer os.Unsetenv("KLEFF_NODE_ID")
	setRequiredKleffVars(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if cfg.GRPCPort != 50051 {
		t.Errorf("Expected default gRPC port 50051, got %d", cfg.GRPCPort)
	}
	if cfg.QueueBackend != QueueBackendMemory {
		t.Errorf("Expected default queue backend 'memory', got '%s'", cfg.QueueBackend)
	}
	if cfg.KubeNamespace != "default" {
		t.Errorf("Expected default kube namespace 'default', got '%s'", cfg.KubeNamespace)
	}
}

func TestConfigQueueBackendViaEnv(t *testing.T) {
	resetViperAndFlags()

	os.Setenv("KLEFF_NODE_ID", "env-node")
	os.Setenv("KLEFF_QUEUE_BACKEND", "redis")
	defer func() {
		os.Unsetenv("KLEFF_NODE_ID")
		os.Unsetenv("KLEFF_QUEUE_BACKEND")
	}()
	setRequiredKleffVars(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if cfg.QueueBackend != QueueBackendRedis {
		t.Errorf("Expected queue backend 'redis', got '%s'", cfg.QueueBackend)
	}
}

func TestConfigNodesAndPortsViaEnv(t *testing.T) {
	resetViperAndFlags()

	os.Setenv("KLEFF_NODE_ID", "test-node")
	os.Setenv("KLEFF_GRPC_PORT", "9090")
	os.Setenv("KLEFF_METRICS_PORT", "8080")
	defer func() {
		os.Unsetenv("KLEFF_NODE_ID")
		os.Unsetenv("KLEFF_GRPC_PORT")
		os.Unsetenv("KLEFF_METRICS_PORT")
	}()
	setRequiredKleffVars(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if cfg.NodeID != "test-node" {
		t.Errorf("Expected node id 'test-node', got '%s'", cfg.NodeID)
	}
	if cfg.GRPCPort != 9090 {
		t.Errorf("Expected grpc port 9090, got %d", cfg.GRPCPort)
	}
	if cfg.MetricsPort != 8080 {
		t.Errorf("Expected metrics port 8080, got %d", cfg.MetricsPort)
	}
}

func TestConfigValidationFailsForInvalidInputs(t *testing.T) {
	resetViperAndFlags()

	os.Setenv("KLEFF_QUEUE_BACKEND", "memory")
	os.Setenv("KLEFF_PLATFORM_URL", "http://platform.test")
	os.Setenv("KLEFF_SHARED_SECRET", "test-secret")
	os.Args = []string{"cmd", "--node.id="}
	_, err := Load()
	if err == nil {
		t.Errorf("Expected validation to fail when node.id is missing")
	}

	resetViperAndFlags()
	os.Setenv("KLEFF_NODE_ID", "valid-node")
	os.Setenv("KLEFF_QUEUE_BACKEND", "invalid-queue")
	os.Setenv("KLEFF_PLATFORM_URL", "http://platform.test")
	os.Setenv("KLEFF_SHARED_SECRET", "test-secret")
	defer func() {
		os.Unsetenv("KLEFF_NODE_ID")
		os.Unsetenv("KLEFF_QUEUE_BACKEND")
		os.Unsetenv("KLEFF_PLATFORM_URL")
		os.Unsetenv("KLEFF_SHARED_SECRET")
	}()
	_, err = Load()
	if err == nil {
		t.Errorf("Expected validation to fail for invalid queue.backend")
	}
}

func TestConfigPrecedence(t *testing.T) {
	resetViperAndFlags()

	yamlContent := []byte(`
node:
  id: file-node
grpc:
  port: 10000
`)
	err := os.WriteFile("config.yaml", yamlContent, 0644)
	if err != nil {
		t.Fatalf("Failed to create mock config file: %v", err)
	}
	defer os.Remove("config.yaml")

	os.Setenv("KLEFF_NODE_ID", "env-node")
	defer os.Unsetenv("KLEFF_NODE_ID")
	setRequiredKleffVars(t)

	os.Args = []string{"cmd", "--node.id=flag-node"}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if cfg.NodeID != "flag-node" {
		t.Errorf("Expected node.id to be 'flag-node' (flag precedence), got '%s'", cfg.NodeID)
	}
	if cfg.GRPCPort != 10000 {
		t.Errorf("Expected grpc.port to be 10000 (file precedence over default), got %d", cfg.GRPCPort)
	}
}
