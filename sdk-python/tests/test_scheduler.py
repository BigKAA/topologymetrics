"""Tests for scheduler.py — CheckScheduler."""

import asyncio
import time
from unittest.mock import AsyncMock

import pytest
from prometheus_client import CollectorRegistry

from dephealth.dependency import CheckConfig, Dependency, DependencyType, Endpoint
from dephealth.metrics import MetricsExporter
from dephealth.scheduler import CheckScheduler, EndpointNotFoundError


def _make_dep(
    name: str = "test",
    interval: float = 0.1,
    initial_delay: float = 0.0,
    host: str = "localhost",
    port: str = "8080",
) -> Dependency:
    return Dependency(
        name=name,
        type=DependencyType.TCP,
        critical=True,
        endpoints=[Endpoint(host=host, port=port)],
        config=CheckConfig(interval=interval, timeout=5.0, initial_delay=initial_delay),
    )


def _make_checker(*, ok: bool = True) -> AsyncMock:
    checker = AsyncMock()
    if ok:
        checker.check = AsyncMock(return_value=None)
    else:
        checker.check = AsyncMock(side_effect=OSError("connection refused"))
    checker.checker_type = lambda: "tcp"
    return checker


def _make_scheduler(
    global_interval: float = 0.1,
) -> tuple[CheckScheduler, CollectorRegistry]:
    registry = CollectorRegistry()
    metrics = MetricsExporter(
        instance_name="test-app", instance_group="test-group", registry=registry
    )
    global_config = CheckConfig(interval=global_interval, timeout=5.0, initial_delay=0)
    scheduler = CheckScheduler(metrics=metrics, global_config=global_config)
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
        """Threshold logic: failure_threshold=2 requires 2 consecutive failures."""
        scheduler, _ = _make_scheduler()
        dep = _make_dep()
        dep = Dependency(
            name=dep.name,
            type=dep.type,
            critical=dep.critical,
            endpoints=dep.endpoints,
            config=CheckConfig(
                interval=dep.config.interval,
                timeout=dep.config.timeout,
                initial_delay=dep.config.initial_delay,
                failure_threshold=2,
                success_threshold=dep.config.success_threshold,
            ),
        )

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
        # First check — ok, second — fail, third — fail -> unhealthy
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

        time.sleep(0.3)

        scheduler.stop_sync()
        assert scheduler.health() == {"test": True}


class TestDynamicAddEndpoint:
    """Tests for CheckScheduler.add_endpoint (async mode)."""

    async def test_add_endpoint(self) -> None:
        """Add endpoint after start, wait, verify health() includes it."""
        scheduler, _ = _make_scheduler()
        await scheduler.start()

        checker = _make_checker(ok=True)
        ep = Endpoint(host="new-host", port="9090")
        await scheduler.add_endpoint("dynamic", DependencyType.TCP, True, ep, checker)

        await asyncio.sleep(0.3)
        await scheduler.stop()

        health = scheduler.health()
        assert "dynamic" in health
        assert health["dynamic"] is True

    async def test_add_endpoint_idempotent(self) -> None:
        """Add same endpoint twice, no error, single entry in health_details."""
        scheduler, _ = _make_scheduler()
        await scheduler.start()

        checker = _make_checker(ok=True)
        ep = Endpoint(host="new-host", port="9090")
        await scheduler.add_endpoint("dynamic", DependencyType.TCP, True, ep, checker)
        await scheduler.add_endpoint("dynamic", DependencyType.TCP, True, ep, checker)

        await asyncio.sleep(0.3)
        await scheduler.stop()

        details = scheduler.health_details()
        matching = [k for k in details if k.startswith("dynamic:")]
        assert len(matching) == 1

    async def test_add_endpoint_before_start(self) -> None:
        """add_endpoint before start raises RuntimeError."""
        scheduler, _ = _make_scheduler()
        checker = _make_checker()
        ep = Endpoint(host="h", port="1")

        with pytest.raises(RuntimeError, match="not started"):
            await scheduler.add_endpoint("dep", DependencyType.TCP, True, ep, checker)

    async def test_add_endpoint_after_stop(self) -> None:
        """add_endpoint after stop raises RuntimeError."""
        scheduler, _ = _make_scheduler()
        await scheduler.start()
        await scheduler.stop()

        checker = _make_checker()
        ep = Endpoint(host="h", port="1")

        with pytest.raises(RuntimeError, match="stopped"):
            await scheduler.add_endpoint("dep", DependencyType.TCP, True, ep, checker)

    async def test_add_endpoint_metrics(self) -> None:
        """Verify health gauge appears with correct labels after dynamic add."""
        scheduler, registry = _make_scheduler()
        await scheduler.start()

        checker = _make_checker(ok=True)
        ep = Endpoint(host="metrics-host", port="7070")
        await scheduler.add_endpoint("metricsvc", DependencyType.HTTP, True, ep, checker)

        await asyncio.sleep(0.3)
        await scheduler.stop()

        samples = list(registry.collect())
        health_samples = [
            s
            for m in samples
            for s in m.samples
            if s.name == "app_dependency_health" and s.labels.get("dependency") == "metricsvc"
        ]
        assert len(health_samples) == 1
        assert health_samples[0].value == 1.0
        assert health_samples[0].labels["host"] == "metrics-host"
        assert health_samples[0].labels["port"] == "7070"


class TestDynamicRemoveEndpoint:
    """Tests for CheckScheduler.remove_endpoint (async mode)."""

    async def test_remove_endpoint(self) -> None:
        """Remove after start, verify disappears from health()."""
        scheduler, _ = _make_scheduler()
        dep = _make_dep(name="removable", host="rem-host", port="5000")
        checker = _make_checker(ok=True)
        scheduler.add(dep, checker)

        await scheduler.start()
        await asyncio.sleep(0.2)
        assert "removable" in scheduler.health()

        await scheduler.remove_endpoint("removable", "rem-host", "5000")
        assert "removable" not in scheduler.health()
        await scheduler.stop()

    async def test_remove_endpoint_idempotent(self) -> None:
        """Remove non-existent endpoint, no error."""
        scheduler, _ = _make_scheduler()
        await scheduler.start()

        # Should not raise.
        await scheduler.remove_endpoint("nonexistent", "h", "1")
        await scheduler.stop()

    async def test_remove_endpoint_metrics_deleted(self) -> None:
        """Verify all metric series removed after remove_endpoint."""
        scheduler, registry = _make_scheduler()
        await scheduler.start()

        checker = _make_checker(ok=True)
        ep = Endpoint(host="del-host", port="4040")
        await scheduler.add_endpoint("delsvc", DependencyType.TCP, True, ep, checker)
        await asyncio.sleep(0.3)

        # Metrics should exist.
        samples_before = [
            s
            for m in registry.collect()
            for s in m.samples
            if s.labels.get("dependency") == "delsvc" and s.name == "app_dependency_health"
        ]
        assert len(samples_before) == 1

        await scheduler.remove_endpoint("delsvc", "del-host", "4040")

        # Health gauge should be gone.
        samples_after = [
            s
            for m in registry.collect()
            for s in m.samples
            if s.labels.get("dependency") == "delsvc" and s.name == "app_dependency_health"
        ]
        assert len(samples_after) == 0
        await scheduler.stop()

    async def test_remove_endpoint_before_start(self) -> None:
        """remove_endpoint before start raises RuntimeError."""
        scheduler, _ = _make_scheduler()
        with pytest.raises(RuntimeError, match="not started"):
            await scheduler.remove_endpoint("dep", "h", "1")


class TestDynamicUpdateEndpoint:
    """Tests for CheckScheduler.update_endpoint (async mode)."""

    async def test_update_endpoint(self) -> None:
        """Update endpoint: old gone, new appears in health()."""
        scheduler, _ = _make_scheduler()
        await scheduler.start()

        checker = _make_checker(ok=True)
        old_ep = Endpoint(host="old-host", port="1111")
        await scheduler.add_endpoint("updsvc", DependencyType.TCP, True, old_ep, checker)
        await asyncio.sleep(0.2)

        new_checker = _make_checker(ok=True)
        new_ep = Endpoint(host="new-host", port="2222")
        await scheduler.update_endpoint("updsvc", "old-host", "1111", new_ep, new_checker)
        await asyncio.sleep(0.2)
        await scheduler.stop()

        details = scheduler.health_details()
        assert "updsvc:old-host:1111" not in details
        assert "updsvc:new-host:2222" in details
        assert details["updsvc:new-host:2222"].healthy is True

    async def test_update_endpoint_not_found(self) -> None:
        """Update non-existent endpoint raises EndpointNotFoundError."""
        scheduler, _ = _make_scheduler()
        await scheduler.start()

        new_checker = _make_checker()
        new_ep = Endpoint(host="new-host", port="2222")

        with pytest.raises(EndpointNotFoundError, match="ghost:x:0"):
            await scheduler.update_endpoint("ghost", "x", "0", new_ep, new_checker)
        await scheduler.stop()

    async def test_update_endpoint_metrics_swap(self) -> None:
        """Old metrics deleted, new metrics present after update."""
        scheduler, registry = _make_scheduler()
        await scheduler.start()

        checker = _make_checker(ok=True)
        old_ep = Endpoint(host="swap-old", port="3333")
        await scheduler.add_endpoint("swapsvc", DependencyType.TCP, True, old_ep, checker)
        await asyncio.sleep(0.3)

        new_checker = _make_checker(ok=True)
        new_ep = Endpoint(host="swap-new", port="4444")
        await scheduler.update_endpoint("swapsvc", "swap-old", "3333", new_ep, new_checker)
        await asyncio.sleep(0.3)
        await scheduler.stop()

        samples = [
            s
            for m in registry.collect()
            for s in m.samples
            if s.labels.get("dependency") == "swapsvc" and s.name == "app_dependency_health"
        ]
        hosts = {s.labels["host"] for s in samples}
        assert "swap-old" not in hosts
        assert "swap-new" in hosts


class TestDynamicStopAfterAdd:
    """Verify clean shutdown after dynamic add."""

    async def test_stop_after_dynamic_add(self) -> None:
        """Add endpoint, then stop(), verify clean shutdown (no hanging tasks)."""
        scheduler, _ = _make_scheduler()
        await scheduler.start()

        checker = _make_checker(ok=True)
        ep = Endpoint(host="stop-host", port="6060")
        await scheduler.add_endpoint("stopsvc", DependencyType.TCP, True, ep, checker)
        await asyncio.sleep(0.2)

        # stop() should complete without timeout.
        await asyncio.wait_for(scheduler.stop(), timeout=5.0)


class TestDynamicConcurrency:
    """Concurrent add/remove/health from multiple tasks."""

    async def test_concurrent_add_remove_health(self) -> None:
        scheduler, _ = _make_scheduler()
        await scheduler.start()

        errors: list[Exception] = []

        async def add_remove(idx: int) -> None:
            try:
                checker = _make_checker(ok=True)
                ep = Endpoint(host=f"host-{idx}", port=str(7000 + idx))
                await scheduler.add_endpoint(f"conc{idx}", DependencyType.TCP, True, ep, checker)
                await asyncio.sleep(0.1)
                scheduler.health()
                await scheduler.remove_endpoint(f"conc{idx}", f"host-{idx}", str(7000 + idx))
            except Exception as e:
                errors.append(e)

        tasks = [asyncio.create_task(add_remove(i)) for i in range(10)]
        await asyncio.gather(*tasks)
        await scheduler.stop()

        assert errors == [], f"Concurrent operations produced errors: {errors}"
        assert scheduler.health() == {}


class TestDynamicSyncMode:
    """Dynamic endpoint management in threading (sync) mode."""

    def test_add_endpoint_sync(self) -> None:
        scheduler, _ = _make_scheduler()
        scheduler.start_sync()

        checker = _make_checker(ok=True)
        ep = Endpoint(host="sync-host", port="8888")
        scheduler.add_endpoint_sync("syncsvc", DependencyType.TCP, True, ep, checker)
        time.sleep(0.3)

        health = scheduler.health()
        assert "syncsvc" in health
        assert health["syncsvc"] is True

        scheduler.stop_sync()

    def test_remove_endpoint_sync(self) -> None:
        scheduler, _ = _make_scheduler()
        dep = _make_dep(name="syncrm", host="sync-rm-host", port="5555")
        checker = _make_checker(ok=True)
        scheduler.add(dep, checker)
        scheduler.start_sync()
        time.sleep(0.2)

        assert "syncrm" in scheduler.health()
        scheduler.remove_endpoint_sync("syncrm", "sync-rm-host", "5555")
        assert "syncrm" not in scheduler.health()

        scheduler.stop_sync()

    def test_update_endpoint_sync(self) -> None:
        scheduler, _ = _make_scheduler()
        scheduler.start_sync()

        checker = _make_checker(ok=True)
        old_ep = Endpoint(host="sync-old", port="1111")
        scheduler.add_endpoint_sync("syncupd", DependencyType.TCP, True, old_ep, checker)
        time.sleep(0.2)

        new_checker = _make_checker(ok=True)
        new_ep = Endpoint(host="sync-new", port="2222")
        scheduler.update_endpoint_sync("syncupd", "sync-old", "1111", new_ep, new_checker)
        time.sleep(0.2)

        details = scheduler.health_details()
        assert "syncupd:sync-old:1111" not in details
        assert "syncupd:sync-new:2222" in details

        scheduler.stop_sync()

    def test_update_endpoint_sync_not_found(self) -> None:
        scheduler, _ = _make_scheduler()
        scheduler.start_sync()

        new_checker = _make_checker()
        new_ep = Endpoint(host="h", port="1")

        with pytest.raises(EndpointNotFoundError):
            scheduler.update_endpoint_sync("ghost", "x", "0", new_ep, new_checker)

        scheduler.stop_sync()
