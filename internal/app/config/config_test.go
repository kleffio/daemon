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

func TestConfigLoadsCorrectlyDefaults(t *testing.T) {
	resetViperAndFlags()

	// Clear out any potential environment variables that could mess up the test
	os.Unsetenv("GSD_RUNTIME_MODE")
	os.Unsetenv("GSD_CLUSTER_REGION")
	
	// Set the required NodeID via env so validation passes
	os.Setenv("GSD_NODE_ID", "default-node")
	defer os.Unsetenv("GSD_NODE_ID")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if cfg.RuntimeMode != RuntimeModeDocker {
		t.Errorf("Expected default runtime mode 'docker', got '%s'", cfg.RuntimeMode)
	}
	if cfg.GRPCPort != 50051 {
		t.Errorf("Expected default gRPC port 50051, got %d", cfg.GRPCPort)
	}
	if cfg.QueueBackend != QueueBackendMemory {
		t.Errorf("Expected default queue backend 'memory', got '%s'", cfg.QueueBackend)
	}
}

func TestConfigRuntimeModeConfigurableViaEnv(t *testing.T) {
	resetViperAndFlags()

	// Set the environment variables
	os.Setenv("GSD_NODE_ID", "env-node")
	os.Setenv("GSD_RUNTIME_MODE", "kubernetes")
	os.Setenv("GSD_QUEUE_BACKEND", "redis")
	defer func() {
		os.Unsetenv("GSD_NODE_ID")
		os.Unsetenv("GSD_RUNTIME_MODE")
		os.Unsetenv("GSD_QUEUE_BACKEND")
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if cfg.RuntimeMode != RuntimeModeKubernetes {
		t.Errorf("Expected runtime mode 'kubernetes', got '%s'", cfg.RuntimeMode)
	}
	if cfg.QueueBackend != QueueBackendRedis {
		t.Errorf("Expected queue backend 'redis', got '%s'", cfg.QueueBackend)
	}
}

func TestConfigNodesAndPortsViaEnv(t *testing.T) {
	resetViperAndFlags()

	os.Setenv("GSD_NODE_ID", "test-node")
	os.Setenv("GSD_GRPC_PORT", "9090")
	os.Setenv("GSD_METRICS_PORT", "8080")
	defer func() {
		os.Unsetenv("GSD_NODE_ID")
		os.Unsetenv("GSD_GRPC_PORT")
		os.Unsetenv("GSD_METRICS_PORT")
	}()

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

	// 1. Missing Node ID
	os.Setenv("GSD_RUNTIME_MODE", "docker")
	os.Setenv("GSD_QUEUE_BACKEND", "memory")
	_, err := Load()
	if err == nil {
		t.Errorf("Expected validation to fail when node.id is missing")
	}

	// 2. Invalid Runtime Mode
	resetViperAndFlags()
	os.Setenv("GSD_NODE_ID", "valid-node")
	os.Setenv("GSD_RUNTIME_MODE", "invalid-runtime")
	_, err = Load()
	if err == nil {
		t.Errorf("Expected validation to fail for invalid runtime.mode")
	}

	// 3. Invalid Queue Backend
	resetViperAndFlags()
	os.Setenv("GSD_NODE_ID", "valid-node")
	os.Setenv("GSD_RUNTIME_MODE", "docker")
	os.Setenv("GSD_QUEUE_BACKEND", "invalid-queue")
	defer func() {
		os.Unsetenv("GSD_NODE_ID")
		os.Unsetenv("GSD_RUNTIME_MODE")
		os.Unsetenv("GSD_QUEUE_BACKEND")
	}()
	_, err = Load()
	if err == nil {
		t.Errorf("Expected validation to fail for invalid queue.backend")
	}
}

func TestConfigPrecedence(t *testing.T) {
	resetViperAndFlags()

	// 1. Create a temporary config file simulating /etc/gameserver-daemon/config.yaml or ./config.yaml
	yamlContent := []byte(`
runtime:
  mode: docker
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

	// 2. Set Env Var (Should override config file)
	os.Setenv("GSD_RUNTIME_MODE", "kubernetes") // Env beats file
	os.Setenv("GSD_NODE_ID", "env-node")        // Env beats file
	defer func() {
		os.Unsetenv("GSD_RUNTIME_MODE")
		os.Unsetenv("GSD_NODE_ID")
	}()

	// 3. Set CLI Flags (Should override everything)
	os.Args = []string{"cmd", "--node.id=flag-node"} // Flag beats env beats file

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Assertions based on standard precedence: Flag > Env > File > Default
	if cfg.NodeID != "flag-node" {
		t.Errorf("Expected node.id to be 'flag-node' (Flag precedence), got '%s'", cfg.NodeID)
	}
	if cfg.RuntimeMode != RuntimeModeKubernetes {
		t.Errorf("Expected runtime.mode to be 'kubernetes' (Env precedence over File), got '%s'", cfg.RuntimeMode)
	}
	if cfg.GRPCPort != 10000 {
		t.Errorf("Expected grpc.port to be 10000 (File precedence over Default), got %d", cfg.GRPCPort)
	}
}
