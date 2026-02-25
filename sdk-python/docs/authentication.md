*[Русская версия](authentication.ru.md)*

# Authentication

This guide covers authentication options for all health checkers
in the dephealth Python SDK.

## HTTP Checker

### Bearer Token

```python
from dephealth.api import http_check

http_check("secure-api",
    url="https://api.svc:443",
    bearer_token="eyJhbGciOiJSUzI1NiIs...",
    critical=True,
)
```

Sends `Authorization: Bearer <token>` header with each check.

### Basic Auth

```python
http_check("basic-api",
    url="http://api.svc:8080",
    basic_auth=("admin", "secret"),
    critical=True,
)
```

Sends `Authorization: Basic <base64>` header.

### Custom Headers

```python
http_check("api-key-service",
    url="http://api.svc:8080",
    headers={"X-Api-Key": "my-secret-key"},
    critical=True,
)
```

Custom headers are sent with every health check request.

### TLS

```python
# Enable TLS (automatic for https:// URLs)
http_check("tls-api",
    url="https://api.svc:443",
    critical=True,
)

# Skip TLS verification (dev/test only)
http_check("self-signed-api",
    url="https://api.svc:443",
    tls_skip_verify=True,
    critical=True,
)
```

## gRPC Checker

### Bearer Token

```python
from dephealth.api import grpc_check

grpc_check("secure-grpc",
    host="grpc.svc",
    port="443",
    tls=True,
    bearer_token="eyJhbGciOiJSUzI1NiIs...",
    critical=True,
)
```

Sends token as gRPC metadata: `authorization: Bearer <token>`.

### Basic Auth

```python
grpc_check("basic-grpc",
    host="grpc.svc",
    port="9090",
    basic_auth=("admin", "secret"),
    critical=True,
)
```

### Custom Metadata

```python
grpc_check("custom-grpc",
    host="grpc.svc",
    port="9090",
    metadata={"x-api-key": "my-secret-key"},
    critical=True,
)
```

### TLS

```python
# Enable TLS
grpc_check("tls-grpc",
    host="grpc.svc",
    port="443",
    tls=True,
    critical=True,
)

# Skip TLS verification
grpc_check("self-signed-grpc",
    host="grpc.svc",
    port="443",
    tls=True,
    tls_skip_verify=True,
    critical=True,
)
```

## Database Checkers

### PostgreSQL

Credentials are provided via the URL:

```python
from dephealth.api import postgres_check

postgres_check("postgres-main",
    url="postgresql://user:password@pg.svc:5432/mydb",
    critical=True,
)
```

With pool integration, the pool already has credentials configured:

```python
import asyncpg

pg_pool = await asyncpg.create_pool("postgresql://user:password@pg.svc:5432/mydb")

postgres_check("postgres-main",
    pool=pg_pool,
    critical=True,
)
```

### MySQL

Credentials via URL:

```python
from dephealth.api import mysql_check

mysql_check("mysql-main",
    url="mysql://user:password@mysql.svc:3306/mydb",
    critical=True,
)
```

### Redis

Via URL or explicit password:

```python
from dephealth.api import redis_check

# Via URL
redis_check("redis-cache",
    url="redis://:password@redis.svc:6379/0",
    critical=False,
)

# Explicit password
redis_check("redis-cache",
    host="redis.svc",
    port="6379",
    password="secret",
    db=0,
    critical=False,
)
```

## AMQP (RabbitMQ)

Credentials via URL:

```python
from dephealth.api import amqp_check

amqp_check("rabbitmq",
    url="amqp://user:password@rabbitmq.svc:5672/vhost",
    critical=True,
)
```

## LDAP

### Anonymous Bind

```python
from dephealth.api import ldap_check

ldap_check("ldap-server",
    url="ldap://ldap.svc:389",
    check_method="ANONYMOUS_BIND",
    critical=False,
)
```

### Simple Bind (Authenticated)

```python
ldap_check("ldap-auth",
    url="ldap://ldap.svc:389",
    check_method="SIMPLE_BIND",
    bind_dn="cn=monitor,dc=corp,dc=com",
    bind_password="secret",
    critical=True,
)
```

### LDAPS (TLS)

```python
ldap_check("ldap-secure",
    url="ldaps://ldap.svc:636",
    check_method="SIMPLE_BIND",
    bind_dn="cn=monitor,dc=corp,dc=com",
    bind_password="secret",
    critical=True,
)
```

### StartTLS

```python
ldap_check("ldap-starttls",
    url="ldap://ldap.svc:389",
    check_method="SIMPLE_BIND",
    bind_dn="cn=monitor,dc=corp,dc=com",
    bind_password="secret",
    start_tls=True,
    critical=True,
)
```

## Auth Error Classification

| Checker | Error Condition | Status Category | Status Detail |
| --- | --- | --- | --- |
| HTTP | 401/403 response | `auth_error` | `auth_error` |
| gRPC | UNAUTHENTICATED status | `auth_error` | `auth_error` |
| PostgreSQL | Invalid credentials | `auth_error` | `auth_error` |
| MySQL | Invalid credentials | `auth_error` | `auth_error` |
| Redis | AUTH failed | `auth_error` | `auth_error` |
| AMQP | AUTH_FAILURE | `auth_error` | `auth_error` |
| LDAP | Bind failed | `auth_error` | `auth_error` |

## Security Best Practices

1. **Never hardcode credentials** — use environment variables or secret stores
2. **Use TLS in production** — enable TLS for HTTP and gRPC checkers
3. **Use pool integration** — pool connections already have credentials configured,
   reducing credential exposure
4. **Minimal permissions** — use read-only credentials for health checks
   (e.g., `SELECT 1` for databases)
5. **Rotate tokens** — periodically rotate bearer tokens and passwords

### Example with Environment Variables

```python
import os

http_check("secure-api",
    url=os.environ["API_URL"],
    bearer_token=os.environ.get("API_TOKEN"),
    critical=True,
)

postgres_check("postgres-main",
    url=os.environ["DATABASE_URL"],
    critical=True,
)
```

## See Also

- [Checkers](checkers.md) — all 9 built-in checkers with examples
- [Configuration](configuration.md) — all options and defaults
- [Connection Pools](connection-pools.md) — pool integration guide
- [Troubleshooting](troubleshooting.md) — common issues and solutions
