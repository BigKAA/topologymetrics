*[English version](overview.md)*

# Обзор стека мониторинга

> Этот документ описывает архитектуру стека мониторинга dephealth:
> сбор метрик, хранение, алертинг и доставку нотификаций.
> Подробности о правилах алертов — в [Правила алертов](alert-rules.ru.md).
> Grafana дашборды — в [Grafana Dashboards](../grafana-dashboards.ru.md).

---

## Архитектура

Стек мониторинга dephealth состоит из четырёх компонентов, соединённых в pipeline:

```text
┌─────────────────────┐
│  Сервисы (SDK)      │  dephealth SDK экспортирует Prometheus-метрики
│  /metrics endpoint  │  app_dependency_health, app_dependency_latency_seconds
└────────┬────────────┘
         │ scrape (pull)
         ▼
┌─────────────────────┐
│  VictoriaMetrics    │  Time-series база данных (TSDB)
│  или Prometheus     │  Собирает и хранит метрики
│  :8428              │
└────────┬────────────┘
         │ query (PromQL / MetricsQL)
         ▼
┌─────────────────────┐
│  VMAlert            │  Движок оценки алертов
│  или Prometheus     │  Вычисляет правила, генерирует алерты
│  :8880              │
└────────┬────────────┘
         │ push alerts
         ▼
┌─────────────────────┐
│  Alertmanager       │  Маршрутизация и нотификации
│  :9093              │  Группировка, подавление, тишина
└────────┬────────────┘
         │ notify
         ▼
┌─────────────────────┐
│  Каналы             │  Telegram, Slack, webhook,
│                     │  PagerDuty, email, ...
└─────────────────────┘
```

Каждый компонент можно заменить независимо. VictoriaMetrics взаимозаменяем с Prometheus, а VMAlert — с правилами алертов Prometheus. Формат правил и PromQL-выражения совместимы.

## Компоненты

| Компонент | Роль | Порт | Образ | Версия |
| --- | --- | --- | --- | --- |
| VictoriaMetrics | TSDB + сбор метрик | 8428 | `victoriametrics/victoria-metrics` | v1.108.1 |
| VMAlert | Вычисление правил алертов | 8880 | `victoriametrics/vmalert` | v1.108.1 |
| Alertmanager | Маршрутизация и нотификации | 9093 | `prom/alertmanager` | v0.28.1 |
| Grafana | Визуализация (дашборды) | 3000 | `grafana/grafana` | 11.6.0 |

Все компоненты разворачиваются через Helm chart `deploy/helm/dephealth-monitoring/`.

## Сбор метрик (Scraping)

VictoriaMetrics (или Prometheus) периодически забирает метрики с endpoint-ов сервисов.

### Параметры по умолчанию

| Параметр | Значение | Описание |
| --- | --- | --- |
| `scrape_interval` | 15s | Как часто собираются метрики |
| `scrape_timeout` | 10s | Таймаут на один запрос |
| `metrics_path` | `/metrics` | Путь endpoint по умолчанию |
| `retentionPeriod` | 7d | Срок хранения метрик |

### Добавление scrape targets

Targets настраиваются в `values.yaml` в секции `scrapeTargets`:

```yaml
scrapeTargets:
  - jobName: my-service           # Prometheus job_name
    host: my-service.default.svc  # DNS-имя Kubernetes-сервиса
    port: "8080"                  # Порт метрик
    namespace: default            # Kubernetes namespace (добавляется как label)
    service: my-service           # Имя сервиса (добавляется как label)
```

Для Spring Boot приложений — переопределите путь к метрикам:

```yaml
  - jobName: my-java-service
    host: my-java-service.default.svc
    port: "8080"
    metricsPath: /actuator/prometheus  # Spring Boot Actuator
    namespace: default
    service: my-java-service
```

Дополнительные targets (например, для conformance-тестов) добавляются через `extraScrapeTargets`:

```yaml
extraScrapeTargets:
  - jobName: extra-service
    host: extra-service.testing.svc
    port: "8080"
    namespace: testing
    service: extra-service
```

### Labels, добавляемые при scraping

Каждый target автоматически получает labels из конфигурации:

| Label | Источник | Пример |
| --- | --- | --- |
| `job` | поле `jobName` | `my-service` |
| `namespace` | поле `namespace` | `default` |
| `service` | поле `service` | `my-service` |
| `instance` | `host:port` | `my-service.default.svc:8080` |

Эти labels используются в правилах алертов для группировки и маршрутизации (см. [Правила алертов](alert-rules.ru.md)).

## Совместимость VictoriaMetrics и Prometheus

Стек мониторинга dephealth по умолчанию использует VictoriaMetrics, но все конфигурации совместимы с Prometheus.

| Возможность | VictoriaMetrics | Prometheus |
| --- | --- | --- |
| Язык запросов | MetricsQL (надмножество PromQL) | PromQL |
| Формат правил алертов | Идентичный YAML | Идентичный YAML |
| Конфиг scraping | Prometheus-совместимый `scrape.yml` | Нативный `prometheus.yml` |
| Вычисление алертов | VMAlert (отдельный бинарник) | Встроен в Prometheus |
| Alertmanager | Тот же Alertmanager | Тот же Alertmanager |

**Ключевые отличия**:

- MetricsQL расширяет PromQL дополнительными функциями (`median`, `limitk`, `range_median` и др.), но все стандартные функции PromQL работают в обоих системах.
- Правила алертов в этом проекте используют только стандартный PromQL — они работают и с VMAlert, и с Prometheus без изменений.
- VMAlert — отдельный процесс, который запрашивает VictoriaMetrics; Prometheus вычисляет правила встроенно.

### Использование с Prometheus

Чтобы использовать Prometheus вместо VictoriaMetrics:

1. Замените формат конфигурации scraping (минимальные изменения — формат `scrape.yml` идентичен).
2. Загрузите правила алертов через `rule_files` в `prometheus.yml` или используйте CRD `PrometheusRule` (с Prometheus Operator).
3. Настройте `--alertmanager.url` в Prometheus на адрес Alertmanager.

## Вычисление алертов

VMAlert вычисляет правила алертов с настраиваемым интервалом (по умолчанию — каждые 15 секунд) и отправляет сработавшие алерты в Alertmanager.

| Параметр | Значение | Helm Value |
| --- | --- | --- |
| Интервал вычисления | 15s | `vmalert.evaluationInterval` |
| Источник данных | VictoriaMetrics | `http://victoriametrics:8428` |
| Нотификатор | Alertmanager | `http://alertmanager:9093` |
| Путь к правилам | `/rules/*.yml` | ConfigMap `vmalert-rules` |

5 встроенных правил алертов описаны в [Правила алертов](alert-rules.ru.md).

## Быстрый старт

### Развёртывание через Helm

```bash
# Конфигурация по умолчанию
helm install dephealth-monitoring deploy/helm/dephealth-monitoring/

# С переопределениями для homelab (свой registry, storage class, маршрут Grafana)
helm install dephealth-monitoring deploy/helm/dephealth-monitoring/ \
  -f deploy/helm/dephealth-monitoring/values-homelab.yaml
```

### Проверка стека

```bash
# Проверить что все поды запущены
kubectl get pods -n dephealth-monitoring

# Доступ к VMAlert UI (port-forward)
kubectl port-forward -n dephealth-monitoring svc/vmalert 8880:8880
# Открыть http://localhost:8880 — статус правил алертов

# Доступ к Alertmanager UI (port-forward)
kubectl port-forward -n dephealth-monitoring svc/alertmanager 9093:9093
# Открыть http://localhost:9093 — активные алерты и silences

# Доступ к Grafana (port-forward)
kubectl port-forward -n dephealth-monitoring svc/grafana 3000:3000
# Открыть http://localhost:3000 (admin / dephealth)
```

## Справочник Helm Chart

Ключевые параметры `values.yaml` для стека мониторинга:

| Секция | Параметр | По умолчанию | Описание |
| --- | --- | --- | --- |
| `global` | `imageRegistry` | `docker.io` | Docker registry для всех образов |
| `global` | `storageClass` | `""` (default) | StorageClass для PVC |
| `global` | `namespace` | `dephealth-monitoring` | Целевой namespace |
| `victoriametrics` | `enabled` | `true` | Включить VictoriaMetrics |
| `victoriametrics` | `retentionPeriod` | `7d` | Срок хранения метрик |
| `victoriametrics` | `storage` | `2Gi` | Размер PVC для TSDB |
| `vmalert` | `enabled` | `true` | Включить VMAlert |
| `vmalert` | `evaluationInterval` | `15s` | Интервал вычисления правил |
| `alertmanager` | `enabled` | `true` | Включить Alertmanager |
| `alertmanager.config` | `groupBy` | `[alertname, namespace, job, dependency]` | Labels для группировки алертов |
| `alertmanager.config` | `groupWait` | `30s` | Ожидание перед отправкой группы |
| `alertmanager.config` | `repeatInterval` | `4h` | Интервал повтора для неразрешённых алертов |
| `alertmanager.config` | `webhookUrl` | `http://localhost:9094/webhook` | Endpoint нотификаций по умолчанию |

Полный справочник: [`deploy/helm/dephealth-monitoring/values.yaml`](../../deploy/helm/dephealth-monitoring/values.yaml)

## Что дальше

- [Правила алертов](alert-rules.ru.md) — подробное описание всех 5 встроенных правил
- [Уменьшение шума](noise-reduction.ru.md) — сценарии, подавление и best practices
- [Конфигурация Alertmanager](alertmanager.ru.md) — маршрутизация, receivers (Telegram, webhook), шаблоны нотификаций
- [Кастомные правила](custom-rules.ru.md) — написание своих правил алертов поверх dephealth-метрик
