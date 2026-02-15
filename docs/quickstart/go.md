*[Русская версия](go.ru.md)*

# Quick Start: Go SDK

Guide to integrating dephealth into a Go service in just a few minutes.

## Installation

```bash
go get github.com/BigKAA/topologymetrics/sdk-go@latest
```

## Minimal Example

Connecting a single HTTP dependency with metrics export:

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

    _ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks" // register checkers
)

func main() {
    dh, err := dephealth.New("my-service",
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
    go http.ListenAndServe(":8080", nil)

    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
    <-sigCh
}
```

After startup, metrics will appear on `/metrics`:

```text
app_dependency_health{name="my-service",dependency="payment-api",type="http",host="payment.svc",port="8080",critical="yes"} 1
app_dependency_latency_seconds_bucket{name="my-service",dependency="payment-api",type="http",host="payment.svc",port="8080",critical="yes",le="0.01"} 42
app_dependency_status{name="my-service",dependency="payment-api",type="http",host="payment.svc",port="8080",critical="yes",status="healthy"} 1
app_dependency_status_detail{name="my-service",dependency="payment-api",type="http",host="payment.svc",port="8080",critical="yes",detail=""} 1
```

## Multiple Dependencies

```go
dh, err := dephealth.New("my-service",
    // Global settings
    dephealth.WithCheckInterval(30 * time.Second),
    dephealth.WithTimeout(3 * time.Second),

    // PostgreSQL — standalone check (new connection)
    dephealth.Postgres("postgres-main",
        dephealth.FromURL(os.Getenv("DATABASE_URL")),
        dephealth.Critical(true),
    ),

    // Redis — standalone check
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

    // RabbitMQ
    dephealth.AMQP("rabbitmq",
        dephealth.FromParams("rabbitmq.svc", "5672"),
        dephealth.WithAMQPURL("amqp://user:pass@rabbitmq.svc:5672/"),
        dephealth.Critical(false),
    ),

    // Kafka
    dephealth.Kafka("kafka",
        dephealth.FromParams("kafka.svc", "9092"),
        dephealth.Critical(false),
    ),
)
```

## Custom Labels

Add custom labels using `WithLabel`:

```go
dephealth.Postgres("postgres-main",
    dephealth.FromURL(os.Getenv("DATABASE_URL")),
    dephealth.Critical(true),
    dephealth.WithLabel("role", "primary"),
    dephealth.WithLabel("shard", "eu-west"),
)
```

Result in metrics:

```text
app_dependency_health{name="my-service",dependency="postgres-main",type="postgres",host="pg.svc",port="5432",critical="yes",role="primary",shard="eu-west"} 1
```

## Connection Pool Integration (contrib)

Preferred mode: SDK uses the existing connection pool of the service
instead of creating new connections. This reflects the actual
ability of the service to work with the dependency.

### PostgreSQL via `*sql.DB`

```go
import (
    "database/sql"

    "github.com/BigKAA/topologymetrics/sdk-go/dephealth"
    "github.com/BigKAA/topologymetrics/sdk-go/dephealth/contrib/sqldb"
    _ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks"
    _ "github.com/jackc/pgx/v5/stdlib"
)

// Use existing connection pool
db, _ := sql.Open("pgx", os.Getenv("DATABASE_URL"))

dh, err := dephealth.New("my-service",
    sqldb.FromDB("postgres-main", db,
        dephealth.FromURL(os.Getenv("DATABASE_URL")),
        dephealth.Critical(true),
    ),
)
```

### MySQL via `*sql.DB`

```go
import "github.com/BigKAA/topologymetrics/sdk-go/dephealth/contrib/sqldb"

db, _ := sql.Open("mysql", "user:pass@tcp(mysql.svc:3306)/mydb")

dh, err := dephealth.New("my-service",
    sqldb.FromMySQLDB("mysql-main", db,
        dephealth.FromParams("mysql.svc", "3306"),
        dephealth.Critical(true),
    ),
)
```

### Redis via go-redis `*redis.Client`

```go
import (
    "github.com/BigKAA/topologymetrics/sdk-go/dephealth/contrib/redispool"
    "github.com/redis/go-redis/v9"
)

client := redis.NewClient(&redis.Options{Addr: "redis.svc:6379"})

dh, err := dephealth.New("my-service",
    // Host and port are extracted automatically from client.Options().Addr
    redispool.FromClient("redis-cache", client,
        dephealth.Critical(false),
    ),
)
```

## Global Options

```go
dh, err := dephealth.New("my-service",
    // Check interval (default 15s)
    dephealth.WithCheckInterval(30 * time.Second),

    // Timeout for each check (default 5s)
    dephealth.WithTimeout(3 * time.Second),

    // Custom Prometheus Registerer
    dephealth.WithRegisterer(customRegisterer),

    // Logger (slog)
    dephealth.WithLogger(slog.Default()),

    // ...dependencies
)
```

## Dependency Options

Each dependency can override global settings:

```go
dephealth.HTTP("slow-service",
    dephealth.FromURL("http://slow.svc:8080"),
    dephealth.CheckInterval(60 * time.Second),  // own interval
    dephealth.Timeout(10 * time.Second),         // own timeout
    dephealth.Critical(true),                    // critical dependency
    dephealth.WithHTTPHealthPath("/ready"),       // health check path
    dephealth.WithHTTPTLS(true),                  // TLS
    dephealth.WithHTTPTLSSkipVerify(true),        // skip certificate verification
)
```

## Configuration via Environment Variables

| Variable | Description | Example |
| --- | --- | --- |
| `DEPHEALTH_NAME` | Application name (overridden by API argument) | `my-service` |
| `DEPHEALTH_<DEP>_CRITICAL` | Dependency criticality | `yes` / `no` |
| `DEPHEALTH_<DEP>_LABEL_<KEY>` | Custom label | `primary` |

`<DEP>` — dependency name in uppercase, hyphens replaced with `_`.

Examples:

```bash
export DEPHEALTH_NAME=my-service
export DEPHEALTH_POSTGRES_MAIN_CRITICAL=yes
export DEPHEALTH_POSTGRES_MAIN_LABEL_ROLE=primary
```

Priority: API values > environment variables.

## Behavior When Required Parameters Are Missing

| Situation | Behavior |
| --- | --- |
| `name` not specified and no `DEPHEALTH_NAME` | Creation error: `missing name` |
| `Critical()` not specified for dependency | Creation error: `missing critical` |
| Invalid label name | Creation error: `invalid label name` |
| Label conflicts with required label | Creation error: `reserved label` |

## Checking Dependency Status

The `Health()` method returns the current status of all endpoints:

```go
health := dh.Health()
// map[string]bool{
//   "postgres-main:pg.svc:5432":   true,
//   "redis-cache:redis.svc:6379":  true,
//   "auth-service:auth.svc:8080":  false,
// }

// Usage for Kubernetes readiness probe
allHealthy := true
for _, ok := range health {
    if !ok {
        allHealthy = false
        break
    }
}
```

## Detailed Health Status

The `HealthDetails()` method returns detailed information about each endpoint,
including status category, failure detail, latency, and custom labels:

```go
details := dh.HealthDetails()
// map[string]dephealth.EndpointStatus{
//   "postgres-main:pg.svc:5432": {
//     Dependency:    "postgres-main",
//     Type:          "postgres",
//     Host:          "pg.svc",
//     Port:          "5432",
//     Healthy:       boolPtr(true),
//     Status:        "ok",
//     Detail:        "ok",
//     Latency:       15 * time.Millisecond,
//     LastCheckedAt: time.Now(),
//     Critical:      true,
//     Labels:        map[string]string{"role": "primary"},
//   },
// }

// Encode as JSON for HTTP endpoint
json.NewEncoder(w).Encode(details)
```

Unlike `Health()` which returns `map[string]bool`, `HealthDetails()` provides
the full `EndpointStatus` struct for each endpoint. Before the first check
completes, `Healthy` is `nil` (unknown) and `Status` is `"unknown"`.

## Metrics Export

dephealth exports four Prometheus metrics:

| Metric | Type | Description |
| --- | --- | --- |
| `app_dependency_health` | Gauge | `1` = available, `0` = unavailable |
| `app_dependency_latency_seconds` | Histogram | Check latency (seconds) |
| `app_dependency_status` | Gauge (enum) | Status category: 8 series per endpoint, exactly one = 1 |
| `app_dependency_status_detail` | Gauge (info) | Detailed reason: e.g. `http_503`, `auth_error` |

Labels: `name`, `dependency`, `type`, `host`, `port`, `critical`.
Additional: `status` (on `app_dependency_status`), `detail` (on `app_dependency_status_detail`).

For export, use the standard `promhttp.Handler()`:

```go
http.Handle("/metrics", promhttp.Handler())
```

## Supported Dependency Types

| Function | Type | Check Method |
| --- | --- | --- |
| `dephealth.HTTP()` | `http` | HTTP GET to health endpoint, expecting 2xx |
| `dephealth.GRPC()` | `grpc` | gRPC Health Check Protocol |
| `dephealth.TCP()` | `tcp` | TCP connection establishment |
| `dephealth.Postgres()` | `postgres` | `SELECT 1` |
| `dephealth.MySQL()` | `mysql` | `SELECT 1` |
| `dephealth.Redis()` | `redis` | `PING` command |
| `dephealth.AMQP()` | `amqp` | Connection check with broker |
| `dephealth.Kafka()` | `kafka` | Metadata request to broker |

## Default Parameters

| Parameter | Value | Description |
| --- | --- | --- |
| `checkInterval` | 15s | Interval between checks |
| `timeout` | 5s | Timeout for a single check |
| `failureThreshold` | 1 | Number of failures before transitioning to unhealthy |
| `successThreshold` | 1 | Number of successes before transitioning to healthy |

## Next Steps

- [Integration Guide](../migration/go.md) — step-by-step integration
  with an existing service
- [Specification Overview](../specification.md) — details of metric contracts and behavior
