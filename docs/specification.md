*[Русская версия](specification.ru.md)*

# dephealth Specification Overview

The dephealth specification is the single source of truth for all SDKs.
It defines metric format, check behavior, and connection configuration.
All SDKs must strictly comply with these contracts.

Full specification documents are located in the [`spec/`](../spec/) directory.

## Metric Contract

> Full document: [`spec/metric-contract.md`](../spec/metric-contract.md)

### Health Metric

```text
app_dependency_health{name="my-service",group="billing-team",dependency="postgres-main",type="postgres",host="pg.svc",port="5432",critical="yes"} 1
```

| Property | Value |
| --- | --- |
| Name | `app_dependency_health` |
| Type | Gauge |
| Values | `1` (available), `0` (unavailable) |
| Required labels | `name`, `group`, `dependency`, `type`, `host`, `port`, `critical` |
| Optional labels | arbitrary via `WithLabel(key, value)` |

### Latency Metric

```text
app_dependency_latency_seconds_bucket{name="my-service",group="billing-team",dependency="postgres-main",type="postgres",host="pg.svc",port="5432",critical="yes",le="0.01"} 42
```

| Property | Value |
| --- | --- |
| Name | `app_dependency_latency_seconds` |
| Type | Histogram |
| Buckets | `0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0, 5.0` |
| Labels | Identical to `app_dependency_health` |

### Status Metric

```text
app_dependency_status{name="my-service",group="billing-team",dependency="postgres-main",type="postgres",host="pg.svc",port="5432",critical="yes",status="ok"} 1
```

| Property | Value |
| --- | --- |
| Name | `app_dependency_status` |
| Type | Gauge (enum pattern) |
| Values | `1` (active status), `0` (inactive status) |
| Status values | `ok`, `timeout`, `connection_error`, `dns_error`, `auth_error`, `tls_error`, `unhealthy`, `error` |
| Labels | Same as `app_dependency_health` + `status` |

All 8 status series are always exported per endpoint. Exactly one = 1, the rest = 0.
No series churn on state changes.

### Status Detail Metric

```text
app_dependency_status_detail{name="my-service",group="billing-team",dependency="postgres-main",type="postgres",host="pg.svc",port="5432",critical="yes",detail="ok"} 1
```

| Property | Value |
| --- | --- |
| Name | `app_dependency_status_detail` |
| Type | Gauge (info pattern) |
| Values | Always `1` |
| Detail values | Checker-specific: `ok`, `timeout`, `connection_refused`, `dns_error`, `http_503`, `grpc_not_serving`, `auth_error`, etc. |
| Labels | Same as `app_dependency_health` + `detail` |

One series per endpoint. When the detail changes, the old series is deleted
and a new one is created (acceptable series churn).

### Label Formation Rules

- `name` — unique application name (format `[a-z][a-z0-9-]*`, 1-63 characters)
- `group` — logical group (format `[a-z][a-z0-9-]*`, 1-63 characters, e.g. `billing-team`)
- `dependency` — logical name (e.g. `postgres-main`, `redis-cache`)
- `type` — dependency type: `http`, `grpc`, `tcp`, `postgres`, `mysql`,
  `redis`, `amqp`, `kafka`
- `host` — DNS name or IP address of the endpoint
- `port` — endpoint port
- `critical` — dependency criticality: `yes` or `no`

Label order: `name`, `group`, `dependency`, `type`, `host`, `port`, `critical`,
then arbitrary labels in alphabetical order.

When a single dependency has multiple endpoints (e.g. primary + replica),
a separate metric is created for each endpoint.

### Custom Labels

Custom labels are added via `WithLabel(key, value)` (Go),
`.label(key, value)` (Java), `labels={"key": "value"}` (Python),
`.Label(key, value)` (C#).

Label names: format `[a-zA-Z_][a-zA-Z0-9_]*`, required labels
(`name`, `group`, `dependency`, `type`, `host`, `port`, `critical`) cannot be overridden.

## Behavior Contract

> Full document: [`spec/check-behavior.md`](../spec/check-behavior.md)

### Check Lifecycle

```text
Initialization → initialDelay → First check → Periodic checks (every checkInterval)
                                                          ↓
                                                   Graceful Shutdown
```

### Default Parameters

| Parameter | Value | Description |
| --- | --- | --- |
| `checkInterval` | 15s | Interval between checks |
| `timeout` | 5s | Single check timeout |
| `initialDelay` | 5s | Delay before the first check |
| `failureThreshold` | 1 | Consecutive failures to transition to unhealthy |
| `successThreshold` | 1 | Consecutive successes to transition to healthy |

### Threshold Logic

- **healthy -> unhealthy**: after `failureThreshold` consecutive failures
- **unhealthy -> healthy**: after `successThreshold` consecutive successes
- **Initial state**: unknown until the first check

### Check Types

| Type | Method | Success Criteria |
| --- | --- | --- |
| `http` | HTTP GET to `healthPath` | 2xx status |
| `grpc` | gRPC Health Check Protocol | `SERVING` |
| `tcp` | TCP connection establishment | Connection established |
| `postgres` | `SELECT 1` | Query executed |
| `mysql` | `SELECT 1` | Query executed |
| `redis` | `PING` | `PONG` response |
| `amqp` | Open/close connection | Connection established |
| `kafka` | Metadata request | Response received |

### Two Operating Modes

- **Standalone**: SDK creates a temporary connection for
  each check. Simple to configure but creates additional load.
- **Connection pool integration**: SDK uses the service's existing pool.
  Reflects the service's actual ability to work with the dependency.
  Recommended for databases and caches.

### Error Handling

Any of the following situations is considered a failed check:

- Timeout (`context deadline exceeded`)
- DNS resolution failure
- Connection refused
- TLS handshake failure
- Unexpected response (non-2xx for HTTP, non-`SERVING` for gRPC)

## Configuration Contract

> Full document: [`spec/config-contract.md`](../spec/config-contract.md)

### Connection Input Formats

| Format | Example |
| --- | --- |
| URL | `postgres://user:pass@host:5432/db` |
| Direct parameters | `host` + `port` |
| Connection string | `Host=host;Port=5432;Database=db` |
| JDBC URL | `jdbc:postgresql://host:5432/db` |

### Auto-detection of Type

Dependency type is determined from the URL scheme:

| Scheme | Type |
| --- | --- |
| `postgres://`, `postgresql://` | `postgres` |
| `mysql://` | `mysql` |
| `redis://`, `rediss://` | `redis` |
| `amqp://`, `amqps://` | `amqp` |
| `http://`, `https://` | `http` |
| `grpc://` | `grpc` |
| `kafka://` | `kafka` |

### Default Ports

| Type | Port |
| --- | --- |
| `postgres` | 5432 |
| `mysql` | 3306 |
| `redis` | 6379 |
| `amqp` | 5672 |
| `http` | 80 / 443 (HTTPS) |
| `grpc` | 443 |
| `kafka` | 9092 |
| `tcp` | (required) |

### Allowed Parameter Ranges

| Parameter | Minimum | Maximum |
| --- | --- | --- |
| `checkInterval` | 1s | 10m |
| `timeout` | 100ms | 30s |
| `initialDelay` | 0 | 5m |
| `failureThreshold` | 1 | 10 |
| `successThreshold` | 1 | 10 |

Additional constraint: `timeout` must be less than `checkInterval`.

### Environment Variables

| Variable | Description |
| --- | --- |
| `DEPHEALTH_NAME` | Application name (overridden by API) |
| `DEPHEALTH_GROUP` | Logical group (overridden by API) |
| `DEPHEALTH_<DEP>_CRITICAL` | Dependency criticality: `yes`/`no` |
| `DEPHEALTH_<DEP>_LABEL_<KEY>` | Custom label for a dependency |

`<DEP>` — dependency name in uppercase, hyphens replaced with `_`.

### DNS Resolution in Kubernetes

In standalone mode, dephealth creates a new connection for each health check,
which triggers DNS resolution every time. In Kubernetes, the default
`/etc/resolv.conf` is configured with `ndots:5` and multiple search domains
(e.g. `<ns>.svc.cluster.local svc.cluster.local cluster.local`).

When a hostname contains fewer dots than the `ndots` value, the resolver
prepends search domain suffixes before trying the name as-is. For example,
a name like `redis.my-namespace.svc` (2 dots < 5) will generate several
failed DNS queries before the correct one succeeds:

```text
redis.my-namespace.svc.app-namespace.svc.cluster.local  → NXDOMAIN
redis.my-namespace.svc.svc.cluster.local                → NXDOMAIN
redis.my-namespace.svc.cluster.local                    → OK
```

To avoid this overhead, use a **trailing dot** to mark the hostname as an
absolute (fully qualified) domain name. This tells the resolver to skip
search domain expansion entirely and issue a single DNS query:

```yaml
# Relative name — triggers search domain expansion (multiple DNS queries)
host: "redis.my-namespace.svc"

# Absolute name (FQDN) — single DNS query, no search expansion
host: "redis.my-namespace.svc.cluster.local."
```

> **Note:** The trailing dot (`.`) is part of the DNS standard (RFC 1035)
> and is supported by all DNS resolvers. The cluster domain (`cluster.local`
> by default) may differ in your environment — check `/etc/resolv.conf`
> inside a pod for the actual value.

This optimization applies to all dependency types and is especially
noticeable for check types with higher connection overhead (gRPC, TLS).

## Programmatic Health Details API

> Full document: [`spec/check-behavior.md` § 8](../spec/check-behavior.md)

The `HealthDetails()` method returns an `EndpointStatus` for each monitored
endpoint with 11 fields:

| Field | Type | Description |
| --- | --- | --- |
| `dependency` | string | Logical dependency name |
| `type` | string | Dependency type (`http`, `postgres`, etc.) |
| `host` | string | Endpoint host |
| `port` | string | Endpoint port |
| `healthy` | bool/null | `true`/`false`/`null` (unknown before first check) |
| `status` | string | Status category: `ok`, `timeout`, `connection_error`, etc. |
| `detail` | string | Detailed reason: `ok`, `http_503`, `auth_error`, etc. |
| `latency` | duration | Last check latency |
| `last_checked_at` | timestamp | Time of last check (null if never checked) |
| `critical` | bool | Dependency criticality |
| `labels` | map | Custom labels |

Key format: `"dependency:host:port"` (same as `Health()`).

Language-specific methods:

- Go: `dh.HealthDetails()` → `map[string]EndpointStatus`
- Java: `depHealth.healthDetails()` → `Map<String, EndpointStatus>`
- Python: `dh.health_details()` → `dict[str, EndpointStatus]`
- C#: `depHealth.HealthDetails()` → `Dictionary<string, EndpointStatus>`

## Conformance Testing

All SDKs pass a unified set of conformance scenarios in Kubernetes:

| Scenario | Verifies |
| --- | --- |
| `basic-health` | All dependencies available -> metrics = 1 |
| `partial-failure` | Partial failure -> correct values |
| `full-failure` | Full dependency failure -> metric = 0 |
| `recovery` | Recovery -> metric returns to 1 |
| `latency` | Histogram buckets present |
| `labels` | Correctness of all labels (name, critical, custom labels) |
| `timeout` | Delay > timeout -> unhealthy |
| `initial-state` | Initial state is correct |
| `health-details` | HealthDetails() returns correct endpoint data |

More details: [`conformance/`](../conformance/)

## Links

- [Go SDK Quick Start](quickstart/go.md)
- [Java SDK Quick Start](quickstart/java.md)
- [Python SDK Quick Start](quickstart/python.md)
- [C# SDK Quick Start](quickstart/csharp.md)
- [Go SDK Integration Guide](migration/go.md)
- [Java SDK Integration Guide](migration/java.md)
- [Python SDK Integration Guide](migration/python.md)
- [C# SDK Integration Guide](migration/csharp.md)
