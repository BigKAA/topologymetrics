using Microsoft.Extensions.Diagnostics.HealthChecks;

namespace DepHealth.AspNetCore.Tests;

public class DepHealthHealthCheckTests
{
    [Fact]
    public async Task CheckHealth_NoChecksCompleted_ReturnsHealthy()
    {
        using var monitor = DepHealthMonitor.CreateBuilder("test-app")
            .AddHttp("test", "http://localhost:8080", critical: true)
            .Build();

        var check = new DepHealthHealthCheck(monitor);
        var result = await check.CheckHealthAsync(
            new HealthCheckContext
            {
                Registration = new HealthCheckRegistration("dephealth", check,
                    HealthStatus.Unhealthy, Array.Empty<string>())
            });

        Assert.Equal(HealthStatus.Healthy, result.Status);
        Assert.Contains("No checks completed yet", result.Description!);
    }
}
