*[Русская версия](configuration.ru.md)*

# Configuration

This guide covers all configuration options for the dephealth Python SDK,
including global settings, per-dependency options, environment variables,
and validation rules.

## Name and Group

```python
dh = DependencyHealth("my-service", "my-team",
    # ... dependency specs
)
```

| Parameter | Required | Validation | Env var fallback |
| --- | --- | --- | --- |
| `name` | Yes | `[a-z][a-z0-9-]*`, 1-63 chars | `DEPHEALTH_NAME` |
| `group` | Yes | `[a-z][a-z0-9-]*`, 1-63 chars | `DEPHEALTH_GROUP` |

Priority: API argument > environment variable.

If both are empty, `DependencyHealth()` raises `ValueError`.

## Global Options

Global options are passed to the `DependencyHealth` constructor and apply to
all dependencies unless overridden per-dependency.

| Option | Type | Default | Range | Description |
| --- | --- | --- | --- | --- |
| `check_interval` | `timedelta \| None` | 15s | 1s -- 5m | Interval between health checks |
| `timeout` | `timedelta \| None` | 5s | 1s -- 60s | Timeout for a single check |
| `registry` | `CollectorRegistry \| None` | default | -- | Custom Prometheus registry |
| `log` | `Logger \| None` | `dephealth` | -- | Custom logger instance |

### Example

```python
from datetime import timedelta
from prometheus_client import CollectorRegistry

custom_registry = CollectorRegistry()

dh = DependencyHealth("my-service", "my-team",
    check_interval=timedelta(seconds=30),
    timeout=timedelta(seconds=3),
    registry=custom_registry,
    # ... dependency specs
)
```

## Common Dependency Options

These options can be applied to any factory function.

| Option | Required | Default | Description |
| --- | --- | --- | --- |
| `url` | One of url/host+port | `""` | Parse host and port from URL |
| `host` + `port` | One of url/host+port | -- | Set host and port explicitly |
| `critical` | Yes | -- | Mark as critical (`True`) or non-critical (`False`) |
| `labels` | No | `None` | Custom Prometheus labels dict |
| `interval` | No | global value | Per-dependency check interval |
| `timeout` | No | global value | Per-dependency timeout |

### Endpoint Specification

Every dependency requires an endpoint. Use one of two methods:

```python
# From URL -- SDK parses host and port
postgres_check("postgres-main",
    url="postgresql://user:pass@pg.svc:5432/mydb",
    critical=True,
)

# From explicit host and port
grpc_check("user-service",
    host="user.svc",
    port="9090",
    critical=True,
)
```

Supported URL schemes: `http`, `https`, `grpc`, `tcp`, `postgresql`,
`postgres`, `mysql`, `redis`, `rediss`, `amqp`, `amqps`, `kafka`,
`ldap`, `ldaps`.

For Kafka, multi-host URLs are supported:
`kafka://broker1:9092,broker2:9092` — each host creates a separate endpoint.

### Critical Flag

The `critical` option is **mandatory** for every dependency. If not set via
API, the SDK checks the environment variable `DEPHEALTH_<DEP>_CRITICAL`
(values: `yes`/`no`, `true`/`false`).

### Custom Labels

```python
postgres_check("postgres-main",
    url="postgresql://user:pass@pg.svc:5432/mydb",
    critical=True,
    labels={"role": "primary", "shard": "eu-west"},
)
```

Label name validation:

- Must match `[a-zA-Z_][a-zA-Z0-9_]*`
- Cannot use reserved names: `name`, `group`, `dependency`, `type`, `host`, `port`, `critical`

## Checker-Specific Options

### HTTP

| Option | Default | Description |
| --- | --- | --- |
| `health_path` | `"/health"` | Health check endpoint path |
| `tls` | `False` | Enable HTTPS |
| `tls_skip_verify` | `False` | Skip TLS certificate verification |
| `headers` | `None` | Custom HTTP headers |
| `bearer_token` | `None` | Bearer token authentication |
| `basic_auth` | `None` | Basic auth `(user, password)` tuple |

### gRPC

| Option | Default | Description |
| --- | --- | --- |
| `service_name` | `""` | Service name (empty = overall server) |
| `tls` | `False` | Enable TLS |
| `tls_skip_verify` | `False` | Skip TLS certificate verification |
| `metadata` | `None` | Custom gRPC metadata |
| `bearer_token` | `None` | Bearer token authentication |
| `basic_auth` | `None` | Basic auth `(user, password)` tuple |

### PostgreSQL

| Option | Default | Description |
| --- | --- | --- |
| `query` | `"SELECT 1"` | SQL query for health check |
| `pool` | `None` | asyncpg pool for pool integration (preferred) |

### MySQL

| Option | Default | Description |
| --- | --- | --- |
| `query` | `"SELECT 1"` | SQL query for health check |
| `pool` | `None` | aiomysql pool for pool integration (preferred) |

### Redis

| Option | Default | Description |
| --- | --- | --- |
| `password` | `None` | Redis password (standalone mode) |
| `db` | `None` | Database number (standalone mode) |
| `client` | `None` | redis-py async client for pool integration (preferred) |

### AMQP

No checker-specific options beyond `url` or `host`/`port`.

### Kafka

No checker-specific options beyond `url` or `host`/`port`.

### LDAP

| Option | Default | Description |
| --- | --- | --- |
| `check_method` | `ROOT_DSE` | Check method: `ANONYMOUS_BIND`, `SIMPLE_BIND`, `ROOT_DSE`, `SEARCH` |
| `bind_dn` | `""` | Bind DN for simple bind or search |
| `bind_password` | `""` | Bind password |
| `base_dn` | `""` | Base DN for search operations |
| `search_filter` | `"(objectClass=*)"` | LDAP search filter |
| `search_scope` | `BASE` | Search scope: `BASE`, `ONE`, `SUB` |
| `start_tls` | `False` | Enable StartTLS (incompatible with `ldaps://`) |
| `tls_skip_verify` | `False` | Skip TLS certificate verification |
| `client` | `None` | ldap3 Connection for pool integration |

### TCP

No checker-specific options beyond `host`/`port`.

## Environment Variables

| Variable | Description | Example |
| --- | --- | --- |
| `DEPHEALTH_NAME` | Application name (fallback if API arg is empty) | `my-service` |
| `DEPHEALTH_GROUP` | Logical group (fallback if API arg is empty) | `my-team` |
| `DEPHEALTH_<DEP>_CRITICAL` | Dependency criticality (`yes`/`no`) | `yes` |
| `DEPHEALTH_<DEP>_LABEL_<KEY>` | Custom label value | `primary` |

`<DEP>` is the dependency name converted to UPPER_SNAKE_CASE:
hyphens to underscores, all uppercase.

Example: dependency `"postgres-main"` produces env prefix
`DEPHEALTH_POSTGRES_MAIN_`.

### Priority Rules

API values always take precedence over environment variables:

1. **name/group**: API argument > `DEPHEALTH_NAME`/`DEPHEALTH_GROUP` > error
2. **critical**: `critical=` option > `DEPHEALTH_<DEP>_CRITICAL` > error
3. **labels**: `labels=` > `DEPHEALTH_<DEP>_LABEL_<KEY>` (API wins on conflict)

### Example

```bash
export DEPHEALTH_NAME=my-service
export DEPHEALTH_GROUP=my-team
export DEPHEALTH_POSTGRES_MAIN_CRITICAL=yes
export DEPHEALTH_POSTGRES_MAIN_LABEL_ROLE=primary
export DEPHEALTH_POSTGRES_MAIN_LABEL_SHARD=eu-west
```

```python
# name and group from env vars, critical and labels from env vars
dh = DependencyHealth("", "",
    postgres_check("postgres-main",
        url=os.environ["DATABASE_URL"],
    ),
    # Critical and labels come from DEPHEALTH_POSTGRES_MAIN_*
)
```

## Option Priority

For interval and timeout, the priority chain is:

```text
per-dependency option > global option > default value
```

| Setting | Per-dependency | Global | Default |
| --- | --- | --- | --- |
| Check interval | `interval=` | `check_interval=` | 15s |
| Timeout | `timeout=` | `timeout=` | 5s |

## Default Values

| Parameter | Value |
| --- | --- |
| Check interval | 15 seconds |
| Timeout | 5 seconds |
| Initial delay | 5 seconds |
| Failure threshold | 1 |
| Success threshold | 1 |
| HTTP health path | `/health` |
| HTTP TLS | `False` |
| Redis DB | `None` |
| Redis password | `None` |
| PostgreSQL query | `SELECT 1` |
| MySQL query | `SELECT 1` |
| gRPC service name | `""` (overall server health) |
| LDAP check method | `ROOT_DSE` |
| LDAP search filter | `(objectClass=*)` |
| LDAP search scope | `BASE` |

## Validation Rules

`DependencyHealth()` validates all configuration and raises `ValueError`
if any rule is violated:

| Rule | Error |
| --- | --- |
| Missing name | `instance name is required: pass it as argument or set DEPHEALTH_NAME` |
| Missing group | `group is required: pass it as argument or set DEPHEALTH_GROUP` |
| Invalid name/group format | `instance name must match [a-z][a-z0-9-]*, got '...'` |
| Name too long | `instance name must be 1-63 characters` |
| Missing critical | validation error |
| Missing URL or host/port | dependency configuration error |
| Invalid label name | `label name must match [a-zA-Z_][a-zA-Z0-9_]*, got '...'` |
| Reserved label name | `label name '...' is reserved` |
| LDAP simple_bind without credentials | `LDAP simple_bind requires bind_dn and bind_password` |
| LDAP search without base_dn | `LDAP search requires base_dn` |
| LDAP start_tls + ldaps | `start_tls and ldaps:// are incompatible` |

## See Also

- [Getting Started](getting-started.md) — basic setup and first example
- [Checkers](checkers.md) — checker-specific options in detail
- [Authentication](authentication.md) — auth options for HTTP and gRPC
- [Connection Pools](connection-pools.md) — integration with asyncpg, redis-py, aiomysql
- [FastAPI Integration](fastapi.md) — lifespan and middleware configuration
- [API Reference](api-reference.md) — complete reference of all public classes
- [Troubleshooting](troubleshooting.md) — common issues and solutions
