*[English version](configuration.md)*

# Конфигурация

Руководство по всем параметрам конфигурации Go SDK dephealth: глобальные
настройки, опции зависимостей, переменные окружения и правила валидации.

## Имя и группа

```go
dh, err := dephealth.New("my-service", "my-team", ...)
```

| Параметр | Обязателен | Валидация | Fallback из env |
| --- | --- | --- | --- |
| `name` | Да | `[a-z][a-z0-9-]*`, 1-63 символов | `DEPHEALTH_NAME` |
| `group` | Да | `[a-z][a-z0-9-]*`, 1-63 символов | `DEPHEALTH_GROUP` |

Приоритет: аргумент API > переменная окружения.

Если оба пусты, `New()` возвращает ошибку.

## Глобальные опции

Глобальные опции передаются в `dephealth.New()` и применяются ко всем
зависимостям, если не переопределены для конкретной зависимости.

| Опция | Тип | По умолчанию | Диапазон | Описание |
| --- | --- | --- | --- | --- |
| `WithCheckInterval(d)` | `time.Duration` | 15 сек | 1с – 10м | Интервал между проверками |
| `WithTimeout(d)` | `time.Duration` | 5 сек | 100мс – 30с | Таймаут одной проверки |
| `WithRegisterer(r)` | `prometheus.Registerer` | `prometheus.DefaultRegisterer` | — | Пользовательский регистратор Prometheus |
| `WithLogger(l)` | `*slog.Logger` | нет | — | Логгер для операций SDK |

### Пример

```go
dh, err := dephealth.New("my-service", "my-team",
    dephealth.WithCheckInterval(30 * time.Second),
    dephealth.WithTimeout(3 * time.Second),
    dephealth.WithLogger(slog.Default()),
    dephealth.WithRegisterer(prometheus.NewRegistry()),
    // ... зависимости
)
```

## Общие опции зависимостей

Эти опции применимы к любому типу зависимости.

| Опция | Обязательна | По умолчанию | Описание |
| --- | --- | --- | --- |
| `FromURL(url)` | Одна из FromURL/FromParams | — | Парсинг host и port из URL |
| `FromParams(host, port)` | Одна из FromURL/FromParams | — | Явное указание host и port |
| `Critical(v)` | Да | — | Пометить как критичную (`true`) или нет (`false`) |
| `WithLabel(key, value)` | Нет | — | Добавить пользовательскую метку Prometheus |
| `CheckInterval(d)` | Нет | глобальное значение | Интервал проверки для зависимости |
| `Timeout(d)` | Нет | глобальное значение | Таймаут для зависимости |

### Указание эндпоинта

Каждая зависимость требует эндпоинт. Используйте один из способов:

```go
// Из URL — SDK парсит host и port
dephealth.FromURL("postgresql://user:pass@pg.svc:5432/mydb")

// Из параметров — явные host и port
dephealth.FromParams("pg.svc", "5432")
```

Поддерживаемые схемы URL: `http`, `https`, `grpc`, `tcp`, `postgresql`,
`postgres`, `mysql`, `redis`, `rediss`, `amqp`, `amqps`, `kafka`.

Для Kafka поддерживаются multi-host URL:
`kafka://broker1:9092,broker2:9092` — каждый хост создаёт отдельный эндпоинт.

### Флаг критичности

Опция `Critical()` **обязательна** для каждой зависимости. Если не
указана, возникает ошибка валидации. При отсутствии в API SDK проверяет
переменную окружения `DEPHEALTH_<DEP>_CRITICAL` (значения: `yes`/`no`,
`true`/`false`).

### Пользовательские метки

```go
dephealth.Postgres("postgres-main",
    dephealth.FromURL(os.Getenv("DATABASE_URL")),
    dephealth.Critical(true),
    dephealth.WithLabel("role", "primary"),
    dephealth.WithLabel("shard", "eu-west"),
)
```

Валидация имён меток:

- Должно соответствовать `[a-z_][a-z0-9_]*`
- Нельзя использовать зарезервированные имена: `name`, `group`, `dependency`, `type`, `host`, `port`, `critical`, `status`, `detail`

## Опции, специфичные для чекеров

### HTTP

| Опция | По умолчанию | Описание |
| --- | --- | --- |
| `WithHTTPHealthPath(path)` | `/health` | Путь эндпоинта проверки |
| `WithHTTPTLS(enabled)` | авто (true для `https://`) | Включить HTTPS |
| `WithHTTPTLSSkipVerify(skip)` | `false` | Пропустить проверку TLS-сертификата |
| `WithHTTPHeaders(headers)` | — | Пользовательские HTTP-заголовки |
| `WithHTTPBearerToken(token)` | — | Аутентификация Bearer-токеном |
| `WithHTTPBasicAuth(user, pass)` | — | Basic-аутентификация |

### gRPC

| Опция | По умолчанию | Описание |
| --- | --- | --- |
| `WithGRPCServiceName(name)` | `""` | Имя сервиса (пустое = весь сервер) |
| `WithGRPCTLS(enabled)` | `false` | Включить TLS |
| `WithGRPCTLSSkipVerify(skip)` | `false` | Пропустить проверку TLS-сертификата |
| `WithGRPCMetadata(md)` | — | Пользовательские gRPC-метаданные |
| `WithGRPCBearerToken(token)` | — | Аутентификация Bearer-токеном |
| `WithGRPCBasicAuth(user, pass)` | — | Basic-аутентификация |

### PostgreSQL

| Опция | По умолчанию | Описание |
| --- | --- | --- |
| `WithPostgresQuery(query)` | `SELECT 1` | SQL-запрос для проверки |

### MySQL

| Опция | По умолчанию | Описание |
| --- | --- | --- |
| `WithMySQLQuery(query)` | `SELECT 1` | SQL-запрос для проверки |

### Redis

| Опция | По умолчанию | Описание |
| --- | --- | --- |
| `WithRedisPassword(password)` | `""` | Пароль Redis (автономный режим) |
| `WithRedisDB(db)` | `0` | Номер базы данных (автономный режим) |

### AMQP

| Опция | По умолчанию | Описание |
| --- | --- | --- |
| `WithAMQPURL(url)` | `amqp://guest:guest@host:port/` | Полный AMQP URL |

### TCP и Kafka

Нет специфичных опций.

## Переменные окружения

| Переменная | Описание | Пример |
| --- | --- | --- |
| `DEPHEALTH_NAME` | Имя приложения (fallback, если аргумент API пуст) | `my-service` |
| `DEPHEALTH_GROUP` | Логическая группа (fallback, если аргумент API пуст) | `my-team` |
| `DEPHEALTH_<DEP>_CRITICAL` | Критичность зависимости (`yes`/`no`) | `yes` |
| `DEPHEALTH_<DEP>_LABEL_<KEY>` | Значение пользовательской метки | `primary` |

`<DEP>` — имя зависимости в формате UPPER_SNAKE_CASE:
дефисы → подчёркивания, всё в верхнем регистре.

Пример: зависимость `"postgres-main"` → env-префикс `DEPHEALTH_POSTGRES_MAIN_`.

### Правила приоритета

Значения API всегда имеют приоритет над переменными окружения:

1. **name/group**: аргумент API > `DEPHEALTH_NAME`/`DEPHEALTH_GROUP` > ошибка
2. **critical**: опция `Critical()` > `DEPHEALTH_<DEP>_CRITICAL` > ошибка
3. **метки**: `WithLabel()` > `DEPHEALTH_<DEP>_LABEL_<KEY>` (API побеждает при конфликте)

### Пример

```bash
export DEPHEALTH_NAME=my-service
export DEPHEALTH_GROUP=my-team
export DEPHEALTH_POSTGRES_MAIN_CRITICAL=yes
export DEPHEALTH_POSTGRES_MAIN_LABEL_ROLE=primary
export DEPHEALTH_POSTGRES_MAIN_LABEL_SHARD=eu-west
```

```go
// name и group из env vars, critical и labels из env vars
dh, err := dephealth.New("", "",
    dephealth.Postgres("postgres-main",
        dephealth.FromURL(os.Getenv("DATABASE_URL")),
        // Critical и labels берутся из DEPHEALTH_POSTGRES_MAIN_*
    ),
)
```

## Приоритет опций

Для интервала и таймаута цепочка приоритетов:

```text
опция зависимости > глобальная опция > значение по умолчанию
```

| Настройка | Per-dependency | Глобальная | По умолчанию |
| --- | --- | --- | --- |
| Интервал проверки | `CheckInterval(d)` | `WithCheckInterval(d)` | 15 сек |
| Таймаут | `Timeout(d)` | `WithTimeout(d)` | 5 сек |

## Значения по умолчанию

| Параметр | Значение |
| --- | --- |
| Интервал проверки | 15 секунд |
| Таймаут | 5 секунд |
| Начальная задержка | 0 (без задержки) |
| Порог отказов | 1 |
| Порог успехов | 1 |
| HTTP health path | `/health` |
| HTTP TLS | `false` (авто-включение для `https://` URL) |
| Redis DB | `0` |
| Redis password | пусто |
| PostgreSQL query | `SELECT 1` |
| MySQL query | `SELECT 1` |
| AMQP URL | `amqp://guest:guest@host:port/` |
| gRPC service name | пусто (состояние всего сервера) |

## Правила валидации

`New()` валидирует всю конфигурацию и возвращает ошибку при нарушении
любого правила:

| Правило | Сообщение об ошибке |
| --- | --- |
| Не указано имя | `missing name: pass as first argument or set DEPHEALTH_NAME` |
| Не указана группа | `missing group: pass as second argument or set DEPHEALTH_GROUP` |
| Неверный формат имени/группы | `invalid name: must match [a-z][a-z0-9-]*, 1-63 chars` |
| Не указан Critical для зависимости | `missing critical for dependency "..."` |
| Не указан URL или host/port | `missing URL or host/port parameters` |
| Неверное имя метки | `invalid label name: ...` |
| Зарезервированное имя метки | `reserved label: ...` |
| Конфликт методов авторизации | `conflicting auth methods: specify only one of ...` |
| Не зарегистрирована фабрика чекера | `no checker factory registered for type "..."` |

## См. также

- [Начало работы](getting-started.ru.md) — базовая настройка и первый пример
- [Чекеры](checkers.ru.md) — специфичные опции чекеров подробно
- [Аутентификация](authentication.ru.md) — опции авторизации для HTTP и gRPC
- [API Reference](api-reference.ru.md) — полный справочник по всем символам
- [Устранение неполадок](troubleshooting.ru.md) — типичные проблемы и решения
