*[Русская версия](alertmanager.ru.md)*

# Alertmanager Configuration

> This document describes the Alertmanager configuration used in the dephealth
> monitoring stack: routing, receivers, inhibit rules, and notification templates.
> For the monitoring stack overview, see [Overview](overview.md).
> For noise reduction details, see [Noise Reduction](noise-reduction.md).

---

## Role of Alertmanager

Alertmanager sits between the alert evaluation engine (VMAlert / Prometheus) and notification channels. It performs four functions:

1. **Grouping** — combines related alerts into a single notification.
2. **Inhibition** — suppresses less important alerts when a more severe one fires.
3. **Routing** — directs alerts to receivers based on labels (severity, namespace, etc.).
4. **Notification** — delivers alerts to channels (Telegram, webhook, Slack, email, etc.).

```text
VMAlert / Prometheus
  │
  │ push firing alerts
  ▼
Alertmanager
  ├── Grouping     (group_by, group_wait)
  ├── Inhibition   (inhibit_rules)
  ├── Routing      (route tree)
  └── Notification (receivers)
        │
        ▼
  Telegram / Webhook / Slack / ...
```

---

## Default Configuration

The full default Alertmanager configuration shipped with the dephealth Helm chart:

```yaml
global:
  resolve_timeout: 5m

route:
  receiver: default
  group_by: ['alertname', 'namespace', 'job', 'dependency']
  group_wait: 30s
  group_interval: 5m
  repeat_interval: 4h

  routes:
    # Critical alerts — fast delivery
    - matchers:
        - severity = "critical"
      receiver: default
      group_wait: 10s
      repeat_interval: 1h

    # Info alerts — silent receiver (UI only)
    - matchers:
        - severity = "info"
      receiver: "null"

receivers:
  - name: default
    webhook_configs:
      - url: http://localhost:9094/webhook
        send_resolved: true

  - name: "null"

inhibit_rules:
  # DependencyDown suppresses DependencyDegraded
  - source_matchers:
      - alertname = "DependencyDown"
    target_matchers:
      - alertname = "DependencyDegraded"
    equal: ['job', 'namespace', 'dependency']

  # DependencyDown suppresses DependencyHighLatency
  - source_matchers:
      - alertname = "DependencyDown"
    target_matchers:
      - alertname = "DependencyHighLatency"
    equal: ['job', 'namespace', 'dependency']

  # DependencyDown suppresses DependencyFlapping
  - source_matchers:
      - alertname = "DependencyDown"
    target_matchers:
      - alertname = "DependencyFlapping"
    equal: ['job', 'namespace', 'dependency']

  # DependencyDegraded suppresses DependencyFlapping
  - source_matchers:
      - alertname = "DependencyDegraded"
    target_matchers:
      - alertname = "DependencyFlapping"
    equal: ['job', 'namespace', 'dependency']

  # Cascade: critical DependencyDown suppresses all warning
  - source_matchers:
      - alertname = "DependencyDown"
      - severity = "critical"
    target_matchers:
      - severity = "warning"
    equal: ['namespace', 'dependency']
```

---

## Routing Tree

The routing tree determines which receiver handles each alert. Routes are evaluated top-down; the first match wins.

### How It Works

```text
All alerts enter the root route
  │
  ├── severity = "critical"  →  default receiver (group_wait: 10s, repeat: 1h)
  │
  ├── severity = "info"      →  null receiver (no notification)
  │
  └── everything else        →  default receiver (group_wait: 30s, repeat: 4h)
      (severity = "warning")
```

### Root Route Parameters

| Parameter | Value | Description |
| --- | --- | --- |
| `receiver` | `default` | Fallback receiver for unmatched alerts |
| `group_by` | `[alertname, namespace, job, dependency]` | Labels for grouping alerts into notifications |
| `group_wait` | `30s` | Wait time after first alert before sending the group |
| `group_interval` | `5m` | Minimum interval between notifications for the same group |
| `repeat_interval` | `4h` | Interval for repeating unresolved alerts |

### Route: Critical Alerts

```yaml
- matchers:
    - severity = "critical"
  receiver: default
  group_wait: 10s
  repeat_interval: 1h
```

- **`group_wait: 10s`** — reduced from 30s for faster delivery. Critical alerts should reach the operator within seconds.
- **`repeat_interval: 1h`** — reduced from 4h. If a critical alert is unresolved for an hour, remind the operator.

### Route: Info Alerts

```yaml
- matchers:
    - severity = "info"
  receiver: "null"
```

Info alerts (DependencyFlapping) are routed to the null receiver — they appear in the Alertmanager UI but do not trigger notifications. See [Noise Reduction: Severity-Based Routing](noise-reduction.md#mechanism-4-severity-based-routing).

### Adding Custom Routes

To route alerts for a specific namespace or team to a different receiver:

```yaml
routes:
  # Existing routes...

  # Production namespace → PagerDuty
  - matchers:
      - namespace = "production"
      - severity = "critical"
    receiver: pagerduty-production

  # Staging namespace → low-priority Slack channel
  - matchers:
      - namespace = "staging"
    receiver: slack-staging
```

**Important**: custom routes should be placed **before** the catch-all routes. Alertmanager evaluates routes top-down and uses the first match.

---

## Receivers

### Default: Webhook

The default configuration uses a generic webhook receiver:

```yaml
receivers:
  - name: default
    webhook_configs:
      - url: http://localhost:9094/webhook
        send_resolved: true
```

| Parameter | Value | Description |
| --- | --- | --- |
| `url` | `http://localhost:9094/webhook` | Webhook endpoint (replace with your service) |
| `send_resolved` | `true` | Send a notification when the alert resolves |

**Webhook payload format**: Alertmanager sends a JSON POST with the following structure:

```json
{
  "version": "4",
  "status": "firing",
  "alerts": [
    {
      "status": "firing",
      "labels": {
        "alertname": "DependencyDown",
        "job": "order-api",
        "namespace": "production",
        "dependency": "user-db",
        "type": "postgres",
        "severity": "critical"
      },
      "annotations": {
        "summary": "Dependency user-db (postgres) is down in service order-api",
        "description": "All endpoints of dependency user-db..."
      },
      "startsAt": "2026-02-12T10:00:00Z",
      "endsAt": "0001-01-01T00:00:00Z",
      "generatorURL": "http://vmalert:8880/vmalert/alert?..."
    }
  ],
  "groupLabels": {
    "alertname": "DependencyDown",
    "namespace": "production",
    "job": "order-api",
    "dependency": "user-db"
  },
  "commonLabels": { ... },
  "commonAnnotations": { ... },
  "externalURL": "http://alertmanager:9093"
}
```

### Telegram

To send alerts to a Telegram chat, use a Telegram bot integration. There are two common approaches:

#### Option A: alertmanager-bot (Recommended)

[alertmanager-bot](https://github.com/metalmatze/alertmanager-bot) is a standalone service that receives Alertmanager webhooks and forwards them to Telegram with formatting.

Deploy the bot alongside Alertmanager and configure the webhook:

```yaml
receivers:
  - name: telegram
    webhook_configs:
      - url: http://alertmanager-bot:8080
        send_resolved: true
```

The bot configuration (environment variables or config file):

```yaml
# alertmanager-bot config
telegram:
  admin: 123456789              # Your Telegram user ID
  token: "bot123:ABC-DEF..."    # Bot token from @BotFather
listen:
  addr: ":8080"
alertmanager:
  url: "http://alertmanager:9093"
```

#### Option B: Direct Telegram API via webhook proxy

Use a lightweight webhook-to-Telegram proxy, or configure Alertmanager's webhook to point to a custom service that calls the Telegram Bot API:

```python
# Minimal Python webhook receiver (example)
from flask import Flask, request
import requests

app = Flask(__name__)
BOT_TOKEN = "bot123:ABC-DEF..."
CHAT_ID = "-1001234567890"

@app.route("/webhook", methods=["POST"])
def webhook():
    data = request.json
    for alert in data.get("alerts", []):
        status = "FIRING" if alert["status"] == "firing" else "RESOLVED"
        severity = alert["labels"].get("severity", "unknown")
        summary = alert["annotations"].get("summary", "No summary")
        text = f"[{status}] [{severity}] {summary}"
        requests.post(
            f"https://api.telegram.org/{BOT_TOKEN}/sendMessage",
            json={"chat_id": CHAT_ID, "text": text, "parse_mode": "HTML"}
        )
    return "ok", 200
```

#### Telegram Message Example

When DependencyDown fires, the Telegram message looks like:

```text
[FIRING] [critical] Dependency user-db (postgres) is down in service order-api

All endpoints of dependency user-db type postgres in service order-api
(namespace: production) have been unavailable for more than 1 minute.

Source: http://vmalert:8880/vmalert/alert?...
```

When the alert resolves:

```text
[RESOLVED] [critical] Dependency user-db (postgres) is down in service order-api
```

### Other Channels

Alertmanager natively supports many notification channels. Configuration examples are available in the official documentation:

| Channel | Alertmanager Docs | Key Configuration |
| --- | --- | --- |
| Slack | [slack_config](https://prometheus.io/docs/alerting/latest/configuration/#slack_config) | `api_url`, `channel`, `title`, `text` |
| PagerDuty | [pagerduty_config](https://prometheus.io/docs/alerting/latest/configuration/#pagerduty_config) | `routing_key`, `severity` |
| Email | [email_config](https://prometheus.io/docs/alerting/latest/configuration/#email_config) | `to`, `from`, `smarthost`, `auth_*` |
| OpsGenie | [opsgenie_config](https://prometheus.io/docs/alerting/latest/configuration/#opsgenie_config) | `api_key`, `priority` |
| Microsoft Teams | [msteams_config](https://prometheus.io/docs/alerting/latest/configuration/#msteams_config) | `webhook_url` |

Full Alertmanager configuration reference: [prometheus.io/docs/alerting/latest/configuration/](https://prometheus.io/docs/alerting/latest/configuration/)

---

## Inhibit Rules

Inhibit rules suppress redundant alerts. The dephealth configuration includes 5 rules organized in a suppression hierarchy:

```text
DependencyDown (critical)
  ├── → DependencyDegraded    (equal: job, namespace, dependency)
  ├── → DependencyHighLatency (equal: job, namespace, dependency)
  ├── → DependencyFlapping    (equal: job, namespace, dependency)
  └── → all warning           (equal: namespace, dependency)

DependencyDegraded (warning)
  └── → DependencyFlapping    (equal: job, namespace, dependency)
```

For a detailed explanation of each rule with real-world scenarios, see [Noise Reduction: Inhibit Rules](noise-reduction.md#inhibit-rules).

### Adding Custom Inhibit Rules

When adding [custom alert rules](custom-rules.md), consider adding corresponding inhibit rules. Example:

```yaml
# Custom rule: CriticalDependencyDown (only critical=yes dependencies)
# Should suppress the built-in DependencyDown for the same dependency
inhibit_rules:
  # ... existing rules ...

  - source_matchers:
      - alertname = "CriticalDependencyDown"
    target_matchers:
      - alertname = "DependencyDown"
    equal: ['job', 'namespace', 'dependency']
```

---

## Notification Templates

Alertmanager uses Go templates for formatting notification messages. The default configuration uses built-in templates, but you can customize them.

### Template Variables

| Variable | Description | Example |
| --- | --- | --- |
| `{{ .Status }}` | Alert status | `firing` or `resolved` |
| `{{ .Labels.alertname }}` | Alert name | `DependencyDown` |
| `{{ .Labels.severity }}` | Severity level | `critical` |
| `{{ .Labels.job }}` | Service name | `order-api` |
| `{{ .Labels.dependency }}` | Dependency name | `user-db` |
| `{{ .Labels.type }}` | Dependency type | `postgres` |
| `{{ .Labels.namespace }}` | Kubernetes namespace | `production` |
| `{{ .Annotations.summary }}` | Alert summary | `Dependency user-db (postgres) is down...` |
| `{{ .Annotations.description }}` | Detailed description | `All endpoints of dependency...` |
| `{{ .StartsAt }}` | Time alert started | `2026-02-12 10:00:00 UTC` |
| `{{ .EndsAt }}` | Time alert resolved | `2026-02-12 10:05:00 UTC` |
| `{{ .GeneratorURL }}` | Link to alert source | `http://vmalert:8880/...` |

### Custom Template Example

To customize the webhook message format, create a template file:

```yaml
# In alertmanager.yml, add:
templates:
  - '/etc/alertmanager/templates/*.tmpl'
```

Template file (`dephealth.tmpl`):

```text
{{ define "dephealth.title" -}}
[{{ .Status | toUpper }}] {{ .Labels.alertname }}: {{ .Labels.dependency }} ({{ .Labels.type }})
{{- end }}

{{ define "dephealth.text" -}}
Service: {{ .Labels.job }}
Namespace: {{ .Labels.namespace }}
Dependency: {{ .Labels.dependency }} ({{ .Labels.type }})
Severity: {{ .Labels.severity }}

{{ .Annotations.description }}

{{ if eq .Status "firing" -}}
Started: {{ .StartsAt.Format "2006-01-02 15:04:05 UTC" }}
{{- else -}}
Resolved: {{ .EndsAt.Format "2006-01-02 15:04:05 UTC" }}
Duration: {{ .EndsAt.Sub .StartsAt }}
{{- end }}

Source: {{ .GeneratorURL }}
{{- end }}
```

Use in webhook config:

```yaml
receivers:
  - name: default
    webhook_configs:
      - url: http://my-webhook:8080/alerts
        send_resolved: true
        title: '{{ template "dephealth.title" . }}'
        text: '{{ template "dephealth.text" . }}'
```

---

## Silences

Silences temporarily suppress notifications for matching alerts. Use them during planned maintenance or known issues.

### Creating a Silence via UI

1. Open Alertmanager UI: `http://alertmanager:9093/#/silences`
2. Click "New Silence"
3. Add matchers (e.g., `dependency = "user-db"`, `namespace = "staging"`)
4. Set duration and comment
5. Click "Create"

### Creating a Silence via API

```bash
# Silence all alerts for dependency "user-db" in staging for 2 hours
curl -X POST http://alertmanager:9093/api/v2/silences \
  -H "Content-Type: application/json" \
  -d '{
    "matchers": [
      {"name": "dependency", "value": "user-db", "isRegex": false},
      {"name": "namespace", "value": "staging", "isRegex": false}
    ],
    "startsAt": "2026-02-12T10:00:00Z",
    "endsAt": "2026-02-12T12:00:00Z",
    "createdBy": "admin",
    "comment": "Planned PostgreSQL maintenance"
  }'
```

### When to Use Silences

| Situation | Silence Matcher | Duration |
| --- | --- | --- |
| Database maintenance | `dependency="user-db"` | Duration of maintenance window |
| Staging environment deploy | `namespace="staging"` | Duration of deploy + buffer |
| Known flapping issue | `alertname="DependencyFlapping", dependency="cache"` | Until root cause is fixed |

---

## Helm Chart Parameters

Alertmanager parameters in `values.yaml`:

| Parameter | Default | Description |
| --- | --- | --- |
| `alertmanager.enabled` | `true` | Enable Alertmanager deployment |
| `alertmanager.image` | `prom/alertmanager` | Container image |
| `alertmanager.tag` | `v0.28.1` | Image tag |
| `alertmanager.config.resolveTimeout` | `5m` | Time before an alert is considered resolved if no update received |
| `alertmanager.config.groupBy` | `[alertname, namespace, job, dependency]` | Labels for grouping |
| `alertmanager.config.groupWait` | `30s` | Wait time before sending first notification |
| `alertmanager.config.groupInterval` | `5m` | Minimum interval between group notifications |
| `alertmanager.config.repeatInterval` | `4h` | Repeat interval for unresolved alerts |
| `alertmanager.config.webhookUrl` | `http://localhost:9094/webhook` | Default webhook URL |
| `alertmanager.resources.requests.cpu` | `50m` | CPU request |
| `alertmanager.resources.requests.memory` | `32Mi` | Memory request |
| `alertmanager.resources.limits.cpu` | `100m` | CPU limit |
| `alertmanager.resources.limits.memory` | `64Mi` | Memory limit |

### Override Examples

```yaml
# values-production.yaml
alertmanager:
  config:
    groupWait: 15s           # Faster initial notification
    repeatInterval: 2h       # More frequent repeats in production
    webhookUrl: "http://alertmanager-bot:8080"  # Telegram bot
```

---

## What's Next

- [Alert Rules](alert-rules.md) — detailed description of all 5 built-in rules
- [Noise Reduction](noise-reduction.md) — scenarios and best practices for reducing alert noise
- [Custom Rules](custom-rules.md) — writing your own alert rules on top of dephealth metrics
