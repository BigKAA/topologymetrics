*[Русская версия](python.ru.md)*

# Guide to Integrating dephealth into an Existing Python Service

Step-by-step instructions for adding dependency monitoring
to a running microservice.

## Migration to v0.5.0

### Breaking: mandatory `group` parameter

v0.5.0 adds a mandatory `group` parameter (logical grouping: team, subsystem, project).

```python
# v0.4.x
dh = DependencyHealth("my-service",
    postgres_check("postgres-main", ...),
)

# v0.5.0
dh = DependencyHealth("my-service", "my-team",
    postgres_check("postgres-main", ...),
)
```

FastAPI:

```python
# v0.5.0
app = FastAPI(
    lifespan=dephealth_lifespan("my-service", "my-team",
        postgres_check("postgres-main", ...),
    )
)
```

Alternative: set `DEPHEALTH_GROUP` environment variable (API takes precedence).

Validation: same rules as `name` — `[a-z][a-z0-9-]*`, 1-63 chars.

---

## Migration to v0.4.1

### New: health_details() API

v0.4.1 adds the `health_details()` method that returns detailed status for each
endpoint. No existing API changes — this is a purely additive feature.

```python
details = dh.health_details()
# dict[str, EndpointStatus]

for key, ep in details.items():
    print(f"{key}: healthy={ep.healthy} status={ep.status} "
          f"detail={ep.detail} latency={ep.latency_millis():.1f}ms")
```

`EndpointStatus` fields: `dependency`, `type`, `host`, `port`,
`healthy` (`bool | None`, `None` = unknown), `status`, `detail`,
`latency`, `last_checked_at`, `critical`, `labels`.

> **Note:** `health_details()` uses per-endpoint keys (`"dep:host:port"`),
> while `health()` uses aggregated keys (`"dep"`). The `to_dict()` method
> serializes `EndpointStatus` to a JSON-compatible dict.

---

## Migration to v0.4.0

### New Status Metrics (no code changes required)

v0.4.0 adds two new automatically exported Prometheus metrics:

| Metric | Type | Description |
| --- | --- | --- |
| `app_dependency_status` | Gauge (enum) | Status category: 8 series per endpoint, exactly one = 1 |
| `app_dependency_status_detail` | Gauge (info) | Detailed failure reason: e.g. `http_503`, `auth_error` |

**No code changes are needed** — the SDK exports these metrics automatically alongside the existing `app_dependency_health` and `app_dependency_latency_seconds`.

### Storage Impact

Each endpoint now produces 9 additional time series (8 for `app_dependency_status` + 1 for `app_dependency_status_detail`). For a service with 5 endpoints, this adds 45 series.

### New PromQL Queries

```promql
# Status category for a dependency
app_dependency_status{dependency="postgres-main", status!=""} == 1

# Detailed failure reason
app_dependency_status_detail{dependency="postgres-main", detail!=""} == 1

# Alert on authentication errors
app_dependency_status{status="auth_error"} == 1
```

For the full list of status values, see [Specification — Status Metrics](../specification.md).

## Migration from v0.1 to v0.2

### API Changes

| v0.1 | v0.2 | Description |
| --- | --- | --- |
| `DependencyHealth(...)` | `DependencyHealth("my-service", ...)` | Required first argument `name` |
| `dephealth_lifespan(...)` | `dephealth_lifespan("my-service", ...)` | Required first argument `name` |
| `critical=True` (optional) | `critical=True/False` (required) | For each factory |
| none | `labels={"key": "value"}` | Arbitrary labels |

### Required Changes

1. Add `name` as the first argument:

```python
# v0.1
dh = DependencyHealth(
    postgres_check("postgres-main", url="postgresql://..."),
)

# v0.2
dh = DependencyHealth("my-service",
    postgres_check("postgres-main", url="postgresql://...", critical=True),
)
```

1. Specify `critical` for each dependency:

```python
# v0.1 — critical is optional
redis_check("redis-cache", url="redis://redis.svc:6379")

# v0.2 — critical is required
redis_check("redis-cache", url="redis://redis.svc:6379", critical=False)
```

1. Update `dephealth_lifespan` (FastAPI):

```python
# v0.1
app = FastAPI(
    lifespan=dephealth_lifespan(
        http_check("api", url="http://api:8080"),
    )
)

# v0.2
app = FastAPI(
    lifespan=dephealth_lifespan("my-service",
        http_check("api", url="http://api:8080", critical=True),
    )
)
```

### New Labels in Metrics

```text
# v0.1
app_dependency_health{dependency="postgres-main",type="postgres",host="pg.svc",port="5432"} 1

# v0.2
app_dependency_health{name="my-service",dependency="postgres-main",type="postgres",host="pg.svc",port="5432",critical="yes"} 1
```

Update PromQL queries and Grafana dashboards to include `name` and `critical` labels.

## Prerequisites

- Python 3.11+
- FastAPI, Flask, Django or any ASGI/WSGI framework
- Network access to dependencies (databases, caches, other services) from the service

## Step 1. Install Dependencies

```bash
pip install dephealth[fastapi]
```

Or with specific checkers:

```bash
pip install dephealth[postgres,redis,grpc,fastapi]
```

## Step 2. Import Packages

Add imports to your service initialization file:

```python
from dephealth.api import (
    DependencyHealth,
    http_check,
    postgres_check,
    redis_check,
)
```

For FastAPI integration:

```python
from dephealth_fastapi import (
    dephealth_lifespan,
    DepHealthMiddleware,
    dependencies_router,
)
```

## Step 3. Create DependencyHealth Instance

### Option A: FastAPI with lifespan (recommended)

The simplest approach using `dephealth_lifespan()`:

```python
from fastapi import FastAPI
from datetime import timedelta

app = FastAPI(
    lifespan=dephealth_lifespan("my-service",
        postgres_check("postgres-main",
            url=os.environ["DATABASE_URL"],
            critical=True,
        ),
        redis_check("redis-cache",
            url=os.environ["REDIS_URL"],
            critical=False,
        ),
        http_check("payment-api",
            url=os.environ["PAYMENT_SERVICE_URL"],
            critical=True,
        ),
        check_interval=timedelta(seconds=15),
    )
)
```

### Option B: Connection pool integration (recommended)

SDK uses existing service connections. Advantages:

- Reflects the actual service's ability to work with dependencies
- Does not create additional load on DB/cache
- Detects pool-related issues (exhaustion, leaks)

```python
from datetime import timedelta
import asyncpg
from redis.asyncio import Redis

# Existing service connections
pg_pool = await asyncpg.create_pool(os.environ["DATABASE_URL"])
redis_client = Redis.from_url(os.environ["REDIS_URL"])

dh = DependencyHealth("my-service",
    check_interval=timedelta(seconds=15),

    # PostgreSQL via existing asyncpg pool
    postgres_check("postgres-main", pool=pg_pool, critical=True),

    # Redis via existing redis-py client
    redis_check("redis-cache", client=redis_client, critical=False),

    # For HTTP/gRPC — standalone only
    http_check("payment-api",
        url=os.environ["PAYMENT_SERVICE_URL"],
        critical=True,
    ),

    grpc_check("auth-service",
        host=os.environ["AUTH_HOST"],
        port=os.environ["AUTH_PORT"],
        critical=True,
    ),
)
```

### Option C: Standalone mode (simple)

SDK creates temporary connections for checks:

```python
dh = DependencyHealth("my-service",
    postgres_check("postgres-main",
        url=os.environ["DATABASE_URL"],
        critical=True,
    ),
    redis_check("redis-cache",
        url=os.environ["REDIS_URL"],
        critical=False,
    ),
    http_check("payment-api",
        url=os.environ["PAYMENT_SERVICE_URL"],
        critical=True,
    ),
)
```

## Step 4. Start and Stop

### FastAPI with lifespan

When using `dephealth_lifespan()`, start/stop happen automatically.
The `DependencyHealth` instance is accessible via `app.state.dephealth`.

### Manual management (asyncio)

```python
async def main():
    dh = DependencyHealth("my-service", ...)

    await dh.start()

    # ... application runs ...

    await dh.stop()
```

### Manual management (threading, fallback)

```python
dh = DependencyHealth("my-service", ...)

dh.start_sync()

# ... application runs ...

dh.stop_sync()
```

## Step 5. Export Metrics

### FastAPI

```python
app = FastAPI(lifespan=dephealth_lifespan("my-service", ...))

# Prometheus metrics at /metrics
app.add_middleware(DepHealthMiddleware)

# Endpoint /health/dependencies
app.include_router(dependencies_router)
```

### Without FastAPI

Use the standard `prometheus_client`:

```python
from prometheus_client import generate_latest, CONTENT_TYPE_LATEST

# In HTTP handler:
def metrics_handler(request):
    return Response(
        content=generate_latest(),
        media_type=CONTENT_TYPE_LATEST,
    )
```

## Step 6. Dependency Status Endpoint (optional)

With FastAPI, `dependencies_router` already provides `/health/dependencies`.

For a custom endpoint:

```python
from fastapi import FastAPI, Response
import json

@app.get("/health/dependencies")
async def health_dependencies():
    dh = app.state.dephealth
    health = dh.health()

    all_healthy = all(health.values())
    status_code = 200 if all_healthy else 503

    return Response(
        content=json.dumps({
            "status": "healthy" if all_healthy else "unhealthy",
            "dependencies": health,
        }),
        media_type="application/json",
        status_code=status_code,
    )
```

## Typical Configurations

### Web service with PostgreSQL and Redis

```python
import asyncpg
from redis.asyncio import Redis

pg_pool = await asyncpg.create_pool(os.environ["DATABASE_URL"])
redis_client = Redis.from_url(os.environ["REDIS_URL"])

app = FastAPI(
    lifespan=dephealth_lifespan("my-service",
        postgres_check("postgres", pool=pg_pool, critical=True),
        redis_check("redis", client=redis_client, critical=False),
    )
)
```

### API Gateway with upstream services

```python
app = FastAPI(
    lifespan=dephealth_lifespan("api-gateway",
        http_check("user-service",
            url="http://user-svc:8080",
            health_path="/healthz",
            critical=True,
        ),
        http_check("order-service",
            url="http://order-svc:8080",
            critical=True,
        ),
        grpc_check("auth-service",
            host="auth-svc",
            port="9090",
            critical=True,
        ),
        check_interval=timedelta(seconds=10),
    )
)
```

### Event processor with Kafka and RabbitMQ

```python
app = FastAPI(
    lifespan=dephealth_lifespan("event-processor",
        kafka_check("kafka-main",
            url="kafka://kafka-1:9092,kafka-2:9092",
            critical=True,
        ),
        amqp_check("rabbitmq",
            url="amqp://user:pass@rabbitmq.svc:5672/",
            critical=True,
        ),
        postgres_check("postgres",
            url=os.environ["DATABASE_URL"],
            critical=False,
        ),
    )
)
```

## Troubleshooting

### Metrics don't appear at `/metrics`

**Check:**

1. `DepHealthMiddleware` is added to the application
2. Lifespan was called correctly (application started without errors)
3. Enough time has passed for the first check

### All dependencies show `0` (unhealthy)

**Check:**

1. Network accessibility of dependencies from service container/pod
2. DNS resolution of service names
3. Correctness of URL/host/port in configuration
4. Timeout (`5s` by default) — is it sufficient for the dependency
5. Logs: `logging.basicConfig(level=logging.INFO)` will show error reasons

### High latency for PostgreSQL/MySQL checks

**Cause**: standalone mode creates a new connection each time.

**Solution**: use pool integration (`postgres_check(..., pool=pool)`).
This eliminates connection establishment overhead.

### gRPC: error `context deadline exceeded`

**Check:**

1. gRPC service is accessible at the specified address
2. Service implements `grpc.health.v1.Health/Check`
3. For gRPC use `host` + `port`, not `url` —
   URL parser may incorrectly handle bare `host:port`
4. If TLS is needed: `grpc_check(..., tls=True)`

### AMQP: connection error to RabbitMQ

**Provide the full URL:**

```python
amqp_check("rabbitmq",
    url="amqp://user:pass@rabbitmq.svc:5672/vhost",
    critical=False,
)
```

### Dependency Naming

Names must follow the rules:

- Length: 1-63 characters
- Format: `[a-z][a-z0-9-]*` (lowercase letters, digits, hyphens)
- Must start with a letter
- Examples: `postgres-main`, `redis-cache`, `auth-service`

## Next Steps

- [Quick Start](../quickstart/python.md) — minimal examples
- [Specification Overview](../specification.md) — details of metric contracts and behavior
