package secretmonitor

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

var sleep = time.Sleep

type Monitor struct {
	config Config
	logger Logger

	stopCh   chan struct{}
	stopOnce sync.Once

	mu      sync.Mutex
	started bool
}

func New(configFile string, opts ...Option) (*Monitor, error) {
	return NewWithConfig(Config{ConfigFile: configFile}, opts...)
}

func NewWithConfig(cfg Config, opts ...Option) (*Monitor, error) {
	if strings.TrimSpace(cfg.ConfigFile) == "" {
		return nil, fmt.Errorf("%w: ConfigFile is required", ErrInvalidConfig)
	}

	o := options{
		logger:        NoopLogger{},
		checkInterval: defaultCheckInterval,
	}
	if cfg.CheckInterval > 0 {
		o.checkInterval = cfg.CheckInterval
	}
	if cfg.Timeout > 0 {
		o.timeout = cfg.Timeout
	}
	for _, opt := range opts {
		opt(&o)
	}

	return &Monitor{
		config: Config{
			ConfigFile:    cfg.ConfigFile,
			CheckInterval: o.checkInterval,
			Timeout:       o.timeout,
		},
		logger: o.logger,
		stopCh: make(chan struct{}),
	}, nil
}

func (m *Monitor) WaitForSecrets(ctx context.Context) error {
	if ctx == nil {
		return fmt.Errorf("%w", ErrNilContext)
	}
	if err := m.begin(); err != nil {
		return err
	}
	defer m.end()

	files, err := readSecretFiles(m.config.ConfigFile)
	if err != nil {
		return err
	}

	m.logger.Info(fmt.Sprintf("Monitoring %d secret files", len(files)))
	for _, file := range files {
		m.logger.Debug(fmt.Sprintf("Watching secret file %q", file))
	}

	var timeoutCh <-chan time.Time
	if m.config.Timeout > 0 {
		timer := time.NewTimer(m.config.Timeout)
		defer timer.Stop()
		timeoutCh = timer.C
	}

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("wait for secrets aborted: %w", ctx.Err())
		case <-timeoutCh:
			return fmt.Errorf("%w after %s", ErrTimeout, m.config.Timeout)
		case <-m.stopCh:
			return fmt.Errorf("%w", ErrStopped)
		default:
		}

		if m.allSecretFilesExist(files) {
			m.logger.Info("All secret files are available")
			return nil
		}

		sleep(m.config.CheckInterval)
	}
}

func (m *Monitor) Stop() {
	m.stopOnce.Do(func() {
		close(m.stopCh)
		m.logger.Warn("Secret monitor stopped")
	})
}

func (m *Monitor) allSecretFilesExist(files []string) bool {
	for _, file := range files {
		if _, err := os.Stat(file); err != nil {
			m.logger.Debug(fmt.Sprintf("Secret file %q is not available yet: %v", file, err))
			return false
		}
	}
	return true
}

func (m *Monitor) begin() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.started {
		return ErrAlreadyRunning
	}
	m.started = true
	return nil
}

func (m *Monitor) end() {
	m.mu.Lock()
	m.started = false
	m.mu.Unlock()
}
