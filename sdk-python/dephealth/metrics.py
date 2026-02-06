"""Prometheus exporter: app_dependency_health и app_dependency_latency_seconds."""

from __future__ import annotations

from prometheus_client import REGISTRY, CollectorRegistry, Gauge, Histogram

from dephealth.dependency import Dependency, Endpoint

_DEFAULT_BUCKETS = (0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0, 5.0)
_LABEL_NAMES = ("dependency", "type", "host", "port")


class MetricsExporter:
    """Экспортирует метрики здоровья зависимостей в Prometheus."""

    def __init__(self, registry: CollectorRegistry | None = None) -> None:
        reg = registry if registry is not None else REGISTRY

        self._health = Gauge(
            "app_dependency_health",
            "Health status of a dependency endpoint (1=healthy, 0=unhealthy)",
            labelnames=_LABEL_NAMES,
            registry=reg,
        )

        self._latency = Histogram(
            "app_dependency_latency_seconds",
            "Latency of dependency health check in seconds",
            labelnames=_LABEL_NAMES,
            buckets=_DEFAULT_BUCKETS,
            registry=reg,
        )

    def set_health(self, dep: Dependency, endpoint: Endpoint, value: float) -> None:
        """Устанавливает значение gauge (1.0 = healthy, 0.0 = unhealthy)."""
        labels = self._labels(dep, endpoint)
        self._health.labels(**labels).set(value)

    def observe_latency(self, dep: Dependency, endpoint: Endpoint, duration: float) -> None:
        """Записывает latency проверки."""
        labels = self._labels(dep, endpoint)
        self._latency.labels(**labels).observe(duration)

    def delete_metrics(self, dep: Dependency, endpoint: Endpoint) -> None:
        """Удаляет метрики для зависимости/endpoint."""
        labels = self._labels(dep, endpoint)
        self._health.remove(*labels.values())
        self._latency.remove(*labels.values())

    @staticmethod
    def _labels(dep: Dependency, endpoint: Endpoint) -> dict[str, str]:
        return {
            "dependency": dep.name,
            "type": str(dep.type),
            "host": endpoint.host,
            "port": endpoint.port,
        }
