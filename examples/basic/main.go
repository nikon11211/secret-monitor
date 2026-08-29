package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	secretmonitor "github.com/nikon11211/secret-monitor"
)

type simpleLogger struct{}

func (l simpleLogger) DebugF(format string, args ...any) { log.Printf("[DEBUG] "+format, args...) }
func (l simpleLogger) Debug(msg string)                  { log.Println("[DEBUG]", msg) }
func (l simpleLogger) Info(msg string)                   { log.Println("[INFO]", msg) }
func (l simpleLogger) Warn(msg string)                   { log.Println("[WARN]", msg) }
func (l simpleLogger) Error(msg string)                  { log.Println("[ERROR]", msg) }

func main() {
	configFile := getEnv("SECRETS_CONFIG", "/tmp/vault/config/secrets.txt")

	monitor, err := secretmonitor.New(
		configFile,
		secretmonitor.WithLogger(simpleLogger{}),
		secretmonitor.WithCheckInterval(2*time.Second),
		secretmonitor.WithTimeout(5*time.Minute),
	)
	if err != nil {
		log.Fatalf("Failed to create secret monitor: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		log.Println("Received shutdown signal, stopping monitor")
		monitor.Stop()
		cancel()
	}()

	log.Printf("Waiting for secrets listed in %s ...", configFile)
	if err := monitor.WaitForSecrets(ctx); err != nil {
		log.Fatalf("Failed waiting for secrets: %v", err)
	}

	fmt.Println("All secrets are available, application can start")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
