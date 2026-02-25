*[English version](getting-started.md)*

# Быстрый старт

Это руководство охватывает установку, базовую настройку и первую проверку
зависимости с dephealth Python SDK.

## Предварительные требования

- Python 3.11 или новее
- pip (или uv, poetry)
- Работающая зависимость для мониторинга (PostgreSQL, Redis, HTTP-сервис и т.д.)

## Установка

Базовая установка (HTTP и TCP чекеры включены):

```bash
pip install dephealth
```

С конкретными чекерами:

```bash
pip install dephealth[postgres,redis]
```

С FastAPI-интеграцией:

```bash
pip install dephealth[fastapi]
```

Все чекеры и FastAPI:

```bash
pip install dephealth[all]
```

### Доступные extras

| Extra | Зависимость | Описание |
| --- | --- | --- |
| `postgres` | asyncpg | PostgreSQL чекер |
| `mysql` | aiomysql | MySQL чекер |
| `redis` | redis[hiredis] | Redis чекер |
| `amqp` | aio-pika | RabbitMQ (AMQP) чекер |
| `kafka` | aiokafka | Kafka чекер |
| `ldap` | ldap3 | LDAP чекер |
| `grpc` | grpcio, grpcio-health-checking | gRPC чекер |
| `fastapi` | fastapi, uvicorn | FastAPI-интеграция |
| `all` | все вышеперечисленное | Всё |

## Минимальный пример

Мониторинг одной HTTP-зависимости:

```python
import asyncio
from dephealth.api import DependencyHealth, http_check

dh = DependencyHealth("my-service", "my-team",
    http_check("payment-api",
        url="http://payment.svc:8080",
        critical=True,
    ),
)

async def main():
    await dh.start()

    # Метрики доступны через prometheus_client
    # ... приложение работает ...

    await dh.stop()

asyncio.run(main())
```

После запуска появляются Prometheus-метрики:

```text
app_dependency_health{name="my-service",group="my-team",dependency="payment-api",type="http",host="payment.svc",port="8080",critical="yes"} 1
app_dependency_latency_seconds_bucket{name="my-service",group="my-team",dependency="payment-api",type="http",host="payment.svc",port="8080",critical="yes",le="0.01"} 42
```

## Ключевые концепции

### Name и Group

Каждый экземпляр `DependencyHealth` требует два идентификатора:

- **name** — уникальное имя приложения (например, `"my-service"`)
- **group** — логическая группа, к которой принадлежит сервис (например, `"my-team"`, `"payments"`)

Оба значения появляются как метки во всех экспортируемых метриках. Правила
валидации: `[a-z][a-z0-9-]*`, 1-63 символа.

Если не переданы как аргументы, SDK читает переменные окружения
`DEPHEALTH_NAME` и `DEPHEALTH_GROUP`.

### Зависимости

Каждая зависимость регистрируется через фабричные функции, передаваемые
в конструктор `DependencyHealth`:

| Фабричная функция | DependencyType | Описание |
| --- | --- | --- |
| `http_check()` | `HTTP` | HTTP-сервис |
| `grpc_check()` | `GRPC` | gRPC-сервис |
| `tcp_check()` | `TCP` | TCP-эндпоинт |
| `postgres_check()` | `POSTGRES` | PostgreSQL |
| `mysql_check()` | `MYSQL` | MySQL |
| `redis_check()` | `REDIS` | Redis |
| `amqp_check()` | `AMQP` | RabbitMQ (AMQP) |
| `kafka_check()` | `KAFKA` | Apache Kafka |
| `ldap_check()` | `LDAP` | LDAP-сервер |

Каждая зависимость требует:

- **Имя** (первый аргумент) — идентифицирует зависимость в метриках
- **Эндпоинт** — через `url=` или `host=` + `port=`
- **Флаг critical** — `critical=True` или `critical=False` (обязателен)

### Жизненный цикл

1. **Создание** — `DependencyHealth("name", "group", ...specs)`
2. **Запуск** — `await dh.start()` запускает периодические проверки
3. **Работа** — проверки выполняются с настроенным интервалом (по умолчанию 15с)
4. **Остановка** — `await dh.stop()` отменяет все задачи проверок

## Несколько зависимостей

```python
from datetime import timedelta
from dephealth.api import (
    DependencyHealth,
    http_check,
    postgres_check,
    redis_check,
    grpc_check,
)

dh = DependencyHealth("my-service", "my-team",
    # Глобальные настройки
    check_interval=timedelta(seconds=30),
    timeout=timedelta(seconds=3),

    # PostgreSQL
    postgres_check("postgres-main",
        url="postgresql://user:pass@pg.svc:5432/mydb",
        critical=True,
    ),

    # Redis
    redis_check("redis-cache",
        url="redis://redis.svc:6379",
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
)
```

## Проверка состояния

### Простой статус

```python
health = dh.health()
# {"postgres-main": True, "redis-cache": True, "auth-service": True}

# Для readiness probe
all_healthy = all(health.values())
```

### Детальный статус

```python
details = dh.health_details()
for key, ep in details.items():
    print(f"{key}: healthy={ep.healthy} status={ep.status} "
          f"latency={ep.latency_millis():.1f}ms")
```

`health_details()` возвращает объект `EndpointStatus` с состоянием здоровья,
категорией статуса, задержкой, временными метками и пользовательскими метками.
До завершения первой проверки `healthy` равен `None`, а `status` — `"unknown"`.

## Следующие шаги

- [Чекеры](checkers.ru.md) — подробное руководство по всем 9 чекерам
- [Конфигурация](configuration.ru.md) — все опции, значения по умолчанию и переменные окружения
- [Connection Pools](connection-pools.ru.md) — интеграция с существующими пулами
- [FastAPI-интеграция](fastapi.ru.md) — lifespan, middleware и health endpoint
- [Аутентификация](authentication.ru.md) — аутентификация для HTTP, gRPC и БД
- [Метрики](metrics.ru.md) — справочник по Prometheus-метрикам и PromQL
- [API Reference](api-reference.ru.md) — полный справочник по публичным классам
- [Troubleshooting](troubleshooting.ru.md) — частые проблемы и решения
- [Миграция](migration.ru.md) — инструкции по обновлению версий
- [Стиль кода](code-style.ru.md) — соглашения по стилю кода Python
- [Примеры](examples/) — полные рабочие примеры
