*[Русская версия](README.ru.md)*

# dephealth

SDK for monitoring microservice dependencies via Prometheus metrics.

## Features

- Automatic health checking for dependencies (PostgreSQL, MySQL, Redis, RabbitMQ, Kafka, HTTP, gRPC, TCP, LDAP)
- Prometheus metrics export: `app_dependency_health` (Gauge 0/1), `app_dependency_latency_seconds` (Histogram), `app_dependency_status` (enum), `app_dependency_status_detail` (info)
- Async architecture built on `async Task`
- ASP.NET Core integration (hosted service, middleware, health endpoints)
- Entity Framework integration for connection pool checks
- Connection pool support (preferred) and standalone checks

## Installation

```bash
dotnet add package DepHealth.AspNetCore
```

For Entity Framework integration:

```bash
dotnet add package DepHealth.EntityFramework
```

## Quick Start

### Standalone

```csharp
using DepHealth;

var dh = DepHealthMonitor.CreateBuilder("my-service", "my-team")
    .AddPostgres("db", "postgres://user:pass@localhost:5432/mydb", critical: true)
    .AddRedis("cache", "redis://localhost:6379", critical: false)
    .Build();

dh.Start();
// Metrics are available via prometheus-net
dh.Stop();
```

### ASP.NET Core

```csharp
using DepHealth;
using DepHealth.AspNetCore;

var builder = WebApplication.CreateBuilder(args);

builder.Services.AddDepHealth("my-service", "my-team", dh => dh
    .AddPostgres("db", builder.Configuration["DATABASE_URL"]!, critical: true)
    .AddRedis("cache", builder.Configuration["REDIS_URL"]!, critical: false)
);

var app = builder.Build();
app.UseDepHealth();
app.Run();
```

## Dynamic Endpoints

Add, remove, or replace monitored endpoints at runtime on a running instance
(v0.6.0+):

```csharp
using DepHealth;
using DepHealth.Checks;

// After dh.Start()...

// Add a new endpoint
dh.AddEndpoint(
    "api-backend",
    DependencyType.Http,
    true,
    new Endpoint("backend-2.svc", "8080"),
    new HttpChecker());

// Remove an endpoint (cancels check task, deletes metrics)
dh.RemoveEndpoint("api-backend", "backend-2.svc", "8080");

// Replace an endpoint atomically
dh.UpdateEndpoint(
    "api-backend",
    "backend-1.svc", "8080",
    new Endpoint("backend-3.svc", "8080"),
    new HttpChecker());
```

See [migration guide](docs/migration.md#v050-to-v060) for details.

## Health Details

```csharp
var details = dh.HealthDetails();
foreach (var (key, ep) in details)
{
    Console.WriteLine($"{key}: healthy={ep.Healthy} status={ep.Status} " +
        $"latency={ep.LatencyMillis:F1}ms");
}
```

## Configuration

| Parameter | Default | Description |
| --- | --- | --- |
| `WithCheckInterval` | `15s` | Check interval |
| `WithCheckTimeout` | `5s` | Check timeout |
| `WithInitialDelay` | `0s` | Initial delay before first check |

## Supported Dependencies

| Type | DependencyType | URL Format |
| --- | --- | --- |
| PostgreSQL | `Postgres` | `postgres://user:pass@host:5432/db` |
| MySQL | `MySql` | `mysql://user:pass@host:3306/db` |
| Redis | `Redis` | `redis://host:6379` |
| RabbitMQ | `Amqp` | `amqp://user:pass@host:5672/vhost` |
| Kafka | `Kafka` | `kafka://host1:9092,host2:9092` |
| HTTP | `Http` | `http://host:8080/health` |
| gRPC | `Grpc` | `host:50051` (via `Host()` + `Port()`) |
| TCP | `Tcp` | `tcp://host:port` |
| LDAP | `Ldap` | `ldap://host:389` or `ldaps://host:636` |

## LDAP Checker

LDAP health checker supports four check methods and multiple TLS modes:

```csharp
using DepHealth.Checks;

// RootDSE check (default)
var checker = new LdapChecker(checkMethod: LdapCheckMethod.RootDse);

// Simple bind with credentials
var checker = new LdapChecker(
    checkMethod: LdapCheckMethod.SimpleBind,
    bindDN: "cn=monitor,dc=corp,dc=com",
    bindPassword: "secret",
    useTls: true);

// Search with StartTLS
var checker = new LdapChecker(
    checkMethod: LdapCheckMethod.Search,
    baseDN: "dc=example,dc=com",
    searchFilter: "(objectClass=organizationalUnit)",
    searchScope: LdapSearchScope.One,
    startTls: true);
```

Check methods: `AnonymousBind`, `SimpleBind`, `RootDse` (default), `Search`.

## Authentication

HTTP and gRPC checkers support Bearer token, Basic Auth, and custom headers/metadata:

```csharp
builder.AddHttp("secure-api", "http://api.svc:8080",
    critical: true,
    bearerToken: "eyJhbG...");

builder.AddGrpc("grpc-backend", "backend.svc", "9090",
    critical: true,
    bearerToken: "eyJhbG...");
```

See [authentication guide](docs/authentication.md) for all options.

## Documentation

Full documentation is available in the [docs/](docs/README.md) directory.

## License

Apache License 2.0 — see [LICENSE](https://github.com/BigKAA/topologymetrics/blob/master/LICENSE).
