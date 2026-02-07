using System.Collections.Concurrent;
using Prometheus;

namespace DepHealth;

/// <summary>
/// Экспортирует метрики app_dependency_health и app_dependency_latency_seconds
/// в prometheus-net CollectorRegistry.
/// </summary>
public sealed class PrometheusExporter
{
    private const string HealthMetric = "app_dependency_health";
    private const string LatencyMetric = "app_dependency_latency_seconds";
    private const string HealthDescription = "Health status of a dependency (1 = healthy, 0 = unhealthy)";
    private const string LatencyDescription = "Latency of dependency health check in seconds";

    private static readonly double[] LatencyBuckets = [0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0, 5.0];
    private static readonly string[] LabelNames = ["dependency", "type", "host", "port"];

    private readonly CollectorRegistry _registry;
    private readonly Gauge _healthGauge;
    private readonly Histogram _latencyHistogram;

    public PrometheusExporter(CollectorRegistry? registry = null)
    {
        _registry = registry ?? Metrics.DefaultRegistry;

        _healthGauge = Metrics.WithCustomRegistry(_registry)
            .CreateGauge(HealthMetric, HealthDescription, new GaugeConfiguration
            {
                LabelNames = LabelNames
            });

        _latencyHistogram = Metrics.WithCustomRegistry(_registry)
            .CreateHistogram(LatencyMetric, LatencyDescription, new HistogramConfiguration
            {
                LabelNames = LabelNames,
                Buckets = LatencyBuckets
            });
    }

    /// <summary>
    /// Устанавливает значение метрики health (0 или 1).
    /// </summary>
    public void SetHealth(Dependency dep, Endpoint ep, double value)
    {
        _healthGauge
            .WithLabels(dep.Name, dep.Type.Label(), ep.Host, ep.Port)
            .Set(value);
    }

    /// <summary>
    /// Записывает задержку проверки в histogram.
    /// </summary>
    public void ObserveLatency(Dependency dep, Endpoint ep, TimeSpan duration)
    {
        _latencyHistogram
            .WithLabels(dep.Name, dep.Type.Label(), ep.Host, ep.Port)
            .Observe(duration.TotalSeconds);
    }
}
