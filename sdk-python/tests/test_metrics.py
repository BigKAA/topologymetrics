"""Тесты для metrics.py — MetricsExporter."""

from prometheus_client import CollectorRegistry

from dephealth.dependency import Dependency, DependencyType, Endpoint
from dephealth.metrics import MetricsExporter


def _dep_and_ep() -> tuple[Dependency, Endpoint]:
    ep = Endpoint(host="localhost", port="5432")
    dep = Dependency(name="db", type=DependencyType.POSTGRES, endpoints=[ep])
    return dep, ep


class TestMetricsExporter:
    def test_set_health(self) -> None:
        registry = CollectorRegistry()
        m = MetricsExporter(registry=registry)
        dep, ep = _dep_and_ep()

        m.set_health(dep, ep, 1.0)

        # Проверяем через registry
        samples = list(registry.collect())
        health_samples = [
            s for metric in samples for s in metric.samples if s.name == "app_dependency_health"
        ]
        assert len(health_samples) == 1
        assert health_samples[0].value == 1.0
        assert health_samples[0].labels["dependency"] == "db"
        assert health_samples[0].labels["type"] == "postgres"

    def test_observe_latency(self) -> None:
        registry = CollectorRegistry()
        m = MetricsExporter(registry=registry)
        dep, ep = _dep_and_ep()

        m.observe_latency(dep, ep, 0.05)

        samples = list(registry.collect())
        count_samples = [
            s
            for metric in samples
            for s in metric.samples
            if s.name == "app_dependency_latency_seconds_count"
        ]
        assert len(count_samples) == 1
        assert count_samples[0].value == 1.0

    def test_delete_metrics(self) -> None:
        registry = CollectorRegistry()
        m = MetricsExporter(registry=registry)
        dep, ep = _dep_and_ep()

        m.set_health(dep, ep, 1.0)
        m.delete_metrics(dep, ep)

        samples = list(registry.collect())
        health_samples = [
            s for metric in samples for s in metric.samples if s.name == "app_dependency_health"
        ]
        assert len(health_samples) == 0

    def test_multiple_dependencies(self) -> None:
        registry = CollectorRegistry()
        m = MetricsExporter(registry=registry)

        ep1 = Endpoint(host="db1", port="5432")
        dep1 = Dependency(name="primary", type=DependencyType.POSTGRES, endpoints=[ep1])
        ep2 = Endpoint(host="cache", port="6379")
        dep2 = Dependency(name="cache", type=DependencyType.REDIS, endpoints=[ep2])

        m.set_health(dep1, ep1, 1.0)
        m.set_health(dep2, ep2, 0.0)

        samples = list(registry.collect())
        health_samples = [
            s for metric in samples for s in metric.samples if s.name == "app_dependency_health"
        ]
        assert len(health_samples) == 2
