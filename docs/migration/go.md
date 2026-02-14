*[Русская версия](go.ru.md)*

# Integration Guide: dephealth in an Existing Go Service

Step-by-step instructions for adding dependency monitoring
to a running microservice.

## Migration to v0.4.0

### New Status Metrics (no code changes required)

v0.4.0 adds two new automatically exported Prometheus metrics:

| Metric | Type | Description |
| --- | --- | --- |
| `app_dependency_status` | Gauge (enum) | Status category: 8 series per endpoint, exactly one = 1 |
| `app_dependency_status_detail` | Gauge (info) | Detailed failure reason: e.g. `http_503`, `auth_error` |

**No code changes are needed** — the SDK exports these metrics automatically alongside the existing `app_dependency_health` and `app_dependency_latency_seconds`.

### Storage Impact

Each endpoint now produces 9 additional time series (8 for `app_dependency_status` + 1 for `app_dependency_status_detail`). For a service with 5 endpoints, this adds 45 series.

### New PromQL Queries

```promql
# Status category for a dependency
app_dependency_status{dependency="postgres-main", status!=""} == 1

# Detailed failure reason
app_dependency_status_detail{dependency="postgres-main", detail!=""} == 1

# Alert on authentication errors
app_dependency_status{status="auth_error"} == 1
```

For the full list of status values, see [Specification — Status Metrics](../specification.md).

## Migration from v0.2 to v0.3

### Breaking change: new module path

In v0.3.0, the module path has changed from `github.com/BigKAA/topologymetrics`
to `github.com/BigKAA/topologymetrics/sdk-go`.

This fixes `go get` functionality — the standard approach for Go modules
in monorepos where `go.mod` is located in a subdirectory.

### Migration steps

1. Update the dependency:

```bash
go get github.com/BigKAA/topologymetrics/sdk-go@latest
```

1. Replace import paths in all files:

```bash
# Bulk replacement (Linux/macOS)
find . -name '*.go' -exec sed -i '' \
  's|github.com/BigKAA/topologymetrics/dephealth|github.com/BigKAA/topologymetrics/sdk-go/dephealth|g' {} +
```

1. Update `go.mod` — remove the old dependency:

```bash
go mod tidy
```

### Import replacement examples

```go
// v0.2
import (
    "github.com/BigKAA/topologymetrics/sdk-go/dephealth"
    "github.com/BigKAA/topologymetrics/sdk-go/dephealth/contrib/sqldb"
    _ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks"
)
```

The API and SDK behavior remain unchanged — only the module path has changed.

---

## Migration from v0.1 to v0.2

### API Changes

| v0.1 | v0.2 | Description |
| --- | --- | --- |
| `dephealth.New(...)` | `dephealth.New("my-service", ...)` | Required first argument `name` |
| `dephealth.Critical(true)` (optional) | `dephealth.Critical(true/false)` (required) | For each dependency |
| `Endpoint.Metadata` | `Endpoint.Labels` | Field renamed |
| `dephealth.WithMetadata(map)` | `dephealth.WithLabel("key", "value")` | Custom labels |
| `WithOptionalLabels(...)` | removed | Custom labels via `WithLabel` |

### Required Changes

1. Add `name` as the first argument to `dephealth.New()`:

```go
// v0.1
dh, err := dephealth.New(
    dephealth.Postgres("postgres-main", ...),
)

// v0.2
dh, err := dephealth.New("my-service",
    dephealth.Postgres("postgres-main", ...),
)
```

1. Specify `Critical()` for each dependency:

```go
// v0.1 — Critical is optional
dephealth.Redis("redis-cache",
    dephealth.FromURL(os.Getenv("REDIS_URL")),
)

// v0.2 — Critical is required
dephealth.Redis("redis-cache",
    dephealth.FromURL(os.Getenv("REDIS_URL")),
    dephealth.Critical(false),
)
```

1. Replace `WithMetadata` with `WithLabel` (if used):

```go
// v0.1
dephealth.WithMetadata(map[string]string{"role": "primary"})

// v0.2
dephealth.WithLabel("role", "primary")
```

### New metric labels

```text
# v0.1
app_dependency_health{dependency="postgres-main",type="postgres",host="pg.svc",port="5432"} 1

# v0.2
app_dependency_health{name="my-service",dependency="postgres-main",type="postgres",host="pg.svc",port="5432",critical="yes"} 1
```

Update PromQL queries and Grafana dashboards to include the `name` and `critical` labels.

## Prerequisites

- Go 1.21+
- Service already exports Prometheus metrics via `promhttp.Handler()`
- Access to dependencies (databases, caches, other services) from the service

## Step 1. Install Dependencies

```bash
go get github.com/BigKAA/topologymetrics/sdk-go@latest
```

## Step 2. Import Packages

Add imports to your service initialization file:

```go
import (
    "github.com/BigKAA/topologymetrics/sdk-go/dephealth"

    // Register built-in checkers — required blank import
    _ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks"
)
```

If using connection pool integration (recommended):

```go
import (
    "github.com/BigKAA/topologymetrics/sdk-go/dephealth/contrib/sqldb"     // for *sql.DB
    "github.com/BigKAA/topologymetrics/sdk-go/dephealth/contrib/redispool"  // for *redis.Client
)
```

## Step 3. Create DepHealth Instance

### Option A: Standalone mode (simple)

The SDK creates temporary connections for health checks. Suitable for HTTP/gRPC
services and situations where connection pools are unavailable.

```go
func initDepHealth() (*dephealth.DepHealth, error) {
    return dephealth.New("my-service",
        dephealth.Postgres("postgres-main",
            dephealth.FromURL(os.Getenv("DATABASE_URL")),
            dephealth.Critical(true),
        ),
        dephealth.Redis("redis-cache",
            dephealth.FromURL(os.Getenv("REDIS_URL")),
            dephealth.Critical(false),
        ),
        dephealth.HTTP("payment-api",
            dephealth.FromURL(os.Getenv("PAYMENT_SERVICE_URL")),
            dephealth.Critical(true),
        ),
    )
}
```

### Option B: Connection pool integration (recommended)

The SDK uses the service's existing connections. Benefits:

- Reflects the service's actual ability to work with dependencies
- Does not create additional load on databases/caches
- Detects pool-related issues (exhaustion, leaks)

```go
func initDepHealth(db *sql.DB, rdb *redis.Client) (*dephealth.DepHealth, error) {
    return dephealth.New("my-service",
        dephealth.WithCheckInterval(15 * time.Second),
        dephealth.WithLogger(slog.Default()),

        // PostgreSQL via existing *sql.DB
        sqldb.FromDB("postgres-main", db,
            dephealth.FromURL(os.Getenv("DATABASE_URL")),
            dephealth.Critical(true),
        ),

        // Redis via existing *redis.Client
        // Host:port extracted automatically
        redispool.FromClient("redis-cache", rdb,
            dephealth.Critical(false),
        ),

        // For HTTP/gRPC — standalone only
        dephealth.HTTP("payment-api",
            dephealth.FromURL(os.Getenv("PAYMENT_SERVICE_URL")),
            dephealth.Critical(true),
        ),

        dephealth.GRPC("auth-service",
            dephealth.FromParams(os.Getenv("AUTH_HOST"), os.Getenv("AUTH_PORT")),
            dephealth.Critical(true),
        ),
    )
}
```

## Step 4. Start and Stop

Integrate `dh.Start()` and `dh.Stop()` into your service lifecycle:

```go
func main() {
    // ... initialize DB, Redis, etc. ...

    dh, err := initDepHealth(db, rdb)
    if err != nil {
        log.Fatal(err)
    }

    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    if err := dh.Start(ctx); err != nil {
        log.Fatal(err)
    }

    // ... start HTTP server ...

    // Graceful shutdown
    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
    <-sigCh

    dh.Stop() // stop checks before server shutdown
    server.Shutdown(context.Background())
}
```

## Step 5. Dependency Status Endpoint (optional)

Add an endpoint for Kubernetes readiness probe or debugging:

```go
func handleDependencies(dh *dephealth.DepHealth) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        health := dh.Health()

        w.Header().Set("Content-Type", "application/json")

        // If there are unhealthy dependencies — return 503
        for _, ok := range health {
            if !ok {
                w.WriteHeader(http.StatusServiceUnavailable)
                json.NewEncoder(w).Encode(health)
                return
            }
        }

        w.WriteHeader(http.StatusOK)
        json.NewEncoder(w).Encode(health)
    }
}

// Register
mux.HandleFunc("/health/dependencies", handleDependencies(dh))
```

## Typical Configurations

### Web service with PostgreSQL and Redis

```go
dh, _ := dephealth.New("my-service",
    sqldb.FromDB("postgres", db,
        dephealth.FromURL(os.Getenv("DATABASE_URL")),
        dephealth.Critical(true),
    ),
    redispool.FromClient("redis", rdb,
        dephealth.Critical(false),
    ),
)
```

### API Gateway with upstream services

```go
dh, _ := dephealth.New("api-gateway",
    dephealth.WithCheckInterval(10 * time.Second),

    dephealth.HTTP("user-service",
        dephealth.FromURL("http://user-svc:8080"),
        dephealth.WithHTTPHealthPath("/healthz"),
        dephealth.Critical(true),
    ),
    dephealth.HTTP("order-service",
        dephealth.FromURL("http://order-svc:8080"),
        dephealth.Critical(true),
    ),
    dephealth.GRPC("auth-service",
        dephealth.FromParams("auth-svc", "9090"),
        dephealth.Critical(true),
    ),
)
```

### Event processor with Kafka and RabbitMQ

```go
dh, _ := dephealth.New("event-processor",
    dephealth.Kafka("kafka-main",
        dephealth.FromParams("kafka.svc", "9092"),
        dephealth.Critical(true),
    ),
    dephealth.AMQP("rabbitmq",
        dephealth.FromParams("rabbitmq.svc", "5672"),
        dephealth.WithAMQPURL("amqp://user:pass@rabbitmq.svc:5672/"),
        dephealth.Critical(true),
    ),
    sqldb.FromDB("postgres", db,
        dephealth.FromURL(os.Getenv("DATABASE_URL")),
        dephealth.Critical(false),
    ),
)
```

### Service with TLS dependencies

```go
dh, _ := dephealth.New("my-service",
    dephealth.HTTP("external-api",
        dephealth.FromURL("https://api.example.com"),
        dephealth.WithHTTPHealthPath("/status"),
        dephealth.Timeout(10 * time.Second),
        dephealth.Critical(true),
        // TLS enabled automatically for https://
    ),
    dephealth.GRPC("secure-service",
        dephealth.FromParams("secure.svc", "443"),
        dephealth.WithGRPCTLS(true),
        dephealth.WithGRPCTLSSkipVerify(true), // for self-signed certificates
        dephealth.Critical(false),
    ),
)
```

## Troubleshooting

### `no checker factory registered for type "..."`

**Cause**: the `checks` package is not imported.

**Solution**: add blank import:

```go
import _ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks"
```

### Metrics do not appear on `/metrics`

**Check:**

1. `dh.Start(ctx)` called without errors
2. `promhttp.Handler()` registered on the `/metrics` route
3. Enough time has passed for the first check (default `initialDelay` = 0
   in the public API, first check runs immediately)

### All dependencies show `0` (unhealthy)

**Check:**

1. Network accessibility of dependencies from the service container/pod
2. DNS resolution of service names
3. Correct URL/host/port in configuration
4. Timeout (default `5s`) — is it sufficient for this dependency
5. Logs: `dephealth.WithLogger(slog.Default())` will show error causes

### High latency in PostgreSQL/MySQL checks

**Cause**: standalone mode creates a new connection each time.

**Solution**: use the contrib module `sqldb.FromDB()` with an existing
connection pool. This eliminates the connection setup overhead.

### gRPC: error `context deadline exceeded`

**Check:**

1. gRPC service is accessible at the specified address
2. Service implements `grpc.health.v1.Health/Check`
3. For gRPC use `FromParams(host, port)`, not `FromURL()` —
   the URL parser may incorrectly handle bare `host:port`
4. If TLS is needed: `dephealth.WithGRPCTLS(true)`

### AMQP: connection error to RabbitMQ

**For AMQP, you must provide the full URL via `WithAMQPURL()`:**

```go
dephealth.AMQP("rabbitmq",
    dephealth.FromParams("rabbitmq.svc", "5672"),   // for metric labels
    dephealth.WithAMQPURL("amqp://user:pass@rabbitmq.svc:5672/vhost"),
    dephealth.Critical(false),
)
```

`FromParams` defines the `host`/`port` metric labels, while `WithAMQPURL` defines
the connection string with credentials.

### Dependency Naming

Names must conform to the following rules:

- Length: 1-63 characters
- Format: `[a-z][a-z0-9-]*` (lowercase letters, digits, hyphens)
- Starts with a letter
- Examples: `postgres-main`, `redis-cache`, `auth-service`

## Next Steps

- [Quick Start](../quickstart/go.md) — minimal examples
- [Specification Overview](../specification.md) — details on metrics contracts and behavior
