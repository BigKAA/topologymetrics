*[Русская версия](configuration.ru.md)*

# Configuration

This guide covers all configuration options for the dephealth C# SDK,
including global settings, per-dependency options, environment variables,
and validation rules.

## Name and Group

```csharp
var dh = DepHealthMonitor.CreateBuilder("my-service", "my-team")
    // ... dependencies
    .Build();
```

| Parameter | Required | Validation | Env var fallback |
| --- | --- | --- | --- |
| `name` | Yes | `[a-z][a-z0-9-]*`, 1-63 chars | `DEPHEALTH_NAME` |
| `group` | Yes | `[a-z][a-z0-9-]*`, 1-63 chars | `DEPHEALTH_GROUP` |

Priority: API argument > environment variable.

If both are empty, `CreateBuilder()` throws a `ValidationException`.

## Global Options

Global options are set on the `DepHealthMonitor.Builder` and apply to all
dependencies unless overridden per-dependency.

| Option | Type | Default | Range | Description |
| --- | --- | --- | --- | --- |
| `WithCheckInterval(TimeSpan)` | `TimeSpan` | 15s | 1s -- 10m | Interval between health checks |
| `WithCheckTimeout(TimeSpan)` | `TimeSpan` | 5s | 100ms -- 30s | Timeout for a single check |
| `WithInitialDelay(TimeSpan)` | `TimeSpan` | 0s | 0 -- 5m | Delay before first check |
| `WithRegistry(CollectorRegistry)` | `CollectorRegistry` | default | -- | Custom Prometheus registry |
| `WithLogger(ILogger)` | `ILogger` | null | -- | Logger for diagnostic messages |

### Example

```csharp
var dh = DepHealthMonitor.CreateBuilder("my-service", "my-team")
    .WithCheckInterval(TimeSpan.FromSeconds(30))
    .WithCheckTimeout(TimeSpan.FromSeconds(3))
    .WithInitialDelay(TimeSpan.FromSeconds(5))
    // ... dependencies
    .Build();
```

## Common Dependency Options

These options can be applied to any dependency type via the `Add*` builder
methods.

| Option | Required | Default | Description |
| --- | --- | --- | --- |
| `url` / `host` + `port` | One of url/host+port | -- | Endpoint specification (see below) |
| `critical` | Yes | -- | Mark as critical (`true`) or non-critical (`false`) |
| `labels` | No | -- | Custom Prometheus labels (`Dictionary<string, string>`) |

### Endpoint Specification

Every dependency requires an endpoint. Use one of two methods:

```csharp
// From URL -- SDK parses host and port
builder.AddPostgres("postgres-main",
    "postgresql://user:pass@pg.svc:5432/mydb",
    critical: true)

// From JDBC URL -- SDK parses host and port
builder.AddPostgres("postgres-main",
    "jdbc:postgresql://pg.svc:5432/mydb",
    critical: true)

// From explicit host and port (gRPC, TCP, LDAP)
builder.AddGrpc("payments-grpc",
    host: "payments.svc",
    port: "9090",
    critical: true)
```

Supported URL schemes: `http`, `https`, `grpc`, `tcp`, `postgresql`,
`postgres`, `mysql`, `redis`, `rediss`, `amqp`, `amqps`, `kafka`,
`ldap`, `ldaps`.

Supported JDBC subprotocols: `jdbc:postgresql://...`,
`jdbc:mysql://...`.

For Kafka, multi-host URLs are supported:
`kafka://broker1:9092,broker2:9092` -- each host creates a separate endpoint.

### Critical Flag

The `critical` option is **mandatory** for every dependency. Omitting it
causes a validation error at `Build()`. If not set via API, the SDK checks
the environment variable `DEPHEALTH_<DEP>_CRITICAL` (values: `yes`/`no`,
`true`/`false`).

### Custom Labels

```csharp
builder.AddPostgres("postgres-main",
    url: Environment.GetEnvironmentVariable("DATABASE_URL")!,
    critical: true,
    labels: new Dictionary<string, string>
    {
        ["role"] = "primary",
        ["shard"] = "eu-west"
    })
```

Label name validation:

- Must match `[a-zA-Z_][a-zA-Z0-9_]*`
- Cannot use reserved names: `name`, `group`, `dependency`, `type`, `host`, `port`, `critical`

## Checker-Specific Options

### HTTP

```csharp
builder.AddHttp(
    name: "payment-api",
    url: "https://payment.svc:8443",
    healthPath: "/healthz",
    critical: true,
    headers: new Dictionary<string, string> { ["X-Internal"] = "1" },
    bearerToken: "secret-token")
```

| Option | Default | Description |
| --- | --- | --- |
| `healthPath` | `/health` | Health check endpoint path |
| `headers` | -- | Custom HTTP headers (`Dictionary<string, string>`) |
| `bearerToken` | -- | Bearer token authentication |
| `basicAuthUsername` | -- | Basic authentication username |
| `basicAuthPassword` | -- | Basic authentication password |

TLS is enabled automatically for `https://` URLs. Only one authentication
method (`bearerToken`, `basicAuth`, or `Authorization` header) may be
specified at a time; mixing them throws a `ValidationException`.

### gRPC

```csharp
builder.AddGrpc(
    name: "inventory-grpc",
    host: "inventory.svc",
    port: "9090",
    tlsEnabled: true,
    critical: false,
    metadata: new Dictionary<string, string> { ["x-request-id"] = "health" },
    bearerToken: "token")
```

| Option | Default | Description |
| --- | --- | --- |
| `tlsEnabled` | `false` | Enable TLS for the gRPC channel |
| `metadata` | -- | Custom gRPC metadata (`Dictionary<string, string>`) |
| `bearerToken` | -- | Bearer token authentication |
| `basicAuthUsername` | -- | Basic authentication username |
| `basicAuthPassword` | -- | Basic authentication password |

The gRPC checker uses the standard
[gRPC Health Checking Protocol](https://github.com/grpc/grpc/blob/master/doc/health-checking.md).
Only one authentication method may be specified at a time.

### PostgreSQL

```csharp
// Standalone mode -- creates a new connection per check
builder.AddPostgres("db-primary",
    url: "postgresql://user:pass@pg.svc:5432/mydb",
    critical: true)

// Pool mode -- reuses NpgsqlDataSource (preferred)
var dataSource = NpgsqlDataSource.Create(connectionString);
builder.AddCustom("db-primary", DependencyType.Postgres,
    host: "pg.svc", port: "5432",
    checker: new PostgresChecker(dataSource),
    critical: true)

// Entity Framework integration
builder.AddNpgsqlFromContext("db-primary", dbContext, critical: true)
```

| Option | Default | Description |
| --- | --- | --- |
| `url` | -- | PostgreSQL connection URL (credentials parsed automatically) |
| `NpgsqlDataSource` | -- | Existing data source for pool integration (preferred) |

Credentials (username, password, database) are parsed from the URL
automatically. The SQL health check query is `SELECT 1`.

### MySQL

```csharp
// Standalone mode -- creates a new connection per check
builder.AddMySql("mysql-main",
    url: "mysql://user:pass@mysql.svc:3306/mydb",
    critical: true)
```

| Option | Default | Description |
| --- | --- | --- |
| `url` | -- | MySQL connection URL (credentials parsed automatically) |

The SQL health check query is `SELECT 1`.

### Redis

```csharp
// Standalone mode -- creates a new connection per check
builder.AddRedis("cache",
    url: "redis://:password@redis.svc:6379",
    critical: false)

// Pool mode -- reuses IConnectionMultiplexer (preferred)
var multiplexer = await ConnectionMultiplexer.ConnectAsync("redis.svc:6379");
builder.AddCustom("cache", DependencyType.Redis,
    host: "redis.svc", port: "6379",
    checker: new RedisChecker(multiplexer),
    critical: false)
```

| Option | Default | Description |
| --- | --- | --- |
| `url` | -- | Redis connection URL (`redis://` or `rediss://`) |
| `IConnectionMultiplexer` | -- | Existing multiplexer for pool integration (preferred) |

The Redis health check sends a `PING` command and expects a `PONG` response.

### AMQP

```csharp
builder.AddAmqp("rabbitmq",
    url: "amqp://user:pass@rabbit.svc:5672/my-vhost",
    critical: true)
```

| Option | Default | Description |
| --- | --- | --- |
| `url` | -- | AMQP connection URL (credentials and vhost parsed automatically) |

Credentials (username, password) and virtual host are parsed from the URL.
Default virtual host is `/`.

### LDAP

```csharp
builder.AddLdap(
    name: "ldap-corp",
    host: "ldap.corp.example.com",
    port: "389",
    checkMethod: LdapCheckMethod.SimpleBind,
    bindDN: "cn=health,dc=corp,dc=example,dc=com",
    bindPassword: "secret",
    critical: false)
```

| Option | Default | Description |
| --- | --- | --- |
| `checkMethod` | `RootDse` | Check method: `AnonymousBind`, `SimpleBind`, `RootDse`, `Search` |
| `bindDN` | `""` | Bind DN for simple bind or search |
| `bindPassword` | `""` | Bind password |
| `baseDN` | `""` | Base DN for search operations |
| `searchFilter` | `(objectClass=*)` | LDAP search filter |
| `searchScope` | `Base` | Search scope: `Base`, `One`, `Sub` |
| `useTls` | `false` | Enable LDAPS (TLS) -- use with `ldaps://` |
| `startTls` | `false` | Enable StartTLS (incompatible with `useTls: true`) |
| `tlsSkipVerify` | `false` | Skip TLS certificate verification |

### TCP and Kafka

```csharp
// TCP
builder.AddTcp("legacy-tcp", host: "legacy.svc", port: "8080", critical: false)

// Kafka
builder.AddKafka("events",
    url: "kafka://broker1:9092,broker2:9092",
    critical: true)
```

No checker-specific options for TCP or Kafka. The TCP checker establishes
a raw TCP connection. The Kafka checker connects to each broker individually.

## Environment Variables

| Variable | Description | Example |
| --- | --- | --- |
| `DEPHEALTH_NAME` | Application name (fallback if API arg is empty) | `my-service` |
| `DEPHEALTH_GROUP` | Logical group (fallback if API arg is empty) | `my-team` |
| `DEPHEALTH_<DEP>_CRITICAL` | Dependency criticality (`yes`/`no`, `true`/`false`) | `yes` |
| `DEPHEALTH_<DEP>_LABEL_<KEY>` | Custom label value | `primary` |

`<DEP>` is the dependency name converted to UPPER_SNAKE_CASE:
hyphens to underscores, all uppercase.

Example: dependency `"postgres-main"` produces env prefix `DEPHEALTH_POSTGRES_MAIN_`.

### Priority Rules

API values always take precedence over environment variables:

1. **name/group**: API argument > `DEPHEALTH_NAME`/`DEPHEALTH_GROUP` > error
2. **critical**: `critical` option > `DEPHEALTH_<DEP>_CRITICAL` > error
3. **labels**: `labels` dictionary > `DEPHEALTH_<DEP>_LABEL_<KEY>` (API wins on conflict)

### Example

```bash
export DEPHEALTH_NAME=my-service
export DEPHEALTH_GROUP=my-team
export DEPHEALTH_POSTGRES_MAIN_CRITICAL=yes
export DEPHEALTH_POSTGRES_MAIN_LABEL_ROLE=primary
export DEPHEALTH_POSTGRES_MAIN_LABEL_SHARD=eu-west
```

```csharp
// name and group from env vars, critical and labels from env vars
var dh = DepHealthMonitor.CreateBuilder("", "")
    .AddPostgres("postgres-main",
        url: Environment.GetEnvironmentVariable("DATABASE_URL")!)
        // critical and labels come from DEPHEALTH_POSTGRES_MAIN_*
    .Build();
```

## Option Priority

For interval and timeout, the priority chain is:

```text
per-dependency option > global option > default value
```

The C# SDK currently applies global options to all dependencies at build time.
Per-dependency overrides can be achieved by using `AddCustom` with a manually
constructed `CheckConfig`.

| Setting | Global option | Default |
| --- | --- | --- |
| Check interval | `WithCheckInterval(TimeSpan)` | 15s |
| Timeout | `WithCheckTimeout(TimeSpan)` | 5s |
| Initial delay | `WithInitialDelay(TimeSpan)` | 0s |

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
| gRPC TLS | `false` |
| LDAP check method | `RootDse` |
| LDAP search filter | `(objectClass=*)` |
| LDAP search scope | `Base` |

## Validation Rules

`Build()` validates all configuration and throws a `ValidationException`
or `ConfigurationException` if any rule is violated:

| Rule | Error message |
| --- | --- |
| Missing name | `instance name must not be empty` |
| Missing group | `group is required: pass to CreateBuilder() or set DEPHEALTH_GROUP` |
| Invalid name/group format | `instance name must match ^[a-z][a-z0-9-]*$, got '...'` |
| Name too long | `instance name must be 1-63 characters, got '...' (N chars)` |
| Missing URL or host/port | `URL must have a scheme (e.g. http://)` |
| Unsupported URL scheme | `Unsupported URL scheme: ...` |
| Invalid port | `Invalid port: '...' in ...` |
| Port out of range | `Port out of range (1-65535): ... in ...` |
| Invalid label name | `label name must match [a-zA-Z_][a-zA-Z0-9_]*, got '...'` |
| Reserved label name | `label name '...' is reserved` |
| Timeout >= interval | `timeout (...) must be less than interval (...)` |
| Interval out of range | `interval must be between 00:00:01 and 00:10:00, got ...` |
| Timeout out of range | `timeout must be between 00:00:00.1000000 and 00:00:30, got ...` |
| Conflicting auth methods (HTTP/gRPC) | `conflicting auth methods: specify only one of bearerToken, basicAuth, or Authorization header` |
| LDAP SimpleBind without credentials | `LDAP simple_bind requires bindDN and bindPassword` |
| LDAP Search without baseDN | `LDAP search requires baseDN` |
| LDAP startTLS + useTls | `startTLS and ldaps:// are incompatible` |

## See Also

- [Getting Started](getting-started.md) -- basic setup and first example
- [Checkers](checkers.md) -- checker-specific options in detail
- [Authentication](authentication.md) -- auth options for HTTP and gRPC
- [Connection Pools](connection-pools.md) -- integration with NpgsqlDataSource and IConnectionMultiplexer
- [ASP.NET Core Integration](aspnetcore.md) -- hosted service and health checks
- [API Reference](api-reference.md) -- complete reference of all public classes
- [Troubleshooting](troubleshooting.md) -- common issues and solutions
