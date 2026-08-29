package secretmonitor

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestParseSecretFiles(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    []string
		wantErr error
	}{
		{
			name:    "empty file",
			content: "",
			wantErr: ErrNoSecrets,
		},
		{
			name:    "whitespace only",
			content: "  \n\t\n \n\n",
			wantErr: ErrNoSecrets,
		},
		{
			name:    "valid paths with whitespace and CRLF",
			content: "/secrets/one\n  /secrets/two  \n/secrets/three\r\n\n",
			want:    []string{"/secrets/one", "/secrets/two", "/secrets/three"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseSecretFiles([]byte(tt.content))
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("expected %d files, got %d: %v", len(tt.want), len(got), got)
			}
			for i, want := range tt.want {
				if got[i] != want {
					t.Fatalf("file %d: expected %q, got %q", i, want, got[i])
				}
			}
		})
	}
}

func TestReadSecretFiles(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "secrets.txt")
	if err := os.WriteFile(path, []byte("/one\n/two\n"), 0o600); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	files, err := readSecretFiles(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 2 || files[0] != "/one" || files[1] != "/two" {
		t.Fatalf("unexpected files: %v", files)
	}

	missing := filepath.Join(dir, "does-not-exist.txt")
	if _, err := readSecretFiles(missing); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected os.ErrNotExist, got %v", err)
	}
}
