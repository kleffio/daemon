package config

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

type RuntimeMode string

const (
	RuntimeModeDocker     RuntimeMode = "docker"
	RuntimeModeKubernetes RuntimeMode = "kubernetes"
)

type QueueBackend string

const (
	QueueBackendMemory QueueBackend = "memory"
	QueueBackendRedis  QueueBackend = "redis"
)

type Config struct {
	RuntimeMode   RuntimeMode  `mapstructure:"runtime.mode"`
	ClusterRegion string       `mapstructure:"cluster.region"`
	NodeID        string       `mapstructure:"node.id"`
	GRPCPort      int          `mapstructure:"grpc.port"`
	MetricsPort   int          `mapstructure:"metrics.port"`
	QueueBackend  QueueBackend `mapstructure:"queue.backend"`
	DatabasePath  string       `mapstructure:"database.path"`
	RedisURL      string       `mapstructure:"redis.url"`
	RedisPassword string       `mapstructure:"redis.password"`
	RedisTLS      bool         `mapstructure:"redis.tls"`
	PlatformURL   string       `mapstructure:"platform.url"`
	SharedSecret  string       `mapstructure:"shared_secret"`
}

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

	if strings.TrimSpace(c.PlatformURL) == "" {
		return fmt.Errorf("KLEFF_PLATFORM_URL is required and cannot be empty")
	}

	if strings.TrimSpace(c.SharedSecret) == "" {
		return fmt.Errorf("KLEFF_SHARED_SECRET is required and cannot be empty")
	}

	return nil
}

// gsdEnvMappings lists every known GSD_* variable and its KLEFF_* replacement.
// The shim copies GSD_* values to KLEFF_* at startup when KLEFF_* is not already set.
var gsdEnvMappings = []struct {
	gsd   string
	kleff string
}{
	{"GSD_RUNTIME_MODE", "KLEFF_RUNTIME_MODE"},
	{"GSD_CLUSTER_REGION", "KLEFF_CLUSTER_REGION"},
	{"GSD_NODE_ID", "KLEFF_NODE_ID"},
	{"GSD_GRPC_PORT", "KLEFF_GRPC_PORT"},
	{"GSD_METRICS_PORT", "KLEFF_METRICS_PORT"},
	{"GSD_QUEUE_BACKEND", "KLEFF_QUEUE_BACKEND"},
	{"GSD_DATABASE_PATH", "KLEFF_DATABASE_PATH"},
	{"GSD_REDIS_URL", "KLEFF_REDIS_URL"},
	{"GSD_REDIS_PASSWORD", "KLEFF_REDIS_PASSWORD"},
	{"GSD_REDIS_TLS", "KLEFF_REDIS_TLS"},
	{"GSD_PLATFORM_URL", "KLEFF_PLATFORM_URL"},
	{"GSD_SHARED_SECRET", "KLEFF_SHARED_SECRET"},
}

// applyDeprecationShim copies any set GSD_* env vars to their KLEFF_* equivalents
// (unless KLEFF_* is already set) and logs a structured warning for each one found.
func applyDeprecationShim() {
	for _, m := range gsdEnvMappings {
		val := os.Getenv(m.gsd)
		if val == "" {
			continue
		}
		if os.Getenv(m.kleff) != "" {
			continue
		}
		os.Setenv(m.kleff, val)
		slog.Warn("deprecated env var in use; rename to KLEFF_* equivalent",
			"deprecated", m.gsd,
			"replacement", m.kleff,
		)
	}
}

func Load() (*Config, error) {
	applyDeprecationShim()

	v := viper.New()

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
	v.SetDefault("database.path", "./data/kleff.db")
	v.SetDefault("redis.url", "redis://localhost:6379/0")
	v.SetDefault("redis.password", "")
	v.SetDefault("redis.tls", false)
	v.SetDefault("platform.url", "")
	v.SetDefault("shared_secret", "")

	v.SetEnvPrefix("kleff")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath("/etc/gameserver-daemon/")
	v.AddConfigPath(".")

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("fatal error config file: %w", err)
		}
	}

	fs := pflag.NewFlagSet("kleff", pflag.ContinueOnError)
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
	config.DatabasePath = v.GetString("database.path")
	config.RedisURL = v.GetString("redis.url")
	config.RedisPassword = v.GetString("redis.password")
	config.RedisTLS = v.GetBool("redis.tls")
	config.PlatformURL = v.GetString("platform.url")
	config.SharedSecret = v.GetString("shared_secret")

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return &config, nil
}
