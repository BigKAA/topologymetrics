*[Русская версия](checkers.ru.md)*

# Health Checkers

The Go SDK includes 9 built-in health checkers for common dependency types.
Each checker implements the `HealthChecker` interface and can be used via
the high-level API (`dephealth.HTTP()`, etc.) or directly via its sub-package.

## HTTP

Checks HTTP endpoints by sending a GET request and expecting a 2xx response.

### Registration

```go
dephealth.HTTP("payment-api",
    dephealth.FromURL("http://payment.svc:8080"),
    dephealth.Critical(true),
)
```

### Options

| Option | Default | Description |
| --- | --- | --- |
| `WithHTTPHealthPath(path)` | `/health` | Path for the health check endpoint |
| `WithHTTPTLS(enabled)` | `false` | Use HTTPS instead of HTTP |
| `WithHTTPTLSSkipVerify(skip)` | `false` | Skip TLS certificate verification |
| `WithHTTPHeaders(headers)` | — | Custom HTTP headers (map[string]string) |
| `WithHTTPBearerToken(token)` | — | Set `Authorization: Bearer <token>` header |
| `WithHTTPBasicAuth(user, pass)` | — | Set `Authorization: Basic <base64>` header |

### Full Example

```go
package main

import (
    "context"
    "log"
    "net/http"

    "github.com/BigKAA/topologymetrics/sdk-go/dephealth"
    "github.com/prometheus/client_golang/prometheus/promhttp"

    _ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/httpcheck"
)

func main() {
    dh, err := dephealth.New("my-service", "my-team",
        dephealth.HTTP("payment-api",
            dephealth.FromURL("https://payment.svc:443"),
            dephealth.Critical(true),
            dephealth.WithHTTPHealthPath("/healthz"),
            dephealth.WithHTTPTLS(true),
            dephealth.WithHTTPTLSSkipVerify(true),
            dephealth.WithHTTPHeaders(map[string]string{
                "X-Request-Source": "dephealth",
            }),
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

### Error Classification

| Condition | Status | Detail |
| --- | --- | --- |
| Response 2xx | `ok` | `ok` |
| Response 401 or 403 | `auth_error` | `auth_error` |
| Response other non-2xx | `unhealthy` | `http_<code>` (e.g., `http_500`) |
| Network error | classified by core | depends on error type |

### Direct Checker Usage

```go
import "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/httpcheck"

checker := httpcheck.New(
    httpcheck.WithHealthPath("/ready"),
    httpcheck.WithTLSEnabled(true),
)

err := checker.Check(ctx, dephealth.Endpoint{Host: "api.svc", Port: "8080"})
```

### Behavior Notes

- Follows HTTP redirects automatically (3xx)
- Creates a new HTTP client for each check
- Sends `User-Agent: dephealth/0.6.0` header
- Custom headers are applied after User-Agent and can override it

---

## gRPC

Checks gRPC services using the
[gRPC Health Checking Protocol](https://github.com/grpc/grpc/blob/master/doc/health-checking.md).

### Registration

```go
dephealth.GRPC("user-service",
    dephealth.FromParams("user.svc", "9090"),
    dephealth.Critical(true),
)
```

### Options

| Option | Default | Description |
| --- | --- | --- |
| `WithGRPCServiceName(name)` | `""` (empty) | Service name to check; empty checks overall server health |
| `WithGRPCTLS(enabled)` | `false` | Enable TLS |
| `WithGRPCTLSSkipVerify(skip)` | `false` | Skip TLS certificate verification |
| `WithGRPCMetadata(md)` | — | Custom gRPC metadata (map[string]string) |
| `WithGRPCBearerToken(token)` | — | Set `authorization: Bearer <token>` metadata |
| `WithGRPCBasicAuth(user, pass)` | — | Set `authorization: Basic <base64>` metadata |

### Full Example

```go
dh, err := dephealth.New("my-service", "my-team",
    // Check a specific gRPC service
    dephealth.GRPC("user-service",
        dephealth.FromParams("user.svc", "9090"),
        dephealth.Critical(true),
        dephealth.WithGRPCServiceName("user.v1.UserService"),
        dephealth.WithGRPCTLS(true),
        dephealth.WithGRPCMetadata(map[string]string{
            "x-request-id": "dephealth",
        }),
    ),

    // Check overall server health (empty service name)
    dephealth.GRPC("grpc-gateway",
        dephealth.FromParams("gateway.svc", "9090"),
        dephealth.Critical(false),
    ),
)
```

### Error Classification

| Condition | Status | Detail |
| --- | --- | --- |
| Response SERVING | `ok` | `ok` |
| gRPC UNAUTHENTICATED | `auth_error` | `auth_error` |
| gRPC PERMISSION_DENIED | `auth_error` | `auth_error` |
| Response NOT_SERVING | `unhealthy` | `grpc_not_serving` |
| Other gRPC status | `unhealthy` | `grpc_unknown` |
| Dial/RPC error | classified by core | depends on error type |

### Direct Checker Usage

```go
import "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/grpccheck"

checker := grpccheck.New(
    grpccheck.WithServiceName("user.v1.UserService"),
    grpccheck.WithTLS(true),
)

err := checker.Check(ctx, dephealth.Endpoint{Host: "user.svc", Port: "9090"})
```

### Behavior Notes

- Uses `passthrough:///` resolver to avoid DNS SRV lookups (important in
  Kubernetes where `ndots:5` causes high latency with `dns:///` resolver)
- Creates a new gRPC connection for each check
- Empty service name checks overall server health

---

## TCP

Checks TCP connectivity by establishing a connection and closing it
immediately. The simplest checker — no application-level protocol involved.

### Registration

```go
dephealth.TCP("memcached",
    dephealth.FromParams("memcached.svc", "11211"),
    dephealth.Critical(false),
)
```

### Options

No checker-specific options. TCP checker is stateless.

### Full Example

```go
dh, err := dephealth.New("my-service", "my-team",
    dephealth.TCP("memcached",
        dephealth.FromParams("memcached.svc", "11211"),
        dephealth.Critical(false),
        dephealth.CheckInterval(10 * time.Second),
    ),
    dephealth.TCP("custom-service",
        dephealth.FromParams("custom.svc", "5555"),
        dephealth.Critical(true),
    ),
)
```

### Error Classification

TCP checker does not produce classified errors. All failures (connection
refused, DNS errors, timeouts) are classified by the core error classifier.

### Direct Checker Usage

```go
import "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/tcpcheck"

checker := tcpcheck.New()
err := checker.Check(ctx, dephealth.Endpoint{Host: "memcached.svc", Port: "11211"})
```

### Behavior Notes

- Only performs TCP handshake (SYN/ACK) — no data is sent or received
- Connection is closed immediately after establishment
- Useful for services that don't have a health check protocol

---

## PostgreSQL

Checks PostgreSQL by executing a query (default: `SELECT 1`). Supports
both standalone mode (new connection) and pool mode (existing `*sql.DB`).

### Registration

```go
dephealth.Postgres("postgres-main",
    dephealth.FromURL("postgresql://user:pass@pg.svc:5432/mydb"),
    dephealth.Critical(true),
)
```

### Options

| Option | Default | Description |
| --- | --- | --- |
| `WithPostgresQuery(query)` | `SELECT 1` | Custom SQL query for health check |

For pool mode, use the `contrib/sqldb` package or create a checker
directly with `pgcheck.WithDB(db)`.

### Full Example

```go
import (
    _ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/pgcheck"
)

dh, err := dephealth.New("my-service", "my-team",
    // Standalone mode — creates new connection for each check
    dephealth.Postgres("postgres-main",
        dephealth.FromURL(os.Getenv("DATABASE_URL")),
        dephealth.Critical(true),
        dephealth.WithPostgresQuery("SELECT 1"),
    ),
)
```

### Pool Mode

Use the existing connection pool for health checks:

```go
import (
    "database/sql"

    "github.com/BigKAA/topologymetrics/sdk-go/dephealth"
    "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/pgcheck"
    _ "github.com/jackc/pgx/v5/stdlib"
)

db, _ := sql.Open("pgx", os.Getenv("DATABASE_URL"))

checker := pgcheck.New(pgcheck.WithDB(db))

dh, err := dephealth.New("my-service", "my-team",
    dephealth.AddDependency("postgres-main", dephealth.TypePostgres, checker,
        dephealth.FromURL(os.Getenv("DATABASE_URL")),
        dephealth.Critical(true),
    ),
)
```

Or use the `contrib/sqldb` helper — see [Connection Pools](connection-pools.md).

### Error Classification

| Condition | Status | Detail |
| --- | --- | --- |
| Query succeeds | `ok` | `ok` |
| SQLSTATE 28000 (Invalid Authorization) | `auth_error` | `auth_error` |
| SQLSTATE 28P01 (Authentication Failed) | `auth_error` | `auth_error` |
| "password authentication failed" in error | `auth_error` | `auth_error` |
| Other errors | classified by core | depends on error type |

### Behavior Notes

- Standalone mode builds DSN `postgres://host:port/postgres` when no URL is
  provided (connects to the default `postgres` database)
- Pool mode reuses the existing connection pool — reflects the actual ability
  of the service to work with the dependency
- Uses `pgx` driver (`github.com/jackc/pgx/v5/stdlib`)

---

## MySQL

Checks MySQL by executing a query (default: `SELECT 1`). Supports both
standalone mode and pool mode.

### Registration

```go
dephealth.MySQL("mysql-main",
    dephealth.FromURL("mysql://user:pass@mysql.svc:3306/mydb"),
    dephealth.Critical(true),
)
```

### Options

| Option | Default | Description |
| --- | --- | --- |
| `WithMySQLQuery(query)` | `SELECT 1` | Custom SQL query for health check |

For pool mode, use the `contrib/sqldb` package or create a checker
directly with `mysqlcheck.WithDB(db)`.

### Full Example

```go
import (
    _ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/mysqlcheck"
)

dh, err := dephealth.New("my-service", "my-team",
    dephealth.MySQL("mysql-main",
        dephealth.FromURL("mysql://user:pass@mysql.svc:3306/mydb"),
        dephealth.Critical(true),
    ),
)
```

### Pool Mode

```go
import (
    "database/sql"

    "github.com/BigKAA/topologymetrics/sdk-go/dephealth"
    "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/mysqlcheck"
    _ "github.com/go-sql-driver/mysql"
)

db, _ := sql.Open("mysql", "user:pass@tcp(mysql.svc:3306)/mydb")

checker := mysqlcheck.New(mysqlcheck.WithDB(db))

dh, err := dephealth.New("my-service", "my-team",
    dephealth.AddDependency("mysql-main", dephealth.TypeMySQL, checker,
        dephealth.FromParams("mysql.svc", "3306"),
        dephealth.Critical(true),
    ),
)
```

### URL to DSN Conversion

The `mysqlcheck` package provides `URLToDSN()` to convert a URL to the
go-sql-driver DSN format:

```go
dsn := mysqlcheck.URLToDSN("mysql://user:pass@host:3306/db?charset=utf8")
// Result: "user:pass@tcp(host:3306)/db?charset=utf8"
```

### Error Classification

| Condition | Status | Detail |
| --- | --- | --- |
| Query succeeds | `ok` | `ok` |
| MySQL error 1045 (Access Denied) | `auth_error` | `auth_error` |
| "Access denied" in error message | `auth_error` | `auth_error` |
| Other errors | classified by core | depends on error type |

### Behavior Notes

- Standalone mode builds DSN `tcp(host:port)/` when no URL is provided
- Uses `go-sql-driver/mysql` driver
- URL parsing preserves query parameters

---

## Redis

Checks Redis by executing the `PING` command. Supports both standalone
mode and pool mode.

### Registration

```go
dephealth.Redis("redis-cache",
    dephealth.FromURL("redis://:password@redis.svc:6379/0"),
    dephealth.Critical(false),
)
```

### Options

| Option | Default | Description |
| --- | --- | --- |
| `WithRedisPassword(password)` | `""` | Password for standalone mode |
| `WithRedisDB(db)` | `0` | Database number for standalone mode |

For pool mode, use the `contrib/redispool` package or create a checker
directly with `redischeck.WithClient(client)`.

### Full Example

```go
import (
    _ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/redischeck"
)

dh, err := dephealth.New("my-service", "my-team",
    // Password from URL
    dephealth.Redis("redis-cache",
        dephealth.FromURL("redis://:mypassword@redis.svc:6379/0"),
        dephealth.Critical(false),
    ),

    // Password via option
    dephealth.Redis("redis-sessions",
        dephealth.FromParams("redis-sessions.svc", "6379"),
        dephealth.WithRedisPassword("secret"),
        dephealth.WithRedisDB(1),
        dephealth.Critical(true),
    ),
)
```

### Pool Mode

```go
import (
    "github.com/BigKAA/topologymetrics/sdk-go/dephealth"
    "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/redischeck"
    "github.com/redis/go-redis/v9"
)

client := redis.NewClient(&redis.Options{
    Addr:     "redis.svc:6379",
    Password: "secret",
    DB:       0,
})

checker := redischeck.New(redischeck.WithClient(client))

dh, err := dephealth.New("my-service", "my-team",
    dephealth.AddDependency("redis-cache", dephealth.TypeRedis, checker,
        dephealth.FromParams("redis.svc", "6379"),
        dephealth.Critical(false),
    ),
)
```

### Error Classification

| Condition | Status | Detail |
| --- | --- | --- |
| PING succeeds | `ok` | `ok` |
| "NOAUTH" in error | `auth_error` | `auth_error` |
| "WRONGPASS" in error | `auth_error` | `auth_error` |
| Connection refused | `connection_error` | `connection_refused` |
| Connection timeout | `connection_error` | `connection_refused` |
| Context deadline exceeded | `connection_error` | `connection_refused` |
| Other errors | classified by core | depends on error type |

### Behavior Notes

- Standalone mode sets fixed internal timeouts: `DialTimeout=3s`,
  `ReadTimeout=3s`, `WriteTimeout=3s`
- `MaxRetries=0` — no automatic retries (scheduler handles retry)
- Password from options takes precedence over password from URL
- DB number from options takes precedence over DB number from URL

---

## AMQP (RabbitMQ)

Checks AMQP brokers by establishing a connection and closing it
immediately. Only standalone mode is supported.

### Registration

```go
dephealth.AMQP("rabbitmq",
    dephealth.FromParams("rabbitmq.svc", "5672"),
    dephealth.Critical(false),
)
```

### Options

| Option | Default | Description |
| --- | --- | --- |
| `WithAMQPURL(url)` | — | Custom AMQP URL (overrides default `amqp://guest:guest@host:port/`) |

### Full Example

```go
import (
    _ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/amqpcheck"
)

dh, err := dephealth.New("my-service", "my-team",
    // Default credentials (guest:guest)
    dephealth.AMQP("rabbitmq",
        dephealth.FromParams("rabbitmq.svc", "5672"),
        dephealth.Critical(false),
    ),

    // Custom credentials via URL
    dephealth.AMQP("rabbitmq-prod",
        dephealth.FromParams("rmq-prod.svc", "5672"),
        dephealth.WithAMQPURL("amqp://myuser:mypass@rmq-prod.svc:5672/myvhost"),
        dephealth.Critical(true),
    ),
)
```

### Error Classification

| Condition | Status | Detail |
| --- | --- | --- |
| Connection established | `ok` | `ok` |
| "403" in error | `auth_error` | `auth_error` |
| "ACCESS_REFUSED" in error | `auth_error` | `auth_error` |
| Other errors | classified by core | depends on error type |

### Behavior Notes

- Default URL: `amqp://guest:guest@host:port/` (RabbitMQ default credentials)
- No pool mode — always creates a new connection
- Uses goroutine wrapper for context cancellation support (amqp091-go
  library does not natively support context)
- Connection is closed immediately after successful establishment

---

## Kafka

Checks Kafka brokers by connecting and requesting cluster metadata.
Stateless checker with no configuration options.

### Registration

```go
dephealth.Kafka("kafka",
    dephealth.FromParams("kafka.svc", "9092"),
    dephealth.Critical(true),
)
```

Multi-host (multiple brokers):

```go
dephealth.Kafka("kafka-cluster",
    dephealth.FromURL("kafka://broker1:9092,broker2:9092,broker3:9092"),
    dephealth.Critical(true),
)
```

> Note: with `FromURL`, each broker creates a separate endpoint — each is
> checked independently and appears as a separate line in metrics.

### Options

No checker-specific options. Kafka checker is stateless.

### Full Example

```go
import (
    _ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/kafkacheck"
)

dh, err := dephealth.New("my-service", "my-team",
    dephealth.Kafka("kafka",
        dephealth.FromParams("kafka.svc", "9092"),
        dephealth.Critical(true),
    ),
)
```

### Error Classification

| Condition | Status | Detail |
| --- | --- | --- |
| Metadata returns brokers | `ok` | `ok` |
| No brokers in metadata | `unhealthy` | `no_brokers` |
| Dial/metadata error | classified by core | depends on error type |

### Behavior Notes

- Connects to broker, requests metadata, closes connection
- Verifies that at least one broker is present in the metadata response
- Uses `kafka-go` library (`github.com/segmentio/kafka-go`)
- No authentication support (plain TCP only)

---

## LDAP

Checks LDAP servers using configurable check methods: anonymous bind, simple
bind, Root DSE search, or custom search. Supports LDAP, LDAPS (TLS), and
StartTLS connections.

### Registration

```go
dephealth.LDAP("ldap-server",
    dephealth.FromParams("ldap.svc", "389"),
    dephealth.Critical(false),
)
```

### Options

| Option | Default | Description |
| --- | --- | --- |
| `WithLDAPCheckMethod(method)` | `root_dse` | Check method: `anonymous_bind`, `simple_bind`, `root_dse`, `search` |
| `WithLDAPBindDN(dn)` | `""` | DN for Simple Bind |
| `WithLDAPBindPassword(password)` | `""` | Password for Simple Bind |
| `WithLDAPBaseDN(baseDN)` | `""` | Base DN for search method |
| `WithLDAPSearchFilter(filter)` | `(objectClass=*)` | LDAP search filter |
| `WithLDAPSearchScope(scope)` | `base` | Search scope: `base`, `one`, `sub` |
| `WithLDAPStartTLS(enabled)` | `false` | Enable StartTLS (only with `ldap://`) |
| `WithLDAPTLSSkipVerify(skip)` | `false` | Skip TLS certificate verification |

### Full Example

```go
import (
    _ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/ldapcheck"
)

dh, err := dephealth.New("my-service", "my-team",
    // Root DSE check (default method)
    dephealth.LDAP("ldap-server",
        dephealth.FromParams("ldap.svc", "389"),
        dephealth.Critical(false),
    ),

    // Simple Bind with credentials
    dephealth.LDAP("active-directory",
        dephealth.FromURL("ldaps://ad.corp.local:636"),
        dephealth.WithLDAPCheckMethod("simple_bind"),
        dephealth.WithLDAPBindDN("cn=healthcheck,ou=service-accounts,dc=corp,dc=local"),
        dephealth.WithLDAPBindPassword("secret"),
        dephealth.Critical(true),
    ),

    // Custom search
    dephealth.LDAP("openldap",
        dephealth.FromParams("openldap.svc", "389"),
        dephealth.WithLDAPCheckMethod("search"),
        dephealth.WithLDAPBaseDN("dc=example,dc=org"),
        dephealth.WithLDAPSearchFilter("(objectClass=organization)"),
        dephealth.WithLDAPSearchScope("base"),
        dephealth.WithLDAPStartTLS(true),
        dephealth.Critical(false),
    ),
)
```

### Error Classification

| Condition | Status | Detail |
| --- | --- | --- |
| Check succeeds | `ok` | `ok` |
| LDAP result 49 (Invalid Credentials) | `auth_error` | `auth_error` |
| LDAP result 50 (Insufficient Access Rights) | `auth_error` | `auth_error` |
| LDAP result Busy/Unavailable/Unwilling | `unhealthy` | `unhealthy` |
| Connection refused | `connection_error` | `connection_refused` |
| DNS error | `dns_error` | `dns_error` |
| TLS/x509 error | `tls_error` | `tls_error` |
| Other errors | classified by core | depends on error type |

### Direct Checker Usage

```go
import "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/ldapcheck"

checker := ldapcheck.New(
    ldapcheck.WithCheckMethod(ldapcheck.MethodSimpleBind),
    ldapcheck.WithBindDN("cn=admin,dc=example,dc=org"),
    ldapcheck.WithBindPassword("secret"),
)

err := checker.Check(ctx, dephealth.Endpoint{Host: "ldap.svc", Port: "389"})
```

### Behavior Notes

- Default check method is `root_dse` — searches for Root DSE attributes
  (works without authentication on most LDAP servers)
- Supports pool mode via `ldapcheck.WithConn(conn)` with existing `*ldap.Conn`
- Dial timeout is 3s or remaining context timeout (whichever is shorter)
- `ldaps://` scheme automatically enables TLS
- StartTLS is only valid with `ldap://` scheme
- Uses `go-ldap/ldap/v3` library

---

## Error Classification Summary

All checkers classify errors into status categories. The core error
classifier handles common error types (timeouts, DNS errors, TLS errors,
connection refused). Checker-specific classification adds protocol-level
detail:

| Status Category | Meaning | Common Causes |
| --- | --- | --- |
| `ok` | Dependency is healthy | Check succeeded |
| `timeout` | Check timed out | Slow network, overloaded service |
| `connection_error` | Cannot connect | Service down, wrong host/port, firewall |
| `dns_error` | DNS resolution failed | Wrong hostname, DNS outage |
| `auth_error` | Authentication failed | Wrong credentials, expired token |
| `tls_error` | TLS handshake failed | Invalid certificate, TLS misconfiguration |
| `unhealthy` | Connected but not healthy | Service reports unhealthy, returns error code |
| `error` | Unexpected error | Unclassified failures |

## See Also

- [Getting Started](getting-started.md) — basic setup and first example
- [Authentication](authentication.md) — detailed auth guide for HTTP and gRPC
- [Connection Pools](connection-pools.md) — pool mode via contrib packages
- [Custom Checkers](custom-checkers.md) — creating your own `HealthChecker`
- [Selective Imports](selective-imports.md) — importing only needed checkers
- [API Reference](api-reference.md) — complete reference of all public symbols
- [Troubleshooting](troubleshooting.md) — common issues and solutions
