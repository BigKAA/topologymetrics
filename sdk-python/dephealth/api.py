"""Публичный API: DependencyHealth и фабрики зависимостей."""

from __future__ import annotations

import logging
import os
import re
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
    validate_labels,
)
from dephealth.metrics import MetricsExporter
from dephealth.parser import parse_params, parse_url
from dephealth.scheduler import CheckScheduler

logger = logging.getLogger("dephealth")

_INSTANCE_NAME_PATTERN = re.compile(r"^[a-z][a-z0-9-]{0,62}$")


def _validate_instance_name(name: str) -> None:
    """Проверяет имя экземпляра (instance name)."""
    if not _INSTANCE_NAME_PATTERN.match(name):
        msg = f"invalid instance name {name!r}: must match [a-z][a-z0-9-]{{{{0,62}}}}"
        raise ValueError(msg)


class _DependencySpec:
    """Спецификация зависимости (результат фабричной функции)."""

    def __init__(
        self,
        name: str,
        dep_type: DependencyType,
        checker: HealthChecker,
        endpoints: list[Endpoint],
        critical: bool,
        interval: timedelta | None = None,
        timeout: timedelta | None = None,
        labels: dict[str, str] | None = None,
    ) -> None:
        self.name = name
        self.dep_type = dep_type
        self.checker = checker
        self.endpoints = endpoints
        self.critical = critical
        self.interval = interval
        self.timeout = timeout
        self.labels = labels or {}


def _apply_env_vars(spec: _DependencySpec) -> None:
    """Применяет переменные окружения DEPHEALTH_<DEP>_CRITICAL и DEPHEALTH_<DEP>_LABEL_<KEY>."""
    dep_key = spec.name.upper().replace("-", "_")

    critical_env = os.environ.get(f"DEPHEALTH_{dep_key}_CRITICAL")
    if critical_env is not None:
        if critical_env.lower() == "yes":
            spec.critical = True
        elif critical_env.lower() == "no":
            spec.critical = False
        else:
            msg = (
                f"invalid value for DEPHEALTH_{dep_key}_CRITICAL: "
                f"{critical_env!r}, expected 'yes' or 'no'"
            )
            raise ValueError(msg)

    label_prefix = f"DEPHEALTH_{dep_key}_LABEL_"
    for key, value in os.environ.items():
        if key.startswith(label_prefix):
            label_name = key[len(label_prefix) :].lower()
            spec.labels[label_name] = value


def _collect_custom_label_keys(specs: tuple[_DependencySpec, ...]) -> tuple[str, ...]:
    """Собирает все произвольные label keys из всех specs и возвращает отсортированный tuple."""
    keys: set[str] = set()
    for spec in specs:
        keys.update(spec.labels.keys())
    return tuple(sorted(keys))


class DependencyHealth:
    """Главный объект SDK: управление мониторингом зависимостей.

    Пример использования:

        dh = DependencyHealth(
            "my-service",
            http_check("payment", url="http://payment:8080", critical=True),
            postgres_check("db", url="postgres://db:5432/mydb", critical=True),
            redis_check("cache", url="redis://cache:6379", critical=False),
            check_interval=timedelta(seconds=30),
        )
        await dh.start()
        # ...
        await dh.stop()
    """

    def __init__(
        self,
        name: str,
        *specs: _DependencySpec,
        check_interval: timedelta | None = None,
        timeout: timedelta | None = None,
        registry: CollectorRegistry | None = None,
        log: logging.Logger | None = None,
    ) -> None:
        instance_name = name or os.environ.get("DEPHEALTH_NAME", "")
        if not instance_name:
            msg = "instance name is required: pass as first argument or set DEPHEALTH_NAME"
            raise ValueError(msg)
        _validate_instance_name(instance_name)

        self._log = log or logger

        for spec in specs:
            _apply_env_vars(spec)

        custom_label_keys = _collect_custom_label_keys(specs)
        self._metrics = MetricsExporter(
            instance_name=instance_name,
            custom_label_names=custom_label_keys,
            registry=registry,
        )
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

            validate_labels(spec.labels)

            endpoints = []
            for ep in spec.endpoints:
                merged_labels = {**spec.labels, **ep.labels}
                endpoints.append(Endpoint(host=ep.host, port=ep.port, labels=merged_labels))

            dep = Dependency(
                name=spec.name,
                type=spec.dep_type,
                critical=spec.critical,
                endpoints=endpoints,
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
    critical: bool,
    timeout: timedelta | None = None,
    interval: timedelta | None = None,
    labels: dict[str, str] | None = None,
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
        labels=labels,
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
    critical: bool,
    timeout: timedelta | None = None,
    interval: timedelta | None = None,
    labels: dict[str, str] | None = None,
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
        labels=labels,
    )


def tcp_check(
    name: str,
    *,
    host: str = "",
    port: str = "",
    critical: bool,
    timeout: timedelta | None = None,
    interval: timedelta | None = None,
    labels: dict[str, str] | None = None,
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
        labels=labels,
    )


def postgres_check(
    name: str,
    *,
    url: str = "",
    host: str = "",
    port: str = "5432",
    query: str = "SELECT 1",
    pool: Any = None,  # noqa: ANN401
    critical: bool,
    timeout: timedelta | None = None,
    interval: timedelta | None = None,
    labels: dict[str, str] | None = None,
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
        labels=labels,
    )


def mysql_check(
    name: str,
    *,
    url: str = "",
    host: str = "",
    port: str = "3306",
    query: str = "SELECT 1",
    pool: Any = None,  # noqa: ANN401
    critical: bool,
    timeout: timedelta | None = None,
    interval: timedelta | None = None,
    labels: dict[str, str] | None = None,
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
        labels=labels,
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
    critical: bool,
    timeout: timedelta | None = None,
    interval: timedelta | None = None,
    labels: dict[str, str] | None = None,
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
        labels=labels,
    )


def amqp_check(
    name: str,
    *,
    url: str = "",
    host: str = "",
    port: str = "5672",
    critical: bool,
    timeout: timedelta | None = None,
    interval: timedelta | None = None,
    labels: dict[str, str] | None = None,
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
        labels=labels,
    )


def kafka_check(
    name: str,
    *,
    url: str = "",
    host: str = "",
    port: str = "9092",
    critical: bool,
    timeout: timedelta | None = None,
    interval: timedelta | None = None,
    labels: dict[str, str] | None = None,
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
        labels=labels,
    )
