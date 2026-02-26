// Example: basic ASP.NET Core application with dephealth monitoring.
// Monitors an HTTP dependency and exposes Prometheus metrics.
//
// Prerequisites:
//   dotnet add package DepHealth.Core
//   dotnet add package DepHealth.AspNetCore
//
// Run:
//   dotnet run

using DepHealth;
using DepHealth.AspNetCore;
using Prometheus;

var builder = WebApplication.CreateBuilder(args);

// Register dephealth with a single HTTP dependency.
builder.Services.AddDepHealth("my-service", "backend", dh => dh
    .AddHttp("payment-api", "https://payment.internal:8443",
        healthPath: "/health", critical: true)
);

var app = builder.Build();

// Expose JSON health status on GET /health/dependencies.
app.MapDepHealthEndpoints();

// ASP.NET Core health checks (integrates with dephealth via IHealthCheck).
app.MapHealthChecks("/health");

// Expose Prometheus metrics on GET /metrics.
app.MapMetrics();

app.MapGet("/", () => new { message = "Hello, World!" });

app.Run();
