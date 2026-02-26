*[Русская версия](troubleshooting.ru.md)*

# Troubleshooting

Common issues and solutions when using the dephealth C# SDK.

## Empty Metrics / No Metrics Exported

**Symptom:** The `/metrics` endpoint returns no `app_dependency_*` metrics.

**Possible causes:**

1. **`Start()` not called.** Metrics are only registered and updated after
   `Start()` is called. Check that `Start()` is invoked and returns without
   error.

   For ASP.NET Core: verify that `DepHealth.AspNetCore` is registered via
   `AddDepHealth` — it calls `Start()` automatically via
   `DepHealthHostedService`.

2. **Wrong Prometheus endpoint.** Metrics are exposed at `/metrics` via
   `prometheus-net.AspNetCore`. Ensure the package is installed and
   `app.MapMetrics()` is called:

   ```csharp
   using Prometheus;

   var app = builder.Build();
   app.UseHttpMetrics();
   app.MapDepHealthEndpoints();
   app.MapMetrics();  // exposes /metrics
   app.Run();
   ```

3. **Registry not connected.** For programmatic API, verify that you pass
   the same `CollectorRegistry` to the builder via `WithRegistry()`, or use
   the default registry `Metrics.DefaultRegistry`:

   ```csharp
   var registry = Metrics.NewCustomRegistry();
   var dh = DepHealthMonitor.CreateBuilder("my-service", "my-team")
       .WithRegistry(registry)
       // ...
       .Build();
   dh.Start();

   // Same registry for /metrics handler
   var metricFactory = Metrics.WithCustomRegistry(registry);
   ```

## All Dependencies Show 0 (Unhealthy)

**Symptom:** `app_dependency_health` is `0` for all dependencies.

**Possible causes:**

1. **Network accessibility** — verify the target services are reachable
   from the service container/pod.

2. **DNS resolution** — check that service names resolve correctly.

3. **Wrong URL/host/port** — double-check configuration values.

4. **Timeout too low** — default is 5s. For slow dependencies, increase:

   ```csharp
   var dh = DepHealthMonitor.CreateBuilder("my-service", "my-team")
       .WithCheckTimeout(TimeSpan.FromSeconds(10))
       .AddPostgres("slow-db",
           Environment.GetEnvironmentVariable("DATABASE_URL")!,
           critical: true)
       .Build();
   ```

5. **Debug logging** — enable SDK debug output via `appsettings.json`:

   ```json
   {
     "Logging": {
       "LogLevel": {
         "DepHealth": "Debug"
       }
     }
   }
   ```

   Or pass a logger directly:

   ```csharp
   using Microsoft.Extensions.Logging;

   var loggerFactory = LoggerFactory.Create(b => b.AddConsole().SetMinimumLevel(LogLevel.Debug));
   var logger = loggerFactory.CreateLogger("DepHealth");

   var dh = DepHealthMonitor.CreateBuilder("my-service", "my-team")
       .WithLogger(logger)
       // ...
       .Build();
   ```

## High Latency for PostgreSQL/MySQL Checks

**Symptom:** `app_dependency_latency_seconds` shows high values (100ms+)
for database checks.

**Cause:** Standalone mode creates a new connection for each check.
This includes TCP handshake, TLS negotiation, and authentication.

**Solution:** Use Entity Framework integration or pool mode with an existing
`NpgsqlDataSource`:

```csharp
// Instead of standalone mode
builder.Services.AddDepHealth("my-service", "my-team", dh => dh
    .AddPostgres("db-primary",
        "postgresql://user:pass@pg.svc:5432/mydb",
        critical: true)
);

// Use existing NpgsqlDataSource (preferred)
var dataSource = NpgsqlDataSource.Create(connectionString);
builder.Services.AddDepHealth("my-service", "my-team", dh => dh
    .AddCustom("db-primary", DependencyType.Postgres,
        host: "pg.svc", port: "5432",
        checker: new PostgresChecker(dataSource),
        critical: true)
);

// Or use Entity Framework integration
builder.Services.AddDepHealth("my-service", "my-team", dh => dh
    .AddNpgsqlFromContext("db-primary", dbContext, critical: true)
);
```

See [Connection Pools](connection-pools.md) for details.

## gRPC: DeadlineExceeded Error

**Symptom:** gRPC checks fail with timeout or show high latency.

**Possible causes:**

1. **gRPC service not accessible** at the specified address.

2. **Service does not implement** `grpc.health.v1.Health/Check` — the
   gRPC Health Checking Protocol must be enabled on the target service.

3. **Use `host` + `port`**, not `url` for gRPC:

   ```csharp
   builder.Services.AddDepHealth("my-service", "my-team", dh => dh
       .AddGrpc("user-service",
           host: "user.svc",
           port: "9090",
           critical: true)
   );
   ```

4. **TLS mismatch** — if the service uses TLS, set `tlsEnabled: true`:

   ```csharp
   .AddGrpc("user-service",
       host: "user.svc",
       port: "443",
       tlsEnabled: true,
       critical: true)
   ```

5. **DNS resolution in Kubernetes** — use FQDN with trailing dot to avoid
   search domain iteration:

   ```csharp
   .AddGrpc("user-service",
       host: "user-service.namespace.svc.cluster.local.",
       port: "9090",
       critical: true)
   ```

## Connection Refused Errors

**Symptom:** `app_dependency_status{status="connection_error"}` is `1`.

**Possible causes:**

1. **Service not running** — verify the target service is up and listening
   on the expected host and port.

2. **Wrong host or port** — double-check the URL or host/port values.

3. **Kubernetes network policies** — ensure traffic is allowed from the
   checker pod to the target service.

4. **Firewall rules** — on non-Kubernetes setups, check firewall rules.

## Timeout Errors

**Symptom:** `app_dependency_status{status="timeout"}` is `1`.

**Possible causes:**

1. **Default timeout too low.** Default is 5s. Increase globally or there
   is no per-dependency override in the current SDK version — use the
   global option:

   ```csharp
   // Global timeout for all dependencies
   var dh = DepHealthMonitor.CreateBuilder("my-service", "my-team")
       .WithCheckTimeout(TimeSpan.FromSeconds(10))
       // ...
       .Build();
   ```

2. **Network latency** — check round-trip time to the target service.

3. **Target overloaded** — the service may be too slow to respond.

## Unexpected Auth Errors

**Symptom:** `app_dependency_status{status="auth_error"}` is `1` when
credentials should be valid.

**Possible causes:**

1. **Credentials not set or incorrect** — verify token or username/password:

   ```csharp
   builder.AddHttp("payment-api",
       url: "https://payment.svc:8443",
       critical: true,
       bearerToken: Environment.GetEnvironmentVariable("API_TOKEN"))

   builder.AddGrpc("payments-grpc",
       host: "payments.svc",
       port: "9090",
       critical: true,
       bearerToken: Environment.GetEnvironmentVariable("GRPC_TOKEN"))
   ```

2. **Token expired** — bearer tokens have a limited lifetime.

3. **Wrong auth method** — some services expect Basic auth, not Bearer.

4. **Database credentials** — for PostgreSQL, MySQL, and AMQP, verify
   credentials are correct in the URL:

   ```csharp
   .AddPostgres("db",
       url: "postgresql://user:password@host:5432/dbname",
       critical: true)
   ```

See [Authentication](authentication.md) for all auth options.

## AMQP: Connection Error to RabbitMQ

**Symptom:** AMQP checker fails to connect.

**Important**: path `/` in URL means vhost `/` (not empty).

```csharp
builder.Services.AddDepHealth("my-service", "my-team", dh => dh
    .AddAmqp("rabbitmq",
        url: "amqp://rabbitmq.svc:5672/",
        critical: false)
);
```

For explicit credentials, embed them in the URL or use the `amqp://user:pass@host:port/vhost`
format:

```csharp
// Explicit credentials and vhost in URL
.AddAmqp("rabbitmq",
    url: "amqp://user:pass@rabbitmq.svc:5672/my-vhost",
    critical: false)
```

## LDAP: Configuration Errors

**Symptom:** LDAP checker throws `ConfigurationException` on startup.

**Common causes:**

1. **`SimpleBind` without credentials:**

   ```csharp
   // Wrong -- missing bindDN and bindPassword
   builder.AddLdap("ldap-corp",
       host: "ldap.svc",
       port: "389",
       checkMethod: LdapCheckMethod.SimpleBind,
       critical: false)

   // Correct
   builder.AddLdap("ldap-corp",
       host: "ldap.svc",
       port: "389",
       checkMethod: LdapCheckMethod.SimpleBind,
       bindDN: "cn=monitor,dc=corp,dc=com",
       bindPassword: "secret",
       critical: false)
   ```

2. **`Search` without baseDN:**

   ```csharp
   // Wrong -- missing baseDN
   builder.AddLdap("ldap-corp",
       host: "ldap.svc",
       port: "389",
       checkMethod: LdapCheckMethod.Search,
       critical: false)

   // Correct
   builder.AddLdap("ldap-corp",
       host: "ldap.svc",
       port: "389",
       checkMethod: LdapCheckMethod.Search,
       baseDN: "dc=example,dc=com",
       critical: false)
   ```

3. **startTLS with `ldaps://`** — these are incompatible:

   ```csharp
   // Wrong -- cannot use both
   builder.AddLdap("ldap-corp",
       host: "ldap.svc",
       port: "636",
       useTls: true,
       startTls: true,  // incompatible with useTls
       critical: false)

   // Correct -- use one or the other
   builder.AddLdap("ldap-corp",
       host: "ldap.svc",
       port: "636",
       useTls: true,    // implicit TLS (ldaps://)
       critical: false)
   // OR
   builder.AddLdap("ldap-corp",
       host: "ldap.svc",
       port: "389",
       startTls: true,  // upgrade to TLS via StartTLS
       critical: false)
   ```

## Custom Labels Not Appearing

**Symptom:** Labels added via the `labels` dictionary are not visible in
metrics.

**Possible causes:**

1. **Invalid label name.** Must match `[a-zA-Z_][a-zA-Z0-9_]*` and not be
   a reserved name.

   Reserved: `name`, `group`, `dependency`, `type`, `host`, `port`, `critical`.

2. **Inconsistent labels across dependencies.** If custom labels are used,
   all endpoints must use the same label names. The SDK collects all label
   names from all dependencies and applies them to all metrics.

## Health() Returns Empty Dictionary

**Symptom:** `monitor.Health()` returns an empty dictionary immediately
after `Start()`.

**Cause:** The first check has not completed yet. There is an initial delay
(default 0s, configurable via `WithInitialDelay`) before the first check
runs. Even with no initial delay, the first check completes asynchronously.

**Solution:** Use `HealthDetails()`. Before the first check completes, it
returns entries with `Healthy = null` and `Status = "unknown"`:

```csharp
var details = monitor.HealthDetails();
foreach (var (key, ep) in details)
{
    if (ep.Healthy is null)
    {
        Console.WriteLine($"{key}: not yet checked");
    }
    else
    {
        Console.WriteLine($"{key}: healthy={ep.Healthy}");
    }
}
```

## ASP.NET Core: Metrics Not at /metrics

**Check:**

1. Package `prometheus-net.AspNetCore` is installed
2. `app.MapMetrics()` is called after `app.Build()`
3. `AddDepHealth` is registered in `builder.Services`
4. The application is not filtering requests to `/metrics` via middleware

## See Also

- [Getting Started](getting-started.md) — installation and basic setup
- [Configuration](configuration.md) — all options, defaults, and validation rules
- [Checkers](checkers.md) — detailed guide for all 9 checkers
- [Metrics](metrics.md) — Prometheus metrics reference and PromQL examples
- [Authentication](authentication.md) — auth options for HTTP, gRPC, and databases
- [Connection Pools](connection-pools.md) — integration with NpgsqlDataSource and IConnectionMultiplexer
- [ASP.NET Core Integration](aspnetcore.md) — hosted service and health endpoints
- [Entity Framework Integration](entity-framework.md) — DbContext-based health checks
