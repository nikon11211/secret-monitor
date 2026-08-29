package secretmonitor

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type recordingLogger struct {
	messages []string
}

func (l *recordingLogger) DebugF(format string, args ...any) {
	l.messages = append(l.messages, "debugf")
}

func (l *recordingLogger) Debug(msg string) {
	l.messages = append(l.messages, "debug")
}

func (l *recordingLogger) Info(msg string) {
	l.messages = append(l.messages, "info")
}

func (l *recordingLogger) Warn(msg string) {
	l.messages = append(l.messages, "warn")
}

func (l *recordingLogger) Error(msg string) {
	l.messages = append(l.messages, "error")
}

func TestNewWithConfigInvalid(t *testing.T) {
	if _, err := NewWithConfig(Config{}); !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("expected ErrInvalidConfig, got %v", err)
	}
	if _, err := NewWithConfig(Config{ConfigFile: "   "}); !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("expected ErrInvalidConfig, got %v", err)
	}
	if _, err := New(""); !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("expected ErrInvalidConfig, got %v", err)
	}
}

func TestNewWithConfigDefaults(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "secrets.txt")
	if err := os.WriteFile(configPath, []byte("/secrets/one\n"), 0o600); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	monitor, err := NewWithConfig(Config{ConfigFile: configPath})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if monitor.config.CheckInterval != defaultCheckInterval {
		t.Fatalf("expected default check interval %s, got %s", defaultCheckInterval, monitor.config.CheckInterval)
	}
	if monitor.config.Timeout != 0 {
		t.Fatalf("expected no timeout, got %s", monitor.config.Timeout)
	}
	if _, ok := monitor.logger.(NoopLogger); !ok {
		t.Fatalf("expected default NoopLogger, got %T", monitor.logger)
	}
}

func TestNewWithConfigFromConfig(t *testing.T) {
	configPath := writeConfig(t, "/secrets/one")

	monitor, err := NewWithConfig(Config{
		ConfigFile:    configPath,
		CheckInterval: 5 * time.Second,
		Timeout:       9 * time.Second,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if monitor.config.CheckInterval != 5*time.Second {
		t.Fatalf("expected check interval 5s, got %s", monitor.config.CheckInterval)
	}
	if monitor.config.Timeout != 9*time.Second {
		t.Fatalf("expected timeout 9s, got %s", monitor.config.Timeout)
	}
}

func TestOptionsOverrideConfig(t *testing.T) {
	dir := t.TempDir()
	secretPath := filepath.Join(dir, "secret.txt")
	if err := os.WriteFile(secretPath, []byte("s3cr3t"), 0o600); err != nil {
		t.Fatalf("failed to write secret: %v", err)
	}
	configPath := writeConfig(t, secretPath)

	logger := &recordingLogger{}
	monitor, err := NewWithConfig(Config{
		ConfigFile:    configPath,
		CheckInterval: 5 * time.Second,
		Timeout:       9 * time.Second,
	},
		WithLogger(logger),
		WithCheckInterval(2*time.Second),
		WithTimeout(3*time.Second),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if monitor.config.CheckInterval != 2*time.Second {
		t.Fatalf("expected check interval 2s, got %s", monitor.config.CheckInterval)
	}
	if monitor.config.Timeout != 3*time.Second {
		t.Fatalf("expected timeout 3s, got %s", monitor.config.Timeout)
	}
	if monitor.logger != logger {
		t.Fatalf("expected custom logger to be applied")
	}

	if err := monitor.WaitForSecrets(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(logger.messages) == 0 {
		t.Fatalf("expected custom logger to be used")
	}
}

func TestOptionsWithZeroValuesKeepDefaults(t *testing.T) {
	configPath := writeConfig(t, "/secrets/one")

	monitor, err := NewWithConfig(
		Config{ConfigFile: configPath, CheckInterval: 5 * time.Second, Timeout: 9 * time.Second},
		WithCheckInterval(0),
		WithTimeout(0),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if monitor.config.CheckInterval != 5*time.Second {
		t.Fatalf("expected check interval 5s, got %s", monitor.config.CheckInterval)
	}
	if monitor.config.Timeout != 9*time.Second {
		t.Fatalf("expected timeout 9s, got %s", monitor.config.Timeout)
	}
}
