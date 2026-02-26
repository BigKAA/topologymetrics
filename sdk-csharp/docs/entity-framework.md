*[Русская версия](entity-framework.ru.md)*

# Entity Framework Integration

The `DepHealth.EntityFramework` package provides integration with
Entity Framework Core. It allows using an existing `DbContext` for
PostgreSQL health checks instead of creating separate connections.

## Installation

```bash
dotnet add package DepHealth.EntityFramework
```

This package depends on `DepHealth.Core` and adds the
`AddNpgsqlFromContext<TContext>()` extension method.

## Why Use EF Integration

| Aspect | Standalone | Entity Framework |
| --- | --- | --- |
| Connection | New per check | Reuses DbContext connection pool |
| Reflects real health | Partially | Yes |
| Detects pool exhaustion | No | Yes |
| Configuration | URL string | Automatic from DbContext |
| Additional load on DB | Yes (extra connections) | No |

## Basic Usage

```csharp
using DepHealth;
using DepHealth.AspNetCore;
using DepHealth.EntityFramework;

var builder = WebApplication.CreateBuilder(args);

// Register your DbContext
builder.Services.AddDbContext<AppDbContext>(options =>
    options.UseNpgsql(builder.Configuration["DATABASE_URL"]));

// Register dephealth with EF integration
builder.Services.AddDepHealth("my-service", "my-team", dh => dh
    .AddNpgsqlFromContext<AppDbContext>("postgres-main", critical: true)
    .AddRedis("redis-cache",
        builder.Configuration["REDIS_URL"]!,
        critical: false)
);

var app = builder.Build();
app.MapDepHealthEndpoints();
app.Run();
```

## How It Works

`AddNpgsqlFromContext<TContext>()` performs the following steps:

1. Resolves `TContext` from the DI container
2. Extracts the connection string via `context.Database.GetConnectionString()`
3. Parses host and port from the connection string
4. Creates a `PostgresChecker` using the connection string
5. Registers the dependency with the builder

The health check executes `SELECT 1` using a connection from the
same pool that the application uses. This means:

- If the pool is exhausted, the health check detects it
- If credentials expire, the health check fails with `auth_error`
- Latency reflects actual database performance as seen by the application

## API Reference

```csharp
public static DepHealthMonitor.Builder AddNpgsqlFromContext<TContext>(
    this DepHealthMonitor.Builder builder,
    string name,
    TContext context,
    bool? critical = null,
    Dictionary<string, string>? labels = null)
    where TContext : DbContext
```

| Parameter | Type | Description |
| --- | --- | --- |
| `name` | `string` | Dependency name |
| `context` | `TContext` | DbContext instance (resolved from DI) |
| `critical` | `bool?` | Critical flag (`null` → env var fallback) |
| `labels` | `Dictionary<string, string>?` | Custom Prometheus labels |

**Returns:** `DepHealthMonitor.Builder` for method chaining.

**Throws:** `ConfigurationException` if the connection string is not found
in the DbContext.

## Custom Labels

```csharp
builder.Services.AddDepHealth("my-service", "my-team", dh => dh
    .AddNpgsqlFromContext<AppDbContext>("postgres-main",
        critical: true,
        labels: new Dictionary<string, string>
        {
            ["role"] = "primary",
            ["shard"] = "eu-west"
        })
);
```

## Multiple DbContexts

```csharp
builder.Services.AddDbContext<OrderDbContext>(options =>
    options.UseNpgsql(builder.Configuration["ORDER_DB_URL"]));
builder.Services.AddDbContext<UserDbContext>(options =>
    options.UseNpgsql(builder.Configuration["USER_DB_URL"]));

builder.Services.AddDepHealth("my-service", "my-team", dh => dh
    .AddNpgsqlFromContext<OrderDbContext>("order-db", critical: true)
    .AddNpgsqlFromContext<UserDbContext>("user-db", critical: true)
);
```

## Connection String Format

The extension parses standard Npgsql connection strings:

```text
Host=pg.svc;Port=5432;Database=mydb;Username=user;Password=pass
```

Or PostgreSQL URL format (converted by Npgsql):

```text
postgresql://user:pass@pg.svc:5432/mydb
```

## Limitations

- Only PostgreSQL is supported via Entity Framework integration.
  For MySQL, Redis, and other databases, use the standard `Add*()` methods.
- The `DbContext` must have a configured connection string
  (via `UseNpgsql()` or similar).
- The health check uses the same connection pool, so pool size
  configuration affects health check availability.

## See Also

- [Getting Started](getting-started.md) — installation and basic setup
- [Connection Pools](connection-pools.md) — pool integration for all databases
- [ASP.NET Core Integration](aspnetcore.md) — DI registration and endpoints
- [Checkers](checkers.md) — PostgreSQL checker details
- [Configuration](configuration.md) — all options and defaults
- [Troubleshooting](troubleshooting.md) — common issues and solutions
