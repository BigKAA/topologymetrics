// Example: ASP.NET Core with Entity Framework Core integration.
// PostgreSQL health check is extracted automatically from DbContext.
//
// Prerequisites:
//   dotnet add package DepHealth.Core
//   dotnet add package DepHealth.AspNetCore
//   dotnet add package DepHealth.EntityFramework
//   dotnet add package Npgsql.EntityFrameworkCore.PostgreSQL
//
// Run:
//   DATABASE_URL="Host=pg.db;Port=5432;Database=orders;Username=app;Password=secret" dotnet run

using DepHealth;
using DepHealth.AspNetCore;
using DepHealth.EntityFramework;
using Microsoft.EntityFrameworkCore;
using Prometheus;

var builder = WebApplication.CreateBuilder(args);

var connectionString = builder.Configuration["DATABASE_URL"]
    ?? "Host=pg.db;Port=5432;Database=orders;Username=app;Password=secret";

// Register the DbContext with Npgsql.
builder.Services.AddDbContext<AppDbContext>(options =>
    options.UseNpgsql(connectionString));

// Register DepHealth using DbContext for PostgreSQL health checks.
// Because AddNpgsqlFromContext requires a DbContext instance,
// we build the monitor manually and register it as a singleton.
builder.Services.AddSingleton(sp =>
{
    var dbContext = sp.GetRequiredService<AppDbContext>();
    return DepHealthMonitor.CreateBuilder("order-service", "backend")
        .AddNpgsqlFromContext("postgres-main", dbContext, critical: true)
        .AddRedis("redis-cache", "redis://redis.cache:6379", critical: false)
        .Build();
});

// Register hosted service for automatic Start/Stop lifecycle.
builder.Services.AddHostedService<DepHealthHostedService>();

var app = builder.Build();

// Expose JSON health status on GET /health/dependencies.
app.MapDepHealthEndpoints();

// ASP.NET Core health checks.
app.MapHealthChecks("/health");

// Expose Prometheus metrics on GET /metrics.
app.MapMetrics();

app.MapGet("/", () => new { message = "Hello, World!" });

app.Run();

// --- Application DbContext ---

public class AppDbContext : DbContext
{
    public AppDbContext(DbContextOptions<AppDbContext> options) : base(options) { }
}
