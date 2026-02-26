using System.Collections.ObjectModel;
using System.Text.Json.Serialization;

namespace DepHealth;

/// <summary>
/// Detailed health check state for a single endpoint.
/// Returned by <see cref="CheckScheduler.HealthDetails"/> and <see cref="DepHealthMonitor.HealthDetails"/>.
/// Immutable.
/// </summary>
public sealed class EndpointStatus
{
    /// <summary>Whether the endpoint is healthy.</summary>
    [JsonPropertyName("healthy")]
    public bool? Healthy { get; }

    /// <summary>Status category (e.g. ok, timeout, connection_error).</summary>
    [JsonPropertyName("status")]
    public string Status { get; }

    /// <summary>Detailed status description.</summary>
    [JsonPropertyName("detail")]
    public string Detail { get; }

    /// <summary>Last check latency.</summary>
    [JsonIgnore]
    public TimeSpan Latency { get; }

    /// <summary>Last check latency in milliseconds.</summary>
    [JsonPropertyName("latency_ms")]
    public double LatencyMillis => Latency.TotalMilliseconds;

    /// <summary>Dependency type.</summary>
    [JsonPropertyName("type")]
    public string Type { get; }

    /// <summary>Dependency name.</summary>
    [JsonPropertyName("name")]
    public string Name { get; }

    /// <summary>Endpoint host.</summary>
    [JsonPropertyName("host")]
    public string Host { get; }

    /// <summary>Endpoint port.</summary>
    [JsonPropertyName("port")]
    public string Port { get; }

    /// <summary>Whether this is a critical dependency.</summary>
    [JsonPropertyName("critical")]
    public bool Critical { get; }

    /// <summary>Timestamp of the last health check.</summary>
    [JsonPropertyName("last_checked_at")]
    public DateTimeOffset? LastCheckedAt { get; }

    /// <summary>Custom Prometheus labels for the endpoint.</summary>
    [JsonPropertyName("labels")]
    public IReadOnlyDictionary<string, string> Labels { get; }

    /// <summary>Creates a new <see cref="EndpointStatus"/> instance.</summary>
    public EndpointStatus(
        bool? healthy,
        string status,
        string detail,
        TimeSpan latency,
        string type,
        string name,
        string host,
        string port,
        bool critical,
        DateTimeOffset? lastCheckedAt,
        Dictionary<string, string> labels)
    {
        Healthy = healthy;
        Status = status;
        Detail = detail;
        Latency = latency;
        Type = type;
        Name = name;
        Host = host;
        Port = port;
        Critical = critical;
        LastCheckedAt = lastCheckedAt;
        Labels = new ReadOnlyDictionary<string, string>(
            new Dictionary<string, string>(labels));
    }
}
