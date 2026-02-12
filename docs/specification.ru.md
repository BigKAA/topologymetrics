*[English version](specification.md)*

# Обзор спецификации dephealth

Спецификация dephealth — единый источник правды для всех SDK.
Она определяет формат метрик, поведение проверок и конфигурацию
соединений. Все SDK должны строго соответствовать этим контрактам.

Полные документы спецификации находятся в каталоге [`spec/`](../spec/).

## Контракт метрик

> Полный документ: [`spec/metric-contract.md`](../spec/metric-contract.ru.md)

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

> Полный документ: [`spec/check-behavior.md`](../spec/check-behavior.ru.md)

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

> Полный документ: [`spec/config-contract.md`](../spec/config-contract.ru.md)

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

### DNS-разрешение в Kubernetes

В автономном режиме dephealth создаёт новое соединение для каждой проверки,
что вызывает DNS-разрешение каждый раз. В Kubernetes файл `/etc/resolv.conf`
по умолчанию настроен с `ndots:5` и несколькими search-доменами
(например, `<ns>.svc.cluster.local svc.cluster.local cluster.local`).

Если имя хоста содержит меньше точек, чем значение `ndots`, резолвер
последовательно подставляет search-суффиксы, прежде чем попробовать имя
как есть. Например, для имени `redis.my-namespace.svc` (2 точки < 5)
будет выполнено несколько неудачных DNS-запросов до успешного:

```text
redis.my-namespace.svc.app-namespace.svc.cluster.local  → NXDOMAIN
redis.my-namespace.svc.svc.cluster.local                → NXDOMAIN
redis.my-namespace.svc.cluster.local                    → OK
```

Чтобы избежать этих накладных расходов, используйте **точку в конце**
имени хоста — это маркер абсолютного (полностью квалифицированного)
доменного имени (FQDN). Резолвер пропустит перебор search-доменов
и выполнит ровно один DNS-запрос:

```yaml
# Относительное имя — вызывает перебор search-доменов (несколько DNS-запросов)
host: "redis.my-namespace.svc"

# Абсолютное имя (FQDN) — один DNS-запрос, без перебора
host: "redis.my-namespace.svc.cluster.local."
```

> **Примечание:** Завершающая точка (`.`) — часть стандарта DNS (RFC 1035),
> поддерживается всеми DNS-резолверами. Домен кластера (`cluster.local`
> по умолчанию) может отличаться в вашем окружении — проверьте
> `/etc/resolv.conf` внутри пода для получения актуального значения.

Эта оптимизация применима ко всем типам зависимостей и особенно
заметна для типов проверок с высокими накладными расходами на соединение
(gRPC, TLS).

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

- [Быстрый старт Go SDK](quickstart/go.ru.md)
- [Быстрый старт Java SDK](quickstart/java.ru.md)
- [Быстрый старт Python SDK](quickstart/python.ru.md)
- [Быстрый старт C# SDK](quickstart/csharp.ru.md)
- [Руководство по интеграции Go SDK](migration/go.ru.md)
- [Руководство по интеграции Java SDK](migration/java.ru.md)
- [Руководство по интеграции Python SDK](migration/python.ru.md)
- [Руководство по интеграции C# SDK](migration/csharp.ru.md)
