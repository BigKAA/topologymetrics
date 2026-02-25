*[English version](configuration.md)*

# Конфигурация

Руководство по всем параметрам конфигурации Java SDK dephealth: глобальные
настройки, опции зависимостей, переменные окружения и правила валидации.

## Имя и группа

```java
DepHealth dh = DepHealth.builder("my-service", "my-team", meterRegistry)
    // ... зависимости
    .build();
```

| Параметр | Обязателен | Валидация | Fallback из env |
| --- | --- | --- | --- |
| `name` | Да | `[a-z][a-z0-9-]*`, 1-63 символов | `DEPHEALTH_NAME` |
| `group` | Да | `[a-z][a-z0-9-]*`, 1-63 символов | `DEPHEALTH_GROUP` |

Приоритет: аргумент API > переменная окружения.

Если оба пусты, `builder()` выбрасывает `ConfigurationException`.

## Глобальные опции

Глобальные опции задаются на `DepHealth.Builder` и применяются ко всем
зависимостям, если не переопределены для конкретной зависимости.

| Опция | Тип | По умолчанию | Диапазон | Описание |
| --- | --- | --- | --- | --- |
| `checkInterval(Duration)` | `Duration` | 15 сек | 1с -- 10м | Интервал между проверками |
| `timeout(Duration)` | `Duration` | 5 сек | 100мс -- 30с | Таймаут одной проверки |

Третий параметр `builder()` -- Micrometer `MeterRegistry`, используемый для
экспорта метрик Prometheus. Переопределить его после создания нельзя.

### Пример

```java
DepHealth dh = DepHealth.builder("my-service", "my-team", meterRegistry)
    .checkInterval(Duration.ofSeconds(30))
    .timeout(Duration.ofSeconds(3))
    // ... зависимости
    .build();
```

## Общие опции зависимостей

Эти опции применимы к любому типу зависимости внутри лямбды
`.dependency()`.

| Опция | Обязательна | По умолчанию | Описание |
| --- | --- | --- | --- |
| `url(String)` | Одна из url/jdbcUrl/host+port | -- | Парсинг host и port из URL |
| `jdbcUrl(String)` | Одна из url/jdbcUrl/host+port | -- | Парсинг host и port из JDBC URL |
| `host(String)` + `port(String)` | Одна из url/jdbcUrl/host+port | -- | Явное указание host и port |
| `critical(boolean)` | Да | -- | Пометить как критичную (`true`) или нет (`false`) |
| `label(String, String)` | Нет | -- | Добавить пользовательскую метку Prometheus |
| `interval(Duration)` | Нет | глобальное значение | Интервал проверки для зависимости |
| `timeout(Duration)` | Нет | глобальное значение | Таймаут для зависимости |

### Указание эндпоинта

Каждая зависимость требует эндпоинт. Используйте один из трёх способов:

```java
// Из URL -- SDK парсит host и port
.dependency("postgres-main", DependencyType.POSTGRES, d -> d
    .url("postgresql://user:pass@pg.svc:5432/mydb")
    .critical(true))

// Из JDBC URL -- SDK парсит host и port
.dependency("postgres-main", DependencyType.POSTGRES, d -> d
    .jdbcUrl("jdbc:postgresql://pg.svc:5432/mydb")
    .critical(true))

// Из явных host и port
.dependency("postgres-main", DependencyType.POSTGRES, d -> d
    .host("pg.svc")
    .port("5432")
    .critical(true))
```

Поддерживаемые схемы URL: `http`, `https`, `grpc`, `tcp`, `postgresql`,
`postgres`, `mysql`, `redis`, `rediss`, `amqp`, `amqps`, `kafka`,
`ldap`, `ldaps`.

Поддерживаемые JDBC-субпротоколы: `jdbc:postgresql://...`,
`jdbc:mysql://...`.

Для Kafka поддерживаются multi-host URL:
`kafka://broker1:9092,broker2:9092` -- каждый хост создаёт отдельный эндпоинт.

### Флаг критичности

Опция `critical()` **обязательна** для каждой зависимости. Если не
указана, возникает ошибка валидации. При отсутствии в API SDK проверяет
переменную окружения `DEPHEALTH_<DEP>_CRITICAL` (значения: `yes`/`no`,
`true`/`false`).

### Пользовательские метки

```java
.dependency("postgres-main", DependencyType.POSTGRES, d -> d
    .url(System.getenv("DATABASE_URL"))
    .critical(true)
    .label("role", "primary")
    .label("shard", "eu-west"))
```

Валидация имён меток:

- Должно соответствовать `[a-zA-Z_][a-zA-Z0-9_]*`
- Нельзя использовать зарезервированные имена: `name`, `group`, `dependency`, `type`, `host`, `port`, `critical`

## Опции, специфичные для чекеров

### HTTP

| Опция | По умолчанию | Описание |
| --- | --- | --- |
| `httpHealthPath(String)` | `/health` | Путь эндпоинта проверки |
| `httpTls(boolean)` | авто (true для `https://`) | Включить HTTPS |
| `httpTlsSkipVerify(boolean)` | `false` | Пропустить проверку TLS-сертификата |
| `httpHeaders(Map<String, String>)` | -- | Пользовательские HTTP-заголовки |
| `httpBearerToken(String)` | -- | Аутентификация Bearer-токеном |
| `httpBasicAuth(String, String)` | -- | Basic-аутентификация (логин, пароль) |

### gRPC

| Опция | По умолчанию | Описание |
| --- | --- | --- |
| `grpcServiceName(String)` | `""` | Имя сервиса (пустое = весь сервер) |
| `grpcTls(boolean)` | `false` | Включить TLS |
| `grpcMetadata(Map<String, String>)` | -- | Пользовательские gRPC-метаданные |
| `grpcBearerToken(String)` | -- | Аутентификация Bearer-токеном |
| `grpcBasicAuth(String, String)` | -- | Basic-аутентификация (логин, пароль) |

### PostgreSQL

| Опция | По умолчанию | Описание |
| --- | --- | --- |
| `dbUsername(String)` | из URL | Имя пользователя БД |
| `dbPassword(String)` | из URL | Пароль БД |
| `dbDatabase(String)` | из URL | Имя базы данных |
| `dbQuery(String)` | `SELECT 1` | SQL-запрос для проверки |
| `dataSource(DataSource)` | -- | DataSource пула соединений (предпочтительно) |

### MySQL

| Опция | По умолчанию | Описание |
| --- | --- | --- |
| `dbUsername(String)` | из URL | Имя пользователя БД |
| `dbPassword(String)` | из URL | Пароль БД |
| `dbDatabase(String)` | из URL | Имя базы данных |
| `dbQuery(String)` | `SELECT 1` | SQL-запрос для проверки |
| `dataSource(DataSource)` | -- | DataSource пула соединений (предпочтительно) |

### Redis

| Опция | По умолчанию | Описание |
| --- | --- | --- |
| `redisPassword(String)` | `""` | Пароль Redis (автономный режим) |
| `redisDb(int)` | `0` | Номер базы данных (автономный режим) |
| `jedisPool(JedisPool)` | -- | JedisPool для интеграции с пулом (предпочтительно) |

### AMQP

| Опция | По умолчанию | Описание |
| --- | --- | --- |
| `amqpUrl(String)` | -- | Полный AMQP URL (перекрывает host/port/credentials) |
| `amqpUsername(String)` | из URL | Имя пользователя AMQP |
| `amqpPassword(String)` | из URL | Пароль AMQP |
| `amqpVirtualHost(String)` | из URL | Виртуальный хост AMQP |

### LDAP

| Опция | По умолчанию | Описание |
| --- | --- | --- |
| `ldapCheckMethod(CheckMethod)` | `ROOT_DSE` | Метод проверки: `ANONYMOUS_BIND`, `SIMPLE_BIND`, `ROOT_DSE`, `SEARCH` |
| `ldapBindDN(String)` | `""` | Bind DN для simple bind или поиска |
| `ldapBindPassword(String)` | `""` | Пароль привязки |
| `ldapBaseDN(String)` | `""` | Базовый DN для поисковых операций |
| `ldapSearchFilter(String)` | `(objectClass=*)` | Фильтр поиска LDAP |
| `ldapSearchScope(LdapSearchScope)` | `BASE` | Область поиска: `BASE`, `ONE`, `SUB` |
| `ldapStartTLS(boolean)` | `false` | Включить StartTLS (несовместимо с `ldaps://`) |
| `ldapTlsSkipVerify(boolean)` | `false` | Пропустить проверку TLS-сертификата |
| `ldapConnection(LDAPConnection)` | -- | Существующее соединение для интеграции с пулом |

### TCP и Kafka

Нет специфичных опций.

## Переменные окружения

| Переменная | Описание | Пример |
| --- | --- | --- |
| `DEPHEALTH_NAME` | Имя приложения (fallback, если аргумент API пуст) | `my-service` |
| `DEPHEALTH_GROUP` | Логическая группа (fallback, если аргумент API пуст) | `my-team` |
| `DEPHEALTH_<DEP>_CRITICAL` | Критичность зависимости (`yes`/`no`) | `yes` |
| `DEPHEALTH_<DEP>_LABEL_<KEY>` | Значение пользовательской метки | `primary` |

`<DEP>` -- имя зависимости в формате UPPER_SNAKE_CASE:
дефисы заменяются на подчёркивания, всё в верхнем регистре.

Пример: зависимость `"postgres-main"` даёт env-префикс `DEPHEALTH_POSTGRES_MAIN_`.

### Правила приоритета

Значения API всегда имеют приоритет над переменными окружения:

1. **name/group**: аргумент API > `DEPHEALTH_NAME`/`DEPHEALTH_GROUP` > ошибка
2. **critical**: опция `critical()` > `DEPHEALTH_<DEP>_CRITICAL` > ошибка
3. **метки**: `label()` > `DEPHEALTH_<DEP>_LABEL_<KEY>` (API побеждает при конфликте)

### Пример

```bash
export DEPHEALTH_NAME=my-service
export DEPHEALTH_GROUP=my-team
export DEPHEALTH_POSTGRES_MAIN_CRITICAL=yes
export DEPHEALTH_POSTGRES_MAIN_LABEL_ROLE=primary
export DEPHEALTH_POSTGRES_MAIN_LABEL_SHARD=eu-west
```

```java
// name и group из env vars, critical и labels из env vars
DepHealth dh = DepHealth.builder("", "", meterRegistry)
    .dependency("postgres-main", DependencyType.POSTGRES, d -> d
        .url(System.getenv("DATABASE_URL")))
        // Critical и labels берутся из DEPHEALTH_POSTGRES_MAIN_*
    .build();
```

## Приоритет опций

Для интервала и таймаута цепочка приоритетов:

```text
опция зависимости > глобальная опция > значение по умолчанию
```

| Настройка | Per-dependency | Глобальная | По умолчанию |
| --- | --- | --- | --- |
| Интервал проверки | `interval(Duration)` | `checkInterval(Duration)` | 15 сек |
| Таймаут | `timeout(Duration)` | `timeout(Duration)` | 5 сек |

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
| gRPC service name | пусто (состояние всего сервера) |
| LDAP check method | `ROOT_DSE` |
| LDAP search filter | `(objectClass=*)` |
| LDAP search scope | `BASE` |

## Правила валидации

`build()` валидирует всю конфигурацию и выбрасывает `ConfigurationException`
или `ValidationException` при нарушении любого правила:

| Правило | Сообщение об ошибке |
| --- | --- |
| Не указано имя | `instance name is required: pass it to builder() or set DEPHEALTH_NAME` |
| Не указана группа | `group is required: pass it to builder() or set DEPHEALTH_GROUP` |
| Неверный формат имени/группы | `instance name must match [a-z][a-z0-9-]*, got '...'` |
| Имя слишком длинное | `instance name must be 1-63 characters, got '...' (N chars)` |
| Не указан Critical для зависимости | ошибка валидации через fallback из env var |
| Не указан URL или host/port | `Dependency must have url, jdbcUrl, or host+port configured` |
| Неверное имя метки | `label name must match [a-zA-Z_][a-zA-Z0-9_]*, got '...'` |
| Зарезервированное имя метки | `label name '...' is reserved and cannot be used as a custom label` |
| Таймаут >= интервал | `timeout (...) must be less than interval (...)` |
| LDAP simple_bind без credentials | `LDAP simple_bind requires bindDN and bindPassword` |
| LDAP search без baseDN | `LDAP search requires baseDN` |
| LDAP startTLS + ldaps | `startTLS and ldaps:// are incompatible` |

## См. также

- [Начало работы](getting-started.ru.md) -- базовая настройка и первый пример
- [Чекеры](checkers.ru.md) -- специфичные опции чекеров подробно
- [Аутентификация](authentication.ru.md) -- опции авторизации для HTTP и gRPC
- [Пулы соединений](connection-pools.ru.md) -- интеграция с DataSource и JedisPool
- [Интеграция со Spring Boot](spring-boot.ru.md) -- автоконфигурация и YAML
- [API Reference](api-reference.ru.md) -- полный справочник по всем классам
- [Устранение неполадок](troubleshooting.ru.md) -- типичные проблемы и решения
