using Microsoft.Extensions.Diagnostics.HealthChecks;

namespace DepHealth.AspNetCore;

/// <summary>
/// Integration with ASP.NET Core Health Checks (/health).
/// </summary>
public sealed class DepHealthHealthCheck : IHealthCheck
{
    private readonly DepHealthMonitor _monitor;

    /// <summary>
    /// Initializes a new instance of the <see cref="DepHealthHealthCheck"/> class.
    /// </summary>
    /// <param name="monitor">The dephealth monitor instance to query.</param>
    public DepHealthHealthCheck(DepHealthMonitor monitor)
    {
        _monitor = monitor;
    }

    /// <summary>
    /// Runs the health check, returning the aggregated status of all monitored dependencies.
    /// </summary>
    public Task<HealthCheckResult> CheckHealthAsync(
        HealthCheckContext context,
        CancellationToken cancellationToken = default)
    {
        var health = _monitor.Health();

        if (health.Count == 0)
        {
            return Task.FromResult(HealthCheckResult.Healthy("No checks completed yet"));
        }

        var unhealthy = health.Where(kv => !kv.Value).Select(kv => kv.Key).ToList();

        if (unhealthy.Count == 0)
        {
            return Task.FromResult(HealthCheckResult.Healthy(
                $"All {health.Count} endpoints healthy"));
        }

        var data = new Dictionary<string, object>();
        foreach (var key in unhealthy)
        {
            data[key] = "unhealthy";
        }

        return Task.FromResult(HealthCheckResult.Unhealthy(
            $"{unhealthy.Count}/{health.Count} endpoints unhealthy",
            data: data));
    }
}
