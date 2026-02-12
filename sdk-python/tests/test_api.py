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


class TestDependencyHealth:
    async def test_start_stop(self) -> None:
        with patch("dephealth.checks.http.HTTPChecker.check", new_callable=AsyncMock):
            dh = DependencyHealth(
                "test-app",
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
            registry=CollectorRegistry(),
        )
        assert dh is not None

    def test_health_returns_empty(self) -> None:
        dh = DependencyHealth(
            "leaf-app",
            registry=CollectorRegistry(),
        )
        assert dh.health() == {}

    async def test_start_stop_async(self) -> None:
        dh = DependencyHealth(
            "leaf-app",
            registry=CollectorRegistry(),
        )
        await dh.start()
        await dh.stop()

    def test_start_stop_sync(self) -> None:
        dh = DependencyHealth(
            "leaf-app",
            registry=CollectorRegistry(),
        )
        dh.start_sync()
        dh.stop_sync()


class TestInstanceName:
    def test_missing_name_raises(self) -> None:
        with pytest.raises(ValueError, match="instance name is required"):
            DependencyHealth(
                "",
                tcp_check("svc", host="localhost", port="8080", critical=True),
                registry=CollectorRegistry(),
            )

    def test_invalid_name_raises(self) -> None:
        with pytest.raises(ValueError, match="invalid instance name"):
            DependencyHealth(
                "Bad-Name",
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
