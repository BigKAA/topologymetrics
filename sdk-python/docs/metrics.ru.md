*[English version](metrics.md)*

# Prometheus-метрики

Руководство по всем Prometheus-метрикам, экспортируемым dephealth Python SDK,
их меткам и примерам PromQL-запросов.

## Обзор

| Метрика | Тип | Описание |
| --- | --- | --- |
| `app_dependency_health` | Gauge | Бинарный индикатор здоровья (0 или 1) |
| `app_dependency_latency_seconds` | Histogram | Распределение задержки проверок |
| `app_dependency_status` | Gauge (enum) | Категория статуса (8 серий на эндпоинт) |
| `app_dependency_status_detail` | Gauge (info) | Детальная причина сбоя |

## Метки

Все метрики имеют общие метки:

| Метка | Описание | Пример |
| --- | --- | --- |
| `name` | Имя приложения | `my-service` |
| `group` | Группа приложения | `my-team` |
| `dependency` | Имя зависимости | `postgres-main` |
| `type` | Тип зависимости | `postgres`, `redis`, `http`, `grpc`, `tcp`, `amqp`, `kafka`, `ldap`, `mysql` |
| `host` | Имя хоста | `pg.svc` |
| `port` | Порт | `5432` |
| `critical` | Флаг критичности | `yes` или `no` |

Пользовательские метки (через параметр `labels=`) добавляются рядом
со стандартными метками.

## app_dependency_health

Бинарный индикатор здоровья для каждого эндпоинта.

- **Тип:** Gauge
- **Значения:** `1` (здоров) или `0` (нездоров)

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

# Статус здоровья конкретного сервиса
app_dependency_health{name="my-service"}

# Процент здоровых зависимостей
avg(app_dependency_health{name="my-service"}) * 100
```

## app_dependency_latency_seconds

Гистограмма задержки проверок для каждого эндпоинта.

- **Тип:** Histogram
- **Бакеты:** `0.001`, `0.005`, `0.01`, `0.05`, `0.1`, `0.5`, `1.0`, `5.0`

```text
app_dependency_latency_seconds_bucket{...,le="0.001"} 0
app_dependency_latency_seconds_bucket{...,le="0.005"} 12
app_dependency_latency_seconds_bucket{...,le="0.01"} 42
app_dependency_latency_seconds_bucket{...,le="0.05"} 99
app_dependency_latency_seconds_bucket{...,le="0.1"} 100
app_dependency_latency_seconds_bucket{...,le="0.5"} 100
app_dependency_latency_seconds_bucket{...,le="1.0"} 100
app_dependency_latency_seconds_bucket{...,le="5.0"} 100
app_dependency_latency_seconds_bucket{...,le="+Inf"} 100
app_dependency_latency_seconds_count{...} 100
app_dependency_latency_seconds_sum{...} 0.42
```

### Примеры PromQL

```promql
# 95-й процентиль задержки по зависимостям
histogram_quantile(0.95,
  rate(app_dependency_latency_seconds_bucket{name="my-service"}[5m]))

# Средняя задержка по зависимостям
rate(app_dependency_latency_seconds_sum[5m])
  / rate(app_dependency_latency_seconds_count[5m])

# Задержка выше 1с (потенциальный таймаут)
histogram_quantile(0.99,
  rate(app_dependency_latency_seconds_bucket[5m])) > 1
```

## app_dependency_status

Enum-pattern gauge: 8 серий на эндпоинт, ровно одна имеет значение `1`.

- **Тип:** Gauge
- **Дополнительная метка:** `status`
- **Значения:** `1` (активный статус) или `0` (неактивный)

Категории статуса:

| Статус | Описание |
| --- | --- |
| `ok` | Проверка успешна |
| `timeout` | Таймаут проверки |
| `connection_error` | Соединение отклонено или сброшено |
| `dns_error` | Ошибка DNS-разрешения |
| `auth_error` | Ошибка аутентификации |
| `tls_error` | Ошибка TLS |
| `unhealthy` | Эндпоинт сообщил о нездоровом состоянии |
| `error` | Другая непредвиденная ошибка |

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
# Текущая категория статуса зависимости
app_dependency_status{dependency="postgres-main",status!=""} == 1

# Все эндпоинты с ошибками аутентификации
app_dependency_status{status="auth_error"} == 1

# Все эндпоинты с таймаутом
app_dependency_status{status="timeout"} == 1

# Количество не-ok эндпоинтов по сервисам
count(app_dependency_status{status!="ok"} == 1) by (name)
```

## app_dependency_status_detail

Info-pattern gauge: предоставляет строку с детальной причиной сбоя.

- **Тип:** Gauge
- **Дополнительная метка:** `detail`
- **Значение:** `1` когда серия существует; серия удаляется и пересоздаётся при смене детали

Значения detail зависят от типа чекера:

| Detail | Описание |
| --- | --- |
| `ok` | Проверка успешна |
| `timeout` | Таймаут проверки |
| `connection_refused` | Соединение отклонено |
| `dns_error` | Ошибка DNS-разрешения |
| `auth_error` | Ошибка аутентификации |
| `tls_error` | Ошибка TLS |
| `http_<code>` | HTTP-код ответа (напр., `http_503`) |
| `grpc_not_serving` | gRPC-сервис не обслуживает |
| `no_brokers` | Нет доступных брокеров Kafka |
| `ldap_no_results` | LDAP-поиск не вернул результатов |

```text
app_dependency_status_detail{...,detail="ok"} 1
```

### Примеры PromQL

```promql
# Детальная причина для конкретной зависимости
app_dependency_status_detail{dependency="postgres-main",detail!=""} == 1

# Все эндпоинты с HTTP 503
app_dependency_status_detail{detail="http_503"} == 1
```

## Экспорт метрик

### FastAPI

```python
from dephealth_fastapi import DepHealthMiddleware

app.add_middleware(DepHealthMiddleware)
# Метрики доступны на GET /metrics
```

### Без FastAPI

Используйте стандартный `prometheus_client`:

```python
from prometheus_client import generate_latest, CONTENT_TYPE_LATEST

# В вашем HTTP-обработчике:
def metrics_handler(request):
    return Response(
        content=generate_latest(),
        media_type=CONTENT_TYPE_LATEST,
    )
```

### Кастомный registry

```python
from prometheus_client import CollectorRegistry

custom_registry = CollectorRegistry()

dh = DependencyHealth("my-service", "my-team",
    registry=custom_registry,
    # ... спецификации зависимостей
)

# Экспорт кастомного registry
from prometheus_client import generate_latest
output = generate_latest(custom_registry)
```

## См. также

- [Быстрый старт](getting-started.ru.md) — базовая настройка и первый пример
- [Чекеры](checkers.ru.md) — все 9 встроенных чекеров
- [Grafana Dashboards](../../docs/grafana-dashboards.ru.md) — конфигурация дашбордов
- [Правила алертинга](../../docs/alerting/alert-rules.ru.md) — конфигурация алертинга
- [Спецификация](../../spec/) — кросс-SDK контракты метрик
- [Troubleshooting](troubleshooting.ru.md) — частые проблемы и решения
