package core

// Logger defines a simple structured logging interface for the CMS.
// It has methods for common log levels. Each method accepts a message
// and optional context arguments. Implementations may ignore unused
// arguments.
//
// Modules and services should depend on this interface rather than a
// concrete logging library. A no-op implementation (NullLogger) is
// provided for testing and default usage until a real logger is injected.
type Logger interface {
    // Debug logs verbose diagnostic information intended for development.
    Debug(msg string, args ...any)

    // Info logs general informational messages about application workflow.
    Info(msg string, args ...any)

    // Warn logs non-critical issues that might require attention.
    Warn(msg string, args ...any)

    // Error logs critical errors that typically indicate a failure.
    Error(msg string, args ...any)
}

// NullLogger is a logger implementation that does nothing.
//
// It is useful as a default or testing logger to avoid nil checks.
// All methods return immediately without producing any output.
type NullLogger struct{}

// NewNullLogger returns a new instance of NullLogger.
func NewNullLogger() *NullLogger {
    return &NullLogger{}
}

// Debug is a no-op for NullLogger.
func (l *NullLogger) Debug(msg string, args ...any) {}

// Info is a no-op for NullLogger.
func (l *NullLogger) Info(msg string, args ...any) {}

// Warn is a no-op for NullLogger.
func (l *NullLogger) Warn(msg string, args ...any) {}

// Error is a no-op for NullLogger.
func (l *NullLogger) Error(msg string, args ...any) {}

// Assert that NullLogger implements Logger.
var _ Logger = (*NullLogger)(nil)
