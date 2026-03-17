package logging

import (
	"log/slog"
	"os"

	"github.com/kleffio/gameserver-daemon/internal/application/ports"
)

type SlogAdapter struct {
	logger *slog.Logger
}

func NewSlogAdapter() *SlogAdapter {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	return &SlogAdapter{
		logger: slog.New(handler),
	}
}

func (s *SlogAdapter) Info(msg string, args ...any) {
	s.logger.Info(msg, args...)
}

func (s *SlogAdapter) Error(msg string, err error, args ...any) {
	allArgs := append([]any{"error", err.Error()}, args...)
	s.logger.Error(msg, allArgs...)
}

func (s *SlogAdapter) Debug(msg string, args ...any) {
	s.logger.Debug(msg, args...)
}

func (s *SlogAdapter) Warn(msg string, args ...any) {
	s.logger.Warn(msg, args...)
}

func (s *SlogAdapter) With(args ...any) ports.Logger {
	return &SlogAdapter{
		logger: s.logger.With(args...),
	}
}
