*[Русская версия](go.ru.md)*

# Code Style Guide: Go SDK

This document describes code style conventions for the Go SDK (`sdk-go/`).
See also: [General Principles](overview.md) | [Testing](testing.md)

## Naming Conventions

### Packages

- Short, lowercase, single-word names
- No underscores or mixedCaps
- Package name should not repeat the import path

```go
package dephealth  // good
package checks     // good

package dep_health  // bad — no underscores
package depHealth   // bad — no mixedCaps
```

### Exported vs Unexported

- Exported (public): `PascalCase` — visible outside the package
- Unexported (private): `camelCase` — package-internal

```go
// Exported
type HealthChecker interface { }
type Dependency struct { }
func New(serviceName string, opts ...Option) (*DependencyHealth, error) { }

// Unexported
type checkResult struct { }
func sanitizeURL(raw string) string { }
```

### Acronyms

Acronyms are all caps when exported, all lower when unexported:

```go
type HTTPChecker struct { }    // not HttpChecker
type GRPCChecker struct { }    // not GrpcChecker
type TCPChecker struct { }     // not TcpChecker

func parseURL(raw string) { }  // not parseUrl
var httpClient *http.Client     // not hTTPClient
```

### Interfaces

- Single-method interfaces: method name + `er` suffix
- Multi-method interfaces: descriptive noun

```go
// Single method — "er" suffix
type HealthChecker interface {
    Check(ctx context.Context, endpoint Endpoint) error
    Type() string
}

// Avoid generic names
type Doer interface { }  // bad — too vague
```

## Package Structure

```text
sdk-go/
├── dephealth/
│   ├── dephealth.go         // New(), DependencyHealth, Start/Stop
│   ├── options.go           // Option type, functional options
│   ├── dependency.go        // Dependency, Endpoint structs
│   ├── checker.go           // HealthChecker interface, sentinel errors
│   ├── scheduler.go         // check scheduler (goroutines)
│   ├── parser.go            // URL/params parser
│   ├── metrics.go           // Prometheus gauges, histograms
│   ├── checks/
│   │   ├── factories.go     // checker registry, init()
│   │   ├── http.go          // HTTPChecker
│   │   ├── grpc.go          // GRPCChecker
│   │   ├── tcp.go           // TCPChecker
│   │   ├── postgres.go      // PostgresChecker
│   │   ├── redis.go         // RedisChecker
│   │   ├── amqp.go          // AMQPChecker
│   │   └── kafka.go         // KafkaChecker
│   └── contrib/             // optional integrations
│       └── sqldb/           // database/sql integration
```

## Error Handling

### Sentinel Errors

Define package-level sentinel errors for common failure modes:

```go
var (
    ErrTimeout           = errors.New("health check timeout")
    ErrConnectionRefused = errors.New("connection refused")
    ErrUnhealthy         = errors.New("dependency unhealthy")
)
```

### Error Wrapping

Always wrap errors with context using `fmt.Errorf` and `%w`:

```go
// Good — wraps with context, preserves sentinel
func (c *HTTPChecker) Check(ctx context.Context, ep Endpoint) error {
    resp, err := c.client.Get(url)
    if err != nil {
        if errors.Is(err, context.DeadlineExceeded) {
            return fmt.Errorf("HTTP check %s:%d: %w", ep.Host, ep.Port, ErrTimeout)
        }
        return fmt.Errorf("HTTP check %s:%d: %w", ep.Host, ep.Port, err)
    }
    if resp.StatusCode >= 300 {
        return fmt.Errorf("HTTP check %s:%d: status %d: %w",
            ep.Host, ep.Port, resp.StatusCode, ErrUnhealthy)
    }
    return nil
}

// Bad — loses context
return err
return errors.New("failed")
```

### Rules

- **No `panic`** in library code — always return errors
- Check errors with `errors.Is()` and `errors.As()`, never compare strings
- Wrap all errors with `%w` to preserve the chain
- Error messages start lowercase, no trailing punctuation

## GoDoc

Comments follow the Go convention: start with the name of the symbol.

```go
// HealthChecker is the interface for dependency health checks.
// Each dependency type (HTTP, gRPC, TCP, Postgres, etc.) implements this interface.
type HealthChecker interface {
    // Check performs a health check against the given endpoint.
    // Returns nil if the endpoint is healthy, or an error describing the failure.
    // The context carries the timeout deadline.
    Check(ctx context.Context, endpoint Endpoint) error

    // Type returns the dependency type this checker handles (e.g. "http", "postgres").
    Type() string
}

// New creates a new DependencyHealth instance with the given service name and options.
// Returns an error if the configuration is invalid (e.g., empty service name).
func New(serviceName string, opts ...Option) (*DependencyHealth, error) { }
```

Rules:

- First sentence: starts with the symbol name (GoDoc convention)
- Complete sentences with periods
- Document all exported symbols
- Document non-obvious behavior and concurrency guarantees

## context.Context

`context.Context` is always the **first parameter** in functions that accept it:

```go
// Good
func (c *HTTPChecker) Check(ctx context.Context, endpoint Endpoint) error { }
func (s *Scheduler) Start(ctx context.Context) error { }

// Bad
func (c *HTTPChecker) Check(endpoint Endpoint, ctx context.Context) error { }
```

Rules:

- Never store `ctx` in a struct — pass it through the call chain
- Use `ctx` for cancellation and timeouts, not for passing values
- Respect context cancellation: check `ctx.Err()` in loops

## Functional Options

Use the functional options pattern for configuration:

```go
// Option configures DependencyHealth.
type Option func(*options)

type options struct {
    checkInterval time.Duration
    timeout       time.Duration
    registry      prometheus.Registerer
}

// WithCheckInterval sets the health check interval. Default: 15s.
func WithCheckInterval(d time.Duration) Option {
    return func(o *options) { o.checkInterval = d }
}

// WithTimeout sets the health check timeout. Default: 5s.
func WithTimeout(d time.Duration) Option {
    return func(o *options) { o.timeout = d }
}
```

Usage:

```go
dh, err := dephealth.New("order-service",
    dephealth.Postgres("postgres-main",
        dephealth.FromURL(os.Getenv("DATABASE_URL")),
        dephealth.Critical(true),
    ),
    dephealth.WithCheckInterval(30 * time.Second),
    dephealth.WithTimeout(10 * time.Second),
)
```

## defer for Cleanup

Use `defer` for resource cleanup. Place it immediately after resource acquisition:

```go
func (c *PostgresChecker) Check(ctx context.Context, ep Endpoint) error {
    conn, err := c.pool.Acquire(ctx)
    if err != nil {
        return fmt.Errorf("postgres check %s:%d: %w", ep.Host, ep.Port, err)
    }
    defer conn.Release()

    _, err = conn.Exec(ctx, "SELECT 1")
    if err != nil {
        return fmt.Errorf("postgres check %s:%d: %w", ep.Host, ep.Port, err)
    }
    return nil
}
```

Rules:

- `defer` immediately after successful acquisition
- Understand `defer` evaluation order (LIFO) when using multiple defers
- Be aware that `defer` in a loop defers until function exit, not loop iteration

## Linter

### golangci-lint v2

Configuration: `sdk-go/.golangci.yml`

Key linters enabled:

- `errcheck` — check that errors are handled
- `govet` — suspicious constructs
- `staticcheck` — advanced static analysis
- `revive` — style and naming
- `goimports` — import formatting
- `misspell` — typos in comments
- `gosec` — security issues

### Running

```bash
cd sdk-go && make lint    # golangci-lint in Docker
cd sdk-go && make fmt     # goimports + gofmt
```

## Additional Conventions

- **Go version**: 1.25+ — use `log/slog`, range-over-func where appropriate
- **Module path**: `github.com/BigKAA/topologymetrics/sdk-go`
- **Zero values**: design types so zero values are useful (or document when they're not)
- **Struct field order**: exported fields first, then unexported; group logically
- **No `init()` in library code** except for checker registration in `checks/factories.go`
- **Table-driven tests**: preferred style (see [Testing](testing.md))
- **Error strings**: lowercase, no trailing punctuation, no "failed to" prefix
- **Receiver names**: short (1-2 letters), consistent across methods of the same type

```go
// Good — short, consistent receiver
func (s *Scheduler) Start(ctx context.Context) error { }
func (s *Scheduler) Stop() { }
func (s *Scheduler) addDependency(d Dependency) { }

// Bad — long, inconsistent
func (scheduler *Scheduler) Start(ctx context.Context) error { }
func (sched *Scheduler) Stop() { }
```
