package logging

import "github.com/kleffio/gameserver-daemon/internal/application/ports"

type NoopLogger struct{}

func NewNoopLogger() *NoopLogger {
	return &NoopLogger{}
}

func (n *NoopLogger) Info(msg string, args ...any)             {}
func (n *NoopLogger) Error(msg string, err error, args ...any) {}
func (n *NoopLogger) Debug(msg string, args ...any)            {}
func (n *NoopLogger) Warn(msg string, args ...any)             {}
func (n *NoopLogger) With(args ...any) ports.Logger            { return n }
