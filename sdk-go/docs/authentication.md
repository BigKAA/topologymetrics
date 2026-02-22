*[Русская версия](authentication.ru.md)*

# Authentication

This guide covers authentication options for all checkers that support
credentials — HTTP, gRPC, and database/cache checkers.

## Overview

| Checker | Auth methods | How credentials are passed |
| --- | --- | --- |
| HTTP | Bearer token, Basic auth, custom headers | `WithHTTPBearerToken`, `WithHTTPBasicAuth`, `WithHTTPHeaders` |
| gRPC | Bearer token, Basic auth, custom metadata | `WithGRPCBearerToken`, `WithGRPCBasicAuth`, `WithGRPCMetadata` |
| PostgreSQL | Username/password | Credentials in URL (`postgresql://user:pass@...`) |
| MySQL | Username/password | Credentials in URL (`mysql://user:pass@...`) |
| Redis | Password | `WithRedisPassword` or password in URL (`redis://:pass@...`) |
| AMQP | Username/password | Credentials in URL via `WithAMQPURL` (`amqp://user:pass@...`) |
| TCP | — | No authentication |
| Kafka | — | No authentication |

## HTTP Authentication

Only one authentication method per dependency is allowed. Specifying
both `WithHTTPBearerToken` and `WithHTTPBasicAuth` causes a validation
error.

### Bearer Token

```go
dephealth.HTTP("secure-api",
    dephealth.FromURL("http://api.svc:8080"),
    dephealth.Critical(true),
    dephealth.WithHTTPBearerToken(os.Getenv("API_TOKEN")),
)
```

Sends `Authorization: Bearer <token>` header with each health check request.

### Basic Auth

```go
dephealth.HTTP("basic-api",
    dephealth.FromURL("http://api.svc:8080"),
    dephealth.Critical(true),
    dephealth.WithHTTPBasicAuth("admin", os.Getenv("API_PASSWORD")),
)
```

Sends `Authorization: Basic <base64(user:pass)>` header.

### Custom Headers

For non-standard authentication schemes (API keys, custom tokens):

```go
dephealth.HTTP("custom-auth-api",
    dephealth.FromURL("http://api.svc:8080"),
    dephealth.Critical(true),
    dephealth.WithHTTPHeaders(map[string]string{
        "X-API-Key":    os.Getenv("API_KEY"),
        "X-API-Secret": os.Getenv("API_SECRET"),
    }),
)
```

Custom headers are applied after the `User-Agent` header and can
override it if needed.

## gRPC Authentication

Only one authentication method per dependency is allowed, same as HTTP.

### Bearer Token

```go
dephealth.GRPC("grpc-backend",
    dephealth.FromParams("backend.svc", "9090"),
    dephealth.Critical(true),
    dephealth.WithGRPCBearerToken(os.Getenv("GRPC_TOKEN")),
)
```

Sends `authorization: Bearer <token>` as gRPC metadata.

### Basic Auth

```go
dephealth.GRPC("grpc-backend",
    dephealth.FromParams("backend.svc", "9090"),
    dephealth.Critical(true),
    dephealth.WithGRPCBasicAuth("admin", os.Getenv("GRPC_PASSWORD")),
)
```

Sends `authorization: Basic <base64(user:pass)>` as gRPC metadata.

### Custom Metadata

```go
dephealth.GRPC("grpc-backend",
    dephealth.FromParams("backend.svc", "9090"),
    dephealth.Critical(true),
    dephealth.WithGRPCMetadata(map[string]string{
        "x-api-key": os.Getenv("GRPC_API_KEY"),
    }),
)
```

## Database Credentials

### PostgreSQL

Credentials are included in the connection URL:

```go
dephealth.Postgres("postgres-main",
    dephealth.FromURL("postgresql://myuser:mypass@pg.svc:5432/mydb"),
    dephealth.Critical(true),
)
```

In practice, use environment variables:

```go
dephealth.Postgres("postgres-main",
    dephealth.FromURL(os.Getenv("DATABASE_URL")),
    dephealth.Critical(true),
)
```

### MySQL

Same approach — credentials in URL:

```go
dephealth.MySQL("mysql-main",
    dephealth.FromURL("mysql://myuser:mypass@mysql.svc:3306/mydb"),
    dephealth.Critical(true),
)
```

### Redis

Password can be set via option or included in the URL:

```go
// Via option
dephealth.Redis("redis-cache",
    dephealth.FromParams("redis.svc", "6379"),
    dephealth.WithRedisPassword(os.Getenv("REDIS_PASSWORD")),
    dephealth.Critical(false),
)

// Via URL
dephealth.Redis("redis-cache",
    dephealth.FromURL("redis://:mypassword@redis.svc:6379/0"),
    dephealth.Critical(false),
)
```

When both are specified, the option value takes precedence over the URL.

### AMQP (RabbitMQ)

Credentials are part of the AMQP URL:

```go
dephealth.AMQP("rabbitmq",
    dephealth.FromParams("rabbitmq.svc", "5672"),
    dephealth.WithAMQPURL(os.Getenv("AMQP_URL")),
    dephealth.Critical(false),
)
```

Default URL (when no `WithAMQPURL` is set): `amqp://guest:guest@host:port/`

## Auth Error Classification

When a dependency rejects credentials, the checker classifies the error
as `auth_error`. This enables specific alerting on authentication failures
without mixing them with connectivity or health issues.

| Checker | Trigger | Status | Detail |
| --- | --- | --- | --- |
| HTTP | Response 401 (Unauthorized) | `auth_error` | `auth_error` |
| HTTP | Response 403 (Forbidden) | `auth_error` | `auth_error` |
| gRPC | Code UNAUTHENTICATED | `auth_error` | `auth_error` |
| gRPC | Code PERMISSION_DENIED | `auth_error` | `auth_error` |
| PostgreSQL | SQLSTATE 28000 | `auth_error` | `auth_error` |
| PostgreSQL | SQLSTATE 28P01 | `auth_error` | `auth_error` |
| MySQL | Error 1045 (Access Denied) | `auth_error` | `auth_error` |
| Redis | NOAUTH error | `auth_error` | `auth_error` |
| Redis | WRONGPASS error | `auth_error` | `auth_error` |
| AMQP | 403 ACCESS_REFUSED | `auth_error` | `auth_error` |

### PromQL Example: Auth Error Alerting

```promql
# Alert when any dependency has auth_error status
app_dependency_status{status="auth_error"} == 1
```

## Security Best Practices

1. **Never hardcode credentials** — always use environment variables or
   secret management systems
2. **Use short-lived tokens** — if your auth system supports token rotation,
   the checker will use the latest token value passed during creation
3. **Prefer TLS** — enable TLS for HTTP and gRPC checkers when using
   authentication to protect credentials in transit
4. **Limit permissions** — use read-only credentials for health checks;
   `SELECT 1` does not require write permissions

## Full Example

```go
package main

import (
    "context"
    "log"
    "net/http"
    "os"

    "github.com/BigKAA/topologymetrics/sdk-go/dephealth"
    "github.com/prometheus/client_golang/prometheus/promhttp"

    _ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/httpcheck"
    _ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/grpccheck"
    _ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/pgcheck"
    _ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/redischeck"
    _ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/amqpcheck"
)

func main() {
    dh, err := dephealth.New("my-service", "my-team",
        // HTTP with Bearer token
        dephealth.HTTP("payment-api",
            dephealth.FromURL("https://payment.svc:443"),
            dephealth.WithHTTPTLS(true),
            dephealth.WithHTTPBearerToken(os.Getenv("PAYMENT_TOKEN")),
            dephealth.Critical(true),
        ),

        // gRPC with custom metadata
        dephealth.GRPC("user-service",
            dephealth.FromParams("user.svc", "9090"),
            dephealth.WithGRPCMetadata(map[string]string{
                "x-api-key": os.Getenv("USER_API_KEY"),
            }),
            dephealth.Critical(true),
        ),

        // PostgreSQL with credentials in URL
        dephealth.Postgres("postgres-main",
            dephealth.FromURL(os.Getenv("DATABASE_URL")),
            dephealth.Critical(true),
        ),

        // Redis with password
        dephealth.Redis("redis-cache",
            dephealth.FromParams("redis.svc", "6379"),
            dephealth.WithRedisPassword(os.Getenv("REDIS_PASSWORD")),
            dephealth.Critical(false),
        ),

        // AMQP with credentials in URL
        dephealth.AMQP("rabbitmq",
            dephealth.FromParams("rabbitmq.svc", "5672"),
            dephealth.WithAMQPURL(os.Getenv("AMQP_URL")),
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

## See Also

- [Checkers](checkers.md) — detailed guide for all 8 checkers
- [Configuration](configuration.md) — environment variables and options
- [Metrics](metrics.md) — Prometheus metrics including status categories
- [Troubleshooting](troubleshooting.md) — common issues and solutions
