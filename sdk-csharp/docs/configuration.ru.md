*[English version](configuration.md)*

# Конфигурация

Руководство по всем параметрам конфигурации C# SDK dephealth: глобальные
настройки, опции зависимостей, переменные окружения и правила валидации.

## Имя и группа

```csharp
var dh = DepHealthMonitor.CreateBuilder("my-service", "my-team")
    // ... зависимости
    .Build();
```

| Параметр | Обязателен | Валидация | Fallback из env |
| --- | --- | --- | --- |
| `name` | Да | `[a-z][a-z0-9-]*`, 1-63 символов | `DEPHEALTH_NAME` |
| `group` | Да | `[a-z][a-z0-9-]*`, 1-63 символов | `DEPHEALTH_GROUP` |

Приоритет: аргумент API > переменная окружения.

Если оба пусты, `CreateBuilder()` выбрасывает `ValidationException`.

## Глобальные опции

Глобальные опции задаются на `DepHealthMonitor.Builder` и применяются ко всем
зависимостям, если не переопределены для конкретной зависимости.

| Опция | Тип | По умолчанию | Диапазон | Описание |
| --- | --- | --- | --- | --- |
| `WithCheckInterval(TimeSpan)` | `TimeSpan` | 15 сек | 1с -- 10м | Интервал между проверками |
| `WithCheckTimeout(TimeSpan)` | `TimeSpan` | 5 сек | 100мс -- 30с | Таймаут одной проверки |
| `WithInitialDelay(TimeSpan)` | `TimeSpan` | 0 сек | 0 -- 5м | Задержка перед первой проверкой |
| `WithRegistry(CollectorRegistry)` | `CollectorRegistry` | дефолтный | -- | Пользовательский Prometheus-реестр |
| `WithLogger(ILogger)` | `ILogger` | null | -- | Логгер для диагностических сообщений |

### Пример

```csharp
var dh = DepHealthMonitor.CreateBuilder("my-service", "my-team")
    .WithCheckInterval(TimeSpan.FromSeconds(30))
    .WithCheckTimeout(TimeSpan.FromSeconds(3))
    .WithInitialDelay(TimeSpan.FromSeconds(5))
    // ... зависимости
    .Build();
```

## Общие опции зависимостей

Эти опции применимы к любому типу зависимости через методы `Add*` строителя.

| Опция | Обязательна | По умолчанию | Описание |
| --- | --- | --- | --- |
| `url` / `host` + `port` | Одна из url/host+port | -- | Спецификация эндпоинта (см. ниже) |
| `critical` | Да | -- | Пометить как критичную (`true`) или нет (`false`) |
| `labels` | Нет | -- | Пользовательские метки Prometheus (`Dictionary<string, string>`) |

### Указание эндпоинта

Каждая зависимость требует эндпоинт. Используйте один из двух способов:

```csharp
// Из URL -- SDK парсит host и port
builder.AddPostgres("postgres-main",
    "postgresql://user:pass@pg.svc:5432/mydb",
    critical: true)

// Из JDBC URL -- SDK парсит host и port
builder.AddPostgres("postgres-main",
    "jdbc:postgresql://pg.svc:5432/mydb",
    critical: true)

// Из явных host и port (gRPC, TCP, LDAP)
builder.AddGrpc("payments-grpc",
    host: "payments.svc",
    port: "9090",
    critical: true)
```

Поддерживаемые схемы URL: `http`, `https`, `grpc`, `tcp`, `postgresql`,
`postgres`, `mysql`, `redis`, `rediss`, `amqp`, `amqps`, `kafka`,
`ldap`, `ldaps`.

Поддерживаемые JDBC-субпротоколы: `jdbc:postgresql://...`,
`jdbc:mysql://...`.

Для Kafka поддерживаются multi-host URL:
`kafka://broker1:9092,broker2:9092` -- каждый хост создаёт отдельный эндпоинт.

### Флаг критичности

Опция `critical` **обязательна** для каждой зависимости. Если не указана,
при вызове `Build()` возникает ошибка валидации. При отсутствии в API SDK
проверяет переменную окружения `DEPHEALTH_<DEP>_CRITICAL` (значения:
`yes`/`no`, `true`/`false`).

### Пользовательские метки

```csharp
builder.AddPostgres("postgres-main",
    url: Environment.GetEnvironmentVariable("DATABASE_URL")!,
    critical: true,
    labels: new Dictionary<string, string>
    {
        ["role"] = "primary",
        ["shard"] = "eu-west"
    })
```

Валидация имён меток:

- Должно соответствовать `[a-zA-Z_][a-zA-Z0-9_]*`
- Нельзя использовать зарезервированные имена: `name`, `group`, `dependency`, `type`, `host`, `port`, `critical`

## Опции, специфичные для чекеров

### HTTP

```csharp
builder.AddHttp(
    name: "payment-api",
    url: "https://payment.svc:8443",
    healthPath: "/healthz",
    critical: true,
    headers: new Dictionary<string, string> { ["X-Internal"] = "1" },
    bearerToken: "secret-token")
```

| Опция | По умолчанию | Описание |
| --- | --- | --- |
| `healthPath` | `/health` | Путь эндпоинта проверки |
| `headers` | -- | Пользовательские HTTP-заголовки (`Dictionary<string, string>`) |
| `bearerToken` | -- | Аутентификация Bearer-токеном |
| `basicAuthUsername` | -- | Логин для Basic-аутентификации |
| `basicAuthPassword` | -- | Пароль для Basic-аутентификации |

TLS включается автоматически для URL с `https://`. Одновременно можно
указать только один метод аутентификации (`bearerToken`, `basicAuth` или
заголовок `Authorization`); смешивание выбросит `ValidationException`.

### gRPC

```csharp
builder.AddGrpc(
    name: "inventory-grpc",
    host: "inventory.svc",
    port: "9090",
    tlsEnabled: true,
    critical: false,
    metadata: new Dictionary<string, string> { ["x-request-id"] = "health" },
    bearerToken: "token")
```

| Опция | По умолчанию | Описание |
| --- | --- | --- |
| `tlsEnabled` | `false` | Включить TLS для gRPC-канала |
| `metadata` | -- | Пользовательские gRPC-метаданные (`Dictionary<string, string>`) |
| `bearerToken` | -- | Аутентификация Bearer-токеном |
| `basicAuthUsername` | -- | Логин для Basic-аутентификации |
| `basicAuthPassword` | -- | Пароль для Basic-аутентификации |

gRPC-чекер использует стандартный
[gRPC Health Checking Protocol](https://github.com/grpc/grpc/blob/master/doc/health-checking.md).
Одновременно допустим только один метод аутентификации.

### PostgreSQL

```csharp
// Автономный режим -- новое соединение на каждую проверку
builder.AddPostgres("db-primary",
    url: "postgresql://user:pass@pg.svc:5432/mydb",
    critical: true)

// Режим пула -- повторное использование NpgsqlDataSource (предпочтительно)
var dataSource = NpgsqlDataSource.Create(connectionString);
builder.AddCustom("db-primary", DependencyType.Postgres,
    host: "pg.svc", port: "5432",
    checker: new PostgresChecker(dataSource),
    critical: true)

// Интеграция с Entity Framework
builder.AddNpgsqlFromContext("db-primary", dbContext, critical: true)
```

| Опция | По умолчанию | Описание |
| --- | --- | --- |
| `url` | -- | URL подключения к PostgreSQL (credentials парсятся автоматически) |
| `NpgsqlDataSource` | -- | Существующий DataSource для интеграции с пулом (предпочтительно) |

Учётные данные (пользователь, пароль, база данных) извлекаются из URL
автоматически. SQL-запрос для проверки: `SELECT 1`.

### MySQL

```csharp
// Автономный режим -- новое соединение на каждую проверку
builder.AddMySql("mysql-main",
    url: "mysql://user:pass@mysql.svc:3306/mydb",
    critical: true)
```

| Опция | По умолчанию | Описание |
| --- | --- | --- |
| `url` | -- | URL подключения к MySQL (credentials парсятся автоматически) |

SQL-запрос для проверки: `SELECT 1`.

### Redis

```csharp
// Автономный режим -- новое соединение на каждую проверку
builder.AddRedis("cache",
    url: "redis://:password@redis.svc:6379",
    critical: false)

// Режим пула -- повторное использование IConnectionMultiplexer (предпочтительно)
var multiplexer = await ConnectionMultiplexer.ConnectAsync("redis.svc:6379");
builder.AddCustom("cache", DependencyType.Redis,
    host: "redis.svc", port: "6379",
    checker: new RedisChecker(multiplexer),
    critical: false)
```

| Опция | По умолчанию | Описание |
| --- | --- | --- |
| `url` | -- | URL подключения к Redis (`redis://` или `rediss://`) |
| `IConnectionMultiplexer` | -- | Существующий мультиплексор для интеграции с пулом (предпочтительно) |

Проверка Redis выполняется командой `PING`, ожидается ответ `PONG`.

### AMQP

```csharp
builder.AddAmqp("rabbitmq",
    url: "amqp://user:pass@rabbit.svc:5672/my-vhost",
    critical: true)
```

| Опция | По умолчанию | Описание |
| --- | --- | --- |
| `url` | -- | URL подключения к AMQP (credentials и vhost парсятся автоматически) |

Учётные данные (пользователь, пароль) и виртуальный хост извлекаются из URL.
Виртуальный хост по умолчанию -- `/`.

### LDAP

```csharp
builder.AddLdap(
    name: "ldap-corp",
    host: "ldap.corp.example.com",
    port: "389",
    checkMethod: LdapCheckMethod.SimpleBind,
    bindDN: "cn=health,dc=corp,dc=example,dc=com",
    bindPassword: "secret",
    critical: false)
```

| Опция | По умолчанию | Описание |
| --- | --- | --- |
| `checkMethod` | `RootDse` | Метод проверки: `AnonymousBind`, `SimpleBind`, `RootDse`, `Search` |
| `bindDN` | `""` | Bind DN для simple bind или поиска |
| `bindPassword` | `""` | Пароль привязки |
| `baseDN` | `""` | Базовый DN для поисковых операций |
| `searchFilter` | `(objectClass=*)` | Фильтр поиска LDAP |
| `searchScope` | `Base` | Область поиска: `Base`, `One`, `Sub` |
| `useTls` | `false` | Включить LDAPS (TLS) -- использовать совместно с `ldaps://` |
| `startTls` | `false` | Включить StartTLS (несовместимо с `useTls: true`) |
| `tlsSkipVerify` | `false` | Пропустить проверку TLS-сертификата |

### TCP и Kafka

```csharp
// TCP
builder.AddTcp("legacy-tcp", host: "legacy.svc", port: "8080", critical: false)

// Kafka
builder.AddKafka("events",
    url: "kafka://broker1:9092,broker2:9092",
    critical: true)
```

Нет специфичных опций для TCP и Kafka. TCP-чекер устанавливает сырое TCP-соединение.
Kafka-чекер подключается к каждому брокеру отдельно.

## Переменные окружения

| Переменная | Описание | Пример |
| --- | --- | --- |
| `DEPHEALTH_NAME` | Имя приложения (fallback, если аргумент API пуст) | `my-service` |
| `DEPHEALTH_GROUP` | Логическая группа (fallback, если аргумент API пуст) | `my-team` |
| `DEPHEALTH_<DEP>_CRITICAL` | Критичность зависимости (`yes`/`no`, `true`/`false`) | `yes` |
| `DEPHEALTH_<DEP>_LABEL_<KEY>` | Значение пользовательской метки | `primary` |

`<DEP>` -- имя зависимости в формате UPPER_SNAKE_CASE:
дефисы заменяются на подчёркивания, всё в верхнем регистре.

Пример: зависимость `"postgres-main"` даёт env-префикс `DEPHEALTH_POSTGRES_MAIN_`.

### Правила приоритета

Значения API всегда имеют приоритет над переменными окружения:

1. **name/group**: аргумент API > `DEPHEALTH_NAME`/`DEPHEALTH_GROUP` > ошибка
2. **critical**: опция `critical` > `DEPHEALTH_<DEP>_CRITICAL` > ошибка
3. **метки**: словарь `labels` > `DEPHEALTH_<DEP>_LABEL_<KEY>` (API побеждает при конфликте)

### Пример

```bash
export DEPHEALTH_NAME=my-service
export DEPHEALTH_GROUP=my-team
export DEPHEALTH_POSTGRES_MAIN_CRITICAL=yes
export DEPHEALTH_POSTGRES_MAIN_LABEL_ROLE=primary
export DEPHEALTH_POSTGRES_MAIN_LABEL_SHARD=eu-west
```

```csharp
// name и group из env vars, critical и labels из env vars
var dh = DepHealthMonitor.CreateBuilder("", "")
    .AddPostgres("postgres-main",
        url: Environment.GetEnvironmentVariable("DATABASE_URL")!)
        // critical и labels берутся из DEPHEALTH_POSTGRES_MAIN_*
    .Build();
```

## Приоритет опций

Для интервала и таймаута цепочка приоритетов:

```text
опция зависимости > глобальная опция > значение по умолчанию
```

В C# SDK глобальные опции применяются ко всем зависимостям во время сборки.
Переопределение на уровне зависимости можно реализовать через `AddCustom`
с явно сконструированным `CheckConfig`.

| Настройка | Глобальная опция | По умолчанию |
| --- | --- | --- |
| Интервал проверки | `WithCheckInterval(TimeSpan)` | 15 сек |
| Таймаут | `WithCheckTimeout(TimeSpan)` | 5 сек |
| Начальная задержка | `WithInitialDelay(TimeSpan)` | 0 сек |

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
| gRPC TLS | `false` |
| LDAP check method | `RootDse` |
| LDAP search filter | `(objectClass=*)` |
| LDAP search scope | `Base` |

## Правила валидации

`Build()` валидирует всю конфигурацию и выбрасывает `ValidationException`
или `ConfigurationException` при нарушении любого правила:

| Правило | Сообщение об ошибке |
| --- | --- |
| Не указано имя | `instance name must not be empty` |
| Не указана группа | `group is required: pass to CreateBuilder() or set DEPHEALTH_GROUP` |
| Неверный формат имени/группы | `instance name must match ^[a-z][a-z0-9-]*$, got '...'` |
| Имя слишком длинное | `instance name must be 1-63 characters, got '...' (N chars)` |
| Не указан URL или host/port | `URL must have a scheme (e.g. http://)` |
| Неподдерживаемая схема URL | `Unsupported URL scheme: ...` |
| Неверный порт | `Invalid port: '...' in ...` |
| Порт вне диапазона | `Port out of range (1-65535): ... in ...` |
| Неверное имя метки | `label name must match [a-zA-Z_][a-zA-Z0-9_]*, got '...'` |
| Зарезервированное имя метки | `label name '...' is reserved` |
| Таймаут >= интервал | `timeout (...) must be less than interval (...)` |
| Интервал вне диапазона | `interval must be between 00:00:01 and 00:10:00, got ...` |
| Таймаут вне диапазона | `timeout must be between 00:00:00.1000000 and 00:00:30, got ...` |
| Конфликт методов аутентификации (HTTP/gRPC) | `conflicting auth methods: specify only one of bearerToken, basicAuth, or Authorization header` |
| LDAP SimpleBind без credentials | `LDAP simple_bind requires bindDN and bindPassword` |
| LDAP Search без baseDN | `LDAP search requires baseDN` |
| LDAP startTLS + useTls | `startTLS and ldaps:// are incompatible` |

## См. также

- [Начало работы](getting-started.ru.md) -- базовая настройка и первый пример
- [Чекеры](checkers.ru.md) -- специфичные опции чекеров подробно
- [Аутентификация](authentication.ru.md) -- опции аутентификации для HTTP и gRPC
- [Пулы соединений](connection-pools.ru.md) -- интеграция с NpgsqlDataSource и IConnectionMultiplexer
- [Интеграция с ASP.NET Core](aspnetcore.ru.md) -- hosted service и health checks
- [API Reference](api-reference.ru.md) -- полный справочник по всем классам
- [Устранение неполадок](troubleshooting.ru.md) -- типичные проблемы и решения
