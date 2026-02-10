[English](#english) | [Русский](#russian)

---

<a id="english"></a>

# dephealth Specification Overview

The dephealth specification is the single source of truth for all SDKs.
It defines metric format, check behavior, and connection configuration.
All SDKs must strictly comply with these contracts.

Full specification documents are located in the [`spec/`](../spec/) directory.

## Metric Contract

> Full document: [`spec/metric-contract.md`](../spec/metric-contract.md)

### Health Metric

```text
app_dependency_health{name="my-service",dependency="postgres-main",type="postgres",host="pg.svc",port="5432",critical="yes"} 1
```

| Property | Value |
| --- | --- |
| Name | `app_dependency_health` |
| Type | Gauge |
| Values | `1` (available), `0` (unavailable) |
| Required labels | `name`, `dependency`, `type`, `host`, `port`, `critical` |
| Optional labels | arbitrary via `WithLabel(key, value)` |

### Latency Metric

```text
app_dependency_latency_seconds_bucket{name="my-service",dependency="postgres-main",type="postgres",host="pg.svc",port="5432",critical="yes",le="0.01"} 42
```

| Property | Value |
| --- | --- |
| Name | `app_dependency_latency_seconds` |
| Type | Histogram |
| Buckets | `0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0, 5.0` |
| Labels | Identical to `app_dependency_health` |

### Label Formation Rules

- `name` — unique application name (format `[a-z][a-z0-9-]*`, 1-63 characters)
- `dependency` — logical name (e.g. `postgres-main`, `redis-cache`)
- `type` — dependency type: `http`, `grpc`, `tcp`, `postgres`, `mysql`,
  `redis`, `amqp`, `kafka`
- `host` — DNS name or IP address of the endpoint
- `port` — endpoint port
- `critical` — dependency criticality: `yes` or `no`

Label order: `name`, `dependency`, `type`, `host`, `port`, `critical`,
then arbitrary labels in alphabetical order.

When a single dependency has multiple endpoints (e.g. primary + replica),
a separate metric is created for each endpoint.

### Custom Labels

Custom labels are added via `WithLabel(key, value)` (Go),
`.label(key, value)` (Java), `labels={"key": "value"}` (Python),
`.Label(key, value)` (C#).

Label names: format `[a-zA-Z_][a-zA-Z0-9_]*`, required labels
(`name`, `dependency`, `type`, `host`, `port`, `critical`) cannot be overridden.

## Behavior Contract

> Full document: [`spec/check-behavior.md`](../spec/check-behavior.md)

### Check Lifecycle

```text
Initialization → initialDelay → First check → Periodic checks (every checkInterval)
                                                          ↓
                                                   Graceful Shutdown
```

### Default Parameters

| Parameter | Value | Description |
| --- | --- | --- |
| `checkInterval` | 15s | Interval between checks |
| `timeout` | 5s | Single check timeout |
| `initialDelay` | 5s | Delay before the first check |
| `failureThreshold` | 1 | Consecutive failures to transition to unhealthy |
| `successThreshold` | 1 | Consecutive successes to transition to healthy |

### Threshold Logic

- **healthy -> unhealthy**: after `failureThreshold` consecutive failures
- **unhealthy -> healthy**: after `successThreshold` consecutive successes
- **Initial state**: unknown until the first check

### Check Types

| Type | Method | Success Criteria |
| --- | --- | --- |
| `http` | HTTP GET to `healthPath` | 2xx status |
| `grpc` | gRPC Health Check Protocol | `SERVING` |
| `tcp` | TCP connection establishment | Connection established |
| `postgres` | `SELECT 1` | Query executed |
| `mysql` | `SELECT 1` | Query executed |
| `redis` | `PING` | `PONG` response |
| `amqp` | Open/close connection | Connection established |
| `kafka` | Metadata request | Response received |

### Two Operating Modes

- **Standalone**: SDK creates a temporary connection for
  each check. Simple to configure but creates additional load.
- **Connection pool integration**: SDK uses the service's existing pool.
  Reflects the service's actual ability to work with the dependency.
  Recommended for databases and caches.

### Error Handling

Any of the following situations is considered a failed check:

- Timeout (`context deadline exceeded`)
- DNS resolution failure
- Connection refused
- TLS handshake failure
- Unexpected response (non-2xx for HTTP, non-`SERVING` for gRPC)

## Configuration Contract

> Full document: [`spec/config-contract.md`](../spec/config-contract.md)

### Connection Input Formats

| Format | Example |
| --- | --- |
| URL | `postgres://user:pass@host:5432/db` |
| Direct parameters | `host` + `port` |
| Connection string | `Host=host;Port=5432;Database=db` |
| JDBC URL | `jdbc:postgresql://host:5432/db` |

### Auto-detection of Type

Dependency type is determined from the URL scheme:

| Scheme | Type |
| --- | --- |
| `postgres://`, `postgresql://` | `postgres` |
| `mysql://` | `mysql` |
| `redis://`, `rediss://` | `redis` |
| `amqp://`, `amqps://` | `amqp` |
| `http://`, `https://` | `http` |
| `grpc://` | `grpc` |
| `kafka://` | `kafka` |

### Default Ports

| Type | Port |
| --- | --- |
| `postgres` | 5432 |
| `mysql` | 3306 |
| `redis` | 6379 |
| `amqp` | 5672 |
| `http` | 80 / 443 (HTTPS) |
| `grpc` | 443 |
| `kafka` | 9092 |
| `tcp` | (required) |

### Allowed Parameter Ranges

| Parameter | Minimum | Maximum |
| --- | --- | --- |
| `checkInterval` | 1s | 10m |
| `timeout` | 100ms | 30s |
| `initialDelay` | 0 | 5m |
| `failureThreshold` | 1 | 10 |
| `successThreshold` | 1 | 10 |

Additional constraint: `timeout` must be less than `checkInterval`.

### Environment Variables

| Variable | Description |
| --- | --- |
| `DEPHEALTH_NAME` | Application name (overridden by API) |
| `DEPHEALTH_<DEP>_CRITICAL` | Dependency criticality: `yes`/`no` |
| `DEPHEALTH_<DEP>_LABEL_<KEY>` | Custom label for a dependency |

`<DEP>` — dependency name in uppercase, hyphens replaced with `_`.

## Conformance Testing

All SDKs pass a unified set of conformance scenarios in Kubernetes:

| Scenario | Verifies |
| --- | --- |
| `basic-health` | All dependencies available -> metrics = 1 |
| `partial-failure` | Partial failure -> correct values |
| `full-failure` | Full dependency failure -> metric = 0 |
| `recovery` | Recovery -> metric returns to 1 |
| `latency` | Histogram buckets present |
| `labels` | Correctness of all labels (name, critical, custom labels) |
| `timeout` | Delay > timeout -> unhealthy |
| `initial-state` | Initial state is correct |

More details: [`conformance/`](../conformance/)

## Links

- [Go SDK Quick Start](quickstart/go.md)
- [Java SDK Quick Start](quickstart/java.md)
- [Python SDK Quick Start](quickstart/python.md)
- [C# SDK Quick Start](quickstart/csharp.md)
- [Go SDK Integration Guide](migration/go.md)
- [Java SDK Integration Guide](migration/java.md)
- [Python SDK Integration Guide](migration/python.md)
- [C# SDK Integration Guide](migration/csharp.md)

---

<a id="russian"></a>

# Обзор спецификации dephealth

Спецификация dephealth — единый источник правды для всех SDK.
Она определяет формат метрик, поведение проверок и конфигурацию
соединений. Все SDK должны строго соответствовать этим контрактам.

Полные документы спецификации находятся в каталоге [`spec/`](../spec/).

## Контракт метрик

> Полный документ: [`spec/metric-contract.md`](../spec/metric-contract.md)

### Метрика здоровья

```text
app_dependency_health{name="my-service",dependency="postgres-main",type="postgres",host="pg.svc",port="5432",critical="yes"} 1
```

| Свойство | Значение |
| --- | --- |
| Имя | `app_dependency_health` |
| Тип | Gauge |
| Значения | `1` (доступен), `0` (недоступен) |
| Обязательные метки | `name`, `dependency`, `type`, `host`, `port`, `critical` |
| Опциональные метки | произвольные через `WithLabel(key, value)` |

### Метрика латентности

```text
app_dependency_latency_seconds_bucket{name="my-service",dependency="postgres-main",type="postgres",host="pg.svc",port="5432",critical="yes",le="0.01"} 42
```

| Свойство | Значение |
| --- | --- |
| Имя | `app_dependency_latency_seconds` |
| Тип | Histogram |
| Бакеты | `0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0, 5.0` |
| Метки | Идентичны `app_dependency_health` |

### Правила формирования меток

- `name` — уникальное имя приложения (формат `[a-z][a-z0-9-]*`, 1-63 символа)
- `dependency` — логическое имя (например, `postgres-main`, `redis-cache`)
- `type` — тип зависимости: `http`, `grpc`, `tcp`, `postgres`, `mysql`,
  `redis`, `amqp`, `kafka`
- `host` — DNS-имя или IP-адрес endpoint
- `port` — порт endpoint
- `critical` — критичность зависимости: `yes` или `no`

Порядок меток: `name`, `dependency`, `type`, `host`, `port`, `critical`,
затем произвольные метки в алфавитном порядке.

При нескольких endpoint-ах одной зависимости (например, primary + replica)
создаётся отдельная метрика для каждого endpoint.

### Произвольные метки

Произвольные метки добавляются через `WithLabel(key, value)` (Go),
`.label(key, value)` (Java), `labels={"key": "value"}` (Python),
`.Label(key, value)` (C#).

Имена меток: формат `[a-zA-Z_][a-zA-Z0-9_]*`, нельзя переопределять
обязательные метки (`name`, `dependency`, `type`, `host`, `port`, `critical`).

## Контракт поведения

> Полный документ: [`spec/check-behavior.md`](../spec/check-behavior.md)

### Жизненный цикл проверки

```text
Инициализация → initialDelay → Первая проверка → Периодические проверки (каждые checkInterval)
                                                          ↓
                                                   Graceful Shutdown
```

### Параметры по умолчанию

| Параметр | Значение | Описание |
| --- | --- | --- |
| `checkInterval` | 15s | Интервал между проверками |
| `timeout` | 5s | Таймаут одной проверки |
| `initialDelay` | 5s | Задержка перед первой проверкой |
| `failureThreshold` | 1 | Неудач подряд для перехода в unhealthy |
| `successThreshold` | 1 | Успехов подряд для перехода в healthy |

### Логика порогов

- **healthy -> unhealthy**: после `failureThreshold` последовательных неудач
- **unhealthy -> healthy**: после `successThreshold` последовательных успехов
- **Начальное состояние**: unknown до первой проверки

### Типы проверок

| Тип | Метод | Критерий успеха |
| --- | --- | --- |
| `http` | HTTP GET к `healthPath` | Статус 2xx |
| `grpc` | gRPC Health Check Protocol | `SERVING` |
| `tcp` | Установка TCP-соединения | Соединение установлено |
| `postgres` | `SELECT 1` | Запрос выполнен |
| `mysql` | `SELECT 1` | Запрос выполнен |
| `redis` | `PING` | Ответ `PONG` |
| `amqp` | Открытие/закрытие соединения | Соединение установлено |
| `kafka` | Metadata request | Ответ получен |

### Два режима работы

- **Автономный (standalone)**: SDK создаёт временное соединение для
  каждой проверки. Простой в настройке, но создаёт дополнительную нагрузку.
- **Интеграция с connection pool**: SDK использует существующий pool сервиса.
  Отражает реальную способность сервиса работать с зависимостью.
  Рекомендуется для БД и кэшей.

### Обработка ошибок

Любая из следующих ситуаций считается неудачной проверкой:

- Таймаут (`context deadline exceeded`)
- DNS resolution failure
- Connection refused
- TLS handshake failure
- Неожиданный ответ (не 2xx для HTTP, не `SERVING` для gRPC)

## Контракт конфигурации

> Полный документ: [`spec/config-contract.md`](../spec/config-contract.md)

### Форматы ввода соединений

| Формат | Пример |
| --- | --- |
| URL | `postgres://user:pass@host:5432/db` |
| Прямые параметры | `host` + `port` |
| Connection string | `Host=host;Port=5432;Database=db` |
| JDBC URL | `jdbc:postgresql://host:5432/db` |

### Автоопределение типа

Тип зависимости определяется из URL-схемы:

| Схема | Тип |
| --- | --- |
| `postgres://`, `postgresql://` | `postgres` |
| `mysql://` | `mysql` |
| `redis://`, `rediss://` | `redis` |
| `amqp://`, `amqps://` | `amqp` |
| `http://`, `https://` | `http` |
| `grpc://` | `grpc` |
| `kafka://` | `kafka` |

### Порты по умолчанию

| Тип | Порт |
| --- | --- |
| `postgres` | 5432 |
| `mysql` | 3306 |
| `redis` | 6379 |
| `amqp` | 5672 |
| `http` | 80 / 443 (HTTPS) |
| `grpc` | 443 |
| `kafka` | 9092 |
| `tcp` | (обязательный) |

### Допустимые диапазоны параметров

| Параметр | Минимум | Максимум |
| --- | --- | --- |
| `checkInterval` | 1s | 10m |
| `timeout` | 100ms | 30s |
| `initialDelay` | 0 | 5m |
| `failureThreshold` | 1 | 10 |
| `successThreshold` | 1 | 10 |

Дополнительное ограничение: `timeout` должен быть меньше `checkInterval`.

### Переменные окружения

| Переменная | Описание |
| --- | --- |
| `DEPHEALTH_NAME` | Имя приложения (перекрывается API) |
| `DEPHEALTH_<DEP>_CRITICAL` | Критичность зависимости: `yes`/`no` |
| `DEPHEALTH_<DEP>_LABEL_<KEY>` | Произвольная метка для зависимости |

`<DEP>` — имя зависимости в верхнем регистре, дефисы заменены на `_`.

## Conformance-тестирование

Все SDK проходят единый набор conformance-сценариев в Kubernetes:

| Сценарий | Проверяет |
| --- | --- |
| `basic-health` | Все зависимости доступны -> метрики = 1 |
| `partial-failure` | Частичный отказ -> правильные значения |
| `full-failure` | Полный отказ зависимости -> метрика = 0 |
| `recovery` | Восстановление -> метрика возвращается к 1 |
| `latency` | Histogram бакеты присутствуют |
| `labels` | Правильность всех меток (name, critical, custom labels) |
| `timeout` | Задержка > timeout -> unhealthy |
| `initial-state` | Начальное состояние корректно |

Подробнее: [`conformance/`](../conformance/)

## Ссылки

- [Быстрый старт Go SDK](quickstart/go.md)
- [Быстрый старт Java SDK](quickstart/java.md)
- [Быстрый старт Python SDK](quickstart/python.md)
- [Быстрый старт C# SDK](quickstart/csharp.md)
- [Руководство по интеграции Go SDK](migration/go.md)
- [Руководство по интеграции Java SDK](migration/java.md)
- [Руководство по интеграции Python SDK](migration/python.md)
- [Руководство по интеграции C# SDK](migration/csharp.md)
