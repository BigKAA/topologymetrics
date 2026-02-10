[English](#english) | [Русский](#russian)

---

<a id="english"></a>

# Configuration Contract

> Specification version: **2.0-draft**
>
> This document describes the input data formats for dependency configuration,
> parsing rules, programmatic API, and configuration via environment variables.
> All SDKs must support the described formats. Compliance is verified
> by conformance tests.

---

## 1. Supported Input Formats

SDK accepts connection information for a dependency in four formats:

| # | Format | Example | Priority |
| --- | --- | --- | --- |
| 1 | Full URL | `postgres://user:pass@host:5432/db` | Primary |
| 2 | Separate parameters | `host=pg.svc, port=5432` | Alternative |
| 3 | Connection string | `Host=pg.svc;Port=5432;Database=db` | For .NET/JDBC |
| 4 | JDBC URL | `jdbc:postgresql://host:5432/db` | For Java |

SDK must correctly extract from any format:

- `host` — endpoint address
- `port` — endpoint port
- `type` — dependency type (if determined from URL scheme)

---

## 2. Format 1: Full URL

### 2.1. Supported Schemes

| Scheme | Determined `type` | Default Port |
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

### 2.2. URL Parsing Rules

General format:

```text
scheme://[user:password@]host[:port][/path][?query]
```

**Extracting host**:

- From the `host` component of the URL.
- IPv6 addresses in URLs are enclosed in square brackets: `[::1]`.
- In the metric's `host` label, IPv6 is written **without** square brackets: `::1`.

**Extracting port**:

- From the `port` component of the URL.
- If port is not specified, the default port for the given scheme is used
  (see table above).

**Extracting type**:

- Automatically from the URL scheme (see table above).
- If the developer explicitly specified `type`, it takes priority.

**Extracting additional data**:

| Data | Source | Used in |
| --- | --- | --- |
| `vhost` | Path component (`amqp://host/vhost`) | `vhost` label, connection to AMQP |
| `database` | Path component (`redis://host/0`) | Redis database selection |
| Credentials | Userinfo (`user:pass@`) | Authentication during autonomous check |

If credentials are specified in the URL, they **MUST** be passed to the checker
for autonomous checking.
Priority: explicit API parameters > credentials from URL > default values.

### 2.3. URLs with Multiple Hosts

Some dependencies support specifying multiple hosts in a URL:

```text
# Kafka brokers
kafka://broker-0:9092,broker-1:9092,broker-2:9092

# PostgreSQL failover (libpq format)
postgres://user:pass@primary:5432,replica:5432/db?target_session_attrs=read-write
```

**Rules**:

- Each `host:port` pair creates a separate `Endpoint`.
- All endpoints are combined into one `Dependency`.
- If port is specified only for the first host, the rest use the same port.
- If port is not specified for any — the default port is used.

### 2.4. URL Parsing Examples

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

## 3. Format 2: Separate Parameters

Minimum configuration: `host` + `port`.

```go
dephealth.Postgres("postgres-main", dephealth.FromParams("pg.svc", "5432"))
```

**Rules**:

- `host` — required. String (hostname, IPv4 or IPv6).
- `port` — required. String with number 1-65535.
- `type` — must be specified explicitly (via factory method or parameter).
- For invalid port (not a number, out of range) — configuration error.

---

## 4. Format 3: Connection String

Format `Key=Value;Key=Value`, common in .NET ecosystem.

### 4.1. Supported Keys for host

Search (case-insensitive, first matching result):

| Key | Description |
| --- | --- |
| `Host` | Primary |
| `Server` | Alternative (SQL Server) |
| `Data Source` | Alternative (Oracle, SQLite) |
| `Address` | Alternative |
| `Addr` | Alternative |
| `Network Address` | Alternative |

### 4.2. Supported Keys for port

| Key | Description |
| --- | --- |
| `Port` | Primary |

If port is not found as a separate key, the format `Host=hostname,port`
(SQL Server convention) and `Host=hostname:port` are checked.

### 4.3. Connection String Parsing Examples

| Connection string | host | port |
| --- | --- | --- |
| `Host=pg.svc;Port=5432;Database=orders` | `pg.svc` | `5432` |
| `Server=pg.svc,5432;Database=orders` | `pg.svc` | `5432` |
| `Host=pg.svc:5432;Database=orders` | `pg.svc` | `5432` |
| `Data Source=pg.svc;Port=5432` | `pg.svc` | `5432` |

### 4.4. Limitations

- `type` is not determined automatically from connection string.
  Developer must specify type explicitly.
- If `host` is not found — configuration error.
- If `port` is not found — default port for specified `type` is used.

---

## 5. Format 4: JDBC URL

Format specific to Java ecosystem.

### 5.1. General Format

```text
jdbc:<subprotocol>://host[:port][/database][?parameters]
```

### 5.2. Supported Subprotocols

| Subprotocol | Determined `type` | Default Port |
| --- | --- | --- |
| `postgresql` | `postgres` | `5432` |
| `mysql` | `mysql` | `3306` |

### 5.3. Parsing Rules

1. Remove `jdbc:` prefix.
2. Parse the remaining part as a standard URL.
3. Extract `host`, `port`, `type` using the same rules.

### 5.4. Examples

| JDBC URL | host | port | type |
| --- | --- | --- | --- |
| `jdbc:postgresql://pg.svc:5432/orders` | `pg.svc` | `5432` | `postgres` |
| `jdbc:postgresql://pg.svc/orders` | `pg.svc` | `5432` | `postgres` |
| `jdbc:mysql://mysql.svc:3306/orders` | `mysql.svc` | `3306` | `mysql` |

---

## 6. Default Port Table

| Type | Default Port | Protocol |
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
| `tcp` | — | TCP (port is required) |

For `type: tcp` the default port is **not defined** — developer must specify it explicitly.

---

## 7. Programmatic Configuration API

### 7.1. Factory Methods (Dependency Creation)

Each dependency type has a factory method:

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

**Parameters**:

- `name` (required) — logical dependency name. Format: `[a-z][a-z0-9-]*`,
  length 1-63 characters. For invalid name — configuration error.
- `source` (required) — configuration source (URL, parameters, pool).
- `opts` (optional) — additional settings (including mandatory `critical`).

### 7.2. Configuration Sources (Source)

```go
// From URL
dephealth.FromURL("postgres://user:pass@host:5432/db")

// From separate parameters
dephealth.FromParams("host", "5432")

// From connection string
dephealth.FromConnectionString("Host=pg.svc;Port=5432;Database=orders")

// From JDBC URL
dephealth.FromJDBC("jdbc:postgresql://host:5432/db")
```

### 7.3. Dependency Options (DependencyOption)

```go
// Criticality — required for each dependency, no default value.
// If not specified — configuration error.
dephealth.Critical(true)  // critical="yes"
dephealth.Critical(false) // critical="no"

// Individual check interval
dephealth.WithCheckInterval(30 * time.Second)

// Individual timeout
dephealth.WithTimeout(10 * time.Second)

// Individual initial delay
dephealth.WithInitialDelay(0)

// Thresholds
dephealth.WithFailureThreshold(3)
dephealth.WithSuccessThreshold(2)

// HTTP-specific
dephealth.WithHealthPath("/ready")
dephealth.WithTLSSkipVerify(true)

// Custom labels
dephealth.WithLabel("role", "primary")
dephealth.WithLabel("shard", "shard-01")
dephealth.WithLabel("vhost", "orders")
```

### 7.4. Global Options (When Creating DepHealth)

```go
dh := dephealth.New("order-api",
    // Global options
    dephealth.WithDefaultCheckInterval(30 * time.Second),
    dephealth.WithDefaultTimeout(10 * time.Second),
    dephealth.WithRegisterer(customRegisterer),
    dephealth.WithLogger(slog.Default()),

    // Dependencies (critical is required for each)
    dephealth.Postgres("postgres-main", dephealth.FromURL(url),
        dephealth.Critical(true)),
    dephealth.Redis("redis-cache", dephealth.FromURL(url),
        dephealth.Critical(false)),
)
```

**Parameters**:

- `name` (required) — unique application name. Format: `[a-z][a-z0-9-]*`,
  length 1-63 characters. Alternatively set via `DEPHEALTH_NAME`
  (API > env var). If missing in both API and env — configuration error.

### 7.5. Validation

SDK validates configuration on `New()` call and returns error for:

| Issue | Error |
| --- | --- |
| Missing application name | `missing name` |
| Invalid application name | `invalid name: "..."` |
| Invalid dependency name | `invalid dependency name: "..."` |
| Missing critical | `missing critical for dependency "..."` |
| Duplicate name + host + port | `duplicate endpoint: "..." host=... port=...` |
| Invalid URL | `invalid URL: "..."` |
| Missing host | `missing host for dependency "..."` |
| Invalid port | `invalid port "..." for dependency "..."` |
| `timeout >= checkInterval` | `timeout must be less than checkInterval for "..."` |
| Unknown type | `unknown dependency type: "..."` |
| Invalid label name | `invalid label name: "..."` |
| Reserved label | `reserved label: "..."` |

### 7.6. Valid Configurations

Configuration without dependencies (zero dependencies) is valid.
Leaf services (without outgoing dependencies) are an acceptable pattern
in microservice topology.

**Behavior with zero dependencies**:

| Operation | Behavior |
| --- | --- |
| `New()` / `build()` | Completes without error |
| `Health()` | Returns empty collection |
| `Start()` / `Stop()` | No-op (does not create threads/goroutines) |
| Metrics | Contains no time series |

---

## 8. Configuration via Environment Variables

### 8.1. Format

Instance-level variable:

```text
DEPHEALTH_NAME=<value>
```

Dependency-level variables:

```text
DEPHEALTH_<DEPENDENCY_NAME>_<PARAM>=<value>
```

- `<DEPENDENCY_NAME>` — dependency name in UPPER_SNAKE_CASE.
  Character `-` is replaced with `_`.
- `<PARAM>` — configuration parameter.

**Duration value format**: numbers in seconds (integer or fractional), without unit suffix.
For example: `CHECK_INTERVAL=30`, `TIMEOUT=5`. SDK converts the number to native duration type.

### 8.2. Supported Parameters

#### Instance Level

| Variable | Description | Example |
| --- | --- | --- |
| `DEPHEALTH_NAME` | Unique application name (`name` label) | `DEPHEALTH_NAME=order-api` |

#### Dependency Level

| Variable | Description | Example |
| --- | --- | --- |
| `DEPHEALTH_<NAME>_URL` | Dependency URL | `DEPHEALTH_POSTGRES_MAIN_URL=postgres://...` |
| `DEPHEALTH_<NAME>_HOST` | Host | `DEPHEALTH_POSTGRES_MAIN_HOST=pg.svc` |
| `DEPHEALTH_<NAME>_PORT` | Port | `DEPHEALTH_POSTGRES_MAIN_PORT=5432` |
| `DEPHEALTH_<NAME>_TYPE` | Dependency type | `DEPHEALTH_POSTGRES_MAIN_TYPE=postgres` |
| `DEPHEALTH_<NAME>_CHECK_INTERVAL` | Check interval (seconds) | `DEPHEALTH_POSTGRES_MAIN_CHECK_INTERVAL=30` |
| `DEPHEALTH_<NAME>_TIMEOUT` | Timeout (seconds) | `DEPHEALTH_POSTGRES_MAIN_TIMEOUT=10` |
| `DEPHEALTH_<NAME>_CRITICAL` | Criticality (`yes` / `no`) | `DEPHEALTH_POSTGRES_MAIN_CRITICAL=yes` |
| `DEPHEALTH_<NAME>_HEALTH_PATH` | HTTP health path | `DEPHEALTH_PAYMENT_SERVICE_HEALTH_PATH=/ready` |

#### Custom Labels

| Variable | Description | Example |
| --- | --- | --- |
| `DEPHEALTH_<NAME>_LABEL_<KEY>` | Custom label | `DEPHEALTH_POSTGRES_MAIN_LABEL_ROLE=primary` |

`<KEY>` — label name in UPPER_SNAKE_CASE, converted to lower_snake_case
in the metric.

### 8.3. Priority

When values conflict:

1. Programmatic API (highest priority).
2. Environment variables.
3. Default values (lowest priority).

### 8.4. Automatic Discovery

SDK does **not** scan environment variables automatically.
To load configuration from env vars, developer calls:

```go
dephealth.FromEnv("POSTGRES_MAIN") // searches for DEPHEALTH_POSTGRES_MAIN_*
```

This creates a Source, analogous to `FromURL` or `FromParams`,
but reads values from environment variables.

**Rationale**: implicit scanning of env vars creates security issues
and complicates debugging. Explicit `FromEnv` call makes configuration transparent.

---

## 9. Handling Special Cases

### 9.1. IPv6 Addresses

| Input | Extracted host | port |
| --- | --- | --- |
| `postgres://[::1]:5432/db` | `::1` | `5432` |
| `postgres://[2001:db8::1]:5432/db` | `2001:db8::1` | `5432` |
| `Host=[::1];Port=5432` | `::1` | `5432` |
| `FromParams("::1", "5432")` | `::1` | `5432` |

In the metric's `host` label, IPv6 is written **without** square brackets.

### 9.2. URL Without User/Password

```text
postgres://pg.svc:5432/orders
```

Valid URL. Credentials are not required for parsing host/port.
In autonomous mode, credentials for checking can be passed separately.

### 9.3. URL with Empty Path

```text
redis://redis.svc:6379
redis://redis.svc:6379/
```

Both are valid. For Redis, empty path is equivalent to database `0`.

### 9.4. Empty or null URL

Configuration error: `missing URL for dependency "..."`.

---

<a id="russian"></a>

# Контракт конфигурации

> Версия спецификации: **2.0-draft**
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

Если credentials указаны в URL, они **ДОЛЖНЫ** быть переданы checker
для автономной проверки.
Приоритет: явные параметры API > credentials из URL > дефолтные значения.

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
- `opts` (опциональные) — дополнительные настройки (включая обязательный `critical`).

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
// Критичность — обязательна для каждой зависимости, без значения по умолчанию.
// Если не указана — ошибка конфигурации.
dephealth.Critical(true)  // critical="yes"
dephealth.Critical(false) // critical="no"

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

// Произвольные метки (custom labels)
dephealth.WithLabel("role", "primary")
dephealth.WithLabel("shard", "shard-01")
dephealth.WithLabel("vhost", "orders")
```

### 7.4. Глобальные опции (при создании DepHealth)

```go
dh := dephealth.New("order-api",
    // Глобальные опции
    dephealth.WithDefaultCheckInterval(30 * time.Second),
    dephealth.WithDefaultTimeout(10 * time.Second),
    dephealth.WithRegisterer(customRegisterer),
    dephealth.WithLogger(slog.Default()),

    // Зависимости (critical обязателен для каждой)
    dephealth.Postgres("postgres-main", dephealth.FromURL(url),
        dephealth.Critical(true)),
    dephealth.Redis("redis-cache", dephealth.FromURL(url),
        dephealth.Critical(false)),
)
```

**Параметры**:

- `name` (обязательный) — уникальное имя приложения. Формат: `[a-z][a-z0-9-]*`,
  длина 1-63 символа. Альтернативно задаётся через `DEPHEALTH_NAME`
  (API > env var). При отсутствии и в API, и в env — ошибка конфигурации.

### 7.5. Валидация

SDK валидирует конфигурацию при вызове `New()` и возвращает ошибку при:

| Проблема | Ошибка |
| --- | --- |
| Отсутствует имя приложения | `missing name` |
| Невалидное имя приложения | `invalid name: "..."` |
| Невалидное имя зависимости | `invalid dependency name: "..."` |
| Отсутствует critical | `missing critical for dependency "..."` |
| Дублирующееся имя + host + port | `duplicate endpoint: "..." host=... port=...` |
| Невалидный URL | `invalid URL: "..."` |
| Отсутствует host | `missing host for dependency "..."` |
| Невалидный порт | `invalid port "..." for dependency "..."` |
| `timeout >= checkInterval` | `timeout must be less than checkInterval for "..."` |
| Неизвестный тип | `unknown dependency type: "..."` |
| Невалидное имя метки | `invalid label name: "..."` |
| Зарезервированная метка | `reserved label: "..."` |

### 7.6. Допустимые конфигурации

Конфигурация без зависимостей (zero dependencies) является валидной.
Leaf-сервисы (без исходящих зависимостей) — допустимый паттерн
в микросервисной топологии.

**Поведение при zero dependencies**:

| Операция | Поведение |
| --- | --- |
| `New()` / `build()` | Завершается без ошибки |
| `Health()` | Возвращает пустую коллекцию |
| `Start()` / `Stop()` | No-op (не создаёт потоков/горутин) |
| Метрики | Не содержат серий данных |

---

## 8. Конфигурация через переменные окружения

### 8.1. Формат

Переменная уровня экземпляра:

```text
DEPHEALTH_NAME=<value>
```

Переменные уровня зависимости:

```text
DEPHEALTH_<DEPENDENCY_NAME>_<PARAM>=<value>
```

- `<DEPENDENCY_NAME>` — имя зависимости в UPPER_SNAKE_CASE.
  Символ `-` заменяется на `_`.
- `<PARAM>` — параметр конфигурации.

**Формат значений длительности**: числа в секундах (целые или дробные), без суффикса единицы измерения.
Например: `CHECK_INTERVAL=30`, `TIMEOUT=5`. SDK самостоятельно преобразует число в нативный тип длительности.

### 8.2. Поддерживаемые параметры

#### Уровень экземпляра

| Переменная | Описание | Пример |
| --- | --- | --- |
| `DEPHEALTH_NAME` | Уникальное имя приложения (метка `name`) | `DEPHEALTH_NAME=order-api` |

#### Уровень зависимости

| Переменная | Описание | Пример |
| --- | --- | --- |
| `DEPHEALTH_<NAME>_URL` | URL зависимости | `DEPHEALTH_POSTGRES_MAIN_URL=postgres://...` |
| `DEPHEALTH_<NAME>_HOST` | Хост | `DEPHEALTH_POSTGRES_MAIN_HOST=pg.svc` |
| `DEPHEALTH_<NAME>_PORT` | Порт | `DEPHEALTH_POSTGRES_MAIN_PORT=5432` |
| `DEPHEALTH_<NAME>_TYPE` | Тип зависимости | `DEPHEALTH_POSTGRES_MAIN_TYPE=postgres` |
| `DEPHEALTH_<NAME>_CHECK_INTERVAL` | Интервал проверки (секунды) | `DEPHEALTH_POSTGRES_MAIN_CHECK_INTERVAL=30` |
| `DEPHEALTH_<NAME>_TIMEOUT` | Таймаут (секунды) | `DEPHEALTH_POSTGRES_MAIN_TIMEOUT=10` |
| `DEPHEALTH_<NAME>_CRITICAL` | Критичность (`yes` / `no`) | `DEPHEALTH_POSTGRES_MAIN_CRITICAL=yes` |
| `DEPHEALTH_<NAME>_HEALTH_PATH` | HTTP health path | `DEPHEALTH_PAYMENT_SERVICE_HEALTH_PATH=/ready` |

#### Произвольные метки

| Переменная | Описание | Пример |
| --- | --- | --- |
| `DEPHEALTH_<NAME>_LABEL_<KEY>` | Произвольная метка | `DEPHEALTH_POSTGRES_MAIN_LABEL_ROLE=primary` |

`<KEY>` — имя метки в UPPER_SNAKE_CASE, преобразуется в lower_snake_case
в метрике.

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
