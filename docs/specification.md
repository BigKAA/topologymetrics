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
app_dependency_health{name="my-service",dependency="postgres-main",type="postgres",host="pg.svc",port="5432",critical="yes"} 1
```

| Property | Value |
| --- | --- |
| Name | `app_dependency_health` |
| Type | Gauge |
| Values | `1` (available), `0` (unavailable) |
| Required labels | `name`, `dependency`, `type`, `host`, `port`, `critical` |
| Optional labels | arbitrary via `WithLabel(key, value)` |

### Latency Metric

```text
app_dependency_latency_seconds_bucket{name="my-service",dependency="postgres-main",type="postgres",host="pg.svc",port="5432",critical="yes",le="0.01"} 42
```

| Property | Value |
| --- | --- |
| Name | `app_dependency_latency_seconds` |
| Type | Histogram |
| Buckets | `0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0, 5.0` |
| Labels | Identical to `app_dependency_health` |

### Label Formation Rules

- `name` — unique application name (format `[a-z][a-z0-9-]*`, 1-63 characters)
- `dependency` — logical name (e.g. `postgres-main`, `redis-cache`)
- `type` — dependency type: `http`, `grpc`, `tcp`, `postgres`, `mysql`,
  `redis`, `amqp`, `kafka`
- `host` — DNS name or IP address of the endpoint
- `port` — endpoint port
- `critical` — dependency criticality: `yes` or `no`

Label order: `name`, `dependency`, `type`, `host`, `port`, `critical`,
then arbitrary labels in alphabetical order.

When a single dependency has multiple endpoints (e.g. primary + replica),
a separate metric is created for each endpoint.

### Custom Labels

Custom labels are added via `WithLabel(key, value)` (Go),
`.label(key, value)` (Java), `labels={"key": "value"}` (Python),
`.Label(key, value)` (C#).

Label names: format `[a-zA-Z_][a-zA-Z0-9_]*`, required labels
(`name`, `dependency`, `type`, `host`, `port`, `critical`) cannot be overridden.

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
| `DEPHEALTH_<DEP>_CRITICAL` | Dependency criticality: `yes`/`no` |
| `DEPHEALTH_<DEP>_LABEL_<KEY>` | Custom label for a dependency |

`<DEP>` — dependency name in uppercase, hyphens replaced with `_`.

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
