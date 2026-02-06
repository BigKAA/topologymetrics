"""ASGI-middleware для экспорта Prometheus-метрик на /metrics."""

from __future__ import annotations

from prometheus_client import CONTENT_TYPE_LATEST, REGISTRY, CollectorRegistry, generate_latest
from starlette.middleware.base import BaseHTTPMiddleware, RequestResponseEndpoint
from starlette.requests import Request
from starlette.responses import Response
from starlette.types import ASGIApp


class DepHealthMiddleware(BaseHTTPMiddleware):
    """Middleware для FastAPI: обслуживает ``/metrics`` endpoint.

    Перехватывает запросы к ``/metrics`` и возвращает Prometheus-метрики.
    Все остальные запросы проходят к приложению без изменений.

    Пример::

        app.add_middleware(DepHealthMiddleware)
        # или с кастомным registry:
        app.add_middleware(DepHealthMiddleware, registry=my_registry, metrics_path="/custom")
    """

    def __init__(
        self,
        app: ASGIApp,
        registry: CollectorRegistry | None = None,
        metrics_path: str = "/metrics",
    ) -> None:
        super().__init__(app)
        self._registry = registry if registry is not None else REGISTRY
        self._metrics_path = metrics_path

    async def dispatch(self, request: Request, call_next: RequestResponseEndpoint) -> Response:
        """Перехватывает /metrics, остальное — передаёт дальше."""
        if request.url.path == self._metrics_path:
            body = generate_latest(self._registry)
            return Response(
                content=body,
                media_type=CONTENT_TYPE_LATEST,
            )
        return await call_next(request)
