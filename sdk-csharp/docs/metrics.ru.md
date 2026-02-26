*[English version](metrics.md)*

# Метрики Prometheus

SDK dephealth экспортирует четыре метрики Prometheus для каждого
мониторируемого эндпоинта зависимости через prometheus-net. Это руководство
описывает каждую метрику, её метки и примеры PromQL.

## Обзор метрик

| Метрика | Тип | Описание |
| --- | --- | --- |
| `app_dependency_health` | Gauge | Статус здоровья: `1` = доступен, `0` = недоступен |
| `app_dependency_latency_seconds` | Histogram | Латентность проверки в секундах |
| `app_dependency_status` | Gauge (enum) | Категория статуса: 8 серий на эндпоинт |
| `app_dependency_status_detail` | Gauge (info) | Детальная причина сбоя |

## Метки

Все четыре метрики используют общий набор меток:

| Метка | Источник | Описание |
| --- | --- | --- |
| `name` | Первый аргумент builder | Имя приложения |
| `group` | Второй аргумент builder | Логическая группа |
| `dependency` | Имя зависимости | Идентификатор зависимости |
| `type` | Тип чекера | `http`, `grpc`, `tcp`, `postgres`, `mysql`, `redis`, `amqp`, `kafka`, `ldap` |
| `host` | `url`/`host` | Хост зависимости |
| `port` | `url`/`port` | Порт зависимости |
| `critical` | `critical` | `yes` или `no` |

Пользовательские метки, добавленные через
`labels: new Dictionary<string, string> { ... }`, появляются после `critical`
в алфавитном порядке.

Дополнительные метки для каждой метрики:

- `app_dependency_status` содержит `status` — одну из 8 категорий статуса
- `app_dependency_status_detail` содержит `detail` — конкретную причину сбоя

## app_dependency_health

Простой бинарный индикатор здоровья.

- Значение `1` — зависимость доступна (последняя проверка успешна)
- Значение `0` — зависимость недоступна (последняя проверка не удалась)

```text
app_dependency_health{name="my-service",group="my-team",dependency="postgres-main",type="postgres",host="pg.svc",port="5432",critical="yes"} 1
app_dependency_health{name="my-service",group="my-team",dependency="redis-cache",type="redis",host="redis.svc",port="6379",critical="no"} 0
```

### Примеры PromQL

```promql
# Все недоступные зависимости
app_dependency_health == 0

# Недоступные критические зависимости
app_dependency_health{critical="yes"} == 0

# Статус конкретной зависимости
app_dependency_health{name="my-service",dependency="postgres-main"}
```

## app_dependency_latency_seconds

Гистограмма латентности проверок.

Границы SLO: `0.001`, `0.005`, `0.01`, `0.05`, `0.1`, `0.5`, `1.0`,
`5.0` секунд.

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
# P95 латентность за последние 5 минут
histogram_quantile(0.95,
  rate(app_dependency_latency_seconds_bucket[5m])
)

# Средняя латентность
rate(app_dependency_latency_seconds_sum[5m])
  / rate(app_dependency_latency_seconds_count[5m])

# P99 для конкретной зависимости
histogram_quantile(0.99,
  rate(app_dependency_latency_seconds_bucket{dependency="postgres-main"}[5m])
)
```

## app_dependency_status

Gauge в enum-паттерне. Для каждого эндпоинта создаётся 8 временных рядов —
по одному на категорию статуса. Ровно один ряд имеет значение `1`,
остальные — `0`.

Категории статуса:

| Метка `status` | Значение |
| --- | --- |
| `ok` | Здоров — проверка успешна |
| `timeout` | Тайм-аут проверки |
| `connection_error` | Невозможно подключиться |
| `dns_error` | Ошибка DNS-разрешения |
| `auth_error` | Ошибка аутентификации/авторизации |
| `tls_error` | Ошибка TLS-рукопожатия |
| `unhealthy` | Подключение есть, но зависимость нездорова |
| `error` | Непредвиденная/неклассифицированная ошибка |

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
# Все эндпоинты с ошибками аутентификации
app_dependency_status{status="auth_error"} == 1

# Все эндпоинты с ошибками соединения
app_dependency_status{status="connection_error"} == 1

# Количество нездоровых эндпоинтов по командам
count(app_dependency_status{status="ok"} == 0) by (group)

# Алерт: критическая зависимость не OK в течение 2 минут
app_dependency_status{status="ok",critical="yes"} == 0
```

## app_dependency_status_detail

Gauge в info-паттерне. Один ряд на уникальное значение detail. Метка
`detail` содержит конкретную причину сбоя.

Типичные значения detail:

| Detail | Источник | Значение |
| --- | --- | --- |
| `ok` | Все чекеры | Проверка успешна |
| `auth_error` | HTTP, gRPC, PG, MySQL, Redis, AMQP, LDAP | Ошибка аутентификации |
| `http_500` | HTTP | Ошибка сервера |
| `http_503` | HTTP | Сервис недоступен |
| `grpc_not_serving` | gRPC | Сервис не обслуживает |
| `grpc_unknown` | gRPC | Неизвестный статус gRPC |
| `no_brokers` | Kafka | Нет брокеров в метаданных |
| `connection_refused` | Redis, ядро | Соединение отклонено |
| `timeout` | Ядро | Тайм-аут проверки |
| `dns_error` | Ядро | Ошибка DNS |
| `tls_error` | Ядро, LDAP | Ошибка TLS |
| `unhealthy` | LDAP | Сервер занят/недоступен |
| `error` | Ядро | Неклассифицированная ошибка |

```text
app_dependency_status_detail{...,detail="ok"} 1
```

При изменении detail (например, с `ok` на `http_503`) старый ряд
удаляется и создаётся новый со значением `1`.

### Примеры PromQL

```promql
# Эндпоинты, возвращающие HTTP 503
app_dependency_status_detail{detail="http_503"} == 1

# Все активные детали (не ok)
app_dependency_status_detail{detail!="ok"} == 1
```

## Экспорт метрик

### ASP.NET Core

Метрики доступны на `/metrics` через `prometheus-net.AspNetCore`. Добавьте
`app.MapMetrics()` в конвейер обработки запросов:

```csharp
using Prometheus;

var builder = WebApplication.CreateBuilder(args);
var app = builder.Build();

app.MapMetrics(); // эндпоинт /metrics

app.Run();
```

Убедитесь, что установлен пакет `prometheus-net.AspNetCore`:

```bash
dotnet add package prometheus-net.AspNetCore
```

### Программный API

Без ASP.NET Core используйте `MetricServer` из prometheus-net или экспортируйте
метрики из кастомного HTTP-обработчика через `Metrics.DefaultRegistry`:

```csharp
using Prometheus;

// Отдельный сервер метрик на порту 9090
var server = new MetricServer(port: 9090);
server.Start();
```

```csharp
using Prometheus;

// В кастомном HTTP-обработчике -- скрейп дефолтного реестра
var metrics = await Metrics.DefaultRegistry.CollectAndExportAsTextAsync(stream);
```

## См. также

- [Начало работы](getting-started.ru.md) — базовая настройка с Prometheus
- [Чекеры](checkers.ru.md) — классификация ошибок по чекерам
- [Дашборды Grafana](../../docs/grafana-dashboards.ru.md) — настройка дашбордов
- [Правила алертинга](../../docs/alerting/alert-rules.ru.md) — алертинг на основе метрик
- [Спецификация метрик](../../spec/metric-contract.md) — формальный контракт метрик
- [Устранение неполадок](troubleshooting.ru.md) — типичные проблемы и решения
