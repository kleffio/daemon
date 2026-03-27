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

// setRequiredKleffVars sets the two new required KLEFF_* vars that every
// test must supply unless it is specifically testing their absence.
func setRequiredKleffVars(t *testing.T) {
	t.Helper()
	os.Setenv("KLEFF_PLATFORM_URL", "http://platform.test")
	os.Setenv("KLEFF_SHARED_SECRET", "test-secret")
	t.Cleanup(func() {
		os.Unsetenv("KLEFF_PLATFORM_URL")
		os.Unsetenv("KLEFF_SHARED_SECRET")
	})
}

func TestConfigLoadsCorrectlyDefaults(t *testing.T) {
	resetViperAndFlags()

	os.Unsetenv("KLEFF_RUNTIME_MODE")
	os.Unsetenv("KLEFF_CLUSTER_REGION")

	os.Setenv("KLEFF_NODE_ID", "default-node")
	defer os.Unsetenv("KLEFF_NODE_ID")
	setRequiredKleffVars(t)

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

	os.Setenv("KLEFF_NODE_ID", "env-node")
	os.Setenv("KLEFF_RUNTIME_MODE", "kubernetes")
	os.Setenv("KLEFF_QUEUE_BACKEND", "redis")
	defer func() {
		os.Unsetenv("KLEFF_NODE_ID")
		os.Unsetenv("KLEFF_RUNTIME_MODE")
		os.Unsetenv("KLEFF_QUEUE_BACKEND")
	}()
	setRequiredKleffVars(t)

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

	os.Setenv("KLEFF_RUNTIME_MODE", "docker")
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
	os.Setenv("KLEFF_RUNTIME_MODE", "invalid-runtime")
	os.Setenv("KLEFF_PLATFORM_URL", "http://platform.test")
	os.Setenv("KLEFF_SHARED_SECRET", "test-secret")
	_, err = Load()
	if err == nil {
		t.Errorf("Expected validation to fail for invalid runtime.mode")
	}

	resetViperAndFlags()
	os.Setenv("KLEFF_NODE_ID", "valid-node")
	os.Setenv("KLEFF_RUNTIME_MODE", "docker")
	os.Setenv("KLEFF_QUEUE_BACKEND", "invalid-queue")
	os.Setenv("KLEFF_PLATFORM_URL", "http://platform.test")
	os.Setenv("KLEFF_SHARED_SECRET", "test-secret")
	defer func() {
		os.Unsetenv("KLEFF_NODE_ID")
		os.Unsetenv("KLEFF_RUNTIME_MODE")
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

	os.Setenv("KLEFF_RUNTIME_MODE", "kubernetes")
	os.Setenv("KLEFF_NODE_ID", "env-node")
	defer func() {
		os.Unsetenv("KLEFF_RUNTIME_MODE")
		os.Unsetenv("KLEFF_NODE_ID")
	}()
	setRequiredKleffVars(t)

	os.Args = []string{"cmd", "--node.id=flag-node"}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

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

// --- New tests for ticket #98 ---

func TestConfigGSDDeprecationShimCopiesValues(t *testing.T) {
	resetViperAndFlags()
	os.Args = []string{"cmd"}

	os.Setenv("GSD_NODE_ID", "shim-node")
	os.Setenv("GSD_RUNTIME_MODE", "docker")
	os.Setenv("GSD_QUEUE_BACKEND", "memory")
	os.Setenv("GSD_PLATFORM_URL", "http://shim-platform.test")
	os.Setenv("GSD_SHARED_SECRET", "shim-secret")
	defer func() {
		os.Unsetenv("GSD_NODE_ID")
		os.Unsetenv("GSD_RUNTIME_MODE")
		os.Unsetenv("GSD_QUEUE_BACKEND")
		os.Unsetenv("GSD_PLATFORM_URL")
		os.Unsetenv("GSD_SHARED_SECRET")
		// shim sets these; clean up
		os.Unsetenv("KLEFF_NODE_ID")
		os.Unsetenv("KLEFF_RUNTIME_MODE")
		os.Unsetenv("KLEFF_QUEUE_BACKEND")
		os.Unsetenv("KLEFF_PLATFORM_URL")
		os.Unsetenv("KLEFF_SHARED_SECRET")
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Expected GSD_* shim to allow agent to start, got error: %v", err)
	}

	if cfg.NodeID != "shim-node" {
		t.Errorf("Expected node.id 'shim-node' via GSD shim, got '%s'", cfg.NodeID)
	}
	if cfg.PlatformURL != "http://shim-platform.test" {
		t.Errorf("Expected platform.url 'http://shim-platform.test' via GSD shim, got '%s'", cfg.PlatformURL)
	}
	if cfg.SharedSecret != "shim-secret" {
		t.Errorf("Expected shared_secret 'shim-secret' via GSD shim, got '%s'", cfg.SharedSecret)
	}
}

func TestConfigGSDShimDoesNotOverrideKleffVars(t *testing.T) {
	resetViperAndFlags()
	os.Args = []string{"cmd"}

	os.Setenv("GSD_NODE_ID", "gsd-node")
	os.Setenv("KLEFF_NODE_ID", "kleff-node")
	os.Setenv("KLEFF_RUNTIME_MODE", "docker")
	os.Setenv("KLEFF_QUEUE_BACKEND", "memory")
	os.Setenv("KLEFF_PLATFORM_URL", "http://platform.test")
	os.Setenv("KLEFF_SHARED_SECRET", "test-secret")
	defer func() {
		os.Unsetenv("GSD_NODE_ID")
		os.Unsetenv("KLEFF_NODE_ID")
		os.Unsetenv("KLEFF_RUNTIME_MODE")
		os.Unsetenv("KLEFF_QUEUE_BACKEND")
		os.Unsetenv("KLEFF_PLATFORM_URL")
		os.Unsetenv("KLEFF_SHARED_SECRET")
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if cfg.NodeID != "kleff-node" {
		t.Errorf("Expected KLEFF_NODE_ID to take precedence over GSD_NODE_ID, got '%s'", cfg.NodeID)
	}
}

func TestConfigPlatformURLRequired(t *testing.T) {
	resetViperAndFlags()

	os.Setenv("KLEFF_NODE_ID", "test-node")
	os.Setenv("KLEFF_SHARED_SECRET", "test-secret")
	defer func() {
		os.Unsetenv("KLEFF_NODE_ID")
		os.Unsetenv("KLEFF_SHARED_SECRET")
	}()

	_, err := Load()
	if err == nil {
		t.Errorf("Expected Load to fail when KLEFF_PLATFORM_URL is unset")
	}
}

func TestConfigSharedSecretRequired(t *testing.T) {
	resetViperAndFlags()

	os.Setenv("KLEFF_NODE_ID", "test-node")
	os.Setenv("KLEFF_PLATFORM_URL", "http://platform.test")
	defer func() {
		os.Unsetenv("KLEFF_NODE_ID")
		os.Unsetenv("KLEFF_PLATFORM_URL")
	}()

	_, err := Load()
	if err == nil {
		t.Errorf("Expected Load to fail when KLEFF_SHARED_SECRET is unset")
	}
}

func TestConfigMetricsPortDefault(t *testing.T) {
	resetViperAndFlags()

	os.Setenv("KLEFF_NODE_ID", "test-node")
	defer os.Unsetenv("KLEFF_NODE_ID")
	setRequiredKleffVars(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if cfg.MetricsPort != 9090 {
		t.Errorf("Expected default KLEFF_METRICS_PORT to be 9090, got %d", cfg.MetricsPort)
	}
}

func TestConfigNewFieldsPopulated(t *testing.T) {
	resetViperAndFlags()

	os.Setenv("KLEFF_NODE_ID", "test-node")
	os.Setenv("KLEFF_PLATFORM_URL", "http://my-platform.internal")
	os.Setenv("KLEFF_SHARED_SECRET", "supersecret")
	defer func() {
		os.Unsetenv("KLEFF_NODE_ID")
		os.Unsetenv("KLEFF_PLATFORM_URL")
		os.Unsetenv("KLEFF_SHARED_SECRET")
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if cfg.PlatformURL != "http://my-platform.internal" {
		t.Errorf("Expected PlatformURL 'http://my-platform.internal', got '%s'", cfg.PlatformURL)
	}
	if cfg.SharedSecret != "supersecret" {
		t.Errorf("Expected SharedSecret 'supersecret', got '%s'", cfg.SharedSecret)
	}
}
