package secretmonitor

import "fmt"

type Logger interface {
	DebugF(format string, args ...any)
	Debug(msg string)
	Info(msg string)
	Warn(msg string)
	Error(msg string)
}

type NoopLogger struct{}

func (n NoopLogger) DebugF(format string, args ...any) {
	_ = fmt.Sprintf("%"+format, args...)
}

func (n NoopLogger) Debug(msg string) {
	_ = msg
}

func (n NoopLogger) Info(msg string) {
	_ = msg
}

func (n NoopLogger) Warn(msg string) {
	_ = msg
}

func (n NoopLogger) Error(msg string) {
	_ = msg
}
