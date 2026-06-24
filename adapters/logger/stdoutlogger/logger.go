package stdoutlogger

import (
	"log/slog"

	"github.com/vernal96/go-cms/core"
)

type Logger struct {
	logger *slog.Logger
}

func New(logger *slog.Logger) *Logger {
	if logger == nil {
		logger = slog.Default()
	}

	return &Logger{
		logger: logger,
	}
}

func (l *Logger) Debug(message string, fields ...core.LogField) {
	l.logger.Debug(message, slogFields(fields)...)
}

func (l *Logger) Info(message string, fields ...core.LogField) {
	l.logger.Info(message, slogFields(fields)...)
}

func (l *Logger) Warn(message string, fields ...core.LogField) {
	l.logger.Warn(message, slogFields(fields)...)
}

func (l *Logger) Error(message string, fields ...core.LogField) {
	l.logger.Error(message, slogFields(fields)...)
}

func slogFields(fields []core.LogField) []any {
	attrs := make([]any, 0, len(fields)*2)
	for _, field := range fields {
		attrs = append(attrs, field.Key, field.Value)
	}

	return attrs
}

var _ core.Logger = (*Logger)(nil)
