*[Русская версия](getting-started.ru.md)*

# Getting Started

This guide covers installation, basic setup, and your first health check
with the dephealth Python SDK.

## Prerequisites

- Python 3.11 or later
- pip (or uv, poetry)
- A running dependency to monitor (PostgreSQL, Redis, HTTP service, etc.)

## Installation

Basic installation (HTTP and TCP checkers included):

```bash
pip install dephealth
```

With specific checkers:

```bash
pip install dephealth[postgres,redis]
```

With FastAPI integration:

```bash
pip install dephealth[fastapi]
```

All checkers and FastAPI:

```bash
pip install dephealth[all]
```

### Available Extras

| Extra | Dependency | Description |
| --- | --- | --- |
| `postgres` | asyncpg | PostgreSQL checker |
| `mysql` | aiomysql | MySQL checker |
| `redis` | redis[hiredis] | Redis checker |
| `amqp` | aio-pika | RabbitMQ (AMQP) checker |
| `kafka` | aiokafka | Kafka checker |
| `ldap` | ldap3 | LDAP checker |
| `grpc` | grpcio, grpcio-health-checking | gRPC checker |
| `fastapi` | fastapi, uvicorn | FastAPI integration |
| `all` | all of the above | Everything |

## Minimal Example

Monitor a single HTTP dependency:

```python
import asyncio
from dephealth.api import DependencyHealth, http_check

dh = DependencyHealth("my-service", "my-team",
    http_check("payment-api",
        url="http://payment.svc:8080",
        critical=True,
    ),
)

async def main():
    await dh.start()

    # Metrics are available via prometheus_client
    # ... application runs ...

    await dh.stop()

asyncio.run(main())
```

After startup, Prometheus metrics appear:

```text
app_dependency_health{name="my-service",group="my-team",dependency="payment-api",type="http",host="payment.svc",port="8080",critical="yes"} 1
app_dependency_latency_seconds_bucket{name="my-service",group="my-team",dependency="payment-api",type="http",host="payment.svc",port="8080",critical="yes",le="0.01"} 42
```

## Key Concepts

### Name and Group

Every `DependencyHealth` instance requires two identifiers:

- **name** — unique application name (e.g., `"my-service"`)
- **group** — logical group the service belongs to (e.g., `"my-team"`, `"payments"`)

Both appear as labels in all exported metrics. Validation rules:
`[a-z][a-z0-9-]*`, 1-63 characters.

If not passed as arguments, the SDK reads `DEPHEALTH_NAME` and
`DEPHEALTH_GROUP` environment variables as fallback.

### Dependencies

Each dependency is registered via factory functions passed to the
`DependencyHealth` constructor:

| Factory function | DependencyType | Description |
| --- | --- | --- |
| `http_check()` | `HTTP` | HTTP service |
| `grpc_check()` | `GRPC` | gRPC service |
| `tcp_check()` | `TCP` | TCP endpoint |
| `postgres_check()` | `POSTGRES` | PostgreSQL database |
| `mysql_check()` | `MYSQL` | MySQL database |
| `redis_check()` | `REDIS` | Redis server |
| `amqp_check()` | `AMQP` | RabbitMQ (AMQP broker) |
| `kafka_check()` | `KAFKA` | Apache Kafka broker |
| `ldap_check()` | `LDAP` | LDAP directory server |

Each dependency requires:

- A **name** (first argument) — identifies the dependency in metrics
- **Endpoint** — specified via `url=` or `host=` + `port=`
- **Critical** flag — `critical=True` or `critical=False` (mandatory)

### Lifecycle

1. **Create** — `DependencyHealth("name", "group", ...specs)`
2. **Start** — `await dh.start()` launches periodic health checks
3. **Run** — checks execute at the configured interval (default 15s)
4. **Stop** — `await dh.stop()` cancels all check tasks

## Multiple Dependencies

```python
from datetime import timedelta
from dephealth.api import (
    DependencyHealth,
    http_check,
    postgres_check,
    redis_check,
    grpc_check,
)

dh = DependencyHealth("my-service", "my-team",
    # Global settings
    check_interval=timedelta(seconds=30),
    timeout=timedelta(seconds=3),

    # PostgreSQL
    postgres_check("postgres-main",
        url="postgresql://user:pass@pg.svc:5432/mydb",
        critical=True,
    ),

    # Redis
    redis_check("redis-cache",
        url="redis://redis.svc:6379",
        critical=False,
    ),

    # HTTP service
    http_check("auth-service",
        url="http://auth.svc:8080",
        health_path="/healthz",
        critical=True,
    ),

    # gRPC service
    grpc_check("user-service",
        host="user.svc",
        port="9090",
        critical=False,
    ),
)
```

## Checking Health Status

### Simple Status

```python
health = dh.health()
# {"postgres-main": True, "redis-cache": True, "auth-service": True}

# Use for readiness probe
all_healthy = all(health.values())
```

### Detailed Status

```python
details = dh.health_details()
for key, ep in details.items():
    print(f"{key}: healthy={ep.healthy} status={ep.status} "
          f"latency={ep.latency_millis():.1f}ms")
```

`health_details()` returns an `EndpointStatus` object with health state,
status category, latency, timestamps, and custom labels. Before the first
check completes, `healthy` is `None` and `status` is `"unknown"`.

## Next Steps

- [Checkers](checkers.md) — detailed guide for all 9 built-in checkers
- [Configuration](configuration.md) — all options, defaults, and environment variables
- [Connection Pools](connection-pools.md) — integration with existing connection pools
- [FastAPI Integration](fastapi.md) — lifespan, middleware, and health endpoint
- [Authentication](authentication.md) — auth for HTTP, gRPC, and database checkers
- [Metrics](metrics.md) — Prometheus metrics reference and PromQL examples
- [API Reference](api-reference.md) — complete reference of all public classes
- [Troubleshooting](troubleshooting.md) — common issues and solutions
- [Migration Guide](migration.md) — version upgrade instructions
- [Code Style](code-style.md) — Python code conventions for this project
- [Examples](examples/) — complete runnable examples
