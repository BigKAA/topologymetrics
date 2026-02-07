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
    lifespan=dephealth_lifespan(
        http_check("payment-api", url="http://payment.svc:8080"),
    )
)

app.add_middleware(DepHealthMiddleware)
```

После запуска на `/metrics` появятся метрики:

```text
app_dependency_health{dependency="payment-api",type="http",host="payment.svc",port="8080"} 1
app_dependency_latency_seconds_bucket{dependency="payment-api",type="http",host="payment.svc",port="8080",le="0.01"} 42
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

dh = DependencyHealth(
    # Глобальные настройки
    check_interval=timedelta(seconds=30),
    timeout=timedelta(seconds=3),

    # PostgreSQL — standalone check (новое соединение)
    postgres_check("postgres-main",
        url="postgresql://user:pass@pg.svc:5432/mydb",
    ),

    # Redis — standalone check
    redis_check("redis-cache",
        url="redis://redis.svc:6379/0",
    ),

    # HTTP-сервис
    http_check("auth-service",
        url="http://auth.svc:8080",
        health_path="/healthz",
    ),

    # gRPC-сервис
    grpc_check("user-service",
        host="user.svc",
        port="9090",
    ),

    # RabbitMQ
    amqp_check("rabbitmq",
        url="amqp://user:pass@rabbitmq.svc:5672/",
    ),

    # Kafka
    kafka_check("kafka",
        host="kafka.svc",
        port="9092",
    ),
)
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

dh = DependencyHealth(
    postgres_check("postgres-main", pool=pool),
)
```

### Redis через redis-py async client

```python
from redis.asyncio import Redis
from dephealth.api import DependencyHealth, redis_check

client = Redis.from_url("redis://redis.svc:6379/0")

dh = DependencyHealth(
    redis_check("redis-cache", client=client),
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

dh = DependencyHealth(
    mysql_check("mysql-main", pool=pool),
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
    lifespan=dephealth_lifespan(
        postgres_check("database",
            url="postgresql://user:pass@db:5432/mydb",
        ),
        redis_check("cache",
            url="redis://redis:6379/0",
        ),
        http_check("payment-svc",
            url="http://payment:8080",
            health_path="/health",
        ),
        grpc_check("user-svc",
            host="users",
            port="50051",
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
    dh = DependencyHealth(
        http_check("api", url="http://api:8080"),
        redis_check("cache", url="redis://redis:6379"),
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

dh = DependencyHealth(
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
)
```

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

Метки: `dependency`, `type`, `host`, `port`.

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
