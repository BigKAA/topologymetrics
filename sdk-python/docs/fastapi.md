*[Русская версия](fastapi.ru.md)*

# FastAPI Integration

This guide covers the `dephealth_fastapi` package: lifespan management,
Prometheus middleware, and the health dependencies endpoint.

## Installation

```bash
pip install dephealth[fastapi]
```

This installs `dephealth`, `fastapi`, and `uvicorn`.

## Quick Start

```python
from fastapi import FastAPI
from dephealth.api import http_check, postgres_check
from dephealth_fastapi import dephealth_lifespan, DepHealthMiddleware, dependencies_router

app = FastAPI(
    lifespan=dephealth_lifespan("my-service", "my-team",
        postgres_check("postgres-main",
            url="postgresql://user:pass@pg.svc:5432/mydb",
            critical=True,
        ),
        http_check("payment-api",
            url="http://payment.svc:8080",
            critical=True,
        ),
    )
)

# Prometheus metrics at /metrics
app.add_middleware(DepHealthMiddleware)

# Health dependencies endpoint at /health/dependencies
app.include_router(dependencies_router)
```

## Lifespan

`dephealth_lifespan()` is a factory that returns a FastAPI lifespan
callable. It handles the full lifecycle: creates a `DependencyHealth`
instance, starts monitoring on app startup, and stops on shutdown.

### Signature

```python
def dephealth_lifespan(
    name: str,
    group: str,
    *specs: _DependencySpec,
    check_interval: timedelta | None = None,
    timeout: timedelta | None = None,
    registry: CollectorRegistry | None = None,
    log: logging.Logger | None = None,
) -> Callable[..., AsyncContextManager[None]]
```

Parameters are the same as `DependencyHealth` constructor. See
[Configuration](configuration.md) for details.

### Usage

```python
from fastapi import FastAPI
from datetime import timedelta

app = FastAPI(
    lifespan=dephealth_lifespan("my-service", "my-team",
        postgres_check("postgres", url="postgresql://...", critical=True),
        redis_check("redis", url="redis://...", critical=False),
        check_interval=timedelta(seconds=30),
    )
)
```

### Accessing DependencyHealth

After app startup, the `DependencyHealth` instance is available via
`app.state.dephealth`:

```python
@app.get("/custom-health")
async def custom_health(request: Request):
    dh = request.app.state.dephealth
    health = dh.health()
    return {"status": "healthy" if all(health.values()) else "degraded"}
```

## Middleware

`DepHealthMiddleware` intercepts requests to `/metrics` and returns
Prometheus text format output.

### Configuration

```python
app.add_middleware(
    DepHealthMiddleware,
    metrics_path="/metrics",       # default
    registry=None,                 # default: prometheus_client default registry
)
```

| Parameter | Type | Default | Description |
| --- | --- | --- | --- |
| `metrics_path` | `str` | `"/metrics"` | URL path for Prometheus scraping |
| `registry` | `CollectorRegistry \| None` | default | Custom Prometheus registry |

### How It Works

1. If request path matches `metrics_path`, the middleware generates Prometheus
   text output and returns it with `Content-Type: text/plain; version=0.0.4`.
2. All other requests pass through to the application.

## Health Dependencies Endpoint

`dependencies_router` provides a single GET endpoint at
`/health/dependencies` that returns dependency health status as JSON.

### Configuration

```python
app.include_router(dependencies_router)
```

### Response Format

**Healthy (200):**

```json
{
    "status": "healthy",
    "dependencies": {
        "postgres-main": true,
        "redis-cache": true,
        "payment-api": true
    }
}
```

**Degraded (503):**

```json
{
    "status": "degraded",
    "dependencies": {
        "postgres-main": true,
        "redis-cache": false,
        "payment-api": true
    }
}
```

### Status Codes

| Code | Condition |
| --- | --- |
| 200 | All dependencies healthy |
| 503 | Any dependency unhealthy or status unknown |

## Complete Example

```python
import os
from datetime import timedelta
from fastapi import FastAPI
import asyncpg
from redis.asyncio import Redis

from dephealth.api import postgres_check, redis_check, http_check, grpc_check
from dephealth_fastapi import dephealth_lifespan, DepHealthMiddleware, dependencies_router

# Connection pools (created before app startup)
pg_pool = None
redis_client = None

async def setup_pools():
    global pg_pool, redis_client
    pg_pool = await asyncpg.create_pool(os.environ["DATABASE_URL"])
    redis_client = Redis.from_url(os.environ["REDIS_URL"])

app = FastAPI(
    lifespan=dephealth_lifespan("my-service", "my-team",
        postgres_check("postgres", pool=pg_pool, critical=True),
        redis_check("redis", client=redis_client, critical=False),
        http_check("payment-api",
            url=os.environ["PAYMENT_URL"],
            bearer_token=os.environ.get("PAYMENT_TOKEN"),
            critical=True,
        ),
        grpc_check("auth-service",
            host="auth.svc",
            port="9090",
            critical=True,
        ),
        check_interval=timedelta(seconds=15),
    )
)

app.add_middleware(DepHealthMiddleware)
app.include_router(dependencies_router)

@app.get("/")
async def root():
    return {"service": "my-service"}
```

## Custom Health Endpoint

If `dependencies_router` does not meet your needs, create a custom endpoint:

```python
from fastapi import FastAPI, Request, Response
import json

@app.get("/health/dependencies")
async def health_dependencies(request: Request):
    dh = request.app.state.dephealth
    health = dh.health()
    all_healthy = all(health.values())

    return Response(
        content=json.dumps({
            "status": "healthy" if all_healthy else "unhealthy",
            "dependencies": health,
        }),
        media_type="application/json",
        status_code=200 if all_healthy else 503,
    )
```

### Detailed Health Endpoint

```python
@app.get("/health/dependencies/details")
async def health_details(request: Request):
    dh = request.app.state.dephealth
    details = dh.health_details()
    return {
        key: ep.to_dict() for key, ep in details.items()
    }
```

## See Also

- [Getting Started](getting-started.md) — basic setup and first example
- [Configuration](configuration.md) — all options and defaults
- [Metrics](metrics.md) — Prometheus metrics reference
- [Troubleshooting](troubleshooting.md) — common issues and solutions
- [Examples](examples/) — complete runnable examples
