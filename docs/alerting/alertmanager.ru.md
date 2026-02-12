*[English version](alertmanager.md)*

# Конфигурация Alertmanager

> Этот документ описывает конфигурацию Alertmanager в стеке мониторинга dephealth:
> маршрутизация, receivers, inhibit rules и шаблоны нотификаций.
> Обзор стека мониторинга — в [Обзор](overview.ru.md).
> Подробности об уменьшении шума — в [Уменьшение шума](noise-reduction.ru.md).

---

## Роль Alertmanager

Alertmanager находится между движком оценки алертов (VMAlert / Prometheus) и каналами нотификаций. Он выполняет четыре функции:

1. **Группировка** — объединяет связанные алерты в одно уведомление.
2. **Подавление** — подавляет менее важные алерты, когда более серьёзный уже сработал.
3. **Маршрутизация** — направляет алерты в receivers на основе labels (severity, namespace и т.д.).
4. **Нотификация** — доставляет алерты в каналы (Telegram, webhook, Slack, email и т.д.).

```text
VMAlert / Prometheus
  │
  │ push firing alerts
  ▼
Alertmanager
  ├── Группировка  (group_by, group_wait)
  ├── Подавление   (inhibit_rules)
  ├── Маршрутизация (route tree)
  └── Нотификация  (receivers)
        │
        ▼
  Telegram / Webhook / Slack / ...
```

---

## Конфигурация по умолчанию

Полная конфигурация Alertmanager, поставляемая с Helm chart dephealth:

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
    # Critical — быстрая доставка
    - matchers:
        - severity = "critical"
      receiver: default
      group_wait: 10s
      repeat_interval: 1h

    # Info — тихий приёмник (только в UI)
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
  # DependencyDown подавляет DependencyDegraded
  - source_matchers:
      - alertname = "DependencyDown"
    target_matchers:
      - alertname = "DependencyDegraded"
    equal: ['job', 'namespace', 'dependency']

  # DependencyDown подавляет DependencyHighLatency
  - source_matchers:
      - alertname = "DependencyDown"
    target_matchers:
      - alertname = "DependencyHighLatency"
    equal: ['job', 'namespace', 'dependency']

  # DependencyDown подавляет DependencyFlapping
  - source_matchers:
      - alertname = "DependencyDown"
    target_matchers:
      - alertname = "DependencyFlapping"
    equal: ['job', 'namespace', 'dependency']

  # DependencyDegraded подавляет DependencyFlapping
  - source_matchers:
      - alertname = "DependencyDegraded"
    target_matchers:
      - alertname = "DependencyFlapping"
    equal: ['job', 'namespace', 'dependency']

  # Каскад: critical DependencyDown подавляет все warning
  - source_matchers:
      - alertname = "DependencyDown"
      - severity = "critical"
    target_matchers:
      - severity = "warning"
    equal: ['namespace', 'dependency']
```

---

## Дерево маршрутизации

Дерево маршрутизации определяет, какой receiver обработает каждый алерт. Маршруты вычисляются сверху вниз — первое совпадение выигрывает.

### Как это работает

```text
Все алерты входят в корневой маршрут
  │
  ├── severity = "critical"  →  default receiver (group_wait: 10s, repeat: 1h)
  │
  ├── severity = "info"      →  null receiver (без нотификации)
  │
  └── всё остальное          →  default receiver (group_wait: 30s, repeat: 4h)
      (severity = "warning")
```

### Параметры корневого маршрута

| Параметр | Значение | Описание |
| --- | --- | --- |
| `receiver` | `default` | Fallback-receiver для несовпавших алертов |
| `group_by` | `[alertname, namespace, job, dependency]` | Labels для группировки алертов |
| `group_wait` | `30s` | Время ожидания после первого алерта перед отправкой группы |
| `group_interval` | `5m` | Минимальный интервал между уведомлениями одной группы |
| `repeat_interval` | `4h` | Интервал повтора неразрешённых алертов |

### Маршрут: Critical-алерты

```yaml
- matchers:
    - severity = "critical"
  receiver: default
  group_wait: 10s
  repeat_interval: 1h
```

- **`group_wait: 10s`** — сокращён с 30s для быстрой доставки. Critical-алерты должны дойти до оператора за секунды.
- **`repeat_interval: 1h`** — сокращён с 4h. Если critical-алерт неразрешён час — напомнить оператору.

### Маршрут: Info-алерты

```yaml
- matchers:
    - severity = "info"
  receiver: "null"
```

Info-алерты (DependencyFlapping) направляются в null receiver — они видны в UI Alertmanager, но не вызывают нотификаций. См. [Уменьшение шума: Маршрутизация по severity](noise-reduction.ru.md#механизм-4-маршрутизация-по-severity).

### Добавление кастомных маршрутов

Для направления алертов конкретного namespace или команды в другой receiver:

```yaml
routes:
  # Существующие маршруты...

  # Production namespace → PagerDuty
  - matchers:
      - namespace = "production"
      - severity = "critical"
    receiver: pagerduty-production

  # Staging namespace → низкоприоритетный Slack-канал
  - matchers:
      - namespace = "staging"
    receiver: slack-staging
```

**Важно**: кастомные маршруты должны быть размещены **перед** catch-all маршрутами. Alertmanager вычисляет маршруты сверху вниз и использует первое совпадение.

---

## Receivers

### По умолчанию: Webhook

Конфигурация по умолчанию использует generic webhook receiver:

```yaml
receivers:
  - name: default
    webhook_configs:
      - url: http://localhost:9094/webhook
        send_resolved: true
```

| Параметр | Значение | Описание |
| --- | --- | --- |
| `url` | `http://localhost:9094/webhook` | Endpoint webhook (замените на ваш сервис) |
| `send_resolved` | `true` | Отправлять нотификацию при разрешении алерта |

**Формат payload webhook**: Alertmanager отправляет JSON POST:

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
        "summary": "Зависимость user-db (postgres) недоступна в сервисе order-api",
        "description": "Все endpoint-ы зависимости user-db..."
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
  "commonLabels": { "..." : "..." },
  "commonAnnotations": { "..." : "..." },
  "externalURL": "http://alertmanager:9093"
}
```

### Telegram

Для отправки алертов в Telegram-чат используйте интеграцию с Telegram-ботом. Два подхода:

#### Вариант A: alertmanager-bot (рекомендуется)

[alertmanager-bot](https://github.com/metalmatze/alertmanager-bot) — отдельный сервис, принимающий webhooks Alertmanager и пересылающий их в Telegram с форматированием.

Разверните бота рядом с Alertmanager и настройте webhook:

```yaml
receivers:
  - name: telegram
    webhook_configs:
      - url: http://alertmanager-bot:8080
        send_resolved: true
```

Конфигурация бота (переменные окружения или файл):

```yaml
# конфигурация alertmanager-bot
telegram:
  admin: 123456789              # Ваш Telegram user ID
  token: "bot123:ABC-DEF..."    # Токен от @BotFather
listen:
  addr: ":8080"
alertmanager:
  url: "http://alertmanager:9093"
```

#### Вариант B: Прямой Telegram API через webhook-прокси

Используйте легковесный webhook-to-Telegram прокси или настройте webhook Alertmanager на кастомный сервис, вызывающий Telegram Bot API:

```python
# Минимальный Python webhook-receiver (пример)
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

#### Пример сообщения в Telegram

При срабатывании DependencyDown сообщение выглядит так:

```text
[FIRING] [critical] Зависимость user-db (postgres) недоступна в сервисе order-api

Все endpoint-ы зависимости user-db типа postgres в сервисе order-api
(namespace: production) недоступны более 1 минуты.

Source: http://vmalert:8880/vmalert/alert?...
```

При разрешении алерта:

```text
[RESOLVED] [critical] Зависимость user-db (postgres) недоступна в сервисе order-api
```

### Другие каналы

Alertmanager нативно поддерживает множество каналов. Примеры конфигурации — в официальной документации:

| Канал | Документация Alertmanager | Ключевые параметры |
| --- | --- | --- |
| Slack | [slack_config](https://prometheus.io/docs/alerting/latest/configuration/#slack_config) | `api_url`, `channel`, `title`, `text` |
| PagerDuty | [pagerduty_config](https://prometheus.io/docs/alerting/latest/configuration/#pagerduty_config) | `routing_key`, `severity` |
| Email | [email_config](https://prometheus.io/docs/alerting/latest/configuration/#email_config) | `to`, `from`, `smarthost`, `auth_*` |
| OpsGenie | [opsgenie_config](https://prometheus.io/docs/alerting/latest/configuration/#opsgenie_config) | `api_key`, `priority` |
| Microsoft Teams | [msteams_config](https://prometheus.io/docs/alerting/latest/configuration/#msteams_config) | `webhook_url` |

Полная документация конфигурации Alertmanager: [prometheus.io/docs/alerting/latest/configuration/](https://prometheus.io/docs/alerting/latest/configuration/)

---

## Inhibit Rules

Inhibit rules подавляют избыточные алерты. Конфигурация dephealth включает 5 правил, организованных в иерархию подавления:

```text
DependencyDown (critical)
  ├── → DependencyDegraded    (equal: job, namespace, dependency)
  ├── → DependencyHighLatency (equal: job, namespace, dependency)
  ├── → DependencyFlapping    (equal: job, namespace, dependency)
  └── → все warning           (equal: namespace, dependency)

DependencyDegraded (warning)
  └── → DependencyFlapping    (equal: job, namespace, dependency)
```

Подробное объяснение каждого правила с реальными сценариями — в [Уменьшение шума: Inhibit Rules](noise-reduction.ru.md#inhibit-rules).

### Добавление кастомных inhibit rules

При добавлении [кастомных правил алертов](custom-rules.ru.md) рассмотрите добавление соответствующих inhibit rules:

```yaml
# Кастомное правило: CriticalDependencyDown (только critical=yes зависимости)
# Должно подавлять встроенный DependencyDown для той же зависимости
inhibit_rules:
  # ... существующие правила ...

  - source_matchers:
      - alertname = "CriticalDependencyDown"
    target_matchers:
      - alertname = "DependencyDown"
    equal: ['job', 'namespace', 'dependency']
```

---

## Шаблоны нотификаций

Alertmanager использует Go-шаблоны для форматирования сообщений. Конфигурация по умолчанию использует встроенные шаблоны, но их можно настроить.

### Переменные шаблонов

| Переменная | Описание | Пример |
| --- | --- | --- |
| `{{ .Status }}` | Статус алерта | `firing` или `resolved` |
| `{{ .Labels.alertname }}` | Имя алерта | `DependencyDown` |
| `{{ .Labels.severity }}` | Уровень severity | `critical` |
| `{{ .Labels.job }}` | Имя сервиса | `order-api` |
| `{{ .Labels.dependency }}` | Имя зависимости | `user-db` |
| `{{ .Labels.type }}` | Тип зависимости | `postgres` |
| `{{ .Labels.namespace }}` | Kubernetes namespace | `production` |
| `{{ .Annotations.summary }}` | Краткое описание | `Зависимость user-db (postgres) недоступна...` |
| `{{ .Annotations.description }}` | Подробное описание | `Все endpoint-ы зависимости...` |
| `{{ .StartsAt }}` | Время начала | `2026-02-12 10:00:00 UTC` |
| `{{ .EndsAt }}` | Время разрешения | `2026-02-12 10:05:00 UTC` |
| `{{ .GeneratorURL }}` | Ссылка на источник | `http://vmalert:8880/...` |

### Пример кастомного шаблона

Для настройки формата сообщений webhook создайте файл шаблона:

```yaml
# В alertmanager.yml добавьте:
templates:
  - '/etc/alertmanager/templates/*.tmpl'
```

Файл шаблона (`dephealth.tmpl`):

```text
{{ define "dephealth.title" -}}
[{{ .Status | toUpper }}] {{ .Labels.alertname }}: {{ .Labels.dependency }} ({{ .Labels.type }})
{{- end }}

{{ define "dephealth.text" -}}
Сервис: {{ .Labels.job }}
Namespace: {{ .Labels.namespace }}
Зависимость: {{ .Labels.dependency }} ({{ .Labels.type }})
Severity: {{ .Labels.severity }}

{{ .Annotations.description }}

{{ if eq .Status "firing" -}}
Начало: {{ .StartsAt.Format "2006-01-02 15:04:05 UTC" }}
{{- else -}}
Разрешено: {{ .EndsAt.Format "2006-01-02 15:04:05 UTC" }}
Длительность: {{ .EndsAt.Sub .StartsAt }}
{{- end }}

Source: {{ .GeneratorURL }}
{{- end }}
```

---

## Silences

Silences временно подавляют нотификации для совпадающих алертов. Используйте их во время планового обслуживания или известных проблем.

### Создание silence через UI

1. Откройте UI Alertmanager: `http://alertmanager:9093/#/silences`
2. Нажмите "New Silence"
3. Добавьте матчеры (например, `dependency = "user-db"`, `namespace = "staging"`)
4. Установите длительность и комментарий
5. Нажмите "Create"

### Создание silence через API

```bash
# Подавить все алерты для зависимости "user-db" в staging на 2 часа
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
    "comment": "Плановое обслуживание PostgreSQL"
  }'
```

### Когда использовать silences

| Ситуация | Silence matcher | Длительность |
| --- | --- | --- |
| Обслуживание БД | `dependency="user-db"` | Длительность maintenance window |
| Деплой staging | `namespace="staging"` | Длительность деплоя + буфер |
| Известный flapping | `alertname="DependencyFlapping", dependency="cache"` | До устранения причины |

---

## Параметры Helm Chart

Параметры Alertmanager в `values.yaml`:

| Параметр | По умолчанию | Описание |
| --- | --- | --- |
| `alertmanager.enabled` | `true` | Включить деплой Alertmanager |
| `alertmanager.image` | `prom/alertmanager` | Образ контейнера |
| `alertmanager.tag` | `v0.28.1` | Тег образа |
| `alertmanager.config.resolveTimeout` | `5m` | Время до автоматического resolved, если нет обновлений |
| `alertmanager.config.groupBy` | `[alertname, namespace, job, dependency]` | Labels для группировки |
| `alertmanager.config.groupWait` | `30s` | Ожидание перед первой нотификацией |
| `alertmanager.config.groupInterval` | `5m` | Минимальный интервал между нотификациями группы |
| `alertmanager.config.repeatInterval` | `4h` | Интервал повтора неразрешённых алертов |
| `alertmanager.config.webhookUrl` | `http://localhost:9094/webhook` | URL webhook по умолчанию |
| `alertmanager.resources.requests.cpu` | `50m` | CPU request |
| `alertmanager.resources.requests.memory` | `32Mi` | Memory request |
| `alertmanager.resources.limits.cpu` | `100m` | CPU limit |
| `alertmanager.resources.limits.memory` | `64Mi` | Memory limit |

### Примеры переопределений

```yaml
# values-production.yaml
alertmanager:
  config:
    groupWait: 15s           # Быстрее начальное уведомление
    repeatInterval: 2h       # Чаще повтор в production
    webhookUrl: "http://alertmanager-bot:8080"  # Telegram-бот
```

---

## Что дальше

- [Правила алертов](alert-rules.ru.md) — подробное описание всех 5 встроенных правил
- [Уменьшение шума](noise-reduction.ru.md) — сценарии и best practices
- [Кастомные правила](custom-rules.ru.md) — написание своих правил поверх dephealth-метрик
