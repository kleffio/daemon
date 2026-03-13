package ports

// Standardized Logging Keys
const (
	LogKeyNodeID     = "node_id"
	LogKeyJobID      = "job_id"
	LogKeyServerID   = "server_id"
	LogKeyTraceID    = "trace_id"
	LogKeyWorkerType = "worker_type"
)

// Logger defines the interface for structured logging across the application.
type Logger interface {
	Info(msg string, args ...any)
	Error(msg string, err error, args ...any)
	Debug(msg string, args ...any)
	Warn(msg string, args ...any)

	// With returns a new Logger that includes the given attributes.
	// This is meant for creating child loggers with contextual data (like job IDs).
	With(args ...any) Logger
}
