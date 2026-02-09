package biz.kryukov.dev.dephealth.metrics;

import biz.kryukov.dev.dephealth.Dependency;
import biz.kryukov.dev.dephealth.Endpoint;

import io.micrometer.core.instrument.DistributionSummary;
import io.micrometer.core.instrument.Gauge;
import io.micrometer.core.instrument.MeterRegistry;
import io.micrometer.core.instrument.Tags;

import java.time.Duration;
import java.util.List;
import java.util.Map;
import java.util.Objects;
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
    private final String instanceName;
    private final List<String> customLabelNames;
    private final ConcurrentHashMap<String, AtomicReference<Double>> healthValues =
            new ConcurrentHashMap<>();
    private final ConcurrentHashMap<String, DistributionSummary> latencySummaries =
            new ConcurrentHashMap<>();

    /**
     * Создаёт экспортер метрик.
     *
     * @param registry      реестр Micrometer
     * @param instanceName  значение метки {@code name} (имя приложения)
     */
    public MetricsExporter(MeterRegistry registry, String instanceName) {
        this(registry, instanceName, List.of());
    }

    /**
     * Создаёт экспортер метрик с поддержкой произвольных меток.
     *
     * @param registry          реестр Micrometer
     * @param instanceName      значение метки {@code name} (имя приложения)
     * @param customLabelNames  имена произвольных меток (отсортированные по алфавиту)
     */
    public MetricsExporter(MeterRegistry registry, String instanceName,
                           List<String> customLabelNames) {
        this.registry = registry;
        this.instanceName = Objects.requireNonNull(instanceName, "instanceName");
        this.customLabelNames = List.copyOf(customLabelNames);
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

    /**
     * Строит теги в порядке: name, dependency, type, host, port, critical, custom (алфавит).
     */
    private Tags buildTags(Dependency dep, Endpoint ep) {
        Tags tags = Tags.of(
                "name", instanceName,
                "dependency", dep.name(),
                "type", dep.type().label(),
                "host", ep.host(),
                "port", ep.port(),
                "critical", Dependency.boolToYesNo(dep.critical())
        );
        // Добавляем произвольные метки из endpoint в алфавитном порядке
        Map<String, String> epLabels = ep.labels();
        for (String labelName : customLabelNames) {
            String value = epLabels.getOrDefault(labelName, "");
            tags = tags.and(labelName, value);
        }
        return tags;
    }

    private static String metricKey(String name, Endpoint ep) {
        return name + ":" + ep.host() + ":" + ep.port();
    }
}
