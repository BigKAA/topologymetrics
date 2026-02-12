*[Русская версия](overview.ru.md)*

# Monitoring Stack Overview

> This document describes the architecture of the dephealth monitoring stack:
> metric collection, storage, alerting, and notification delivery.
> For alert rule details, see [Alert Rules](alert-rules.md).
> For Grafana dashboards, see [Grafana Dashboards](../grafana-dashboards.md).

---

## Architecture

The dephealth monitoring stack consists of four components connected in a pipeline:

```text
┌─────────────────────┐
│  Services (SDK)     │  dephealth SDK exports Prometheus metrics
│  /metrics endpoint  │  app_dependency_health, app_dependency_latency_seconds
└────────┬────────────┘
         │ scrape (pull)
         ▼
┌─────────────────────┐
│  VictoriaMetrics    │  Time-series database (TSDB)
│  or Prometheus      │  Collects and stores metrics
│  :8428              │
└────────┬────────────┘
         │ query (PromQL / MetricsQL)
         ▼
┌─────────────────────┐
│  VMAlert            │  Alert evaluation engine
│  or Prometheus      │  Evaluates rules, fires alerts
│  :8880              │
└────────┬────────────┘
         │ push alerts
         ▼
┌─────────────────────┐
│  Alertmanager       │  Alert routing and notification
│  :9093              │  Grouping, inhibition, silencing
└────────┬────────────┘
         │ notify
         ▼
┌─────────────────────┐
│  Channels           │  Telegram, Slack, webhook,
│                     │  PagerDuty, email, ...
└─────────────────────┘
```

Each component can be replaced independently. VictoriaMetrics is interchangeable with Prometheus, and VMAlert with Prometheus alerting rules — the alert rule format and PromQL expressions are compatible.

## Components

| Component | Role | Default Port | Image | Version |
| --- | --- | --- | --- | --- |
| VictoriaMetrics | TSDB + metric scraping | 8428 | `victoriametrics/victoria-metrics` | v1.108.1 |
| VMAlert | Alert rule evaluation | 8880 | `victoriametrics/vmalert` | v1.108.1 |
| Alertmanager | Alert routing and notifications | 9093 | `prom/alertmanager` | v0.28.1 |
| Grafana | Visualization (dashboards) | 3000 | `grafana/grafana` | 11.6.0 |

All components are deployed via the Helm chart `deploy/helm/dephealth-monitoring/`.

## Metric Collection (Scraping)

VictoriaMetrics (or Prometheus) periodically pulls metrics from service endpoints.

### Default Parameters

| Parameter | Value | Description |
| --- | --- | --- |
| `scrape_interval` | 15s | How often metrics are collected |
| `scrape_timeout` | 10s | Timeout per scrape request |
| `metrics_path` | `/metrics` | Default endpoint path |
| `retentionPeriod` | 7d | How long metrics are stored |

### Adding Scrape Targets

Targets are configured in `values.yaml` under `scrapeTargets`:

```yaml
scrapeTargets:
  - jobName: my-service           # Prometheus job_name
    host: my-service.default.svc  # Kubernetes service DNS
    port: "8080"                  # Metrics port
    namespace: default            # Kubernetes namespace (added as label)
    service: my-service           # Service name (added as label)
```

For Spring Boot applications, override the metrics path:

```yaml
  - jobName: my-java-service
    host: my-java-service.default.svc
    port: "8080"
    metricsPath: /actuator/prometheus  # Spring Boot Actuator
    namespace: default
    service: my-java-service
```

Additional targets (e.g., for conformance tests) can be added via `extraScrapeTargets`:

```yaml
extraScrapeTargets:
  - jobName: extra-service
    host: extra-service.testing.svc
    port: "8080"
    namespace: testing
    service: extra-service
```

### Labels Added by Scraping

Each scraped target automatically receives labels from the configuration:

| Label | Source | Example |
| --- | --- | --- |
| `job` | `jobName` field | `my-service` |
| `namespace` | `namespace` field | `default` |
| `service` | `service` field | `my-service` |
| `instance` | `host:port` | `my-service.default.svc:8080` |

These labels are used in alert rules for grouping and routing (see [Alert Rules](alert-rules.md)).

## VictoriaMetrics and Prometheus Compatibility

The dephealth monitoring stack uses VictoriaMetrics by default, but all configurations are compatible with Prometheus.

| Feature | VictoriaMetrics | Prometheus |
| --- | --- | --- |
| Query language | MetricsQL (superset of PromQL) | PromQL |
| Alert rules format | Same YAML format | Same YAML format |
| Scrape config | Prometheus-compatible `scrape.yml` | Native `prometheus.yml` |
| Alert evaluation | VMAlert (separate binary) | Built into Prometheus |
| Alertmanager | Same Alertmanager | Same Alertmanager |

**Key differences**:

- MetricsQL extends PromQL with additional functions (`median`, `limitk`, `range_median`, etc.), but all standard PromQL functions work in both.
- The alert rules in this project use only standard PromQL, so they work with both VMAlert and Prometheus without modification.
- VMAlert is a standalone process that queries VictoriaMetrics; Prometheus evaluates rules internally.

### Using with Prometheus

To use Prometheus instead of VictoriaMetrics:

1. Replace the scrape config format (minimal changes — the `scrape.yml` format is identical).
2. Load alert rules via `rule_files` in `prometheus.yml` or use the `PrometheusRule` CRD (with Prometheus Operator).
3. Configure `--alertmanager.url` in Prometheus to point to Alertmanager.

## Alert Evaluation

VMAlert evaluates alert rules at a configurable interval (default: every 15 seconds) and sends firing alerts to Alertmanager.

| Parameter | Value | Helm Value |
| --- | --- | --- |
| Evaluation interval | 15s | `vmalert.evaluationInterval` |
| Datasource | VictoriaMetrics | `http://victoriametrics:8428` |
| Notifier | Alertmanager | `http://alertmanager:9093` |
| Rules path | `/rules/*.yml` | ConfigMap `vmalert-rules` |

The 5 built-in alert rules are described in [Alert Rules](alert-rules.md).

## Quick Start

### Deploy with Helm

```bash
# Default configuration
helm install dephealth-monitoring deploy/helm/dephealth-monitoring/

# With homelab overrides (custom registry, storage class, Grafana route)
helm install dephealth-monitoring deploy/helm/dephealth-monitoring/ \
  -f deploy/helm/dephealth-monitoring/values-homelab.yaml
```

### Verify the Stack

```bash
# Check all pods are running
kubectl get pods -n dephealth-monitoring

# Access VMAlert UI (port-forward)
kubectl port-forward -n dephealth-monitoring svc/vmalert 8880:8880
# Open http://localhost:8880 — shows alert rule status

# Access Alertmanager UI (port-forward)
kubectl port-forward -n dephealth-monitoring svc/alertmanager 9093:9093
# Open http://localhost:9093 — shows active alerts and silences

# Access Grafana (port-forward)
kubectl port-forward -n dephealth-monitoring svc/grafana 3000:3000
# Open http://localhost:3000 (admin / dephealth)
```

## Helm Chart Reference

Key `values.yaml` parameters for the monitoring stack:

| Section | Parameter | Default | Description |
| --- | --- | --- | --- |
| `global` | `imageRegistry` | `docker.io` | Docker registry for all images |
| `global` | `storageClass` | `""` (default) | StorageClass for PVC |
| `global` | `namespace` | `dephealth-monitoring` | Target namespace |
| `victoriametrics` | `enabled` | `true` | Enable VictoriaMetrics |
| `victoriametrics` | `retentionPeriod` | `7d` | Metric retention period |
| `victoriametrics` | `storage` | `2Gi` | PVC size for TSDB |
| `vmalert` | `enabled` | `true` | Enable VMAlert |
| `vmalert` | `evaluationInterval` | `15s` | Rule evaluation interval |
| `alertmanager` | `enabled` | `true` | Enable Alertmanager |
| `alertmanager.config` | `groupBy` | `[alertname, namespace, job, dependency]` | Alert grouping labels |
| `alertmanager.config` | `groupWait` | `30s` | Wait before sending group |
| `alertmanager.config` | `repeatInterval` | `4h` | Repeat interval for unresolved alerts |
| `alertmanager.config` | `webhookUrl` | `http://localhost:9094/webhook` | Default notification endpoint |

Full reference: [`deploy/helm/dephealth-monitoring/values.yaml`](../../deploy/helm/dephealth-monitoring/values.yaml)

## What's Next

- [Alert Rules](alert-rules.md) — detailed description of all 5 built-in alert rules
- [Noise Reduction](noise-reduction.md) — scenarios, inhibition, and best practices for reducing alert noise
- [Alertmanager Configuration](alertmanager.md) — routing, receivers (Telegram, webhook), notification templates
- [Custom Rules](custom-rules.md) — writing your own alert rules on top of dephealth metrics
