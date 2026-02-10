[English](#english) | [Русский](#russian)

---

<a id="english"></a>

# Guide to Integrating dephealth into an Existing Python Service

Step-by-step instructions for adding dependency monitoring
to a running microservice.

## Migration from v0.1 to v0.2

### API Changes

| v0.1 | v0.2 | Description |
| --- | --- | --- |
| `DependencyHealth(...)` | `DependencyHealth("my-service", ...)` | Required first argument `name` |
| `dephealth_lifespan(...)` | `dephealth_lifespan("my-service", ...)` | Required first argument `name` |
| `critical=True` (optional) | `critical=True/False` (required) | For each factory |
| none | `labels={"key": "value"}` | Arbitrary labels |

### Required Changes

1. Add `name` as the first argument:

```python
# v0.1
dh = DependencyHealth(
    postgres_check("postgres-main", url="postgresql://..."),
)

# v0.2
dh = DependencyHealth("my-service",
    postgres_check("postgres-main", url="postgresql://...", critical=True),
)
```

1. Specify `critical` for each dependency:

```python
# v0.1 — critical is optional
redis_check("redis-cache", url="redis://redis.svc:6379")

# v0.2 — critical is required
redis_check("redis-cache", url="redis://redis.svc:6379", critical=False)
```

1. Update `dephealth_lifespan` (FastAPI):

```python
# v0.1
app = FastAPI(
    lifespan=dephealth_lifespan(
        http_check("api", url="http://api:8080"),
    )
)

# v0.2
app = FastAPI(
    lifespan=dephealth_lifespan("my-service",
        http_check("api", url="http://api:8080", critical=True),
    )
)
```

### New Labels in Metrics

```text
# v0.1
app_dependency_health{dependency="postgres-main",type="postgres",host="pg.svc",port="5432"} 1

# v0.2
app_dependency_health{name="my-service",dependency="postgres-main",type="postgres",host="pg.svc",port="5432",critical="yes"} 1
```

Update PromQL queries and Grafana dashboards to include `name` and `critical` labels.

## Prerequisites

- Python 3.11+
- FastAPI, Flask, Django or any ASGI/WSGI framework
- Network access to dependencies (databases, caches, other services) from the service

## Step 1. Install Dependencies

```bash
pip install dephealth[fastapi]
```

Or with specific checkers:

```bash
pip install dephealth[postgres,redis,grpc,fastapi]
```

## Step 2. Import Packages

Add imports to your service initialization file:

```python
from dephealth.api import (
    DependencyHealth,
    http_check,
    postgres_check,
    redis_check,
)
```

For FastAPI integration:

```python
from dephealth_fastapi import (
    dephealth_lifespan,
    DepHealthMiddleware,
    dependencies_router,
)
```

## Step 3. Create DependencyHealth Instance

### Option A: FastAPI with lifespan (recommended)

The simplest approach using `dephealth_lifespan()`:

```python
from fastapi import FastAPI
from datetime import timedelta

app = FastAPI(
    lifespan=dephealth_lifespan("my-service",
        postgres_check("postgres-main",
            url=os.environ["DATABASE_URL"],
            critical=True,
        ),
        redis_check("redis-cache",
            url=os.environ["REDIS_URL"],
            critical=False,
        ),
        http_check("payment-api",
            url=os.environ["PAYMENT_SERVICE_URL"],
            critical=True,
        ),
        check_interval=timedelta(seconds=15),
    )
)
```

### Option B: Connection pool integration (recommended)

SDK uses existing service connections. Advantages:

- Reflects the actual service's ability to work with dependencies
- Does not create additional load on DB/cache
- Detects pool-related issues (exhaustion, leaks)

```python
from datetime import timedelta
import asyncpg
from redis.asyncio import Redis

# Existing service connections
pg_pool = await asyncpg.create_pool(os.environ["DATABASE_URL"])
redis_client = Redis.from_url(os.environ["REDIS_URL"])

dh = DependencyHealth("my-service",
    check_interval=timedelta(seconds=15),

    # PostgreSQL via existing asyncpg pool
    postgres_check("postgres-main", pool=pg_pool, critical=True),

    # Redis via existing redis-py client
    redis_check("redis-cache", client=redis_client, critical=False),

    # For HTTP/gRPC — standalone only
    http_check("payment-api",
        url=os.environ["PAYMENT_SERVICE_URL"],
        critical=True,
    ),

    grpc_check("auth-service",
        host=os.environ["AUTH_HOST"],
        port=os.environ["AUTH_PORT"],
        critical=True,
    ),
)
```

### Option C: Standalone mode (simple)

SDK creates temporary connections for checks:

```python
dh = DependencyHealth("my-service",
    postgres_check("postgres-main",
        url=os.environ["DATABASE_URL"],
        critical=True,
    ),
    redis_check("redis-cache",
        url=os.environ["REDIS_URL"],
        critical=False,
    ),
    http_check("payment-api",
        url=os.environ["PAYMENT_SERVICE_URL"],
        critical=True,
    ),
)
```

## Step 4. Start and Stop

### FastAPI with lifespan

When using `dephealth_lifespan()`, start/stop happen automatically.
The `DependencyHealth` instance is accessible via `app.state.dephealth`.

### Manual management (asyncio)

```python
async def main():
    dh = DependencyHealth("my-service", ...)

    await dh.start()

    # ... application runs ...

    await dh.stop()
```

### Manual management (threading, fallback)

```python
dh = DependencyHealth("my-service", ...)

dh.start_sync()

# ... application runs ...

dh.stop_sync()
```

## Step 5. Export Metrics

### FastAPI

```python
app = FastAPI(lifespan=dephealth_lifespan("my-service", ...))

# Prometheus metrics at /metrics
app.add_middleware(DepHealthMiddleware)

# Endpoint /health/dependencies
app.include_router(dependencies_router)
```

### Without FastAPI

Use the standard `prometheus_client`:

```python
from prometheus_client import generate_latest, CONTENT_TYPE_LATEST

# In HTTP handler:
def metrics_handler(request):
    return Response(
        content=generate_latest(),
        media_type=CONTENT_TYPE_LATEST,
    )
```

## Step 6. Dependency Status Endpoint (optional)

With FastAPI, `dependencies_router` already provides `/health/dependencies`.

For a custom endpoint:

```python
from fastapi import FastAPI, Response
import json

@app.get("/health/dependencies")
async def health_dependencies():
    dh = app.state.dephealth
    health = dh.health()

    all_healthy = all(health.values())
    status_code = 200 if all_healthy else 503

    return Response(
        content=json.dumps({
            "status": "healthy" if all_healthy else "unhealthy",
            "dependencies": health,
        }),
        media_type="application/json",
        status_code=status_code,
    )
```

## Typical Configurations

### Web service with PostgreSQL and Redis

```python
import asyncpg
from redis.asyncio import Redis

pg_pool = await asyncpg.create_pool(os.environ["DATABASE_URL"])
redis_client = Redis.from_url(os.environ["REDIS_URL"])

app = FastAPI(
    lifespan=dephealth_lifespan("my-service",
        postgres_check("postgres", pool=pg_pool, critical=True),
        redis_check("redis", client=redis_client, critical=False),
    )
)
```

### API Gateway with upstream services

```python
app = FastAPI(
    lifespan=dephealth_lifespan("api-gateway",
        http_check("user-service",
            url="http://user-svc:8080",
            health_path="/healthz",
            critical=True,
        ),
        http_check("order-service",
            url="http://order-svc:8080",
            critical=True,
        ),
        grpc_check("auth-service",
            host="auth-svc",
            port="9090",
            critical=True,
        ),
        check_interval=timedelta(seconds=10),
    )
)
```

### Event processor with Kafka and RabbitMQ

```python
app = FastAPI(
    lifespan=dephealth_lifespan("event-processor",
        kafka_check("kafka-main",
            url="kafka://kafka-1:9092,kafka-2:9092",
            critical=True,
        ),
        amqp_check("rabbitmq",
            url="amqp://user:pass@rabbitmq.svc:5672/",
            critical=True,
        ),
        postgres_check("postgres",
            url=os.environ["DATABASE_URL"],
            critical=False,
        ),
    )
)
```

## Troubleshooting

### Metrics don't appear at `/metrics`

**Check:**

1. `DepHealthMiddleware` is added to the application
2. Lifespan was called correctly (application started without errors)
3. Enough time has passed for the first check

### All dependencies show `0` (unhealthy)

**Check:**

1. Network accessibility of dependencies from service container/pod
2. DNS resolution of service names
3. Correctness of URL/host/port in configuration
4. Timeout (`5s` by default) — is it sufficient for the dependency
5. Logs: `logging.basicConfig(level=logging.INFO)` will show error reasons

### High latency for PostgreSQL/MySQL checks

**Cause**: standalone mode creates a new connection each time.

**Solution**: use pool integration (`postgres_check(..., pool=pool)`).
This eliminates connection establishment overhead.

### gRPC: error `context deadline exceeded`

**Check:**

1. gRPC service is accessible at the specified address
2. Service implements `grpc.health.v1.Health/Check`
3. For gRPC use `host` + `port`, not `url` —
   URL parser may incorrectly handle bare `host:port`
4. If TLS is needed: `grpc_check(..., tls=True)`

### AMQP: connection error to RabbitMQ

**Provide the full URL:**

```python
amqp_check("rabbitmq",
    url="amqp://user:pass@rabbitmq.svc:5672/vhost",
    critical=False,
)
```

### Dependency Naming

Names must follow the rules:

- Length: 1-63 characters
- Format: `[a-z][a-z0-9-]*` (lowercase letters, digits, hyphens)
- Must start with a letter
- Examples: `postgres-main`, `redis-cache`, `auth-service`

## Next Steps

- [Quick Start](../quickstart/python.md) — minimal examples
- [Specification Overview](../specification.md) — details of metric contracts and behavior

---

<a id="russian"></a>

# Руководство по интеграции dephealth в существующий Python-сервис

Пошаговая инструкция по добавлению мониторинга зависимостей
в работающий микросервис.

## Миграция с v0.1 на v0.2

### Изменения API

| v0.1 | v0.2 | Описание |
| --- | --- | --- |
| `DependencyHealth(...)` | `DependencyHealth("my-service", ...)` | Обязательный первый аргумент `name` |
| `dephealth_lifespan(...)` | `dephealth_lifespan("my-service", ...)` | Обязательный первый аргумент `name` |
| `critical=True` (необязателен) | `critical=True/False` (обязателен) | Для каждой фабрики |
| нет | `labels={"key": "value"}` | Произвольные метки |

### Обязательные изменения

1. Добавьте `name` первым аргументом:

```python
# v0.1
dh = DependencyHealth(
    postgres_check("postgres-main", url="postgresql://..."),
)

# v0.2
dh = DependencyHealth("my-service",
    postgres_check("postgres-main", url="postgresql://...", critical=True),
)
```

1. Укажите `critical` для каждой зависимости:

```python
# v0.1 — critical необязателен
redis_check("redis-cache", url="redis://redis.svc:6379")

# v0.2 — critical обязателен
redis_check("redis-cache", url="redis://redis.svc:6379", critical=False)
```

1. Обновите `dephealth_lifespan` (FastAPI):

```python
# v0.1
app = FastAPI(
    lifespan=dephealth_lifespan(
        http_check("api", url="http://api:8080"),
    )
)

# v0.2
app = FastAPI(
    lifespan=dephealth_lifespan("my-service",
        http_check("api", url="http://api:8080", critical=True),
    )
)
```

### Новые метки в метриках

```text
# v0.1
app_dependency_health{dependency="postgres-main",type="postgres",host="pg.svc",port="5432"} 1

# v0.2
app_dependency_health{name="my-service",dependency="postgres-main",type="postgres",host="pg.svc",port="5432",critical="yes"} 1
```

Обновите PromQL-запросы и дашборды Grafana, добавив метки `name` и `critical`.

## Предварительные требования

- Python 3.11+
- FastAPI, Flask, Django или любой ASGI/WSGI-фреймворк
- Доступ к зависимостям (БД, кэш, другие сервисы) из сервиса

## Шаг 1. Установка зависимостей

```bash
pip install dephealth[fastapi]
```

Или с конкретными чекерами:

```bash
pip install dephealth[postgres,redis,grpc,fastapi]
```

## Шаг 2. Импорт пакетов

Добавьте импорты в файл с инициализацией сервиса:

```python
from dephealth.api import (
    DependencyHealth,
    http_check,
    postgres_check,
    redis_check,
)
```

Для FastAPI-интеграции:

```python
from dephealth_fastapi import (
    dephealth_lifespan,
    DepHealthMiddleware,
    dependencies_router,
)
```

## Шаг 3. Создание экземпляра DependencyHealth

### Вариант A: FastAPI с lifespan (рекомендуется)

Самый простой способ — через `dephealth_lifespan()`:

```python
from fastapi import FastAPI
from datetime import timedelta

app = FastAPI(
    lifespan=dephealth_lifespan("my-service",
        postgres_check("postgres-main",
            url=os.environ["DATABASE_URL"],
            critical=True,
        ),
        redis_check("redis-cache",
            url=os.environ["REDIS_URL"],
            critical=False,
        ),
        http_check("payment-api",
            url=os.environ["PAYMENT_SERVICE_URL"],
            critical=True,
        ),
        check_interval=timedelta(seconds=15),
    )
)
```

### Вариант B: Интеграция с connection pool (рекомендуется)

SDK использует существующие подключения сервиса. Преимущества:

- Отражает реальную способность сервиса работать с зависимостью
- Не создаёт дополнительную нагрузку на БД/кэш
- Обнаруживает проблемы с пулом (исчерпание, утечки)

```python
from datetime import timedelta
import asyncpg
from redis.asyncio import Redis

# Существующие подключения сервиса
pg_pool = await asyncpg.create_pool(os.environ["DATABASE_URL"])
redis_client = Redis.from_url(os.environ["REDIS_URL"])

dh = DependencyHealth("my-service",
    check_interval=timedelta(seconds=15),

    # PostgreSQL через существующий asyncpg pool
    postgres_check("postgres-main", pool=pg_pool, critical=True),

    # Redis через существующий redis-py client
    redis_check("redis-cache", client=redis_client, critical=False),

    # Для HTTP/gRPC — только standalone
    http_check("payment-api",
        url=os.environ["PAYMENT_SERVICE_URL"],
        critical=True,
    ),

    grpc_check("auth-service",
        host=os.environ["AUTH_HOST"],
        port=os.environ["AUTH_PORT"],
        critical=True,
    ),
)
```

### Вариант C: Standalone-режим (простой)

SDK создаёт временные соединения для проверок:

```python
dh = DependencyHealth("my-service",
    postgres_check("postgres-main",
        url=os.environ["DATABASE_URL"],
        critical=True,
    ),
    redis_check("redis-cache",
        url=os.environ["REDIS_URL"],
        critical=False,
    ),
    http_check("payment-api",
        url=os.environ["PAYMENT_SERVICE_URL"],
        critical=True,
    ),
)
```

## Шаг 4. Запуск и остановка

### FastAPI с lifespan

При использовании `dephealth_lifespan()` start/stop происходят
автоматически. Экземпляр `DependencyHealth` доступен через `app.state.dephealth`.

### Ручное управление (asyncio)

```python
async def main():
    dh = DependencyHealth("my-service", ...)

    await dh.start()

    # ... приложение работает ...

    await dh.stop()
```

### Ручное управление (threading, fallback)

```python
dh = DependencyHealth("my-service", ...)

dh.start_sync()

# ... приложение работает ...

dh.stop_sync()
```

## Шаг 5. Экспорт метрик

### FastAPI

```python
app = FastAPI(lifespan=dephealth_lifespan("my-service", ...))

# Prometheus-метрики на /metrics
app.add_middleware(DepHealthMiddleware)

# Endpoint /health/dependencies
app.include_router(dependencies_router)
```

### Без FastAPI

Используйте стандартный `prometheus_client`:

```python
from prometheus_client import generate_latest, CONTENT_TYPE_LATEST

# В HTTP-обработчике:
def metrics_handler(request):
    return Response(
        content=generate_latest(),
        media_type=CONTENT_TYPE_LATEST,
    )
```

## Шаг 6. Endpoint для состояния зависимостей (опционально)

С FastAPI `dependencies_router` уже предоставляет `/health/dependencies`.

Для кастомного endpoint:

```python
from fastapi import FastAPI, Response
import json

@app.get("/health/dependencies")
async def health_dependencies():
    dh = app.state.dephealth
    health = dh.health()

    all_healthy = all(health.values())
    status_code = 200 if all_healthy else 503

    return Response(
        content=json.dumps({
            "status": "healthy" if all_healthy else "unhealthy",
            "dependencies": health,
        }),
        media_type="application/json",
        status_code=status_code,
    )
```

## Типичные конфигурации

### Веб-сервис с PostgreSQL и Redis

```python
import asyncpg
from redis.asyncio import Redis

pg_pool = await asyncpg.create_pool(os.environ["DATABASE_URL"])
redis_client = Redis.from_url(os.environ["REDIS_URL"])

app = FastAPI(
    lifespan=dephealth_lifespan("my-service",
        postgres_check("postgres", pool=pg_pool, critical=True),
        redis_check("redis", client=redis_client, critical=False),
    )
)
```

### API Gateway с upstream-сервисами

```python
app = FastAPI(
    lifespan=dephealth_lifespan("api-gateway",
        http_check("user-service",
            url="http://user-svc:8080",
            health_path="/healthz",
            critical=True,
        ),
        http_check("order-service",
            url="http://order-svc:8080",
            critical=True,
        ),
        grpc_check("auth-service",
            host="auth-svc",
            port="9090",
            critical=True,
        ),
        check_interval=timedelta(seconds=10),
    )
)
```

### Обработчик событий с Kafka и RabbitMQ

```python
app = FastAPI(
    lifespan=dephealth_lifespan("event-processor",
        kafka_check("kafka-main",
            url="kafka://kafka-1:9092,kafka-2:9092",
            critical=True,
        ),
        amqp_check("rabbitmq",
            url="amqp://user:pass@rabbitmq.svc:5672/",
            critical=True,
        ),
        postgres_check("postgres",
            url=os.environ["DATABASE_URL"],
            critical=False,
        ),
    )
)
```

## Troubleshooting

### Метрики не появляются на `/metrics`

**Проверьте:**

1. `DepHealthMiddleware` добавлен в приложение
2. Lifespan вызван корректно (приложение стартовало без ошибок)
3. Прошло достаточно времени для первой проверки

### Все зависимости показывают `0` (unhealthy)

**Проверьте:**

1. Сетевая доступность зависимостей из контейнера/пода сервиса
2. DNS-резолвинг имён сервисов
3. Правильность URL/host/port в конфигурации
4. Таймаут (`5s` по умолчанию) — достаточен ли для данной зависимости
5. Логи: `logging.basicConfig(level=logging.INFO)` покажет причины ошибок

### Высокая латентность проверок PostgreSQL/MySQL

**Причина**: standalone-режим создаёт новое соединение каждый раз.

**Решение**: используйте pool-интеграцию (`postgres_check(..., pool=pool)`).
Это исключает overhead на установку соединения.

### gRPC: ошибка `context deadline exceeded`

**Проверьте:**

1. gRPC-сервис доступен по указанному адресу
2. Сервис реализует `grpc.health.v1.Health/Check`
3. Для gRPC используйте `host` + `port`, а не `url` —
   URL-парсер может некорректно обработать bare `host:port`
4. Если нужен TLS: `grpc_check(..., tls=True)`

### AMQP: ошибка подключения к RabbitMQ

**Передайте полный URL:**

```python
amqp_check("rabbitmq",
    url="amqp://user:pass@rabbitmq.svc:5672/vhost",
    critical=False,
)
```

### Именование зависимостей

Имена должны соответствовать правилам:

- Длина: 1-63 символа
- Формат: `[a-z][a-z0-9-]*` (строчные буквы, цифры, дефисы)
- Начинается с буквы
- Примеры: `postgres-main`, `redis-cache`, `auth-service`

## Следующие шаги

- [Быстрый старт](../quickstart/python.md) — минимальные примеры
- [Обзор спецификации](../specification.md) — детали контрактов метрик и поведения
