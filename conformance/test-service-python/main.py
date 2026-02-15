"""Conformance test service (Python): 7 зависимостей для conformance-сценариев.

Зависимости:
    - PostgreSQL primary + replica
    - Redis
    - RabbitMQ (AMQP)
    - Kafka
    - HTTP-заглушка
    - gRPC-заглушка

Эндпоинты:
    GET /           — JSON со статусом сервиса
    GET /metrics    — Prometheus-метрики
    GET /health     — health check
    GET /health/dependencies — детальный статус зависимостей
"""

from __future__ import annotations

import logging
import os
from datetime import timedelta

from fastapi import FastAPI, Request
from fastapi.responses import JSONResponse, PlainTextResponse

from dephealth.api import (
    amqp_check,
    grpc_check,
    http_check,
    kafka_check,
    postgres_check,
    redis_check,
)
from dephealth_fastapi import DepHealthMiddleware, dependencies_router, dephealth_lifespan

logger = logging.getLogger("dephealth-conformance-python")

# --- Конфигурация ---

PRIMARY_DATABASE_URL = os.environ.get(
    "PRIMARY_DATABASE_URL",
    "postgres://dephealth:dephealth-test-pass@postgres-primary.dephealth-conformance.svc:5432/dephealth?sslmode=disable",
)
REPLICA_DATABASE_URL = os.environ.get(
    "REPLICA_DATABASE_URL",
    "postgres://dephealth:dephealth-test-pass@postgres-replica.dephealth-conformance.svc:5432/dephealth?sslmode=disable",
)
REDIS_URL = os.environ.get(
    "REDIS_URL",
    "redis://redis.dephealth-conformance.svc:6379/0",
)
RABBITMQ_URL = os.environ.get(
    "RABBITMQ_URL",
    "amqp://dephealth:dephealth-test-pass@rabbitmq.dephealth-conformance.svc:5672/",
)
KAFKA_HOST = os.environ.get("KAFKA_HOST", "kafka.dephealth-conformance.svc")
KAFKA_PORT = os.environ.get("KAFKA_PORT", "9092")
HTTP_STUB_URL = os.environ.get(
    "HTTP_STUB_URL",
    "http://http-stub.dephealth-conformance.svc:8080",
)
GRPC_STUB_HOST = os.environ.get("GRPC_STUB_HOST", "grpc-stub.dephealth-conformance.svc")
GRPC_STUB_PORT = os.environ.get("GRPC_STUB_PORT", "9090")
CHECK_INTERVAL = int(os.environ.get("CHECK_INTERVAL", "10"))

# --- Приложение ---

app = FastAPI(
    title="dephealth-conformance-python",
    version="0.1.0",
    lifespan=dephealth_lifespan(
        "conformance-service",
        # PostgreSQL primary
        postgres_check("postgres-primary", url=PRIMARY_DATABASE_URL, critical=True),
        # PostgreSQL replica
        postgres_check("postgres-replica", url=REPLICA_DATABASE_URL, critical=False),
        # Redis
        redis_check("redis-cache", url=REDIS_URL, critical=True),
        # RabbitMQ
        amqp_check("rabbitmq", url=RABBITMQ_URL, critical=False),
        # Kafka
        kafka_check("kafka-main", host=KAFKA_HOST, port=KAFKA_PORT, critical=False),
        # HTTP stub
        http_check("http-service", url=HTTP_STUB_URL, health_path="/health", critical=False),
        # gRPC stub
        grpc_check("grpc-service", host=GRPC_STUB_HOST, port=GRPC_STUB_PORT, critical=False),
        check_interval=timedelta(seconds=CHECK_INTERVAL),
    ),
)

app.add_middleware(DepHealthMiddleware)
app.include_router(dependencies_router)


@app.get("/")
async def index() -> JSONResponse:
    """Информация о сервисе."""
    return JSONResponse(
        content={
            "service": "dephealth-conformance-python",
            "version": "0.1.0",
            "description": "Conformance test service для dephealth Python SDK (7 зависимостей)",
        }
    )


@app.get("/health")
async def health() -> PlainTextResponse:
    """Health check для Kubernetes probes."""
    return PlainTextResponse("OK")


@app.get("/health-details")
async def health_details(request: Request) -> JSONResponse:
    """Detailed health status for each endpoint (HealthDetails API)."""
    dh = getattr(request.app.state, "dephealth", None)
    if dh is None:
        return JSONResponse(content={}, status_code=503)
    details = dh.health_details()
    result = {key: es.to_dict() for key, es in details.items()}
    return JSONResponse(content=result)
