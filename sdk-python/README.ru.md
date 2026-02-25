*[English version](README.md)*

# dephealth

SDK для мониторинга зависимостей микросервисов через метрики Prometheus.

## Возможности

- Автоматическая проверка здоровья зависимостей (PostgreSQL, MySQL, Redis, RabbitMQ, Kafka, HTTP, gRPC, TCP, LDAP)
- Экспорт метрик Prometheus: `app_dependency_health` (Gauge 0/1), `app_dependency_latency_seconds` (Histogram), `app_dependency_status` (enum), `app_dependency_status_detail` (info)
- Асинхронная архитектура на базе `asyncio`
- Интеграция с FastAPI (middleware, lifespan, endpoints)
- Поддержка connection pool (предпочтительно) и автономных проверок

## Установка

```bash
# Базовая установка
pip install dephealth

# С определёнными чекерами
pip install dephealth[postgres,redis]

# Все чекеры + FastAPI
pip install dephealth[all]
```

## Быстрый старт

### Автономный режим

```python
from dephealth import DepHealth

dh = DepHealth()
dh.add("postgres", url="postgresql://user:pass@localhost:5432/mydb")
dh.add("redis", url="redis://localhost:6379")

await dh.start()
# Метрики доступны через prometheus_client
await dh.stop()
```

### FastAPI

```python
from fastapi import FastAPI
from dephealth_fastapi import DepHealthFastAPI

app = FastAPI()
dh = DepHealthFastAPI(app)
dh.add("postgres", url="postgresql://user:pass@localhost:5432/mydb")
```

## Динамические эндпоинты

Добавление, удаление и замена мониторируемых эндпоинтов в рантайме на
работающем экземпляре (v0.6.0+):

```python
from dephealth import DependencyType, Endpoint
from dephealth.checks.http import HTTPChecker

# После dh.start()...

# Добавить новый эндпоинт
await dh.add_endpoint(
    "api-backend",
    DependencyType.HTTP,
    True,
    Endpoint(host="backend-2.svc", port="8080"),
    HTTPChecker(),
)

# Удалить эндпоинт (отменяет задачу, удаляет метрики)
await dh.remove_endpoint("api-backend", "backend-2.svc", "8080")

# Заменить эндпоинт атомарно
await dh.update_endpoint(
    "api-backend",
    "backend-1.svc", "8080",
    Endpoint(host="backend-3.svc", port="8080"),
    HTTPChecker(),
)
```

Синхронные варианты: `add_endpoint_sync()`,
`remove_endpoint_sync()`, `update_endpoint_sync()`.

Подробности в [руководстве по миграции](docs/migration.ru.md#v050--v060).

## Детализация здоровья

```python
details = dh.health_details()
for key, ep in details.items():
    print(f"{key}: healthy={ep.healthy} status={ep.status} "
          f"latency={ep.latency_millis():.1f}ms")
```

## Конфигурация

| Параметр | По умолчанию | Описание |
| --- | --- | --- |
| `interval` | `15` | Интервал проверки (секунды) |
| `timeout` | `5` | Таймаут проверки (секунды) |

## Поддерживаемые зависимости

| Тип | Extra | Формат URL |
| --- | --- | --- |
| PostgreSQL | `postgres` | `postgresql://user:pass@host:5432/db` |
| MySQL | `mysql` | `mysql://user:pass@host:3306/db` |
| Redis | `redis` | `redis://host:6379` |
| RabbitMQ | `amqp` | `amqp://user:pass@host:5672/vhost` |
| Kafka | `kafka` | `kafka://host1:9092,host2:9092` |
| HTTP | — | `http://host:8080/health` |
| gRPC | `grpc` | `host:50051` (через `FromParams`) |
| TCP | — | `tcp://host:port` |
| LDAP | `ldap` | `ldap://host:389` или `ldaps://host:636` |

## LDAP-чекер

LDAP-чекер поддерживает четыре метода проверки и несколько режимов TLS:

```python
from dephealth.checks.ldap import LdapChecker, LdapCheckMethod, LdapSearchScope

# Запрос RootDSE (по умолчанию)
ldap_checker = LdapChecker(check_method=LdapCheckMethod.ROOT_DSE)

# Простая привязка с учётными данными
ldap_checker = LdapChecker(
    check_method=LdapCheckMethod.SIMPLE_BIND,
    bind_dn="cn=monitor,dc=corp,dc=com",
    bind_password="secret",
    use_tls=True,
)

# Поиск с StartTLS
ldap_checker = LdapChecker(
    check_method=LdapCheckMethod.SEARCH,
    base_dn="dc=example,dc=com",
    search_filter="(objectClass=organizationalUnit)",
    search_scope=LdapSearchScope.ONE,
    start_tls=True,
)
```

Методы проверки: `ANONYMOUS_BIND`, `SIMPLE_BIND`, `ROOT_DSE` (по умолчанию), `SEARCH`.

## Аутентификация

HTTP и gRPC чекеры поддерживают Bearer token, Basic Auth и пользовательские заголовки/метаданные:

```python
http_check("secure-api",
    url="http://api.svc:8080",
    critical=True,
    bearer_token="eyJhbG...",
)

grpc_check("grpc-backend",
    host="backend.svc",
    port=9090,
    critical=True,
    bearer_token="eyJhbG...",
)
```

Все опции описаны в [руководстве по аутентификации](docs/authentication.ru.md).

## Документация

Полная документация доступна в директории [docs/](docs/README.md).

## Лицензия

Apache License 2.0 — см. [LICENSE](https://github.com/BigKAA/topologymetrics/blob/master/LICENSE).
