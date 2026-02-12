*[Русская версия](alert-rules.ru.md)*

# Alert Rules

> This document provides a detailed description of all 5 built-in alert rules
> shipped with the dephealth monitoring stack.
> For the monitoring stack overview, see [Overview](overview.md).
> For noise reduction techniques, see [Noise Reduction](noise-reduction.md).

---

## Summary

| Alert | Severity | For | Trigger Condition |
| --- | --- | --- | --- |
| [DependencyDown](#dependencydown) | critical | 1m | All endpoints of a dependency are unavailable |
| [DependencyDegraded](#dependencydegraded) | warning | 2m | Some endpoints are down, some are up |
| [DependencyHighLatency](#dependencyhighlatency) | warning | 5m | P99 latency exceeds 1 second |
| [DependencyFlapping](#dependencyflapping) | info | 0m | Status changed more than 4 times in 15 minutes |
| [DependencyAbsent](#dependencyabsent) | warning | 5m | No service exports `app_dependency_health` |

All rules use standard PromQL and are compatible with both VMAlert and Prometheus.

---

<a id="dependencydown"></a>

## 1. DependencyDown

**Complete failure**: all endpoints of a dependency are unavailable.

### Rule

```yaml
- alert: DependencyDown
  expr: |
    min by (job, namespace, dependency, type) (
      app_dependency_health
    ) == 0
  for: 1m
  labels:
    severity: critical
  annotations:
    summary: "Dependency {{ $labels.dependency }} ({{ $labels.type }}) is down in service {{ $labels.job }}"
    description: "All endpoints of dependency {{ $labels.dependency }} type {{ $labels.type }} in service {{ $labels.job }} (namespace: {{ $labels.namespace }}) have been unavailable for more than 1 minute."
```

### PromQL Breakdown

```text
min by (job, namespace, dependency, type) (
  app_dependency_health
) == 0
```

| Fragment | Meaning |
| --- | --- |
| `app_dependency_health` | Gauge metric: `1` = available, `0` = unavailable. One time series per endpoint (host:port). |
| `min by (job, namespace, dependency, type)` | Takes the **minimum** value across all endpoints of the same dependency. If any endpoint is healthy (`1`), the minimum is `1` and the alert does NOT fire. |
| `== 0` | The alert fires only when **all** endpoints are down (minimum = 0). |

**Why `min` and not `max` or `avg`?**

- `min == 0` means every single endpoint is down — complete failure.
- `max == 0` would give the same result (if max is 0, all are 0), but `min` makes the intent clearer.
- `avg` would be misleading: `avg == 0.5` could mean 1 of 2 endpoints is down — that's DependencyDegraded, not DependencyDown.

### Why `for: 1m`

- **Too short (0s–15s)**: network hiccups, pod restarts, and DNS propagation cause transient failures. Alerting immediately would produce noise.
- **1 minute**: long enough to filter single scrape failures (scrape interval = 15s, so 1m ≈ 4 scrapes). If the dependency is still down after 4 consecutive checks, it's a real problem.
- **Too long (5m+)**: for a complete failure, 5 minutes of silence is unacceptable. The operator needs to know quickly.

### Why `severity: critical`

Complete failure of a dependency usually means the service cannot fulfill its primary function. This requires immediate human attention, possibly a page.

### Example Alert

When this rule fires, Alertmanager receives an alert with these labels:

```json
{
  "alertname": "DependencyDown",
  "job": "order-api",
  "namespace": "production",
  "dependency": "user-db",
  "type": "postgres",
  "severity": "critical"
}
```

### Interaction with Other Rules

When DependencyDown fires, Alertmanager [inhibit rules](noise-reduction.md#inhibit-rules) suppress:

- DependencyDegraded (same dependency)
- DependencyHighLatency (same dependency)
- DependencyFlapping (same dependency)

This prevents alert storms. See [Noise Reduction: Scenario 1](noise-reduction.md#scenario-1-database-master-down).

---

<a id="dependencydegraded"></a>

## 2. DependencyDegraded

**Partial failure**: some endpoints of a dependency are down, but at least one is still healthy.

### Rule

```yaml
- alert: DependencyDegraded
  expr: |
    (
      count by (job, namespace, dependency, type) (
        app_dependency_health == 0
      ) > 0
    )
    and
    (
      count by (job, namespace, dependency, type) (
        app_dependency_health == 1
      ) > 0
    )
  for: 2m
  labels:
    severity: warning
  annotations:
    summary: "Dependency {{ $labels.dependency }} ({{ $labels.type }}) is degraded in service {{ $labels.job }}"
    description: "Some endpoints of dependency {{ $labels.dependency }} type {{ $labels.type }} in service {{ $labels.job }} (namespace: {{ $labels.namespace }}) have been unavailable for more than 2 minutes."
```

### PromQL Breakdown

```text
(
  count by (...) (app_dependency_health == 0) > 0   -- at least one endpoint is DOWN
)
and
(
  count by (...) (app_dependency_health == 1) > 0   -- at least one endpoint is UP
)
```

| Fragment | Meaning |
| --- | --- |
| `app_dependency_health == 0` | Selects only unhealthy endpoints (value = 0). |
| `count by (job, namespace, dependency, type) (...) > 0` | Counts unhealthy endpoints per dependency. `> 0` means at least one exists. |
| `and` | Both conditions must be true simultaneously. |
| Second `count ... == 1 ... > 0` | At least one endpoint is still healthy. |

**Why this structure?**

The `and` operator ensures the alert fires **only** when there's a mix of healthy and unhealthy endpoints. If all endpoints are down, the first condition is true but the second is false → DependencyDegraded does NOT fire (DependencyDown fires instead).

### Why `for: 2m`

- **Longer than DependencyDown (1m)**: partial degradation is often transient — a single replica restarting, a rolling update in progress, or a brief network partition.
- **2 minutes**: allows time for Kubernetes to reschedule pods or for a rolling restart to complete.
- **Not too long (5m+)**: if the degradation persists, the operator should know.

### Why `severity: warning`

The service is still partially functional. It's important but not an emergency — the remaining healthy endpoints handle traffic. Warning is appropriate for "investigate soon, not immediately."

### When Does This Rule Apply?

This rule is only meaningful for dependencies with **multiple endpoints** (e.g., PostgreSQL primary + replica, Redis cluster, multiple HTTP instances). For dependencies with a single endpoint, the state is always binary: DependencyDown or healthy.

### Example Alert

```json
{
  "alertname": "DependencyDegraded",
  "job": "order-api",
  "namespace": "production",
  "dependency": "cache",
  "type": "redis",
  "severity": "warning"
}
```

### Interaction with Other Rules

- **Suppressed by** DependencyDown (inhibit rule: if all endpoints are down, DependencyDown takes priority).
- **Suppresses** DependencyFlapping (inhibit rule: degradation already indicates instability).

See [Noise Reduction: Scenario 2](noise-reduction.md#scenario-2-one-replica-out-of-three-down).

---

<a id="dependencyhighlatency"></a>

## 3. DependencyHighLatency

**Performance degradation**: the dependency responds, but too slowly.

### Rule

```yaml
- alert: DependencyHighLatency
  expr: |
    histogram_quantile(0.99,
      rate(app_dependency_latency_seconds_bucket[5m])
    ) > 1
  for: 5m
  labels:
    severity: warning
  annotations:
    summary: "High latency for dependency {{ $labels.dependency }} ({{ $labels.type }}) in service {{ $labels.job }}"
    description: "P99 latency of dependency {{ $labels.dependency }} health check exceeds 1 second for more than 5 minutes."
```

### PromQL Breakdown

```text
histogram_quantile(0.99,
  rate(app_dependency_latency_seconds_bucket[5m])
) > 1
```

| Fragment | Meaning |
| --- | --- |
| `app_dependency_latency_seconds_bucket` | Histogram metric: distribution of health check latencies across buckets. |
| `rate(...[5m])` | Per-second rate of bucket counter increases over 5 minutes. Smooths out spikes. |
| `histogram_quantile(0.99, ...)` | Computes the 99th percentile (P99) from the histogram. 99% of checks complete within this time. |
| `> 1` | Threshold: 1 second. If P99 exceeds 1s, the dependency is too slow. |

**Why P99 and not average?**

- Average hides outliers: if 99% of checks take 10ms but 1% takes 30s, the average is ~300ms — looks fine.
- P99 catches the tail: the slowest 1% of checks. If P99 > 1s, there's a real performance problem.

**Why `rate` over 5 minutes?**

- Shorter windows (1m) are noisy — a single slow check can spike the percentile.
- 5 minutes provides a stable signal while remaining responsive.

### Why `for: 5m`

- **Latency fluctuates naturally**: database connection pools warm up, GC pauses, network congestion — all cause transient latency spikes.
- **5 minutes of `for`** on top of 5-minute `rate` window means the P99 must be consistently high for ~10 minutes total before the alert fires.
- This is intentionally conservative to avoid false positives.

### Why `severity: warning`

The dependency is **available** (health = 1) but slow. This is a performance issue, not an outage. Warning is appropriate — investigate, but don't page at 3 AM.

### Customizing the Threshold

The default threshold (1 second) may not be suitable for all dependency types. See [Custom Rules](custom-rules.md) for examples of per-type thresholds (e.g., 500ms for PostgreSQL, 2s for external HTTP APIs).

### Example Alert

```json
{
  "alertname": "DependencyHighLatency",
  "job": "order-api",
  "namespace": "production",
  "dependency": "payment-api",
  "type": "http",
  "severity": "warning"
}
```

### Interaction with Other Rules

- **Suppressed by** DependencyDown (inhibit rule: if the dependency is completely down, latency is irrelevant).
- **Independent of** DependencyDegraded — latency and partial failure are orthogonal problems.

See [Noise Reduction: Scenario 3](noise-reduction.md#scenario-3-http-service-responding-slowly).

---

<a id="dependencyflapping"></a>

## 4. DependencyFlapping

**Instability**: the dependency status keeps switching between healthy and unhealthy.

### Rule

```yaml
- alert: DependencyFlapping
  expr: |
    changes(app_dependency_health[15m]) > 4
  for: 0m
  labels:
    severity: info
  annotations:
    summary: "Dependency {{ $labels.dependency }} ({{ $labels.type }}) is flapping in service {{ $labels.job }}"
    description: "Dependency {{ $labels.dependency }} status has changed {{ $value }} times in the last 15 minutes."
```

### PromQL Breakdown

```text
changes(app_dependency_health[15m]) > 4
```

| Fragment | Meaning |
| --- | --- |
| `app_dependency_health` | Gauge: `1` or `0`. |
| `changes(...[15m])` | Counts the number of times the value changed in the last 15 minutes. Each 0→1 or 1→0 transition is one change. |
| `> 4` | More than 4 changes in 15 minutes — approximately one switch every 3.75 minutes or more often. |

**Why threshold of 4?**

- 1–2 changes in 15 minutes is normal: a brief network hiccup, a pod restart.
- 4+ changes means the dependency is unstable — repeatedly going up and down. This pattern usually indicates a deeper issue (resource exhaustion, network instability, misconfigured health check).

### Why `for: 0m`

- The `changes()` function already looks at a 15-minute window — the temporal smoothing is built into the expression.
- Adding `for` on top would delay the signal further, defeating the purpose of detecting instability in real time.

### Why `severity: info`

Flapping is a **diagnostic signal**, not an action item:

- It tells the operator "something is unstable — investigate the root cause."
- By default, `info` alerts are routed to the null receiver (Alertmanager UI only, no notifications).
- If flapping is important in your environment, you can change the receiver in Alertmanager configuration.

### Example Alert

```json
{
  "alertname": "DependencyFlapping",
  "job": "order-api",
  "namespace": "production",
  "dependency": "cache",
  "type": "redis",
  "severity": "info"
}
```

The `$value` in the description shows the actual number of changes (e.g., "status has changed 7 times").

### Interaction with Other Rules

- **Suppressed by** DependencyDown (if the dependency is completely down, flapping is not the issue).
- **Suppressed by** DependencyDegraded (partial failure already indicates the problem).

See [Noise Reduction: Scenario 4](noise-reduction.md#scenario-4-unstable-network-flapping).

---

<a id="dependencyabsent"></a>

## 5. DependencyAbsent

**Missing metrics**: no service exports the `app_dependency_health` metric.

### Rule

```yaml
- alert: DependencyAbsent
  expr: |
    absent(app_dependency_health{job=~".+"})
  for: 5m
  labels:
    severity: warning
  annotations:
    summary: "dephealth metrics are absent"
    description: "No service is exporting the app_dependency_health metric for more than 5 minutes."
```

### PromQL Breakdown

```text
absent(app_dependency_health{job=~".+"})
```

| Fragment | Meaning |
| --- | --- |
| `app_dependency_health` | The base health metric. |
| `{job=~".+"}` | Label matcher: requires at least one non-empty `job` label. This prevents the alert from firing on an empty TSDB (before any service is deployed). |
| `absent(...)` | Returns `1` if the vector is empty (no matching time series exist), otherwise returns nothing. |

**Why `{job=~".+"}`?**

Without this filter, `absent(app_dependency_health)` would fire immediately after deploying the monitoring stack — before any service starts exporting metrics. The label matcher ensures the alert only fires when previously existing metrics disappear.

### Why `for: 5m`

- **Service restarts**: during a rolling update, there's a brief gap when the old pod is terminated and the new one hasn't started. 5 minutes covers typical restart times.
- **Stale series**: VictoriaMetrics/Prometheus keeps stale series for a configurable period. 5 minutes is long enough for the scraper to detect the absence.
- **Deploy pipelines**: in CI/CD, services may be temporarily unavailable during deployment.

### Why `severity: warning`

Missing metrics usually means a deployment or scraping problem, not a dependency failure. It requires investigation but is not immediately critical.

### Common Causes

1. **Scrape target misconfiguration**: wrong host, port, or namespace in `values.yaml`.
2. **Service not running**: pods are in CrashLoopBackOff or not deployed.
3. **Metrics endpoint changed**: Spring Boot Actuator path changed, or the SDK was removed.
4. **Network policy**: firewall or NetworkPolicy blocks scraping.

### Example Alert

```json
{
  "alertname": "DependencyAbsent",
  "severity": "warning"
}
```

Note: this alert has no `job`, `dependency`, or `type` labels because there are no matching time series to extract them from.

### Interaction with Other Rules

- **Independent**: this alert fires when there is no data at all — other rules cannot fire without data.
- **Not suppressed** by any inhibit rule.

See [Noise Reduction: Scenario 5](noise-reduction.md#scenario-5-service-restart-or-deploy).

---

## Rule Placement

### VMAlert (VictoriaMetrics)

Rules are stored in a ConfigMap mounted to VMAlert:

```text
ConfigMap: vmalert-rules
  └── dephealth.yml     ← all 5 rules
      └── mounted at /rules/dephealth.yml
```

VMAlert loads rules from `/rules/*.yml` and evaluates them at the configured interval.

### Prometheus

For Prometheus, the same rules can be loaded via:

**Option 1** — `rule_files` in `prometheus.yml`:

```yaml
rule_files:
  - /etc/prometheus/rules/dephealth.yml
```

**Option 2** — `PrometheusRule` CRD (with Prometheus Operator):

```yaml
apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  name: dephealth-rules
  labels:
    release: prometheus   # must match Prometheus Operator selector
spec:
  groups:
    - name: dephealth.rules
      rules:
        # ... same rules as above ...
```

---

## What's Next

- [Noise Reduction](noise-reduction.md) — how these rules interact, inhibition, and real-world scenarios
- [Alertmanager Configuration](alertmanager.md) — routing, receivers, and notification delivery
- [Custom Rules](custom-rules.md) — writing your own rules on top of dephealth metrics
