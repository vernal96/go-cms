package core

type Logger interface {
	Debug(message string, fields ...LogField)
	Info(message string, fields ...LogField)
	Warn(message string, fields ...LogField)
	Error(message string, fields ...LogField)
}

type LogField struct {
	Key   string
	Value any
}

type NullLogger struct{}

func (NullLogger) Debug(message string, fields ...LogField) {}
func (NullLogger) Info(message string, fields ...LogField)  {}
func (NullLogger) Warn(message string, fields ...LogField)  {}
func (NullLogger) Error(message string, fields ...LogField) {}

var _ Logger = NullLogger{}
