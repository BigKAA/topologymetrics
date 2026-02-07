# Руководство по интеграции dephealth в существующий Python-сервис

Пошаговая инструкция по добавлению мониторинга зависимостей
в работающий микросервис.

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
    lifespan=dephealth_lifespan(
        postgres_check("postgres-main",
            url=os.environ["DATABASE_URL"],
        ),
        redis_check("redis-cache",
            url=os.environ["REDIS_URL"],
        ),
        http_check("payment-api",
            url=os.environ["PAYMENT_SERVICE_URL"],
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

dh = DependencyHealth(
    check_interval=timedelta(seconds=15),

    # PostgreSQL через существующий asyncpg pool
    postgres_check("postgres-main", pool=pg_pool),

    # Redis через существующий redis-py client
    redis_check("redis-cache", client=redis_client),

    # Для HTTP/gRPC — только standalone
    http_check("payment-api",
        url=os.environ["PAYMENT_SERVICE_URL"],
    ),

    grpc_check("auth-service",
        host=os.environ["AUTH_HOST"],
        port=os.environ["AUTH_PORT"],
    ),
)
```

### Вариант C: Standalone-режим (простой)

SDK создаёт временные соединения для проверок:

```python
dh = DependencyHealth(
    postgres_check("postgres-main",
        url=os.environ["DATABASE_URL"],
    ),
    redis_check("redis-cache",
        url=os.environ["REDIS_URL"],
    ),
    http_check("payment-api",
        url=os.environ["PAYMENT_SERVICE_URL"],
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
    dh = DependencyHealth(...)

    await dh.start()

    # ... приложение работает ...

    await dh.stop()
```

### Ручное управление (threading, fallback)

```python
dh = DependencyHealth(...)

dh.start_sync()

# ... приложение работает ...

dh.stop_sync()
```

## Шаг 5. Экспорт метрик

### FastAPI

```python
app = FastAPI(lifespan=dephealth_lifespan(...))

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
    lifespan=dephealth_lifespan(
        postgres_check("postgres", pool=pg_pool),
        redis_check("redis", client=redis_client),
    )
)
```

### API Gateway с upstream-сервисами

```python
app = FastAPI(
    lifespan=dephealth_lifespan(
        http_check("user-service",
            url="http://user-svc:8080",
            health_path="/healthz",
        ),
        http_check("order-service",
            url="http://order-svc:8080",
        ),
        grpc_check("auth-service",
            host="auth-svc",
            port="9090",
        ),
        check_interval=timedelta(seconds=10),
    )
)
```

### Обработчик событий с Kafka и RabbitMQ

```python
app = FastAPI(
    lifespan=dephealth_lifespan(
        kafka_check("kafka-main",
            url="kafka://kafka-1:9092,kafka-2:9092",
        ),
        amqp_check("rabbitmq",
            url="amqp://user:pass@rabbitmq.svc:5672/",
        ),
        postgres_check("postgres",
            url=os.environ["DATABASE_URL"],
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
