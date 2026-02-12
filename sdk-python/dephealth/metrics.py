"""Prometheus exporter: app_dependency_health and app_dependency_latency_seconds."""

from __future__ import annotations

from prometheus_client import REGISTRY, CollectorRegistry, Gauge, Histogram

from dephealth.dependency import Dependency, Endpoint, bool_to_yes_no

_DEFAULT_BUCKETS = (0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0, 5.0)
_REQUIRED_LABEL_NAMES = ("name", "dependency", "type", "host", "port", "critical")


class MetricsExporter:
    """Export dependency health metrics to Prometheus."""

    def __init__(
        self,
        instance_name: str,
        custom_label_names: tuple[str, ...] = (),
        registry: CollectorRegistry | None = None,
    ) -> None:
        self._instance_name = instance_name
        self._custom_label_names = custom_label_names
        reg = registry if registry is not None else REGISTRY

        label_names = _REQUIRED_LABEL_NAMES + self._custom_label_names

        self._health = Gauge(
            "app_dependency_health",
            "Health status of a dependency (1 = healthy, 0 = unhealthy)",
            labelnames=label_names,
            registry=reg,
        )

        self._latency = Histogram(
            "app_dependency_latency_seconds",
            "Latency of dependency health check in seconds",
            labelnames=label_names,
            buckets=_DEFAULT_BUCKETS,
            registry=reg,
        )

    def set_health(self, dep: Dependency, endpoint: Endpoint, value: float) -> None:
        """Set the gauge value (1.0 = healthy, 0.0 = unhealthy)."""
        labels = self._labels(dep, endpoint)
        self._health.labels(**labels).set(value)

    def observe_latency(self, dep: Dependency, endpoint: Endpoint, duration: float) -> None:
        """Record the check latency."""
        labels = self._labels(dep, endpoint)
        self._latency.labels(**labels).observe(duration)

    def delete_metrics(self, dep: Dependency, endpoint: Endpoint) -> None:
        """Delete metrics for a dependency/endpoint."""
        labels = self._labels(dep, endpoint)
        self._health.remove(*labels.values())
        self._latency.remove(*labels.values())

    def _labels(self, dep: Dependency, endpoint: Endpoint) -> dict[str, str]:
        result: dict[str, str] = {
            "name": self._instance_name,
            "dependency": dep.name,
            "type": str(dep.type),
            "host": endpoint.host,
            "port": endpoint.port,
            "critical": bool_to_yes_no(dep.critical),
        }
        for key in self._custom_label_names:
            result[key] = endpoint.labels.get(key, "")
        return result
