"""Tests for FastAPI integration: middleware, lifespan, endpoints."""

from __future__ import annotations

from unittest.mock import AsyncMock, patch

import pytest
from fastapi import FastAPI
from httpx import ASGITransport, AsyncClient
from prometheus_client import CollectorRegistry

from dephealth.api import DependencyHealth
from dephealth_fastapi import (
    DepHealthMiddleware,
    dependencies_router,
    dephealth_lifespan,
)

# --- Fixtures ---


@pytest.fixture()
def registry() -> CollectorRegistry:
    """Isolated registry for tests."""
    return CollectorRegistry()


def _make_app_with_middleware(registry: CollectorRegistry) -> FastAPI:
    """Create a FastAPI application with metrics middleware."""
    app = FastAPI()
    app.add_middleware(DepHealthMiddleware, registry=registry)

    @app.get("/")
    async def root() -> dict[str, str]:
        return {"status": "ok"}

    return app


# --- Middleware tests ---


class TestDepHealthMiddleware:
    """Tests for DepHealthMiddleware."""

    async def test_metrics_endpoint_returns_prometheus(self, registry: CollectorRegistry) -> None:
        """GET /metrics returns Prometheus format."""
        app = _make_app_with_middleware(registry)
        async with AsyncClient(transport=ASGITransport(app=app), base_url="http://test") as client:
            resp = await client.get("/metrics")
        assert resp.status_code == 200
        assert "text/plain" in resp.headers["content-type"]

    async def test_other_routes_pass_through(self, registry: CollectorRegistry) -> None:
        """Requests to other paths pass through."""
        app = _make_app_with_middleware(registry)
        async with AsyncClient(transport=ASGITransport(app=app), base_url="http://test") as client:
            resp = await client.get("/")
        assert resp.status_code == 200
        assert resp.json() == {"status": "ok"}

    async def test_custom_metrics_path(self, registry: CollectorRegistry) -> None:
        """Middleware works with a custom path."""
        app = FastAPI()
        app.add_middleware(DepHealthMiddleware, registry=registry, metrics_path="/custom")
        async with AsyncClient(transport=ASGITransport(app=app), base_url="http://test") as client:
            resp = await client.get("/custom")
        assert resp.status_code == 200
        assert "text/plain" in resp.headers["content-type"]


# --- Endpoint tests ---


class TestDependenciesEndpoint:
    """Tests for /health/dependencies."""

    async def test_no_dephealth_returns_503(self) -> None:
        """If DependencyHealth is not initialized, return 503."""
        app = FastAPI()
        app.include_router(dependencies_router)
        async with AsyncClient(transport=ASGITransport(app=app), base_url="http://test") as client:
            resp = await client.get("/health/dependencies")
        assert resp.status_code == 503
        data = resp.json()
        assert data["status"] == "unknown"

    async def test_all_healthy_returns_200(self) -> None:
        """All dependencies healthy — 200."""
        app = FastAPI()
        app.include_router(dependencies_router)

        mock_dh = AsyncMock(spec=DependencyHealth)
        mock_dh.health.return_value = {"db": True, "cache": True}
        app.state.dephealth = mock_dh  # type: ignore[attr-defined]

        async with AsyncClient(transport=ASGITransport(app=app), base_url="http://test") as client:
            resp = await client.get("/health/dependencies")
        assert resp.status_code == 200
        data = resp.json()
        assert data["status"] == "healthy"
        assert data["dependencies"]["db"] is True

    async def test_degraded_returns_503(self) -> None:
        """Some dependencies unhealthy — 503 + status=degraded."""
        app = FastAPI()
        app.include_router(dependencies_router)

        mock_dh = AsyncMock(spec=DependencyHealth)
        mock_dh.health.return_value = {"db": True, "cache": False}
        app.state.dephealth = mock_dh  # type: ignore[attr-defined]

        async with AsyncClient(transport=ASGITransport(app=app), base_url="http://test") as client:
            resp = await client.get("/health/dependencies")
        assert resp.status_code == 503
        data = resp.json()
        assert data["status"] == "degraded"
        assert data["dependencies"]["cache"] is False


# --- Lifespan tests ---


class TestDephealthLifespan:
    """Tests for dephealth_lifespan."""

    async def test_lifespan_starts_and_stops(self, registry: CollectorRegistry) -> None:
        """Lifespan starts and stops DependencyHealth."""
        with (
            patch.object(DependencyHealth, "start", new_callable=AsyncMock) as mock_start,
            patch.object(DependencyHealth, "stop", new_callable=AsyncMock) as mock_stop,
        ):
            lifespan_fn = dephealth_lifespan("test-app", registry=registry)
            app = FastAPI()

            async with lifespan_fn(app) as state:  # type: ignore[misc]
                assert "dephealth" in state
                mock_start.assert_called_once()

            mock_stop.assert_called_once()

    async def test_lifespan_sets_app_state(self, registry: CollectorRegistry) -> None:
        """Lifespan sets app.state.dephealth."""
        with (
            patch.object(DependencyHealth, "start", new_callable=AsyncMock),
            patch.object(DependencyHealth, "stop", new_callable=AsyncMock),
        ):
            lifespan_fn = dephealth_lifespan("test-app", registry=registry)
            app = FastAPI()

            async with lifespan_fn(app) as _:  # type: ignore[misc]
                assert hasattr(app.state, "dephealth")
                assert isinstance(app.state.dephealth, DependencyHealth)
