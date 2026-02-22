*[English version](metrics.md)*

# Метрики Prometheus

SDK dephealth экспортирует четыре метрики Prometheus для каждого
мониторируемого эндпоинта зависимости. Руководство описывает каждую
метрику, её метки и приводит примеры PromQL.

## Обзор метрик

| Метрика | Тип | Описание |
| --- | --- | --- |
| `app_dependency_health` | Gauge | Статус здоровья: `1` = здоров, `0` = нездоров |
| `app_dependency_latency_seconds` | Histogram | Задержка проверки в секундах |
| `app_dependency_status` | Gauge (enum) | Категория статуса: 8 серий на эндпоинт |
| `app_dependency_status_detail` | Gauge (info) | Детальная причина сбоя |

## Метки

Все четыре метрики имеют общий набор меток:

| Метка | Источник | Описание |
| --- | --- | --- |
| `name` | Первый аргумент `New()` | Имя приложения |
| `group` | Второй аргумент `New()` | Логическая группа |
| `dependency` | Имя зависимости | Идентификатор зависимости |
| `type` | Тип чекера | `http`, `grpc`, `tcp`, `postgres`, `mysql`, `redis`, `amqp`, `kafka` |
| `host` | `FromURL`/`FromParams` | Хост зависимости |
| `port` | `FromURL`/`FromParams` | Порт зависимости |
| `critical` | `Critical()` | `yes` или `no` |

Пользовательские метки, добавленные через `WithLabel()`, появляются
после `critical` в алфавитном порядке.

Дополнительные метки:

- `app_dependency_status` имеет `status` — одна из 8 категорий статуса
- `app_dependency_status_detail` имеет `detail` — конкретная причина сбоя

## app_dependency_health

Простой бинарный индикатор здоровья.

- Значение `1` — зависимость здорова (последняя проверка успешна)
- Значение `0` — зависимость нездорова (последняя проверка неуспешна)

```text
app_dependency_health{name="my-service",group="my-team",dependency="postgres-main",type="postgres",host="pg.svc",port="5432",critical="yes"} 1
app_dependency_health{name="my-service",group="my-team",dependency="redis-cache",type="redis",host="redis.svc",port="6379",critical="no"} 0
```

### Примеры PromQL

```promql
# Все нездоровые зависимости
app_dependency_health == 0

# Нездоровые критичные зависимости
app_dependency_health{critical="yes"} == 0

# Статус здоровья конкретной зависимости
app_dependency_health{name="my-service",dependency="postgres-main"}
```

## app_dependency_latency_seconds

Гистограмма задержки проверки.

Бакеты: `0.001`, `0.005`, `0.01`, `0.05`, `0.1`, `0.5`, `1.0`, `5.0` секунд.

```text
app_dependency_latency_seconds_bucket{...,le="0.001"} 0
app_dependency_latency_seconds_bucket{...,le="0.005"} 42
app_dependency_latency_seconds_bucket{...,le="0.01"} 45
app_dependency_latency_seconds_bucket{...,le="0.05"} 48
app_dependency_latency_seconds_bucket{...,le="0.1"} 48
app_dependency_latency_seconds_bucket{...,le="0.5"} 48
app_dependency_latency_seconds_bucket{...,le="1"} 48
app_dependency_latency_seconds_bucket{...,le="5"} 48
app_dependency_latency_seconds_bucket{...,le="+Inf"} 48
app_dependency_latency_seconds_sum{...} 0.192
app_dependency_latency_seconds_count{...} 48
```

### Примеры PromQL

```promql
# P95 задержка за последние 5 минут
histogram_quantile(0.95,
  rate(app_dependency_latency_seconds_bucket[5m])
)

# Средняя задержка
rate(app_dependency_latency_seconds_sum[5m])
  / rate(app_dependency_latency_seconds_count[5m])

# P99 для конкретной зависимости
histogram_quantile(0.99,
  rate(app_dependency_latency_seconds_bucket{dependency="postgres-main"}[5m])
)
```

## app_dependency_status

Gauge с enum-паттерном. Для каждого эндпоинта создаётся 8 временных рядов —
по одному на категорию статуса. Ровно один ряд имеет значение `1`,
остальные — `0`.

Категории статусов:

| Метка `status` | Значение |
| --- | --- |
| `ok` | Здоров — проверка успешна |
| `timeout` | Таймаут проверки |
| `connection_error` | Невозможно подключиться |
| `dns_error` | Ошибка DNS-разрешения |
| `auth_error` | Ошибка аутентификации/авторизации |
| `tls_error` | Ошибка TLS-рукопожатия |
| `unhealthy` | Подключён, но зависимость сообщает о проблеме |
| `error` | Неожиданная/неклассифицированная ошибка |

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

### Примеры PromQL

```promql
# Все эндпоинты с ошибками авторизации
app_dependency_status{status="auth_error"} == 1

# Все эндпоинты с ошибками подключения
app_dependency_status{status="connection_error"} == 1

# Количество нездоровых эндпоинтов по группам
count(app_dependency_status{status="ok"} == 0) by (group)

# Алерт: любая критичная зависимость не OK 2 минуты
app_dependency_status{status="ok",critical="yes"} == 0
```

## app_dependency_status_detail

Gauge с info-паттерном. Один ряд на уникальное значение детализации.
Метка `detail` содержит конкретную причину сбоя.

Типичные значения detail:

| Detail | Источник | Значение |
| --- | --- | --- |
| `ok` | Все чекеры | Проверка успешна |
| `auth_error` | HTTP, gRPC, PG, MySQL, Redis, AMQP | Ошибка авторизации |
| `http_500` | HTTP | Ошибка сервера |
| `http_503` | HTTP | Сервис недоступен |
| `grpc_not_serving` | gRPC | Сервис не обслуживает |
| `grpc_unknown` | gRPC | Неизвестный gRPC-статус |
| `no_brokers` | Kafka | Нет брокеров в метаданных |
| `connection_refused` | Redis, ядро | Отказ соединения |
| `timeout` | Ядро | Таймаут проверки |
| `dns_error` | Ядро | Ошибка DNS |
| `tls_error` | Ядро | Ошибка TLS |
| `error` | Ядро | Неклассифицированная ошибка |

```text
app_dependency_status_detail{...,detail="ok"} 1
```

При изменении детализации (напр., с `ok` на `http_503`) старый ряд
получает значение `0`, а новый — `1`.

### Примеры PromQL

```promql
# Эндпоинты, возвращающие HTTP 503
app_dependency_status_detail{detail="http_503"} == 1

# Все активные детализации (не ok)
app_dependency_status_detail{detail!="ok"} == 1
```

## Пользовательский регистратор Prometheus

По умолчанию метрики регистрируются в `prometheus.DefaultRegisterer`.
Для использования пользовательского реестра:

```go
registry := prometheus.NewRegistry()

dh, err := dephealth.New("my-service", "my-team",
    dephealth.WithRegisterer(registry),
    // ... зависимости
)
```

Сценарии использования:

- Отдельный реестр для тестирования
- Интеграция с push gateway
- Несколько экземпляров `DepHealth` (каждому нужен свой регистратор для
  избежания ошибок дублирования метрик)

## Экспорт метрик

Используйте стандартный `promhttp.Handler()`:

```go
http.Handle("/metrics", promhttp.Handler())
```

Или с пользовательским реестром:

```go
http.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
```

## См. также

- [Начало работы](getting-started.ru.md) — базовая настройка с Prometheus
- [Чекеры](checkers.ru.md) — классификация ошибок по чекерам
- [Дашборды Grafana](../../docs/grafana-dashboards.ru.md) — настройка дашбордов
- [Правила алертинга](../../docs/alerting/alert-rules.ru.md) — алертинг на основе метрик
- [Спецификация метрик](../../spec/metric-contract.ru.md) — формальный контракт метрик
- [Устранение неполадок](troubleshooting.ru.md) — типичные проблемы и решения
