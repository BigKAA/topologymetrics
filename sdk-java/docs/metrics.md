*[Русская версия](metrics.ru.md)*

# Prometheus Metrics

The dephealth SDK exports four Prometheus metrics for each monitored
dependency endpoint via Micrometer. This guide describes each metric,
its labels, and provides PromQL examples.

## Metrics Overview

| Metric | Type | Description |
| --- | --- | --- |
| `app_dependency_health` | Gauge | Health status: `1` = healthy, `0` = unhealthy |
| `app_dependency_latency_seconds` | DistributionSummary | Check latency in seconds |
| `app_dependency_status` | Gauge (enum) | Status category: 8 series per endpoint |
| `app_dependency_status_detail` | Gauge (info) | Detailed failure reason |

## Labels

All four metrics share a common set of labels:

| Label | Source | Description |
| --- | --- | --- |
| `name` | Builder first arg | Application name |
| `group` | Builder second arg | Logical group |
| `dependency` | Dependency name | Dependency identifier |
| `type` | Checker type | `http`, `grpc`, `tcp`, `postgres`, `mysql`, `redis`, `amqp`, `kafka`, `ldap` |
| `host` | `url()`/`host()` | Dependency host |
| `port` | `url()`/`port()` | Dependency port |
| `critical` | `critical()` | `yes` or `no` |

Custom labels added via `.label()` appear after `critical` in
alphabetical order.

Additional labels per metric:

- `app_dependency_status` has `status` — one of 8 status categories
- `app_dependency_status_detail` has `detail` — specific failure reason

## app_dependency_health

Simple binary health indicator.

- Value `1` — dependency is healthy (last check succeeded)
- Value `0` — dependency is unhealthy (last check failed)

```text
app_dependency_health{name="my-service",group="my-team",dependency="postgres-main",type="postgres",host="pg.svc",port="5432",critical="yes"} 1
app_dependency_health{name="my-service",group="my-team",dependency="redis-cache",type="redis",host="redis.svc",port="6379",critical="no"} 0
```

### PromQL Examples

```promql
# All unhealthy dependencies
app_dependency_health == 0

# Unhealthy critical dependencies
app_dependency_health{critical="yes"} == 0

# Health status of a specific dependency
app_dependency_health{name="my-service",dependency="postgres-main"}
```

## app_dependency_latency_seconds

Distribution summary of health check latency with histogram buckets.

SLO boundaries: `0.001`, `0.005`, `0.01`, `0.05`, `0.1`, `0.5`, `1.0`,
`5.0` seconds.

```text
app_dependency_latency_seconds_bucket{...,le="0.001"} 0
app_dependency_latency_seconds_bucket{...,le="0.005"} 42
app_dependency_latency_seconds_bucket{...,le="0.01"} 45
app_dependency_latency_seconds_bucket{...,le="0.05"} 48
app_dependency_latency_seconds_bucket{...,le="0.1"} 48
app_dependency_latency_seconds_bucket{...,le="0.5"} 48
app_dependency_latency_seconds_bucket{...,le="1"} 48
app_dependency_latency_seconds_bucket{...,le="5"} 48
app_dependency_latency_seconds_bucket{...,le="+Inf"} 48
app_dependency_latency_seconds_sum{...} 0.192
app_dependency_latency_seconds_count{...} 48
```

### PromQL Examples

```promql
# P95 latency over last 5 minutes
histogram_quantile(0.95,
  rate(app_dependency_latency_seconds_bucket[5m])
)

# Average latency
rate(app_dependency_latency_seconds_sum[5m])
  / rate(app_dependency_latency_seconds_count[5m])

# P99 for a specific dependency
histogram_quantile(0.99,
  rate(app_dependency_latency_seconds_bucket{dependency="postgres-main"}[5m])
)
```

## app_dependency_status

Enum-pattern gauge. For each endpoint, 8 time series are created — one
per status category. Exactly one series has value `1`, the rest have `0`.

Status categories:

| `status` label | Meaning |
| --- | --- |
| `ok` | Healthy — check succeeded |
| `timeout` | Check timed out |
| `connection_error` | Cannot connect to dependency |
| `dns_error` | DNS resolution failed |
| `auth_error` | Authentication/authorization failed |
| `tls_error` | TLS handshake failed |
| `unhealthy` | Connected but dependency reports unhealthy |
| `error` | Unexpected/unclassified error |

```text
app_dependency_status{...,status="ok"} 1
app_dependency_status{...,status="timeout"} 0
app_dependency_status{...,status="connection_error"} 0
app_dependency_status{...,status="dns_error"} 0
app_dependency_status{...,status="auth_error"} 0
app_dependency_status{...,status="tls_error"} 0
app_dependency_status{...,status="unhealthy"} 0
app_dependency_status{...,status="error"} 0
```

### PromQL Examples

```promql
# All endpoints with auth errors
app_dependency_status{status="auth_error"} == 1

# All endpoints with connection errors
app_dependency_status{status="connection_error"} == 1

# Count of unhealthy endpoints by team
count(app_dependency_status{status="ok"} == 0) by (group)

# Alert: any critical dependency not OK for 2 minutes
app_dependency_status{status="ok",critical="yes"} == 0
```

## app_dependency_status_detail

Info-pattern gauge. One series per unique detail value. The `detail`
label provides a specific failure reason.

Common detail values:

| Detail | Source | Meaning |
| --- | --- | --- |
| `ok` | All checkers | Check succeeded |
| `auth_error` | HTTP, gRPC, PG, MySQL, Redis, AMQP, LDAP | Auth failure |
| `http_500` | HTTP | Server error |
| `http_503` | HTTP | Service unavailable |
| `grpc_not_serving` | gRPC | Service not serving |
| `grpc_unknown` | gRPC | Unknown gRPC status |
| `no_brokers` | Kafka | No brokers in metadata |
| `connection_refused` | Redis, core | Connection refused |
| `timeout` | Core | Check timed out |
| `dns_error` | Core | DNS error |
| `tls_error` | Core, LDAP | TLS error |
| `unhealthy` | LDAP | Server busy/unavailable |
| `error` | Core | Unclassified error |

```text
app_dependency_status_detail{...,detail="ok"} 1
```

When the detail changes (e.g., from `ok` to `http_503`), the old series
is deleted and a new one is created with value `1`.

### PromQL Examples

```promql
# Endpoints returning HTTP 503
app_dependency_status_detail{detail="http_503"} == 1

# All active details (not ok)
app_dependency_status_detail{detail!="ok"} == 1
```

## Exposing Metrics

### Spring Boot

Metrics are automatically available at `/actuator/prometheus`. Ensure
the endpoint is exposed:

```yaml
management:
  endpoints:
    web:
      exposure:
        include: health, prometheus, dependencies
```

### Programmatic API

Export via Micrometer `PrometheusMeterRegistry`:

```java
import io.micrometer.prometheus.PrometheusMeterRegistry;

// In HTTP handler for /metrics:
String metrics = meterRegistry.scrape();
response.setContentType("text/plain; version=0.0.4");
response.getWriter().write(metrics);
```

## See Also

- [Getting Started](getting-started.md) — basic setup with Prometheus
- [Checkers](checkers.md) — error classification per checker
- [Grafana Dashboards](../../docs/grafana-dashboards.md) — dashboard configuration
- [Alert Rules](../../docs/alerting/alert-rules.md) — alerting based on these metrics
- [Metric Specification](../../spec/metric-contract.md) — formal metric contract
- [Troubleshooting](troubleshooting.md) — common issues and solutions
