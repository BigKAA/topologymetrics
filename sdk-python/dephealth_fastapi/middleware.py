"""ASGI middleware for exporting Prometheus metrics at /metrics."""

from __future__ import annotations

from prometheus_client import CONTENT_TYPE_LATEST, REGISTRY, CollectorRegistry, generate_latest
from starlette.middleware.base import BaseHTTPMiddleware, RequestResponseEndpoint
from starlette.requests import Request
from starlette.responses import Response
from starlette.types import ASGIApp


class DepHealthMiddleware(BaseHTTPMiddleware):
    """FastAPI middleware that serves the ``/metrics`` endpoint.

    Intercepts requests to ``/metrics`` and returns Prometheus metrics.
    All other requests are passed through to the application unchanged.

    Example::

        app.add_middleware(DepHealthMiddleware)
        # or with a custom registry:
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
        """Intercept /metrics; pass everything else through."""
        if request.url.path == self._metrics_path:
            body = generate_latest(self._registry)
            return Response(
                content=body,
                media_type=CONTENT_TYPE_LATEST,
            )
        return await call_next(request)
