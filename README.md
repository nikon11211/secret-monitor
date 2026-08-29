<p align="center">
  <h1 align="center">secret-monitor - Dependency-Free Secret File Availability Monitor for Go</h1>
</p>

<p align="center">
  <a href="https://pkg.go.dev/github.com/nikon11211/secret-monitor">
    <img src="https://pkg.go.dev/badge/github.com/nikon11211/secret-monitor.svg" alt="Go Reference"/>
  </a>
  <a href="https://goreportcard.com/report/github.com/nikon11211/secret-monitor">
    <img src="https://goreportcard.com/badge/github.com/nikon11211/secret-monitor" alt="Go Report Card"/>
  </a>
  <a href="https://github.com/nikon11211/secret-monitor/actions/workflows/test.yaml">
    <img src="https://github.com/nikon11211/secret-monitor/actions/workflows/test.yaml/badge.svg" alt="Tests"/>
  </a>
  <a href="https://codecov.io/gh/nikon11211/secret-monitor">
    <img src="https://codecov.io/gh/nikon11211/secret-monitor/branch/main/graph/badge.svg" alt="Coverage"/>
  </a>
  <a href="https://sonarcloud.io/summary/overall?id=nikon11211_secret-monitor">
    <img src="https://sonarcloud.io/api/project_badges/measure?project=nikon11211_secret-monitor&metric=coverage" alt="SonarCloud Coverage"/>
  </a>
  <a href="https://opensource.org/licenses/MIT">
    <img src="https://img.shields.io/badge/License-MIT-yellow.svg" alt="License: MIT"/>
  </a>
  <a href="https://golang.org/">
    <img src="https://img.shields.io/badge/Go-%3E%3D%201.26-blue" alt="Go Version"/>
  </a>
</p>

<p align="center">
  <b>A tiny, dependency-free Go library that polls a list of secret files until every one of them exists on disk</b><br/>
  <i>Config file driven • Context cancellation • Timeout support • Pluggable logging • 100% test coverage</i>
</p>

---

## Why secret-monitor?

Containerized applications often receive secrets (mounts, `init` containers, sidecar fetchers) asynchronously: the process starts before the secret files are ready. Secret files can also be rotated or re-created at runtime. This library solves the startup race with a small, focused API — no frameworks, no external dependencies, just the Go standard library.

### Replacing vault init containers in Kubernetes / OpenShift

In Kubernetes and OpenShift environments it is a common pattern to use a **HashiCorp Vault `init` sidecar container** that fetches secrets from Vault and writes them to a shared `emptyDir` volume before the main application container starts. While this approach works, it introduces several operational concerns:

- **Extra container per pod** — increases resource consumption and pod startup latency.
- **Vault dependency at deploy time** — the init container must reach Vault, handle auth, and retry on transient failures.
- **Fragile lifecycle coupling** — the main container's readiness depends entirely on the init container completing successfully; if it fails or is slow, the whole pod is delayed.
- **No built-in runtime rotation support** — once the init container exits, secret refresh requires a sidecar or pod restart.

**secret-monitor** offers a lightweight, code-level alternative:

| | Vault init container | secret-monitor |
|---|---|---|
| **Extra containers** | 1 sidecar per pod | 0 — runs inside your app |
| **External dependencies** | Vault client libraries / HTTP | None — stdlib only |
| **Secret delivery** | Init container writes files, then exits | App polls a config file listing secret paths |
| **Runtime rotation** | Requires sidecar or restart | Native — poll loop detects new/updated files |
| **Startup latency** | Depends on Vault response time | Controlled via `CheckInterval` and `Timeout` |
| **Failure mode** | Pod stuck in `Init` state | Structured errors (`ErrTimeout`, `ErrStopped`) |

**Typical Kubernetes / OpenShift workflow:**

1. Use a Vault **sidecar injector** (or an `init` container) to populate a mounted volume with secret files — or rely on Kubernetes `Secret` / `ConfigMap` volume mounts.
2. Embed `secret-monitor` in your Go application to wait until every expected file is present before proceeding.
3. If secrets may be rotated at runtime (e.g. via Vault dynamic secrets or an external operator), the same poll loop transparently detects the new files.

```go
// Kubernetes example — wait for Vault-injected secrets
monitor, err := secretmonitor.New("/etc/secrets/config.txt",
    secretmonitor.WithTimeout(5*time.Minute),
    secretmonitor.WithCheckInterval(2*time.Second),
)
if err != nil {
    log.Fatal(err)
}
defer monitor.Stop()

if err := monitor.WaitForSecrets(context.Background()); err != nil {
    log.Fatalf("secrets not available: %v", err)
}
// All secret files are on disk — safe to start serving
```

> **TL;DR** — `secret-monitor` lets you remove the `vault init` sidecar from your pod spec and handle secret availability checks in-process, reducing overhead, eliminating a failure point, and giving you native runtime rotation support for free.

---

## Features

<table>
<tr>
<td width="50%">

### Core
- Config-file driven: plain-text file listing one secret path per line
- Polling loop using `os.Stat` — no filesystem watches, no daemons
- `context.Context` cancellation and deadline support
- Optional overall timeout via `Config.Timeout` (returns `ErrTimeout`)
- Explicit `Stop()` to abort an in-progress wait
- `New` and `NewWithConfig` constructors with functional options

</td>
<td width="50%">

### Engineering
- Zero dependencies — standard library only
- Pluggable `Logger` interface, silent `NoopLogger` by default
- Sentinel errors (`ErrNoSecrets`, `ErrTimeout`, `ErrStopped`, ...) with `%w` wrapping
- Concurrency-safe: guards against concurrent `WaitForSecrets` calls
- 100% statement coverage, race-detector clean, benchmarks included

</td>
</tr>
</table>

## Installation

```bash
go get github.com/nikon11211/secret-monitor
```

## Architecture

```
┌──────────────────────────────────────────────────────────────┐
│                     Your Application                         │
├──────────────────────────────────────────────────────────────┤
│                       secret-monitor                         │
│                                                              │
│   ┌──────────────┐         ┌──────────────────────────┐      │
│   │    Config    │         │         Monitor          │      │
│   │  ConfigFile  │────────▶│  New / NewWithConfig     │      │
│   │ CheckInterval│         │  WithLogger & friends    │      │
│   │   Timeout    │         └───────────┬──────────────┘      │
│   └──────────────┘                     │                     │
│                                        ▼                     │
│   ┌────────────────────────────────────────────────────┐     │
│   │          WaitForSecrets(ctx) poll loop             │     │
│   │                                                    │     │
│   │   ┌────────┐      ┌────────┐     ┌─────────────┐   │     │
│   │   │ctx.Done│      │ stopCh │     │  timeoutCh  │   │     │
│   │   └────────┘      └────────┘     └─────────────┘   │     │
│   │             select│      │      │                  │     │
│   │                   ▼      ▼      ▼                  │     │
│   │      os.Stat each secret file → all exist? → nil   │     │
│   └────────────────────────────────────────────────────┘     │
└──────────────────────────────────────────────────────────────┘
```

## Quick Start

```go
package main

import (
	"context"
	"log"
	"time"

	secretmonitor "github.com/nikon11211/secret-monitor"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	monitor, err := secretmonitor.New(
		"/etc/secrets/config.txt",
		secretmonitor.WithCheckInterval(2*time.Second),
		secretmonitor.WithTimeout(5*time.Minute),
		secretmonitor.WithLogger(logger{}),
	)
	if err != nil {
		log.Fatalf("failed to create monitor: %v", err)
	}
	defer monitor.Stop()

	if err := monitor.WaitForSecrets(ctx); err != nil {
		log.Fatalf("secrets not available: %v", err)
	}
	log.Println("all secrets are available")
}

type logger struct{}

func (logger) DebugF(format string, args ...any) { log.Printf("[DEBUG] "+format, args...) }
func (logger) Debug(msg string)                  { log.Println("[DEBUG]", msg) }
func (logger) Info(msg string)                   { log.Println("[INFO]", msg) }
func (logger) Warn(msg string)                   { log.Println("[WARN]", msg) }
func (logger) Error(msg string)                  { log.Println("[ERROR]", msg) }
```

A complete runnable example lives in [`examples/basic`](examples/basic).

### Config file format

A plain-text file, one secret file path per line. Blank lines and surrounding whitespace are ignored:

```
/etc/secrets/db-password
/etc/secrets/api-token
/etc/secrets/tls-cert.crt
```

## Configuration Reference

```go
type Config struct {
	ConfigFile    string        // Path to the secret list file (required)
	CheckInterval time.Duration // Poll delay; 0 defaults to time.Second
	Timeout       time.Duration // Overall wait budget; 0 means no timeout
}
```

| Option | Description |
| --- | --- |
| `WithLogger(l Logger)` | Replace the default `NoopLogger` |
| `WithCheckInterval(d time.Duration)` | Override the polling interval |
| `WithTimeout(d time.Duration)` | Override the overall timeout |

### Errors

| Sentinel | Meaning |
| --- | --- |
| `ErrInvalidConfig` | `ConfigFile` is empty |
| `ErrNoSecrets` | Config file contains no secret paths |
| `ErrTimeout` | `Timeout` expired before all secrets appeared |
| `ErrStopped` | `Stop()` was called during the wait |
| `ErrAlreadyRunning` | `WaitForSecrets` called while a wait is in progress |
| `ErrNilContext` | `WaitForSecrets` called with a nil context |

Underlying causes (e.g. `os.ErrNotExist` for a missing config file, `context.Canceled`, `context.DeadlineExceeded`) are always reachable via `errors.Is`.

## Testing & Benchmarks

```go
// Run all tests
go test ./...

// Run with race detection and coverage (examples excluded)
go test -race -coverprofile=coverage.txt -covermode=atomic $(go list ./... | grep -v /examples)

// Inspect per-function coverage
go tool cover -func=coverage.txt

// View coverage in the browser
go tool cover -html=coverage.txt

// Run benchmarks
go test -bench=. -benchmem -run '^$' .
```

| Benchmark                        | What it measures                            |
|----------------------------------|---------------------------------------------|
| `BenchmarkParseSecretFiles`      | Parsing the secret-list config file         |
| `BenchmarkAllSecretFilesExist`   | Snapshot availability check via `os.Stat`   |
| `BenchmarkWaitForSecretsAllAvailable` | Full wait path when secrets already exist |

## Contributing

We welcome contributions! Here's how you can help:

1. **Fork** the repository
2. **Create** a feature branch (`git checkout -b feature/amazing-feature`)
3. **Commit** your changes (`git commit -m 'Add amazing feature'`)
4. **Push** to the branch (`git push origin feature/amazing-feature`)
5. **Open** a Pull Request

Please keep the 100% coverage gate green — new behavior must ship with tests.

## License

MIT License - see [LICENSE](LICENSE) for details.

## Acknowledgments

- The Go standard library — the only thing this library depends on
- [Go Report Card](https://goreportcard.com) and [pkg.go.dev](https://pkg.go.dev) — keeping quality visible

---

<p align="center">
  <b>Made with ❤️ for the Go community</b><br/>
  <sub>Wait for your secrets, start with confidence</sub>
</p>
