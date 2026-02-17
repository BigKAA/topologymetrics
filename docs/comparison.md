*[Русская версия](comparison.ru.md)*

# SDK Comparison by Language

All four SDKs implement a unified [specification](specification.md)
and pass identical conformance tests (8 scenarios).

## Features

| Feature | Go | Python | Java | C# |
| --- | --- | --- | --- | --- |
| Language version | Go 1.21+ | Python 3.11+ | Java 21+ | .NET 8+ |
| Async | goroutines | asyncio | threads | async/await (Task) |
| Metrics | prometheus/client_golang | prometheus-client | Micrometer | prometheus-net |
| Configuration | Go options | kwargs | Builder pattern | Builder pattern |
| Connection pool | contrib (sqldb, redispool) | pool/client params | DataSource, JedisPool | EF Core DbContext |
| Conformance | 8/8 | 8/8 | 8/8 | 8/8 |

## Supported Checkers

| Type | Go | Python | Java | C# |
| --- | --- | --- | --- | --- |
| HTTP | `dephealth.HTTP()` | `http_check()` | `DependencyType.HTTP` | `DependencyType.Http` |
| gRPC | `dephealth.GRPC()` | `grpc_check()` | `DependencyType.GRPC` | `DependencyType.Grpc` |
| TCP | `dephealth.TCP()` | `tcp_check()` | `DependencyType.TCP` | `DependencyType.Tcp` |
| PostgreSQL | `dephealth.Postgres()` | `postgres_check()` | `DependencyType.POSTGRES` | `DependencyType.Postgres` |
| MySQL | `dephealth.MySQL()` | `mysql_check()` | `DependencyType.MYSQL` | `DependencyType.MySql` |
| Redis | `dephealth.Redis()` | `redis_check()` | `DependencyType.REDIS` | `DependencyType.Redis` |
| AMQP | `dephealth.AMQP()` | `amqp_check()` | `DependencyType.AMQP` | `DependencyType.Amqp` |
| Kafka | `dephealth.Kafka()` | `kafka_check()` | `DependencyType.KAFKA` | `DependencyType.Kafka` |

## Framework Integrations

| Framework | SDK | Provides |
| --- | --- | --- |
| net/http (stdlib) | Go | `promhttp.Handler()` for `/metrics` |
| FastAPI | Python | Lifespan, Middleware (`/metrics`), Router (`/health/dependencies`) |
| Spring Boot | Java | Auto-configuration, Actuator Health Indicator, `/actuator/prometheus`, `/actuator/dependencies` |
| ASP.NET Core | C# | DI registration, Middleware (`/metrics`, `/health/dependencies`) |

## Connection Pool Integration

| DB/Cache | Go | Python | Java | C# |
| --- | --- | --- | --- | --- |
| PostgreSQL | `contrib/sqldb.FromDB()` | `pool=asyncpg.Pool` | `dataSource(DataSource)` | EF Core `DbContext` |
| MySQL | `contrib/sqldb.FromMySQLDB()` | `pool=aiomysql.Pool` | `dataSource(DataSource)` | connection string |
| Redis | `contrib/redispool.FromClient()` | `client=redis.Redis` | `jedisPool(JedisPool)` | — |

## Installation

| Language | Command |
| --- | --- |
| Go | `go get github.com/BigKAA/topologymetrics/sdk-go@latest` |
| Python | `pip install dephealth[fastapi]` |
| Java | Maven: `biz.kryukov.dev:dephealth-spring-boot-starter:0.4.2` |
| C# | `dotnet add package DepHealth.AspNetCore` |

## Exported Metrics

Identical for all SDKs:

| Metric | Type | Description |
| --- | --- | --- |
| `app_dependency_health` | Gauge | `1` = healthy, `0` = unhealthy |
| `app_dependency_latency_seconds` | Histogram | Check latency (seconds) |
| `app_dependency_status` | Gauge (enum) | Status category: 8 series per endpoint, exactly one = 1 |
| `app_dependency_status_detail` | Gauge (info) | Detailed failure reason: e.g. `http_503`, `auth_error` |

Labels: `name`, `dependency`, `type`, `host`, `port`, `critical` + custom via `WithLabel`.
Additional: `status` (on `app_dependency_status`), `detail` (on `app_dependency_status_detail`).

Histogram buckets: `0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0, 5.0`

HELP strings:

- `Health status of a dependency (1 = healthy, 0 = unhealthy)`
- `Latency of dependency health check in seconds`
- `Status category of a dependency health check`
- `Detailed status information for a dependency health check`

## Default Parameters

Identical for all SDKs:

| Parameter | Value |
| --- | --- |
| `checkInterval` | 15s |
| `timeout` | 5s |
| `failureThreshold` | 1 |
| `successThreshold` | 1 |

## Dependencies (runtime)

### Go

- `github.com/prometheus/client_golang` — metrics
- All checkers are built-in (no external dependencies for HTTP, TCP, Postgres, Redis)

### Python

- `prometheus-client` — metrics
- `aiohttp` — HTTP checker
- Optional: `asyncpg`, `aiomysql`, `redis`, `aio-pika`, `aiokafka`, `grpcio`

### Java

- `micrometer-core` + `micrometer-registry-prometheus` — metrics
- `slf4j-api` — logging
- Optional: `postgresql`, `mysql-connector-j`, `jedis`, `grpc-netty-shaded`, `amqp-client`, `kafka-clients`

### C\#

- `prometheus-net` — metrics
- Checkers: `Npgsql`, `MySqlConnector`, `StackExchange.Redis`, `RabbitMQ.Client`, `Confluent.Kafka`, `Grpc.Net.Client`

## Quick Links

| | Go | Python | Java | C# |
| --- | --- | --- | --- | --- |
| Quickstart | [go.md](quickstart/go.md) | [python.md](quickstart/python.md) | [java.md](quickstart/java.md) | [csharp.md](quickstart/csharp.md) |
| Migration | [go.md](migration/go.md) | [python.md](migration/python.md) | [java.md](migration/java.md) | [csharp.md](migration/csharp.md) |
