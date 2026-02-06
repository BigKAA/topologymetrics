package com.github.bigkaa.dephealth.metrics;

import com.github.bigkaa.dephealth.CheckConfig;
import com.github.bigkaa.dephealth.Dependency;
import com.github.bigkaa.dephealth.DependencyType;
import com.github.bigkaa.dephealth.Endpoint;

import io.micrometer.core.instrument.Gauge;
import io.micrometer.core.instrument.Meter;
import io.micrometer.core.instrument.MeterRegistry;
import io.micrometer.core.instrument.simple.SimpleMeterRegistry;

import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;

import java.time.Duration;

import static org.junit.jupiter.api.Assertions.*;

class MetricsExporterTest {

    private MeterRegistry registry;
    private MetricsExporter exporter;

    @BeforeEach
    void setUp() {
        registry = new SimpleMeterRegistry();
        exporter = new MetricsExporter(registry);
    }

    private Dependency testDep(String name, DependencyType type) {
        return Dependency.builder(name, type)
                .endpoint(new Endpoint("localhost", "5432"))
                .build();
    }

    @Test
    void setHealthRegistersGauge() {
        Dependency dep = testDep("test-db", DependencyType.POSTGRES);
        Endpoint ep = dep.endpoints().get(0);

        exporter.setHealth(dep, ep, 1.0);

        Gauge gauge = registry.find("app_dependency_health")
                .tag("dependency", "test-db")
                .tag("type", "postgres")
                .tag("host", "localhost")
                .tag("port", "5432")
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
    void observeLatencyRecords() {
        Dependency dep = testDep("test-db", DependencyType.POSTGRES);
        Endpoint ep = dep.endpoints().get(0);

        exporter.observeLatency(dep, ep, Duration.ofMillis(100));

        Meter meter = registry.find("app_dependency_latency_seconds")
                .tag("dependency", "test-db")
                .meter();

        assertNotNull(meter);
    }

    @Test
    void multipleEndpointsSeparateMetrics() {
        Endpoint ep1 = new Endpoint("host1", "5432");
        Endpoint ep2 = new Endpoint("host2", "5432");

        Dependency dep = Dependency.builder("test-db", DependencyType.POSTGRES)
                .endpoints(java.util.List.of(ep1, ep2))
                .build();

        exporter.setHealth(dep, ep1, 1.0);
        exporter.setHealth(dep, ep2, 0.0);

        assertEquals(1.0, registry.find("app_dependency_health")
                .tag("host", "host1").gauge().value());
        assertEquals(0.0, registry.find("app_dependency_health")
                .tag("host", "host2").gauge().value());
    }
}
