using System.Text.Json;
using Microsoft.AspNetCore.Builder;
using Microsoft.AspNetCore.Http;
using Microsoft.AspNetCore.Routing;
using Microsoft.Extensions.DependencyInjection;
using Prometheus;

namespace DepHealth.AspNetCore;

/// <summary>
/// Extension methods for integrating dephealth with ASP.NET Core DI.
/// </summary>
public static class ServiceCollectionExtensions
{
    /// <summary>
    /// Registers dephealth in the DI container.
    /// </summary>
    /// <param name="services">The service collection.</param>
    /// <param name="name">Unique application name.</param>
    /// <param name="configure">Dependencies configuration.</param>
    public static IServiceCollection AddDepHealth(
        this IServiceCollection services,
        string name,
        Action<DepHealthMonitor.Builder> configure)
    {
        var builder = DepHealthMonitor.CreateBuilder(name);
        configure(builder);
        var monitor = builder.Build();

        services.AddSingleton(monitor);
        services.AddHostedService<DepHealthHostedService>();
        services.AddHealthChecks()
            .AddCheck<DepHealthHealthCheck>("dephealth");

        return services;
    }

    /// <summary>
    /// Maps the /health/dependencies endpoint with a JSON response about dependency status.
    /// </summary>
    public static IEndpointRouteBuilder MapDepHealthEndpoints(this IEndpointRouteBuilder endpoints)
    {
        endpoints.MapGet("/health/dependencies", (DepHealthMonitor monitor) =>
        {
            var health = monitor.Health();
            var result = health.ToDictionary(
                kv => kv.Key,
                kv => kv.Value ? "healthy" : "unhealthy");

            return Results.Json(result, new JsonSerializerOptions
            {
                WriteIndented = true
            });
        });

        return endpoints;
    }
}
