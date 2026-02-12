package biz.kryukov.dev.dephealth.metrics;

import biz.kryukov.dev.dephealth.Dependency;
import biz.kryukov.dev.dephealth.DependencyType;
import biz.kryukov.dev.dephealth.Endpoint;

import io.micrometer.core.instrument.Gauge;
import io.micrometer.core.instrument.Meter;
import io.micrometer.core.instrument.MeterRegistry;
import io.micrometer.core.instrument.Tag;
import io.micrometer.core.instrument.simple.SimpleMeterRegistry;

import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;

import java.time.Duration;
import java.util.List;
import java.util.Map;
import java.util.stream.Collectors;

import static org.junit.jupiter.api.Assertions.*;

class MetricsExporterTest {

    private MeterRegistry registry;
    private MetricsExporter exporter;

    @BeforeEach
    void setUp() {
        registry = new SimpleMeterRegistry();
        exporter = new MetricsExporter(registry, "test-app");
    }

    private Dependency testDep(String name, DependencyType type) {
        return Dependency.builder(name, type)
                .endpoint(new Endpoint("localhost", "5432"))
                .critical(true)
                .build();
    }

    @Test
    void setHealthRegistersGaugeWithNameAndCritical() {
        Dependency dep = testDep("test-db", DependencyType.POSTGRES);
        Endpoint ep = dep.endpoints().get(0);

        exporter.setHealth(dep, ep, 1.0);

        Gauge gauge = registry.find("app_dependency_health")
                .tag("name", "test-app")
                .tag("dependency", "test-db")
                .tag("type", "postgres")
                .tag("host", "localhost")
                .tag("port", "5432")
                .tag("critical", "yes")
                .gauge();

        assertNotNull(gauge);
        assertEquals(1.0, gauge.value());
    }

    @Test
    void setHealthUpdatesValue() {
        Dependency dep = testDep("test-db", DependencyType.POSTGRES);
        Endpoint ep = dep.endpoints().get(0);

        exporter.setHealth(dep, ep, 1.0);
        exporter.setHealth(dep, ep, 0.0);

        Gauge gauge = registry.find("app_dependency_health")
                .tag("dependency", "test-db")
                .gauge();

        assertNotNull(gauge);
        assertEquals(0.0, gauge.value());
    }

    @Test
    void criticalNoTag() {
        Dependency dep = Dependency.builder("test-db", DependencyType.POSTGRES)
                .endpoint(new Endpoint("localhost", "5432"))
                .critical(false)
                .build();
        Endpoint ep = dep.endpoints().get(0);

        exporter.setHealth(dep, ep, 1.0);

        Gauge gauge = registry.find("app_dependency_health")
                .tag("critical", "no")
                .gauge();

        assertNotNull(gauge);
    }

    @Test
    void observeLatencyRecords() {
        Dependency dep = testDep("test-db", DependencyType.POSTGRES);
        Endpoint ep = dep.endpoints().get(0);

        exporter.observeLatency(dep, ep, Duration.ofMillis(100));

        Meter meter = registry.find("app_dependency_latency_seconds")
                .tag("dependency", "test-db")
                .tag("name", "test-app")
                .meter();

        assertNotNull(meter);
    }

    @Test
    void multipleEndpointsSeparateMetrics() {
        Endpoint ep1 = new Endpoint("host1", "5432");
        Endpoint ep2 = new Endpoint("host2", "5432");

        Dependency dep = Dependency.builder("test-db", DependencyType.POSTGRES)
                .endpoints(java.util.List.of(ep1, ep2))
                .critical(true)
                .build();

        exporter.setHealth(dep, ep1, 1.0);
        exporter.setHealth(dep, ep2, 0.0);

        assertEquals(1.0, registry.find("app_dependency_health")
                .tag("host", "host1").gauge().value());
        assertEquals(0.0, registry.find("app_dependency_health")
                .tag("host", "host2").gauge().value());
    }

    @Test
    void customLabelsIncluded() {
        MetricsExporter exporterWithLabels = new MetricsExporter(
                registry, "test-app", List.of("region", "zone"));

        Endpoint ep = new Endpoint("localhost", "5432",
                Map.of("region", "us-east", "zone", "a"));

        Dependency dep = Dependency.builder("test-db", DependencyType.POSTGRES)
                .endpoint(ep)
                .critical(true)
                .build();

        exporterWithLabels.setHealth(dep, ep, 1.0);

        Gauge gauge = registry.find("app_dependency_health")
                .tag("region", "us-east")
                .tag("zone", "a")
                .gauge();

        assertNotNull(gauge);
    }

    @Test
    void tagOrderIsCorrect() {
        MetricsExporter exporterWithLabels = new MetricsExporter(
                registry, "test-app", List.of("region"));

        Endpoint ep = new Endpoint("localhost", "5432", Map.of("region", "eu"));

        Dependency dep = Dependency.builder("test-db", DependencyType.POSTGRES)
                .endpoint(ep)
                .critical(true)
                .build();

        exporterWithLabels.setHealth(dep, ep, 1.0);

        Gauge gauge = registry.find("app_dependency_health")
                .tag("name", "test-app")
                .gauge();

        assertNotNull(gauge);
        // Verify tag order: name, dependency, type, host, port, critical, custom
        List<String> tagKeys = gauge.getId().getTags().stream()
                .map(Tag::getKey)
                .collect(Collectors.toList());
        assertEquals(List.of("critical", "dependency", "host", "name", "port", "region", "type"),
                tagKeys);
    }
}
