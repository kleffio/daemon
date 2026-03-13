package logging

import "github.com/kleffio/gameserver-daemon/internal/application/ports"

// NoopLogger is a fake logger that drops all messages.
// It is intended exclusively for unit tests to satisfy the ports.Logger interface
// without cluttering stdout or throwing nil pointer panics.
type NoopLogger struct{}

// NewNoopLogger creates a new fake logger.
func NewNoopLogger() *NoopLogger {
	return &NoopLogger{}
}

func (n *NoopLogger) Info(msg string, args ...any)             {}
func (n *NoopLogger) Error(msg string, err error, args ...any) {}
func (n *NoopLogger) Debug(msg string, args ...any)            {}
func (n *NoopLogger) Warn(msg string, args ...any)             {}
func (n *NoopLogger) With(args ...any) ports.Logger            { return n }
