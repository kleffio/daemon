package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// RuntimeMode defines the environment where the daemon is being deployed.
type RuntimeMode string

const (
	RuntimeModeDocker     RuntimeMode = "docker"
	RuntimeModeKubernetes RuntimeMode = "kubernetes"
)

// QueueBackend defines the messaging system for distributed communication.
type QueueBackend string

const (
	QueueBackendMemory QueueBackend = "memory"
	QueueBackendRedis  QueueBackend = "redis"
)

// Config holds the application configuration
type Config struct {
	RuntimeMode   RuntimeMode  `mapstructure:"runtime.mode"`
	ClusterRegion string       `mapstructure:"cluster.region"`
	NodeID        string       `mapstructure:"node.id"`
	GRPCPort      int          `mapstructure:"grpc.port"`
	MetricsPort   int          `mapstructure:"metrics.port"`
	QueueBackend  QueueBackend `mapstructure:"queue.backend"`
}

// Validate ensures all loaded configuration variables are correct and complete before usage.
func (c *Config) Validate() error {
	switch c.RuntimeMode {
	case RuntimeModeDocker, RuntimeModeKubernetes:
		// valid
	default:
		return fmt.Errorf("invalid runtime.mode: %q (must be 'docker' or 'kubernetes')", c.RuntimeMode)
	}

	switch c.QueueBackend {
	case QueueBackendMemory, QueueBackendRedis:
		// valid
	default:
		return fmt.Errorf("invalid queue.backend: %q (must be 'memory' or 'redis')", c.QueueBackend)
	}

	if strings.TrimSpace(c.NodeID) == "" {
		return fmt.Errorf("node.id is required and cannot be empty")
	}

	return nil
}

// Load reads in config file and ENV variables if set.
func Load() (*Config, error) {
	v := viper.New()

	// Default Settings
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown-node"
	}

	v.SetDefault("runtime.mode", string(RuntimeModeDocker))
	v.SetDefault("cluster.region", "local")
	v.SetDefault("node.id", hostname)
	v.SetDefault("grpc.port", 50051)
	v.SetDefault("metrics.port", 9090)
	v.SetDefault("queue.backend", string(QueueBackendMemory))

	// Environment Variables
	v.SetEnvPrefix("gsd")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv() // read in environment variables that match

	// Config File
	v.SetConfigName("config") // name of config file (without extension)
	v.SetConfigType("yaml")   // REQUIRED if the config file does not have the extension in the name
	v.AddConfigPath("/etc/gameserver-daemon/")
	v.AddConfigPath(".")

	// Ignore if config file is not found, we rely heavily on env vars/flags
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			// Config file was found but another error was produced
			return nil, fmt.Errorf("fatal error config file: %w", err)
		}
	}

	// CLI Flags
	fs := pflag.NewFlagSet("gsd", pflag.ContinueOnError)
	fs.ParseErrorsWhitelist.UnknownFlags = true

	fs.String("runtime.mode", v.GetString("runtime.mode"), "Runtime mode for the daemon (e.g. docker, kubernetes)")
	fs.String("cluster.region", v.GetString("cluster.region"), "Cluster region this daemon belongs to")
	fs.String("node.id", v.GetString("node.id"), "Unique identifier for this daemon node")
	fs.Int("grpc.port", v.GetInt("grpc.port"), "Port for the gRPC server to listen on")
	fs.Int("metrics.port", v.GetInt("metrics.port"), "Port for Prometheus metrics exposure")
	fs.String("queue.backend", v.GetString("queue.backend"), "Backend for the message queue (e.g. memory, redis)")

	fs.Parse(os.Args[1:])
	v.BindPFlags(fs)

	var config Config
	config.RuntimeMode = RuntimeMode(v.GetString("runtime.mode"))
	config.ClusterRegion = v.GetString("cluster.region")
	config.NodeID = v.GetString("node.id")
	config.GRPCPort = v.GetInt("grpc.port")
	config.MetricsPort = v.GetInt("metrics.port")
	config.QueueBackend = QueueBackend(v.GetString("queue.backend"))

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return &config, nil
}
