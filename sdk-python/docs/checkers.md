*[Русская версия](checkers.ru.md)*

# Health Checkers

This guide covers all 9 built-in health checkers in the dephealth Python SDK.

## Overview

| Checker | Factory | Extra | Check Method |
| --- | --- | --- | --- |
| [HTTP](#http) | `http_check()` | — | GET request to health path |
| [gRPC](#grpc) | `grpc_check()` | `grpc` | gRPC Health/Check protocol |
| [TCP](#tcp) | `tcp_check()` | — | TCP connection |
| [PostgreSQL](#postgresql) | `postgres_check()` | `postgres` | SQL query execution |
| [MySQL](#mysql) | `mysql_check()` | `mysql` | SQL query execution |
| [Redis](#redis) | `redis_check()` | `redis` | PING command |
| [AMQP](#amqp) | `amqp_check()` | `amqp` | Connection test |
| [Kafka](#kafka) | `kafka_check()` | `kafka` | Cluster metadata fetch |
| [LDAP](#ldap) | `ldap_check()` | `ldap` | Bind/search/RootDSE |

## HTTP

Performs an HTTP GET request to the configured health path.

### Options

| Option | Default | Description |
| --- | --- | --- |
| `url` | `""` | HTTP URL (parsed for host/port/scheme) |
| `host` | `""` | Host (if `url` not provided) |
| `port` | `"80"` | Port |
| `health_path` | `"/health"` | Health endpoint path |
| `tls` | `False` | Enable HTTPS |
| `tls_skip_verify` | `False` | Skip TLS certificate verification |
| `headers` | `None` | Custom HTTP headers |
| `bearer_token` | `None` | Bearer token for Authorization header |
| `basic_auth` | `None` | Basic auth `(user, password)` tuple |

### Example

```python
from dephealth.api import http_check

# Basic
http_check("payment-api",
    url="http://payment.svc:8080",
    critical=True,
)

# With authentication
http_check("secure-api",
    url="https://api.svc:443",
    health_path="/healthz",
    bearer_token="eyJhbG...",
    critical=True,
)

# With custom headers
http_check("internal-api",
    url="http://api.svc:8080",
    headers={"X-Api-Key": "secret"},
    critical=False,
)
```

### Error Classification

| Error | Status Category | Status Detail |
| --- | --- | --- |
| Connection refused | `connection_error` | `connection_refused` |
| DNS resolution failed | `dns_error` | `dns_error` |
| Request timed out | `timeout` | `timeout` |
| TLS error | `tls_error` | `tls_error` |
| HTTP 401/403 | `auth_error` | `auth_error` |
| HTTP >= 300 | `unhealthy` | `http_<status_code>` |

## gRPC

Uses the standard gRPC Health Checking Protocol
(`grpc.health.v1.Health/Check`).

### Options

| Option | Default | Description |
| --- | --- | --- |
| `url` | `""` | gRPC URL (parsed for host/port) |
| `host` | `""` | Host (if `url` not provided) |
| `port` | `"50051"` | Port |
| `service_name` | `""` | Service name (empty = overall server) |
| `tls` | `False` | Enable TLS |
| `tls_skip_verify` | `False` | Skip TLS verification |
| `metadata` | `None` | Custom gRPC metadata |
| `bearer_token` | `None` | Bearer token |
| `basic_auth` | `None` | Basic auth `(user, password)` tuple |

### Example

```python
from dephealth.api import grpc_check

grpc_check("user-service",
    host="user.svc",
    port="9090",
    critical=True,
)

# With TLS and auth
grpc_check("secure-grpc",
    host="grpc.svc",
    port="443",
    tls=True,
    bearer_token="eyJhbG...",
    critical=True,
)
```

### Error Classification

| Error | Status Category | Status Detail |
| --- | --- | --- |
| Connection refused | `connection_error` | `connection_refused` |
| DNS resolution failed | `dns_error` | `dns_error` |
| Deadline exceeded | `timeout` | `timeout` |
| TLS error | `tls_error` | `tls_error` |
| Unauthenticated | `auth_error` | `auth_error` |
| NOT_SERVING | `unhealthy` | `grpc_not_serving` |

## TCP

Opens a TCP connection to verify reachability. No data is sent.

### Options

| Option | Default | Description |
| --- | --- | --- |
| `host` | — | Host (required) |
| `port` | — | Port (required) |

### Example

```python
from dephealth.api import tcp_check

tcp_check("memcached",
    host="memcached.svc",
    port="11211",
    critical=False,
)
```

### Error Classification

| Error | Status Category | Status Detail |
| --- | --- | --- |
| Connection refused | `connection_error` | `connection_refused` |
| DNS resolution failed | `dns_error` | `dns_error` |
| Connection timed out | `timeout` | `timeout` |

## PostgreSQL

Executes a SQL query via asyncpg. Supports standalone mode (new connection)
and pool mode (reuses existing asyncpg pool).

### Options

| Option | Default | Description |
| --- | --- | --- |
| `url` | `""` | PostgreSQL URL (`postgresql://user:pass@host:5432/db`) |
| `host` | `""` | Host (if `url` not provided) |
| `port` | `"5432"` | Port |
| `query` | `"SELECT 1"` | SQL query for health check |
| `pool` | `None` | asyncpg pool (preferred for production) |

### Example

```python
from dephealth.api import postgres_check

# Standalone mode
postgres_check("postgres-main",
    url="postgresql://user:pass@pg.svc:5432/mydb",
    critical=True,
)

# Pool mode (preferred)
import asyncpg
pg_pool = await asyncpg.create_pool("postgresql://user:pass@pg.svc:5432/mydb")

postgres_check("postgres-main",
    pool=pg_pool,
    critical=True,
)
```

### Error Classification

| Error | Status Category | Status Detail |
| --- | --- | --- |
| Connection refused | `connection_error` | `connection_refused` |
| DNS resolution failed | `dns_error` | `dns_error` |
| Query timed out | `timeout` | `timeout` |
| Auth failed | `auth_error` | `auth_error` |
| Query error | `error` | error details |

## MySQL

Executes a SQL query via aiomysql. Supports standalone mode (new connection)
and pool mode (reuses existing aiomysql pool).

### Options

| Option | Default | Description |
| --- | --- | --- |
| `url` | `""` | MySQL URL (`mysql://user:pass@host:3306/db`) |
| `host` | `""` | Host (if `url` not provided) |
| `port` | `"3306"` | Port |
| `query` | `"SELECT 1"` | SQL query for health check |
| `pool` | `None` | aiomysql pool (preferred for production) |

### Example

```python
from dephealth.api import mysql_check

# Standalone mode
mysql_check("mysql-main",
    url="mysql://user:pass@mysql.svc:3306/mydb",
    critical=True,
)

# Pool mode (preferred)
import aiomysql
mysql_pool = await aiomysql.create_pool(
    host="mysql.svc", port=3306, user="user", password="pass", db="mydb",
)

mysql_check("mysql-main",
    pool=mysql_pool,
    critical=True,
)
```

### Error Classification

| Error | Status Category | Status Detail |
| --- | --- | --- |
| Connection refused | `connection_error` | `connection_refused` |
| DNS resolution failed | `dns_error` | `dns_error` |
| Query timed out | `timeout` | `timeout` |
| Auth failed | `auth_error` | `auth_error` |
| Query error | `error` | error details |

## Redis

Sends a PING command. Supports standalone mode (new connection)
and pool mode (reuses existing redis-py async client).

### Options

| Option | Default | Description |
| --- | --- | --- |
| `url` | `""` | Redis URL (`redis://host:6379`) |
| `host` | `""` | Host (if `url` not provided) |
| `port` | `"6379"` | Port |
| `password` | `None` | Password (standalone mode) |
| `db` | `None` | Database number (standalone mode) |
| `client` | `None` | redis-py async client (preferred for production) |

### Example

```python
from dephealth.api import redis_check

# Standalone mode
redis_check("redis-cache",
    url="redis://redis.svc:6379",
    critical=False,
)

# Pool mode (preferred)
from redis.asyncio import Redis
redis_client = Redis.from_url("redis://redis.svc:6379")

redis_check("redis-cache",
    client=redis_client,
    critical=False,
)
```

### Error Classification

| Error | Status Category | Status Detail |
| --- | --- | --- |
| Connection refused | `connection_error` | `connection_refused` |
| DNS resolution failed | `dns_error` | `dns_error` |
| PING timed out | `timeout` | `timeout` |
| Auth failed | `auth_error` | `auth_error` |

## AMQP

Establishes a connection to a RabbitMQ broker via aio-pika.

### Options

| Option | Default | Description |
| --- | --- | --- |
| `url` | `""` | AMQP URL (`amqp://user:pass@host:5672/vhost`) |
| `host` | `""` | Host (if `url` not provided) |
| `port` | `"5672"` | Port |

### Example

```python
from dephealth.api import amqp_check

amqp_check("rabbitmq",
    url="amqp://user:pass@rabbitmq.svc:5672/",
    critical=True,
)
```

### Error Classification

| Error | Status Category | Status Detail |
| --- | --- | --- |
| Connection refused | `connection_error` | `connection_refused` |
| DNS resolution failed | `dns_error` | `dns_error` |
| Connection timed out | `timeout` | `timeout` |
| Auth failed | `auth_error` | `auth_error` |

## Kafka

Fetches cluster metadata via aiokafka to verify broker availability.

### Options

| Option | Default | Description |
| --- | --- | --- |
| `url` | `""` | Kafka URL (`kafka://host1:9092,host2:9092`) |
| `host` | `""` | Host (if `url` not provided) |
| `port` | `"9092"` | Port |

Multi-host URLs create separate endpoints per broker.

### Example

```python
from dephealth.api import kafka_check

kafka_check("kafka-main",
    url="kafka://kafka-1.svc:9092,kafka-2.svc:9092",
    critical=True,
)
```

### Error Classification

| Error | Status Category | Status Detail |
| --- | --- | --- |
| Connection refused | `connection_error` | `connection_refused` |
| DNS resolution failed | `dns_error` | `dns_error` |
| Metadata fetch timed out | `timeout` | `timeout` |
| No brokers available | `unhealthy` | `no_brokers` |

## LDAP

Verifies LDAP directory server availability using one of four check methods.
Supports LDAP, LDAPS, and StartTLS.

### Check Methods

| Method | Description |
| --- | --- |
| `ANONYMOUS_BIND` | Anonymous LDAP bind |
| `SIMPLE_BIND` | Authenticated bind (requires `bind_dn` and `bind_password`) |
| `ROOT_DSE` | Read Root DSE (default, no credentials required) |
| `SEARCH` | LDAP search (requires `base_dn`) |

### Options

| Option | Default | Description |
| --- | --- | --- |
| `url` | `""` | LDAP URL (`ldap://host:389` or `ldaps://host:636`) |
| `host` | `""` | Host (if `url` not provided) |
| `port` | `"389"` | Port |
| `check_method` | `ROOT_DSE` | Check method |
| `bind_dn` | `""` | Bind DN |
| `bind_password` | `""` | Bind password |
| `base_dn` | `""` | Base DN for search |
| `search_filter` | `"(objectClass=*)"` | Search filter |
| `search_scope` | `BASE` | Search scope: `BASE`, `ONE`, `SUB` |
| `start_tls` | `False` | Enable StartTLS |
| `tls_skip_verify` | `False` | Skip TLS verification |
| `client` | `None` | ldap3 Connection for pool integration |

### Example

```python
from dephealth.api import ldap_check

# RootDSE check (default, no credentials)
ldap_check("ldap-server",
    url="ldap://ldap.svc:389",
    critical=False,
)

# Simple bind with credentials
ldap_check("ldap-auth",
    url="ldaps://ldap.svc:636",
    check_method="SIMPLE_BIND",
    bind_dn="cn=monitor,dc=corp,dc=com",
    bind_password="secret",
    critical=True,
)

# Search with StartTLS
ldap_check("ldap-search",
    host="ldap.svc",
    port="389",
    check_method="SEARCH",
    base_dn="dc=example,dc=com",
    search_filter="(objectClass=organizationalUnit)",
    search_scope="ONE",
    start_tls=True,
    critical=False,
)
```

### TLS Modes

| Mode | URL Scheme | `start_tls` | Description |
| --- | --- | --- | --- |
| Plain LDAP | `ldap://` | `False` | No encryption |
| LDAPS | `ldaps://` | `False` | TLS from connection start |
| StartTLS | `ldap://` | `True` | Upgrade to TLS after connection |

> `start_tls=True` and `ldaps://` are mutually exclusive.

### Error Classification

| Error | Status Category | Status Detail |
| --- | --- | --- |
| Connection refused | `connection_error` | `connection_refused` |
| DNS resolution failed | `dns_error` | `dns_error` |
| Connection timed out | `timeout` | `timeout` |
| TLS error | `tls_error` | `tls_error` |
| Bind failed | `auth_error` | `auth_error` |
| Search returned no results | `unhealthy` | `ldap_no_results` |

## Error Classification Summary

All checkers classify errors into these status categories:

| Status Category | Description |
| --- | --- |
| `ok` | Check succeeded |
| `timeout` | Check timed out |
| `connection_error` | Connection refused or reset |
| `dns_error` | DNS resolution failed |
| `auth_error` | Authentication or authorization failed |
| `tls_error` | TLS/SSL error |
| `unhealthy` | Endpoint reachable but reported unhealthy |
| `error` | Other unexpected error |

## See Also

- [Getting Started](getting-started.md) — basic setup and first example
- [Configuration](configuration.md) — all options and defaults
- [Authentication](authentication.md) — auth options in detail
- [Connection Pools](connection-pools.md) — pool integration guide
- [Metrics](metrics.md) — Prometheus metrics reference
- [API Reference](api-reference.md) — complete reference of all classes
