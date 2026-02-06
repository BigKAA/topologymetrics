"""Публичный API: DependencyHealth и фабрики зависимостей."""

from __future__ import annotations

import logging
from datetime import timedelta
from typing import Any

from prometheus_client import CollectorRegistry

from dephealth.checker import HealthChecker
from dephealth.checks.amqp import AMQPChecker
from dephealth.checks.grpc import GRPCChecker
from dephealth.checks.http import HTTPChecker
from dephealth.checks.kafka import KafkaChecker
from dephealth.checks.mysql import MySQLChecker
from dephealth.checks.postgres import PostgresChecker
from dephealth.checks.redis import RedisChecker
from dephealth.checks.tcp import TCPChecker
from dephealth.dependency import (
    CheckConfig,
    Dependency,
    DependencyType,
    Endpoint,
)
from dephealth.metrics import MetricsExporter
from dephealth.parser import parse_params, parse_url
from dephealth.scheduler import CheckScheduler

logger = logging.getLogger("dephealth")


class _DependencySpec:
    """Спецификация зависимости (результат фабричной функции)."""

    def __init__(
        self,
        name: str,
        dep_type: DependencyType,
        checker: HealthChecker,
        endpoints: list[Endpoint],
        critical: bool = True,
        interval: timedelta | None = None,
        timeout: timedelta | None = None,
    ) -> None:
        self.name = name
        self.dep_type = dep_type
        self.checker = checker
        self.endpoints = endpoints
        self.critical = critical
        self.interval = interval
        self.timeout = timeout


class DependencyHealth:
    """Главный объект SDK: управление мониторингом зависимостей.

    Пример использования:

        dh = DependencyHealth(
            http_check("payment", url="http://payment:8080"),
            postgres_check("db", url="postgres://db:5432/mydb"),
            redis_check("cache", url="redis://cache:6379"),
            check_interval=timedelta(seconds=30),
        )
        await dh.start()
        # ...
        await dh.stop()
    """

    def __init__(
        self,
        *specs: _DependencySpec,
        check_interval: timedelta | None = None,
        timeout: timedelta | None = None,
        registry: CollectorRegistry | None = None,
        log: logging.Logger | None = None,
    ) -> None:
        self._log = log or logger
        self._metrics = MetricsExporter(registry=registry)
        self._scheduler = CheckScheduler(metrics=self._metrics, log=self._log)

        for spec in specs:
            interval = spec.interval or check_interval
            to = spec.timeout or timeout

            config = CheckConfig(
                initial_delay=0,
            )
            if interval is not None:
                config.interval = interval.total_seconds()
            if to is not None:
                config.timeout = to.total_seconds()

            dep = Dependency(
                name=spec.name,
                type=spec.dep_type,
                critical=spec.critical,
                endpoints=spec.endpoints,
                config=config,
            )
            dep.validate()
            self._scheduler.add(dep, spec.checker)

    async def start(self) -> None:
        """Запуск мониторинга (asyncio)."""
        await self._scheduler.start()

    async def stop(self) -> None:
        """Остановка мониторинга (asyncio)."""
        await self._scheduler.stop()

    def start_sync(self) -> None:
        """Запуск мониторинга (threading fallback)."""
        self._scheduler.start_sync()

    def stop_sync(self) -> None:
        """Остановка мониторинга (threading fallback)."""
        self._scheduler.stop_sync()

    def health(self) -> dict[str, bool]:
        """Текущее состояние всех зависимостей."""
        return self._scheduler.health()


# --- Фабрики зависимостей ---


def _endpoints_from_url(url: str) -> list[Endpoint]:
    """Парсит URL и возвращает список Endpoint."""
    parsed = parse_url(url)
    return [Endpoint(host=p.host, port=p.port) for p in parsed]


def http_check(
    name: str,
    *,
    url: str = "",
    host: str = "",
    port: str = "80",
    health_path: str = "/health",
    tls: bool = False,
    tls_skip_verify: bool = False,
    critical: bool = True,
    timeout: timedelta | None = None,
    interval: timedelta | None = None,
) -> _DependencySpec:
    """Создаёт HTTP-проверку."""
    endpoints = _endpoints_from_url(url) if url else [parse_params(host, port)]
    checker = HTTPChecker(
        health_path=health_path,
        tls=tls,
        tls_skip_verify=tls_skip_verify,
        timeout=timeout.total_seconds() if timeout else 5.0,
    )
    return _DependencySpec(
        name=name,
        dep_type=DependencyType.HTTP,
        checker=checker,
        endpoints=endpoints,
        critical=critical,
        interval=interval,
        timeout=timeout,
    )


def grpc_check(
    name: str,
    *,
    url: str = "",
    host: str = "",
    port: str = "443",
    service_name: str = "",
    tls: bool = False,
    tls_skip_verify: bool = False,
    critical: bool = True,
    timeout: timedelta | None = None,
    interval: timedelta | None = None,
) -> _DependencySpec:
    """Создаёт gRPC-проверку."""
    if url:
        parsed = parse_url(url)
        endpoints = [Endpoint(host=p.host, port=p.port) for p in parsed]
    else:
        endpoints = [parse_params(host, port)]
    checker = GRPCChecker(
        service_name=service_name,
        tls=tls,
        tls_skip_verify=tls_skip_verify,
        timeout=timeout.total_seconds() if timeout else 5.0,
    )
    return _DependencySpec(
        name=name,
        dep_type=DependencyType.GRPC,
        checker=checker,
        endpoints=endpoints,
        critical=critical,
        interval=interval,
        timeout=timeout,
    )


def tcp_check(
    name: str,
    *,
    host: str = "",
    port: str = "",
    critical: bool = True,
    timeout: timedelta | None = None,
    interval: timedelta | None = None,
) -> _DependencySpec:
    """Создаёт TCP-проверку."""
    endpoints = [parse_params(host, port)]
    checker = TCPChecker(timeout=timeout.total_seconds() if timeout else 5.0)
    return _DependencySpec(
        name=name,
        dep_type=DependencyType.TCP,
        checker=checker,
        endpoints=endpoints,
        critical=critical,
        interval=interval,
        timeout=timeout,
    )


def postgres_check(
    name: str,
    *,
    url: str = "",
    host: str = "",
    port: str = "5432",
    query: str = "SELECT 1",
    pool: Any = None,  # noqa: ANN401
    critical: bool = True,
    timeout: timedelta | None = None,
    interval: timedelta | None = None,
) -> _DependencySpec:
    """Создаёт PostgreSQL-проверку."""
    endpoints = _endpoints_from_url(url) if url else [parse_params(host, port)]
    checker = PostgresChecker(
        timeout=timeout.total_seconds() if timeout else 5.0,
        query=query,
        pool=pool,
        dsn=url,
    )
    return _DependencySpec(
        name=name,
        dep_type=DependencyType.POSTGRES,
        checker=checker,
        endpoints=endpoints,
        critical=critical,
        interval=interval,
        timeout=timeout,
    )


def mysql_check(
    name: str,
    *,
    url: str = "",
    host: str = "",
    port: str = "3306",
    query: str = "SELECT 1",
    pool: Any = None,  # noqa: ANN401
    critical: bool = True,
    timeout: timedelta | None = None,
    interval: timedelta | None = None,
) -> _DependencySpec:
    """Создаёт MySQL-проверку."""
    endpoints = _endpoints_from_url(url) if url else [parse_params(host, port)]
    checker = MySQLChecker(
        timeout=timeout.total_seconds() if timeout else 5.0,
        query=query,
        pool=pool,
    )
    return _DependencySpec(
        name=name,
        dep_type=DependencyType.MYSQL,
        checker=checker,
        endpoints=endpoints,
        critical=critical,
        interval=interval,
        timeout=timeout,
    )


def redis_check(
    name: str,
    *,
    url: str = "",
    host: str = "",
    port: str = "6379",
    password: str = "",
    db: int = 0,
    client: Any = None,  # noqa: ANN401
    critical: bool = True,
    timeout: timedelta | None = None,
    interval: timedelta | None = None,
) -> _DependencySpec:
    """Создаёт Redis-проверку."""
    endpoints = _endpoints_from_url(url) if url else [parse_params(host, port)]
    checker = RedisChecker(
        timeout=timeout.total_seconds() if timeout else 5.0,
        password=password,
        db=db,
        client=client,
    )
    return _DependencySpec(
        name=name,
        dep_type=DependencyType.REDIS,
        checker=checker,
        endpoints=endpoints,
        critical=critical,
        interval=interval,
        timeout=timeout,
    )


def amqp_check(
    name: str,
    *,
    url: str = "",
    host: str = "",
    port: str = "5672",
    critical: bool = True,
    timeout: timedelta | None = None,
    interval: timedelta | None = None,
) -> _DependencySpec:
    """Создаёт AMQP-проверку."""
    endpoints = _endpoints_from_url(url) if url else [parse_params(host, port)]
    checker = AMQPChecker(
        timeout=timeout.total_seconds() if timeout else 5.0,
        url=url,
    )
    return _DependencySpec(
        name=name,
        dep_type=DependencyType.AMQP,
        checker=checker,
        endpoints=endpoints,
        critical=critical,
        interval=interval,
        timeout=timeout,
    )


def kafka_check(
    name: str,
    *,
    url: str = "",
    host: str = "",
    port: str = "9092",
    critical: bool = True,
    timeout: timedelta | None = None,
    interval: timedelta | None = None,
) -> _DependencySpec:
    """Создаёт Kafka-проверку."""
    endpoints = _endpoints_from_url(url) if url else [parse_params(host, port)]
    checker = KafkaChecker(timeout=timeout.total_seconds() if timeout else 5.0)
    return _DependencySpec(
        name=name,
        dep_type=DependencyType.KAFKA,
        checker=checker,
        endpoints=endpoints,
        critical=critical,
        interval=interval,
        timeout=timeout,
    )
