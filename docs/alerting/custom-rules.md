*[Русская версия](custom-rules.ru.md)*

# Custom Alert Rules

> This document explains how to write your own alert rules on top of dephealth
> metrics. It provides templates, ready-to-use examples, and integration
> instructions for both VMAlert and Prometheus.
> For the built-in rules, see [Alert Rules](alert-rules.md).
> For noise reduction when adding custom rules, see [Noise Reduction](noise-reduction.md).

---

## When to Write Custom Rules

The 5 built-in rules cover common scenarios. Custom rules are needed when:

- **Different thresholds** — latency threshold of 500ms for databases vs 2s for external HTTP APIs.
- **Critical-only alerting** — alert only for dependencies marked `critical=yes`.
- **SLO-based monitoring** — alert when availability drops below a target percentage (e.g., 99%).
- **Per-namespace policies** — stricter alerting for production, relaxed for staging.
- **Per-team routing** — add team labels for routing to specific receivers.
- **Complete vs partial failure** — distinguish "all endpoints down" from "1 of N down" (the built-in DependencyDown uses `min`, which fires when any endpoint is down).

---

## Rule Template

Every alert rule follows this structure:

```yaml
groups:
  - name: custom.dephealth.rules
    rules:
      - alert: MyCustomAlert          # Alert name (PascalCase, descriptive)
        expr: |                        # PromQL expression
          <promql_expression>
        for: <duration>                # How long the condition must hold
        labels:
          severity: <critical|warning|info>
          <custom_label>: <value>      # Optional: team, environment, etc.
        annotations:
          summary: "<short description with {{ $labels.dependency }}>"
          description: "<detailed description>"
```

### Naming Conventions

- **Alert name**: `PascalCase`, descriptive. Prefix custom alerts to distinguish from built-in ones (e.g., `Custom` or your team name).
- **Group name**: use `custom.dephealth.rules` or `<team>.dephealth.rules` to separate from the built-in `dephealth.rules`.
- **Labels**: add `team`, `environment`, or `tier` labels for routing and filtering.

### Available Metrics

| Metric | Type | Labels | Description |
| --- | --- | --- | --- |
| `app_dependency_health` | Gauge (0/1) | `name`, `dependency`, `type`, `host`, `port`, `critical` | Current health status |
| `app_dependency_latency_seconds` | Histogram | `name`, `dependency`, `type`, `host`, `port`, `critical` | Health check latency distribution |

The `name` label identifies the application exporting metrics. The `dependency` label identifies the target dependency. See [Metric Contract](../../spec/metric-contract.md) for full details.

### Available Label Values

| Label | Possible Values |
| --- | --- |
| `type` | `http`, `grpc`, `tcp`, `postgres`, `mysql`, `redis`, `amqp`, `kafka` |
| `critical` | `yes`, `no` |

---

## Example 1: Critical Dependencies Only

Alert only for dependencies marked as critical — for a high-priority paging channel.

```yaml
- alert: CriticalDependencyDown
  expr: |
    min by (job, namespace, dependency, type) (
      app_dependency_health{critical="yes"}
    ) == 0
  for: 30s
  labels:
    severity: critical
    tier: platform
  annotations:
    summary: "CRITICAL dependency {{ $labels.dependency }} ({{ $labels.type }}) is down in {{ $labels.job }}"
    description: "Critical dependency {{ $labels.dependency }} in service {{ $labels.job }} (namespace: {{ $labels.namespace }}) has been unavailable for more than 30 seconds. This dependency is marked critical=yes, meaning the service cannot function without it."
```

**Key differences from built-in DependencyDown**:

| Parameter | Built-in | This rule |
| --- | --- | --- |
| Filter | All dependencies | Only `critical="yes"` |
| `for` | 1m | 30s (faster — critical deps need faster response) |
| Extra label | — | `tier: platform` (for routing) |

**Routing**: Add a route in Alertmanager to send `tier=platform` alerts to a high-priority channel:

```yaml
routes:
  - matchers:
      - tier = "platform"
    receiver: pagerduty-platform
    group_wait: 5s
```

**Inhibit rule**: Consider suppressing the built-in DependencyDown when this rule fires:

```yaml
inhibit_rules:
  - source_matchers:
      - alertname = "CriticalDependencyDown"
    target_matchers:
      - alertname = "DependencyDown"
    equal: ['job', 'namespace', 'dependency']
```

---

## Example 2: Per-Type Latency Thresholds

Different dependency types have different acceptable latency. Databases should respond within 500ms; external HTTP APIs may take up to 2 seconds.

### Database Latency (Strict)

```yaml
- alert: DatabaseHighLatency
  expr: |
    histogram_quantile(0.99,
      rate(app_dependency_latency_seconds_bucket{type=~"postgres|mysql"}[5m])
    ) > 0.5
  for: 3m
  labels:
    severity: warning
    tier: data
  annotations:
    summary: "Database {{ $labels.dependency }} P99 latency > 500ms in {{ $labels.job }}"
    description: "P99 latency for {{ $labels.dependency }} ({{ $labels.type }}) in service {{ $labels.job }} exceeds 500ms for more than 3 minutes. Current P99: {{ $value | printf \"%.3f\" }}s."
```

### Cache Latency (Strict)

```yaml
- alert: CacheHighLatency
  expr: |
    histogram_quantile(0.99,
      rate(app_dependency_latency_seconds_bucket{type="redis"}[5m])
    ) > 0.1
  for: 3m
  labels:
    severity: warning
    tier: data
  annotations:
    summary: "Cache {{ $labels.dependency }} P99 latency > 100ms in {{ $labels.job }}"
    description: "P99 latency for Redis cache {{ $labels.dependency }} in service {{ $labels.job }} exceeds 100ms for more than 3 minutes. Current P99: {{ $value | printf \"%.3f\" }}s."
```

### External HTTP API Latency (Relaxed)

```yaml
- alert: ExternalApiHighLatency
  expr: |
    histogram_quantile(0.99,
      rate(app_dependency_latency_seconds_bucket{type="http"}[5m])
    ) > 2
  for: 5m
  labels:
    severity: warning
  annotations:
    summary: "External API {{ $labels.dependency }} P99 latency > 2s in {{ $labels.job }}"
    description: "P99 latency for HTTP dependency {{ $labels.dependency }} in service {{ $labels.job }} exceeds 2 seconds for more than 5 minutes. Current P99: {{ $value | printf \"%.3f\" }}s."
```

**Comparison of thresholds**:

| Rule | Type Filter | P99 Threshold | `for` |
| --- | --- | --- | --- |
| Built-in DependencyHighLatency | All types | 1s | 5m |
| DatabaseHighLatency | postgres, mysql | 500ms | 3m |
| CacheHighLatency | redis | 100ms | 3m |
| ExternalApiHighLatency | http | 2s | 5m |

**Inhibit rules**: Consider suppressing the built-in DependencyHighLatency when a type-specific rule fires:

```yaml
inhibit_rules:
  - source_matchers:
      - alertname =~ "DatabaseHighLatency|CacheHighLatency|ExternalApiHighLatency"
    target_matchers:
      - alertname = "DependencyHighLatency"
    equal: ['job', 'namespace', 'dependency']
```

---

## Example 3: SLO-Based Availability

Alert when a dependency's availability drops below a target percentage over a time window.

### Availability < 99% Over 1 Hour

```yaml
- alert: DependencyBelowSLO
  expr: |
    avg_over_time(app_dependency_health[1h]) < 0.99
  for: 10m
  labels:
    severity: warning
  annotations:
    summary: "SLO violation: {{ $labels.dependency }} availability < 99% in {{ $labels.job }}"
    description: "Dependency {{ $labels.dependency }} ({{ $labels.type }}) in service {{ $labels.job }} has {{ $value | printf \"%.2f\" | humanizePercentage }} availability over the last hour. SLO target: 99%."
```

**How it works**:

- `avg_over_time(health[1h])` computes the average of 0s and 1s over 1 hour. If the dependency was down for 36 seconds in the last hour, the average is (3600-36)/3600 = 0.99 — exactly at the threshold.
- `for: 10m` adds a stability filter — the SLO must be violated for 10 consecutive minutes.

### Critical SLO: Availability < 95% Over 30 Minutes

```yaml
- alert: DependencySLOCritical
  expr: |
    avg_over_time(app_dependency_health[30m]) < 0.95
  for: 5m
  labels:
    severity: critical
  annotations:
    summary: "Critical SLO violation: {{ $labels.dependency }} availability < 95% in {{ $labels.job }}"
    description: "Dependency {{ $labels.dependency }} ({{ $labels.type }}) in service {{ $labels.job }} has {{ $value | printf \"%.2f\" | humanizePercentage }} availability over the last 30 minutes. This indicates a severe and persistent problem."
```

**SLO summary**:

| Rule | Window | Threshold | Severity | Meaning |
| --- | --- | --- | --- | --- |
| DependencyBelowSLO | 1h | < 99% | warning | Minor SLO violation, investigate |
| DependencySLOCritical | 30m | < 95% | critical | Severe degradation, act immediately |

---

## Example 4: Per-Namespace Alerting

Different environments need different alerting policies.

### Production: Strict

```yaml
- alert: ProductionDependencyDown
  expr: |
    min by (job, namespace, dependency, type) (
      app_dependency_health{namespace="production"}
    ) == 0
  for: 30s
  labels:
    severity: critical
    environment: production
  annotations:
    summary: "[PROD] {{ $labels.dependency }} ({{ $labels.type }}) is down in {{ $labels.job }}"
    description: "Production dependency {{ $labels.dependency }} in service {{ $labels.job }} has been unavailable for more than 30 seconds."
```

### Staging: Relaxed

```yaml
- alert: StagingDependencyDown
  expr: |
    min by (job, namespace, dependency, type) (
      app_dependency_health{namespace="staging"}
    ) == 0
  for: 5m
  labels:
    severity: warning
    environment: staging
  annotations:
    summary: "[STAGING] {{ $labels.dependency }} ({{ $labels.type }}) is down in {{ $labels.job }}"
    description: "Staging dependency {{ $labels.dependency }} in service {{ $labels.job }} has been unavailable for more than 5 minutes."
```

**Routing by environment**:

```yaml
routes:
  - matchers:
      - environment = "production"
      - severity = "critical"
    receiver: pagerduty-production
    group_wait: 5s

  - matchers:
      - environment = "staging"
    receiver: slack-staging
    repeat_interval: 12h
```

---

## Example 5: Complete vs Partial Failure

The built-in DependencyDown uses `min`, which fires when **any** endpoint is down. To distinguish between partial and complete failure:

### All Endpoints Down (Complete Failure)

```yaml
- alert: DependencyAllEndpointsDown
  expr: |
    (
      count by (job, namespace, dependency, type) (
        app_dependency_health
      )
      -
      count by (job, namespace, dependency, type) (
        app_dependency_health == 0
      )
    ) == 0
  for: 1m
  labels:
    severity: critical
  annotations:
    summary: "ALL endpoints of {{ $labels.dependency }} ({{ $labels.type }}) are down in {{ $labels.job }}"
    description: "Every endpoint of dependency {{ $labels.dependency }} in service {{ $labels.job }} is unavailable. Total endpoints: {{ with printf `count(app_dependency_health{job=\"%s\",dependency=\"%s\"})` .Labels.job .Labels.dependency | query }}{{ . | first | value }}{{ end }}."
```

**How it works**: `total count - count of zeros == 0` means all endpoints are at 0. This is different from `min == 0`, which fires when even one endpoint is down.

### Some Endpoints Down (Partial Failure)

This is the built-in DependencyDegraded rule. Use it as-is or adjust the `for` duration.

### Percentage of Endpoints Down

```yaml
- alert: DependencyMajorityDown
  expr: |
    (
      count by (job, namespace, dependency, type) (
        app_dependency_health == 0
      )
      /
      count by (job, namespace, dependency, type) (
        app_dependency_health
      )
    ) > 0.5
  for: 2m
  labels:
    severity: critical
  annotations:
    summary: "More than 50% of {{ $labels.dependency }} ({{ $labels.type }}) endpoints are down in {{ $labels.job }}"
    description: "{{ $value | printf \"%.0f\" | humanizePercentage }} of endpoints for dependency {{ $labels.dependency }} in service {{ $labels.job }} are unavailable."
```

---

## Recording Rules

Recording rules precompute expensive queries and store the results as new time series. Use them when the same expensive expression is used in multiple alerts or dashboards.

### Availability Over 1 Hour

```yaml
groups:
  - name: dephealth.recording
    interval: 1m
    rules:
      - record: dephealth:dependency_availability:avg1h
        expr: |
          avg_over_time(app_dependency_health[1h])

      - record: dephealth:dependency_latency_p99:5m
        expr: |
          histogram_quantile(0.99,
            rate(app_dependency_latency_seconds_bucket[5m])
          )

      - record: dephealth:dependency_flap_rate:15m
        expr: |
          changes(app_dependency_health[15m])
```

### Using Recording Rules in Alerts

```yaml
# Instead of recalculating avg_over_time every evaluation
- alert: DependencyBelowSLO
  expr: |
    dephealth:dependency_availability:avg1h < 0.99
  for: 10m
  labels:
    severity: warning
```

### Using Recording Rules in Grafana

Recording rules create standard time series that can be used in Grafana queries:

```promql
# Dashboard panel: availability heatmap
dephealth:dependency_availability:avg1h{namespace="production"}

# Dashboard panel: latency overview
dephealth:dependency_latency_p99:5m{type="postgres"}
```

### Naming Convention

Follow the Prometheus recording rule naming convention:

```text
<namespace>:<metric_name>:<aggregation_window>
```

- `dephealth:` — namespace prefix for all dephealth recording rules.
- `dependency_availability` — what is being measured.
- `avg1h` — aggregation type and window.

---

## Adding Rules to the Helm Chart

### Current Structure

The built-in rules are in a ConfigMap:

```text
ConfigMap: vmalert-rules
  └── dephealth.yml (5 built-in rules)
```

### Option 1: Additional ConfigMap (Manual)

Create a separate ConfigMap for custom rules and mount it to VMAlert:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: custom-alert-rules
  namespace: dephealth-monitoring
data:
  custom-rules.yml: |
    groups:
      - name: custom.dephealth.rules
        rules:
          - alert: CriticalDependencyDown
            expr: |
              min by (job, namespace, dependency, type) (
                app_dependency_health{critical="yes"}
              ) == 0
            for: 30s
            labels:
              severity: critical
            # ...
```

Add the volume mount to the VMAlert Deployment:

```yaml
volumeMounts:
  - name: rules
    mountPath: /rules
    readOnly: true
  - name: custom-rules          # Add this
    mountPath: /custom-rules
    readOnly: true
volumes:
  - name: rules
    configMap:
      name: vmalert-rules
  - name: custom-rules          # Add this
    configMap:
      name: custom-alert-rules
```

Update VMAlert args to include the custom rules path:

```yaml
args:
  - -rule=/rules/*.yml
  - -rule=/custom-rules/*.yml   # Add this
```

### Option 2: PrometheusRule CRD (Prometheus Operator)

If using Prometheus Operator, create a PrometheusRule resource:

```yaml
apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  name: custom-dephealth-rules
  labels:
    release: prometheus
spec:
  groups:
    - name: custom.dephealth.rules
      rules:
        - alert: CriticalDependencyDown
          expr: |
            min by (job, namespace, dependency, type) (
              app_dependency_health{critical="yes"}
            ) == 0
          for: 30s
          labels:
            severity: critical
          annotations:
            summary: "CRITICAL dependency {{ $labels.dependency }} is down"
```

---

## Checklist for Custom Rules

Before deploying a custom rule, verify:

- [ ] **PromQL is valid** — test the expression in VictoriaMetrics or Prometheus UI.
- [ ] **`for` duration is appropriate** — not too short (noise), not too long (delayed detection).
- [ ] **Severity matches impact** — critical for outages, warning for degradation, info for diagnostics.
- [ ] **Labels for routing** — add `team`, `environment`, or `tier` if you need custom routing.
- [ ] **Inhibit rules updated** — add inhibit rules to prevent overlap with built-in rules.
- [ ] **Annotations are useful** — include dependency name, type, service, and actionable information.
- [ ] **Recording rules for reuse** — if the same expression is used in multiple places, create a recording rule.

---

## What's Next

- [Alert Rules](alert-rules.md) — the 5 built-in rules (base for customization)
- [Noise Reduction](noise-reduction.md) — ensure your custom rules don't introduce noise
- [Alertmanager Configuration](alertmanager.md) — routing custom rules to the right receivers
- [Overview](overview.md) — monitoring stack architecture
