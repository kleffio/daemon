package logging

import (
	"log/slog"
	"os"

	"github.com/kleffio/gameserver-daemon/internal/application/ports"
)

// SlogAdapter implements ports.Logger using the standard library slog package.
type SlogAdapter struct {
	logger *slog.Logger
}

// NewSlogAdapter creates a new JSON structured logger.
func NewSlogAdapter() *SlogAdapter {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	return &SlogAdapter{
		logger: slog.New(handler),
	}
}

// Info logs at LevelInfo.
func (s *SlogAdapter) Info(msg string, args ...any) {
	s.logger.Info(msg, args...)
}

// Error logs at LevelError. It explicitly handles the error object as a structured attribute.
func (s *SlogAdapter) Error(msg string, err error, args ...any) {
	// Prepend the error explicitly so it's always captured distinctly
	allArgs := append([]any{"error", err.Error()}, args...)
	s.logger.Error(msg, allArgs...)
}

// Debug logs at LevelDebug.
func (s *SlogAdapter) Debug(msg string, args ...any) {
	s.logger.Debug(msg, args...)
}

// Warn logs at LevelWarn.
func (s *SlogAdapter) Warn(msg string, args ...any) {
	s.logger.Warn(msg, args...)
}

// With returns a new Logger that includes the given attributes.
func (s *SlogAdapter) With(args ...any) ports.Logger {
	return &SlogAdapter{
		logger: s.logger.With(args...),
	}
}
