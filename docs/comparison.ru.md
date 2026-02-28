*[English version](comparison.md)*

# Сравнение SDK по языкам

Все четыре SDK реализуют единую [спецификацию](specification.ru.md)
и проходят идентичные conformance-тесты (17 сценариев).

## Возможности

| Возможность | Go | Python | Java | C# |
| --- | --- | --- | --- | --- |
| Версия языка | Go 1.21+ | Python 3.11+ | Java 21+ | .NET 8+ |
| Async | goroutines | asyncio | threads | async/await (Task) |
| Метрики | prometheus/client_golang | prometheus-client | Micrometer | prometheus-net |
| Конфигурация | Go options | kwargs | Builder pattern | Builder pattern |
| Connection pool | contrib (sqldb, redispool) | pool/client params | DataSource, JedisPool | EF Core DbContext |
| Conformance | 17/17 | 17/17 | 17/17 | 17/17 |

## Поддерживаемые чекеры

| Тип | Go | Python | Java | C# |
| --- | --- | --- | --- | --- |
| HTTP | `dephealth.HTTP()` | `http_check()` | `DependencyType.HTTP` | `DependencyType.Http` |
| gRPC | `dephealth.GRPC()` | `grpc_check()` | `DependencyType.GRPC` | `DependencyType.Grpc` |
| TCP | `dephealth.TCP()` | `tcp_check()` | `DependencyType.TCP` | `DependencyType.Tcp` |
| PostgreSQL | `dephealth.Postgres()` | `postgres_check()` | `DependencyType.POSTGRES` | `DependencyType.Postgres` |
| MySQL | `dephealth.MySQL()` | `mysql_check()` | `DependencyType.MYSQL` | `DependencyType.MySql` |
| Redis | `dephealth.Redis()` | `redis_check()` | `DependencyType.REDIS` | `DependencyType.Redis` |
| AMQP | `dephealth.AMQP()` | `amqp_check()` | `DependencyType.AMQP` | `DependencyType.Amqp` |
| Kafka | `dephealth.Kafka()` | `kafka_check()` | `DependencyType.KAFKA` | `DependencyType.Kafka` |
| LDAP | `dephealth.LDAP()` | `ldap_check()` | `DependencyType.LDAP` | `DependencyType.Ldap` |

## Фреймворк-интеграции

| Фреймворк | SDK | Что предоставляет |
| --- | --- | --- |
| net/http (stdlib) | Go | `promhttp.Handler()` для `/metrics` |
| FastAPI | Python | Lifespan, Middleware (`/metrics`), Router (`/health/dependencies`) |
| Spring Boot | Java | Auto-configuration, Actuator Health Indicator, `/actuator/prometheus`, `/actuator/dependencies` |
| ASP.NET Core | C# | DI registration, Middleware (`/metrics`, `/health/dependencies`) |

## Connection pool интеграция

| БД/Кэш | Go | Python | Java | C# |
| --- | --- | --- | --- | --- |
| PostgreSQL | `contrib/sqldb.FromDB()` | `pool=asyncpg.Pool` | `dataSource(DataSource)` | EF Core `DbContext` |
| MySQL | `contrib/sqldb.FromMySQLDB()` | `pool=aiomysql.Pool` | `dataSource(DataSource)` | connection string |
| Redis | `contrib/redispool.FromClient()` | `client=redis.Redis` | `jedisPool(JedisPool)` | — |

## Установка

| Язык | Команда |
| --- | --- |
| Go | `go get github.com/BigKAA/topologymetrics/sdk-go@latest` |
| Python | `pip install dephealth[fastapi]` |
| Java | Maven: `biz.kryukov.dev:dephealth-spring-boot-starter:0.5.0` |
| C# | `dotnet add package DepHealth.AspNetCore` |

## Экспортируемые метрики

Идентичны для всех SDK:

| Метрика | Тип | Описание |
| --- | --- | --- |
| `app_dependency_health` | Gauge | `1` = healthy, `0` = unhealthy |
| `app_dependency_latency_seconds` | Histogram | Латентность проверки (секунды) |
| `app_dependency_status` | Gauge (enum) | Категория статуса: 8 серий на endpoint, ровно одна = 1 |
| `app_dependency_status_detail` | Gauge (info) | Детальная причина сбоя: напр. `http_503`, `auth_error` |

Метки: `name`, `group`, `dependency`, `type`, `host`, `port`, `critical` + произвольные через `WithLabel`.
Дополнительные: `status` (на `app_dependency_status`), `detail` (на `app_dependency_status_detail`).

Бакеты histogram: `0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0, 5.0`

HELP-строки:

- `Health status of a dependency (1 = healthy, 0 = unhealthy)`
- `Latency of dependency health check in seconds`
- `Category of the last check result`
- `Detailed reason of the last check result`

## Параметры по умолчанию

Одинаковы для всех SDK:

| Параметр | Значение |
| --- | --- |
| `checkInterval` | 15s |
| `timeout` | 5s |
| `failureThreshold` | 1 |
| `successThreshold` | 1 |

## Зависимости (runtime)

### Go

- `github.com/prometheus/client_golang` — метрики
- Все чекеры встроены (без внешних зависимостей для HTTP, TCP, Postgres, Redis)

### Python

- `prometheus-client` — метрики
- `aiohttp` — HTTP checker
- Опционально: `asyncpg`, `aiomysql`, `redis`, `aio-pika`, `aiokafka`, `grpcio`

### Java

- `micrometer-core` + `micrometer-registry-prometheus` — метрики
- `slf4j-api` — логирование
- Опционально: `postgresql`, `mysql-connector-j`, `jedis`, `grpc-netty-shaded`, `amqp-client`, `kafka-clients`

### C\#

- `prometheus-net` — метрики
- Чекеры: `Npgsql`, `MySqlConnector`, `StackExchange.Redis`, `RabbitMQ.Client`, `Confluent.Kafka`, `Grpc.Net.Client`

## Быстрые ссылки

| | Go | Python | Java | C# |
| --- | --- | --- | --- | --- |
| Начало работы | [Go](../sdk-go/docs/getting-started.ru.md) | [Python](../sdk-python/docs/getting-started.ru.md) | [Java](../sdk-java/docs/getting-started.ru.md) | [C#](../sdk-csharp/docs/getting-started.ru.md) |
| Миграция | [Go](../sdk-go/docs/migration.ru.md) | [Python](../sdk-python/docs/migration.ru.md) | [Java](../sdk-java/docs/migration.ru.md) | [C#](../sdk-csharp/docs/migration.ru.md) |
