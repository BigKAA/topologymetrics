*[English version](alert-rules.md)*

# Правила алертов

> Этот документ содержит подробное описание всех 5 встроенных правил алертов,
> поставляемых со стеком мониторинга dephealth.
> Обзор стека мониторинга — в [Обзор](overview.ru.md).
> Техники уменьшения шума — в [Уменьшение шума](noise-reduction.ru.md).

---

## Сводная таблица

| Алерт | Severity | For | Условие срабатывания |
| --- | --- | --- | --- |
| [DependencyDown](#dependencydown) | critical | 1m | Все endpoint-ы зависимости недоступны |
| [DependencyDegraded](#dependencydegraded) | warning | 2m | Часть endpoint-ов упала, часть работает |
| [DependencyHighLatency](#dependencyhighlatency) | warning | 5m | P99 латентность превышает 1 секунду |
| [DependencyFlapping](#dependencyflapping) | info | 0m | Статус менялся более 4 раз за 15 минут |
| [DependencyAbsent](#dependencyabsent) | warning | 5m | Ни один сервис не экспортирует `app_dependency_health` |

Все правила используют стандартный PromQL и совместимы с VMAlert и Prometheus.

---

<a id="dependencydown"></a>

## 1. DependencyDown

**Полный отказ**: все endpoint-ы зависимости недоступны.

### Правило

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
    summary: "Зависимость {{ $labels.dependency }} ({{ $labels.type }}) недоступна в сервисе {{ $labels.job }}"
    description: "Все endpoint-ы зависимости {{ $labels.dependency }} типа {{ $labels.type }} в сервисе {{ $labels.job }} (namespace: {{ $labels.namespace }}) недоступны более 1 минуты."
```

### Разбор PromQL

```text
min by (job, namespace, dependency, type) (
  app_dependency_health
) == 0
```

| Фрагмент | Значение |
| --- | --- |
| `app_dependency_health` | Gauge-метрика: `1` = доступен, `0` = недоступен. Один time series на каждый endpoint (host:port). |
| `min by (job, namespace, dependency, type)` | Берёт **минимальное** значение среди всех endpoint-ов одной зависимости. Если хоть один endpoint здоров (`1`), минимум = `1` и алерт НЕ срабатывает. |
| `== 0` | Алерт срабатывает только когда **все** endpoint-ы упали (минимум = 0). |

**Почему `min`, а не `max` или `avg`?**

- `min == 0` означает, что каждый endpoint недоступен — полный отказ.
- `max == 0` даст тот же результат (если максимум = 0, все = 0), но `min` яснее передаёт намерение.
- `avg` вводит в заблуждение: `avg == 0.5` может означать 1 из 2 endpoint-ов упал — это DependencyDegraded, не DependencyDown.

### Почему `for: 1m`

- **Слишком коротко (0s–15s)**: сетевые сбои, перезапуски pod-ов и распространение DNS вызывают кратковременные отказы. Мгновенный алерт создаёт шум.
- **1 минута**: достаточно для фильтрации единичных сбоев scrape (интервал scrape = 15s, 1m ≈ 4 scrape). Если зависимость всё ещё недоступна после 4 проверок — это реальная проблема.
- **Слишком долго (5m+)**: при полном отказе 5 минут тишины — неприемлемо. Оператор должен узнать быстро.

### Почему `severity: critical`

Полный отказ зависимости обычно означает, что сервис не может выполнять свою основную функцию. Требуется немедленное внимание оператора.

### Пример алерта

При срабатывании этого правила Alertmanager получает алерт с labels:

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

### Взаимодействие с другими правилами

При срабатывании DependencyDown [inhibit rules](noise-reduction.ru.md#inhibit-rules) в Alertmanager подавляют:

- DependencyDegraded (та же зависимость)
- DependencyHighLatency (та же зависимость)
- DependencyFlapping (та же зависимость)

Это предотвращает шторм алертов. См. [Уменьшение шума: Сценарий 1](noise-reduction.ru.md#сценарий-1-бд-master-упал).

---

<a id="dependencydegraded"></a>

## 2. DependencyDegraded

**Частичный отказ**: часть endpoint-ов зависимости недоступна, но хотя бы один работает.

### Правило

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
    summary: "Зависимость {{ $labels.dependency }} ({{ $labels.type }}) частично деградирована в сервисе {{ $labels.job }}"
    description: "Часть endpoint-ов зависимости {{ $labels.dependency }} типа {{ $labels.type }} в сервисе {{ $labels.job }} (namespace: {{ $labels.namespace }}) недоступна более 2 минут."
```

### Разбор PromQL

```text
(
  count by (...) (app_dependency_health == 0) > 0   -- хотя бы один endpoint DOWN
)
and
(
  count by (...) (app_dependency_health == 1) > 0   -- хотя бы один endpoint UP
)
```

| Фрагмент | Значение |
| --- | --- |
| `app_dependency_health == 0` | Выбирает только нездоровые endpoint-ы (значение = 0). |
| `count by (...) > 0` | Подсчитывает нездоровые endpoint-ы. `> 0` = хотя бы один существует. |
| `and` | Оба условия должны быть истинны одновременно. |
| Второй `count ... == 1 ... > 0` | Хотя бы один endpoint ещё здоров. |

**Почему такая структура?**

Оператор `and` гарантирует, что алерт срабатывает **только** при смешанном состоянии. Если все endpoint-ы упали — первое условие истинно, но второе ложно → DependencyDegraded НЕ срабатывает (вместо него срабатывает DependencyDown).

### Почему `for: 2m`

- **Дольше чем DependencyDown (1m)**: частичная деградация часто временна — перезапуск одной реплики, rolling update, кратковременная сетевая проблема.
- **2 минуты**: даёт время Kubernetes перепланировать pod-ы или завершить rolling restart.
- **Не слишком долго (5m+)**: если деградация сохраняется, оператор должен знать.

### Почему `severity: warning`

Сервис частично функционирует. Это важно, но не аварийно — оставшиеся здоровые endpoint-ы обрабатывают трафик. Warning = "разберитесь в ближайшее время, не немедленно."

### Когда это правило актуально

Правило имеет смысл только для зависимостей с **несколькими endpoint-ами** (PostgreSQL primary + replica, кластер Redis, несколько HTTP-инстансов). Для зависимостей с одним endpoint состояние всегда бинарное: DependencyDown или здоров.

### Пример алерта

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

### Взаимодействие с другими правилами

- **Подавляется** DependencyDown (inhibit rule: если все endpoint-ы упали, DependencyDown приоритетнее).
- **Подавляет** DependencyFlapping (inhibit rule: деградация уже указывает на нестабильность).

См. [Уменьшение шума: Сценарий 2](noise-reduction.ru.md#сценарий-2-одна-из-трёх-реплик-упала).

---

<a id="dependencyhighlatency"></a>

## 3. DependencyHighLatency

**Деградация производительности**: зависимость отвечает, но слишком медленно.

### Правило

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
    summary: "Высокая латентность зависимости {{ $labels.dependency }} ({{ $labels.type }}) в сервисе {{ $labels.job }}"
    description: "P99 латентность проверки зависимости {{ $labels.dependency }} превышает 1 секунду в течение 5 минут."
```

### Разбор PromQL

```text
histogram_quantile(0.99,
  rate(app_dependency_latency_seconds_bucket[5m])
) > 1
```

| Фрагмент | Значение |
| --- | --- |
| `app_dependency_latency_seconds_bucket` | Гистограмма: распределение латентности health check по бакетам. |
| `rate(...[5m])` | Скорость прироста счётчиков бакетов за 5 минут. Сглаживает всплески. |
| `histogram_quantile(0.99, ...)` | Вычисляет 99-й перцентиль (P99). 99% проверок завершаются быстрее этого значения. |
| `> 1` | Порог: 1 секунда. Если P99 превышает 1s — зависимость слишком медленна. |

**Почему P99, а не среднее?**

- Среднее скрывает выбросы: если 99% проверок занимают 10ms, а 1% — 30s, среднее ~300ms — выглядит нормально.
- P99 ловит хвост распределения. Если P99 > 1s — есть реальная проблема производительности.

**Почему `rate` за 5 минут?**

- Короткие окна (1m) создают шум — одна медленная проверка может резко поднять перцентиль.
- 5 минут даёт стабильный сигнал, оставаясь достаточно отзывчивым.

### Почему `for: 5m`

- **Латентность флуктуирует естественно**: прогрев connection pool, GC-паузы, сетевые перегрузки — всё это вызывает кратковременные всплески.
- **5 минут `for`** поверх 5-минутного окна `rate` означает, что P99 должен быть стабильно высоким ~10 минут суммарно.
- Намеренно консервативно для избежания ложных срабатываний.

### Почему `severity: warning`

Зависимость **доступна** (health = 1), но медленна. Это проблема производительности, не отказ. Warning = "разберитесь, но не будите в 3 часа ночи."

### Настройка порога

Порог по умолчанию (1 секунда) может не подходить для всех типов зависимостей. См. [Кастомные правила](custom-rules.ru.md) для примеров порогов по типам (500ms для PostgreSQL, 2s для внешних HTTP API).

### Пример алерта

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

### Взаимодействие с другими правилами

- **Подавляется** DependencyDown (inhibit rule: если зависимость полностью недоступна, латентность неактуальна).
- **Независим** от DependencyDegraded — латентность и частичный отказ — ортогональные проблемы.

См. [Уменьшение шума: Сценарий 3](noise-reduction.ru.md#сценарий-3-http-сервис-отвечает-медленно).

---

<a id="dependencyflapping"></a>

## 4. DependencyFlapping

**Нестабильность**: статус зависимости постоянно переключается между здоровым и нездоровым.

### Правило

```yaml
- alert: DependencyFlapping
  expr: |
    changes(app_dependency_health[15m]) > 4
  for: 0m
  labels:
    severity: info
  annotations:
    summary: "Зависимость {{ $labels.dependency }} ({{ $labels.type }}) нестабильна в сервисе {{ $labels.job }}"
    description: "Статус зависимости {{ $labels.dependency }} менялся {{ $value }} раз за последние 15 минут."
```

### Разбор PromQL

```text
changes(app_dependency_health[15m]) > 4
```

| Фрагмент | Значение |
| --- | --- |
| `app_dependency_health` | Gauge: `1` или `0`. |
| `changes(...[15m])` | Считает количество изменений значения за 15 минут. Каждый переход 0→1 или 1→0 — одно изменение. |
| `> 4` | Более 4 изменений за 15 минут — примерно одно переключение каждые 3.75 минуты или чаще. |

**Почему порог 4?**

- 1–2 изменения за 15 минут — нормально: кратковременный сетевой сбой, перезапуск pod-а.
- 4+ изменений означает нестабильность — зависимость постоянно падает и поднимается. Такой паттерн обычно указывает на глубокую проблему (исчерпание ресурсов, нестабильность сети, некорректная конфигурация health check).

### Почему `for: 0m`

- Функция `changes()` уже смотрит на 15-минутное окно — временное сглаживание встроено в выражение.
- Добавление `for` поверх задержит сигнал ещё больше, что противоречит цели обнаружения нестабильности в реальном времени.

### Почему `severity: info`

Flapping — это **диагностический сигнал**, а не призыв к действию:

- Говорит оператору "что-то нестабильно — найдите причину."
- По умолчанию `info`-алерты направляются в null receiver (только в UI Alertmanager, без нотификаций).
- Если flapping важен в вашем окружении — измените receiver в конфигурации Alertmanager.

### Пример алерта

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

`$value` в описании показывает фактическое количество изменений (например, "статус менялся 7 раз").

### Взаимодействие с другими правилами

- **Подавляется** DependencyDown (если зависимость полностью упала, flapping не является проблемой).
- **Подавляется** DependencyDegraded (частичный отказ уже указывает на проблему).

См. [Уменьшение шума: Сценарий 4](noise-reduction.ru.md#сценарий-4-нестабильная-сеть-flapping).

---

<a id="dependencyabsent"></a>

## 5. DependencyAbsent

**Отсутствие метрик**: ни один сервис не экспортирует метрику `app_dependency_health`.

### Правило

```yaml
- alert: DependencyAbsent
  expr: |
    absent(app_dependency_health{job=~".+"})
  for: 5m
  labels:
    severity: warning
  annotations:
    summary: "Метрики dephealth отсутствуют"
    description: "Ни один сервис не экспортирует метрику app_dependency_health в течение 5 минут."
```

### Разбор PromQL

```text
absent(app_dependency_health{job=~".+"})
```

| Фрагмент | Значение |
| --- | --- |
| `app_dependency_health` | Базовая метрика здоровья. |
| `{job=~".+"}` | Label matcher: требует хотя бы один непустой label `job`. Предотвращает срабатывание на пустой TSDB (до деплоя любого сервиса). |
| `absent(...)` | Возвращает `1`, если вектор пуст (нет подходящих time series), иначе ничего не возвращает. |

**Зачем `{job=~".+"}`?**

Без этого фильтра `absent(app_dependency_health)` сработает сразу после деплоя стека мониторинга — до того как хоть один сервис начнёт экспортировать метрики. Label matcher гарантирует, что алерт срабатывает только когда ранее существовавшие метрики пропали.

### Почему `for: 5m`

- **Перезапуск сервисов**: во время rolling update есть короткий промежуток, когда старый pod завершён, а новый ещё не готов. 5 минут покрывает типичное время перезапуска.
- **Stale series**: VictoriaMetrics/Prometheus хранят устаревшие series настраиваемое время. 5 минут достаточно для обнаружения отсутствия scraper-ом.
- **CI/CD pipelines**: сервисы могут быть временно недоступны во время деплоя.

### Почему `severity: warning`

Отсутствие метрик обычно означает проблему с деплоем или scraping, а не с зависимостями. Требует расследования, но не критично.

### Типичные причины

1. **Неправильная конфигурация scrape target**: ошибка в host, port или namespace в `values.yaml`.
2. **Сервис не запущен**: pod-ы в CrashLoopBackOff или не развёрнуты.
3. **Изменился endpoint метрик**: путь Spring Boot Actuator изменился, или SDK был удалён.
4. **Сетевая политика**: firewall или NetworkPolicy блокирует scraping.

### Пример алерта

```json
{
  "alertname": "DependencyAbsent",
  "severity": "warning"
}
```

Обратите внимание: у этого алерта нет labels `job`, `dependency` или `type`, потому что нет подходящих time series для их извлечения.

### Взаимодействие с другими правилами

- **Независим**: этот алерт срабатывает когда данных нет вообще — другие правила не могут сработать без данных.
- **Не подавляется** никакими inhibit rules.

См. [Уменьшение шума: Сценарий 5](noise-reduction.ru.md#сценарий-5-перезапуск-сервиса-или-деплой).

---

## Размещение правил

### VMAlert (VictoriaMetrics)

Правила хранятся в ConfigMap, подмонтированном к VMAlert:

```text
ConfigMap: vmalert-rules
  └── dephealth.yml     ← все 5 правил
      └── смонтирован в /rules/dephealth.yml
```

VMAlert загружает правила из `/rules/*.yml` и вычисляет их с настроенным интервалом.

### Prometheus

Для Prometheus те же правила загружаются через:

**Вариант 1** — `rule_files` в `prometheus.yml`:

```yaml
rule_files:
  - /etc/prometheus/rules/dephealth.yml
```

**Вариант 2** — CRD `PrometheusRule` (с Prometheus Operator):

```yaml
apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  name: dephealth-rules
  labels:
    release: prometheus   # должен совпадать с селектором Prometheus Operator
spec:
  groups:
    - name: dephealth.rules
      rules:
        # ... те же правила ...
```

---

## Что дальше

- [Уменьшение шума](noise-reduction.ru.md) — как правила взаимодействуют, подавление и реальные сценарии
- [Конфигурация Alertmanager](alertmanager.ru.md) — маршрутизация, receivers, доставка нотификаций
- [Кастомные правила](custom-rules.ru.md) — написание своих правил поверх dephealth-метрик
