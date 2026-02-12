using Prometheus;

namespace DepHealth;

/// <summary>
/// Exports app_dependency_health and app_dependency_latency_seconds metrics
/// to the prometheus-net CollectorRegistry.
/// </summary>
public sealed class PrometheusExporter
{
    private const string HealthMetric = "app_dependency_health";
    private const string LatencyMetric = "app_dependency_latency_seconds";
    private const string HealthDescription = "Health status of a dependency (1 = healthy, 0 = unhealthy)";
    private const string LatencyDescription = "Latency of dependency health check in seconds";

    private static readonly double[] LatencyBuckets = [0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0, 5.0];
    private static readonly string[] RequiredLabelNames = ["name", "dependency", "type", "host", "port", "critical"];

    private readonly string _instanceName;
    private readonly string[] _customLabelNames;
    private readonly CollectorRegistry _registry;
    private readonly Gauge _healthGauge;
    private readonly Histogram _latencyHistogram;

    public PrometheusExporter(string instanceName, string[]? customLabelNames = null,
        CollectorRegistry? registry = null)
    {
        _instanceName = instanceName;
        _customLabelNames = customLabelNames ?? [];
        Array.Sort(_customLabelNames, StringComparer.Ordinal);
        _registry = registry ?? Metrics.DefaultRegistry;

        var allLabelNames = BuildLabelNames();

        _healthGauge = Metrics.WithCustomRegistry(_registry)
            .CreateGauge(HealthMetric, HealthDescription, new GaugeConfiguration
            {
                LabelNames = allLabelNames
            });

        _latencyHistogram = Metrics.WithCustomRegistry(_registry)
            .CreateHistogram(LatencyMetric, LatencyDescription, new HistogramConfiguration
            {
                LabelNames = allLabelNames,
                Buckets = LatencyBuckets
            });
    }

    /// <summary>
    /// Sets the health metric value (0 or 1).
    /// </summary>
    public void SetHealth(Dependency dep, Endpoint ep, double value)
    {
        _healthGauge
            .WithLabels(BuildLabelValues(dep, ep))
            .Set(value);
    }

    /// <summary>
    /// Records check latency into the histogram.
    /// </summary>
    public void ObserveLatency(Dependency dep, Endpoint ep, TimeSpan duration)
    {
        _latencyHistogram
            .WithLabels(BuildLabelValues(dep, ep))
            .Observe(duration.TotalSeconds);
    }

    private string[] BuildLabelNames()
    {
        var names = new string[RequiredLabelNames.Length + _customLabelNames.Length];
        RequiredLabelNames.CopyTo(names, 0);
        _customLabelNames.CopyTo(names, RequiredLabelNames.Length);
        return names;
    }

    private string[] BuildLabelValues(Dependency dep, Endpoint ep)
    {
        var values = new string[RequiredLabelNames.Length + _customLabelNames.Length];
        values[0] = _instanceName;
        values[1] = dep.Name;
        values[2] = dep.Type.Label();
        values[3] = ep.Host;
        values[4] = ep.Port;
        values[5] = Dependency.BoolToYesNo(dep.Critical);

        for (var i = 0; i < _customLabelNames.Length; i++)
        {
            ep.Labels.TryGetValue(_customLabelNames[i], out var val);
            values[RequiredLabelNames.Length + i] = val ?? "";
        }

        return values;
    }
}
