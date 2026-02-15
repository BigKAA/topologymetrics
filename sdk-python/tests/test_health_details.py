"""Tests for HealthDetails() API."""

import asyncio
import json
from datetime import UTC, datetime, timedelta
from unittest.mock import AsyncMock, patch

from prometheus_client import CollectorRegistry

from dephealth.api import DependencyHealth, tcp_check
from dephealth.check_result import STATUS_ERROR, STATUS_OK, STATUS_UNKNOWN
from dephealth.dependency import CheckConfig, Dependency, DependencyType, Endpoint
from dephealth.endpoint_status import EndpointStatus
from dephealth.metrics import MetricsExporter
from dephealth.scheduler import CheckScheduler


def _make_dep(
    name: str = "test",
    interval: float = 0.1,
    initial_delay: float = 0.0,
    host: str = "localhost",
    port: str = "8080",
    critical: bool = True,
    labels: dict[str, str] | None = None,
) -> Dependency:
    eps = [Endpoint(host=host, port=port, labels=labels or {})]
    return Dependency(
        name=name,
        type=DependencyType.TCP,
        critical=critical,
        endpoints=eps,
        config=CheckConfig(interval=interval, timeout=5.0, initial_delay=initial_delay),
    )


def _make_scheduler() -> tuple[CheckScheduler, CollectorRegistry]:
    registry = CollectorRegistry()
    metrics = MetricsExporter(instance_name="test-app", registry=registry)
    scheduler = CheckScheduler(metrics=metrics)
    return scheduler, registry


class TestHealthDetailsScheduler:
    def test_empty_before_add(self) -> None:
        """Before adding dependencies, health_details returns empty dict."""
        scheduler, _ = _make_scheduler()
        assert scheduler.health_details() == {}

    def test_unknown_state_before_start(self) -> None:
        """After add but before first check, endpoints are UNKNOWN."""
        scheduler, _ = _make_scheduler()
        dep = _make_dep()

        checker = AsyncMock()
        scheduler.add(dep, checker)

        details = scheduler.health_details()
        assert len(details) == 1
        key = "test:localhost:8080"
        es = details[key]

        assert es.healthy is None
        assert es.status == STATUS_UNKNOWN
        assert es.detail == "unknown"
        assert es.latency == 0.0
        assert es.last_checked_at is None

        # Static fields populated.
        assert es.type == "tcp"
        assert es.name == "test"
        assert es.host == "localhost"
        assert es.port == "8080"
        assert es.critical is True
        assert es.labels == {}

    async def test_healthy_endpoint(self) -> None:
        """After successful check, endpoint is healthy with status=ok."""
        scheduler, _ = _make_scheduler()
        dep = _make_dep()

        checker = AsyncMock()
        checker.check = AsyncMock(return_value=None)
        scheduler.add(dep, checker)
        await scheduler.start()
        await asyncio.sleep(0.3)
        await scheduler.stop()

        details = scheduler.health_details()
        key = "test:localhost:8080"
        es = details[key]

        assert es.healthy is True
        assert es.status == STATUS_OK
        assert es.detail == "ok"
        assert es.latency > 0
        assert es.last_checked_at is not None

    async def test_unhealthy_endpoint(self) -> None:
        """After failed check, endpoint is unhealthy with error status."""
        scheduler, _ = _make_scheduler()
        dep = _make_dep()

        checker = AsyncMock()
        checker.check = AsyncMock(side_effect=OSError("connection refused"))
        scheduler.add(dep, checker)
        await scheduler.start()
        await asyncio.sleep(0.3)
        await scheduler.stop()

        details = scheduler.health_details()
        key = "test:localhost:8080"
        es = details[key]

        assert es.healthy is False
        assert es.status == STATUS_ERROR
        assert es.detail == "error"
        assert es.latency > 0

    async def test_keys_correspond_to_health(self) -> None:
        """All dependencies from health() have corresponding entries in health_details()."""
        scheduler, _ = _make_scheduler()

        checker = AsyncMock()
        checker.check = AsyncMock(return_value=None)

        dep1 = _make_dep(name="dep-a", host="host-1", port="1111")
        dep2 = _make_dep(name="dep-b", host="host-2", port="2222")

        scheduler.add(dep1, checker)
        scheduler.add(dep2, checker)
        await scheduler.start()
        await asyncio.sleep(0.3)
        await scheduler.stop()

        health = scheduler.health()
        details = scheduler.health_details()

        # Every dependency in health() must have at least one entry in health_details().
        for dep_name in health:
            matching = [k for k in details if k.startswith(f"{dep_name}:")]
            assert len(matching) > 0, f"dependency {dep_name!r} not in health_details()"

        assert len(details) >= len(health)

    async def test_after_stop(self) -> None:
        """After stop, health_details returns last known state."""
        scheduler, _ = _make_scheduler()
        dep = _make_dep()

        checker = AsyncMock()
        checker.check = AsyncMock(return_value=None)
        scheduler.add(dep, checker)
        await scheduler.start()
        await asyncio.sleep(0.3)
        await scheduler.stop()

        details = scheduler.health_details()
        key = "test:localhost:8080"
        assert key in details
        assert details[key].healthy is True

    def test_labels_empty(self) -> None:
        """Labels should be empty dict, not None, when no labels set."""
        scheduler, _ = _make_scheduler()
        dep = _make_dep()

        checker = AsyncMock()
        scheduler.add(dep, checker)

        details = scheduler.health_details()
        key = "test:localhost:8080"
        es = details[key]
        assert es.labels == {}
        assert es.labels is not None

    def test_labels_present(self) -> None:
        """Labels should reflect endpoint labels."""
        scheduler, _ = _make_scheduler()
        dep = _make_dep(labels={"role": "primary"})

        checker = AsyncMock()
        scheduler.add(dep, checker)

        details = scheduler.health_details()
        key = "test:localhost:8080"
        assert details[key].labels == {"role": "primary"}

    async def test_result_map_independent(self) -> None:
        """Modifying the returned map should not affect internal state."""
        scheduler, _ = _make_scheduler()
        dep = _make_dep()

        checker = AsyncMock()
        checker.check = AsyncMock(return_value=None)
        scheduler.add(dep, checker)
        await scheduler.start()
        await asyncio.sleep(0.3)
        await scheduler.stop()

        details1 = scheduler.health_details()
        key = "test:localhost:8080"

        # Modify the returned map.
        del details1[key]

        # Get fresh details — should be unaffected.
        details2 = scheduler.health_details()
        assert key in details2
        assert details2[key].name == "test"

    async def test_multiple_endpoints(self) -> None:
        """Multiple endpoints for a single dependency produce separate entries."""
        scheduler, _ = _make_scheduler()
        dep = Dependency(
            name="multi",
            type=DependencyType.TCP,
            critical=False,
            endpoints=[
                Endpoint(host="host-a", port="1111"),
                Endpoint(host="host-b", port="2222"),
            ],
            config=CheckConfig(interval=0.1, timeout=5.0, initial_delay=0.0),
        )

        checker = AsyncMock()
        checker.check = AsyncMock(return_value=None)
        scheduler.add(dep, checker)
        await scheduler.start()
        await asyncio.sleep(0.3)
        await scheduler.stop()

        details = scheduler.health_details()
        assert "multi:host-a:1111" in details
        assert "multi:host-b:2222" in details
        assert len(details) == 2


class TestHealthDetailsFacade:
    async def test_facade_delegates(self) -> None:
        """DependencyHealth.health_details() delegates to scheduler."""
        with patch("dephealth.checks.tcp.TCPChecker.check", new_callable=AsyncMock):
            dh = DependencyHealth(
                "test-app",
                tcp_check("svc", host="localhost", port="8080", critical=True),
                check_interval=timedelta(seconds=1),
                registry=CollectorRegistry(),
            )
            await dh.start()
            await asyncio.sleep(0.2)
            details = dh.health_details()
            await dh.stop()

            assert isinstance(details, dict)
            assert "svc:localhost:8080" in details

    def test_facade_empty_before_add(self) -> None:
        """DependencyHealth with no deps returns empty health_details."""
        dh = DependencyHealth("leaf-app", registry=CollectorRegistry())
        assert dh.health_details() == {}


class TestEndpointStatus:
    def test_latency_millis(self) -> None:
        """latency_millis() converts seconds to milliseconds."""
        es = EndpointStatus(
            healthy=True,
            status="ok",
            detail="ok",
            latency=0.0025,
            type="tcp",
            name="test",
            host="localhost",
            port="8080",
            critical=True,
            last_checked_at=None,
        )
        assert es.latency_millis() == 2.5

    def test_frozen(self) -> None:
        """EndpointStatus is frozen (immutable)."""
        es = EndpointStatus(
            healthy=True,
            status="ok",
            detail="ok",
            latency=0.001,
            type="tcp",
            name="test",
            host="localhost",
            port="8080",
            critical=True,
            last_checked_at=None,
        )
        import dataclasses

        assert dataclasses.fields(es)  # Confirm it's a dataclass.
        try:
            es.healthy = False  # type: ignore[misc]
            raise AssertionError("Should not be mutable")  # noqa: TRY301
        except dataclasses.FrozenInstanceError:
            pass

    def test_to_dict_healthy(self) -> None:
        """to_dict() produces correct JSON-compatible dict for a healthy endpoint."""
        now = datetime(2026, 2, 14, 10, 30, 0, tzinfo=UTC)
        es = EndpointStatus(
            healthy=True,
            status="ok",
            detail="ok",
            latency=0.0023,
            type="postgres",
            name="postgres-main",
            host="pg.svc",
            port="5432",
            critical=True,
            last_checked_at=now,
            labels={"role": "primary"},
        )

        d = es.to_dict()

        assert d["healthy"] is True
        assert d["status"] == "ok"
        assert d["detail"] == "ok"
        assert d["latency_ms"] == 2.3
        assert d["type"] == "postgres"
        assert d["name"] == "postgres-main"
        assert d["host"] == "pg.svc"
        assert d["port"] == "5432"
        assert d["critical"] is True
        assert d["last_checked_at"] is not None
        assert d["labels"] == {"role": "primary"}

        # Verify JSON serializable.
        json_str = json.dumps(d)
        assert "latency_ms" in json_str

    def test_to_dict_unknown(self) -> None:
        """to_dict() produces correct dict for UNKNOWN state."""
        es = EndpointStatus(
            healthy=None,
            status="unknown",
            detail="unknown",
            latency=0.0,
            type="redis",
            name="redis-cache",
            host="redis.svc",
            port="6379",
            critical=False,
            last_checked_at=None,
            labels={},
        )

        d = es.to_dict()

        assert d["healthy"] is None
        assert d["status"] == "unknown"
        assert d["latency_ms"] == 0.0
        assert d["last_checked_at"] is None
        assert d["labels"] == {}

        # Verify JSON serializable — healthy and last_checked_at should be null.
        parsed = json.loads(json.dumps(d))
        assert parsed["healthy"] is None
        assert parsed["last_checked_at"] is None

    def test_to_dict_roundtrip(self) -> None:
        """to_dict() result can be serialized to JSON and back."""
        now = datetime(2026, 2, 14, 10, 30, 0, tzinfo=UTC)
        es = EndpointStatus(
            healthy=False,
            status="timeout",
            detail="timeout",
            latency=5.0,
            type="http",
            name="api-gw",
            host="api.svc",
            port="8080",
            critical=True,
            last_checked_at=now,
            labels={"env": "prod"},
        )

        d = es.to_dict()
        json_str = json.dumps(d)
        parsed = json.loads(json_str)

        assert parsed["healthy"] is False
        assert parsed["status"] == "timeout"
        assert parsed["name"] == "api-gw"
        assert parsed["latency_ms"] == 5000.0
        assert parsed["labels"]["env"] == "prod"
