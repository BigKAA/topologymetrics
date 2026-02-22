*[Русская версия](connection-pools.ru.md)*

# Connection Pool Integration

dephealth supports two modes for checking dependencies:

- **Standalone mode** — SDK creates a new connection for each health check
- **Pool mode** — SDK uses the existing connection pool of your service

Pool mode is preferred because it reflects the actual ability of the service
to work with the dependency. If the connection pool is exhausted, the health
check will detect it.

## Standalone vs Pool Mode

| Aspect | Standalone | Pool |
| --- | --- | --- |
| Connection | New per check | Reuses existing pool |
| Reflects real health | Partially | Yes |
| Setup | Simple — just URL | Requires passing pool/client object |
| External dependencies | None (uses checker's driver) | Your application's driver |
| Detects pool exhaustion | No | Yes |

## contrib Packages

The `contrib/` directory provides helper functions for common drivers.
These functions wrap the pool/client object and return a `dephealth.Option`
that can be passed directly to `dephealth.New()`.

### PostgreSQL via `*sql.DB`

Package: `contrib/sqldb`

```go
import (
    "database/sql"
    "os"

    "github.com/BigKAA/topologymetrics/sdk-go/dephealth"
    "github.com/BigKAA/topologymetrics/sdk-go/dephealth/contrib/sqldb"
    _ "github.com/jackc/pgx/v5/stdlib"
)

// Create connection pool as usual
db, err := sql.Open("pgx", os.Getenv("DATABASE_URL"))
if err != nil {
    log.Fatal(err)
}

dh, err := dephealth.New("my-service", "my-team",
    sqldb.FromDB("postgres-main", db,
        dephealth.FromURL(os.Getenv("DATABASE_URL")),
        dephealth.Critical(true),
    ),
)
```

`sqldb.FromDB()` creates a PostgreSQL checker that uses the provided
`*sql.DB` for health checks. You must provide `FromURL()` or
`FromParams()` so that the SDK knows the host and port for metric labels.

### MySQL via `*sql.DB`

Package: `contrib/sqldb`

```go
import (
    "database/sql"

    "github.com/BigKAA/topologymetrics/sdk-go/dephealth"
    "github.com/BigKAA/topologymetrics/sdk-go/dephealth/contrib/sqldb"
    _ "github.com/go-sql-driver/mysql"
)

db, err := sql.Open("mysql", "user:pass@tcp(mysql.svc:3306)/mydb")
if err != nil {
    log.Fatal(err)
}

dh, err := dephealth.New("my-service", "my-team",
    sqldb.FromMySQLDB("mysql-main", db,
        dephealth.FromParams("mysql.svc", "3306"),
        dephealth.Critical(true),
    ),
)
```

### Redis via `*redis.Client`

Package: `contrib/redispool`

```go
import (
    "github.com/BigKAA/topologymetrics/sdk-go/dephealth"
    "github.com/BigKAA/topologymetrics/sdk-go/dephealth/contrib/redispool"
    "github.com/redis/go-redis/v9"
)

client := redis.NewClient(&redis.Options{
    Addr:     "redis.svc:6379",
    Password: "secret",
    DB:       0,
})

dh, err := dephealth.New("my-service", "my-team",
    redispool.FromClient("redis-cache", client,
        dephealth.Critical(false),
    ),
)
```

`redispool.FromClient()` automatically extracts host and port from
`client.Options().Addr`, so `FromURL()` or `FromParams()` is not
required (but can be provided to override).

## Direct Checker Pool Mode

If you need more control, you can create checkers directly with pool
options and register them via `AddDependency()`:

### PostgreSQL

```go
import (
    "github.com/BigKAA/topologymetrics/sdk-go/dephealth"
    "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/pgcheck"
)

db, _ := sql.Open("pgx", os.Getenv("DATABASE_URL"))

checker := pgcheck.New(
    pgcheck.WithDB(db),
    pgcheck.WithQuery("SELECT 1"),
)

dh, err := dephealth.New("my-service", "my-team",
    dephealth.AddDependency("postgres-main", dephealth.TypePostgres, checker,
        dephealth.FromURL(os.Getenv("DATABASE_URL")),
        dephealth.Critical(true),
    ),
)
```

### MySQL

```go
import (
    "github.com/BigKAA/topologymetrics/sdk-go/dephealth"
    "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/mysqlcheck"
)

db, _ := sql.Open("mysql", "user:pass@tcp(mysql.svc:3306)/mydb")

checker := mysqlcheck.New(
    mysqlcheck.WithDB(db),
    mysqlcheck.WithQuery("SELECT 1"),
)

dh, err := dephealth.New("my-service", "my-team",
    dephealth.AddDependency("mysql-main", dephealth.TypeMySQL, checker,
        dephealth.FromParams("mysql.svc", "3306"),
        dephealth.Critical(true),
    ),
)
```

### Redis

```go
import (
    "github.com/BigKAA/topologymetrics/sdk-go/dephealth"
    "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/redischeck"
    "github.com/redis/go-redis/v9"
)

client := redis.NewClient(&redis.Options{
    Addr: "redis.svc:6379",
})

checker := redischeck.New(redischeck.WithClient(client))

dh, err := dephealth.New("my-service", "my-team",
    dephealth.AddDependency("redis-cache", dephealth.TypeRedis, checker,
        dephealth.FromParams("redis.svc", "6379"),
        dephealth.Critical(false),
    ),
)
```

## contrib vs Direct: When to Use Which

| Use case | Recommendation |
| --- | --- |
| Standard setup, one pool per dependency | `contrib/sqldb` or `contrib/redispool` |
| Custom health check query | Direct checker with `pgcheck.WithQuery()` |
| Multiple databases on same `*sql.DB` pool | Direct checker (specify query for each) |
| Non-standard driver or wrapper | Direct checker with pool object |

## Full Example: Mixed Modes

```go
package main

import (
    "context"
    "database/sql"
    "log"
    "net/http"
    "os"

    "github.com/BigKAA/topologymetrics/sdk-go/dephealth"
    "github.com/BigKAA/topologymetrics/sdk-go/dephealth/contrib/redispool"
    "github.com/BigKAA/topologymetrics/sdk-go/dephealth/contrib/sqldb"
    "github.com/prometheus/client_golang/prometheus/promhttp"
    "github.com/redis/go-redis/v9"
    _ "github.com/jackc/pgx/v5/stdlib"

    _ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/httpcheck"
)

func main() {
    // Existing connection pools
    db, _ := sql.Open("pgx", os.Getenv("DATABASE_URL"))
    rdb := redis.NewClient(&redis.Options{Addr: "redis.svc:6379"})

    dh, err := dephealth.New("my-service", "my-team",
        // Pool mode — PostgreSQL
        sqldb.FromDB("postgres-main", db,
            dephealth.FromURL(os.Getenv("DATABASE_URL")),
            dephealth.Critical(true),
        ),

        // Pool mode — Redis
        redispool.FromClient("redis-cache", rdb,
            dephealth.Critical(false),
        ),

        // Standalone mode — HTTP (no pool needed)
        dephealth.HTTP("payment-api",
            dephealth.FromURL("http://payment.svc:8080"),
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

## See Also

- [Checkers](checkers.md) — all checker details including pool mode sections
- [Custom Checkers](custom-checkers.md) — creating your own HealthChecker
- [API Reference](api-reference.md) — `AddDependency`, contrib functions
- [Troubleshooting](troubleshooting.md) — common issues and solutions
