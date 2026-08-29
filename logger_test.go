package secretmonitor

import "testing"

func TestNoopLogger(t *testing.T) {
	logger := NoopLogger{}
	logger.DebugF("formatted %d", 42)
	logger.Debug("debug message")
	logger.Info("info message")
	logger.Warn("warn message")
	logger.Error("error message")
}
