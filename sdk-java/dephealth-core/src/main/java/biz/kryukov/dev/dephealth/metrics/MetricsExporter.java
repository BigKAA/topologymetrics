package biz.kryukov.dev.dephealth.metrics;

import biz.kryukov.dev.dephealth.Dependency;
import biz.kryukov.dev.dephealth.Endpoint;
import biz.kryukov.dev.dephealth.StatusCategory;

import io.micrometer.core.instrument.DistributionSummary;
import io.micrometer.core.instrument.Gauge;
import io.micrometer.core.instrument.Meter;
import io.micrometer.core.instrument.MeterRegistry;
import io.micrometer.core.instrument.Tags;

import java.time.Duration;
import java.util.List;
import java.util.Map;
import java.util.Objects;
import java.util.concurrent.ConcurrentHashMap;
import java.util.concurrent.atomic.AtomicReference;

/**
 * Exports dependency health metrics to a Micrometer MeterRegistry.
 *
 * <p>Metrics exported:
 * <ul>
 *   <li>{@code app_dependency_health} — Gauge (0/1)</li>
 *   <li>{@code app_dependency_latency_seconds} — Histogram</li>
 *   <li>{@code app_dependency_status} — Gauge (enum pattern, 8 series per endpoint)</li>
 *   <li>{@code app_dependency_status_detail} — Gauge (info pattern, 1 series per endpoint)</li>
 * </ul>
 */
public final class MetricsExporter {

    private static final String HEALTH_METRIC = "app_dependency_health";
    private static final String LATENCY_METRIC = "app_dependency_latency_seconds";
    private static final String STATUS_METRIC = "app_dependency_status";
    private static final String STATUS_DETAIL_METRIC = "app_dependency_status_detail";
    private static final String HEALTH_DESCRIPTION =
            "Health status of a dependency (1 = healthy, 0 = unhealthy)";
    private static final String LATENCY_DESCRIPTION =
            "Latency of dependency health check in seconds";
    private static final String STATUS_DESCRIPTION =
            "Category of the last check result";
    private static final String STATUS_DETAIL_DESCRIPTION =
            "Detailed reason of the last check result";

    private static final double[] LATENCY_SLOS = {0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0, 5.0};

    private final MeterRegistry registry;
    private final String instanceName;
    private final String instanceGroup;
    private final List<String> customLabelNames;
    private final ConcurrentHashMap<String, AtomicReference<Double>> healthValues =
            new ConcurrentHashMap<>();
    private final ConcurrentHashMap<String, DistributionSummary> latencySummaries =
            new ConcurrentHashMap<>();

    /** Status enum: key → array of 8 AtomicReference (one per status category). */
    private final ConcurrentHashMap<String, AtomicReference<Double>[]> statusValues =
            new ConcurrentHashMap<>();
    /** Detail: key → AtomicReference for the current detail gauge value. */
    private final ConcurrentHashMap<String, AtomicReference<Double>> detailValues =
            new ConcurrentHashMap<>();
    /** Detail: key → the Meter registered for the current detail tag value. */
    private final ConcurrentHashMap<String, Meter> detailMeters = new ConcurrentHashMap<>();
    /** Detail: key → previous detail tag value (for delete-on-change). */
    private final ConcurrentHashMap<String, String> prevDetails = new ConcurrentHashMap<>();

    /**
     * Creates a metrics exporter.
     *
     * @param registry      Micrometer meter registry
     * @param instanceName  value of the {@code name} label (application name)
     * @param instanceGroup value of the {@code group} label (logical group)
     */
    public MetricsExporter(MeterRegistry registry, String instanceName, String instanceGroup) {
        this(registry, instanceName, instanceGroup, List.of());
    }

    /**
     * Creates a metrics exporter with custom label support.
     *
     * @param registry          Micrometer meter registry
     * @param instanceName      value of the {@code name} label (application name)
     * @param instanceGroup     value of the {@code group} label (logical group)
     * @param customLabelNames  custom label names (sorted alphabetically)
     */
    public MetricsExporter(MeterRegistry registry, String instanceName, String instanceGroup,
                           List<String> customLabelNames) {
        this.registry = registry;
        this.instanceName = Objects.requireNonNull(instanceName, "instanceName");
        this.instanceGroup = Objects.requireNonNull(instanceGroup, "instanceGroup");
        this.customLabelNames = List.copyOf(customLabelNames);
    }

    /**
     * Sets the health metric value (0 or 1).
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
     * Records the check latency into the histogram.
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
     * Sets the status enum gauge: exactly one of 8 values = 1, rest = 0.
     */
    @SuppressWarnings("unchecked")
    public void setStatus(Dependency dep, Endpoint ep, String category) {
        String key = metricKey(dep.name(), ep);
        AtomicReference<Double>[] refs = statusValues.computeIfAbsent(key, k -> {
            Tags baseTags = buildTags(dep, ep);
            List<String> cats = StatusCategory.ALL;
            AtomicReference<Double>[] arr = new AtomicReference[cats.size()];
            for (int i = 0; i < cats.size(); i++) {
                arr[i] = new AtomicReference<>(0.0);
                Tags tags = baseTags.and("status", cats.get(i));
                Gauge.builder(STATUS_METRIC, arr[i], AtomicReference::get)
                        .description(STATUS_DESCRIPTION)
                        .tags(tags)
                        .register(registry);
            }
            return arr;
        });
        List<String> cats = StatusCategory.ALL;
        for (int i = 0; i < cats.size(); i++) {
            refs[i].set(cats.get(i).equals(category) ? 1.0 : 0.0);
        }
    }

    /**
     * Sets the status detail gauge (delete-on-change pattern).
     */
    public void setStatusDetail(Dependency dep, Endpoint ep, String detail) {
        String key = metricKey(dep.name(), ep);
        String prev = prevDetails.get(key);

        if (prev != null && !prev.equals(detail)) {
            // Remove the old detail meter.
            Meter oldMeter = detailMeters.remove(key);
            if (oldMeter != null) {
                registry.remove(oldMeter);
            }
            detailValues.remove(key);
        }

        prevDetails.put(key, detail);

        AtomicReference<Double> ref = detailValues.computeIfAbsent(key, k -> {
            AtomicReference<Double> newRef = new AtomicReference<>(1.0);
            Tags tags = buildTags(dep, ep).and("detail", detail);
            Meter meter = Gauge.builder(STATUS_DETAIL_METRIC, newRef, AtomicReference::get)
                    .description(STATUS_DETAIL_DESCRIPTION)
                    .tags(tags)
                    .register(registry);
            detailMeters.put(key, meter);
            return newRef;
        });
        ref.set(1.0);
    }

    /**
     * Deletes all metric series for the given dependency endpoint.
     * Removes health gauge, latency summary, all 8 status gauges, and status detail gauge.
     *
     * @param dep dependency
     * @param ep  endpoint to remove metrics for
     */
    public void deleteMetrics(Dependency dep, Endpoint ep) {
        String key = metricKey(dep.name(), ep);

        // Remove health gauge
        AtomicReference<Double> healthRef = healthValues.remove(key);
        if (healthRef != null) {
            Tags tags = buildTags(dep, ep);
            registry.find(HEALTH_METRIC).tags(tags).meters()
                    .forEach(registry::remove);
        }

        // Remove latency distribution summary
        DistributionSummary summary = latencySummaries.remove(key);
        if (summary != null) {
            registry.remove(summary);
        }

        // Remove all 8 status gauges
        AtomicReference<Double>[] statusRefs = statusValues.remove(key);
        if (statusRefs != null) {
            Tags baseTags = buildTags(dep, ep);
            for (String cat : StatusCategory.ALL) {
                Tags tags = baseTags.and("status", cat);
                registry.find(STATUS_METRIC).tags(tags).meters()
                        .forEach(registry::remove);
            }
        }

        // Remove status detail gauge
        Meter detailMeter = detailMeters.remove(key);
        if (detailMeter != null) {
            registry.remove(detailMeter);
        }
        detailValues.remove(key);
        prevDetails.remove(key);
    }

    /**
     * Builds tags in order: name, group, dependency, type, host, port, critical, custom (alphabetical).
     */
    Tags buildTags(Dependency dep, Endpoint ep) {
        Tags tags = Tags.of(
                "name", instanceName,
                "group", instanceGroup,
                "dependency", dep.name(),
                "type", dep.type().label(),
                "host", ep.host(),
                "port", ep.port(),
                "critical", Dependency.boolToYesNo(dep.critical())
        );
        // Add custom labels from endpoint in alphabetical order
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
