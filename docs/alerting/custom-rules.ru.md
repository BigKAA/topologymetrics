*[English version](custom-rules.md)*

# Кастомные правила алертов

> Этот документ объясняет, как писать свои правила алертов поверх dephealth-метрик.
> Здесь шаблоны, готовые примеры и инструкции по интеграции для VMAlert и Prometheus.
> Встроенные правила — в [Правила алертов](alert-rules.ru.md).
> Noise reduction при добавлении правил — в [Уменьшение шума](noise-reduction.ru.md).

---

## Когда нужны кастомные правила

5 встроенных правил покрывают типичные сценарии. Кастомные правила нужны когда:

- **Разные пороги** — порог латентности 500ms для БД vs 2s для внешних HTTP API.
- **Только critical** — алерт только для зависимостей с `critical=yes`.
- **SLO-мониторинг** — алерт при падении доступности ниже целевого процента (например, 99%).
- **Политики по namespace** — строже для production, мягче для staging.
- **Маршрутизация по командам** — labels `team` для направления в конкретные receivers.
- **Полный vs частичный отказ** — различение "все endpoint-ы упали" от "1 из N упал" (встроенный DependencyDown использует `min`, который срабатывает при падении любого endpoint).

---

## Шаблон правила

Каждое правило следует этой структуре:

```yaml
groups:
  - name: custom.dephealth.rules
    rules:
      - alert: MyCustomAlert          # Имя (PascalCase, описательное)
        expr: |                        # PromQL-выражение
          <promql_expression>
        for: <duration>                # Как долго условие должно быть истинным
        labels:
          severity: <critical|warning|info>
          <custom_label>: <value>      # Опционально: team, environment и т.д.
        annotations:
          summary: "<краткое описание с {{ $labels.dependency }}>"
          description: "<подробное описание>"
```

### Соглашения об именовании

- **Имя алерта**: `PascalCase`, описательное. Добавляйте префикс для отличия от встроенных (например, `Custom` или имя команды).
- **Имя группы**: `custom.dephealth.rules` или `<team>.dephealth.rules` для отделения от встроенной `dephealth.rules`.
- **Labels**: добавляйте `team`, `environment` или `tier` для маршрутизации и фильтрации.

### Доступные метрики

| Метрика | Тип | Labels | Описание |
| --- | --- | --- | --- |
| `app_dependency_health` | Gauge (0/1) | `name`, `dependency`, `type`, `host`, `port`, `critical` | Текущий статус здоровья |
| `app_dependency_latency_seconds` | Histogram | `name`, `dependency`, `type`, `host`, `port`, `critical` | Распределение латентности health check |
| `app_dependency_status` | Gauge (enum) | `name`, `dependency`, `type`, `host`, `port`, `critical`, `status` | Категория статуса — 8 серий на endpoint, ровно одна = 1 |
| `app_dependency_status_detail` | Gauge (info) | `name`, `dependency`, `type`, `host`, `port`, `critical`, `detail` | Детальная причина — 1 серия на endpoint |

Label `name` идентифицирует приложение, экспортирующее метрики. Label `dependency` идентифицирует целевую зависимость. Полные детали — в [Контракт метрик](../../spec/metric-contract.ru.md).

### Доступные значения labels

| Label | Возможные значения |
| --- | --- |
| `type` | `http`, `grpc`, `tcp`, `postgres`, `mysql`, `redis`, `amqp`, `kafka` |
| `critical` | `yes`, `no` |
| `status` | `ok`, `timeout`, `connection_error`, `dns_error`, `auth_error`, `tls_error`, `unhealthy`, `error` |
| `detail` | `ok`, `timeout`, `connection_refused`, `dns_error`, `auth_error`, `tls_error`, `http_NNN`, `grpc_not_serving`, `grpc_unknown`, `unhealthy`, `no_brokers`, `error` |

---

## Пример 1: Только critical-зависимости

Алерт только для зависимостей, помеченных как critical — для высокоприоритетного канала.

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
    summary: "CRITICAL зависимость {{ $labels.dependency }} ({{ $labels.type }}) недоступна в {{ $labels.job }}"
    description: "Critical-зависимость {{ $labels.dependency }} в сервисе {{ $labels.job }} (namespace: {{ $labels.namespace }}) недоступна более 30 секунд. Зависимость помечена critical=yes — сервис не может функционировать без неё."
```

**Ключевые отличия от встроенного DependencyDown**:

| Параметр | Встроенный | Это правило |
| --- | --- | --- |
| Фильтр | Все зависимости | Только `critical="yes"` |
| `for` | 1m | 30s (быстрее — critical-зависимости требуют быстрой реакции) |
| Доп. label | — | `tier: platform` (для маршрутизации) |

**Маршрутизация**: добавьте маршрут в Alertmanager для `tier=platform`:

```yaml
routes:
  - matchers:
      - tier = "platform"
    receiver: pagerduty-platform
    group_wait: 5s
```

**Inhibit rule**: рассмотрите подавление встроенного DependencyDown при срабатывании этого правила:

```yaml
inhibit_rules:
  - source_matchers:
      - alertname = "CriticalDependencyDown"
    target_matchers:
      - alertname = "DependencyDown"
    equal: ['job', 'namespace', 'dependency']
```

---

## Пример 2: Пороги латентности по типам

Разные типы зависимостей имеют разную допустимую латентность. БД должны отвечать за 500ms, внешние HTTP API — до 2 секунд.

### Латентность БД (строгий порог)

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
    summary: "БД {{ $labels.dependency }} P99 латентность > 500ms в {{ $labels.job }}"
    description: "P99 латентность {{ $labels.dependency }} ({{ $labels.type }}) в сервисе {{ $labels.job }} превышает 500ms более 3 минут. Текущий P99: {{ $value | printf \"%.3f\" }}s."
```

### Латентность кэша (строгий порог)

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
    summary: "Кэш {{ $labels.dependency }} P99 латентность > 100ms в {{ $labels.job }}"
    description: "P99 латентность Redis-кэша {{ $labels.dependency }} в сервисе {{ $labels.job }} превышает 100ms более 3 минут. Текущий P99: {{ $value | printf \"%.3f\" }}s."
```

### Латентность внешних HTTP API (мягкий порог)

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
    summary: "Внешний API {{ $labels.dependency }} P99 латентность > 2s в {{ $labels.job }}"
    description: "P99 латентность HTTP-зависимости {{ $labels.dependency }} в сервисе {{ $labels.job }} превышает 2 секунды более 5 минут. Текущий P99: {{ $value | printf \"%.3f\" }}s."
```

**Сравнение порогов**:

| Правило | Фильтр типа | Порог P99 | `for` |
| --- | --- | --- | --- |
| Встроенный DependencyHighLatency | Все типы | 1s | 5m |
| DatabaseHighLatency | postgres, mysql | 500ms | 3m |
| CacheHighLatency | redis | 100ms | 3m |
| ExternalApiHighLatency | http | 2s | 5m |

**Inhibit rules**: рассмотрите подавление встроенного DependencyHighLatency:

```yaml
inhibit_rules:
  - source_matchers:
      - alertname =~ "DatabaseHighLatency|CacheHighLatency|ExternalApiHighLatency"
    target_matchers:
      - alertname = "DependencyHighLatency"
    equal: ['job', 'namespace', 'dependency']
```

---

## Пример 3: SLO-мониторинг доступности

Алерт при падении доступности зависимости ниже целевого процента за временное окно.

### Доступность < 99% за 1 час

```yaml
- alert: DependencyBelowSLO
  expr: |
    avg_over_time(app_dependency_health[1h]) < 0.99
  for: 10m
  labels:
    severity: warning
  annotations:
    summary: "Нарушение SLO: {{ $labels.dependency }} доступность < 99% в {{ $labels.job }}"
    description: "Зависимость {{ $labels.dependency }} ({{ $labels.type }}) в сервисе {{ $labels.job }} имеет {{ $value | printf \"%.2f\" | humanizePercentage }} доступность за последний час. Целевой SLO: 99%."
```

**Как это работает**:

- `avg_over_time(health[1h])` вычисляет среднее из 0 и 1 за 1 час. Если зависимость была недоступна 36 секунд за час, среднее = (3600-36)/3600 = 0.99 — ровно на пороге.
- `for: 10m` добавляет фильтр стабильности — SLO должен быть нарушен 10 минут подряд.

### Критический SLO: доступность < 95% за 30 минут

```yaml
- alert: DependencySLOCritical
  expr: |
    avg_over_time(app_dependency_health[30m]) < 0.95
  for: 5m
  labels:
    severity: critical
  annotations:
    summary: "Критическое нарушение SLO: {{ $labels.dependency }} доступность < 95% в {{ $labels.job }}"
    description: "Зависимость {{ $labels.dependency }} ({{ $labels.type }}) в сервисе {{ $labels.job }} имеет {{ $value | printf \"%.2f\" | humanizePercentage }} доступность за последние 30 минут. Серьёзная и устойчивая проблема."
```

**Сводка SLO**:

| Правило | Окно | Порог | Severity | Значение |
| --- | --- | --- | --- | --- |
| DependencyBelowSLO | 1h | < 99% | warning | Незначительное нарушение SLO |
| DependencySLOCritical | 30m | < 95% | critical | Серьёзная деградация, действовать немедленно |

---

## Пример 4: Алерты по namespace

Разные окружения требуют разных политик.

### Production: строгий

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
    summary: "[PROD] {{ $labels.dependency }} ({{ $labels.type }}) недоступен в {{ $labels.job }}"
    description: "Production-зависимость {{ $labels.dependency }} в сервисе {{ $labels.job }} недоступна более 30 секунд."
```

### Staging: мягкий

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
    summary: "[STAGING] {{ $labels.dependency }} ({{ $labels.type }}) недоступен в {{ $labels.job }}"
    description: "Staging-зависимость {{ $labels.dependency }} в сервисе {{ $labels.job }} недоступна более 5 минут."
```

**Маршрутизация по окружению**:

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

## Пример 5: Полный vs частичный отказ

Встроенный DependencyDown использует `min`, который срабатывает при падении **любого** endpoint. Для различения:

### Все endpoint-ы упали (полный отказ)

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
    summary: "ВСЕ endpoint-ы {{ $labels.dependency }} ({{ $labels.type }}) упали в {{ $labels.job }}"
    description: "Каждый endpoint зависимости {{ $labels.dependency }} в сервисе {{ $labels.job }} недоступен."
```

**Как работает**: `общее количество - количество нулей == 0` означает, что все endpoint-ы на 0. Отличается от `min == 0`, который срабатывает при падении даже одного.

### Большинство endpoint-ов упало

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
    summary: "Более 50% endpoint-ов {{ $labels.dependency }} ({{ $labels.type }}) упали в {{ $labels.job }}"
    description: "{{ $value | printf \"%.0f\" | humanizePercentage }} endpoint-ов зависимости {{ $labels.dependency }} в сервисе {{ $labels.job }} недоступны."
```

---

## Пример 6: Алерты по категории статуса

Метрика `app_dependency_status` позволяет точно алертить на основе **причины** недоступности зависимости, а не только факта недоступности.

### Ошибки аутентификации

Алерт при отказе зависимости из-за неверных credentials — обычно требует немедленных действий (ротация секретов, исправление конфига).

```yaml
- alert: DependencyAuthError
  expr: |
    app_dependency_status{status="auth_error"} == 1
  for: 1m
  labels:
    severity: critical
    category: security
  annotations:
    summary: "Ошибка аутентификации {{ $labels.dependency }} ({{ $labels.type }}) в {{ $labels.job }}"
    description: "Зависимость {{ $labels.dependency }} в сервисе {{ $labels.job }} недоступна из-за ошибки аутентификации. Проверьте credentials и секреты."
```

### Ошибки DNS

DNS-ошибки часто указывают на инфраструктурные проблемы (CoreDNS, удалённый сервис, опечатка в hostname).

```yaml
- alert: DependencyDnsError
  expr: |
    app_dependency_status{status="dns_error"} == 1
  for: 2m
  labels:
    severity: warning
    category: infrastructure
  annotations:
    summary: "DNS-ошибка для {{ $labels.dependency }} в {{ $labels.job }}"
    description: "Невозможно разрешить hostname зависимости {{ $labels.dependency }} ({{ $labels.type }}) в сервисе {{ $labels.job }}. Проверьте DNS-конфигурацию и существование целевого сервиса."
```

### Ошибки TLS-сертификатов

TLS-ошибки указывают на просроченные или неправильно настроенные сертификаты.

```yaml
- alert: DependencyTlsError
  expr: |
    app_dependency_status{status="tls_error"} == 1
  for: 1m
  labels:
    severity: critical
    category: security
  annotations:
    summary: "TLS-ошибка для {{ $labels.dependency }} ({{ $labels.type }}) в {{ $labels.job }}"
    description: "TLS handshake не удался для зависимости {{ $labels.dependency }} в сервисе {{ $labels.job }}. Проверьте валидность сертификата и цепочку доверия."
```

### Конкретные HTTP-коды

Используйте `app_dependency_status_detail` для алертов на конкретные причины отказа.

```yaml
- alert: DependencyHttp503
  expr: |
    app_dependency_status_detail{detail="http_503"} == 1
  for: 2m
  labels:
    severity: warning
  annotations:
    summary: "{{ $labels.dependency }} возвращает 503 в {{ $labels.job }}"
    description: "HTTP-зависимость {{ $labels.dependency }} в сервисе {{ $labels.job }} возвращает 503 Service Unavailable более 2 минут."
```

### Распределение категорий статуса

Recording rule для отслеживания распределения категорий статуса по всем зависимостям:

```yaml
- record: dephealth:status_distribution:count
  expr: |
    count by (status) (app_dependency_status == 1)
```

---

## Recording Rules

Recording rules предвычисляют тяжёлые запросы и сохраняют результаты как новые time series. Используйте их, когда одно и то же выражение используется в нескольких алертах или дашбордах.

### Доступность за 1 час

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

### Использование recording rules в алертах

```yaml
# Вместо повторного вычисления avg_over_time при каждой оценке
- alert: DependencyBelowSLO
  expr: |
    dephealth:dependency_availability:avg1h < 0.99
  for: 10m
  labels:
    severity: warning
```

### Использование recording rules в Grafana

Recording rules создают стандартные time series для Grafana:

```promql
# Панель дашборда: heatmap доступности
dephealth:dependency_availability:avg1h{namespace="production"}

# Панель дашборда: обзор латентности
dephealth:dependency_latency_p99:5m{type="postgres"}
```

### Соглашение об именовании

Следуйте конвенции Prometheus для recording rules:

```text
<namespace>:<metric_name>:<aggregation_window>
```

- `dephealth:` — prefix namespace для всех recording rules dephealth.
- `dependency_availability` — что измеряется.
- `avg1h` — тип агрегации и окно.

---

## Добавление правил в Helm chart

### Текущая структура

Встроенные правила находятся в ConfigMap:

```text
ConfigMap: vmalert-rules
  └── dephealth.yml (5 встроенных правил)
```

### Вариант 1: Дополнительный ConfigMap (вручную)

Создайте отдельный ConfigMap для кастомных правил и подмонтируйте к VMAlert:

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

Добавьте volume mount к Deployment VMAlert:

```yaml
volumeMounts:
  - name: rules
    mountPath: /rules
    readOnly: true
  - name: custom-rules          # Добавить
    mountPath: /custom-rules
    readOnly: true
volumes:
  - name: rules
    configMap:
      name: vmalert-rules
  - name: custom-rules          # Добавить
    configMap:
      name: custom-alert-rules
```

Обновите аргументы VMAlert:

```yaml
args:
  - -rule=/rules/*.yml
  - -rule=/custom-rules/*.yml   # Добавить
```

### Вариант 2: PrometheusRule CRD (Prometheus Operator)

При использовании Prometheus Operator создайте ресурс PrometheusRule:

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
            summary: "CRITICAL зависимость {{ $labels.dependency }} недоступна"
```

---

## Чек-лист для кастомных правил

Перед деплоем кастомного правила проверьте:

- [ ] **PromQL валиден** — протестируйте выражение в UI VictoriaMetrics или Prometheus.
- [ ] **`for` duration подходит** — не слишком коротко (шум), не слишком долго (задержка обнаружения).
- [ ] **Severity соответствует влиянию** — critical для отказов, warning для деградации, info для диагностики.
- [ ] **Labels для маршрутизации** — добавьте `team`, `environment` или `tier` если нужна кастомная маршрутизация.
- [ ] **Inhibit rules обновлены** — добавьте inhibit rules для предотвращения перекрытия со встроенными.
- [ ] **Annotations полезны** — включите имя зависимости, тип, сервис и действенную информацию.
- [ ] **Recording rules для переиспользования** — если одно выражение используется в нескольких местах, создайте recording rule.

---

## Что дальше

- [Правила алертов](alert-rules.ru.md) — 5 встроенных правил (база для кастомизации)
- [Уменьшение шума](noise-reduction.ru.md) — убедитесь, что кастомные правила не добавляют шум
- [Конфигурация Alertmanager](alertmanager.ru.md) — маршрутизация кастомных правил в нужные receivers
- [Обзор](overview.ru.md) — архитектура стека мониторинга
