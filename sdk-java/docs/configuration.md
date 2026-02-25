*[Русская версия](configuration.ru.md)*

# Configuration

This guide covers all configuration options for the dephealth Java SDK,
including global settings, per-dependency options, environment variables,
and validation rules.

## Name and Group

```java
DepHealth dh = DepHealth.builder("my-service", "my-team", meterRegistry)
    // ... dependencies
    .build();
```

| Parameter | Required | Validation | Env var fallback |
| --- | --- | --- | --- |
| `name` | Yes | `[a-z][a-z0-9-]*`, 1-63 chars | `DEPHEALTH_NAME` |
| `group` | Yes | `[a-z][a-z0-9-]*`, 1-63 chars | `DEPHEALTH_GROUP` |

Priority: API argument > environment variable.

If both are empty, `builder()` throws a `ConfigurationException`.

## Global Options

Global options are set on the `DepHealth.Builder` and apply to all
dependencies unless overridden per-dependency.

| Option | Type | Default | Range | Description |
| --- | --- | --- | --- | --- |
| `checkInterval(Duration)` | `Duration` | 15s | 1s -- 10m | Interval between health checks |
| `timeout(Duration)` | `Duration` | 5s | 100ms -- 30s | Timeout for a single check |

The third parameter of `builder()` is a Micrometer `MeterRegistry` used for
Prometheus metric export. There is no option to override it after
construction.

### Example

```java
DepHealth dh = DepHealth.builder("my-service", "my-team", meterRegistry)
    .checkInterval(Duration.ofSeconds(30))
    .timeout(Duration.ofSeconds(3))
    // ... dependencies
    .build();
```

## Common Dependency Options

These options can be applied to any dependency type inside the
`.dependency()` lambda.

| Option | Required | Default | Description |
| --- | --- | --- | --- |
| `url(String)` | One of url/jdbcUrl/host+port | -- | Parse host and port from URL |
| `jdbcUrl(String)` | One of url/jdbcUrl/host+port | -- | Parse host and port from JDBC URL |
| `host(String)` + `port(String)` | One of url/jdbcUrl/host+port | -- | Set host and port explicitly |
| `critical(boolean)` | Yes | -- | Mark as critical (`true`) or non-critical (`false`) |
| `label(String, String)` | No | -- | Add a custom Prometheus label |
| `interval(Duration)` | No | global value | Per-dependency check interval |
| `timeout(Duration)` | No | global value | Per-dependency timeout |

### Endpoint Specification

Every dependency requires an endpoint. Use one of three methods:

```java
// From URL -- SDK parses host and port
.dependency("postgres-main", DependencyType.POSTGRES, d -> d
    .url("postgresql://user:pass@pg.svc:5432/mydb")
    .critical(true))

// From JDBC URL -- SDK parses host and port
.dependency("postgres-main", DependencyType.POSTGRES, d -> d
    .jdbcUrl("jdbc:postgresql://pg.svc:5432/mydb")
    .critical(true))

// From explicit host and port
.dependency("postgres-main", DependencyType.POSTGRES, d -> d
    .host("pg.svc")
    .port("5432")
    .critical(true))
```

Supported URL schemes: `http`, `https`, `grpc`, `tcp`, `postgresql`,
`postgres`, `mysql`, `redis`, `rediss`, `amqp`, `amqps`, `kafka`,
`ldap`, `ldaps`.

Supported JDBC subprotocols: `jdbc:postgresql://...`,
`jdbc:mysql://...`.

For Kafka, multi-host URLs are supported:
`kafka://broker1:9092,broker2:9092` -- each host creates a separate endpoint.

### Critical Flag

The `critical()` option is **mandatory** for every dependency. Omitting it
causes a validation error. If not set via API, the SDK checks the
environment variable `DEPHEALTH_<DEP>_CRITICAL` (values: `yes`/`no`,
`true`/`false`).

### Custom Labels

```java
.dependency("postgres-main", DependencyType.POSTGRES, d -> d
    .url(System.getenv("DATABASE_URL"))
    .critical(true)
    .label("role", "primary")
    .label("shard", "eu-west"))
```

Label name validation:

- Must match `[a-zA-Z_][a-zA-Z0-9_]*`
- Cannot use reserved names: `name`, `group`, `dependency`, `type`, `host`, `port`, `critical`

## Checker-Specific Options

### HTTP

| Option | Default | Description |
| --- | --- | --- |
| `httpHealthPath(String)` | `/health` | Health check endpoint path |
| `httpTls(boolean)` | auto (true for `https://`) | Enable HTTPS |
| `httpTlsSkipVerify(boolean)` | `false` | Skip TLS certificate verification |
| `httpHeaders(Map<String, String>)` | -- | Custom HTTP headers |
| `httpBearerToken(String)` | -- | Bearer token authentication |
| `httpBasicAuth(String, String)` | -- | Basic authentication (username, password) |

### gRPC

| Option | Default | Description |
| --- | --- | --- |
| `grpcServiceName(String)` | `""` | Service name (empty = overall server) |
| `grpcTls(boolean)` | `false` | Enable TLS |
| `grpcMetadata(Map<String, String>)` | -- | Custom gRPC metadata |
| `grpcBearerToken(String)` | -- | Bearer token authentication |
| `grpcBasicAuth(String, String)` | -- | Basic authentication (username, password) |

### PostgreSQL

| Option | Default | Description |
| --- | --- | --- |
| `dbUsername(String)` | from URL | Database username |
| `dbPassword(String)` | from URL | Database password |
| `dbDatabase(String)` | from URL | Database name |
| `dbQuery(String)` | `SELECT 1` | SQL query for health check |
| `dataSource(DataSource)` | -- | Connection pool DataSource (preferred) |

### MySQL

| Option | Default | Description |
| --- | --- | --- |
| `dbUsername(String)` | from URL | Database username |
| `dbPassword(String)` | from URL | Database password |
| `dbDatabase(String)` | from URL | Database name |
| `dbQuery(String)` | `SELECT 1` | SQL query for health check |
| `dataSource(DataSource)` | -- | Connection pool DataSource (preferred) |

### Redis

| Option | Default | Description |
| --- | --- | --- |
| `redisPassword(String)` | `""` | Redis password (standalone mode) |
| `redisDb(int)` | `0` | Database number (standalone mode) |
| `jedisPool(JedisPool)` | -- | JedisPool for pool integration (preferred) |

### AMQP

| Option | Default | Description |
| --- | --- | --- |
| `amqpUrl(String)` | -- | Full AMQP URL (overrides host/port/credentials) |
| `amqpUsername(String)` | from URL | AMQP username |
| `amqpPassword(String)` | from URL | AMQP password |
| `amqpVirtualHost(String)` | from URL | AMQP virtual host |

### LDAP

| Option | Default | Description |
| --- | --- | --- |
| `ldapCheckMethod(CheckMethod)` | `ROOT_DSE` | Check method: `ANONYMOUS_BIND`, `SIMPLE_BIND`, `ROOT_DSE`, `SEARCH` |
| `ldapBindDN(String)` | `""` | Bind DN for simple bind or search |
| `ldapBindPassword(String)` | `""` | Bind password |
| `ldapBaseDN(String)` | `""` | Base DN for search operations |
| `ldapSearchFilter(String)` | `(objectClass=*)` | LDAP search filter |
| `ldapSearchScope(LdapSearchScope)` | `BASE` | Search scope: `BASE`, `ONE`, `SUB` |
| `ldapStartTLS(boolean)` | `false` | Enable StartTLS (incompatible with `ldaps://`) |
| `ldapTlsSkipVerify(boolean)` | `false` | Skip TLS certificate verification |
| `ldapConnection(LDAPConnection)` | -- | Existing connection for pool integration |

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
hyphens to underscores, all uppercase.

Example: dependency `"postgres-main"` produces env prefix `DEPHEALTH_POSTGRES_MAIN_`.

### Priority Rules

API values always take precedence over environment variables:

1. **name/group**: API argument > `DEPHEALTH_NAME`/`DEPHEALTH_GROUP` > error
2. **critical**: `critical()` option > `DEPHEALTH_<DEP>_CRITICAL` > error
3. **labels**: `label()` > `DEPHEALTH_<DEP>_LABEL_<KEY>` (API wins on conflict)

### Example

```bash
export DEPHEALTH_NAME=my-service
export DEPHEALTH_GROUP=my-team
export DEPHEALTH_POSTGRES_MAIN_CRITICAL=yes
export DEPHEALTH_POSTGRES_MAIN_LABEL_ROLE=primary
export DEPHEALTH_POSTGRES_MAIN_LABEL_SHARD=eu-west
```

```java
// name and group from env vars, critical and labels from env vars
DepHealth dh = DepHealth.builder("", "", meterRegistry)
    .dependency("postgres-main", DependencyType.POSTGRES, d -> d
        .url(System.getenv("DATABASE_URL")))
        // Critical and labels come from DEPHEALTH_POSTGRES_MAIN_*
    .build();
```

## Option Priority

For interval and timeout, the priority chain is:

```text
per-dependency option > global option > default value
```

| Setting | Per-dependency | Global | Default |
| --- | --- | --- | --- |
| Check interval | `interval(Duration)` | `checkInterval(Duration)` | 15s |
| Timeout | `timeout(Duration)` | `timeout(Duration)` | 5s |

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
| gRPC service name | empty (overall server health) |
| LDAP check method | `ROOT_DSE` |
| LDAP search filter | `(objectClass=*)` |
| LDAP search scope | `BASE` |

## Validation Rules

`build()` validates all configuration and throws a `ConfigurationException`
or `ValidationException` if any rule is violated:

| Rule | Error message |
| --- | --- |
| Missing name | `instance name is required: pass it to builder() or set DEPHEALTH_NAME` |
| Missing group | `group is required: pass it to builder() or set DEPHEALTH_GROUP` |
| Invalid name/group format | `instance name must match [a-z][a-z0-9-]*, got '...'` |
| Name too long | `instance name must be 1-63 characters, got '...' (N chars)` |
| Missing critical for dependency | validation error from env var fallback |
| Missing URL or host/port | `Dependency must have url, jdbcUrl, or host+port configured` |
| Invalid label name | `label name must match [a-zA-Z_][a-zA-Z0-9_]*, got '...'` |
| Reserved label name | `label name '...' is reserved and cannot be used as a custom label` |
| Timeout >= interval | `timeout (...) must be less than interval (...)` |
| LDAP simple_bind without credentials | `LDAP simple_bind requires bindDN and bindPassword` |
| LDAP search without baseDN | `LDAP search requires baseDN` |
| LDAP startTLS + ldaps | `startTLS and ldaps:// are incompatible` |

## See Also

- [Getting Started](getting-started.md) -- basic setup and first example
- [Checkers](checkers.md) -- checker-specific options in detail
- [Authentication](authentication.md) -- auth options for HTTP and gRPC
- [Connection Pools](connection-pools.md) -- integration with DataSource and JedisPool
- [Spring Boot Integration](spring-boot.md) -- auto-configuration and YAML
- [API Reference](api-reference.md) -- complete reference of all public classes
- [Troubleshooting](troubleshooting.md) -- common issues and solutions
