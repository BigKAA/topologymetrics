using Prometheus;

namespace DepHealth;

/// <summary>
/// Exports app_dependency_health, app_dependency_latency_seconds,
/// app_dependency_status, and app_dependency_status_detail metrics
/// to the prometheus-net CollectorRegistry.
/// </summary>
public sealed class PrometheusExporter
{
    private const string HealthMetric = "app_dependency_health";
    private const string LatencyMetric = "app_dependency_latency_seconds";
    private const string StatusMetric = "app_dependency_status";
    private const string StatusDetailMetric = "app_dependency_status_detail";
    private const string HealthDescription = "Health status of a dependency (1 = healthy, 0 = unhealthy)";
    private const string LatencyDescription = "Latency of dependency health check in seconds";
    private const string StatusDescription = "Category of the last check result";
    private const string StatusDetailDescription = "Detailed reason of the last check result";

    private static readonly double[] LatencyBuckets = [0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0, 5.0];
    private static readonly string[] RequiredLabelNames = ["name", "group", "dependency", "type", "host", "port", "critical"];

    private readonly string _instanceName;
    private readonly string _instanceGroup;
    private readonly string[] _customLabelNames;
    private readonly Gauge _healthGauge;
    private readonly Histogram _latencyHistogram;
    private readonly Gauge _statusGauge;
    private readonly Gauge _statusDetailGauge;

    private readonly object _detailLock = new();
    private readonly Dictionary<string, string> _prevDetails = new();

    /// <summary>
    /// Initializes a new instance of the <see cref="PrometheusExporter"/> class
    /// and creates the Prometheus metric collectors.
    /// </summary>
    /// <param name="instanceName">Unique application instance name used as a label value.</param>
    /// <param name="instanceGroup">Logical group for this service instance.</param>
    /// <param name="customLabelNames">Optional additional label names to include in metrics.</param>
    /// <param name="registry">Optional custom Prometheus collector registry. Defaults to <see cref="Metrics.DefaultRegistry"/>.</param>
    public PrometheusExporter(string instanceName, string instanceGroup,
        string[]? customLabelNames = null, CollectorRegistry? registry = null)
    {
        _instanceName = instanceName;
        _instanceGroup = instanceGroup;
        _customLabelNames = customLabelNames ?? [];
        Array.Sort(_customLabelNames, StringComparer.Ordinal);
        var resolvedRegistry = registry ?? Metrics.DefaultRegistry;

        var allLabelNames = BuildLabelNames();
        var statusLabelNames = BuildLabelNamesWithExtra("status");
        var detailLabelNames = BuildLabelNamesWithExtra("detail");

        _healthGauge = Metrics.WithCustomRegistry(resolvedRegistry)
            .CreateGauge(HealthMetric, HealthDescription, new GaugeConfiguration
            {
                LabelNames = allLabelNames
            });

        _latencyHistogram = Metrics.WithCustomRegistry(resolvedRegistry)
            .CreateHistogram(LatencyMetric, LatencyDescription, new HistogramConfiguration
            {
                LabelNames = allLabelNames,
                Buckets = LatencyBuckets
            });

        _statusGauge = Metrics.WithCustomRegistry(resolvedRegistry)
            .CreateGauge(StatusMetric, StatusDescription, new GaugeConfiguration
            {
                LabelNames = statusLabelNames
            });

        _statusDetailGauge = Metrics.WithCustomRegistry(resolvedRegistry)
            .CreateGauge(StatusDetailMetric, StatusDetailDescription, new GaugeConfiguration
            {
                LabelNames = detailLabelNames
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

    /// <summary>
    /// Sets the status enum gauge: exactly one category = 1, rest = 0.
    /// </summary>
    public void SetStatus(Dependency dep, Endpoint ep, string category)
    {
        var baseValues = BuildLabelValues(dep, ep);

        foreach (var cat in StatusCategory.All)
        {
            var values = new string[baseValues.Length + 1];
            baseValues.CopyTo(values, 0);
            values[^1] = cat;

            _statusGauge
                .WithLabels(values)
                .Set(cat == category ? 1.0 : 0.0);
        }
    }

    /// <summary>
    /// Sets the status detail info gauge. Removes old series on detail change.
    /// </summary>
    public void SetStatusDetail(Dependency dep, Endpoint ep, string detail)
    {
        var baseValues = BuildLabelValues(dep, ep);
        var key = EndpointKey(dep, ep);

        lock (_detailLock)
        {
            if (_prevDetails.TryGetValue(key, out var prev) && prev != detail)
            {
                // Remove old detail series
                var oldValues = new string[baseValues.Length + 1];
                baseValues.CopyTo(oldValues, 0);
                oldValues[^1] = prev;
                _statusDetailGauge.RemoveLabelled(oldValues);
            }

            _prevDetails[key] = detail;
        }

        var values = new string[baseValues.Length + 1];
        baseValues.CopyTo(values, 0);
        values[^1] = detail;

        _statusDetailGauge
            .WithLabels(values)
            .Set(1.0);
    }

    /// <summary>
    /// Removes all metric series for the given dependency endpoint.
    /// </summary>
    public void DeleteMetrics(Dependency dep, Endpoint ep)
    {
        var baseValues = BuildLabelValues(dep, ep);

        _healthGauge.RemoveLabelled(baseValues);
        _latencyHistogram.RemoveLabelled(baseValues);

        // Remove all status series
        foreach (var cat in StatusCategory.All)
        {
            var values = new string[baseValues.Length + 1];
            baseValues.CopyTo(values, 0);
            values[^1] = cat;
            _statusGauge.RemoveLabelled(values);
        }

        // Remove detail series
        var key = EndpointKey(dep, ep);
        lock (_detailLock)
        {
            if (_prevDetails.TryGetValue(key, out var prev))
            {
                var detailValues = new string[baseValues.Length + 1];
                baseValues.CopyTo(detailValues, 0);
                detailValues[^1] = prev;
                _statusDetailGauge.RemoveLabelled(detailValues);
                _prevDetails.Remove(key);
            }
        }
    }

    private string[] BuildLabelNames()
    {
        var names = new string[RequiredLabelNames.Length + _customLabelNames.Length];
        RequiredLabelNames.CopyTo(names, 0);
        _customLabelNames.CopyTo(names, RequiredLabelNames.Length);
        return names;
    }

    private string[] BuildLabelNamesWithExtra(string extraLabel)
    {
        var names = new string[RequiredLabelNames.Length + _customLabelNames.Length + 1];
        RequiredLabelNames.CopyTo(names, 0);
        _customLabelNames.CopyTo(names, RequiredLabelNames.Length);
        names[^1] = extraLabel;
        return names;
    }

    private string[] BuildLabelValues(Dependency dep, Endpoint ep)
    {
        var values = new string[RequiredLabelNames.Length + _customLabelNames.Length];
        values[0] = _instanceName;
        values[1] = _instanceGroup;
        values[2] = dep.Name;
        values[3] = dep.Type.Label();
        values[4] = ep.Host;
        values[5] = ep.Port;
        values[6] = Dependency.BoolToYesNo(dep.Critical);

        for (var i = 0; i < _customLabelNames.Length; i++)
        {
            ep.Labels.TryGetValue(_customLabelNames[i], out var val);
            values[RequiredLabelNames.Length + i] = val ?? "";
        }

        return values;
    }

    private static string EndpointKey(Dependency dep, Endpoint ep) => $"{dep.Name}:{ep.Host}:{ep.Port}";
}
