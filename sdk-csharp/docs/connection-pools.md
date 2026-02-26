*[Русская версия](connection-pools.ru.md)*

# Connection Pool Integration

dephealth supports two modes for checking dependencies:

- **Standalone mode** — SDK creates a new connection for each health check
- **Pool mode** — SDK uses the existing connection pool of your service

Pool mode is preferred because it reflects the actual ability of the service
to work with the dependency. If the connection pool is exhausted, the health
check will detect it.

## Standalone vs Pool Mode

| Aspect | Standalone | Pool |
| --- | --- | --- |
| Connection | New per check | Reuses existing pool |
| Reflects real health | Partially | Yes |
| Setup | Simple — just URL | Requires passing pool object |
| External dependencies | None (uses checker's driver) | Your application's driver |
| Detects pool exhaustion | No | Yes |

## PostgreSQL via NpgsqlDataSource

Pass your existing `NpgsqlDataSource` to the dependency builder:

```csharp
using Npgsql;

NpgsqlDataSource dataSource = ...; // existing from DI

var dh = DepHealthMonitor.CreateBuilder("my-service", "my-team")
    .AddCustom("postgres-main", DependencyType.Postgres,
        "pg.svc", "5432",
        new PostgresChecker(dataSource),
        critical: true)
    .Build();
```

The checker calls `NpgsqlDataSource.OpenConnectionAsync()`, executes `SELECT 1`,
and closes the connection. Host and port are specified explicitly in `AddCustom`.

You can also provide the `NpgsqlDataSource` when using `AddNpgsqlFromContext`
is not available — for example, when working outside ASP.NET Core DI:

```csharp
using Npgsql;
using DepHealth.Checks;

NpgsqlDataSource dataSource = NpgsqlDataSource.Create(connectionString);

var checker = new PostgresChecker(dataSource);

var dh = DepHealthMonitor.CreateBuilder("my-service", "my-team")
    .AddCustom("postgres-main", DependencyType.Postgres,
        "pg.svc", "5432", checker, critical: true)
    .Build();
```

## PostgreSQL via Entity Framework

When using ASP.NET Core with Entity Framework Core, use `AddNpgsqlFromContext<TContext>()`:

```csharp
using DepHealth.EntityFramework;

builder.Services.AddDepHealth("my-service", "my-team", dh => dh
    .AddNpgsqlFromContext<AppDbContext>("postgres-main", critical: true)
);
```

This extension resolves `AppDbContext` from the DI container, extracts the
connection string, and creates a `PostgresChecker` using the same pool.
See [Entity Framework Integration](entity-framework.md) for details.

## Redis via IConnectionMultiplexer

Pass your existing `IConnectionMultiplexer` (StackExchange.Redis) to the
dependency builder:

```csharp
using StackExchange.Redis;

IConnectionMultiplexer redis = ...; // existing from DI

var dh = DepHealthMonitor.CreateBuilder("my-service", "my-team")
    .AddCustom("redis-cache", DependencyType.Redis,
        "redis.svc", "6379",
        new RedisChecker(redis),
        critical: false)
    .Build();
```

The checker calls `IConnectionMultiplexer.GetDatabase().PingAsync()` on the
provided instance. This reflects the real ability of the application to reach
Redis through the same multiplexer it uses for all other operations.

## LDAP via ILdapConnection

Pass your existing `ILdapConnection` (Novell.Directory.Ldap) to the
dependency builder:

```csharp
ILdapConnection ldapConn = ...; // existing connection

var dh = DepHealthMonitor.CreateBuilder("my-service", "my-team")
    .AddCustom("directory", DependencyType.Ldap,
        "ldap.svc", "389",
        new LdapChecker(ldapConn, checkMethod: LdapCheckMethod.RootDse),
        critical: true)
    .Build();
```

When using an existing connection, TLS settings (`useTls`, `startTls`) are
managed by the connection itself, not by the checker.

## Direct Checker Pool Mode

If you need more control, you can create checkers directly with pool
objects and register them via `AddEndpoint()`:

### PostgreSQL with NpgsqlDataSource

```csharp
using DepHealth.Checks;
using Npgsql;

NpgsqlDataSource dataSource = ...;

var checker = new PostgresChecker(dataSource);

// After dh.Build() and dh.Start()
dh.AddEndpoint("postgres-main", DependencyType.Postgres, critical: true,
    new Endpoint("pg.svc", "5432"),
    checker);
```

### Redis with IConnectionMultiplexer

```csharp
using DepHealth.Checks;
using StackExchange.Redis;

IConnectionMultiplexer multiplexer = ...;

var checker = new RedisChecker(multiplexer);

dh.AddEndpoint("redis-cache", DependencyType.Redis, critical: false,
    new Endpoint("redis.svc", "6379"),
    checker);
```

## Standalone vs Pool: When to Use Which

| Use case | Recommendation |
| --- | --- |
| Standard setup, one pool per dependency | Pool mode via `NpgsqlDataSource` / `IConnectionMultiplexer` |
| No existing pool (external services) | Standalone via URL |
| HTTP and gRPC services | Standalone only (no pool needed) |
| EF Core application | Entity Framework integration |
| LDAP with managed connection | Pool mode via `ILdapConnection` |

## Full Example: Mixed Modes

```csharp
using DepHealth;
using DepHealth.Checks;
using Npgsql;
using StackExchange.Redis;

// Existing connection pools
NpgsqlDataSource dataSource = NpgsqlDataSource.Create(
    Environment.GetEnvironmentVariable("DATABASE_URL")!);

IConnectionMultiplexer redis =
    await ConnectionMultiplexer.ConnectAsync(
        Environment.GetEnvironmentVariable("REDIS_URL")!);

var dh = DepHealthMonitor.CreateBuilder("my-service", "my-team")
    // Pool mode — PostgreSQL
    .AddCustom("postgres-main", DependencyType.Postgres,
        "pg.svc", "5432",
        new PostgresChecker(dataSource),
        critical: true)

    // Pool mode — Redis
    .AddCustom("redis-cache", DependencyType.Redis,
        "redis.svc", "6379",
        new RedisChecker(redis),
        critical: false)

    // Standalone mode — HTTP (no pool needed)
    .AddHttp("payment-api", "http://payment.svc:8080", critical: true)

    // Standalone mode — gRPC (no pool needed)
    .AddGrpc("user-service", host: "user.svc", port: "9090", critical: true)

    .Build();

dh.Start();
```

## ASP.NET Core Pool Integration

When using ASP.NET Core DI, inject pools from the service provider
and pass them to the builder:

```csharp
using DepHealth;
using DepHealth.AspNetCore;
using DepHealth.Checks;
using Npgsql;
using StackExchange.Redis;

var builder = WebApplication.CreateBuilder(args);

// Register NpgsqlDataSource in DI
builder.Services.AddNpgsqlDataSource(
    builder.Configuration["DATABASE_URL"]!);

// Register IConnectionMultiplexer in DI
builder.Services.AddSingleton<IConnectionMultiplexer>(_ =>
    ConnectionMultiplexer.Connect(
        builder.Configuration["REDIS_URL"]!));

// Register DepHealth using DI pools
builder.Services.AddSingleton(sp =>
{
    var dataSource = sp.GetRequiredService<NpgsqlDataSource>();
    var redis = sp.GetRequiredService<IConnectionMultiplexer>();

    return DepHealthMonitor.CreateBuilder("my-service", "my-team")
        .AddCustom("postgres-main", DependencyType.Postgres,
            "pg.svc", "5432",
            new PostgresChecker(dataSource),
            critical: true)
        .AddCustom("redis-cache", DependencyType.Redis,
            "redis.svc", "6379",
            new RedisChecker(redis),
            critical: false)
        .AddHttp("auth-service", "http://auth.svc:8080", critical: true)
        .Build();
});

builder.Services.AddHostedService<DepHealthHostedService>();

var app = builder.Build();
app.MapDepHealthEndpoints();
app.Run();
```

This overrides the `AddDepHealth` shorthand but still gets lifecycle
management and endpoint integration from `DepHealth.AspNetCore`.

## See Also

- [Checkers](checkers.md) — all checker details including pool options
- [Entity Framework Integration](entity-framework.md) — EF Core pool integration
- [ASP.NET Core Integration](aspnetcore.md) — DI registration with pools
- [Configuration](configuration.md) — connection string and interval options
- [API Reference](api-reference.md) — builder methods reference
