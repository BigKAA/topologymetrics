# dephealth

Go SDK for monitoring microservice dependencies via Prometheus metrics.

## Features

- Automatic health checking for dependencies (PostgreSQL, MySQL, Redis, RabbitMQ, Kafka, HTTP, gRPC, TCP)
- Prometheus metrics export: `app_dependency_health` (Gauge 0/1), `app_dependency_latency_seconds` (Histogram), `app_dependency_status` (enum), `app_dependency_status_detail` (info)
- Connection pool support (preferred) and standalone checks
- Functional options pattern for configuration
- contrib/ packages for popular drivers (pgx, go-redis, go-sql-driver)

## Installation

```bash
go get github.com/BigKAA/topologymetrics/sdk-go/dephealth
```

## Quick Start

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

    // Register checker factories
    _ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/pgcheck"
    _ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/redischeck"
)

func main() {
    dh, err := dephealth.New("my-service", "my-team",
        dephealth.Postgres("postgres",
            dephealth.FromURL("postgresql://user:pass@localhost:5432/mydb"),
            dephealth.Critical(true),
        ),
        dephealth.Redis("redis",
            dephealth.FromURL("redis://localhost:6379"),
            dephealth.Critical(false),
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

## Configuration

| Option | Default | Description |
| --- | --- | --- |
| `WithCheckInterval(d)` | `15s` | Interval between health checks |
| `WithTimeout(d)` | `5s` | Timeout for a single check |
| `WithRegisterer(r)` | default | Custom Prometheus registerer |
| `WithLogger(l)` | none | `*slog.Logger` for SDK operations |

## Supported Dependencies

| Type | URL Format |
| --- | --- |
| PostgreSQL | `postgresql://user:pass@host:5432/db` |
| MySQL | `mysql://user:pass@host:3306/db` |
| Redis | `redis://host:6379` |
| RabbitMQ | `amqp://user:pass@host:5672/vhost` |
| Kafka | `kafka://host1:9092,host2:9092` |
| HTTP | `http://host:8080/health` |
| gRPC | via `dephealth.FromParams(host, port)` |
| TCP | `tcp://host:port` |

## Health Details

```go
details := dh.HealthDetails()
for key, ep := range details {
    fmt.Printf("%s: healthy=%v status=%s latency=%v\n",
        key, *ep.Healthy, ep.Status, ep.Latency)
}
```

## Dynamic Endpoints

Add, remove, or update monitored endpoints at runtime on a running
`DepHealth` instance. Useful for applications that discover dependencies
dynamically (e.g., storage elements registered via REST API).

```go
import "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/httpcheck"

// Add a new endpoint after Start()
err := dh.AddEndpoint("api-backend", dephealth.TypeHTTP, true,
    dephealth.Endpoint{Host: "backend-2.svc", Port: "8080"},
    httpcheck.New(),
)

// Remove an endpoint (cancels its goroutine, deletes metrics)
err = dh.RemoveEndpoint("api-backend", "backend-2.svc", "8080")

// Atomically replace an endpoint with a new one
err = dh.UpdateEndpoint("api-backend", "backend-1.svc", "8080",
    dephealth.Endpoint{Host: "backend-3.svc", Port: "8080"},
    httpcheck.New(),
)
```

All three methods are thread-safe and idempotent (`AddEndpoint` ignores
duplicates, `RemoveEndpoint` ignores missing endpoints).

## Connection Pool Integration

```go
import "github.com/BigKAA/topologymetrics/sdk-go/dephealth/contrib/sqldb"

db, _ := sql.Open("pgx", connString)

dh, _ := dephealth.New("my-service", "my-team",
    sqldb.FromDB("postgres", db,
        dephealth.FromURL("postgresql://localhost:5432/mydb"),
        dephealth.Critical(true),
    ),
)
```

See [Connection Pools](docs/connection-pools.md) for Redis and MySQL pool
integration.

## Selective Imports

By default, importing the `checks` package registers all checker factories:

```go
import _ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks" // all checkers
```

To reduce binary size, import only the checkers you need:

```go
import (
    _ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/httpcheck"
    _ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/pgcheck"
)
```

Available sub-packages: `tcpcheck`, `httpcheck`, `grpccheck`, `pgcheck`,
`mysqlcheck`, `redischeck`, `amqpcheck`, `kafkacheck`.

## Authentication

HTTP and gRPC checkers support Bearer token, Basic Auth, and custom headers/metadata:

```go
dephealth.HTTP("secure-api",
    dephealth.FromURL("http://api.svc:8080"),
    dephealth.Critical(true),
    dephealth.WithHTTPBearerToken("eyJhbG..."),
)

dephealth.GRPC("grpc-backend",
    dephealth.FromParams("backend.svc", "9090"),
    dephealth.Critical(true),
    dephealth.WithGRPCBearerToken("eyJhbG..."),
)
```

See [Authentication](docs/authentication.md) for all options.

## Documentation

| Guide | Description |
| --- | --- |
| [Getting Started](docs/getting-started.md) | Installation, basic setup, first health check |
| [Checkers](docs/checkers.md) | Detailed guide for all 8 built-in checkers |
| [Configuration](docs/configuration.md) | All options, defaults, environment variables |
| [Connection Pools](docs/connection-pools.md) | Integration with existing connection pools |
| [Custom Checkers](docs/custom-checkers.md) | Implementing your own health checker |
| [Authentication](docs/authentication.md) | Auth for HTTP, gRPC, and database checkers |
| [Metrics](docs/metrics.md) | Prometheus metrics reference and PromQL examples |
| [Selective Imports](docs/selective-imports.md) | Binary size optimization with split packages |
| [API Reference](docs/api-reference.md) | Complete reference of all public symbols |
| [Troubleshooting](docs/troubleshooting.md) | Common issues and solutions |
| [Quick Start Guide](../docs/quickstart/go.md) | Extended examples with environment variables |
| [Migration v0.5 to v0.6](../docs/migration/v050-to-v060.md) | Migration guide for v0.6.0 split checkers |
| [Migration v0.6 to v0.7](../docs/migration/v060-to-v070.md) | Migration guide for v0.7.0 dynamic endpoints |

## License

Apache License 2.0 — see [LICENSE](../LICENSE).
