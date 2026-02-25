"""Public API: DependencyHealth and dependency factory functions."""

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
from dephealth.checks.ldap import LdapChecker, LdapCheckMethod, LdapSearchScope
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
from dephealth.endpoint_status import EndpointStatus
from dephealth.metrics import MetricsExporter
from dephealth.parser import parse_params, parse_url
from dephealth.scheduler import CheckScheduler

logger = logging.getLogger("dephealth")

_INSTANCE_NAME_PATTERN = re.compile(r"^[a-z][a-z0-9-]{0,62}$")


def _validate_instance_name(name: str) -> None:
    """Validate the instance name."""
    if not _INSTANCE_NAME_PATTERN.match(name):
        msg = f"invalid instance name {name!r}: must match [a-z][a-z0-9-]{{{{0,62}}}}"
        raise ValueError(msg)


class _DependencySpec:
    """Dependency specification (result of a factory function)."""

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
    """Apply DEPHEALTH_<DEP>_CRITICAL and DEPHEALTH_<DEP>_LABEL_<KEY> environment variables."""
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
    """Collect all custom label keys from all specs and return a sorted tuple."""
    keys: set[str] = set()
    for spec in specs:
        keys.update(spec.labels.keys())
    return tuple(sorted(keys))


class DependencyHealth:
    """Main SDK object: manages dependency health monitoring.

    Usage example::

        dh = DependencyHealth(
            "my-service",
            "my-team",
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
        group: str,
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

        instance_group = group or os.environ.get("DEPHEALTH_GROUP", "")
        if not instance_group:
            msg = "group is required: pass as second argument or set DEPHEALTH_GROUP"
            raise ValueError(msg)
        _validate_instance_name(instance_group)

        self._log = log or logger

        for spec in specs:
            _apply_env_vars(spec)

        custom_label_keys = _collect_custom_label_keys(specs)
        self._metrics = MetricsExporter(
            instance_name=instance_name,
            instance_group=instance_group,
            custom_label_names=custom_label_keys,
            registry=registry,
        )

        global_config_kwargs: dict[str, Any] = {"initial_delay": 0}
        if check_interval is not None:
            global_config_kwargs["interval"] = check_interval.total_seconds()
        if timeout is not None:
            global_config_kwargs["timeout"] = timeout.total_seconds()
        global_config = CheckConfig(**global_config_kwargs)

        self._scheduler = CheckScheduler(
            metrics=self._metrics, global_config=global_config, log=self._log
        )

        for spec in specs:
            interval = spec.interval or check_interval
            to = spec.timeout or timeout

            config_kwargs: dict[str, Any] = {"initial_delay": 0}
            if interval is not None:
                config_kwargs["interval"] = interval.total_seconds()
            if to is not None:
                config_kwargs["timeout"] = to.total_seconds()
            config = CheckConfig(**config_kwargs)

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
        """Start monitoring (asyncio)."""
        await self._scheduler.start()

    async def stop(self) -> None:
        """Stop monitoring (asyncio)."""
        await self._scheduler.stop()

    def start_sync(self) -> None:
        """Start monitoring (threading fallback)."""
        self._scheduler.start_sync()

    def stop_sync(self) -> None:
        """Stop monitoring (threading fallback)."""
        self._scheduler.stop_sync()

    def health(self) -> dict[str, bool]:
        """Return current health status of all dependencies."""
        return self._scheduler.health()

    def health_details(self) -> dict[str, EndpointStatus]:
        """Return detailed health status of all endpoints."""
        return self._scheduler.health_details()

    # --- Dynamic endpoint management ---

    async def add_endpoint(
        self,
        dep_name: str,
        dep_type: DependencyType,
        critical: bool,
        endpoint: Endpoint,
        checker: HealthChecker,
    ) -> None:
        """Add a new endpoint at runtime (asyncio mode)."""
        _validate_dep_name(dep_name)
        _validate_dep_type(dep_type)
        _validate_endpoint(endpoint)
        await self._scheduler.add_endpoint(dep_name, dep_type, critical, endpoint, checker)

    async def remove_endpoint(self, dep_name: str, host: str, port: str) -> None:
        """Remove an endpoint at runtime (asyncio mode)."""
        await self._scheduler.remove_endpoint(dep_name, host, port)

    async def update_endpoint(
        self,
        dep_name: str,
        old_host: str,
        old_port: str,
        new_endpoint: Endpoint,
        checker: HealthChecker,
    ) -> None:
        """Replace an endpoint at runtime (asyncio mode)."""
        _validate_endpoint(new_endpoint)
        await self._scheduler.update_endpoint(dep_name, old_host, old_port, new_endpoint, checker)

    def add_endpoint_sync(
        self,
        dep_name: str,
        dep_type: DependencyType,
        critical: bool,
        endpoint: Endpoint,
        checker: HealthChecker,
    ) -> None:
        """Add a new endpoint at runtime (threading mode)."""
        _validate_dep_name(dep_name)
        _validate_dep_type(dep_type)
        _validate_endpoint(endpoint)
        self._scheduler.add_endpoint_sync(dep_name, dep_type, critical, endpoint, checker)

    def remove_endpoint_sync(self, dep_name: str, host: str, port: str) -> None:
        """Remove an endpoint at runtime (threading mode)."""
        self._scheduler.remove_endpoint_sync(dep_name, host, port)

    def update_endpoint_sync(
        self,
        dep_name: str,
        old_host: str,
        old_port: str,
        new_endpoint: Endpoint,
        checker: HealthChecker,
    ) -> None:
        """Replace an endpoint at runtime (threading mode)."""
        _validate_endpoint(new_endpoint)
        self._scheduler.update_endpoint_sync(dep_name, old_host, old_port, new_endpoint, checker)


def _validate_dep_name(name: str) -> None:
    """Validate a dependency name for dynamic endpoint operations."""
    from dephealth.dependency import validate_name

    validate_name(name)


def _validate_dep_type(dep_type: DependencyType) -> None:
    """Validate that dep_type is a valid DependencyType."""
    if not isinstance(dep_type, DependencyType):
        try:
            DependencyType(dep_type)
        except ValueError:
            msg = f"invalid dependency type: {dep_type!r}"
            raise ValueError(msg) from None


def _validate_endpoint(ep: Endpoint) -> None:
    """Validate endpoint host, port, and labels for dynamic operations."""
    if not ep.host:
        msg = "endpoint host must not be empty"
        raise ValueError(msg)
    if not ep.port:
        msg = "endpoint port must not be empty"
        raise ValueError(msg)
    if ep.labels:
        validate_labels(ep.labels)


# --- Dependency factory functions ---


def _endpoints_from_url(url: str) -> list[Endpoint]:
    """Parse a URL and return a list of Endpoints."""
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
    headers: dict[str, str] | None = None,
    bearer_token: str | None = None,
    basic_auth: tuple[str, str] | None = None,
    critical: bool,
    timeout: timedelta | None = None,
    interval: timedelta | None = None,
    labels: dict[str, str] | None = None,
) -> _DependencySpec:
    """Create an HTTP health check."""
    endpoints = _endpoints_from_url(url) if url else [parse_params(host, port)]
    checker = HTTPChecker(
        health_path=health_path,
        tls=tls,
        tls_skip_verify=tls_skip_verify,
        headers=headers,
        bearer_token=bearer_token,
        basic_auth=basic_auth,
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
    metadata: dict[str, str] | None = None,
    bearer_token: str | None = None,
    basic_auth: tuple[str, str] | None = None,
    critical: bool,
    timeout: timedelta | None = None,
    interval: timedelta | None = None,
    labels: dict[str, str] | None = None,
) -> _DependencySpec:
    """Create a gRPC health check."""
    if url:
        parsed = parse_url(url)
        endpoints = [Endpoint(host=p.host, port=p.port) for p in parsed]
    else:
        endpoints = [parse_params(host, port)]
    checker = GRPCChecker(
        service_name=service_name,
        tls=tls,
        tls_skip_verify=tls_skip_verify,
        metadata=metadata,
        bearer_token=bearer_token,
        basic_auth=basic_auth,
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
    """Create a TCP health check."""
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
    """Create a PostgreSQL health check."""
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
    """Create a MySQL health check."""
    endpoints = _endpoints_from_url(url) if url else [parse_params(host, port)]
    checker = MySQLChecker(
        timeout=timeout.total_seconds() if timeout else 5.0,
        query=query,
        pool=pool,
        dsn=url,
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
    """Create a Redis health check."""
    endpoints = _endpoints_from_url(url) if url else [parse_params(host, port)]
    checker = RedisChecker(
        timeout=timeout.total_seconds() if timeout else 5.0,
        password=password,
        db=db,
        client=client,
        url=url,
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
    """Create an AMQP health check."""
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


def ldap_check(
    name: str,
    *,
    url: str = "",
    host: str = "",
    port: str = "389",
    check_method: str = "root_dse",
    bind_dn: str = "",
    bind_password: str = "",
    base_dn: str = "",
    search_filter: str = "(objectClass=*)",
    search_scope: str = "base",
    start_tls: bool = False,
    tls_skip_verify: bool = False,
    client: Any = None,  # noqa: ANN401
    critical: bool,
    timeout: timedelta | None = None,
    interval: timedelta | None = None,
    labels: dict[str, str] | None = None,
) -> _DependencySpec:
    """Create an LDAP health check.

    Args:
        name: Dependency name.
        url: LDAP URL (ldap:// or ldaps://).
        host: LDAP server host (alternative to url).
        port: LDAP server port (default 389).
        check_method: Check method: anonymous_bind, simple_bind, root_dse, search.
        bind_dn: DN for simple_bind method.
        bind_password: Password for simple_bind method.
        base_dn: Base DN for search method.
        search_filter: LDAP filter for search method.
        search_scope: Search scope: base, one, sub.
        start_tls: Use StartTLS (only with ldap://).
        tls_skip_verify: Skip TLS certificate verification.
        client: Existing ldap3 Connection for pool mode.
        critical: Whether this dependency is critical.
        timeout: Check timeout.
        interval: Check interval.
        labels: Custom labels.
    """
    method = LdapCheckMethod(check_method)
    scope = LdapSearchScope(search_scope)

    if method == LdapCheckMethod.SIMPLE_BIND and (not bind_dn or not bind_password):
        msg = "simple_bind requires bind_dn and bind_password"
        raise ValueError(msg)
    if method == LdapCheckMethod.SEARCH and not base_dn:
        msg = "search method requires base_dn"
        raise ValueError(msg)

    use_tls = False
    if url:
        if url.lower().startswith("ldaps://"):
            use_tls = True
        if use_tls and start_tls:
            msg = "startTLS is incompatible with ldaps:// scheme"
            raise ValueError(msg)
        endpoints = _endpoints_from_url(url)
    else:
        endpoints = [parse_params(host, port)]

    checker = LdapChecker(
        timeout=timeout.total_seconds() if timeout else 5.0,
        check_method=method,
        bind_dn=bind_dn,
        bind_password=bind_password,
        base_dn=base_dn,
        search_filter=search_filter,
        search_scope=scope,
        use_tls=use_tls,
        start_tls=start_tls,
        tls_skip_verify=tls_skip_verify,
        client=client,
    )
    return _DependencySpec(
        name=name,
        dep_type=DependencyType.LDAP,
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
    """Create a Kafka health check."""
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
