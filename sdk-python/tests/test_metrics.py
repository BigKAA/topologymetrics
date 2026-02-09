"""Тесты для metrics.py — MetricsExporter."""

from prometheus_client import CollectorRegistry

from dephealth.dependency import Dependency, DependencyType, Endpoint
from dephealth.metrics import MetricsExporter


def _dep_and_ep(
    critical: bool = True,
    labels: dict[str, str] | None = None,
) -> tuple[Dependency, Endpoint]:
    ep = Endpoint(host="localhost", port="5432", labels=labels or {})
    dep = Dependency(name="db", type=DependencyType.POSTGRES, critical=critical, endpoints=[ep])
    return dep, ep


class TestMetricsExporter:
    def test_set_health(self) -> None:
        registry = CollectorRegistry()
        m = MetricsExporter(instance_name="test-app", registry=registry)
        dep, ep = _dep_and_ep()

        m.set_health(dep, ep, 1.0)

        samples = list(registry.collect())
        health_samples = [
            s for metric in samples for s in metric.samples if s.name == "app_dependency_health"
        ]
        assert len(health_samples) == 1
        assert health_samples[0].value == 1.0
        assert health_samples[0].labels["name"] == "test-app"
        assert health_samples[0].labels["dependency"] == "db"
        assert health_samples[0].labels["type"] == "postgres"
        assert health_samples[0].labels["critical"] == "yes"

    def test_critical_no(self) -> None:
        registry = CollectorRegistry()
        m = MetricsExporter(instance_name="test-app", registry=registry)
        dep, ep = _dep_and_ep(critical=False)

        m.set_health(dep, ep, 1.0)

        samples = list(registry.collect())
        health_samples = [
            s for metric in samples for s in metric.samples if s.name == "app_dependency_health"
        ]
        assert health_samples[0].labels["critical"] == "no"

    def test_observe_latency(self) -> None:
        registry = CollectorRegistry()
        m = MetricsExporter(instance_name="test-app", registry=registry)
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
        m = MetricsExporter(instance_name="test-app", registry=registry)
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
        m = MetricsExporter(instance_name="test-app", registry=registry)

        ep1 = Endpoint(host="db1", port="5432")
        dep1 = Dependency(
            name="primary", type=DependencyType.POSTGRES, critical=True, endpoints=[ep1]
        )
        ep2 = Endpoint(host="cache", port="6379")
        dep2 = Dependency(name="cache", type=DependencyType.REDIS, critical=False, endpoints=[ep2])

        m.set_health(dep1, ep1, 1.0)
        m.set_health(dep2, ep2, 0.0)

        samples = list(registry.collect())
        health_samples = [
            s for metric in samples for s in metric.samples if s.name == "app_dependency_health"
        ]
        assert len(health_samples) == 2

    def test_custom_labels(self) -> None:
        registry = CollectorRegistry()
        m = MetricsExporter(
            instance_name="test-app",
            custom_label_names=("env", "region"),
            registry=registry,
        )
        ep = Endpoint(host="db1", port="5432", labels={"env": "prod", "region": "us"})
        dep = Dependency(
            name="primary", type=DependencyType.POSTGRES, critical=True, endpoints=[ep]
        )

        m.set_health(dep, ep, 1.0)

        samples = list(registry.collect())
        health_samples = [
            s for metric in samples for s in metric.samples if s.name == "app_dependency_health"
        ]
        assert len(health_samples) == 1
        assert health_samples[0].labels["env"] == "prod"
        assert health_samples[0].labels["region"] == "us"

    def test_custom_labels_missing_defaults_to_empty(self) -> None:
        registry = CollectorRegistry()
        m = MetricsExporter(
            instance_name="test-app",
            custom_label_names=("env",),
            registry=registry,
        )
        ep = Endpoint(host="db1", port="5432")
        dep = Dependency(
            name="primary", type=DependencyType.POSTGRES, critical=True, endpoints=[ep]
        )

        m.set_health(dep, ep, 1.0)

        samples = list(registry.collect())
        health_samples = [
            s for metric in samples for s in metric.samples if s.name == "app_dependency_health"
        ]
        assert health_samples[0].labels["env"] == ""

    def test_label_order(self) -> None:
        """Порядок меток: name, dependency, type, host, port, critical, custom (алфавит)."""
        registry = CollectorRegistry()
        m = MetricsExporter(
            instance_name="test-app",
            custom_label_names=("env", "region"),
            registry=registry,
        )
        ep = Endpoint(host="db1", port="5432", labels={"env": "prod", "region": "us"})
        dep = Dependency(
            name="primary", type=DependencyType.POSTGRES, critical=True, endpoints=[ep]
        )

        labels = m._labels(dep, ep)
        keys = list(labels.keys())
        assert keys == ["name", "dependency", "type", "host", "port", "critical", "env", "region"]
