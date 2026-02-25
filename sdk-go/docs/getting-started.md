*[Русская версия](getting-started.ru.md)*

# Getting Started

This guide covers installation, basic setup, and your first health check
with the dephealth Go SDK.

## Prerequisites

- Go 1.21 or later
- A running dependency to monitor (PostgreSQL, Redis, HTTP service, etc.)

## Installation

```bash
go get github.com/BigKAA/topologymetrics/sdk-go@latest
```

The module path is `github.com/BigKAA/topologymetrics/sdk-go/dephealth`.

## Checker Registration

Before creating a `DepHealth` instance, you must register the checker
factories for the dependency types you plan to monitor. There are two ways:

**Import all checkers at once:**

```go
import _ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks"
```

**Import only what you need (reduces binary size):**

```go
import (
    _ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/httpcheck"
    _ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/pgcheck"
)
```

See [Selective Imports](selective-imports.md) for details on binary size
optimization and available sub-packages.

## Minimal Example

Monitor a single HTTP dependency and expose Prometheus metrics:

```go
package main

import (
    "context"
    "log"
    "net/http"
    "os"
    "os/signal"
    "syscall"

    "github.com/BigKAA/topologymetrics/sdk-go/dephealth"
    "github.com/prometheus/client_golang/prometheus/promhttp"

    // Register HTTP checker factory
    _ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/httpcheck"
)

func main() {
    dh, err := dephealth.New("my-service", "my-team",
        dephealth.HTTP("payment-api",
            dephealth.FromURL("http://payment.svc:8080"),
            dephealth.Critical(true),
        ),
    )
    if err != nil {
        log.Fatal(err)
    }

    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    if err := dh.Start(ctx); err != nil {
        log.Fatal(err)
    }
    defer dh.Stop()

    http.Handle("/metrics", promhttp.Handler())
    go func() {
        log.Fatal(http.ListenAndServe(":8080", nil))
    }()

    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
    <-sigCh
}
```

After startup, Prometheus metrics appear at `http://localhost:8080/metrics`:

```text
app_dependency_health{name="my-service",group="my-team",dependency="payment-api",type="http",host="payment.svc",port="8080",critical="yes"} 1
app_dependency_latency_seconds_bucket{name="my-service",group="my-team",dependency="payment-api",type="http",host="payment.svc",port="8080",critical="yes",le="0.01"} 42
```

## Key Concepts

### Name and Group

Every `DepHealth` instance requires two identifiers:

- **name** — unique application name (e.g., `"my-service"`)
- **group** — logical group the service belongs to (e.g., `"my-team"`, `"payments"`)

Both appear as labels in all exported metrics. Validation rules:
`[a-z][a-z0-9-]*`, 1-63 characters.

If not passed as arguments, the SDK reads `DEPHEALTH_NAME` and
`DEPHEALTH_GROUP` environment variables as fallback.

### Dependencies

Each dependency is registered via a factory function matching its type:

| Function | Dependency type |
| --- | --- |
| `dephealth.HTTP()` | HTTP service |
| `dephealth.GRPC()` | gRPC service |
| `dephealth.TCP()` | TCP endpoint |
| `dephealth.Postgres()` | PostgreSQL database |
| `dephealth.MySQL()` | MySQL database |
| `dephealth.Redis()` | Redis server |
| `dephealth.AMQP()` | RabbitMQ (AMQP broker) |
| `dephealth.Kafka()` | Apache Kafka broker |

Each dependency requires:

- A **name** (first argument) — identifies the dependency in metrics
- **Endpoint** — specified via `FromURL()` or `FromParams()`
- **Critical** flag — `Critical(true)` or `Critical(false)` (mandatory)

### Lifecycle

1. **Create** — `dephealth.New(name, group, opts...)`
2. **Start** — `dh.Start(ctx)` launches periodic health checks
3. **Run** — checks execute at the configured interval (default 15s)
4. **Stop** — `dh.Stop()` stops checks and waits for goroutines to finish

## Multiple Dependencies

```go
dh, err := dephealth.New("my-service", "my-team",
    // Global settings
    dephealth.WithCheckInterval(30 * time.Second),
    dephealth.WithTimeout(3 * time.Second),

    // PostgreSQL
    dephealth.Postgres("postgres-main",
        dephealth.FromURL(os.Getenv("DATABASE_URL")),
        dephealth.Critical(true),
    ),

    // Redis
    dephealth.Redis("redis-cache",
        dephealth.FromURL(os.Getenv("REDIS_URL")),
        dephealth.Critical(false),
    ),

    // HTTP service
    dephealth.HTTP("auth-service",
        dephealth.FromURL("http://auth.svc:8080"),
        dephealth.WithHTTPHealthPath("/healthz"),
        dephealth.Critical(true),
    ),

    // gRPC service
    dephealth.GRPC("user-service",
        dephealth.FromParams("user.svc", "9090"),
        dephealth.Critical(false),
    ),
)
```

## Checking Health Status

### Simple Status

```go
health := dh.Health()
// map[string]bool{
//   "postgres-main:pg.svc:5432":  true,
//   "redis-cache:redis.svc:6379": true,
//   "auth-service:auth.svc:8080": false,
// }
```

### Detailed Status

```go
details := dh.HealthDetails()
for key, ep := range details {
    fmt.Printf("%s: healthy=%v status=%s latency=%v\n",
        key, *ep.Healthy, ep.Status, ep.Latency)
}
```

`HealthDetails()` returns an `EndpointStatus` struct with health state,
status category, latency, timestamps, and custom labels. Before the first
check completes, `Healthy` is `nil` and `Status` is `"unknown"`.

## Next Steps

- [Checkers](checkers.md) — detailed guide for all 8 built-in checkers
- [Configuration](configuration.md) — all options, defaults, and environment variables
- [Connection Pools](connection-pools.md) — integration with existing connection pools
- [Authentication](authentication.md) — auth for HTTP, gRPC, and database checkers
- [Metrics](metrics.md) — Prometheus metrics reference and PromQL examples
- [API Reference](api-reference.md) — complete reference of all public symbols
- [Troubleshooting](troubleshooting.md) — common issues and solutions
- [Migration Guide](migration.md) — version upgrade instructions
- [Code Style](code-style.md) — Go code conventions for this project
- [Examples](examples/) — complete runnable examples
