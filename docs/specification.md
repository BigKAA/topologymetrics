# Обзор спецификации dephealth

Спецификация dephealth — единый источник правды для всех SDK.
Она определяет формат метрик, поведение проверок и конфигурацию
соединений. Все SDK должны строго соответствовать этим контрактам.

Полные документы спецификации находятся в каталоге [`spec/`](../spec/).

## Контракт метрик

> Полный документ: [`spec/metric-contract.md`](../spec/metric-contract.md)

### Метрика здоровья

```text
app_dependency_health{dependency="postgres-main",type="postgres",host="pg.svc",port="5432"} 1
```

| Свойство | Значение |
| --- | --- |
| Имя | `app_dependency_health` |
| Тип | Gauge |
| Значения | `1` (доступен), `0` (недоступен) |
| Обязательные метки | `dependency`, `type`, `host`, `port` |
| Опциональные метки | `role`, `shard`, `vhost` |

### Метрика латентности

```text
app_dependency_latency_seconds_bucket{dependency="postgres-main",type="postgres",host="pg.svc",port="5432",le="0.01"} 42
```

| Свойство | Значение |
| --- | --- |
| Имя | `app_dependency_latency_seconds` |
| Тип | Histogram |
| Бакеты | `0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0, 5.0` |
| Метки | Идентичны `app_dependency_health` |

### Правила формирования меток

- `dependency` — логическое имя (например, `postgres-main`, `redis-cache`)
- `type` — тип зависимости: `http`, `grpc`, `tcp`, `postgres`, `mysql`,
  `redis`, `amqp`, `kafka`
- `host` — DNS-имя или IP-адрес endpoint
- `port` — порт endpoint

При нескольких endpoint-ах одной зависимости (например, primary + replica)
создаётся отдельная метрика для каждого endpoint.

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

## Conformance-тестирование

Все SDK проходят единый набор conformance-сценариев в Kubernetes:

| Сценарий | Проверяет |
| --- | --- |
| `basic-health` | Все зависимости доступны -> метрики = 1 |
| `partial-failure` | Частичный отказ -> правильные значения |
| `full-failure` | Полный отказ зависимости -> метрика = 0 |
| `recovery` | Восстановление -> метрика возвращается к 1 |
| `latency` | Histogram бакеты присутствуют |
| `labels` | Правильность всех меток |
| `timeout` | Задержка > timeout -> unhealthy |
| `initial-state` | Начальное состояние корректно |

Подробнее: [`conformance/`](../conformance/)

## Ссылки

- [Быстрый старт Go SDK](quickstart/go.md)
- [Руководство по интеграции Go SDK](migration/go.md)
