package secretmonitor

import (
	"fmt"
	"os"
	"strings"
	"time"
)

const defaultCheckInterval = time.Second

type Config struct {
	ConfigFile    string
	CheckInterval time.Duration
	Timeout       time.Duration
}

func readSecretFiles(path string) ([]string, error) {
	content, err := os.ReadFile(path) // #nosec G304 -- file path comes from trusted configuration
	if err != nil {
		return nil, fmt.Errorf("unable to read secret config file %q: %w", path, err)
	}
	return parseSecretFiles(content)
}

func parseSecretFiles(content []byte) ([]string, error) {
	files := make([]string, 0)
	for line := range strings.SplitSeq(string(content), "\n") {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			files = append(files, trimmed)
		}
	}
	if len(files) == 0 {
		return nil, ErrNoSecrets
	}
	return files, nil
}
