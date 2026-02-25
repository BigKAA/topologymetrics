"""Tests for api.py — DependencyHealth and factory functions."""

import asyncio
import os
from datetime import timedelta
from unittest.mock import AsyncMock, patch

import pytest
from prometheus_client import CollectorRegistry

from dephealth.api import (
    DependencyHealth,
    http_check,
    mysql_check,
    postgres_check,
    redis_check,
    tcp_check,
)
from dephealth.dependency import DependencyType, Endpoint
from dephealth.scheduler import EndpointNotFoundError


class TestDependencyHealth:
    async def test_start_stop(self) -> None:
        with patch("dephealth.checks.http.HTTPChecker.check", new_callable=AsyncMock):
            dh = DependencyHealth(
                "test-app",
                "test-group",
                http_check("api", url="http://localhost:8080", critical=True),
                check_interval=timedelta(seconds=1),
                registry=CollectorRegistry(),
            )
            await dh.start()
            await asyncio.sleep(0.2)
            await dh.stop()

    async def test_health_returns_dict(self) -> None:
        with patch("dephealth.checks.tcp.TCPChecker.check", new_callable=AsyncMock):
            dh = DependencyHealth(
                "test-app",
                "test-group",
                tcp_check("svc", host="localhost", port="8080", critical=True),
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
                "test-app",
                "test-group",
                http_check("api", url="http://localhost:8080", critical=True),
                tcp_check("db", host="localhost", port="5432", critical=False),
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
                "test-app",
                "test-group",
                tcp_check("svc", host="localhost", port="8080", critical=True),
                check_interval=timedelta(seconds=1),
                registry=CollectorRegistry(),
            )
            dh.start_sync()
            time.sleep(0.2)
            result = dh.health()
            dh.stop_sync()

            assert "svc" in result


class TestZeroDependencies:
    def test_creation_succeeds(self) -> None:
        dh = DependencyHealth(
            "leaf-app",
            "test-group",
            registry=CollectorRegistry(),
        )
        assert dh is not None

    def test_health_returns_empty(self) -> None:
        dh = DependencyHealth(
            "leaf-app",
            "test-group",
            registry=CollectorRegistry(),
        )
        assert dh.health() == {}

    async def test_start_stop_async(self) -> None:
        dh = DependencyHealth(
            "leaf-app",
            "test-group",
            registry=CollectorRegistry(),
        )
        await dh.start()
        await dh.stop()

    def test_start_stop_sync(self) -> None:
        dh = DependencyHealth(
            "leaf-app",
            "test-group",
            registry=CollectorRegistry(),
        )
        dh.start_sync()
        dh.stop_sync()


class TestInstanceName:
    def test_missing_name_raises(self) -> None:
        with pytest.raises(ValueError, match="instance name is required"):
            DependencyHealth(
                "",
                "test-group",
                tcp_check("svc", host="localhost", port="8080", critical=True),
                registry=CollectorRegistry(),
            )

    def test_invalid_name_raises(self) -> None:
        with pytest.raises(ValueError, match="invalid instance name"):
            DependencyHealth(
                "Bad-Name",
                "test-group",
                tcp_check("svc", host="localhost", port="8080", critical=True),
                registry=CollectorRegistry(),
            )

    def test_name_from_env(self) -> None:
        with (
            patch.dict(os.environ, {"DEPHEALTH_NAME": "env-app"}),
            patch("dephealth.checks.tcp.TCPChecker.check", new_callable=AsyncMock),
        ):
            dh = DependencyHealth(
                "",
                "test-group",
                tcp_check("svc", host="localhost", port="8080", critical=True),
                registry=CollectorRegistry(),
            )
            assert dh is not None

    def test_api_name_overrides_env(self) -> None:
        with (
            patch.dict(os.environ, {"DEPHEALTH_NAME": "env-app"}),
            patch("dephealth.checks.tcp.TCPChecker.check", new_callable=AsyncMock),
        ):
            dh = DependencyHealth(
                "api-app",
                "test-group",
                tcp_check("svc", host="localhost", port="8080", critical=True),
                registry=CollectorRegistry(),
            )
            assert dh is not None


class TestInstanceGroup:
    def test_missing_group_raises(self) -> None:
        with pytest.raises(ValueError, match="group is required"):
            DependencyHealth(
                "test-app",
                "",
                tcp_check("svc", host="localhost", port="8080", critical=True),
                registry=CollectorRegistry(),
            )

    def test_invalid_group_raises(self) -> None:
        with pytest.raises(ValueError, match="invalid instance name"):
            DependencyHealth(
                "test-app",
                "Bad-Group",
                tcp_check("svc", host="localhost", port="8080", critical=True),
                registry=CollectorRegistry(),
            )

    def test_group_from_env(self) -> None:
        with (
            patch.dict(os.environ, {"DEPHEALTH_GROUP": "env-group"}),
            patch("dephealth.checks.tcp.TCPChecker.check", new_callable=AsyncMock),
        ):
            dh = DependencyHealth(
                "test-app",
                "",
                tcp_check("svc", host="localhost", port="8080", critical=True),
                registry=CollectorRegistry(),
            )
            assert dh is not None

    def test_api_group_overrides_env(self) -> None:
        with (
            patch.dict(os.environ, {"DEPHEALTH_GROUP": "env-group"}),
            patch("dephealth.checks.tcp.TCPChecker.check", new_callable=AsyncMock),
        ):
            dh = DependencyHealth(
                "test-app",
                "api-group",
                tcp_check("svc", host="localhost", port="8080", critical=True),
                registry=CollectorRegistry(),
            )
            assert dh is not None


class TestCriticalRequired:
    def test_critical_true(self) -> None:
        spec = tcp_check("svc", host="localhost", port="8080", critical=True)
        assert spec.critical is True

    def test_critical_false(self) -> None:
        spec = tcp_check("svc", host="localhost", port="8080", critical=False)
        assert spec.critical is False


class TestLabels:
    def test_labels_passed_to_spec(self) -> None:
        spec = tcp_check(
            "svc",
            host="localhost",
            port="8080",
            critical=True,
            labels={"env": "prod"},
        )
        assert spec.labels == {"env": "prod"}

    def test_labels_in_metrics(self) -> None:
        with patch("dephealth.checks.tcp.TCPChecker.check", new_callable=AsyncMock):
            registry = CollectorRegistry()
            dh = DependencyHealth(
                "test-app",
                "test-group",
                tcp_check(
                    "svc",
                    host="localhost",
                    port="8080",
                    critical=True,
                    labels={"env": "prod"},
                ),
                check_interval=timedelta(seconds=1),
                registry=registry,
            )
            assert dh is not None


class TestEnvVars:
    def test_critical_from_env(self) -> None:
        with (
            patch.dict(os.environ, {"DEPHEALTH_SVC_CRITICAL": "no"}),
            patch("dephealth.checks.tcp.TCPChecker.check", new_callable=AsyncMock),
        ):
            registry = CollectorRegistry()
            dh = DependencyHealth(
                "test-app",
                "test-group",
                tcp_check("svc", host="localhost", port="8080", critical=True),
                check_interval=timedelta(seconds=1),
                registry=registry,
            )
            assert dh is not None

    def test_critical_env_invalid(self) -> None:
        with (
            patch.dict(os.environ, {"DEPHEALTH_SVC_CRITICAL": "maybe"}),
            pytest.raises(ValueError, match="invalid value"),
        ):
            DependencyHealth(
                "test-app",
                "test-group",
                tcp_check("svc", host="localhost", port="8080", critical=True),
                registry=CollectorRegistry(),
            )

    def test_label_from_env(self) -> None:
        with (
            patch.dict(os.environ, {"DEPHEALTH_SVC_LABEL_ENV": "staging"}),
            patch("dephealth.checks.tcp.TCPChecker.check", new_callable=AsyncMock),
        ):
            registry = CollectorRegistry()
            dh = DependencyHealth(
                "test-app",
                "test-group",
                tcp_check("svc", host="localhost", port="8080", critical=True),
                check_interval=timedelta(seconds=1),
                registry=registry,
            )
            assert dh is not None

    def test_critical_env_with_hyphen_dep_name(self) -> None:
        """Dependency name with hyphen: my-db -> DEPHEALTH_MY_DB_CRITICAL."""
        with (
            patch.dict(os.environ, {"DEPHEALTH_MY_DB_CRITICAL": "yes"}),
            patch("dephealth.checks.postgres.PostgresChecker.check", new_callable=AsyncMock),
        ):
            registry = CollectorRegistry()
            dh = DependencyHealth(
                "test-app",
                "test-group",
                postgres_check(
                    "my-db",
                    url="postgres://localhost:5432/test",
                    critical=False,
                ),
                check_interval=timedelta(seconds=1),
                registry=registry,
            )
            assert dh is not None


class TestURLCredentials:
    def test_mysql_dsn_passed(self) -> None:
        """mysql_check(url=...) should pass dsn to MySQLChecker."""
        spec = mysql_check(
            "mysql-db",
            url="mysql://user:pass@mysql.svc:3306/mydb",
            critical=True,
        )
        assert spec.checker._dsn == "mysql://user:pass@mysql.svc:3306/mydb"

    def test_redis_url_passed(self) -> None:
        """redis_check(url=...) should pass url to RedisChecker."""
        spec = redis_check(
            "redis-cache",
            url="redis://:secret@redis.svc:6379/2",
            critical=False,
        )
        assert spec.checker._url == "redis://:secret@redis.svc:6379/2"

    def test_redis_explicit_password_priority(self) -> None:
        """Explicit password takes priority over URL."""
        spec = redis_check(
            "redis-cache",
            url="redis://:url-pass@redis.svc:6379/0",
            password="explicit-pass",
            critical=False,
        )
        assert spec.checker._password == "explicit-pass"


# --- Phase 5: Dynamic endpoint facade tests ---


def _make_checker(*, ok: bool = True) -> AsyncMock:
    """Create a mock HealthChecker."""
    checker = AsyncMock()
    if ok:
        checker.check = AsyncMock(return_value=None)
    else:
        checker.check = AsyncMock(side_effect=OSError("connection refused"))
    checker.checker_type = lambda: "tcp"
    return checker


def _make_dh(**kwargs: object) -> DependencyHealth:
    """Create a DependencyHealth with no initial dependencies."""
    defaults = {
        "name": "test-app",
        "group": "test-group",
        "check_interval": timedelta(seconds=1),
        "registry": CollectorRegistry(),
    }
    defaults.update(kwargs)
    return DependencyHealth(**defaults)


class TestDynamicAddEndpointFacade:
    """Integration tests for DependencyHealth.add_endpoint."""

    async def test_add_endpoint(self) -> None:
        """Create DependencyHealth, start, add_endpoint, verify health()."""
        dh = _make_dh()
        await dh.start()

        checker = _make_checker(ok=True)
        ep = Endpoint(host="facade-host", port="9090")
        await dh.add_endpoint("newsvc", DependencyType.TCP, True, ep, checker)

        await asyncio.sleep(0.3)
        health = dh.health()
        await dh.stop()

        assert "newsvc" in health
        assert health["newsvc"] is True

    async def test_add_endpoint_invalid_name(self) -> None:
        """Invalid dep name raises ValueError."""
        dh = _make_dh()
        await dh.start()

        checker = _make_checker()
        ep = Endpoint(host="h", port="1")

        with pytest.raises(ValueError, match="invalid dependency name"):
            await dh.add_endpoint("!!!bad", DependencyType.TCP, True, ep, checker)

        await dh.stop()

    async def test_add_endpoint_invalid_type(self) -> None:
        """Unknown type raises ValueError."""
        dh = _make_dh()
        await dh.start()

        checker = _make_checker()
        ep = Endpoint(host="h", port="1")

        with pytest.raises(ValueError, match="invalid dependency type"):
            await dh.add_endpoint("svc", "unknown_type", True, ep, checker)  # type: ignore[arg-type]

        await dh.stop()

    async def test_add_endpoint_missing_host(self) -> None:
        """Empty host raises ValueError."""
        dh = _make_dh()
        await dh.start()

        checker = _make_checker()
        ep = Endpoint(host="", port="1")

        with pytest.raises(ValueError, match="host must not be empty"):
            await dh.add_endpoint("svc", DependencyType.TCP, True, ep, checker)

        await dh.stop()

    async def test_add_endpoint_missing_port(self) -> None:
        """Empty port raises ValueError."""
        dh = _make_dh()
        await dh.start()

        checker = _make_checker()
        ep = Endpoint(host="h", port="")

        with pytest.raises(ValueError, match="port must not be empty"):
            await dh.add_endpoint("svc", DependencyType.TCP, True, ep, checker)

        await dh.stop()

    async def test_add_endpoint_reserved_label(self) -> None:
        """Reserved label raises ValueError."""
        dh = _make_dh()
        await dh.start()

        checker = _make_checker()
        ep = Endpoint(host="h", port="1", labels={"host": "reserved"})

        with pytest.raises(ValueError, match="reserved"):
            await dh.add_endpoint("svc", DependencyType.TCP, True, ep, checker)

        await dh.stop()


class TestDynamicRemoveEndpointFacade:
    """Integration tests for DependencyHealth.remove_endpoint."""

    async def test_remove_endpoint(self) -> None:
        """Remove, verify gone from health()."""
        dh = _make_dh()
        await dh.start()

        checker = _make_checker(ok=True)
        ep = Endpoint(host="rm-host", port="5050")
        await dh.add_endpoint("rmsvc", DependencyType.TCP, True, ep, checker)
        await asyncio.sleep(0.2)
        assert "rmsvc" in dh.health()

        await dh.remove_endpoint("rmsvc", "rm-host", "5050")
        assert "rmsvc" not in dh.health()

        await dh.stop()


class TestDynamicUpdateEndpointFacade:
    """Integration tests for DependencyHealth.update_endpoint."""

    async def test_update_endpoint(self) -> None:
        """Update, verify old gone and new present."""
        dh = _make_dh()
        await dh.start()

        checker = _make_checker(ok=True)
        old_ep = Endpoint(host="upd-old", port="3333")
        await dh.add_endpoint("updsvc", DependencyType.TCP, True, old_ep, checker)
        await asyncio.sleep(0.2)

        new_checker = _make_checker(ok=True)
        new_ep = Endpoint(host="upd-new", port="4444")
        await dh.update_endpoint("updsvc", "upd-old", "3333", new_ep, new_checker)
        await asyncio.sleep(0.2)

        details = dh.health_details()
        await dh.stop()

        assert "updsvc:upd-old:3333" not in details
        assert "updsvc:upd-new:4444" in details

    async def test_update_endpoint_missing_new_host(self) -> None:
        """Empty new host raises ValueError."""
        dh = _make_dh()
        await dh.start()

        checker = _make_checker(ok=True)
        old_ep = Endpoint(host="upd-old", port="3333")
        await dh.add_endpoint("updsvc", DependencyType.TCP, True, old_ep, checker)
        await asyncio.sleep(0.1)

        new_checker = _make_checker()
        new_ep = Endpoint(host="", port="4444")

        with pytest.raises(ValueError, match="host must not be empty"):
            await dh.update_endpoint("updsvc", "upd-old", "3333", new_ep, new_checker)

        await dh.stop()

    async def test_update_endpoint_not_found(self) -> None:
        """Update non-existent endpoint raises EndpointNotFoundError."""
        dh = _make_dh()
        await dh.start()

        new_checker = _make_checker()
        new_ep = Endpoint(host="h", port="1")

        with pytest.raises(EndpointNotFoundError):
            await dh.update_endpoint("ghost", "x", "0", new_ep, new_checker)

        await dh.stop()


class TestDynamicGlobalConfig:
    """Verify dynamic endpoints inherit global check_interval/timeout."""

    async def test_add_endpoint_inherits_global_config(self) -> None:
        """Dynamic endpoint uses global interval/timeout from DependencyHealth."""
        registry = CollectorRegistry()
        dh = DependencyHealth(
            "test-app",
            "test-group",
            check_interval=timedelta(seconds=2),
            timeout=timedelta(seconds=3),
            registry=registry,
        )
        await dh.start()

        checker = _make_checker(ok=True)
        ep = Endpoint(host="cfg-host", port="6060")
        await dh.add_endpoint("cfgsvc", DependencyType.TCP, True, ep, checker)

        # Verify config via scheduler internals.
        state_key = "cfgsvc:cfg-host:6060"
        entry = dh._scheduler._find_entry(state_key)
        assert entry is not None
        assert entry.dep.config.interval == 2.0
        assert entry.dep.config.timeout == 3.0

        await dh.stop()
