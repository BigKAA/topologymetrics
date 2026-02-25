*[English version](checkers.md)*

# Чекеры

Руководство по всем 9 встроенным чекерам dephealth Python SDK.

## Обзор

| Чекер | Фабрика | Extra | Метод проверки |
| --- | --- | --- | --- |
| [HTTP](#http) | `http_check()` | — | GET-запрос к health path |
| [gRPC](#grpc) | `grpc_check()` | `grpc` | gRPC Health/Check протокол |
| [TCP](#tcp) | `tcp_check()` | — | TCP-соединение |
| [PostgreSQL](#postgresql) | `postgres_check()` | `postgres` | Выполнение SQL-запроса |
| [MySQL](#mysql) | `mysql_check()` | `mysql` | Выполнение SQL-запроса |
| [Redis](#redis) | `redis_check()` | `redis` | Команда PING |
| [AMQP](#amqp) | `amqp_check()` | `amqp` | Проверка соединения |
| [Kafka](#kafka) | `kafka_check()` | `kafka` | Получение метаданных кластера |
| [LDAP](#ldap) | `ldap_check()` | `ldap` | Bind/search/RootDSE |

## HTTP

Выполняет HTTP GET-запрос к настроенному health path.

### Опции

| Опция | По умолчанию | Описание |
| --- | --- | --- |
| `url` | `""` | HTTP URL (парсится для host/port/scheme) |
| `host` | `""` | Хост (если `url` не указан) |
| `port` | `"80"` | Порт |
| `health_path` | `"/health"` | Путь для проверки |
| `tls` | `False` | Включить HTTPS |
| `tls_skip_verify` | `False` | Пропустить проверку TLS-сертификата |
| `headers` | `None` | Кастомные HTTP-заголовки |
| `bearer_token` | `None` | Bearer-токен для заголовка Authorization |
| `basic_auth` | `None` | Basic auth `(user, password)` |

### Пример

```python
from dephealth.api import http_check

# Базовый
http_check("payment-api",
    url="http://payment.svc:8080",
    critical=True,
)

# С аутентификацией
http_check("secure-api",
    url="https://api.svc:443",
    health_path="/healthz",
    bearer_token="eyJhbG...",
    critical=True,
)

# С кастомными заголовками
http_check("internal-api",
    url="http://api.svc:8080",
    headers={"X-Api-Key": "secret"},
    critical=False,
)
```

### Классификация ошибок

| Ошибка | Категория статуса | Детали статуса |
| --- | --- | --- |
| Соединение отклонено | `connection_error` | `connection_refused` |
| Ошибка DNS | `dns_error` | `dns_error` |
| Таймаут запроса | `timeout` | `timeout` |
| Ошибка TLS | `tls_error` | `tls_error` |
| HTTP 401/403 | `auth_error` | `auth_error` |
| HTTP >= 300 | `unhealthy` | `http_<status_code>` |

## gRPC

Использует стандартный gRPC Health Checking Protocol
(`grpc.health.v1.Health/Check`).

### Опции

| Опция | По умолчанию | Описание |
| --- | --- | --- |
| `url` | `""` | gRPC URL (парсится для host/port) |
| `host` | `""` | Хост (если `url` не указан) |
| `port` | `"50051"` | Порт |
| `service_name` | `""` | Имя сервиса (пусто = общее состояние) |
| `tls` | `False` | Включить TLS |
| `tls_skip_verify` | `False` | Пропустить проверку TLS |
| `metadata` | `None` | Кастомные gRPC-метаданные |
| `bearer_token` | `None` | Bearer-токен |
| `basic_auth` | `None` | Basic auth `(user, password)` |

### Пример

```python
from dephealth.api import grpc_check

grpc_check("user-service",
    host="user.svc",
    port="9090",
    critical=True,
)

# С TLS и аутентификацией
grpc_check("secure-grpc",
    host="grpc.svc",
    port="443",
    tls=True,
    bearer_token="eyJhbG...",
    critical=True,
)
```

### Классификация ошибок

| Ошибка | Категория статуса | Детали статуса |
| --- | --- | --- |
| Соединение отклонено | `connection_error` | `connection_refused` |
| Ошибка DNS | `dns_error` | `dns_error` |
| Deadline exceeded | `timeout` | `timeout` |
| Ошибка TLS | `tls_error` | `tls_error` |
| Unauthenticated | `auth_error` | `auth_error` |
| NOT_SERVING | `unhealthy` | `grpc_not_serving` |

## TCP

Открывает TCP-соединение для проверки доступности. Данные не отправляются.

### Опции

| Опция | По умолчанию | Описание |
| --- | --- | --- |
| `host` | — | Хост (обязателен) |
| `port` | — | Порт (обязателен) |

### Пример

```python
from dephealth.api import tcp_check

tcp_check("memcached",
    host="memcached.svc",
    port="11211",
    critical=False,
)
```

### Классификация ошибок

| Ошибка | Категория статуса | Детали статуса |
| --- | --- | --- |
| Соединение отклонено | `connection_error` | `connection_refused` |
| Ошибка DNS | `dns_error` | `dns_error` |
| Таймаут | `timeout` | `timeout` |

## PostgreSQL

Выполняет SQL-запрос через asyncpg. Поддерживает standalone-режим
(новое соединение) и pool-режим (через существующий asyncpg pool).

### Опции

| Опция | По умолчанию | Описание |
| --- | --- | --- |
| `url` | `""` | PostgreSQL URL (`postgresql://user:pass@host:5432/db`) |
| `host` | `""` | Хост (если `url` не указан) |
| `port` | `"5432"` | Порт |
| `query` | `"SELECT 1"` | SQL-запрос для проверки |
| `pool` | `None` | asyncpg pool (предпочтительно) |

### Пример

```python
from dephealth.api import postgres_check

# Standalone-режим
postgres_check("postgres-main",
    url="postgresql://user:pass@pg.svc:5432/mydb",
    critical=True,
)

# Pool-режим (предпочтительно)
import asyncpg
pg_pool = await asyncpg.create_pool("postgresql://user:pass@pg.svc:5432/mydb")

postgres_check("postgres-main",
    pool=pg_pool,
    critical=True,
)
```

### Классификация ошибок

| Ошибка | Категория статуса | Детали статуса |
| --- | --- | --- |
| Соединение отклонено | `connection_error` | `connection_refused` |
| Ошибка DNS | `dns_error` | `dns_error` |
| Таймаут запроса | `timeout` | `timeout` |
| Ошибка аутентификации | `auth_error` | `auth_error` |
| Ошибка запроса | `error` | детали ошибки |

## MySQL

Выполняет SQL-запрос через aiomysql. Поддерживает standalone-режим
и pool-режим.

### Опции

| Опция | По умолчанию | Описание |
| --- | --- | --- |
| `url` | `""` | MySQL URL (`mysql://user:pass@host:3306/db`) |
| `host` | `""` | Хост (если `url` не указан) |
| `port` | `"3306"` | Порт |
| `query` | `"SELECT 1"` | SQL-запрос для проверки |
| `pool` | `None` | aiomysql pool (предпочтительно) |

### Пример

```python
from dephealth.api import mysql_check

# Standalone-режим
mysql_check("mysql-main",
    url="mysql://user:pass@mysql.svc:3306/mydb",
    critical=True,
)

# Pool-режим (предпочтительно)
import aiomysql
mysql_pool = await aiomysql.create_pool(
    host="mysql.svc", port=3306, user="user", password="pass", db="mydb",
)

mysql_check("mysql-main",
    pool=mysql_pool,
    critical=True,
)
```

### Классификация ошибок

| Ошибка | Категория статуса | Детали статуса |
| --- | --- | --- |
| Соединение отклонено | `connection_error` | `connection_refused` |
| Ошибка DNS | `dns_error` | `dns_error` |
| Таймаут запроса | `timeout` | `timeout` |
| Ошибка аутентификации | `auth_error` | `auth_error` |
| Ошибка запроса | `error` | детали ошибки |

## Redis

Отправляет команду PING. Поддерживает standalone-режим и pool-режим
(через существующий redis-py async клиент).

### Опции

| Опция | По умолчанию | Описание |
| --- | --- | --- |
| `url` | `""` | Redis URL (`redis://host:6379`) |
| `host` | `""` | Хост (если `url` не указан) |
| `port` | `"6379"` | Порт |
| `password` | `None` | Пароль (standalone-режим) |
| `db` | `None` | Номер БД (standalone-режим) |
| `client` | `None` | redis-py async клиент (предпочтительно) |

### Пример

```python
from dephealth.api import redis_check

# Standalone-режим
redis_check("redis-cache",
    url="redis://redis.svc:6379",
    critical=False,
)

# Pool-режим (предпочтительно)
from redis.asyncio import Redis
redis_client = Redis.from_url("redis://redis.svc:6379")

redis_check("redis-cache",
    client=redis_client,
    critical=False,
)
```

### Классификация ошибок

| Ошибка | Категория статуса | Детали статуса |
| --- | --- | --- |
| Соединение отклонено | `connection_error` | `connection_refused` |
| Ошибка DNS | `dns_error` | `dns_error` |
| Таймаут PING | `timeout` | `timeout` |
| Ошибка аутентификации | `auth_error` | `auth_error` |

## AMQP

Устанавливает соединение с брокером RabbitMQ через aio-pika.

### Опции

| Опция | По умолчанию | Описание |
| --- | --- | --- |
| `url` | `""` | AMQP URL (`amqp://user:pass@host:5672/vhost`) |
| `host` | `""` | Хост (если `url` не указан) |
| `port` | `"5672"` | Порт |

### Пример

```python
from dephealth.api import amqp_check

amqp_check("rabbitmq",
    url="amqp://user:pass@rabbitmq.svc:5672/",
    critical=True,
)
```

### Классификация ошибок

| Ошибка | Категория статуса | Детали статуса |
| --- | --- | --- |
| Соединение отклонено | `connection_error` | `connection_refused` |
| Ошибка DNS | `dns_error` | `dns_error` |
| Таймаут | `timeout` | `timeout` |
| Ошибка аутентификации | `auth_error` | `auth_error` |

## Kafka

Получает метаданные кластера через aiokafka для проверки доступности брокера.

### Опции

| Опция | По умолчанию | Описание |
| --- | --- | --- |
| `url` | `""` | Kafka URL (`kafka://host1:9092,host2:9092`) |
| `host` | `""` | Хост (если `url` не указан) |
| `port` | `"9092"` | Порт |

Multi-host URL создают отдельные эндпоинты для каждого брокера.

### Пример

```python
from dephealth.api import kafka_check

kafka_check("kafka-main",
    url="kafka://kafka-1.svc:9092,kafka-2.svc:9092",
    critical=True,
)
```

### Классификация ошибок

| Ошибка | Категория статуса | Детали статуса |
| --- | --- | --- |
| Соединение отклонено | `connection_error` | `connection_refused` |
| Ошибка DNS | `dns_error` | `dns_error` |
| Таймаут | `timeout` | `timeout` |
| Нет доступных брокеров | `unhealthy` | `no_brokers` |

## LDAP

Проверяет доступность LDAP-сервера одним из четырёх методов.
Поддерживает LDAP, LDAPS и StartTLS.

### Методы проверки

| Метод | Описание |
| --- | --- |
| `ANONYMOUS_BIND` | Анонимный LDAP bind |
| `SIMPLE_BIND` | Аутентифицированный bind (требует `bind_dn` и `bind_password`) |
| `ROOT_DSE` | Чтение Root DSE (по умолчанию, без credentials) |
| `SEARCH` | LDAP-поиск (требует `base_dn`) |

### Опции

| Опция | По умолчанию | Описание |
| --- | --- | --- |
| `url` | `""` | LDAP URL (`ldap://host:389` или `ldaps://host:636`) |
| `host` | `""` | Хост (если `url` не указан) |
| `port` | `"389"` | Порт |
| `check_method` | `ROOT_DSE` | Метод проверки |
| `bind_dn` | `""` | Bind DN |
| `bind_password` | `""` | Пароль для bind |
| `base_dn` | `""` | Base DN для поиска |
| `search_filter` | `"(objectClass=*)"` | Фильтр поиска |
| `search_scope` | `BASE` | Область поиска: `BASE`, `ONE`, `SUB` |
| `start_tls` | `False` | Включить StartTLS |
| `tls_skip_verify` | `False` | Пропустить проверку TLS |
| `client` | `None` | ldap3 Connection для pool-интеграции |

### Пример

```python
from dephealth.api import ldap_check

# RootDSE (по умолчанию, без credentials)
ldap_check("ldap-server",
    url="ldap://ldap.svc:389",
    critical=False,
)

# Simple bind с credentials
ldap_check("ldap-auth",
    url="ldaps://ldap.svc:636",
    check_method="SIMPLE_BIND",
    bind_dn="cn=monitor,dc=corp,dc=com",
    bind_password="secret",
    critical=True,
)

# Search с StartTLS
ldap_check("ldap-search",
    host="ldap.svc",
    port="389",
    check_method="SEARCH",
    base_dn="dc=example,dc=com",
    search_filter="(objectClass=organizationalUnit)",
    search_scope="ONE",
    start_tls=True,
    critical=False,
)
```

### Режимы TLS

| Режим | Схема URL | `start_tls` | Описание |
| --- | --- | --- | --- |
| Обычный LDAP | `ldap://` | `False` | Без шифрования |
| LDAPS | `ldaps://` | `False` | TLS с начала соединения |
| StartTLS | `ldap://` | `True` | Апгрейд до TLS после соединения |

> `start_tls=True` и `ldaps://` взаимоисключающие.

### Классификация ошибок

| Ошибка | Категория статуса | Детали статуса |
| --- | --- | --- |
| Соединение отклонено | `connection_error` | `connection_refused` |
| Ошибка DNS | `dns_error` | `dns_error` |
| Таймаут | `timeout` | `timeout` |
| Ошибка TLS | `tls_error` | `tls_error` |
| Ошибка bind | `auth_error` | `auth_error` |
| Поиск без результатов | `unhealthy` | `ldap_no_results` |

## Сводка классификации ошибок

Все чекеры классифицируют ошибки в следующие категории статуса:

| Категория статуса | Описание |
| --- | --- |
| `ok` | Проверка успешна |
| `timeout` | Таймаут проверки |
| `connection_error` | Соединение отклонено или сброшено |
| `dns_error` | Ошибка DNS-разрешения |
| `auth_error` | Ошибка аутентификации или авторизации |
| `tls_error` | Ошибка TLS/SSL |
| `unhealthy` | Эндпоинт доступен, но сообщил о нездоровом состоянии |
| `error` | Другая непредвиденная ошибка |

## См. также

- [Быстрый старт](getting-started.ru.md) — базовая настройка и первый пример
- [Конфигурация](configuration.ru.md) — все опции и значения по умолчанию
- [Аутентификация](authentication.ru.md) — детальные опции аутентификации
- [Connection Pools](connection-pools.ru.md) — руководство по pool-интеграции
- [Метрики](metrics.ru.md) — справочник Prometheus-метрик
- [API Reference](api-reference.ru.md) — полный справочник по классам
