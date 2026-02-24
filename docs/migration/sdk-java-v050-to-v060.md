*[Русская версия](sdk-java-v050-to-v060.ru.md)*

# Java SDK: Migration from v0.5.0 to v0.6.0

Migration guide for the Java SDK v0.6.0 release.

> This release affects **Java SDK only**. Go SDK remains at v0.7.0;
> Python and C# SDKs remain at v0.5.0.

## What Changed

Three new methods on `DepHealth` allow dynamic endpoint management at runtime:

| Method | Description |
| --- | --- |
| `addEndpoint` | Add a monitored endpoint after `start()` |
| `removeEndpoint` | Remove an endpoint (cancels scheduled task, deletes metrics) |
| `updateEndpoint` | Atomically replace an endpoint with a new one |

A new exception `EndpointNotFoundException` (extends `DepHealthException`)
is thrown by `updateEndpoint` when the old endpoint does not exist.

---

## Do I Need to Change My Code?

**No.** This is a fully backward-compatible release. All existing code
continues to work without modification.

---

## New Feature: Dynamic Endpoints

Prior to v0.6.0, all dependencies had to be registered upfront via
`.dependency()` calls on the builder. Once `build()` was called, the
set of monitored endpoints was frozen.

Starting with v0.6.0, you can add, remove, and update endpoints on a
running `DepHealth` instance:

```java
import biz.kryukov.dev.dephealth.*;

// After depHealth.start()...

// Add a new endpoint
depHealth.addEndpoint("api-backend", DependencyType.HTTP, true,
    new Endpoint("backend-2.svc", "8080"),
    new HttpHealthChecker());

// Remove an endpoint
depHealth.removeEndpoint("api-backend", "backend-2.svc", "8080");

// Replace an endpoint atomically
depHealth.updateEndpoint("api-backend", "backend-1.svc", "8080",
    new Endpoint("backend-3.svc", "8080"),
    new HttpHealthChecker());
```

### Key Behaviors

- **Thread-safe:** all three methods are synchronized and can be called
  from multiple threads.
- **Idempotent:** `addEndpoint` returns silently if the endpoint already
  exists. `removeEndpoint` returns silently if the endpoint is not found.
- **Global config inheritance:** dynamically added endpoints use the
  global check interval and timeout configured in the builder.
- **Metrics lifecycle:** `removeEndpoint` and `updateEndpoint` delete
  all Prometheus metrics for the old endpoint (health, latency, status,
  status\_detail).

### Validation

`addEndpoint` and `updateEndpoint` validate inputs before proceeding:

- `depName` must match `[a-z][a-z0-9-]*`, max 63 chars
- `depType` must not be null
- `ep.host()` and `ep.port()` must be non-empty
- `ep.labels()` must not use reserved label names

Invalid inputs throw `ValidationException`.

### Error Handling

```java
try {
    depHealth.updateEndpoint("api", "old-host", "8080", newEp, checker);
} catch (EndpointNotFoundException e) {
    // old endpoint does not exist — use addEndpoint instead
} catch (IllegalStateException e) {
    // scheduler not started or already stopped
}
```

---

## Internal Changes

- `CheckScheduler` now stores per-endpoint `ScheduledFuture` for
  cancellation support.
- `states` map changed from `HashMap` to `ConcurrentHashMap` for
  safe concurrent iteration in `health()` / `healthDetails()`.
- `ScheduledThreadPoolExecutor` replaces fixed-size
  `ScheduledExecutorService` (supports dynamic resizing).
- `MetricsExporter.deleteMetrics()` removes all 4 metric families
  for a given endpoint.

---

## Version Update

```xml
<!-- v0.5.0 -->
<version>0.5.0</version>

<!-- v0.6.0 -->
<version>0.6.0</version>
```
