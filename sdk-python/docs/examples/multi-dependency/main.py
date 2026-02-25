# Example: multiple dependencies with pool integration, custom labels,
# a Kubernetes readiness probe, and a JSON health details endpoint.
#
# Install:
#   pip install dephealth[fastapi,postgres,redis,kafka]
#   pip install asyncpg
#
# Run:
#   uvicorn main:app --host 0.0.0.0 --port 8080

from collections.abc import AsyncIterator
from contextlib import asynccontextmanager
from datetime import timedelta

import asyncpg
from fastapi import FastAPI, Request
from fastapi.responses import JSONResponse, PlainTextResponse
from redis.asyncio import Redis

from dephealth.api import (
    DependencyHealth,
    http_check,
    kafka_check,
    postgres_check,
    redis_check,
)
from dephealth_fastapi import DepHealthMiddleware, dependencies_router


@asynccontextmanager
async def lifespan(app: FastAPI) -> AsyncIterator[None]:
    # Create connection pools before DependencyHealth.
    pg_pool = await asyncpg.create_pool(
        "postgresql://app:secret@pg.db:5432/orders",
        min_size=5,
        max_size=20,
    )
    redis_client = Redis.from_url("redis://redis.cache:6379", decode_responses=True)

    dh = DependencyHealth(
        "order-service",
        "backend",
        # PostgreSQL — pool integration (reuses existing connections).
        postgres_check(
            "postgres-main",
            pool=pg_pool,
            host="pg.db",
            port="5432",
            critical=True,
            labels={"env": "production"},
        ),
        # Redis — client integration.
        redis_check(
            "redis-cache",
            client=redis_client,
            host="redis.cache",
            port="6379",
            critical=False,
            labels={"env": "production"},
        ),
        # HTTP with Bearer token authentication.
        http_check(
            "auth-service",
            url="https://auth.internal:8443",
            health_path="/healthz",
            bearer_token="my-service-token",
            critical=True,
            labels={"env": "production"},
        ),
        # Kafka brokers (multiple hosts parsed from URL).
        kafka_check(
            "events-kafka",
            url="kafka://kafka-0.broker:9092,kafka-1.broker:9092,kafka-2.broker:9092",
            critical=True,
            labels={"env": "production"},
        ),
        check_interval=timedelta(seconds=10),
        timeout=timedelta(seconds=3),
    )

    app.state.dephealth = dh
    app.state.pg_pool = pg_pool
    app.state.redis_client = redis_client

    await dh.start()
    try:
        yield
    finally:
        await dh.stop()
        await pg_pool.close()
        await redis_client.aclose()


app = FastAPI(title="Multi-dependency example", lifespan=lifespan)

# Expose Prometheus metrics on GET /metrics.
app.add_middleware(DepHealthMiddleware)

# Expose JSON health status on GET /health/dependencies.
app.include_router(dependencies_router)


@app.get("/readyz")
async def readiness(request: Request) -> PlainTextResponse:
    """Kubernetes readiness probe: 200 if all critical deps are healthy."""
    dh: DependencyHealth = request.app.state.dephealth
    health = dh.health()

    if all(health.values()):
        return PlainTextResponse("ok", status_code=200)
    return PlainTextResponse("not ready", status_code=503)


@app.get("/healthz")
async def health_details(request: Request) -> JSONResponse:
    """Debug endpoint: detailed JSON health status per endpoint."""
    dh: DependencyHealth = request.app.state.dephealth
    details = dh.health_details()

    return JSONResponse(
        content={k: v.to_dict() for k, v in details.items()},
    )
