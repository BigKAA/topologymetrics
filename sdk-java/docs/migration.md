*[Русская версия](migration.ru.md)*

# Migration Guide

Version upgrade instructions for the Java SDK.

## v0.6.0 to v0.8.0

### New: LDAP Health Checker

v0.8.0 adds a new LDAP health checker with full protocol support. No
existing API changes — this is a purely additive feature.

| Feature | Description |
| --- | --- |
| Check methods | `anonymous_bind`, `simple_bind`, `root_dse` (default), `search` |
| Protocols | `ldap://` (port 389), `ldaps://` (port 636), StartTLS |
| TLS options | `startTLS`, `tlsSkipVerify` |
| Pool mode | Accept existing `LDAPConnection` for health checks |

#### Basic Usage

```java
var dh = DepHealth.builder("my-service", "my-team", registry)
    .dependency("directory", DependencyType.LDAP, d -> d
        .url("ldap://ldap.svc:389")
        .critical(true))
    .build();
```

#### Check Methods

```java
// RootDSE query (default)
.dependency("ldap", DependencyType.LDAP, d -> d
    .url("ldap://ldap.svc:389")
    .critical(true))

// Simple bind with credentials
.dependency("ad", DependencyType.LDAP, d -> d
    .url("ldaps://ad.corp:636")
    .ldapCheckMethod("simple_bind")
    .ldapBindDN("cn=monitor,dc=corp,dc=com")
    .ldapBindPassword("secret")
    .critical(true))

// Search
.dependency("directory", DependencyType.LDAP, d -> d
    .url("ldap://ldap.svc:389")
    .ldapCheckMethod("search")
    .ldapBaseDN("dc=example,dc=com")
    .ldapSearchFilter("(objectClass=organizationalUnit)")
    .ldapSearchScope("one")
    .critical(true))
```

#### StartTLS

```java
.dependency("ldap-starttls", DependencyType.LDAP, d -> d
    .url("ldap://ldap.svc:389")
    .ldapStartTLS(true)
    .ldapTlsSkipVerify(true)
    .critical(true))
```

#### Pool Mode

```java
import com.unboundid.ldap.sdk.LDAPConnection;

LDAPConnection conn = ...; // existing connection

.dependency("directory", DependencyType.LDAP, d -> d
    .ldapConnection(conn)
    .ldapCheckMethod("root_dse")
    .critical(true))
```

#### Spring Boot

```yaml
dephealth:
  dependencies:
    directory:
      type: ldap
      url: ldap://ldap.svc:389
      critical: true
      ldap-check-method: root_dse
```

#### Error Classification

| Condition | Status | Detail |
| --- | --- | --- |
| Check succeeded | `ok` | `ok` |
| LDAP code 49 (Invalid Credentials) | `auth_error` | `auth_error` |
| LDAP code 50 (Insufficient Access Rights) | `auth_error` | `auth_error` |
| TLS/StartTLS failure | `tls_error` | `tls_error` |
| Connection refused | `connection_error` | `connection_refused` |
| DNS resolution failed | `dns_error` | `dns_error` |
| Timeout | `timeout` | `timeout` |
| Server down/busy/unavailable | `unhealthy` | `unhealthy` |

#### Configuration Validation

| Condition | Error |
| --- | --- |
| `simple_bind` without `bindDN` + `bindPassword` | Config error |
| `search` without `baseDN` | Config error |
| `startTLS=true` with `ldaps://` | Config error (incompatible) |

#### Version Update

```xml
<!-- v0.6.0 -->
<version>0.6.0</version>

<!-- v0.8.0 -->
<version>0.8.0</version>
```

---

## v0.5.0 to v0.6.0

### New: Dynamic Endpoint Management

v0.6.0 adds three methods for managing endpoints at runtime. No existing API
changes — this is a purely additive feature.

| Method | Description |
| --- | --- |
| `addEndpoint` | Add a monitored endpoint after `start()` |
| `removeEndpoint` | Remove an endpoint (cancels scheduled task, deletes metrics) |
| `updateEndpoint` | Atomically replace an endpoint with a new one |

A new exception `EndpointNotFoundException` (extends `DepHealthException`)
is thrown by `updateEndpoint` when the old endpoint does not exist.

```java
// After dh.start()...

// Add a new endpoint
dh.addEndpoint("api-backend", DependencyType.HTTP, true,
    new Endpoint("backend-2.svc", "8080"),
    HttpHealthChecker.builder().build());

// Remove an endpoint (idempotent)
dh.removeEndpoint("api-backend", "backend-2.svc", "8080");

// Replace an endpoint atomically
dh.updateEndpoint("api-backend", "backend-1.svc", "8080",
    new Endpoint("backend-3.svc", "8080"),
    HttpHealthChecker.builder().build());
```

#### Key Behaviors

- **Thread-safe** — all three methods are synchronized.
- **Idempotent** — `addEndpoint` is no-op if endpoint exists;
  `removeEndpoint` is no-op if endpoint is not found.
- Dynamic endpoints inherit the global check interval and timeout.
- `removeEndpoint` / `updateEndpoint` delete all Prometheus metrics for
  the old endpoint.
- `updateEndpoint` throws `EndpointNotFoundException` if the old endpoint
  does not exist.

#### Validation

`addEndpoint` and `updateEndpoint` validate inputs:

- `depName` must match `[a-z][a-z0-9-]*`, max 63 chars
- `depType` must not be null
- `ep.host()` and `ep.port()` must be non-empty
- `ep.labels()` must not use reserved label names

Invalid inputs throw `ValidationException`.

#### Error Handling

```java
try {
    dh.updateEndpoint("api", "old-host", "8080", newEp, checker);
} catch (EndpointNotFoundException e) {
    // old endpoint does not exist — use addEndpoint instead
} catch (IllegalStateException e) {
    // scheduler not started or already stopped
}
```

#### Internal Changes

- `CheckScheduler` stores per-endpoint `ScheduledFuture` for cancellation.
- `states` map changed from `HashMap` to `ConcurrentHashMap`.
- `ScheduledThreadPoolExecutor` replaces fixed-size `ScheduledExecutorService`.
- `MetricsExporter.deleteMetrics()` removes all 4 metric families for an endpoint.

#### Version Update

```xml
<!-- v0.5.0 -->
<version>0.5.0</version>

<!-- v0.6.0 -->
<version>0.6.0</version>
```

---

## v0.4.x to v0.5.0

### Breaking: Mandatory `group` Parameter

v0.5.0 adds a mandatory `group` parameter (logical grouping: team, subsystem,
project).

Programmatic API:

```java
// v0.4.x
DepHealth dh = DepHealth.builder("my-service", meterRegistry)
    .dependency(...)
    .build();

// v0.5.0
DepHealth dh = DepHealth.builder("my-service", "my-team", meterRegistry)
    .dependency(...)
    .build();
```

Spring Boot YAML:

```yaml
# v0.5.0 — add group
dephealth:
  name: my-service
  group: my-team
  dependencies: ...
```

Alternative: set `DEPHEALTH_GROUP` environment variable (API takes precedence).

Validation: same rules as `name` — `[a-z][a-z0-9-]*`, 1-63 chars.

---

## v0.4.0 to v0.4.1

### New: healthDetails() API

v0.4.1 adds the `healthDetails()` method that returns detailed status for each
endpoint. No existing API changes — purely additive.

```java
Map<String, EndpointStatus> details = dh.healthDetails();

for (var entry : details.entrySet()) {
    EndpointStatus ep = entry.getValue();
    System.out.printf("%s: healthy=%s status=%s detail=%s latency=%s%n",
        entry.getKey(), ep.isHealthy(), ep.getStatus(), ep.getDetail(),
        ep.getLatencyMillis());
}
```

`EndpointStatus` fields: `getName()`, `getType()`, `getHost()`, `getPort()`,
`isHealthy()` (`Boolean`, `null` = unknown), `getStatus()`, `getDetail()`,
`getLatency()`, `getLastCheckedAt()`, `isCritical()`, `getLabels()`.

Before the first check, `isHealthy()` is `null` and `getStatus()` is `"unknown"`.

---

## v0.3.x to v0.4.0

### New Status Metrics (No Code Changes Required)

v0.4.0 adds two new automatically exported Prometheus metrics:

| Metric | Type | Description |
| --- | --- | --- |
| `app_dependency_status` | Gauge (enum) | Status category: 8 series per endpoint, exactly one = 1 |
| `app_dependency_status_detail` | Gauge (info) | Detailed failure reason: e.g. `http_503`, `auth_error` |

**No code changes needed** — the SDK exports these metrics automatically
alongside the existing `app_dependency_health` and
`app_dependency_latency_seconds`.

#### Storage Impact

Each endpoint now produces 9 additional time series (8 for
`app_dependency_status` + 1 for `app_dependency_status_detail`). For a
service with 5 endpoints, this adds 45 series.

#### New PromQL Queries

```promql
# Status category for a dependency
app_dependency_status{dependency="postgres-main", status!=""} == 1

# Detailed failure reason
app_dependency_status_detail{dependency="postgres-main", detail!=""} == 1

# Alert on authentication errors
app_dependency_status{status="auth_error"} == 1
```

---

## v0.1 to v0.2

### API Changes

| v0.1 | v0.2 | Description |
| --- | --- | --- |
| `DepHealth.builder(registry)` | `DepHealth.builder("my-service", registry)` | Required first argument `name` |
| `.critical(true)` (optional) | `.critical(true/false)` (required) | For each dependency |
| none | `.label("key", "value")` | Custom labels |
| `dephealth.name` (none) | `dephealth.name: my-service` | In application.yml |

### Required Changes

1. Add `name` to builder:

```java
// v0.1
DepHealth dh = DepHealth.builder(meterRegistry)
    .dependency("postgres-main", DependencyType.POSTGRES, d -> d
        .url("postgres://user:pass@pg.svc:5432/mydb")
        .critical(true))
    .build();

// v0.2
DepHealth dh = DepHealth.builder("my-service", meterRegistry)
    .dependency("postgres-main", DependencyType.POSTGRES, d -> d
        .url("postgres://user:pass@pg.svc:5432/mydb")
        .critical(true))
    .build();
```

1. Specify `.critical()` for each dependency:

```java
// v0.1 — critical is optional
.dependency("redis-cache", DependencyType.REDIS, d -> d
    .url("redis://redis.svc:6379"))

// v0.2 — critical is required
.dependency("redis-cache", DependencyType.REDIS, d -> d
    .url("redis://redis.svc:6379")
    .critical(false))
```

1. Update `application.yml` (Spring Boot):

```yaml
# v0.2
dephealth:
  name: my-service
  dependencies:
    redis-cache:
      type: redis
      url: ${REDIS_URL}
      critical: false
```

1. Update dependency version:

```xml
<version>0.2.2</version>
```

### New Labels in Metrics

```text
# v0.1
app_dependency_health{dependency="postgres-main",type="postgres",host="pg.svc",port="5432"} 1

# v0.2
app_dependency_health{name="my-service",dependency="postgres-main",type="postgres",host="pg.svc",port="5432",critical="yes"} 1
```

Update PromQL queries and Grafana dashboards by adding `name` and `critical` labels.

## See Also

- [Getting Started](getting-started.md) — installation and setup
- [Checkers](checkers.md) — LDAP checker details
- [Configuration](configuration.md) — all options and defaults
- [API Reference](api-reference.md) — complete public API
