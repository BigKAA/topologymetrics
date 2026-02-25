*[English version](python.md)*

# Руководство по интеграции dephealth в существующий Python-сервис

Пошаговая инструкция по добавлению мониторинга зависимостей
в работающий микросервис.

## Миграция на v0.5.0

### Обязательный параметр `group`

v0.5.0 добавляет обязательный параметр `group` (логическая группировка: команда, подсистема, проект).

```python
# v0.4.x
dh = DependencyHealth("my-service",
    postgres_check("postgres-main", ...),
)

# v0.5.0
dh = DependencyHealth("my-service", "my-team",
    postgres_check("postgres-main", ...),
)
```

FastAPI:

```python
# v0.5.0
app = FastAPI(
    lifespan=dephealth_lifespan("my-service", "my-team",
        postgres_check("postgres-main", ...),
    )
)
```

Альтернатива: переменная окружения `DEPHEALTH_GROUP` (API имеет приоритет).

Валидация: те же правила, что и для `name` — `[a-z][a-z0-9-]*`, 1-63 символа.

---

## Миграция на v0.4.1

### Новое: health_details() API

В v0.4.1 добавлен метод `health_details()`, возвращающий детальный статус каждого
endpoint-а. Изменений в существующем API нет — это чисто аддитивная функция.

```python
details = dh.health_details()
# dict[str, EndpointStatus]

for key, ep in details.items():
    print(f"{key}: healthy={ep.healthy} status={ep.status} "
          f"detail={ep.detail} latency={ep.latency_millis():.1f}ms")
```

Поля `EndpointStatus`: `dependency`, `type`, `host`, `port`,
`healthy` (`bool | None`, `None` = неизвестно), `status`, `detail`,
`latency`, `last_checked_at`, `critical`, `labels`.

> **Примечание:** `health_details()` использует ключи по endpoint-ам (`"dep:host:port"`),
> тогда как `health()` — агрегированные ключи (`"dep"`). Метод `to_dict()`
> сериализует `EndpointStatus` в JSON-совместимый словарь.

---

## Миграция на v0.4.0

### Новые метрики статуса (изменения кода не требуются)

v0.4.0 добавляет две новые автоматически экспортируемые метрики Prometheus:

| Метрика | Тип | Описание |
| --- | --- | --- |
| `app_dependency_status` | Gauge (enum) | Категория статуса: 8 серий на endpoint, ровно одна = 1 |
| `app_dependency_status_detail` | Gauge (info) | Детальная причина сбоя: напр. `http_503`, `auth_error` |

**Изменения кода не требуются** — SDK экспортирует эти метрики автоматически наряду с существующими `app_dependency_health` и `app_dependency_latency_seconds`.

### Влияние на хранилище

Каждый endpoint теперь создаёт 9 дополнительных временных рядов (8 для `app_dependency_status` + 1 для `app_dependency_status_detail`). Для сервиса с 5 endpoint-ами это добавляет 45 рядов.

### Новые PromQL-запросы

```promql
# Категория статуса зависимости
app_dependency_status{dependency="postgres-main", status!=""} == 1

# Детальная причина сбоя
app_dependency_status_detail{dependency="postgres-main", detail!=""} == 1

# Алерт на ошибки аутентификации
app_dependency_status{status="auth_error"} == 1
```

Полный список значений статуса см. в [Спецификация — Метрики статуса](../specification.ru.md).

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

- [Быстрый старт](../quickstart/python.ru.md) — минимальные примеры
- [Обзор спецификации](../specification.ru.md) — детали контрактов метрик и поведения
