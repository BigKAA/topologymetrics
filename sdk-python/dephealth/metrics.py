"""Prometheus exporter for dependency health metrics.

Exports:
- app_dependency_health (Gauge, 0/1)
- app_dependency_latency_seconds (Histogram)
- app_dependency_status (Gauge, enum pattern — 8 series per endpoint)
- app_dependency_status_detail (Gauge, info pattern — 1 series per endpoint)
"""

from __future__ import annotations

import contextlib
import threading

from prometheus_client import REGISTRY, CollectorRegistry, Gauge, Histogram

from dephealth.check_result import ALL_STATUS_CATEGORIES
from dephealth.dependency import Dependency, Endpoint, bool_to_yes_no

_DEFAULT_BUCKETS = (0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0, 5.0)
_REQUIRED_LABEL_NAMES = ("name", "group", "dependency", "type", "host", "port", "critical")


class MetricsExporter:
    """Export dependency health metrics to Prometheus."""

    def __init__(
        self,
        instance_name: str,
        instance_group: str,
        custom_label_names: tuple[str, ...] = (),
        registry: CollectorRegistry | None = None,
    ) -> None:
        self._instance_name = instance_name
        self._instance_group = instance_group
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

        status_labels = label_names + ("status",)
        self._status = Gauge(
            "app_dependency_status",
            "Category of the last check result",
            labelnames=status_labels,
            registry=reg,
        )

        detail_labels = label_names + ("detail",)
        self._status_detail = Gauge(
            "app_dependency_status_detail",
            "Detailed reason of the last check result",
            labelnames=detail_labels,
            registry=reg,
        )

        self._prev_details: dict[str, str] = {}
        self._detail_lock = threading.Lock()

    def set_health(self, dep: Dependency, endpoint: Endpoint, value: float) -> None:
        """Set the gauge value (1.0 = healthy, 0.0 = unhealthy)."""
        labels = self._labels(dep, endpoint)
        self._health.labels(**labels).set(value)

    def observe_latency(self, dep: Dependency, endpoint: Endpoint, duration: float) -> None:
        """Record the check latency."""
        labels = self._labels(dep, endpoint)
        self._latency.labels(**labels).observe(duration)

    def set_status(self, dep: Dependency, endpoint: Endpoint, category: str) -> None:
        """Set the status enum gauge (exactly one of 8 values = 1, rest = 0)."""
        base = self._labels(dep, endpoint)
        for cat in ALL_STATUS_CATEGORIES:
            lbl = {**base, "status": cat}
            self._status.labels(**lbl).set(1.0 if cat == category else 0.0)

    def set_status_detail(self, dep: Dependency, endpoint: Endpoint, detail: str) -> None:
        """Set the status detail gauge (delete-on-change pattern)."""
        base = self._labels(dep, endpoint)
        key = _endpoint_key(dep, endpoint)

        with self._detail_lock:
            prev = self._prev_details.get(key)
            if prev is not None and prev != detail:
                old_values = list(base.values()) + [prev]
                with contextlib.suppress(KeyError):
                    self._status_detail.remove(*old_values)
            self._prev_details[key] = detail

        lbl = {**base, "detail": detail}
        self._status_detail.labels(**lbl).set(1.0)

    def delete_metrics(self, dep: Dependency, endpoint: Endpoint) -> None:
        """Delete metrics for a dependency/endpoint."""
        labels = self._labels(dep, endpoint)
        label_values = list(labels.values())
        self._health.remove(*label_values)
        self._latency.remove(*label_values)

        # Delete all status enum series.
        for cat in ALL_STATUS_CATEGORIES:
            with contextlib.suppress(KeyError):
                self._status.remove(*(label_values + [cat]))

        # Delete status detail series.
        key = _endpoint_key(dep, endpoint)
        with self._detail_lock:
            prev = self._prev_details.pop(key, None)
        if prev is not None:
            with contextlib.suppress(KeyError):
                self._status_detail.remove(*(label_values + [prev]))

    def _labels(self, dep: Dependency, endpoint: Endpoint) -> dict[str, str]:
        result: dict[str, str] = {
            "name": self._instance_name,
            "group": self._instance_group,
            "dependency": dep.name,
            "type": str(dep.type),
            "host": endpoint.host,
            "port": endpoint.port,
            "critical": bool_to_yes_no(dep.critical),
        }
        for key in self._custom_label_names:
            result[key] = endpoint.labels.get(key, "")
        return result


def _endpoint_key(dep: Dependency, endpoint: Endpoint) -> str:
    """Build a unique key for tracking per-endpoint detail state."""
    return f"{dep.name}/{endpoint.host}:{endpoint.port}"
