"""Тесты для api.py — DependencyHealth и фабрики."""

import asyncio
from datetime import timedelta
from unittest.mock import AsyncMock, patch

from prometheus_client import CollectorRegistry

from dephealth.api import (
    DependencyHealth,
    http_check,
    tcp_check,
)


class TestDependencyHealth:
    async def test_start_stop(self) -> None:
        with patch("dephealth.checks.http.HTTPChecker.check", new_callable=AsyncMock):
            dh = DependencyHealth(
                http_check("api", url="http://localhost:8080"),
                check_interval=timedelta(seconds=1),
                registry=CollectorRegistry(),
            )
            await dh.start()
            await asyncio.sleep(0.2)
            await dh.stop()

    async def test_health_returns_dict(self) -> None:
        with patch("dephealth.checks.tcp.TCPChecker.check", new_callable=AsyncMock):
            dh = DependencyHealth(
                tcp_check("svc", host="localhost", port="8080"),
                check_interval=timedelta(seconds=1),
                registry=CollectorRegistry(),
            )
            await dh.start()
            await asyncio.sleep(0.2)
            result = dh.health()
            await dh.stop()

            assert isinstance(result, dict)
            assert "svc" in result

    async def test_multiple_deps(self) -> None:
        with (
            patch("dephealth.checks.http.HTTPChecker.check", new_callable=AsyncMock),
            patch("dephealth.checks.tcp.TCPChecker.check", new_callable=AsyncMock),
        ):
            dh = DependencyHealth(
                http_check("api", url="http://localhost:8080"),
                tcp_check("db", host="localhost", port="5432"),
                check_interval=timedelta(seconds=1),
                registry=CollectorRegistry(),
            )
            await dh.start()
            await asyncio.sleep(0.2)
            result = dh.health()
            await dh.stop()

            assert "api" in result
            assert "db" in result

    def test_sync_mode(self) -> None:
        import time

        with patch("dephealth.checks.tcp.TCPChecker.check", new_callable=AsyncMock):
            dh = DependencyHealth(
                tcp_check("svc", host="localhost", port="8080"),
                check_interval=timedelta(seconds=1),
                registry=CollectorRegistry(),
            )
            dh.start_sync()
            time.sleep(0.2)
            result = dh.health()
            dh.stop_sync()

            assert "svc" in result
