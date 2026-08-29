package secretmonitor

import "time"

type Option func(*options)

type options struct {
	logger        Logger
	checkInterval time.Duration
	timeout       time.Duration
}

func WithLogger(l Logger) Option {
	return func(o *options) {
		o.logger = l
	}
}

func WithCheckInterval(d time.Duration) Option {
	return func(o *options) {
		if d > 0 {
			o.checkInterval = d
		}
	}
}

func WithTimeout(d time.Duration) Option {
	return func(o *options) {
		if d > 0 {
			o.timeout = d
		}
	}
}
