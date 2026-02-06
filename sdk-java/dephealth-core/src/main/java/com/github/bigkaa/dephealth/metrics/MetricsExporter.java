package com.github.bigkaa.dephealth.metrics;

import com.github.bigkaa.dephealth.Dependency;
import com.github.bigkaa.dephealth.Endpoint;

import io.micrometer.core.instrument.DistributionSummary;
import io.micrometer.core.instrument.Gauge;
import io.micrometer.core.instrument.MeterRegistry;
import io.micrometer.core.instrument.Tags;

import java.time.Duration;
import java.util.concurrent.ConcurrentHashMap;
import java.util.concurrent.atomic.AtomicReference;

/**
 * Экспортирует метрики app_dependency_health и app_dependency_latency_seconds
 * в Micrometer MeterRegistry.
 */
public final class MetricsExporter {

    private static final String HEALTH_METRIC = "app_dependency_health";
    private static final String LATENCY_METRIC = "app_dependency_latency_seconds";
    private static final String HEALTH_DESCRIPTION =
            "Health status of a dependency (1 = healthy, 0 = unhealthy)";
    private static final String LATENCY_DESCRIPTION =
            "Latency of dependency health check in seconds";

    private static final double[] LATENCY_SLOS = {0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0, 5.0};

    private final MeterRegistry registry;
    private final ConcurrentHashMap<String, AtomicReference<Double>> healthValues =
            new ConcurrentHashMap<>();
    private final ConcurrentHashMap<String, DistributionSummary> latencySummaries =
            new ConcurrentHashMap<>();

    public MetricsExporter(MeterRegistry registry) {
        this.registry = registry;
    }

    /**
     * Устанавливает значение метрики health (0 или 1).
     */
    public void setHealth(Dependency dep, Endpoint ep, double value) {
        String key = metricKey(dep.name(), ep);
        AtomicReference<Double> ref = healthValues.computeIfAbsent(key, k -> {
            AtomicReference<Double> newRef = new AtomicReference<>(value);
            Tags tags = buildTags(dep, ep);
            Gauge.builder(HEALTH_METRIC, newRef, AtomicReference::get)
                    .description(HEALTH_DESCRIPTION)
                    .tags(tags)
                    .register(registry);
            return newRef;
        });
        ref.set(value);
    }

    /**
     * Записывает задержку проверки в histogram.
     */
    public void observeLatency(Dependency dep, Endpoint ep, Duration duration) {
        String key = metricKey(dep.name(), ep);
        DistributionSummary summary = latencySummaries.computeIfAbsent(key, k -> {
            Tags tags = buildTags(dep, ep);
            return DistributionSummary.builder(LATENCY_METRIC)
                    .description(LATENCY_DESCRIPTION)
                    .tags(tags)
                    .serviceLevelObjectives(LATENCY_SLOS)
                    .register(registry);
        });
        summary.record(duration.toNanos() / 1_000_000_000.0);
    }

    private static Tags buildTags(Dependency dep, Endpoint ep) {
        Tags tags = Tags.of(
                "dependency", dep.name(),
                "type", dep.type().label(),
                "host", ep.host(),
                "port", ep.port()
        );
        // Добавляем optional labels из metadata
        for (var entry : ep.metadata().entrySet()) {
            tags = tags.and(entry.getKey(), entry.getValue());
        }
        return tags;
    }

    private static String metricKey(String name, Endpoint ep) {
        return name + ":" + ep.host() + ":" + ep.port();
    }
}
