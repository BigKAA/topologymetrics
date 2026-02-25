*[English version](configuration.md)*

# Конфигурация

Руководство по всем параметрам конфигурации dephealth Python SDK,
включая глобальные настройки, опции для каждой зависимости, переменные
окружения и правила валидации.

## Name и Group

```python
dh = DependencyHealth("my-service", "my-team",
    # ... спецификации зависимостей
)
```

| Параметр | Обязательный | Валидация | Fallback из env |
| --- | --- | --- | --- |
| `name` | Да | `[a-z][a-z0-9-]*`, 1-63 символа | `DEPHEALTH_NAME` |
| `group` | Да | `[a-z][a-z0-9-]*`, 1-63 символа | `DEPHEALTH_GROUP` |

Приоритет: аргумент API > переменная окружения.

Если оба пусты, `DependencyHealth()` бросает `ValueError`.

## Глобальные опции

Глобальные опции передаются в конструктор `DependencyHealth` и применяются
ко всем зависимостям, если не переопределены для конкретной зависимости.

| Опция | Тип | По умолчанию | Диапазон | Описание |
| --- | --- | --- | --- | --- |
| `check_interval` | `timedelta \| None` | 15с | 1с -- 5м | Интервал между проверками |
| `timeout` | `timedelta \| None` | 5с | 1с -- 60с | Таймаут одной проверки |
| `registry` | `CollectorRegistry \| None` | default | -- | Кастомный Prometheus registry |
| `log` | `Logger \| None` | `dephealth` | -- | Кастомный логгер |

### Пример

```python
from datetime import timedelta
from prometheus_client import CollectorRegistry

custom_registry = CollectorRegistry()

dh = DependencyHealth("my-service", "my-team",
    check_interval=timedelta(seconds=30),
    timeout=timedelta(seconds=3),
    registry=custom_registry,
    # ... спецификации зависимостей
)
```

## Общие опции зависимостей

Эти опции могут применяться к любой фабричной функции.

| Опция | Обязательная | По умолчанию | Описание |
| --- | --- | --- | --- |
| `url` | Одно из url/host+port | `""` | Парсинг host и port из URL |
| `host` + `port` | Одно из url/host+port | -- | Явное указание host и port |
| `critical` | Да | -- | Критичная (`True`) или некритичная (`False`) |
| `labels` | Нет | `None` | Словарь пользовательских Prometheus-меток |
| `interval` | Нет | глобальное значение | Интервал проверки для зависимости |
| `timeout` | Нет | глобальное значение | Таймаут проверки для зависимости |

### Указание эндпоинта

Каждая зависимость требует эндпоинт. Используйте один из двух способов:

```python
# Из URL — SDK парсит host и port
postgres_check("postgres-main",
    url="postgresql://user:pass@pg.svc:5432/mydb",
    critical=True,
)

# Явное указание host и port
grpc_check("user-service",
    host="user.svc",
    port="9090",
    critical=True,
)
```

Поддерживаемые схемы URL: `http`, `https`, `grpc`, `tcp`, `postgresql`,
`postgres`, `mysql`, `redis`, `rediss`, `amqp`, `amqps`, `kafka`,
`ldap`, `ldaps`.

Для Kafka поддерживаются multi-host URL:
`kafka://broker1:9092,broker2:9092` — каждый хост создаёт отдельный эндпоинт.

### Флаг critical

Опция `critical` **обязательна** для каждой зависимости. Если не задана через
API, SDK проверяет переменную окружения `DEPHEALTH_<DEP>_CRITICAL`
(значения: `yes`/`no`, `true`/`false`).

### Пользовательские метки

```python
postgres_check("postgres-main",
    url="postgresql://user:pass@pg.svc:5432/mydb",
    critical=True,
    labels={"role": "primary", "shard": "eu-west"},
)
```

Валидация имён меток:

- Должно соответствовать `[a-zA-Z_][a-zA-Z0-9_]*`
- Нельзя использовать зарезервированные имена: `name`, `group`, `dependency`, `type`, `host`, `port`, `critical`

## Опции конкретных чекеров

### HTTP

| Опция | По умолчанию | Описание |
| --- | --- | --- |
| `health_path` | `"/health"` | Путь для проверки здоровья |
| `tls` | `False` | Включить HTTPS |
| `tls_skip_verify` | `False` | Пропустить проверку TLS-сертификата |
| `headers` | `None` | Кастомные HTTP-заголовки |
| `bearer_token` | `None` | Bearer-токен |
| `basic_auth` | `None` | Basic auth `(user, password)` |

### gRPC

| Опция | По умолчанию | Описание |
| --- | --- | --- |
| `service_name` | `""` | Имя сервиса (пусто = общее состояние) |
| `tls` | `False` | Включить TLS |
| `tls_skip_verify` | `False` | Пропустить проверку TLS-сертификата |
| `metadata` | `None` | Кастомные gRPC-метаданные |
| `bearer_token` | `None` | Bearer-токен |
| `basic_auth` | `None` | Basic auth `(user, password)` |

### PostgreSQL

| Опция | По умолчанию | Описание |
| --- | --- | --- |
| `query` | `"SELECT 1"` | SQL-запрос для проверки |
| `pool` | `None` | asyncpg pool (предпочтительно) |

### MySQL

| Опция | По умолчанию | Описание |
| --- | --- | --- |
| `query` | `"SELECT 1"` | SQL-запрос для проверки |
| `pool` | `None` | aiomysql pool (предпочтительно) |

### Redis

| Опция | По умолчанию | Описание |
| --- | --- | --- |
| `password` | `None` | Пароль Redis (standalone-режим) |
| `db` | `None` | Номер БД (standalone-режим) |
| `client` | `None` | redis-py async клиент (предпочтительно) |

### AMQP

Нет специфичных опций кроме `url` или `host`/`port`.

### Kafka

Нет специфичных опций кроме `url` или `host`/`port`.

### LDAP

| Опция | По умолчанию | Описание |
| --- | --- | --- |
| `check_method` | `ROOT_DSE` | Метод проверки: `ANONYMOUS_BIND`, `SIMPLE_BIND`, `ROOT_DSE`, `SEARCH` |
| `bind_dn` | `""` | Bind DN для simple bind или search |
| `bind_password` | `""` | Пароль для bind |
| `base_dn` | `""` | Base DN для поисковых операций |
| `search_filter` | `"(objectClass=*)"` | LDAP-фильтр поиска |
| `search_scope` | `BASE` | Область поиска: `BASE`, `ONE`, `SUB` |
| `start_tls` | `False` | Включить StartTLS (несовместимо с `ldaps://`) |
| `tls_skip_verify` | `False` | Пропустить проверку TLS-сертификата |
| `client` | `None` | ldap3 Connection для pool-интеграции |

### TCP

Нет специфичных опций кроме `host`/`port`.

## Переменные окружения

| Переменная | Описание | Пример |
| --- | --- | --- |
| `DEPHEALTH_NAME` | Имя приложения (fallback) | `my-service` |
| `DEPHEALTH_GROUP` | Логическая группа (fallback) | `my-team` |
| `DEPHEALTH_<DEP>_CRITICAL` | Критичность зависимости (`yes`/`no`) | `yes` |
| `DEPHEALTH_<DEP>_LABEL_<KEY>` | Значение пользовательской метки | `primary` |

`<DEP>` — имя зависимости в UPPER_SNAKE_CASE:
дефисы заменяются подчёркиваниями, всё в верхнем регистре.

Пример: зависимость `"postgres-main"` → префикс `DEPHEALTH_POSTGRES_MAIN_`.

### Правила приоритета

Значения API всегда имеют приоритет над переменными окружения:

1. **name/group**: аргумент API > `DEPHEALTH_NAME`/`DEPHEALTH_GROUP` > ошибка
2. **critical**: опция `critical=` > `DEPHEALTH_<DEP>_CRITICAL` > ошибка
3. **labels**: `labels=` > `DEPHEALTH_<DEP>_LABEL_<KEY>` (API выигрывает при конфликте)

### Пример

```bash
export DEPHEALTH_NAME=my-service
export DEPHEALTH_GROUP=my-team
export DEPHEALTH_POSTGRES_MAIN_CRITICAL=yes
export DEPHEALTH_POSTGRES_MAIN_LABEL_ROLE=primary
export DEPHEALTH_POSTGRES_MAIN_LABEL_SHARD=eu-west
```

```python
# name и group из env vars, critical и labels из env vars
dh = DependencyHealth("", "",
    postgres_check("postgres-main",
        url=os.environ["DATABASE_URL"],
    ),
    # Critical и labels из DEPHEALTH_POSTGRES_MAIN_*
)
```

## Приоритет опций

Для интервала и таймаута цепочка приоритетов:

```text
опция зависимости > глобальная опция > значение по умолчанию
```

| Настройка | Для зависимости | Глобальная | По умолчанию |
| --- | --- | --- | --- |
| Интервал проверки | `interval=` | `check_interval=` | 15с |
| Таймаут | `timeout=` | `timeout=` | 5с |

## Значения по умолчанию

| Параметр | Значение |
| --- | --- |
| Интервал проверки | 15 секунд |
| Таймаут | 5 секунд |
| Начальная задержка | 5 секунд |
| Порог неудач | 1 |
| Порог успехов | 1 |
| HTTP health path | `/health` |
| HTTP TLS | `False` |
| Redis DB | `None` |
| Redis password | `None` |
| PostgreSQL query | `SELECT 1` |
| MySQL query | `SELECT 1` |
| gRPC service name | `""` (общее состояние сервера) |
| LDAP check method | `ROOT_DSE` |
| LDAP search filter | `(objectClass=*)` |
| LDAP search scope | `BASE` |

## Правила валидации

`DependencyHealth()` валидирует всю конфигурацию и бросает `ValueError`
при нарушении правил:

| Правило | Ошибка |
| --- | --- |
| Отсутствует name | `instance name is required: pass it as argument or set DEPHEALTH_NAME` |
| Отсутствует group | `group is required: pass it as argument or set DEPHEALTH_GROUP` |
| Неверный формат name/group | `instance name must match [a-z][a-z0-9-]*, got '...'` |
| Имя слишком длинное | `instance name must be 1-63 characters` |
| Отсутствует critical | ошибка валидации |
| Отсутствует URL или host/port | ошибка конфигурации зависимости |
| Неверное имя метки | `label name must match [a-zA-Z_][a-zA-Z0-9_]*, got '...'` |
| Зарезервированное имя метки | `label name '...' is reserved` |
| LDAP simple_bind без credentials | `LDAP simple_bind requires bind_dn and bind_password` |
| LDAP search без base_dn | `LDAP search requires base_dn` |
| LDAP start_tls + ldaps | `start_tls and ldaps:// are incompatible` |

## См. также

- [Быстрый старт](getting-started.ru.md) — базовая настройка и первый пример
- [Чекеры](checkers.ru.md) — детальные опции чекеров
- [Аутентификация](authentication.ru.md) — аутентификация для HTTP и gRPC
- [Connection Pools](connection-pools.ru.md) — интеграция с asyncpg, redis-py, aiomysql
- [FastAPI-интеграция](fastapi.ru.md) — конфигурация lifespan и middleware
- [API Reference](api-reference.ru.md) — полный справочник по публичным классам
- [Troubleshooting](troubleshooting.ru.md) — частые проблемы и решения
