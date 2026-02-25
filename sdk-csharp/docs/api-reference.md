*[Русская версия](api-reference.ru.md)*

# C# SDK: API Reference

## DepHealthMonitor

Main SDK class. Manages dependency health monitoring, metrics export, and
dynamic endpoint lifecycle.

### Factory Method

```csharp
DepHealthMonitor.CreateBuilder(string name, string group)
```

| Parameter | Type | Description |
| --- | --- | --- |
| `name` | `string` | Application name (or `DEPHEALTH_NAME` env var). Must match `[a-z][a-z0-9-]{0,62}` |
| `group` | `string` | Application group (or `DEPHEALTH_GROUP` env var). Same format as `name` |

### Builder Methods

#### Configuration

```csharp
builder.WithCheckInterval(TimeSpan interval)   // Default: 15s
builder.WithCheckTimeout(TimeSpan timeout)      // Default: 5s
builder.WithInitialDelay(TimeSpan delay)        // Default: 0s
builder.WithRegistry(CollectorRegistry registry)
builder.WithLogger(ILogger logger)
```

#### Adding Dependencies

```csharp
builder.AddHttp(name, url, healthPath: "/health", critical: null, labels: null,
    headers: null, bearerToken: null, basicAuthUsername: null, basicAuthPassword: null)

builder.AddGrpc(name, host, port, tlsEnabled: false, critical: null, labels: null,
    metadata: null, bearerToken: null, basicAuthUsername: null, basicAuthPassword: null)

builder.AddTcp(name, host, port, critical: null, labels: null)

builder.AddPostgres(name, url, critical: null, labels: null)

builder.AddMySql(name, url, critical: null, labels: null)

builder.AddRedis(name, url, critical: null, labels: null)

builder.AddAmqp(name, url, critical: null, labels: null)

builder.AddKafka(name, url, critical: null, labels: null)

builder.AddCustom(name, type, host, port, checker, critical: null, labels: null)
```

#### Build

```csharp
DepHealthMonitor Build()
```

### Lifecycle Methods

#### `Start()`

Start periodic health checks. Creates one `async Task` per endpoint.

#### `Stop()`

Stop all health check tasks.

#### `Dispose()`

Dispose resources (calls `Stop()` if needed).

### Query Methods

#### `Health() -> Dictionary<string, bool>`

Return current health status grouped by dependency name. A dependency is
healthy if at least one of its endpoints is healthy.

#### `HealthDetails() -> Dictionary<string, EndpointStatus>`

Return detailed status for every endpoint. Keys use `"dependency:host:port"`
format.

### Dynamic Endpoint Management

Added in v0.6.0. All methods require the scheduler to be started (via
`Start()`).

#### `AddEndpoint(depName, depType, critical, ep, checker) -> void`

Add a new monitored endpoint at runtime.

```csharp
public void AddEndpoint(
    string depName,
    DependencyType depType,
    bool critical,
    Endpoint ep,
    IHealthChecker checker)
```

| Parameter | Type | Description |
| --- | --- | --- |
| `depName` | `string` | Dependency name. Must match `[a-z][a-z0-9-]{0,62}` |
| `depType` | `DependencyType` | Dependency type (`Http`, `Postgres`, etc.) |
| `critical` | `bool` | Whether the dependency is critical |
| `ep` | `Endpoint` | Endpoint to monitor |
| `checker` | `IHealthChecker` | Health checker implementation |

**Idempotent:** returns silently if the endpoint already exists.

**Throws:**

- `ValidationException` — invalid `depName`, `depType`, or empty `host`/`port`
- `InvalidOperationException` — scheduler not started or already stopped

#### `RemoveEndpoint(depName, host, port) -> void`

Remove a monitored endpoint at runtime. Cancels the check task and deletes
all Prometheus metrics for the endpoint.

```csharp
public void RemoveEndpoint(
    string depName,
    string host,
    string port)
```

**Idempotent:** returns silently if the endpoint does not exist.

**Throws:** `InvalidOperationException` — scheduler not started.

#### `UpdateEndpoint(depName, oldHost, oldPort, newEp, checker) -> void`

Atomically replace an endpoint. Removes the old endpoint (cancels task,
deletes metrics) and adds the new one.

```csharp
public void UpdateEndpoint(
    string depName,
    string oldHost,
    string oldPort,
    Endpoint newEp,
    IHealthChecker checker)
```

**Throws:**

- `EndpointNotFoundException` — old endpoint does not exist
- `ValidationException` — invalid new endpoint (`host`/`port` empty, reserved labels)
- `InvalidOperationException` — scheduler not started or already stopped

---

## Types

### `Endpoint`

```csharp
public sealed class Endpoint
{
    public string Host { get; }
    public string Port { get; }
    public Dictionary<string, string> Labels { get; }
}
```

### `DependencyType`

Enum: `Http`, `Grpc`, `Tcp`, `Postgres`, `MySql`, `Redis`, `Amqp`, `Kafka`.

### `EndpointStatus`

```csharp
public sealed class EndpointStatus
{
    public string Dependency { get; }
    public string Type { get; }
    public string Host { get; }
    public string Port { get; }
    public bool? Healthy { get; }
    public string Status { get; }
    public string Detail { get; }
    public double Latency { get; }
    public DateTime? LastCheckedAt { get; }
    public bool Critical { get; }
    public Dictionary<string, string> Labels { get; }
}
```

Properties:

- `LatencyMillis` — latency in milliseconds
- JSON serialization uses `System.Text.Json` with snake_case naming

### `IHealthChecker`

```csharp
public interface IHealthChecker
{
    Task CheckAsync(Endpoint endpoint, CancellationToken cancellationToken);
}
```

---

## Exceptions

| Exception | Description |
| --- | --- |
| `DepHealthException` | Base class for check failures |
| `CheckTimeoutException` | Check timed out |
| `ConnectionRefusedException` | Connection refused |
| `DnsException` | DNS resolution failed |
| `TlsException` | TLS handshake failed |
| `AuthException` | Authentication/authorization failed |
| `UnhealthyException` | Endpoint reported unhealthy status |
| `ValidationException` | Input validation failed |
| `EndpointNotFoundException` | Dynamic update/remove target not found (v0.6.0) |
