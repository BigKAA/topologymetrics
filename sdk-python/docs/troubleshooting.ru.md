*[English version](troubleshooting.md)*

# Troubleshooting

Частые проблемы и решения для dephealth Python SDK.

## Метрики не появляются на /metrics

**Симптомы:** `/metrics` возвращает пустой ответ или метрики dephealth не появляются.

**Проверьте:**

1. `DepHealthMiddleware` добавлен в приложение FastAPI:

   ```python
   app.add_middleware(DepHealthMiddleware)
   ```

2. `dephealth_lifespan()` корректно установлен как `lifespan`:

   ```python
   app = FastAPI(lifespan=dephealth_lifespan("name", "group", ...))
   ```

3. Приложение стартовало без ошибок (проверьте логи)

4. Подождите хотя бы один интервал проверки (по умолчанию 15с) для завершения первой проверки

5. При использовании кастомного registry убедитесь, что middleware использует тот же:

   ```python
   custom_registry = CollectorRegistry()
   dh = DependencyHealth("name", "group", ..., registry=custom_registry)
   app.add_middleware(DepHealthMiddleware, registry=custom_registry)
   ```

## Все зависимости показывают unhealthy (0)

**Симптомы:** `app_dependency_health` равен `0` для всех зависимостей.

**Проверьте:**

1. **Сетевой доступ**: зависимости доступны из контейнера/пода

   ```bash
   # Изнутри контейнера
   curl http://payment.svc:8080/health
   nc -zv pg.svc 5432
   ```

2. **DNS-резолвинг**: имена сервисов резолвятся корректно

3. **URL/host/port**: конфигурация корректна

4. **Таймаут**: 5с по умолчанию может быть недостаточно для медленных зависимостей.
   Увеличьте через `timeout=timedelta(seconds=10)`

5. **Логи**: включите debug-логирование для деталей:

   ```python
   import logging
   logging.basicConfig(level=logging.DEBUG)
   ```

## Высокая латентность проверок БД

**Симптомы:** `app_dependency_latency_seconds` показывает высокие значения для PostgreSQL/MySQL.

**Причина:** standalone-режим создаёт новое соединение на каждую проверку,
включая TCP handshake, TLS negotiation и аутентификацию.

**Решение:** используйте pool-интеграцию:

```python
# Вместо
postgres_check("db", url="postgresql://...", critical=True)

# Используйте
pg_pool = await asyncpg.create_pool("postgresql://...")
postgres_check("db", pool=pg_pool, critical=True)
```

Это устраняет overhead на установку соединения. См.
[Connection Pools](connection-pools.ru.md) для деталей.

## gRPC: context deadline exceeded

**Симптомы:** gRPC-проверки падают с таймаутом/deadline exceeded.

**Проверьте:**

1. gRPC-сервис доступен по указанному адресу

2. Сервис реализует `grpc.health.v1.Health/Check`

3. Используйте `host` + `port`, а не `url` для gRPC:

   ```python
   # Правильно
   grpc_check("grpc-svc", host="grpc.svc", port="9090", critical=True)

   # Может не работать
   grpc_check("grpc-svc", url="grpc.svc:9090", critical=True)
   ```

4. Если нужен TLS: `grpc_check(..., tls=True)`

5. Увеличьте таймаут для медленных сервисов:

   ```python
   grpc_check("grpc-svc", host="grpc.svc", port="9090",
       critical=True, timeout=timedelta(seconds=10))
   ```

## Ошибки Connection Refused

**Симптомы:** `app_dependency_status{status="connection_error"} == 1`

**Проверьте:**

1. Зависимость запущена и слушает на ожидаемом порту
2. Правила firewall разрешают соединение
3. В Kubernetes: сервис и pod-селекторы корректны
4. Порт совпадает с реальным портом прослушивания зависимости

## Ошибки таймаута

**Симптомы:** `app_dependency_status{status="timeout"} == 1`

**Проверьте:**

1. Сетевая задержка между сервисом и зависимостью
2. Зависимость под высокой нагрузкой
3. Таймаут по умолчанию (5с) может быть слишком коротким — увеличьте:

   ```python
   postgres_check("slow-db", url="...", critical=True,
       timeout=timedelta(seconds=15))
   ```

4. DNS-резолвинг может быть медленным — проверьте конфигурацию DNS

## Ошибки аутентификации

**Симптомы:** `app_dependency_status{status="auth_error"} == 1`

**Проверьте:**

1. Credentials корректны и не просрочены
2. Bearer-токены валидны и не истекли
3. Пользователь БД имеет необходимые привилегии
4. Пароль Redis совпадает с конфигурацией сервера
5. AMQP vhost доступен с указанными credentials

## AMQP: ошибка подключения к RabbitMQ

**Симптомы:** AMQP-проверки падают с ошибками соединения.

**Укажите полный URL со всеми компонентами:**

```python
amqp_check("rabbitmq",
    url="amqp://user:pass@rabbitmq.svc:5672/vhost",
    critical=False,
)
```

Частые проблемы:

- Пропущен vhost в URL (используйте `/` для vhost по умолчанию)
- Неверный порт (5672 для AMQP, 5671 для AMQPS)
- Нужна URL-кодировка для спецсимволов в пароле

## Ошибки конфигурации LDAP

**Симптомы:** LDAP-проверки немедленно падают с `ValueError`.

**Частые причины:**

1. `SIMPLE_BIND` без credentials:

   ```python
   # Ошибка: LDAP simple_bind requires bind_dn and bind_password
   ldap_check("ldap", url="ldap://...", check_method="SIMPLE_BIND",
       critical=True)

   # Исправление: укажите bind_dn и bind_password
   ldap_check("ldap", url="ldap://...", check_method="SIMPLE_BIND",
       bind_dn="cn=admin,dc=corp", bind_password="secret", critical=True)
   ```

2. `SEARCH` без `base_dn`:

   ```python
   # Ошибка: LDAP search requires base_dn
   ldap_check("ldap", url="ldap://...", check_method="SEARCH",
       critical=True)

   # Исправление: укажите base_dn
   ldap_check("ldap", url="ldap://...", check_method="SEARCH",
       base_dn="dc=example,dc=com", critical=True)
   ```

3. `start_tls` с `ldaps://`:

   ```python
   # Ошибка: start_tls and ldaps:// are incompatible
   ldap_check("ldap", url="ldaps://ldap.svc:636",
       start_tls=True, critical=True)

   # Исправление: используйте что-то одно
   ldap_check("ldap", url="ldaps://ldap.svc:636", critical=True)
   ldap_check("ldap", url="ldap://ldap.svc:389",
       start_tls=True, critical=True)
   ```

## Пользовательские метки не появляются

**Проверьте:**

1. Метки переданы как словарь:

   ```python
   postgres_check("db", url="...", critical=True,
       labels={"role": "primary"})
   ```

2. Имена меток валидны: `[a-zA-Z_][a-zA-Z0-9_]*`

3. Имена меток не используют зарезервированные: `name`, `group`, `dependency`,
   `type`, `host`, `port`, `critical`

## health() возвращает пустой словарь

**Проверьте:**

1. `start()` или `start_sync()` вызваны до `health()`
2. Прошёл хотя бы один интервал проверки
3. Зависимости зарегистрированы (не пустой `DependencyHealth("name", "group")`)

## Ошибки именования зависимостей

Имена должны соответствовать правилам:

- Длина: 1-63 символа
- Формат: `[a-z][a-z0-9-]*` (строчные буквы, цифры, дефисы)
- Должно начинаться с буквы

Допустимо: `postgres-main`, `redis-cache`, `auth-service`

Недопустимо: `Postgres`, `redis_cache`, `123-service`, `-invalid`

## См. также

- [Быстрый старт](getting-started.ru.md) — базовая настройка и первый пример
- [Конфигурация](configuration.ru.md) — все опции, значения по умолчанию и валидация
- [Чекеры](checkers.ru.md) — все 9 встроенных чекеров
- [Connection Pools](connection-pools.ru.md) — руководство по pool-интеграции
- [Метрики](metrics.ru.md) — справочник Prometheus-метрик
