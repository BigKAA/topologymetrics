// Example: multiple dependencies with custom labels,
// a Kubernetes readiness probe, and a JSON health details endpoint.
//
// Prerequisites:
//   dotnet add package DepHealth.Core
//   dotnet add package DepHealth.AspNetCore
//
// Run:
//   dotnet run

using System.Text.Json;
using DepHealth;
using DepHealth.AspNetCore;
using Prometheus;

var builder = WebApplication.CreateBuilder(args);

// Register dephealth with multiple dependencies and global options.
builder.Services.AddDepHealth("order-service", "backend", dh => dh
    .WithCheckInterval(TimeSpan.FromSeconds(10))
    .WithCheckTimeout(TimeSpan.FromSeconds(3))

    // PostgreSQL database.
    .AddPostgres("postgres-main",
        "postgres://app:secret@pg.db:5432/orders",
        critical: true,
        labels: new Dictionary<string, string> { ["env"] = "production" })

    // Redis cache.
    .AddRedis("redis-cache",
        "redis://:redis-pass@redis.cache:6379",
        critical: false,
        labels: new Dictionary<string, string> { ["env"] = "production" })

    // HTTP with Bearer token authentication.
    .AddHttp("auth-service",
        "https://auth.internal:8443",
        healthPath: "/healthz",
        bearerToken: "my-service-token",
        critical: true,
        labels: new Dictionary<string, string> { ["env"] = "production" })

    // gRPC dependency.
    .AddGrpc("recommendation-grpc",
        "recommend.internal", "9090",
        critical: false,
        labels: new Dictionary<string, string> { ["env"] = "production" })

    // Kafka brokers.
    .AddKafka("events-kafka",
        "kafka://kafka-0.broker:9092,kafka-1.broker:9092,kafka-2.broker:9092",
        critical: true,
        labels: new Dictionary<string, string> { ["env"] = "production" })
);

var app = builder.Build();

// JSON dependency health status.
app.MapDepHealthEndpoints();

// Prometheus metrics.
app.MapMetrics();

// Kubernetes readiness probe: 200 if all critical deps healthy, 503 otherwise.
app.MapGet("/readyz", (DepHealthMonitor monitor) =>
{
    var health = monitor.Health();
    var ready = health.Values.All(ok => ok);
    return ready
        ? Results.Text("ok", statusCode: 200)
        : Results.Text("not ready", statusCode: 503);
});

// Debug endpoint: detailed JSON health status per endpoint.
app.MapGet("/healthz", (DepHealthMonitor monitor) =>
{
    var details = monitor.HealthDetails();
    return Results.Json(details, new JsonSerializerOptions { WriteIndented = true });
});

app.Run();
