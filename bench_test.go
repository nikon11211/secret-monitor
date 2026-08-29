package secretmonitor

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func BenchmarkParseSecretFiles(b *testing.B) {
	var sb strings.Builder
	for i := range 100 {
		sb.WriteString(fmt.Sprintf("/tmp/secrets/secret-%d.txt\n", i))
	}
	content := []byte(sb.String())

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		files, err := parseSecretFiles(content)
		if err != nil {
			b.Fatal(err)
		}
		if len(files) != 100 {
			b.Fatalf("expected 100 files, got %d", len(files))
		}
	}
}

func BenchmarkAllSecretFilesExist(b *testing.B) {
	dir := b.TempDir()
	files := make([]string, 5)
	for i := range files {
		files[i] = filepath.Join(dir, fmt.Sprintf("secret-%d.txt", i))
		if err := os.WriteFile(files[i], []byte("s3cr3t"), 0o600); err != nil {
			b.Fatal(err)
		}
	}

	monitor := &Monitor{}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if !monitor.allSecretFilesExist(files) {
			b.Fatal("expected all files to exist")
		}
	}
}

func BenchmarkWaitForSecretsAllAvailable(b *testing.B) {
	dir := b.TempDir()
	files := make([]string, 3)
	var sb strings.Builder
	for i := range files {
		files[i] = filepath.Join(dir, fmt.Sprintf("secret-%d.txt", i))
		if err := os.WriteFile(files[i], []byte("s3cr3t"), 0o600); err != nil {
			b.Fatal(err)
		}
		sb.WriteString(files[i] + "\n")
	}
	configPath := filepath.Join(dir, "secrets.txt")
	if err := os.WriteFile(configPath, []byte(sb.String()), 0o600); err != nil {
		b.Fatal(err)
	}

	monitor, err := New(configPath, WithCheckInterval(time.Millisecond))
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if err := monitor.WaitForSecrets(context.Background()); err != nil {
			b.Fatal(err)
		}
	}
}
