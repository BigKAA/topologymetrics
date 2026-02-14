*[Русская версия](metric-contract.ru.md)*

# Metric Contract

> Specification version: **3.0-draft**
>
> This document is the single source of truth for the format of metrics
> exported by all dephealth SDKs. All implementations must follow this contract.
> Compliance is verified by conformance tests.

---

## 1. General Principles

- All metrics are exported in **Prometheus text exposition format**
  (compatible with OpenMetrics).
- Metrics endpoint: `GET /metrics` (or path configured by the developer).
- Prefix for all metrics: `app_dependency_`.
- Metric and label names use only lowercase letters, digits, and `_`
  (according to [Prometheus naming conventions](https://prometheus.io/docs/practices/naming/)).

---

## 2. Health Metric: `app_dependency_health`

### 2.1. Description

Gauge metric reflecting the current availability status of a dependency.

### 2.2. Properties

| Property | Value |
| --- | --- |
| Name | `app_dependency_health` |
| Type | Gauge |
| Allowed values | `1` (available), `0` (unavailable) |
| Unit | dimensionless |

### 2.3. Required Labels

| Label | Description | Formation rules | Example |
| --- | --- | --- | --- |
| `name` | Unique name of the application exporting metrics | Lowercase letters, digits, `-`. Length: 1-63 characters. Format: `[a-z][a-z0-9-]*` | `order-api` |
| `dependency` | Logical name of the dependency, set by the developer. For services with dephealth SDK, the value must match the `name` of the target service | Lowercase letters, digits, `-`. Length: 1-63 characters. Format: `[a-z][a-z0-9-]*` | `payment-api` |
| `type` | Connection type / protocol | One of: `http`, `grpc`, `tcp`, `postgres`, `mysql`, `redis`, `amqp`, `kafka` | `postgres` |
| `host` | Endpoint address (hostname or IP) | As-is from configuration. IPv6 without square brackets | `pg-master.db.svc.cluster.local` |
| `port` | Endpoint port | String with number 1-65535. If port is not specified, the default port for the type is used | `5432` |
| `critical` | Criticality of the dependency for application operation | One of: `yes` (application cannot function without the dependency), `no` (degradation is acceptable). Required, no default value | `yes` |

### 2.4. Custom Labels

Developers can add arbitrary labels via `WithLabel(key, value)`.

**Rules**:

- Label name: format `[a-zA-Z_][a-zA-Z0-9_]*` (Prometheus naming conventions).
- Overriding required labels is forbidden: `name`, `dependency`, `type`,
  `host`, `port`, `critical`. Attempting to do so results in a configuration error.
- If a label is not specified, it is **not included** in the metric
  (rather than being output with an empty value).

**Usage examples**:

| Label | Description | Example |
| --- | --- | --- |
| `role` | Role of the instance in the cluster | `primary`, `replica` |
| `shard` | Shard identifier | `shard-01`, `0` |
| `vhost` | Virtual host (for AMQP) | `/`, `production` |
| `env` | Environment | `production`, `staging` |

### 2.5. Initial Value

Before the first check completes (during `initialDelay` + first cycle),
the metric is **not exported**. After the first successful or unsuccessful check,
the metric appears with a value of `1` or `0` respectively.

**Rationale**: absence of the metric instead of an arbitrary initial value
allows alerts to correctly handle service startup through `absent()`.

---

## 3. Latency Metric: `app_dependency_latency_seconds`

### 3.1. Description

Histogram metric recording the execution time of each health check.

### 3.2. Properties

| Property | Value |
| --- | --- |
| Name | `app_dependency_latency_seconds` |
| Type | Histogram |
| Unit | seconds |
| Buckets | `0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0, 5.0` |

### 3.3. Labels

Identical to `app_dependency_health` (required and optional).

### 3.4. What is Measured

Time from the start of `HealthChecker.Check()` call to receiving the result
(success or error). Includes:

- Connection establishment (if in standalone mode)
- Check execution (SQL query, HTTP request, etc.)
- Response receipt

Does not include:

- Wait time in scheduler queue
- Result processing time (updating metrics)

### 3.5. Behavior on Error

Latency is recorded **always** — for both successful and unsuccessful checks.
A timeout results in recording a value equal to the configured `timeout`
(or the actual time until timeout trigger).

### 3.6. Initial Value

Histogram appears after the first check (simultaneously with `app_dependency_health`).

---

## 4. Output Format `/metrics`

### 4.1. Prometheus text exposition format

SDK exports metrics in standard format:

```text
# HELP app_dependency_health Health status of a dependency (1 = healthy, 0 = unhealthy)
# TYPE app_dependency_health gauge
app_dependency_health{name="order-api",dependency="postgres-main",type="postgres",host="pg-master.db.svc",port="5432",critical="yes"} 1
app_dependency_health{name="order-api",dependency="redis-cache",type="redis",host="redis-0.cache.svc",port="6379",critical="no"} 1
app_dependency_health{name="order-api",dependency="payment-api",type="http",host="payment-svc.payments.svc",port="8080",critical="yes"} 0

# HELP app_dependency_latency_seconds Latency of dependency health check in seconds
# TYPE app_dependency_latency_seconds histogram
app_dependency_latency_seconds_bucket{name="order-api",dependency="postgres-main",type="postgres",host="pg-master.db.svc",port="5432",critical="yes",le="0.001"} 0
app_dependency_latency_seconds_bucket{name="order-api",dependency="postgres-main",type="postgres",host="pg-master.db.svc",port="5432",critical="yes",le="0.005"} 8
app_dependency_latency_seconds_bucket{name="order-api",dependency="postgres-main",type="postgres",host="pg-master.db.svc",port="5432",critical="yes",le="0.01"} 15
app_dependency_latency_seconds_bucket{name="order-api",dependency="postgres-main",type="postgres",host="pg-master.db.svc",port="5432",critical="yes",le="0.05"} 20
app_dependency_latency_seconds_bucket{name="order-api",dependency="postgres-main",type="postgres",host="pg-master.db.svc",port="5432",critical="yes",le="0.1"} 20
app_dependency_latency_seconds_bucket{name="order-api",dependency="postgres-main",type="postgres",host="pg-master.db.svc",port="5432",critical="yes",le="0.5"} 20
app_dependency_latency_seconds_bucket{name="order-api",dependency="postgres-main",type="postgres",host="pg-master.db.svc",port="5432",critical="yes",le="1"} 20
app_dependency_latency_seconds_bucket{name="order-api",dependency="postgres-main",type="postgres",host="pg-master.db.svc",port="5432",critical="yes",le="5"} 20
app_dependency_latency_seconds_bucket{name="order-api",dependency="postgres-main",type="postgres",host="pg-master.db.svc",port="5432",critical="yes",le="+Inf"} 20
app_dependency_latency_seconds_sum{name="order-api",dependency="postgres-main",type="postgres",host="pg-master.db.svc",port="5432",critical="yes"} 0.085
app_dependency_latency_seconds_count{name="order-api",dependency="postgres-main",type="postgres",host="pg-master.db.svc",port="5432",critical="yes"} 20
```

### 4.2. Format Requirements

- `# HELP` and `# TYPE` lines are mandatory for each metric.
- `# HELP` text is fixed (see examples above) and must not differ between SDKs.
- Label order: `name`, `dependency`, `type`, `host`, `port`, `critical`,
  then custom labels in alphabetical order.
- Label values are escaped according to Prometheus exposition format:
  characters `\`, `"`, `\n` are replaced with `\\`, `\"`, `\n`.

---

## 5. Behavior with Multiple Endpoints

One dependency can have multiple endpoints (database replicas, cluster nodes).

### 5.1. Rule: One Metric per Endpoint

Each endpoint produces a **separate** metric series. No aggregation is performed.

**Example**: PostgreSQL with primary and replica:

```text
app_dependency_health{name="order-api",dependency="postgres-main",type="postgres",host="pg-primary.db.svc",port="5432",critical="yes",role="primary"} 1
app_dependency_health{name="order-api",dependency="postgres-main",type="postgres",host="pg-replica.db.svc",port="5432",critical="yes",role="replica"} 1

app_dependency_latency_seconds_bucket{name="order-api",dependency="postgres-main",type="postgres",host="pg-primary.db.svc",port="5432",critical="yes",role="primary",le="0.005"} 10
app_dependency_latency_seconds_bucket{name="order-api",dependency="postgres-main",type="postgres",host="pg-replica.db.svc",port="5432",critical="yes",role="replica",le="0.005"} 8
```

### 5.2. Rationale

- Allows precise identification of which endpoint is unavailable.
- Alerting can be configured at the individual endpoint level
  (e.g., `DependencyDegraded` on partial failure).
- Aggregation when necessary is performed at the PromQL level:
  `min by (name, dependency) (app_dependency_health{dependency="postgres-main"})`.

### 5.3. Kafka: Multiple Brokers

For Kafka, each broker is a separate endpoint:

```text
app_dependency_health{name="order-api",dependency="kafka-main",type="kafka",host="kafka-0.kafka.svc",port="9092",critical="yes"} 1
app_dependency_health{name="order-api",dependency="kafka-main",type="kafka",host="kafka-1.kafka.svc",port="9092",critical="yes"} 1
app_dependency_health{name="order-api",dependency="kafka-main",type="kafka",host="kafka-2.kafka.svc",port="9092",critical="yes"} 0
```

---

## 6. Examples of Typical Configurations

### 6.1. Minimal Configuration (one service, one dependency)

```text
# HELP app_dependency_health Health status of a dependency (1 = healthy, 0 = unhealthy)
# TYPE app_dependency_health gauge
app_dependency_health{name="my-service",dependency="redis-cache",type="redis",host="redis.default.svc",port="6379",critical="no"} 1

# HELP app_dependency_latency_seconds Latency of dependency health check in seconds
# TYPE app_dependency_latency_seconds histogram
app_dependency_latency_seconds_bucket{name="my-service",dependency="redis-cache",type="redis",host="redis.default.svc",port="6379",critical="no",le="0.001"} 5
app_dependency_latency_seconds_bucket{name="my-service",dependency="redis-cache",type="redis",host="redis.default.svc",port="6379",critical="no",le="0.005"} 10
app_dependency_latency_seconds_bucket{name="my-service",dependency="redis-cache",type="redis",host="redis.default.svc",port="6379",critical="no",le="0.01"} 10
app_dependency_latency_seconds_bucket{name="my-service",dependency="redis-cache",type="redis",host="redis.default.svc",port="6379",critical="no",le="0.05"} 10
app_dependency_latency_seconds_bucket{name="my-service",dependency="redis-cache",type="redis",host="redis.default.svc",port="6379",critical="no",le="0.1"} 10
app_dependency_latency_seconds_bucket{name="my-service",dependency="redis-cache",type="redis",host="redis.default.svc",port="6379",critical="no",le="0.5"} 10
app_dependency_latency_seconds_bucket{name="my-service",dependency="redis-cache",type="redis",host="redis.default.svc",port="6379",critical="no",le="1"} 10
app_dependency_latency_seconds_bucket{name="my-service",dependency="redis-cache",type="redis",host="redis.default.svc",port="6379",critical="no",le="5"} 10
app_dependency_latency_seconds_bucket{name="my-service",dependency="redis-cache",type="redis",host="redis.default.svc",port="6379",critical="no",le="+Inf"} 10
app_dependency_latency_seconds_sum{name="my-service",dependency="redis-cache",type="redis",host="redis.default.svc",port="6379",critical="no"} 0.025
app_dependency_latency_seconds_count{name="my-service",dependency="redis-cache",type="redis",host="redis.default.svc",port="6379",critical="no"} 10
```

### 6.2. Typical Microservice (multiple dependencies of different types)

```text
# HELP app_dependency_health Health status of a dependency (1 = healthy, 0 = unhealthy)
# TYPE app_dependency_health gauge
app_dependency_health{name="order-api",dependency="postgres-main",type="postgres",host="pg.db.svc",port="5432",critical="yes"} 1
app_dependency_health{name="order-api",dependency="redis-cache",type="redis",host="redis.cache.svc",port="6379",critical="no"} 1
app_dependency_health{name="order-api",dependency="payment-api",type="http",host="payment.payments.svc",port="8080",critical="yes"} 1
app_dependency_health{name="order-api",dependency="auth-api",type="grpc",host="auth.auth.svc",port="9090",critical="yes"} 0
app_dependency_health{name="order-api",dependency="rabbitmq",type="amqp",host="rabbit.mq.svc",port="5672",critical="no"} 1
```

### 6.3. Service with AMQP and Custom Labels

```text
app_dependency_health{name="order-api",dependency="rabbitmq-orders",type="amqp",host="rabbit.mq.svc",port="5672",critical="yes",vhost="orders"} 1
app_dependency_health{name="order-api",dependency="rabbitmq-notifications",type="amqp",host="rabbit.mq.svc",port="5672",critical="no",vhost="notifications"} 1
```

### 6.4. Service in Degraded State (partial failure)

```text
app_dependency_health{name="order-api",dependency="postgres-main",type="postgres",host="pg-primary.db.svc",port="5432",critical="yes",role="primary"} 1
app_dependency_health{name="order-api",dependency="postgres-main",type="postgres",host="pg-replica-1.db.svc",port="5432",critical="yes",role="replica"} 0
app_dependency_health{name="order-api",dependency="postgres-main",type="postgres",host="pg-replica-2.db.svc",port="5432",critical="yes",role="replica"} 1
```

---

## 7. Useful PromQL Queries

For reference: typical queries to be used in Grafana and alerts.

```promql
# All unhealthy dependencies
app_dependency_health == 0

# Unhealthy dependencies of a specific service (by name)
app_dependency_health{name="order-api"} == 0

# All unhealthy critical dependencies
app_dependency_health{critical="yes"} == 0

# Aggregated dependency health (at least one endpoint down)
min by (name, dependency) (app_dependency_health) == 0

# P99 check latency over 5 minutes
histogram_quantile(0.99, rate(app_dependency_latency_seconds_bucket[5m]))

# Average latency by dependency
rate(app_dependency_latency_seconds_sum[5m]) / rate(app_dependency_latency_seconds_count[5m])

# Flapping dependencies — frequent toggles
changes(app_dependency_health[15m]) > 4

# Dependency graph: all edges (name -> dependency)
group by (name, dependency, type, critical) (app_dependency_health)

# All services that order-api depends on
app_dependency_health{name="order-api"}

# All services that depend on payment-api
app_dependency_health{dependency="payment-api"}
```

---

## 8. Status Metric: `app_dependency_status`

### 8.1. Description

Gauge metric (enum pattern) reflecting the **category** of the last check result.
For each endpoint, **all 8 values** of the `status` label are always exported.
Exactly one of them equals `1`, the remaining 7 equal `0`.
This eliminates series churn when the status changes.

### 8.2. Properties

| Property | Value |
| --- | --- |
| Name | `app_dependency_status` |
| Type | Gauge |
| HELP text | `Category of the last check result` |
| Allowed values | `1` (active status), `0` (inactive status) |

### 8.3. Status Values

| Value | Description | Typical Situations |
| --- | --- | --- |
| `ok` | Check succeeded, dependency is available | HTTP 2xx, gRPC SERVING, TCP connected, SQL SELECT 1 OK, Redis PONG |
| `timeout` | Check timeout exceeded | Connection timeout, query timeout, gRPC DEADLINE_EXCEEDED, context deadline exceeded |
| `connection_error` | Unable to establish TCP connection | Connection refused (RST), host unreachable, network unreachable, port not listening |
| `dns_error` | DNS name resolution failure | Hostname not found, DNS lookup failure, NXDOMAIN |
| `auth_error` | Authentication/authorization failure | Wrong DB credentials, Redis NOAUTH/WRONGPASS, AMQP 403 Access Refused |
| `tls_error` | TLS/SSL error | Certificate validation failed, TLS handshake error, expired certificate |
| `unhealthy` | Service responded but reports an unhealthy state | HTTP 4xx/5xx, gRPC NOT_SERVING, Kafka no brokers, Redis non-PONG, AMQP connection not open |
| `error` | Other unclassified errors | Unexpected exceptions, panics, pool exhaustion, query syntax error |

### 8.4. Labels

Same required and custom labels as `app_dependency_health` (section 2.3, 2.4),
plus the `status` label at the end.

Label order: `name`, `dependency`, `type`, `host`, `port`, `critical`,
custom labels in alphabetical order, `status`.

### 8.5. Initial Value

Before the first check completes, the metric is **not exported**
(same behavior as `app_dependency_health`, see section 2.5).
After the first check, all 8 series appear simultaneously.

### 8.6. Output Example

Endpoint `pg.svc:5432` is available (status = ok):

```text
# HELP app_dependency_status Category of the last check result
# TYPE app_dependency_status gauge
app_dependency_status{name="order-api",dependency="postgres-main",type="postgres",host="pg.svc",port="5432",critical="yes",status="ok"} 1
app_dependency_status{name="order-api",dependency="postgres-main",type="postgres",host="pg.svc",port="5432",critical="yes",status="timeout"} 0
app_dependency_status{name="order-api",dependency="postgres-main",type="postgres",host="pg.svc",port="5432",critical="yes",status="connection_error"} 0
app_dependency_status{name="order-api",dependency="postgres-main",type="postgres",host="pg.svc",port="5432",critical="yes",status="dns_error"} 0
app_dependency_status{name="order-api",dependency="postgres-main",type="postgres",host="pg.svc",port="5432",critical="yes",status="auth_error"} 0
app_dependency_status{name="order-api",dependency="postgres-main",type="postgres",host="pg.svc",port="5432",critical="yes",status="tls_error"} 0
app_dependency_status{name="order-api",dependency="postgres-main",type="postgres",host="pg.svc",port="5432",critical="yes",status="unhealthy"} 0
app_dependency_status{name="order-api",dependency="postgres-main",type="postgres",host="pg.svc",port="5432",critical="yes",status="error"} 0
```

---

## 9. Status Detail Metric: `app_dependency_status_detail`

### 9.1. Description

Gauge metric (info pattern) containing the **detailed reason** of the last check result.
One series per endpoint, value is always `1`. The `detail` label contains the specific reason.
When the reason changes, the old series is deleted and a new one is created
(acceptable series churn for info-style metrics).

### 9.2. Properties

| Property | Value |
| --- | --- |
| Name | `app_dependency_status_detail` |
| Type | Gauge |
| HELP text | `Detailed reason of the last check result` |
| Allowed values | `1` (always) |

### 9.3. Detail Values by Checker Type

| Checker Type | Possible detail values |
| --- | --- |
| HTTP | `ok`, `timeout`, `connection_refused`, `dns_error`, `tls_error`, `http_NNN` (specific HTTP code: `http_404`, `http_503`, etc.), `error` |
| gRPC | `ok`, `timeout`, `connection_refused`, `dns_error`, `tls_error`, `grpc_not_serving`, `grpc_unknown`, `error` |
| TCP | `ok`, `timeout`, `connection_refused`, `dns_error`, `error` |
| PostgreSQL | `ok`, `timeout`, `connection_refused`, `dns_error`, `auth_error`, `tls_error`, `error` |
| MySQL | `ok`, `timeout`, `connection_refused`, `dns_error`, `auth_error`, `tls_error`, `error` |
| Redis | `ok`, `timeout`, `connection_refused`, `dns_error`, `auth_error`, `unhealthy`, `error` |
| AMQP | `ok`, `timeout`, `connection_refused`, `dns_error`, `auth_error`, `tls_error`, `unhealthy`, `error` |
| Kafka | `ok`, `timeout`, `connection_refused`, `dns_error`, `no_brokers`, `error` |

### 9.4. Mapping detail to status (Category)

Each `detail` value maps to exactly one `status` category (section 8.3):

| detail | status |
| --- | --- |
| `ok` | `ok` |
| `timeout` | `timeout` |
| `connection_refused`, `network_unreachable`, `host_unreachable` | `connection_error` |
| `dns_error` | `dns_error` |
| `auth_error` | `auth_error` |
| `tls_error` | `tls_error` |
| `http_NNN`, `grpc_not_serving`, `grpc_unknown`, `unhealthy`, `no_brokers` | `unhealthy` |
| `error`, `pool_exhausted`, `query_error` | `error` |

### 9.5. Labels

Same required and custom labels as `app_dependency_health` (section 2.3, 2.4),
plus the `detail` label at the end.

Label order: `name`, `dependency`, `type`, `host`, `port`, `critical`,
custom labels in alphabetical order, `detail`.

### 9.6. Initial Value

Before the first check completes, the metric is **not exported**
(same behavior as `app_dependency_health`, see section 2.5).

### 9.7. Output Example

`payment-api` returned HTTP 503:

```text
# HELP app_dependency_status_detail Detailed reason of the last check result
# TYPE app_dependency_status_detail gauge
app_dependency_status_detail{name="order-api",dependency="payment-api",type="http",host="payment.svc",port="8080",critical="yes",detail="http_503"} 1
```

### 9.8. Storage Impact

Each endpoint produces:

- `app_dependency_health`: 1 series
- `app_dependency_latency_seconds`: 10 series (8 buckets + sum + count)
- `app_dependency_status`: **8 series** (one per status value)
- `app_dependency_status_detail`: **1 series**

Total: +9 series per endpoint compared to the base (health + latency).

---

## 10. Extended PromQL Queries

In addition to the queries in section 7, the following queries leverage the new
status and detail metrics:

```promql
# Current status category for all dependencies
app_dependency_status == 1

# All dependencies with timeout
app_dependency_status{status="timeout"} == 1

# All dependencies with authentication error (alert-friendly)
app_dependency_status{status="auth_error"} == 1

# Detailed reason for a specific dependency
app_dependency_status_detail{name="order-api",dependency="payment-api"}

# All HTTP 503 errors across the cluster
app_dependency_status_detail{detail="http_503"}

# Flapping detection (no series churn — works correctly!)
changes(app_dependency_status{status="ok"}[15m]) > 4

# Distribution by status
count by (status) (app_dependency_status == 1)

# Correlation: unhealthy dependencies with details via join
app_dependency_status{status="unhealthy"} == 1
  AND on (name, dependency, type, host, port)
app_dependency_status_detail

# Alert: critical dependency with non-ok status for > 5 minutes
app_dependency_status{status!="ok",critical="yes"} == 1
  AND on (name, dependency, type, host, port)
(app_dependency_status offset 5m {status!="ok"} == 1)
```
