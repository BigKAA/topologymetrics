# Контракт конфигурации

> Версия спецификации: **1.0-draft**
>
> Этот документ описывает форматы входных данных для конфигурации зависимостей,
> правила парсинга, программный API и конфигурацию через переменные окружения.
> Все SDK обязаны поддерживать описанные форматы. Соответствие проверяется
> conformance-тестами.

---

## 1. Поддерживаемые форматы ввода

SDK принимает информацию о соединении с зависимостью в четырёх форматах:

| # | Формат | Пример | Приоритет |
| --- | --- | --- | --- |
| 1 | Полный URL | `postgres://user:pass@host:5432/db` | Основной |
| 2 | Отдельные параметры | `host=pg.svc, port=5432` | Альтернативный |
| 3 | Connection string | `Host=pg.svc;Port=5432;Database=db` | Для .NET/JDBC |
| 4 | JDBC URL | `jdbc:postgresql://host:5432/db` | Для Java |

SDK должен корректно извлечь из любого формата:

- `host` — адрес endpoint-а
- `port` — порт endpoint-а
- `type` — тип зависимости (если определяется из схемы URL)

---

## 2. Формат 1: Полный URL

### 2.1. Поддерживаемые схемы

| Схема | Определяемый `type` | Порт по умолчанию |
| --- | --- | --- |
| `postgres://`, `postgresql://` | `postgres` | `5432` |
| `mysql://` | `mysql` | `3306` |
| `redis://` | `redis` | `6379` |
| `rediss://` | `redis` | `6379` |
| `amqp://` | `amqp` | `5672` |
| `amqps://` | `amqp` | `5671` |
| `http://` | `http` | `80` |
| `https://` | `http` | `443` |
| `grpc://` | `grpc` | `443` |
| `kafka://` | `kafka` | `9092` |

### 2.2. Правила парсинга URL

Общий формат:

```text
scheme://[user:password@]host[:port][/path][?query]
```

**Извлечение host**:

- Из компонента `host` URL.
- IPv6 адреса в URL заключены в квадратные скобки: `[::1]`.
- В метке `host` метрики IPv6 записывается **без** квадратных скобок: `::1`.

**Извлечение port**:

- Из компонента `port` URL.
- Если порт не указан, используется порт по умолчанию для данной схемы
  (см. таблицу выше).

**Извлечение type**:

- Автоматически по схеме URL (см. таблицу выше).
- Если разработчик явно указал `type`, он имеет приоритет.

**Извлечение дополнительных данных**:

| Данные | Источник | Используется в |
| --- | --- | --- |
| `vhost` | Path компонент (`amqp://host/vhost`) | Метка `vhost`, подключение к AMQP |
| `database` | Path компонент (`redis://host/0`) | Выбор БД Redis |
| Credentials | Userinfo (`user:pass@`) | Аутентификация при автономной проверке |

### 2.3. URL с несколькими хостами

Некоторые зависимости поддерживают указание нескольких хостов в URL:

```text
# Kafka брокеры
kafka://broker-0:9092,broker-1:9092,broker-2:9092

# PostgreSQL failover (libpq format)
postgres://user:pass@primary:5432,replica:5432/db?target_session_attrs=read-write
```

**Правила**:

- Каждая пара `host:port` создаёт отдельный `Endpoint`.
- Все endpoint-ы объединяются в одну `Dependency`.
- Если порт указан только для первого хоста, остальные используют тот же порт.
- Если порт не указан ни для одного — используется порт по умолчанию.

### 2.4. Примеры парсинга URL

| URL | host | port | type |
| --- | --- | --- | --- |
| `postgres://app:pass@pg.svc:5432/orders` | `pg.svc` | `5432` | `postgres` |
| `postgres://app:pass@pg.svc/orders` | `pg.svc` | `5432` | `postgres` |
| `redis://redis.svc:6379/0` | `redis.svc` | `6379` | `redis` |
| `redis://redis.svc` | `redis.svc` | `6379` | `redis` |
| `rediss://redis.svc:6380/0` | `redis.svc` | `6380` | `redis` |
| `http://payment.svc:8080/health` | `payment.svc` | `8080` | `http` |
| `https://payment.svc/health` | `payment.svc` | `443` | `http` |
| `amqp://user:pass@rabbit.svc/orders` | `rabbit.svc` | `5672` | `amqp` |
| `kafka://broker-0.svc:9092` | `broker-0.svc` | `9092` | `kafka` |
| `postgres://[::1]:5432/db` | `::1` | `5432` | `postgres` |

---

## 3. Формат 2: Отдельные параметры

Минимальная конфигурация: `host` + `port`.

```go
dephealth.Postgres("postgres-main", dephealth.FromParams("pg.svc", "5432"))
```

**Правила**:

- `host` — обязательный. Строка (hostname, IPv4 или IPv6).
- `port` — обязательный. Строка с числом 1-65535.
- `type` — должен быть указан явно (через фабричный метод или параметр).
- При невалидном порте (не число, вне диапазона) — ошибка конфигурации.

---

## 4. Формат 3: Connection string

Формат `Key=Value;Key=Value`, распространённый в .NET-экосистеме.

### 4.1. Поддерживаемые ключи для host

Поиск (case-insensitive, первое найденное совпадение):

| Ключ | Описание |
| --- | --- |
| `Host` | Основной |
| `Server` | Альтернативный (SQL Server) |
| `Data Source` | Альтернативный (Oracle, SQLite) |
| `Address` | Альтернативный |
| `Addr` | Альтернативный |
| `Network Address` | Альтернативный |

### 4.2. Поддерживаемые ключи для port

| Ключ | Описание |
| --- | --- |
| `Port` | Основной |

Если порт не найден отдельным ключом, проверяется формат `Host=hostname,port`
(SQL Server convention) и `Host=hostname:port`.

### 4.3. Примеры парсинга connection string

| Connection string | host | port |
| --- | --- | --- |
| `Host=pg.svc;Port=5432;Database=orders` | `pg.svc` | `5432` |
| `Server=pg.svc,5432;Database=orders` | `pg.svc` | `5432` |
| `Host=pg.svc:5432;Database=orders` | `pg.svc` | `5432` |
| `Data Source=pg.svc;Port=5432` | `pg.svc` | `5432` |

### 4.4. Ограничения

- `type` не определяется автоматически из connection string.
  Разработчик должен указать тип явно.
- Если `host` не найден — ошибка конфигурации.
- Если `port` не найден — используется порт по умолчанию для указанного `type`.

---

## 5. Формат 4: JDBC URL

Формат, специфичный для Java-экосистемы.

### 5.1. Общий формат

```text
jdbc:<subprotocol>://host[:port][/database][?parameters]
```

### 5.2. Поддерживаемые subprotocol

| Subprotocol | Определяемый `type` | Порт по умолчанию |
| --- | --- | --- |
| `postgresql` | `postgres` | `5432` |
| `mysql` | `mysql` | `3306` |

### 5.3. Правила парсинга

1. Удалить префикс `jdbc:`.
2. Парсить оставшуюся часть как стандартный URL.
3. Извлечь `host`, `port`, `type` по тем же правилам.

### 5.4. Примеры

| JDBC URL | host | port | type |
| --- | --- | --- | --- |
| `jdbc:postgresql://pg.svc:5432/orders` | `pg.svc` | `5432` | `postgres` |
| `jdbc:postgresql://pg.svc/orders` | `pg.svc` | `5432` | `postgres` |
| `jdbc:mysql://mysql.svc:3306/orders` | `mysql.svc` | `3306` | `mysql` |

---

## 6. Таблица портов по умолчанию

| Тип | Порт по умолчанию | Протокол |
| --- | --- | --- |
| `postgres` | `5432` | TCP |
| `mysql` | `3306` | TCP |
| `redis` | `6379` | TCP |
| `amqp` | `5672` | TCP |
| `amqp` (TLS) | `5671` | TCP + TLS |
| `http` | `80` | TCP |
| `http` (TLS) | `443` | TCP + TLS |
| `grpc` | `443` | TCP (HTTP/2) |
| `kafka` | `9092` | TCP |
| `tcp` | — | TCP (порт обязателен) |

Для `type: tcp` порт по умолчанию **не определён** — разработчик обязан указать его явно.

---

## 7. Программный API конфигурации

### 7.1. Фабричные методы (создание зависимости)

Каждый тип зависимости имеет фабричный метод:

```go
// Go
dephealth.HTTP(name, source, ...opts)
dephealth.GRPC(name, source, ...opts)
dephealth.TCP(name, source, ...opts)
dephealth.Postgres(name, source, ...opts)
dephealth.MySQL(name, source, ...opts)
dephealth.Redis(name, source, ...opts)
dephealth.AMQP(name, source, ...opts)
dephealth.Kafka(name, source, ...opts)
```

**Параметры**:

- `name` (обязательный) — логическое имя зависимости. Формат: `[a-z][a-z0-9-]*`,
  длина 1-63 символа. При невалидном имени — ошибка конфигурации.
- `source` (обязательный) — источник конфигурации (URL, параметры, pool).
- `opts` (опциональные) — дополнительные настройки.

### 7.2. Источники конфигурации (Source)

```go
// Из URL
dephealth.FromURL("postgres://user:pass@host:5432/db")

// Из отдельных параметров
dephealth.FromParams("host", "5432")

// Из connection string
dephealth.FromConnectionString("Host=pg.svc;Port=5432;Database=orders")

// Из JDBC URL
dephealth.FromJDBC("jdbc:postgresql://host:5432/db")
```

### 7.3. Опции зависимости (DependencyOption)

```go
// Критичность (влияет на readiness)
dephealth.Critical(true)

// Индивидуальный интервал проверки
dephealth.WithCheckInterval(30 * time.Second)

// Индивидуальный таймаут
dephealth.WithTimeout(10 * time.Second)

// Индивидуальная начальная задержка
dephealth.WithInitialDelay(0)

// Пороги
dephealth.WithFailureThreshold(3)
dephealth.WithSuccessThreshold(2)

// HTTP-специфичные
dephealth.WithHealthPath("/ready")
dephealth.WithTLSSkipVerify(true)

// Метаданные (опциональные метки)
dephealth.WithMetadata("role", "primary")
dephealth.WithMetadata("shard", "shard-01")
```

### 7.4. Глобальные опции (при создании DepHealth)

```go
dh := dephealth.New(
    // Глобальные опции
    dephealth.WithDefaultCheckInterval(30 * time.Second),
    dephealth.WithDefaultTimeout(10 * time.Second),
    dephealth.WithRegisterer(customRegisterer),
    dephealth.WithLogger(slog.Default()),

    // Зависимости
    dephealth.Postgres("postgres-main", dephealth.FromURL(url)),
    dephealth.Redis("redis-cache", dephealth.FromURL(url)),
)
```

### 7.5. Валидация

SDK валидирует конфигурацию при вызове `New()` и возвращает ошибку при:

| Проблема | Ошибка |
| --- | --- |
| Невалидное имя зависимости | `invalid dependency name: "..."` |
| Дублирующееся имя + host + port | `duplicate endpoint: "..." host=... port=...` |
| Невалидный URL | `invalid URL: "..."` |
| Отсутствует host | `missing host for dependency "..."` |
| Невалидный порт | `invalid port "..." for dependency "..."` |
| `timeout >= checkInterval` | `timeout must be less than checkInterval for "..."` |
| Неизвестный тип | `unknown dependency type: "..."` |

---

## 8. Конфигурация через переменные окружения

### 8.1. Формат

```text
DEPHEALTH_<DEPENDENCY_NAME>_<PARAM>=<value>
```

- `<DEPENDENCY_NAME>` — имя зависимости в UPPER_SNAKE_CASE.
  Символ `-` заменяется на `_`.
- `<PARAM>` — параметр конфигурации.

### 8.2. Поддерживаемые параметры

| Переменная | Описание | Пример |
| --- | --- | --- |
| `DEPHEALTH_<NAME>_URL` | URL зависимости | `DEPHEALTH_POSTGRES_MAIN_URL=postgres://...` |
| `DEPHEALTH_<NAME>_HOST` | Хост | `DEPHEALTH_POSTGRES_MAIN_HOST=pg.svc` |
| `DEPHEALTH_<NAME>_PORT` | Порт | `DEPHEALTH_POSTGRES_MAIN_PORT=5432` |
| `DEPHEALTH_<NAME>_TYPE` | Тип зависимости | `DEPHEALTH_POSTGRES_MAIN_TYPE=postgres` |
| `DEPHEALTH_<NAME>_CHECK_INTERVAL` | Интервал проверки | `DEPHEALTH_POSTGRES_MAIN_CHECK_INTERVAL=30s` |
| `DEPHEALTH_<NAME>_TIMEOUT` | Таймаут | `DEPHEALTH_POSTGRES_MAIN_TIMEOUT=10s` |
| `DEPHEALTH_<NAME>_CRITICAL` | Критичность | `DEPHEALTH_POSTGRES_MAIN_CRITICAL=true` |
| `DEPHEALTH_<NAME>_HEALTH_PATH` | HTTP health path | `DEPHEALTH_PAYMENT_SERVICE_HEALTH_PATH=/ready` |

### 8.3. Приоритет

При конфликте значений:

1. Программный API (наивысший приоритет).
2. Переменные окружения.
3. Значения по умолчанию (наименьший приоритет).

### 8.4. Автоматическое обнаружение

SDK **не** сканирует переменные окружения автоматически.
Для загрузки конфигурации из env vars разработчик вызывает:

```go
dephealth.FromEnv("POSTGRES_MAIN") // ищет DEPHEALTH_POSTGRES_MAIN_*
```

Это создаёт Source, аналогичный `FromURL` или `FromParams`,
но читающий значения из переменных окружения.

**Обоснование**: неявное сканирование env vars создаёт проблемы с безопасностью
и усложняет отладку. Явный вызов `FromEnv` делает конфигурацию прозрачной.

---

## 9. Обработка специальных случаев

### 9.1. IPv6 адреса

| Ввод | Извлечённый host | port |
| --- | --- | --- |
| `postgres://[::1]:5432/db` | `::1` | `5432` |
| `postgres://[2001:db8::1]:5432/db` | `2001:db8::1` | `5432` |
| `Host=[::1];Port=5432` | `::1` | `5432` |
| `FromParams("::1", "5432")` | `::1` | `5432` |

В метке `host` метрики IPv6 записывается **без** квадратных скобок.

### 9.2. URL без пользователя / пароля

```text
postgres://pg.svc:5432/orders
```

Валидный URL. Credentials не обязательны для парсинга host/port.
В автономном режиме credentials для проверки могут быть переданы отдельно.

### 9.3. URL с пустым путём

```text
redis://redis.svc:6379
redis://redis.svc:6379/
```

Оба валидны. Для Redis пустой путь эквивалентен базе `0`.

### 9.4. Пустой или null URL

Ошибка конфигурации: `missing URL for dependency "..."`.
