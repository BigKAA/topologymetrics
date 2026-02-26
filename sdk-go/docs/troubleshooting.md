*[Русская версия](troubleshooting.ru.md)*

# Troubleshooting

Common issues and solutions when using the dephealth Go SDK.

## Empty Metrics / No Metrics Exported

**Symptom:** The `/metrics` endpoint returns no `app_dependency_*` metrics.

**Possible causes:**

1. **Missing checker import.** Checker factories must be registered via a
   blank import before calling `dephealth.New()`:

   ```go
   import _ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks"
   ```

   Or import individual sub-packages:

   ```go
   import (
       _ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/httpcheck"
       _ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/pgcheck"
   )
   ```

2. **`Start()` not called.** Metrics are only registered and updated after
   `dh.Start(ctx)` is called. Check that `Start()` is invoked and returns
   no error.

3. **Wrong Prometheus handler.** Make sure you expose `promhttp.Handler()`
   on the path your scraper expects (typically `/metrics`):

   ```go
   http.Handle("/metrics", promhttp.Handler())
   ```

   If you use a custom `prometheus.Registry` via `WithRegisterer()`, you must
   use `promhttp.HandlerFor(registry, promhttp.HandlerOpts{})` instead of
   `promhttp.Handler()`.

## No Checker Factory Registered

**Symptom:** `dephealth.New()` returns error
`no checker factory registered for type "http"` (or another type).

**Cause:** The sub-package for the requested checker type is not imported.

**Solution:** Add the appropriate blank import. For example, if you use
`dephealth.HTTP()`:

```go
import _ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/httpcheck"
```

See [Selective Imports](selective-imports.md) for the full list of
sub-packages and import paths.

## High gRPC Latency (100-500ms)

**Symptom:** gRPC health checks show 100-500ms latency even when the target
service is on the same network.

**Cause:** The default gRPC DNS resolver performs SRV + A lookups. In
Kubernetes with `ndots:5`, each lookup iterates through all search domains
before falling back to the actual FQDN.

**Solution:** Use a fully qualified domain name (FQDN) with a trailing dot:

```go
dephealth.GRPC("user-service",
    dephealth.FromParams("user-service.namespace.svc.cluster.local.", "9090"),
    dephealth.Critical(true),
)
```

The trailing dot tells the resolver to skip the search domain iteration.
The SDK already uses `passthrough:///` resolver internally, but the DNS
lookup still happens at connection time.

See the [DNS Resolution in Kubernetes](../../docs/specification.md#dns-resolution-in-kubernetes)
section in the specification for details.

## Connection Refused Errors

**Symptom:** `app_dependency_status{status="connection_error"}` is `1` and
the detail label shows `connection_refused`.

**Possible causes:**

1. **Service not running** — verify the target service is up and listening
   on the expected host and port.

2. **Wrong host or port** — double-check the URL or parameters passed to
   `FromURL()` / `FromParams()`.

3. **Kubernetes network policies** — if running in Kubernetes, ensure network
   policies allow traffic from the checker pod to the target service.

4. **Firewall rules** — on non-Kubernetes setups, check firewall rules
   between the checker and the target.

## Timeout Errors

**Symptom:** `app_dependency_status{status="timeout"}` is `1`.

**Possible causes:**

1. **Default timeout too low.** The default timeout is 5 seconds. For slow
   dependencies (e.g., cold database connections, cross-region services),
   increase it:

   ```go
   // Global: apply to all dependencies
   dephealth.WithTimeout(10 * time.Second)

   // Per-dependency: override for a specific dependency
   dephealth.Postgres("slow-db",
       dephealth.FromURL("postgresql://db.svc:5432/mydb"),
       dephealth.Critical(true),
       dephealth.Timeout(10 * time.Second),
   )
   ```

2. **Network latency** — check round-trip time to the target service. Use
   `app_dependency_latency_seconds` histogram to track actual check times.

3. **Target overloaded** — the target service may be too slow to respond
   within the timeout window.

## Unexpected Auth Errors

**Symptom:** `app_dependency_status{status="auth_error"}` is `1` when
credentials should be valid.

**Possible causes:**

1. **Credentials not passed or incorrect** — verify the token or
   username/password are set correctly:

   ```go
   // HTTP: check token value
   dephealth.WithHTTPBearerToken(os.Getenv("API_TOKEN"))

   // gRPC: check token value
   dephealth.WithGRPCBearerToken(os.Getenv("GRPC_TOKEN"))
   ```

2. **Token expired** — bearer tokens have a limited lifetime. If your token
   expires between restarts, ensure the environment variable or source is
   refreshed.

3. **Wrong auth method** — some services expect Basic auth, not Bearer, or
   vice versa. Check the target service documentation.

4. **Database credentials in URL** — for PostgreSQL, MySQL, and AMQP,
   credentials are part of the connection URL:

   ```go
   dephealth.Postgres("db",
       dephealth.FromURL("postgresql://user:password@host:5432/dbname"),
       dephealth.Critical(true),
   )
   ```

See [Authentication](authentication.md) for details on auth options per
checker type.

## Duplicate Metric Registration

**Symptom:** Panic at startup:
`duplicate metrics collector registration attempted`.

**Cause:** Two `DepHealth` instances are registering metrics with the same
Prometheus registerer (typically `prometheus.DefaultRegisterer`).

**Solution:** Use separate registerers for each instance:

```go
reg1 := prometheus.NewRegistry()
reg2 := prometheus.NewRegistry()

dh1, _ := dephealth.New("service-a", "team-a",
    dephealth.WithRegisterer(reg1),
    // ...
)

dh2, _ := dephealth.New("service-b", "team-b",
    dephealth.WithRegisterer(reg2),
    // ...
)
```

In practice, most applications need only one `DepHealth` instance.

## Custom Labels Not Appearing

**Symptom:** Labels added via `WithLabel()` are not visible in metrics.

**Possible causes:**

1. **Invalid label name.** Label names must match `[a-z_][a-z0-9_]*` and
   must not be a reserved name.

   Reserved label names: `name`, `group`, `dependency`, `type`, `host`,
   `port`, `critical`.

   ```go
   // Valid
   dephealth.WithLabel("region", "eu-west")

   // Invalid — uppercase
   dephealth.WithLabel("Region", "eu-west")

   // Invalid — reserved name
   dephealth.WithLabel("type", "primary")
   ```

2. **Inconsistent labels across dependencies.** All dependencies within a
   `DepHealth` instance must use the same set of custom label names. If one
   dependency has `WithLabel("env", "prod")` and another does not, validation
   will fail.

## Health() Returns Empty Map

**Symptom:** `dh.Health()` returns an empty map immediately after `Start()`.

**Cause:** The first health check has not completed yet. There is an initial
delay equal to the check interval (default 15 seconds) before the first
check runs.

**Solution:** Use `HealthDetails()` instead. Before the first check
completes, `HealthDetails()` returns entries with `Healthy: nil` and
`Status: "unknown"`, so you can distinguish between "not yet checked" and
"unhealthy":

```go
details := dh.HealthDetails()
for key, ep := range details {
    if ep.Healthy == nil {
        fmt.Printf("%s: not yet checked\n", key)
    } else {
        fmt.Printf("%s: healthy=%v\n", key, *ep.Healthy)
    }
}
```

## Debug Logging

To enable debug output from the SDK, pass a `*slog.Logger` via
`WithLogger()`:

```go
logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
    Level: slog.LevelDebug,
}))

dh, err := dephealth.New("my-service", "my-team",
    dephealth.WithLogger(logger),
    // ...
)
```

Log messages include:

- Dependency registration details
- Check results (success/failure, latency, status category)
- Connection errors with full error messages
- Metric registration events

## See Also

- [Getting Started](getting-started.md) — installation and basic setup
- [Configuration](configuration.md) — all options, defaults, and validation rules
- [Checkers](checkers.md) — detailed guide for all 8 checkers
- [Metrics](metrics.md) — Prometheus metrics reference and PromQL examples
- [Authentication](authentication.md) — auth options for HTTP, gRPC, and databases
- [Selective Imports](selective-imports.md) — import optimization
- [Examples](examples/) — runnable code examples
