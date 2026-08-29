package secretmonitor

import "errors"

var (
	ErrInvalidConfig  = errors.New("secretmonitor: invalid configuration")
	ErrNoSecrets      = errors.New("secretmonitor: config file contains no secret files")
	ErrTimeout        = errors.New("secretmonitor: timed out waiting for secret files")
	ErrStopped        = errors.New("secretmonitor: monitoring stopped")
	ErrAlreadyRunning = errors.New("secretmonitor: wait for secrets already in progress")
	ErrNilContext     = errors.New("secretmonitor: nil context")
)
