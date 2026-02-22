*[Русская версия](configuration.ru.md)*

# Configuration

This guide covers all configuration options for the dephealth Go SDK,
including global settings, per-dependency options, environment variables,
and validation rules.

## Name and Group

```go
dh, err := dephealth.New("my-service", "my-team", ...)
```

| Parameter | Required | Validation | Env var fallback |
| --- | --- | --- | --- |
| `name` | Yes | `[a-z][a-z0-9-]*`, 1-63 chars | `DEPHEALTH_NAME` |
| `group` | Yes | `[a-z][a-z0-9-]*`, 1-63 chars | `DEPHEALTH_GROUP` |

Priority: API argument > environment variable.

If both are empty, `New()` returns an error.

## Global Options

Global options are passed to `dephealth.New()` and apply to all
dependencies unless overridden per-dependency.

| Option | Type | Default | Range | Description |
| --- | --- | --- | --- | --- |
| `WithCheckInterval(d)` | `time.Duration` | 15s | 1s – 10m | Interval between health checks |
| `WithTimeout(d)` | `time.Duration` | 5s | 100ms – 30s | Timeout for a single check |
| `WithRegisterer(r)` | `prometheus.Registerer` | `prometheus.DefaultRegisterer` | — | Custom Prometheus registerer |
| `WithLogger(l)` | `*slog.Logger` | none | — | Logger for SDK operations |

### Example

```go
dh, err := dephealth.New("my-service", "my-team",
    dephealth.WithCheckInterval(30 * time.Second),
    dephealth.WithTimeout(3 * time.Second),
    dephealth.WithLogger(slog.Default()),
    dephealth.WithRegisterer(prometheus.NewRegistry()),
    // ... dependencies
)
```

## Common Dependency Options

These options can be applied to any dependency type.

| Option | Required | Default | Description |
| --- | --- | --- | --- |
| `FromURL(url)` | One of FromURL/FromParams | — | Parse host and port from URL |
| `FromParams(host, port)` | One of FromURL/FromParams | — | Set host and port explicitly |
| `Critical(v)` | Yes | — | Mark as critical (`true`) or non-critical (`false`) |
| `WithLabel(key, value)` | No | — | Add a custom Prometheus label |
| `CheckInterval(d)` | No | global value | Per-dependency check interval |
| `Timeout(d)` | No | global value | Per-dependency timeout |

### Endpoint Specification

Every dependency requires an endpoint. Use one of:

```go
// From URL — SDK parses host and port
dephealth.FromURL("postgresql://user:pass@pg.svc:5432/mydb")

// From parameters — explicit host and port
dephealth.FromParams("pg.svc", "5432")
```

Supported URL schemes: `http`, `https`, `grpc`, `tcp`, `postgresql`,
`postgres`, `mysql`, `redis`, `rediss`, `amqp`, `amqps`, `kafka`.

For Kafka, multi-host URLs are supported:
`kafka://broker1:9092,broker2:9092` — each host creates a separate endpoint.

### Critical Flag

The `Critical()` option is **mandatory** for every dependency. Omitting it
causes a validation error. If not set via API, the SDK checks the
environment variable `DEPHEALTH_<DEP>_CRITICAL` (values: `yes`/`no`,
`true`/`false`).

### Custom Labels

```go
dephealth.Postgres("postgres-main",
    dephealth.FromURL(os.Getenv("DATABASE_URL")),
    dephealth.Critical(true),
    dephealth.WithLabel("role", "primary"),
    dephealth.WithLabel("shard", "eu-west"),
)
```

Label name validation:

- Must match `[a-z_][a-z0-9_]*`
- Cannot use reserved names: `name`, `group`, `dependency`, `type`, `host`, `port`, `critical`, `status`, `detail`

## Checker-Specific Options

### HTTP

| Option | Default | Description |
| --- | --- | --- |
| `WithHTTPHealthPath(path)` | `/health` | Health check endpoint path |
| `WithHTTPTLS(enabled)` | auto (true for `https://`) | Enable HTTPS |
| `WithHTTPTLSSkipVerify(skip)` | `false` | Skip TLS certificate verification |
| `WithHTTPHeaders(headers)` | — | Custom HTTP headers |
| `WithHTTPBearerToken(token)` | — | Bearer token authentication |
| `WithHTTPBasicAuth(user, pass)` | — | Basic authentication |

### gRPC

| Option | Default | Description |
| --- | --- | --- |
| `WithGRPCServiceName(name)` | `""` | Service name (empty = overall server) |
| `WithGRPCTLS(enabled)` | `false` | Enable TLS |
| `WithGRPCTLSSkipVerify(skip)` | `false` | Skip TLS certificate verification |
| `WithGRPCMetadata(md)` | — | Custom gRPC metadata |
| `WithGRPCBearerToken(token)` | — | Bearer token authentication |
| `WithGRPCBasicAuth(user, pass)` | — | Basic authentication |

### PostgreSQL

| Option | Default | Description |
| --- | --- | --- |
| `WithPostgresQuery(query)` | `SELECT 1` | SQL query for health check |

### MySQL

| Option | Default | Description |
| --- | --- | --- |
| `WithMySQLQuery(query)` | `SELECT 1` | SQL query for health check |

### Redis

| Option | Default | Description |
| --- | --- | --- |
| `WithRedisPassword(password)` | `""` | Redis password (standalone mode) |
| `WithRedisDB(db)` | `0` | Database number (standalone mode) |

### AMQP

| Option | Default | Description |
| --- | --- | --- |
| `WithAMQPURL(url)` | `amqp://guest:guest@host:port/` | Full AMQP URL |

### TCP and Kafka

No checker-specific options.

## Environment Variables

| Variable | Description | Example |
| --- | --- | --- |
| `DEPHEALTH_NAME` | Application name (fallback if API arg is empty) | `my-service` |
| `DEPHEALTH_GROUP` | Logical group (fallback if API arg is empty) | `my-team` |
| `DEPHEALTH_<DEP>_CRITICAL` | Dependency criticality (`yes`/`no`) | `yes` |
| `DEPHEALTH_<DEP>_LABEL_<KEY>` | Custom label value | `primary` |

`<DEP>` is the dependency name converted to UPPER_SNAKE_CASE:
hyphens → underscores, all uppercase.

Example: dependency `"postgres-main"` → env prefix `DEPHEALTH_POSTGRES_MAIN_`.

### Priority Rules

API values always take precedence over environment variables:

1. **name/group**: API argument > `DEPHEALTH_NAME`/`DEPHEALTH_GROUP` > error
2. **critical**: `Critical()` option > `DEPHEALTH_<DEP>_CRITICAL` > error
3. **labels**: `WithLabel()` > `DEPHEALTH_<DEP>_LABEL_<KEY>` (API wins on conflict)

### Example

```bash
export DEPHEALTH_NAME=my-service
export DEPHEALTH_GROUP=my-team
export DEPHEALTH_POSTGRES_MAIN_CRITICAL=yes
export DEPHEALTH_POSTGRES_MAIN_LABEL_ROLE=primary
export DEPHEALTH_POSTGRES_MAIN_LABEL_SHARD=eu-west
```

```go
// name and group from env vars, critical and labels from env vars
dh, err := dephealth.New("", "",
    dephealth.Postgres("postgres-main",
        dephealth.FromURL(os.Getenv("DATABASE_URL")),
        // Critical and labels come from DEPHEALTH_POSTGRES_MAIN_*
    ),
)
```

## Option Priority

For interval and timeout, the priority chain is:

```text
per-dependency option > global option > default value
```

| Setting | Per-dependency | Global | Default |
| --- | --- | --- | --- |
| Check interval | `CheckInterval(d)` | `WithCheckInterval(d)` | 15s |
| Timeout | `Timeout(d)` | `WithTimeout(d)` | 5s |

## Default Values

| Parameter | Value |
| --- | --- |
| Check interval | 15 seconds |
| Timeout | 5 seconds |
| Initial delay | 0 (no delay) |
| Failure threshold | 1 |
| Success threshold | 1 |
| HTTP health path | `/health` |
| HTTP TLS | `false` (auto-enabled for `https://` URLs) |
| Redis DB | `0` |
| Redis password | empty |
| PostgreSQL query | `SELECT 1` |
| MySQL query | `SELECT 1` |
| AMQP URL | `amqp://guest:guest@host:port/` |
| gRPC service name | empty (overall server health) |

## Validation Rules

`New()` validates all configuration and returns an error if any rule
is violated:

| Rule | Error message |
| --- | --- |
| Missing name | `missing name: pass as first argument or set DEPHEALTH_NAME` |
| Missing group | `missing group: pass as second argument or set DEPHEALTH_GROUP` |
| Invalid name/group format | `invalid name: must match [a-z][a-z0-9-]*, 1-63 chars` |
| Missing Critical for dependency | `missing critical for dependency "..."` |
| Missing URL or host/port | `missing URL or host/port parameters` |
| Invalid label name | `invalid label name: ...` |
| Reserved label name | `reserved label: ...` |
| Conflicting auth methods | `conflicting auth methods: specify only one of ...` |
| No checker factory registered | `no checker factory registered for type "..."` |

## See Also

- [Getting Started](getting-started.md) — basic setup and first example
- [Checkers](checkers.md) — checker-specific options in detail
- [Authentication](authentication.md) — auth options for HTTP and gRPC
- [API Reference](api-reference.md) — complete reference of all symbols
- [Troubleshooting](troubleshooting.md) — common issues and solutions
