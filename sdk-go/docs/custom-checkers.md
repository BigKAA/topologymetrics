*[Русская версия](custom-checkers.ru.md)*

# Custom Checkers

You can implement your own health checker for any dependency type not
covered by the built-in checkers. This guide covers the `HealthChecker`
interface, error classification, and registration.

## HealthChecker Interface

```go
type HealthChecker interface {
    Check(ctx context.Context, endpoint Endpoint) error
    Type() string
}
```

- `Check()` — performs a health check against the given endpoint. Returns
  `nil` if healthy, or an error describing the failure. The context
  carries the timeout deadline.
- `Type()` — returns the dependency type string (e.g., `"elasticsearch"`).

## Basic Example: Elasticsearch Checker

```go
package escheck

import (
    "context"
    "fmt"
    "net/http"

    "github.com/BigKAA/topologymetrics/sdk-go/dephealth"
)

// Checker implements dephealth.HealthChecker for Elasticsearch.
type Checker struct{}

func New() *Checker {
    return &Checker{}
}

func (c *Checker) Type() string {
    return "elasticsearch"
}

func (c *Checker) Check(ctx context.Context, endpoint dephealth.Endpoint) error {
    url := fmt.Sprintf("http://%s:%s/_cluster/health", endpoint.Host, endpoint.Port)

    req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
    if err != nil {
        return fmt.Errorf("create request: %w", err)
    }

    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return fmt.Errorf("elasticsearch health check: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("elasticsearch returned status %d", resp.StatusCode)
    }

    return nil
}
```

### Registering the Custom Checker

Use `AddDependency()` to register a custom checker:

```go
package main

import (
    "context"
    "log"
    "net/http"

    "github.com/BigKAA/topologymetrics/sdk-go/dephealth"
    "github.com/prometheus/client_golang/prometheus/promhttp"

    "myapp/escheck"
)

func main() {
    esChecker := escheck.New()

    dh, err := dephealth.New("my-service", "my-team",
        dephealth.AddDependency("elasticsearch", "elasticsearch", esChecker,
            dephealth.FromParams("es.svc", "9200"),
            dephealth.Critical(true),
        ),
    )
    if err != nil {
        log.Fatal(err)
    }

    ctx := context.Background()
    if err := dh.Start(ctx); err != nil {
        log.Fatal(err)
    }
    defer dh.Stop()

    http.Handle("/metrics", promhttp.Handler())
    log.Fatal(http.ListenAndServe(":8080", nil))
}
```

The second argument to `AddDependency()` is the dependency type string
(appears as the `type` label in metrics). You can use any string.

## Error Classification

By default, errors from `Check()` are classified by the core classifier
(timeouts, DNS errors, TLS errors, connection refused). To provide
protocol-specific classification, implement the `ClassifiedError` interface
or return a `ClassifiedCheckError`.

### ClassifiedError Interface

```go
type ClassifiedError interface {
    error
    StatusCategory() dephealth.StatusCategory
    StatusDetail() string
}
```

### ClassifiedCheckError Struct

The SDK provides a ready-to-use implementation:

```go
type ClassifiedCheckError struct {
    Category dephealth.StatusCategory
    Detail   string
    Cause    error
}
```

Methods:

- `Error() string` — returns the cause error message
- `Unwrap() error` — returns the cause for `errors.Is`/`errors.As`
- `StatusCategory() StatusCategory` — returns the status category
- `StatusDetail() string` — returns the detail string

### Available Status Categories

```go
const (
    StatusOK              StatusCategory = "ok"
    StatusTimeout         StatusCategory = "timeout"
    StatusConnectionError StatusCategory = "connection_error"
    StatusDNSError        StatusCategory = "dns_error"
    StatusAuthError       StatusCategory = "auth_error"
    StatusTLSError        StatusCategory = "tls_error"
    StatusUnhealthy       StatusCategory = "unhealthy"
    StatusError           StatusCategory = "error"
)
```

### Example: Checker with Error Classification

```go
func (c *Checker) Check(ctx context.Context, endpoint dephealth.Endpoint) error {
    url := fmt.Sprintf("http://%s:%s/_cluster/health", endpoint.Host, endpoint.Port)

    req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
    if err != nil {
        return fmt.Errorf("create request: %w", err)
    }

    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        // Network errors will be classified by the core classifier
        return fmt.Errorf("elasticsearch check: %w", err)
    }
    defer resp.Body.Close()

    // Auth errors
    if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
        return &dephealth.ClassifiedCheckError{
            Category: dephealth.StatusAuthError,
            Detail:   "auth_error",
            Cause:    fmt.Errorf("elasticsearch returned %d", resp.StatusCode),
        }
    }

    // Unhealthy but reachable
    if resp.StatusCode != http.StatusOK {
        return &dephealth.ClassifiedCheckError{
            Category: dephealth.StatusUnhealthy,
            Detail:   fmt.Sprintf("es_%d", resp.StatusCode),
            Cause:    fmt.Errorf("elasticsearch returned %d", resp.StatusCode),
        }
    }

    return nil
}
```

### Classification Priority

The core classifier checks errors in this order:

1. **ClassifiedError interface** — highest priority (your custom classification)
2. **Sentinel errors** — `ErrTimeout`, `ErrConnectionRefused`, `ErrUnhealthy`
3. **Platform errors** — `context.DeadlineExceeded`, `*net.DNSError`,
   `*net.OpError` (ECONNREFUSED), `*tls.CertificateVerificationError`
4. **Fallback** — `StatusError` with detail `"error"`

## Registering a Checker Factory

If you want to use your custom checker with the URL-based API
(`dephealth.New()` with a custom type), register a factory:

```go
package escheck

import "github.com/BigKAA/topologymetrics/sdk-go/dephealth"

const TypeElasticsearch dephealth.DependencyType = "elasticsearch"

func init() {
    dephealth.RegisterCheckerFactory(TypeElasticsearch, NewFromConfig)
}

func NewFromConfig(dc *dephealth.DependencyConfig) dephealth.HealthChecker {
    return New()
}
```

After importing this package, you could create a helper option function:

```go
func Elasticsearch(name string, opts ...dephealth.DependencyOption) dephealth.Option {
    return dephealth.AddDependency(name, TypeElasticsearch, New(), opts...)
}
```

Usage:

```go
import "myapp/escheck"

dh, err := dephealth.New("my-service", "my-team",
    escheck.Elasticsearch("es-cluster",
        dephealth.FromParams("es.svc", "9200"),
        dephealth.Critical(true),
    ),
)
```

## Testing Custom Checkers

```go
package escheck_test

import (
    "context"
    "testing"

    "github.com/BigKAA/topologymetrics/sdk-go/dephealth"
    "myapp/escheck"
)

func TestChecker_Type(t *testing.T) {
    c := escheck.New()
    if c.Type() != "elasticsearch" {
        t.Errorf("expected type elasticsearch, got %s", c.Type())
    }
}

func TestChecker_Check_Success(t *testing.T) {
    // Start a test HTTP server that responds with 200
    // ...

    c := escheck.New()
    err := c.Check(context.Background(), dephealth.Endpoint{
        Host: "localhost",
        Port: "9200",
    })
    if err != nil {
        t.Errorf("unexpected error: %v", err)
    }
}

func TestChecker_Check_AuthError(t *testing.T) {
    // Start a test HTTP server that responds with 401
    // ...

    c := escheck.New()
    err := c.Check(context.Background(), dephealth.Endpoint{
        Host: "localhost",
        Port: "9200",
    })

    var ce dephealth.ClassifiedError
    if !errors.As(err, &ce) {
        t.Fatal("expected ClassifiedError")
    }
    if ce.StatusCategory() != dephealth.StatusAuthError {
        t.Errorf("expected auth_error, got %s", ce.StatusCategory())
    }
}
```

## See Also

- [Checkers](checkers.md) — built-in checkers reference
- [Selective Imports](selective-imports.md) — factory registration via `init()`
- [API Reference](api-reference.md) — `HealthChecker`, `ClassifiedError`, `AddDependency`
- [Troubleshooting](troubleshooting.md) — common issues and solutions
