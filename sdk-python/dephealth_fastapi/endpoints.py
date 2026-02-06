"""Эндпоинт /health/dependencies — JSON со статусом зависимостей."""

from __future__ import annotations

from fastapi import APIRouter, Request
from fastapi.responses import JSONResponse

from dephealth.api import DependencyHealth

dependencies_router = APIRouter()


@dependencies_router.get("/health/dependencies")
async def health_dependencies(request: Request) -> JSONResponse:
    """Возвращает JSON со статусом всех зависимостей.

    Формат ответа::

        {
            "status": "healthy",
            "dependencies": {
                "postgres": true,
                "redis": true,
                "http-stub": false
            }
        }
    """
    dh: DependencyHealth | None = getattr(request.app.state, "dephealth", None)
    if dh is None:
        return JSONResponse(
            content={"status": "unknown", "dependencies": {}},
            status_code=503,
        )

    health = dh.health()
    all_healthy = all(health.values()) if health else False
    status = "healthy" if all_healthy else "degraded"

    return JSONResponse(
        content={"status": status, "dependencies": health},
        status_code=200 if all_healthy else 503,
    )
