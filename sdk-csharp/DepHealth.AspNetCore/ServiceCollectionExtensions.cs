using Microsoft.AspNetCore.Builder;
using Microsoft.AspNetCore.Http;
using Microsoft.AspNetCore.Routing;
using Microsoft.Extensions.DependencyInjection;
using Prometheus;
using System.Text.Json;

namespace DepHealth.AspNetCore;

/// <summary>
/// Extension methods для интеграции dephealth с ASP.NET Core DI.
/// </summary>
public static class ServiceCollectionExtensions
{
    /// <summary>
    /// Регистрирует dephealth в DI-контейнере.
    /// </summary>
    /// <param name="services">Коллекция сервисов.</param>
    /// <param name="configure">Конфигурация зависимостей.</param>
    public static IServiceCollection AddDepHealth(
        this IServiceCollection services,
        Action<DepHealthMonitor.Builder> configure)
    {
        var builder = DepHealthMonitor.CreateBuilder();
        configure(builder);
        var monitor = builder.Build();

        services.AddSingleton(monitor);
        services.AddHostedService<DepHealthHostedService>();
        services.AddHealthChecks()
            .AddCheck<DepHealthHealthCheck>("dephealth");

        return services;
    }

    /// <summary>
    /// Маппит эндпоинт /health/dependencies с JSON-ответом о состоянии зависимостей.
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
