"""Тестовый Python-сервис для демонстрации dephealth SDK.

Зависимости: PostgreSQL, Redis, HTTP-заглушка, gRPC-заглушка.

Эндпоинты:
    GET /           — JSON со статусом сервиса
    GET /metrics    — Prometheus-метрики
    GET /health     — health check для Kubernetes probes
    GET /health/dependencies — детальный статус зависимостей
"""

from __future__ import annotations

import logging
import os
from datetime import timedelta

from fastapi import FastAPI
from fastapi.responses import JSONResponse, PlainTextResponse

from dephealth.api import (
    grpc_check,
    http_check,
    postgres_check,
    redis_check,
)
from dephealth_fastapi import DepHealthMiddleware, dependencies_router, dephealth_lifespan

logger = logging.getLogger("dephealth-test-python")

# --- Конфигурация ---

DATABASE_URL = os.environ.get(
    "DATABASE_URL",
    "postgres://dephealth:dephealth-test-pass@postgres:5432/dephealth?sslmode=disable",
)
REDIS_URL = os.environ.get("REDIS_URL", "redis://redis:6379/0")
HTTP_STUB_URL = os.environ.get("HTTP_STUB_URL", "http://http-stub:8080")
GRPC_STUB_HOST = os.environ.get("GRPC_STUB_HOST", "grpc-stub")
GRPC_STUB_PORT = os.environ.get("GRPC_STUB_PORT", "9090")
CHECK_INTERVAL = int(os.environ.get("CHECK_INTERVAL", "10"))

# --- Приложение ---

app = FastAPI(
    title="dephealth-test-python",
    version="0.1.0",
    lifespan=dephealth_lifespan(
        postgres_check("postgres", url=DATABASE_URL),
        redis_check("redis", url=REDIS_URL),
        http_check("http-stub", url=HTTP_STUB_URL, health_path="/health"),
        grpc_check("grpc-stub", host=GRPC_STUB_HOST, port=GRPC_STUB_PORT),
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
            "service": "dephealth-test-python",
            "version": "0.1.0",
            "description": "Тестовый сервис для демонстрации dephealth Python SDK",
        }
    )


@app.get("/health")
async def health() -> PlainTextResponse:
    """Health check для Kubernetes probes."""
    return PlainTextResponse("OK")
