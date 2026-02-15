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
    [JsonPropertyName("healthy")]
    public bool? Healthy { get; }

    [JsonPropertyName("status")]
    public string Status { get; }

    [JsonPropertyName("detail")]
    public string Detail { get; }

    [JsonIgnore]
    public TimeSpan Latency { get; }

    [JsonPropertyName("latency_ms")]
    public double LatencyMillis => Latency.TotalMilliseconds;

    [JsonPropertyName("type")]
    public string Type { get; }

    [JsonPropertyName("name")]
    public string Name { get; }

    [JsonPropertyName("host")]
    public string Host { get; }

    [JsonPropertyName("port")]
    public string Port { get; }

    [JsonPropertyName("critical")]
    public bool Critical { get; }

    [JsonPropertyName("last_checked_at")]
    public DateTimeOffset? LastCheckedAt { get; }

    [JsonPropertyName("labels")]
    public IReadOnlyDictionary<string, string> Labels { get; }

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
