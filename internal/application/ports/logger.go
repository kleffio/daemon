package ports


const (
	LogKeyNodeID     = "node_id"
	LogKeyJobID      = "job_id"
	LogKeyServerID   = "server_id"
	LogKeyTraceID    = "trace_id"
	LogKeyWorkerType = "worker_type"
)


type Logger interface {
	Info(msg string, args ...any)
	Error(msg string, err error, args ...any)
	Debug(msg string, args ...any)
	Warn(msg string, args ...any)


	With(args ...any) Logger
}
