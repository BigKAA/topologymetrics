*[English version](authentication.md)*

# Аутентификация

Руководство по опциям аутентификации для всех чекеров
dephealth Python SDK.

## HTTP-чекер

### Bearer-токен

```python
from dephealth.api import http_check

http_check("secure-api",
    url="https://api.svc:443",
    bearer_token="eyJhbGciOiJSUzI1NiIs...",
    critical=True,
)
```

Отправляет заголовок `Authorization: Bearer <token>` с каждой проверкой.

### Basic Auth

```python
http_check("basic-api",
    url="http://api.svc:8080",
    basic_auth=("admin", "secret"),
    critical=True,
)
```

Отправляет заголовок `Authorization: Basic <base64>`.

### Кастомные заголовки

```python
http_check("api-key-service",
    url="http://api.svc:8080",
    headers={"X-Api-Key": "my-secret-key"},
    critical=True,
)
```

Кастомные заголовки отправляются с каждым запросом проверки.

### TLS

```python
# Включить TLS (автоматически для https:// URL)
http_check("tls-api",
    url="https://api.svc:443",
    critical=True,
)

# Пропустить проверку TLS (только dev/test)
http_check("self-signed-api",
    url="https://api.svc:443",
    tls_skip_verify=True,
    critical=True,
)
```

## gRPC-чекер

### Bearer-токен

```python
from dephealth.api import grpc_check

grpc_check("secure-grpc",
    host="grpc.svc",
    port="443",
    tls=True,
    bearer_token="eyJhbGciOiJSUzI1NiIs...",
    critical=True,
)
```

Отправляет токен как gRPC metadata: `authorization: Bearer <token>`.

### Basic Auth

```python
grpc_check("basic-grpc",
    host="grpc.svc",
    port="9090",
    basic_auth=("admin", "secret"),
    critical=True,
)
```

### Кастомные метаданные

```python
grpc_check("custom-grpc",
    host="grpc.svc",
    port="9090",
    metadata={"x-api-key": "my-secret-key"},
    critical=True,
)
```

### TLS

```python
# Включить TLS
grpc_check("tls-grpc",
    host="grpc.svc",
    port="443",
    tls=True,
    critical=True,
)

# Пропустить проверку TLS
grpc_check("self-signed-grpc",
    host="grpc.svc",
    port="443",
    tls=True,
    tls_skip_verify=True,
    critical=True,
)
```

## Чекеры баз данных

### PostgreSQL

Credentials передаются через URL:

```python
from dephealth.api import postgres_check

postgres_check("postgres-main",
    url="postgresql://user:password@pg.svc:5432/mydb",
    critical=True,
)
```

С pool-интеграцией пул уже содержит настроенные credentials:

```python
import asyncpg

pg_pool = await asyncpg.create_pool("postgresql://user:password@pg.svc:5432/mydb")

postgres_check("postgres-main",
    pool=pg_pool,
    critical=True,
)
```

### MySQL

Credentials через URL:

```python
from dephealth.api import mysql_check

mysql_check("mysql-main",
    url="mysql://user:password@mysql.svc:3306/mydb",
    critical=True,
)
```

### Redis

Через URL или явный пароль:

```python
from dephealth.api import redis_check

# Через URL
redis_check("redis-cache",
    url="redis://:password@redis.svc:6379/0",
    critical=False,
)

# Явный пароль
redis_check("redis-cache",
    host="redis.svc",
    port="6379",
    password="secret",
    db=0,
    critical=False,
)
```

## AMQP (RabbitMQ)

Credentials через URL:

```python
from dephealth.api import amqp_check

amqp_check("rabbitmq",
    url="amqp://user:password@rabbitmq.svc:5672/vhost",
    critical=True,
)
```

## LDAP

### Анонимный bind

```python
from dephealth.api import ldap_check

ldap_check("ldap-server",
    url="ldap://ldap.svc:389",
    check_method="ANONYMOUS_BIND",
    critical=False,
)
```

### Simple bind (аутентифицированный)

```python
ldap_check("ldap-auth",
    url="ldap://ldap.svc:389",
    check_method="SIMPLE_BIND",
    bind_dn="cn=monitor,dc=corp,dc=com",
    bind_password="secret",
    critical=True,
)
```

### LDAPS (TLS)

```python
ldap_check("ldap-secure",
    url="ldaps://ldap.svc:636",
    check_method="SIMPLE_BIND",
    bind_dn="cn=monitor,dc=corp,dc=com",
    bind_password="secret",
    critical=True,
)
```

### StartTLS

```python
ldap_check("ldap-starttls",
    url="ldap://ldap.svc:389",
    check_method="SIMPLE_BIND",
    bind_dn="cn=monitor,dc=corp,dc=com",
    bind_password="secret",
    start_tls=True,
    critical=True,
)
```

## Классификация ошибок аутентификации

| Чекер | Условие ошибки | Категория статуса | Детали статуса |
| --- | --- | --- | --- |
| HTTP | Ответ 401/403 | `auth_error` | `auth_error` |
| gRPC | Статус UNAUTHENTICATED | `auth_error` | `auth_error` |
| PostgreSQL | Неверные credentials | `auth_error` | `auth_error` |
| MySQL | Неверные credentials | `auth_error` | `auth_error` |
| Redis | AUTH failed | `auth_error` | `auth_error` |
| AMQP | AUTH_FAILURE | `auth_error` | `auth_error` |
| LDAP | Ошибка bind | `auth_error` | `auth_error` |

## Лучшие практики безопасности

1. **Никогда не хардкодьте credentials** — используйте переменные окружения или хранилища секретов
2. **Используйте TLS в продакшне** — включайте TLS для HTTP и gRPC чекеров
3. **Используйте pool-интеграцию** — pool-соединения уже содержат настроенные credentials,
   снижая exposure
4. **Минимальные привилегии** — используйте read-only credentials для проверок
   (напр., `SELECT 1` для баз данных)
5. **Ротация токенов** — периодически ротируйте bearer-токены и пароли

### Пример с переменными окружения

```python
import os

http_check("secure-api",
    url=os.environ["API_URL"],
    bearer_token=os.environ.get("API_TOKEN"),
    critical=True,
)

postgres_check("postgres-main",
    url=os.environ["DATABASE_URL"],
    critical=True,
)
```

## См. также

- [Чекеры](checkers.ru.md) — все 9 чекеров с примерами
- [Конфигурация](configuration.ru.md) — все опции и значения по умолчанию
- [Connection Pools](connection-pools.ru.md) — руководство по pool-интеграции
- [Troubleshooting](troubleshooting.ru.md) — частые проблемы и решения
