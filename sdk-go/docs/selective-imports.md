*[Русская версия](selective-imports.ru.md)*

# Selective Imports

Starting with v0.6.0, the Go SDK supports selective imports — you can
import only the checkers your service actually uses, reducing binary size
and the number of transitive dependencies.

## The Problem

In v0.5.0 and earlier, the `checks` package was monolithic. Importing it
pulled all 8 checker implementations and their dependencies:

```go
import _ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks"
```

This meant that even if your service only needed HTTP and PostgreSQL checks,
the compiled binary included gRPC, Kafka, AMQP, and other libraries.

## The Solution

In v0.6.0, each checker lives in its own sub-package under `checks/`.
Import only what you need:

```go
import (
    _ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/httpcheck"
    _ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/pgcheck"
)
```

The rest of the API stays the same — you still call `dephealth.HTTP()`,
`dephealth.Postgres()`, etc.

## Available Sub-Packages

| Sub-package | Import path | External dependencies |
| --- | --- | --- |
| `httpcheck` | `.../checks/httpcheck` | stdlib only |
| `tcpcheck` | `.../checks/tcpcheck` | stdlib only |
| `grpccheck` | `.../checks/grpccheck` | `google.golang.org/grpc` |
| `pgcheck` | `.../checks/pgcheck` | `github.com/jackc/pgx/v5` |
| `mysqlcheck` | `.../checks/mysqlcheck` | `github.com/go-sql-driver/mysql` |
| `redischeck` | `.../checks/redischeck` | `github.com/redis/go-redis/v9` |
| `amqpcheck` | `.../checks/amqpcheck` | `github.com/rabbitmq/amqp091-go` |
| `kafkacheck` | `.../checks/kafkacheck` | `github.com/segmentio/kafka-go` |

Full import paths use the module prefix
`github.com/BigKAA/topologymetrics/sdk-go/dephealth/`.

## How It Works

Each sub-package registers its checker factory via `init()`:

```go
// Inside checks/httpcheck/httpcheck.go
func init() {
    dephealth.RegisterCheckerFactory(dephealth.TypeHTTP, NewFromConfig)
}
```

When you blank-import a sub-package (`import _ ".../checks/httpcheck"`),
its `init()` runs and registers the factory. After that, `dephealth.HTTP()`
can create instances of the HTTP checker.

If you call `dephealth.HTTP()` without importing `httpcheck` (or `checks`),
`New()` returns an error: `no checker factory registered for type "http"`.

## Full Import (All Checkers)

The `checks` package still works as before. It blank-imports all 8
sub-packages:

```go
import _ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks"
```

This is equivalent to importing all 8 sub-packages individually. Use this
when binary size is not a concern or when you need all checker types.

## Example: Selective Import

A service that only monitors PostgreSQL and Redis:

```go
package main

import (
    "context"
    "log"
    "net/http"

    "github.com/BigKAA/topologymetrics/sdk-go/dephealth"
    "github.com/prometheus/client_golang/prometheus/promhttp"

    // Only register the checkers we need
    _ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/pgcheck"
    _ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/redischeck"
)

func main() {
    dh, err := dephealth.New("my-service", "my-team",
        dephealth.Postgres("postgres-main",
            dephealth.FromURL("postgresql://localhost:5432/mydb"),
            dephealth.Critical(true),
        ),
        dephealth.Redis("redis-cache",
            dephealth.FromParams("localhost", "6379"),
            dephealth.Critical(false),
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

This binary will **not** include gRPC, Kafka, AMQP, HTTP checker, MySQL,
or TCP checker libraries.

## Backward Compatibility

The `checks` package provides deprecated type aliases and constructor
wrappers for all checkers. Existing code using `checks.HTTPChecker`,
`checks.NewHTTPChecker()`, etc. continues to compile without changes:

```go
// Old style (still works, but deprecated)
import "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks"

checker := checks.NewHTTPChecker(checks.WithHealthPath("/ready"))
```

```go
// New style (recommended)
import "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/httpcheck"

checker := httpcheck.New(httpcheck.WithHealthPath("/ready"))
```

All deprecated aliases are listed in `checks/compat.go` with godoc
references to the new packages.

## Contrib Packages

The `contrib/` packages (`sqldb`, `redispool`) work with both import
styles. They import their respective checker sub-packages internally,
so you don't need an additional blank import when using contrib:

```go
import (
    "github.com/BigKAA/topologymetrics/sdk-go/dephealth/contrib/sqldb"
    // No need for: _ ".../checks/pgcheck" — sqldb imports it
)
```

## Migration from v0.5.0

If you're upgrading from v0.5.0, see the
[Migration Guide](../../docs/migration/v050-to-v060.md) for step-by-step
instructions.

## See Also

- [Getting Started](getting-started.md) — basic setup and first example
- [Checkers](checkers.md) — detailed guide for all 8 checkers
- [Troubleshooting](troubleshooting.md) — common issues and solutions
- [Migration Guide v0.5.0 → v0.6.0](../../docs/migration/v050-to-v060.md)
