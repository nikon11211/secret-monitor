package secretmonitor

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func writeConfig(t *testing.T, paths ...string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "secrets.txt")
	var content strings.Builder
	for _, p := range paths {
		content.WriteString(p + "\n")
	}
	if err := os.WriteFile(path, []byte(content.String()), 0o600); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}
	return path
}

func stubSleep() func() {
	original := sleep
	sleep = func(time.Duration) {}
	return func() { sleep = original }
}

func TestWaitForSecretsSuccess(t *testing.T) {
	restore := stubSleep()
	defer restore()

	dir := t.TempDir()
	secretPath := filepath.Join(dir, "secret.txt")
	configPath := writeConfig(t, secretPath)

	monitor, err := New(configPath, WithCheckInterval(10*time.Millisecond))
	if err != nil {
		t.Fatalf("failed to create monitor: %v", err)
	}

	go func() {
		time.Sleep(20 * time.Millisecond)
		if err := os.WriteFile(secretPath, []byte("s3cr3t"), 0o600); err != nil {
			t.Errorf("failed to write secret file: %v", err)
		}
	}()

	if err := monitor.WaitForSecrets(context.Background()); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestWaitForSecretsTimeout(t *testing.T) {
	restore := stubSleep()
	defer restore()

	dir := t.TempDir()
	configPath := writeConfig(t, filepath.Join(dir, "never-exists.txt"))

	monitor, err := NewWithConfig(Config{
		ConfigFile:    configPath,
		CheckInterval: time.Millisecond,
		Timeout:       30 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("failed to create monitor: %v", err)
	}

	err = monitor.WaitForSecrets(context.Background())
	if !errors.Is(err, ErrTimeout) {
		t.Fatalf("expected ErrTimeout, got %v", err)
	}
}

func TestWaitForSecretsContextCancel(t *testing.T) {
	restore := stubSleep()
	defer restore()

	configPath := writeConfig(t, filepath.Join(t.TempDir(), "never-exists.txt"))

	monitor, err := New(configPath)
	if err != nil {
		t.Fatalf("failed to create monitor: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err = monitor.WaitForSecrets(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected wrapped context.Canceled, got %v", err)
	}
}

func TestWaitForSecretsContextDeadline(t *testing.T) {
	restore := stubSleep()
	defer restore()

	configPath := writeConfig(t, filepath.Join(t.TempDir(), "never-exists.txt"))

	monitor, err := New(configPath)
	if err != nil {
		t.Fatalf("failed to create monitor: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()

	err = monitor.WaitForSecrets(ctx)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected wrapped context.DeadlineExceeded, got %v", err)
	}
}

func TestWaitForSecretsMissingConfigFile(t *testing.T) {
	monitor, err := New(filepath.Join(t.TempDir(), "missing.txt"))
	if err != nil {
		t.Fatalf("failed to create monitor: %v", err)
	}

	err = monitor.WaitForSecrets(context.Background())
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected wrapped os.ErrNotExist, got %v", err)
	}
}

func TestWaitForSecretsNoSecrets(t *testing.T) {
	restore := stubSleep()
	defer restore()

	monitor, err := New(writeConfig(t))
	if err != nil {
		t.Fatalf("failed to create monitor: %v", err)
	}

	err = monitor.WaitForSecrets(context.Background())
	if !errors.Is(err, ErrNoSecrets) {
		t.Fatalf("expected ErrNoSecrets, got %v", err)
	}
}

func TestWaitForSecretsNilContext(t *testing.T) {
	monitor, err := New(writeConfig(t, "/secrets/one"))
	if err != nil {
		t.Fatalf("failed to create monitor: %v", err)
	}

	err = monitor.WaitForSecrets(nil)
	if !errors.Is(err, ErrNilContext) {
		t.Fatalf("expected ErrNilContext, got %v", err)
	}
}

func TestStopBeforeWait(t *testing.T) {
	restore := stubSleep()
	defer restore()

	configPath := writeConfig(t, filepath.Join(t.TempDir(), "never-exists.txt"))

	monitor, err := New(configPath)
	if err != nil {
		t.Fatalf("failed to create monitor: %v", err)
	}

	monitor.Stop()
	monitor.Stop()

	err = monitor.WaitForSecrets(context.Background())
	if !errors.Is(err, ErrStopped) {
		t.Fatalf("expected ErrStopped, got %v", err)
	}
}

func TestWaitForSecretsAlreadyRunningAndStop(t *testing.T) {
	release := make(chan struct{})
	original := sleep
	sleep = func(time.Duration) { <-release }
	defer func() { sleep = original }()

	configPath := writeConfig(t, filepath.Join(t.TempDir(), "never-exists.txt"))

	monitor, err := New(configPath, WithCheckInterval(time.Millisecond))
	if err != nil {
		t.Fatalf("failed to create monitor: %v", err)
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- monitor.WaitForSecrets(context.Background())
	}()

	deadline := time.Now().Add(5 * time.Second)
	for {
		monitor.mu.Lock()
		started := monitor.started
		monitor.mu.Unlock()
		if started {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("monitor did not start in time")
		}
		time.Sleep(time.Millisecond)
	}

	err = monitor.WaitForSecrets(context.Background())
	if !errors.Is(err, ErrAlreadyRunning) {
		t.Fatalf("expected ErrAlreadyRunning, got %v", err)
	}

	close(release)
	monitor.Stop()

	select {
	case err := <-errCh:
		if !errors.Is(err, ErrStopped) {
			t.Fatalf("expected ErrStopped, got %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("wait did not stop in time")
	}
}

func TestWaitForSecretsCanRunAgainAfterCompletion(t *testing.T) {
	restore := stubSleep()
	defer restore()

	dir := t.TempDir()
	secretPath := filepath.Join(dir, "secret.txt")
	if err := os.WriteFile(secretPath, []byte("s3cr3t"), 0o600); err != nil {
		t.Fatalf("failed to write secret: %v", err)
	}
	configPath := writeConfig(t, secretPath)

	monitor, err := New(configPath)
	if err != nil {
		t.Fatalf("failed to create monitor: %v", err)
	}

	for i := range 2 {
		if err := monitor.WaitForSecrets(context.Background()); err != nil {
			t.Fatalf("wait %d failed: %v", i+1, err)
		}
	}
}
