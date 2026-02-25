*[English version](connection-pools.md)*

# Connection Pools

Руководство по интеграции dephealth с существующими пулами соединений
вашего приложения для более реалистичного мониторинга.

## Standalone vs Pool-режим

| | Standalone-режим | Pool-режим |
| --- | --- | --- |
| **Соединение** | Новое соединение на каждую проверку | Заимствует из существующего пула |
| **Настройка** | Просто указать URL | Передать объект pool/client |
| **Overhead** | Установка соединения каждый раз | Минимальный (переиспользует пул) |
| **Что тестирует** | Сетевая доступность + auth | Реальное здоровье пула + запрос |
| **Проблемы пула** | Не обнаруживает | Обнаруживает (исчерпание, утечки) |
| **Рекомендуется для** | Простые сетапы, dev/test | Продакшн |

## PostgreSQL (asyncpg)

```python
import asyncpg
from dephealth.api import DependencyHealth, postgres_check

# Создание пула, как обычно в вашем приложении
pg_pool = await asyncpg.create_pool(
    "postgresql://user:pass@pg.svc:5432/mydb",
    min_size=5,
    max_size=20,
)

dh = DependencyHealth("my-service", "my-team",
    postgres_check("postgres-main",
        pool=pg_pool,
        critical=True,
    ),
)
```

Когда указан `pool`, чекер заимствует соединение из пула,
выполняет `SELECT 1` (или кастомный запрос) и возвращает соединение.

## MySQL (aiomysql)

```python
import aiomysql
from dephealth.api import DependencyHealth, mysql_check

mysql_pool = await aiomysql.create_pool(
    host="mysql.svc",
    port=3306,
    user="user",
    password="pass",
    db="mydb",
    minsize=5,
    maxsize=20,
)

dh = DependencyHealth("my-service", "my-team",
    mysql_check("mysql-main",
        pool=mysql_pool,
        critical=True,
    ),
)
```

## Redis (redis-py)

```python
from redis.asyncio import Redis
from dephealth.api import DependencyHealth, redis_check

redis_client = Redis.from_url(
    "redis://redis.svc:6379",
    max_connections=20,
)

dh = DependencyHealth("my-service", "my-team",
    redis_check("redis-cache",
        client=redis_client,
        critical=False,
    ),
)
```

Когда указан `client`, чекер вызывает `PING` на существующем клиенте
вместо создания нового соединения.

## LDAP (ldap3)

```python
from ldap3 import Server, Connection
from dephealth.api import DependencyHealth, ldap_check

ldap_server = Server("ldap://ldap.svc:389")
ldap_conn = Connection(ldap_server, auto_bind=True)

dh = DependencyHealth("my-service", "my-team",
    ldap_check("ldap-server",
        client=ldap_conn,
        check_method="ROOT_DSE",
        critical=False,
    ),
)
```

## Смешанные режимы

Можно использовать разные режимы для разных зависимостей:

```python
import asyncpg
from redis.asyncio import Redis
from dephealth.api import (
    DependencyHealth,
    postgres_check,
    redis_check,
    http_check,
    kafka_check,
)

pg_pool = await asyncpg.create_pool("postgresql://user:pass@pg.svc:5432/mydb")
redis_client = Redis.from_url("redis://redis.svc:6379")

dh = DependencyHealth("my-service", "my-team",
    # Pool-режим — тестирует реальное здоровье пула
    postgres_check("postgres", pool=pg_pool, critical=True),
    redis_check("redis", client=redis_client, critical=False),

    # Standalone-режим — тестирует сетевую доступность
    http_check("payment-api",
        url="http://payment.svc:8080",
        critical=True,
    ),
    kafka_check("kafka",
        url="kafka://kafka-1.svc:9092,kafka-2.svc:9092",
        critical=True,
    ),
)
```

## Когда использовать какой режим

### Pool-режим рекомендуется когда

- Нужно обнаруживать исчерпание пула и утечки соединений
- Нужно проверить реальную способность приложения запрашивать зависимость
- Работа в продакшне
- Драйвер зависимости поддерживает async-пулы

### Standalone-режим рекомендуется когда

- Нет пула соединений (напр., внешние HTTP/gRPC-сервисы)
- Нужно проверить базовую сетевую доступность
- Разработка или тестирование
- Тип зависимости не поддерживает пулы (HTTP, gRPC, TCP, AMQP, Kafka)

## Поддерживаемые типы пулов

| Чекер | Параметр пула | Тип пула |
| --- | --- | --- |
| PostgreSQL | `pool=` | `asyncpg.Pool` |
| MySQL | `pool=` | `aiomysql.Pool` |
| Redis | `client=` | `redis.asyncio.Redis` |
| LDAP | `client=` | `ldap3.Connection` |

HTTP, gRPC, TCP, AMQP и Kafka чекеры всегда используют standalone-режим.

## FastAPI-интеграция с пулами

```python
import os
import asyncpg
from redis.asyncio import Redis
from contextlib import asynccontextmanager
from fastapi import FastAPI

from dephealth.api import DependencyHealth, postgres_check, redis_check
from dephealth_fastapi import DepHealthMiddleware, dependencies_router

@asynccontextmanager
async def lifespan(app: FastAPI):
    # Создание пулов
    pg_pool = await asyncpg.create_pool(os.environ["DATABASE_URL"])
    redis_client = Redis.from_url(os.environ["REDIS_URL"])

    # Создание и запуск dephealth
    dh = DependencyHealth("my-service", "my-team",
        postgres_check("postgres", pool=pg_pool, critical=True),
        redis_check("redis", client=redis_client, critical=False),
    )
    await dh.start()
    app.state.dephealth = dh
    app.state.pg_pool = pg_pool
    app.state.redis = redis_client

    yield

    # Очистка
    await dh.stop()
    redis_client.close()
    await pg_pool.close()

app = FastAPI(lifespan=lifespan)
app.add_middleware(DepHealthMiddleware)
app.include_router(dependencies_router)
```

## См. также

- [Быстрый старт](getting-started.ru.md) — базовая настройка и первый пример
- [Чекеры](checkers.ru.md) — все 9 встроенных чекеров
- [Конфигурация](configuration.ru.md) — все опции и значения по умолчанию
- [FastAPI-интеграция](fastapi.ru.md) — lifespan и middleware
- [Troubleshooting](troubleshooting.ru.md) — частые проблемы и решения
