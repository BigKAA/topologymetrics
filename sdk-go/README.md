# dephealth

Go SDK for monitoring microservice dependencies via Prometheus metrics.

## Features

- Automatic health checking for dependencies (PostgreSQL, MySQL, Redis, RabbitMQ, Kafka, HTTP, gRPC, TCP)
- Prometheus metrics export: `app_dependency_health` (Gauge 0/1) and `app_dependency_latency_seconds` (Histogram)
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
    "log"
    "net/http"

    "github.com/BigKAA/topologymetrics/sdk-go/dephealth"
    "github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
    dh, err := dephealth.New(
        dephealth.WithDependency("postgres", "postgresql://user:pass@localhost:5432/mydb"),
        dephealth.WithDependency("redis", "redis://localhost:6379"),
    )
    if err != nil {
        log.Fatal(err)
    }

    dh.Start()
    defer dh.Stop()

    http.Handle("/metrics", promhttp.Handler())
    log.Fatal(http.ListenAndServe(":8080", nil))
}
```

## Configuration

| Option | Default | Description |
|--------|---------|-------------|
| `WithInterval` | `15s` | Check interval |
| `WithTimeout` | `5s` | Check timeout |
| `WithFailureThreshold` | `1` | Consecutive failures before unhealthy |
| `WithSuccessThreshold` | `1` | Consecutive successes before healthy |

## Supported Dependencies

| Type | URL Format |
|------|-----------|
| PostgreSQL | `postgresql://user:pass@host:5432/db` |
| MySQL | `mysql://user:pass@host:3306/db` |
| Redis | `redis://host:6379` |
| RabbitMQ | `amqp://user:pass@host:5672/vhost` |
| Kafka | `kafka://host1:9092,host2:9092` |
| HTTP | `http://host:8080/health` |
| gRPC | via `dephealth.FromParams(host, port)` |
| TCP | `tcp://host:port` |

## Connection Pool Integration

```go
import "github.com/BigKAA/topologymetrics/sdk-go/dephealth/contrib/pgxcheck"

pool, _ := pgxpool.New(ctx, connString)
checker := pgxcheck.New(pool)

dh, _ := dephealth.New(
    dephealth.WithChecker("postgres", checker,
        dephealth.FromURL("postgresql://localhost:5432/mydb")),
)
```

## License

MIT — see [LICENSE](./LICENSE).
