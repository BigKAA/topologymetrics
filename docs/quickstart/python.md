[English](#english) | [Русский](#russian)

---

<a id="english"></a>

# Quick Start: Python SDK

Guide to integrating dephealth into a Python service in a few minutes.

## Installation

```bash
pip install dephealth
```

With FastAPI support:

```bash
pip install dephealth[fastapi]
```

With specific checkers:

```bash
pip install dephealth[postgres,redis,kafka]
```

All dependencies:

```bash
pip install dephealth[all]
```

## Minimal Example

Connecting a single HTTP dependency with metrics export:

```python
import asyncio
from dephealth.api import DependencyHealth, http_check
from dephealth_fastapi import dephealth_lifespan, DepHealthMiddleware
from fastapi import FastAPI

app = FastAPI(
    lifespan=dephealth_lifespan("my-service",
        http_check("payment-api", url="http://payment.svc:8080", critical=True),
    )
)

app.add_middleware(DepHealthMiddleware)
```

After startup, metrics will appear at `/metrics`:

```text
app_dependency_health{name="my-service",dependency="payment-api",type="http",host="payment.svc",port="8080",critical="yes"} 1
app_dependency_latency_seconds_bucket{name="my-service",dependency="payment-api",type="http",host="payment.svc",port="8080",critical="yes",le="0.01"} 42
```

## Multiple Dependencies

```python
from datetime import timedelta
from dephealth.api import (
    DependencyHealth,
    http_check,
    grpc_check,
    postgres_check,
    redis_check,
    amqp_check,
    kafka_check,
)

dh = DependencyHealth("my-service",
    # Global settings
    check_interval=timedelta(seconds=30),
    timeout=timedelta(seconds=3),

    # PostgreSQL — standalone check (new connection)
    postgres_check("postgres-main",
        url="postgresql://user:pass@pg.svc:5432/mydb",
        critical=True,
    ),

    # Redis — standalone check
    redis_check("redis-cache",
        url="redis://redis.svc:6379/0",
        critical=False,
    ),

    # HTTP service
    http_check("auth-service",
        url="http://auth.svc:8080",
        health_path="/healthz",
        critical=True,
    ),

    # gRPC service
    grpc_check("user-service",
        host="user.svc",
        port="9090",
        critical=False,
    ),

    # RabbitMQ
    amqp_check("rabbitmq",
        url="amqp://user:pass@rabbitmq.svc:5672/",
        critical=False,
    ),

    # Kafka
    kafka_check("kafka",
        host="kafka.svc",
        port="9092",
        critical=False,
    ),
)
```

## Custom Labels

Add custom labels via the `labels` parameter:

```python
postgres_check("postgres-main",
    url="postgresql://user:pass@pg.svc:5432/mydb",
    critical=True,
    labels={"role": "primary", "shard": "eu-west"},
)
```

Result in metrics:

```text
app_dependency_health{name="my-service",dependency="postgres-main",type="postgres",host="pg.svc",port="5432",critical="yes",role="primary",shard="eu-west"} 1
```

## Connection Pool Integration

Preferred mode: SDK uses the service's existing connection pool
instead of creating new connections. This reflects the real
ability of the service to work with the dependency.

### PostgreSQL via asyncpg pool

```python
import asyncpg
from dephealth.api import DependencyHealth, postgres_check

pool = await asyncpg.create_pool("postgresql://user:pass@pg.svc:5432/mydb")

dh = DependencyHealth("my-service",
    postgres_check("postgres-main", pool=pool, critical=True),
)
```

### Redis via redis-py async client

```python
from redis.asyncio import Redis
from dephealth.api import DependencyHealth, redis_check

client = Redis.from_url("redis://redis.svc:6379/0")

dh = DependencyHealth("my-service",
    redis_check("redis-cache", client=client, critical=False),
)
```

### MySQL via aiomysql pool

```python
import aiomysql
from dephealth.api import DependencyHealth, mysql_check

pool = await aiomysql.create_pool(
    host="mysql.svc", port=3306,
    user="root", password="secret", db="mydb",
)

dh = DependencyHealth("my-service",
    mysql_check("mysql-main", pool=pool, critical=True),
)
```

## FastAPI Integration

Complete example with lifespan, middleware, and health endpoint:

```python
from datetime import timedelta
from fastapi import FastAPI
from dephealth.api import (
    http_check, postgres_check, redis_check, grpc_check,
)
from dephealth_fastapi import (
    dephealth_lifespan,
    DepHealthMiddleware,
    dependencies_router,
)

app = FastAPI(
    lifespan=dephealth_lifespan("my-service",
        postgres_check("database",
            url="postgresql://user:pass@db:5432/mydb",
            critical=True,
        ),
        redis_check("cache",
            url="redis://redis:6379/0",
            critical=False,
        ),
        http_check("payment-svc",
            url="http://payment:8080",
            health_path="/health",
            critical=True,
        ),
        grpc_check("user-svc",
            host="users",
            port="50051",
            critical=False,
        ),
        check_interval=timedelta(seconds=15),
        timeout=timedelta(seconds=5),
    )
)

# Export Prometheus metrics at /metrics
app.add_middleware(DepHealthMiddleware)

# Endpoint /health/dependencies
app.include_router(dependencies_router)
```

FastAPI integration components:

| Component | Purpose |
| --- | --- |
| `dephealth_lifespan()` | Lifespan factory: start/stop on application startup/shutdown |
| `DepHealthMiddleware` | ASGI middleware for `/metrics` (Prometheus text format) |
| `dependencies_router` | APIRouter with `/health/dependencies` endpoint |

### Endpoint `/health/dependencies`

```json
{
    "status": "healthy",
    "dependencies": {
        "database": true,
        "cache": true,
        "payment-svc": false
    }
}
```

Status code: `200` (all healthy) or `503` (has unhealthy).

## Usage Without a Framework

```python
import asyncio
from datetime import timedelta
from dephealth.api import DependencyHealth, http_check, redis_check

async def main():
    dh = DependencyHealth("my-service",
        http_check("api", url="http://api:8080", critical=True),
        redis_check("cache", url="redis://redis:6379", critical=False),
        check_interval=timedelta(seconds=15),
    )

    await dh.start()

    # ... application is running ...
    status = dh.health()
    # {"api": True, "cache": False}

    await dh.stop()

asyncio.run(main())
```

## Global Options

```python
import logging
from datetime import timedelta
from prometheus_client import CollectorRegistry

dh = DependencyHealth("my-service",
    # Check interval (default 15s)
    check_interval=timedelta(seconds=30),

    # Timeout for each check (default 5s)
    timeout=timedelta(seconds=3),

    # Custom Prometheus Registry
    registry=CollectorRegistry(),

    # Custom logger
    log=logging.getLogger("my-app.dephealth"),

    # ...dependencies
)
```

## Dependency Options

Each dependency can override global settings:

```python
http_check("slow-service",
    url="http://slow.svc:8080",
    health_path="/ready",             # health check path
    tls=True,                         # HTTPS
    tls_skip_verify=True,             # skip certificate verification
    interval=timedelta(seconds=60),   # custom interval
    timeout=timedelta(seconds=10),    # custom timeout
    critical=True,                    # critical dependency
    labels={"env": "staging"},        # custom labels
)
```

## Configuration via Environment Variables

| Variable | Description | Example |
| --- | --- | --- |
| `DEPHEALTH_NAME` | Application name (overridden by API argument) | `my-service` |
| `DEPHEALTH_<DEP>_CRITICAL` | Dependency criticality | `yes` / `no` |
| `DEPHEALTH_<DEP>_LABEL_<KEY>` | Custom label | `primary` |

`<DEP>` — dependency name in uppercase, hyphens replaced with `_`.

Examples:

```bash
export DEPHEALTH_NAME=my-service
export DEPHEALTH_POSTGRES_MAIN_CRITICAL=yes
export DEPHEALTH_POSTGRES_MAIN_LABEL_ROLE=primary
```

Priority: API values > environment variables.

## Behavior When Required Parameters Are Missing

| Situation | Behavior |
| --- | --- |
| No `name` specified and no `DEPHEALTH_NAME` | Error on creation: `missing name` |
| No `critical` specified for dependency | Error on creation: `missing critical` |
| Invalid label name | Error on creation: `invalid label name` |
| Label conflicts with required label | Error on creation: `reserved label` |

## Checking Dependency Status

The `health()` method returns the current state of all dependencies:

```python
health = dh.health()
# {"postgres-main": True, "redis-cache": True, "auth-service": False}

# Usage for readiness probe
all_healthy = all(health.values())
```

## Metrics Export

dephealth exports two Prometheus metrics:

| Metric | Type | Description |
| --- | --- | --- |
| `app_dependency_health` | Gauge | `1` = available, `0` = unavailable |
| `app_dependency_latency_seconds` | Histogram | Check latency (seconds) |

Labels: `name`, `dependency`, `type`, `host`, `port`, `critical`.

## Supported Dependency Types

| Function | Type | Check Method |
| --- | --- | --- |
| `http_check()` | `http` | HTTP GET to health endpoint, expecting 2xx |
| `grpc_check()` | `grpc` | gRPC Health Check Protocol |
| `tcp_check()` | `tcp` | TCP connection establishment |
| `postgres_check()` | `postgres` | `SELECT 1` via asyncpg |
| `mysql_check()` | `mysql` | `SELECT 1` via aiomysql |
| `redis_check()` | `redis` | `PING` command |
| `amqp_check()` | `amqp` | Broker connection check |
| `kafka_check()` | `kafka` | Metadata request to broker |

## Extras (Optional Dependencies)

| Extra | Packages |
| --- | --- |
| `postgres` | asyncpg |
| `mysql` | aiomysql |
| `redis` | redis[hiredis] |
| `amqp` | aio-pika |
| `kafka` | aiokafka |
| `grpc` | grpcio, grpcio-health-checking |
| `fastapi` | fastapi, uvicorn |
| `all` | all of the above |

## Default Parameters

| Parameter | Value | Description |
| --- | --- | --- |
| `checkInterval` | 15s | Interval between checks |
| `timeout` | 5s | Timeout for a single check |
| `failureThreshold` | 1 | Number of failures before transitioning to unhealthy |
| `successThreshold` | 1 | Number of successes before transitioning to healthy |

## Next Steps

- [Integration Guide](../migration/python.md) — step-by-step integration
  into an existing service
- [Specification Overview](../specification.md) — details of metrics and behavior contracts

---

<a id="russian"></a>

# Быстрый старт: Python SDK

Руководство по подключению dephealth к Python-сервису за несколько минут.

## Установка

```bash
pip install dephealth
```

С поддержкой FastAPI:

```bash
pip install dephealth[fastapi]
```

С конкретными чекерами:

```bash
pip install dephealth[postgres,redis,kafka]
```

Все зависимости:

```bash
pip install dephealth[all]
```

## Минимальный пример

Подключение одной HTTP-зависимости с экспортом метрик:

```python
import asyncio
from dephealth.api import DependencyHealth, http_check
from dephealth_fastapi import dephealth_lifespan, DepHealthMiddleware
from fastapi import FastAPI

app = FastAPI(
    lifespan=dephealth_lifespan("my-service",
        http_check("payment-api", url="http://payment.svc:8080", critical=True),
    )
)

app.add_middleware(DepHealthMiddleware)
```

После запуска на `/metrics` появятся метрики:

```text
app_dependency_health{name="my-service",dependency="payment-api",type="http",host="payment.svc",port="8080",critical="yes"} 1
app_dependency_latency_seconds_bucket{name="my-service",dependency="payment-api",type="http",host="payment.svc",port="8080",critical="yes",le="0.01"} 42
```

## Несколько зависимостей

```python
from datetime import timedelta
from dephealth.api import (
    DependencyHealth,
    http_check,
    grpc_check,
    postgres_check,
    redis_check,
    amqp_check,
    kafka_check,
)

dh = DependencyHealth("my-service",
    # Глобальные настройки
    check_interval=timedelta(seconds=30),
    timeout=timedelta(seconds=3),

    # PostgreSQL — standalone check (новое соединение)
    postgres_check("postgres-main",
        url="postgresql://user:pass@pg.svc:5432/mydb",
        critical=True,
    ),

    # Redis — standalone check
    redis_check("redis-cache",
        url="redis://redis.svc:6379/0",
        critical=False,
    ),

    # HTTP-сервис
    http_check("auth-service",
        url="http://auth.svc:8080",
        health_path="/healthz",
        critical=True,
    ),

    # gRPC-сервис
    grpc_check("user-service",
        host="user.svc",
        port="9090",
        critical=False,
    ),

    # RabbitMQ
    amqp_check("rabbitmq",
        url="amqp://user:pass@rabbitmq.svc:5672/",
        critical=False,
    ),

    # Kafka
    kafka_check("kafka",
        host="kafka.svc",
        port="9092",
        critical=False,
    ),
)
```

## Произвольные метки

Добавляйте произвольные метки через параметр `labels`:

```python
postgres_check("postgres-main",
    url="postgresql://user:pass@pg.svc:5432/mydb",
    critical=True,
    labels={"role": "primary", "shard": "eu-west"},
)
```

Результат в метриках:

```text
app_dependency_health{name="my-service",dependency="postgres-main",type="postgres",host="pg.svc",port="5432",critical="yes",role="primary",shard="eu-west"} 1
```

## Интеграция с connection pool

Предпочтительный режим: SDK использует существующий connection pool
сервиса вместо создания новых соединений. Это отражает реальную
способность сервиса работать с зависимостью.

### PostgreSQL через asyncpg pool

```python
import asyncpg
from dephealth.api import DependencyHealth, postgres_check

pool = await asyncpg.create_pool("postgresql://user:pass@pg.svc:5432/mydb")

dh = DependencyHealth("my-service",
    postgres_check("postgres-main", pool=pool, critical=True),
)
```

### Redis через redis-py async client

```python
from redis.asyncio import Redis
from dephealth.api import DependencyHealth, redis_check

client = Redis.from_url("redis://redis.svc:6379/0")

dh = DependencyHealth("my-service",
    redis_check("redis-cache", client=client, critical=False),
)
```

### MySQL через aiomysql pool

```python
import aiomysql
from dephealth.api import DependencyHealth, mysql_check

pool = await aiomysql.create_pool(
    host="mysql.svc", port=3306,
    user="root", password="secret", db="mydb",
)

dh = DependencyHealth("my-service",
    mysql_check("mysql-main", pool=pool, critical=True),
)
```

## FastAPI интеграция

Полный пример с lifespan, middleware и health endpoint:

```python
from datetime import timedelta
from fastapi import FastAPI
from dephealth.api import (
    http_check, postgres_check, redis_check, grpc_check,
)
from dephealth_fastapi import (
    dephealth_lifespan,
    DepHealthMiddleware,
    dependencies_router,
)

app = FastAPI(
    lifespan=dephealth_lifespan("my-service",
        postgres_check("database",
            url="postgresql://user:pass@db:5432/mydb",
            critical=True,
        ),
        redis_check("cache",
            url="redis://redis:6379/0",
            critical=False,
        ),
        http_check("payment-svc",
            url="http://payment:8080",
            health_path="/health",
            critical=True,
        ),
        grpc_check("user-svc",
            host="users",
            port="50051",
            critical=False,
        ),
        check_interval=timedelta(seconds=15),
        timeout=timedelta(seconds=5),
    )
)

# Экспорт Prometheus-метрик на /metrics
app.add_middleware(DepHealthMiddleware)

# Endpoint /health/dependencies
app.include_router(dependencies_router)
```

Компоненты FastAPI-интеграции:

| Компонент | Назначение |
| --- | --- |
| `dephealth_lifespan()` | Lifespan-фабрика: start/stop при запуске/остановке приложения |
| `DepHealthMiddleware` | ASGI middleware для `/metrics` (Prometheus text format) |
| `dependencies_router` | APIRouter с endpoint `/health/dependencies` |

### Endpoint `/health/dependencies`

```json
{
    "status": "healthy",
    "dependencies": {
        "database": true,
        "cache": true,
        "payment-svc": false
    }
}
```

Статус-код: `200` (все healthy) или `503` (есть unhealthy).

## Использование без фреймворка

```python
import asyncio
from datetime import timedelta
from dephealth.api import DependencyHealth, http_check, redis_check

async def main():
    dh = DependencyHealth("my-service",
        http_check("api", url="http://api:8080", critical=True),
        redis_check("cache", url="redis://redis:6379", critical=False),
        check_interval=timedelta(seconds=15),
    )

    await dh.start()

    # ... приложение работает ...
    status = dh.health()
    # {"api": True, "cache": False}

    await dh.stop()

asyncio.run(main())
```

## Глобальные опции

```python
import logging
from datetime import timedelta
from prometheus_client import CollectorRegistry

dh = DependencyHealth("my-service",
    # Интервал проверки (по умолчанию 15s)
    check_interval=timedelta(seconds=30),

    # Таймаут каждой проверки (по умолчанию 5s)
    timeout=timedelta(seconds=3),

    # Кастомный Prometheus Registry
    registry=CollectorRegistry(),

    # Кастомный логгер
    log=logging.getLogger("my-app.dephealth"),

    # ...зависимости
)
```

## Опции зависимостей

Каждая зависимость может переопределить глобальные настройки:

```python
http_check("slow-service",
    url="http://slow.svc:8080",
    health_path="/ready",             # путь health check
    tls=True,                         # HTTPS
    tls_skip_verify=True,             # пропустить проверку сертификата
    interval=timedelta(seconds=60),   # свой интервал
    timeout=timedelta(seconds=10),    # свой таймаут
    critical=True,                    # критическая зависимость
    labels={"env": "staging"},        # произвольные метки
)
```

## Конфигурация через переменные окружения

| Переменная | Описание | Пример |
| --- | --- | --- |
| `DEPHEALTH_NAME` | Имя приложения (перекрывается аргументом API) | `my-service` |
| `DEPHEALTH_<DEP>_CRITICAL` | Критичность зависимости | `yes` / `no` |
| `DEPHEALTH_<DEP>_LABEL_<KEY>` | Произвольная метка | `primary` |

`<DEP>` — имя зависимости в верхнем регистре, дефисы заменены на `_`.

Примеры:

```bash
export DEPHEALTH_NAME=my-service
export DEPHEALTH_POSTGRES_MAIN_CRITICAL=yes
export DEPHEALTH_POSTGRES_MAIN_LABEL_ROLE=primary
```

Приоритет: значения из API > переменные окружения.

## Поведение при отсутствии обязательных параметров

| Ситуация | Поведение |
| --- | --- |
| Не указан `name` и нет `DEPHEALTH_NAME` | Ошибка при создании: `missing name` |
| Не указан `critical` для зависимости | Ошибка при создании: `missing critical` |
| Недопустимое имя метки | Ошибка при создании: `invalid label name` |
| Метка совпадает с обязательной | Ошибка при создании: `reserved label` |

## Проверка состояния зависимостей

Метод `health()` возвращает текущее состояние всех зависимостей:

```python
health = dh.health()
# {"postgres-main": True, "redis-cache": True, "auth-service": False}

# Использование для readiness probe
all_healthy = all(health.values())
```

## Экспорт метрик

dephealth экспортирует две метрики Prometheus:

| Метрика | Тип | Описание |
| --- | --- | --- |
| `app_dependency_health` | Gauge | `1` = доступен, `0` = недоступен |
| `app_dependency_latency_seconds` | Histogram | Латентность проверки (секунды) |

Метки: `name`, `dependency`, `type`, `host`, `port`, `critical`.

## Поддерживаемые типы зависимостей

| Функция | Тип | Метод проверки |
| --- | --- | --- |
| `http_check()` | `http` | HTTP GET к health endpoint, ожидание 2xx |
| `grpc_check()` | `grpc` | gRPC Health Check Protocol |
| `tcp_check()` | `tcp` | Установка TCP-соединения |
| `postgres_check()` | `postgres` | `SELECT 1` через asyncpg |
| `mysql_check()` | `mysql` | `SELECT 1` через aiomysql |
| `redis_check()` | `redis` | Команда `PING` |
| `amqp_check()` | `amqp` | Проверка соединения с брокером |
| `kafka_check()` | `kafka` | Metadata request к брокеру |

## Extras (опциональные зависимости)

| Extra | Пакеты |
| --- | --- |
| `postgres` | asyncpg |
| `mysql` | aiomysql |
| `redis` | redis[hiredis] |
| `amqp` | aio-pika |
| `kafka` | aiokafka |
| `grpc` | grpcio, grpcio-health-checking |
| `fastapi` | fastapi, uvicorn |
| `all` | все вышеперечисленные |

## Параметры по умолчанию

| Параметр | Значение | Описание |
| --- | --- | --- |
| `checkInterval` | 15s | Интервал между проверками |
| `timeout` | 5s | Таймаут одной проверки |
| `failureThreshold` | 1 | Число неудач до перехода в unhealthy |
| `successThreshold` | 1 | Число успехов до перехода в healthy |

## Следующие шаги

- [Руководство по интеграции](../migration/python.md) — пошаговое подключение
  к существующему сервису
- [Обзор спецификации](../specification.md) — детали контрактов метрик и поведения
