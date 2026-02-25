*[English version](fastapi.md)*

# FastAPI-интеграция

Руководство по пакету `dephealth_fastapi`: управление жизненным циклом,
Prometheus middleware и эндпоинт состояния зависимостей.

## Установка

```bash
pip install dephealth[fastapi]
```

Устанавливает `dephealth`, `fastapi` и `uvicorn`.

## Быстрый старт

```python
from fastapi import FastAPI
from dephealth.api import http_check, postgres_check
from dephealth_fastapi import dephealth_lifespan, DepHealthMiddleware, dependencies_router

app = FastAPI(
    lifespan=dephealth_lifespan("my-service", "my-team",
        postgres_check("postgres-main",
            url="postgresql://user:pass@pg.svc:5432/mydb",
            critical=True,
        ),
        http_check("payment-api",
            url="http://payment.svc:8080",
            critical=True,
        ),
    )
)

# Prometheus-метрики на /metrics
app.add_middleware(DepHealthMiddleware)

# Эндпоинт состояния зависимостей на /health/dependencies
app.include_router(dependencies_router)
```

## Lifespan

`dephealth_lifespan()` — фабрика, возвращающая callable для lifespan
FastAPI. Управляет полным жизненным циклом: создаёт экземпляр
`DependencyHealth`, запускает мониторинг при старте приложения
и останавливает при завершении.

### Сигнатура

```python
def dephealth_lifespan(
    name: str,
    group: str,
    *specs: _DependencySpec,
    check_interval: timedelta | None = None,
    timeout: timedelta | None = None,
    registry: CollectorRegistry | None = None,
    log: logging.Logger | None = None,
) -> Callable[..., AsyncContextManager[None]]
```

Параметры аналогичны конструктору `DependencyHealth`. См.
[Конфигурация](configuration.ru.md) для деталей.

### Использование

```python
from fastapi import FastAPI
from datetime import timedelta

app = FastAPI(
    lifespan=dephealth_lifespan("my-service", "my-team",
        postgres_check("postgres", url="postgresql://...", critical=True),
        redis_check("redis", url="redis://...", critical=False),
        check_interval=timedelta(seconds=30),
    )
)
```

### Доступ к DependencyHealth

После запуска приложения экземпляр `DependencyHealth` доступен через
`app.state.dephealth`:

```python
@app.get("/custom-health")
async def custom_health(request: Request):
    dh = request.app.state.dephealth
    health = dh.health()
    return {"status": "healthy" if all(health.values()) else "degraded"}
```

## Middleware

`DepHealthMiddleware` перехватывает запросы к `/metrics` и возвращает
вывод в текстовом формате Prometheus.

### Конфигурация

```python
app.add_middleware(
    DepHealthMiddleware,
    metrics_path="/metrics",       # по умолчанию
    registry=None,                 # по умолчанию: стандартный prometheus_client registry
)
```

| Параметр | Тип | По умолчанию | Описание |
| --- | --- | --- | --- |
| `metrics_path` | `str` | `"/metrics"` | URL-путь для Prometheus scraping |
| `registry` | `CollectorRegistry \| None` | default | Кастомный Prometheus registry |

### Как это работает

1. Если путь запроса совпадает с `metrics_path`, middleware генерирует
   текстовый вывод Prometheus и возвращает его с `Content-Type: text/plain; version=0.0.4`.
2. Все остальные запросы передаются приложению.

## Эндпоинт состояния зависимостей

`dependencies_router` предоставляет GET-эндпоинт `/health/dependencies`,
возвращающий статус зависимостей в формате JSON.

### Конфигурация

```python
app.include_router(dependencies_router)
```

### Формат ответа

**Здоров (200):**

```json
{
    "status": "healthy",
    "dependencies": {
        "postgres-main": true,
        "redis-cache": true,
        "payment-api": true
    }
}
```

**Деградация (503):**

```json
{
    "status": "degraded",
    "dependencies": {
        "postgres-main": true,
        "redis-cache": false,
        "payment-api": true
    }
}
```

### Коды ответа

| Код | Условие |
| --- | --- |
| 200 | Все зависимости здоровы |
| 503 | Любая зависимость нездорова или статус неизвестен |

## Полный пример

```python
import os
from datetime import timedelta
from fastapi import FastAPI
import asyncpg
from redis.asyncio import Redis

from dephealth.api import postgres_check, redis_check, http_check, grpc_check
from dephealth_fastapi import dephealth_lifespan, DepHealthMiddleware, dependencies_router

# Пулы соединений (создаются до запуска приложения)
pg_pool = None
redis_client = None

async def setup_pools():
    global pg_pool, redis_client
    pg_pool = await asyncpg.create_pool(os.environ["DATABASE_URL"])
    redis_client = Redis.from_url(os.environ["REDIS_URL"])

app = FastAPI(
    lifespan=dephealth_lifespan("my-service", "my-team",
        postgres_check("postgres", pool=pg_pool, critical=True),
        redis_check("redis", client=redis_client, critical=False),
        http_check("payment-api",
            url=os.environ["PAYMENT_URL"],
            bearer_token=os.environ.get("PAYMENT_TOKEN"),
            critical=True,
        ),
        grpc_check("auth-service",
            host="auth.svc",
            port="9090",
            critical=True,
        ),
        check_interval=timedelta(seconds=15),
    )
)

app.add_middleware(DepHealthMiddleware)
app.include_router(dependencies_router)

@app.get("/")
async def root():
    return {"service": "my-service"}
```

## Кастомный health-эндпоинт

Если `dependencies_router` не подходит, создайте свой эндпоинт:

```python
from fastapi import FastAPI, Request, Response
import json

@app.get("/health/dependencies")
async def health_dependencies(request: Request):
    dh = request.app.state.dephealth
    health = dh.health()
    all_healthy = all(health.values())

    return Response(
        content=json.dumps({
            "status": "healthy" if all_healthy else "unhealthy",
            "dependencies": health,
        }),
        media_type="application/json",
        status_code=200 if all_healthy else 503,
    )
```

### Детальный health-эндпоинт

```python
@app.get("/health/dependencies/details")
async def health_details(request: Request):
    dh = request.app.state.dephealth
    details = dh.health_details()
    return {
        key: ep.to_dict() for key, ep in details.items()
    }
```

## См. также

- [Быстрый старт](getting-started.ru.md) — базовая настройка и первый пример
- [Конфигурация](configuration.ru.md) — все опции и значения по умолчанию
- [Метрики](metrics.ru.md) — справочник Prometheus-метрик
- [Troubleshooting](troubleshooting.ru.md) — частые проблемы и решения
- [Примеры](examples/) — полные рабочие примеры
