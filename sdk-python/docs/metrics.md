*[Русская версия](metrics.ru.md)*

# Prometheus Metrics

This guide covers all Prometheus metrics exported by the dephealth Python SDK,
their labels, and PromQL query examples.

## Overview

| Metric | Type | Description |
| --- | --- | --- |
| `app_dependency_health` | Gauge | Binary health indicator (0 or 1) |
| `app_dependency_latency_seconds` | Histogram | Check latency distribution |
| `app_dependency_status` | Gauge (enum) | Status category (8 series per endpoint) |
| `app_dependency_status_detail` | Gauge (info) | Detailed failure reason |

## Labels

All metrics share these common labels:

| Label | Description | Example |
| --- | --- | --- |
| `name` | Application name | `my-service` |
| `group` | Application group | `my-team` |
| `dependency` | Dependency name | `postgres-main` |
| `type` | Dependency type | `postgres`, `redis`, `http`, `grpc`, `tcp`, `amqp`, `kafka`, `ldap`, `mysql` |
| `host` | Endpoint hostname | `pg.svc` |
| `port` | Endpoint port | `5432` |
| `critical` | Criticality flag | `yes` or `no` |

Custom labels (specified via `labels=` parameter) are added alongside the
standard labels.

## app_dependency_health

Binary health indicator for each endpoint.

- **Type:** Gauge
- **Values:** `1` (healthy) or `0` (unhealthy)

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

# Health status for a specific service
app_dependency_health{name="my-service"}

# Percentage of healthy dependencies
avg(app_dependency_health{name="my-service"}) * 100
```

## app_dependency_latency_seconds

Histogram of check latency for each endpoint.

- **Type:** Histogram
- **Buckets:** `0.001`, `0.005`, `0.01`, `0.05`, `0.1`, `0.5`, `1.0`, `5.0`

```text
app_dependency_latency_seconds_bucket{...,le="0.001"} 0
app_dependency_latency_seconds_bucket{...,le="0.005"} 12
app_dependency_latency_seconds_bucket{...,le="0.01"} 42
app_dependency_latency_seconds_bucket{...,le="0.05"} 99
app_dependency_latency_seconds_bucket{...,le="0.1"} 100
app_dependency_latency_seconds_bucket{...,le="0.5"} 100
app_dependency_latency_seconds_bucket{...,le="1.0"} 100
app_dependency_latency_seconds_bucket{...,le="5.0"} 100
app_dependency_latency_seconds_bucket{...,le="+Inf"} 100
app_dependency_latency_seconds_count{...} 100
app_dependency_latency_seconds_sum{...} 0.42
```

### PromQL Examples

```promql
# 95th percentile latency per dependency
histogram_quantile(0.95,
  rate(app_dependency_latency_seconds_bucket{name="my-service"}[5m]))

# Average latency per dependency
rate(app_dependency_latency_seconds_sum[5m])
  / rate(app_dependency_latency_seconds_count[5m])

# Latency above 1s (potential timeout)
histogram_quantile(0.99,
  rate(app_dependency_latency_seconds_bucket[5m])) > 1
```

## app_dependency_status

Enum-pattern gauge: 8 series per endpoint, exactly one has value `1`.

- **Type:** Gauge
- **Additional label:** `status`
- **Values:** `1` (active status) or `0` (inactive)

Status categories:

| Status | Description |
| --- | --- |
| `ok` | Check succeeded |
| `timeout` | Check timed out |
| `connection_error` | Connection refused or reset |
| `dns_error` | DNS resolution failed |
| `auth_error` | Authentication failed |
| `tls_error` | TLS error |
| `unhealthy` | Endpoint reported unhealthy |
| `error` | Other unexpected error |

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
# Current status category for a dependency
app_dependency_status{dependency="postgres-main",status!=""} == 1

# All endpoints with auth errors
app_dependency_status{status="auth_error"} == 1

# All endpoints with timeout
app_dependency_status{status="timeout"} == 1

# Count of non-ok endpoints per service
count(app_dependency_status{status!="ok"} == 1) by (name)
```

## app_dependency_status_detail

Info-pattern gauge: provides a detailed failure reason string.

- **Type:** Gauge
- **Additional label:** `detail`
- **Value:** `1` when series exists; series is deleted and re-created when detail changes

Detail values depend on the checker type:

| Detail | Description |
| --- | --- |
| `ok` | Check succeeded |
| `timeout` | Check timed out |
| `connection_refused` | Connection refused |
| `dns_error` | DNS resolution failed |
| `auth_error` | Authentication failed |
| `tls_error` | TLS error |
| `http_<code>` | HTTP status code (e.g., `http_503`) |
| `grpc_not_serving` | gRPC service not serving |
| `no_brokers` | No Kafka brokers available |
| `ldap_no_results` | LDAP search returned no results |

```text
app_dependency_status_detail{...,detail="ok"} 1
```

### PromQL Examples

```promql
# Detailed reason for a specific dependency
app_dependency_status_detail{dependency="postgres-main",detail!=""} == 1

# All endpoints returning HTTP 503
app_dependency_status_detail{detail="http_503"} == 1
```

## Metrics Exposure

### FastAPI

```python
from dephealth_fastapi import DepHealthMiddleware

app.add_middleware(DepHealthMiddleware)
# Metrics available at GET /metrics
```

### Without FastAPI

Use the standard `prometheus_client`:

```python
from prometheus_client import generate_latest, CONTENT_TYPE_LATEST

# In your HTTP handler:
def metrics_handler(request):
    return Response(
        content=generate_latest(),
        media_type=CONTENT_TYPE_LATEST,
    )
```

### Custom Registry

```python
from prometheus_client import CollectorRegistry

custom_registry = CollectorRegistry()

dh = DependencyHealth("my-service", "my-team",
    registry=custom_registry,
    # ... dependency specs
)

# Expose custom registry
from prometheus_client import generate_latest
output = generate_latest(custom_registry)
```

## See Also

- [Getting Started](getting-started.md) — basic setup and first example
- [Checkers](checkers.md) — all 9 built-in checkers
- [Grafana Dashboards](../../docs/grafana-dashboards.md) — dashboard configuration
- [Alert Rules](../../docs/alerting/alert-rules.md) — alerting configuration
- [Specification](../../spec/) — cross-SDK metric contracts
- [Troubleshooting](troubleshooting.md) — common issues and solutions
