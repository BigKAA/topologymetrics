"""Тесты для scheduler.py — CheckScheduler."""

import asyncio
from unittest.mock import AsyncMock

from prometheus_client import CollectorRegistry

from dephealth.dependency import CheckConfig, Dependency, DependencyType, Endpoint
from dephealth.metrics import MetricsExporter
from dephealth.scheduler import CheckScheduler


def _make_dep(interval: float = 0.1, initial_delay: float = 0.0) -> Dependency:
    return Dependency(
        name="test",
        type=DependencyType.TCP,
        endpoints=[Endpoint(host="localhost", port="8080")],
        config=CheckConfig(interval=interval, timeout=5.0, initial_delay=initial_delay),
    )


def _make_scheduler() -> tuple[CheckScheduler, CollectorRegistry]:
    registry = CollectorRegistry()
    metrics = MetricsExporter(registry=registry)
    scheduler = CheckScheduler(metrics=metrics)
    return scheduler, registry


class TestCheckScheduler:
    async def test_healthy_after_check(self) -> None:
        scheduler, _ = _make_scheduler()
        dep = _make_dep()

        checker = AsyncMock()
        checker.check = AsyncMock(return_value=None)
        checker.checker_type = lambda: "tcp"

        scheduler.add(dep, checker)
        await scheduler.start()
        await asyncio.sleep(0.3)
        await scheduler.stop()

        assert scheduler.health() == {"test": True}
        assert checker.check.call_count >= 1

    async def test_unhealthy_on_failure(self) -> None:
        scheduler, _ = _make_scheduler()
        dep = _make_dep()

        checker = AsyncMock()
        checker.check = AsyncMock(side_effect=OSError("connection refused"))
        checker.checker_type = lambda: "tcp"

        scheduler.add(dep, checker)
        await scheduler.start()
        await asyncio.sleep(0.3)
        await scheduler.stop()

        assert scheduler.health() == {"test": False}

    async def test_threshold_logic(self) -> None:
        """Проверка порогов: failure_threshold=2 → нужно 2 неудачи подряд."""
        scheduler, _ = _make_scheduler()
        dep = _make_dep()
        dep.config.failure_threshold = 2

        call_count = 0

        async def check_side_effect(ep: Endpoint) -> None:
            nonlocal call_count
            call_count += 1
            if call_count >= 2:
                msg = "fail"
                raise OSError(msg)

        checker = AsyncMock()
        checker.check = AsyncMock(side_effect=check_side_effect)
        checker.checker_type = lambda: "tcp"

        scheduler.add(dep, checker)
        await scheduler.start()
        # Первая проверка — ок, вторая — fail, третья — fail → unhealthy
        await asyncio.sleep(0.5)
        await scheduler.stop()

        assert scheduler.health() == {"test": False}

    async def test_metrics_updated(self) -> None:
        scheduler, registry = _make_scheduler()
        dep = _make_dep()

        checker = AsyncMock()
        checker.check = AsyncMock(return_value=None)
        checker.checker_type = lambda: "tcp"

        scheduler.add(dep, checker)
        await scheduler.start()
        await asyncio.sleep(0.3)
        await scheduler.stop()

        samples = list(registry.collect())
        health_samples = [
            s for m in samples for s in m.samples if s.name == "app_dependency_health"
        ]
        assert len(health_samples) == 1
        assert health_samples[0].value == 1.0

    def test_sync_mode(self) -> None:
        scheduler, _ = _make_scheduler()
        dep = _make_dep()

        checker = AsyncMock()
        checker.check = AsyncMock(return_value=None)
        checker.checker_type = lambda: "tcp"

        scheduler.add(dep, checker)
        scheduler.start_sync()

        import time

        time.sleep(0.3)

        scheduler.stop_sync()
        assert scheduler.health() == {"test": True}
