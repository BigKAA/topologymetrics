*[English version](migration.md)*

# Руководство по миграции

Инструкции по обновлению версий Java SDK.

## v0.6.0 → v0.8.0

### Новое: LDAP-чекер

v0.8.0 добавляет новый LDAP-чекер с полной поддержкой протокола.
Изменений в существующем API нет — это чисто аддитивная функция.

| Возможность | Описание |
| --- | --- |
| Методы проверки | `anonymous_bind`, `simple_bind`, `root_dse` (по умолчанию), `search` |
| Протоколы | `ldap://` (порт 389), `ldaps://` (порт 636), StartTLS |
| TLS-опции | `startTLS`, `tlsSkipVerify` |
| Режим пула | Принимает существующее `LDAPConnection` для проверок |

#### Базовое использование

```java
var dh = DepHealth.builder("my-service", "my-team", registry)
    .dependency("directory", DependencyType.LDAP, d -> d
        .url("ldap://ldap.svc:389")
        .critical(true))
    .build();
```

#### Методы проверки

```java
// Запрос RootDSE (по умолчанию)
.dependency("ldap", DependencyType.LDAP, d -> d
    .url("ldap://ldap.svc:389")
    .critical(true))

// Простая привязка с учётными данными
.dependency("ad", DependencyType.LDAP, d -> d
    .url("ldaps://ad.corp:636")
    .ldapCheckMethod("simple_bind")
    .ldapBindDN("cn=monitor,dc=corp,dc=com")
    .ldapBindPassword("secret")
    .critical(true))

// Поиск
.dependency("directory", DependencyType.LDAP, d -> d
    .url("ldap://ldap.svc:389")
    .ldapCheckMethod("search")
    .ldapBaseDN("dc=example,dc=com")
    .ldapSearchFilter("(objectClass=organizationalUnit)")
    .ldapSearchScope("one")
    .critical(true))
```

#### StartTLS

```java
.dependency("ldap-starttls", DependencyType.LDAP, d -> d
    .url("ldap://ldap.svc:389")
    .ldapStartTLS(true)
    .ldapTlsSkipVerify(true)
    .critical(true))
```

#### Режим пула

```java
import com.unboundid.ldap.sdk.LDAPConnection;

LDAPConnection conn = ...; // существующее соединение

.dependency("directory", DependencyType.LDAP, d -> d
    .ldapConnection(conn)
    .ldapCheckMethod("root_dse")
    .critical(true))
```

#### Spring Boot

```yaml
dephealth:
  dependencies:
    directory:
      type: ldap
      url: ldap://ldap.svc:389
      critical: true
      ldap-check-method: root_dse
```

#### Классификация ошибок

| Условие | Статус | Детализация |
| --- | --- | --- |
| Проверка успешна | `ok` | `ok` |
| LDAP код 49 (Invalid Credentials) | `auth_error` | `auth_error` |
| LDAP код 50 (Insufficient Access Rights) | `auth_error` | `auth_error` |
| Ошибка TLS/StartTLS | `tls_error` | `tls_error` |
| Соединение отклонено | `connection_error` | `connection_refused` |
| Ошибка DNS-разрешения | `dns_error` | `dns_error` |
| Тайм-аут | `timeout` | `timeout` |
| Сервер недоступен/занят | `unhealthy` | `unhealthy` |

#### Валидация конфигурации

| Условие | Ошибка |
| --- | --- |
| `simple_bind` без `bindDN` + `bindPassword` | Ошибка конфигурации |
| `search` без `baseDN` | Ошибка конфигурации |
| `startTLS=true` с `ldaps://` | Ошибка конфигурации (несовместимо) |

#### Обновление версии

```xml
<!-- v0.6.0 -->
<version>0.6.0</version>

<!-- v0.8.0 -->
<version>0.8.0</version>
```

---

## v0.5.0 → v0.6.0

### Новое: динамическое управление эндпоинтами

В v0.6.0 добавлены три метода для управления эндпоинтами в рантайме.
Изменений в существующем API нет — это чисто аддитивная функция.

| Метод | Описание |
| --- | --- |
| `addEndpoint` | Добавить мониторируемый эндпоинт после `start()` |
| `removeEndpoint` | Удалить эндпоинт (отменяет задачу, удаляет метрики) |
| `updateEndpoint` | Атомарно заменить эндпоинт новым |

Новое исключение `EndpointNotFoundException` (наследует `DepHealthException`)
выбрасывается методом `updateEndpoint`, если старый эндпоинт не найден.

```java
// После dh.start()...

// Добавить новый эндпоинт
dh.addEndpoint("api-backend", DependencyType.HTTP, true,
    new Endpoint("backend-2.svc", "8080"),
    HttpHealthChecker.builder().build());

// Удалить эндпоинт (идемпотентно)
dh.removeEndpoint("api-backend", "backend-2.svc", "8080");

// Заменить эндпоинт атомарно
dh.updateEndpoint("api-backend", "backend-1.svc", "8080",
    new Endpoint("backend-3.svc", "8080"),
    HttpHealthChecker.builder().build());
```

#### Ключевые особенности

- **Потокобезопасность** — все три метода синхронизированы.
- **Идемпотентность** — `addEndpoint` не делает ничего, если эндпоинт
  существует; `removeEndpoint` — если не найден.
- Динамические эндпоинты наследуют глобальный интервал и тайм-аут.
- `removeEndpoint` / `updateEndpoint` удаляют все метрики Prometheus.
- `updateEndpoint` выбрасывает `EndpointNotFoundException`.

#### Валидация

`addEndpoint` и `updateEndpoint` проверяют входные параметры:

- `depName` должен соответствовать `[a-z][a-z0-9-]*`, макс. 63 символа
- `depType` не должен быть null
- `ep.host()` и `ep.port()` не должны быть пустыми
- `ep.labels()` не должны содержать зарезервированные имена

Некорректные данные приводят к `ValidationException`.

#### Обработка ошибок

```java
try {
    dh.updateEndpoint("api", "old-host", "8080", newEp, checker);
} catch (EndpointNotFoundException e) {
    // старый эндпоинт не существует — используйте addEndpoint
} catch (IllegalStateException e) {
    // планировщик не запущен или уже остановлен
}
```

#### Внутренние изменения

- `CheckScheduler` хранит `ScheduledFuture` для каждого эндпоинта.
- Карта `states` изменена с `HashMap` на `ConcurrentHashMap`.
- `ScheduledThreadPoolExecutor` заменяет `ScheduledExecutorService`.
- `MetricsExporter.deleteMetrics()` удаляет все 4 семейства метрик.

#### Обновление версии

```xml
<!-- v0.5.0 -->
<version>0.5.0</version>

<!-- v0.6.0 -->
<version>0.6.0</version>
```

---

## v0.4.x → v0.5.0

### Обязательный параметр `group`

v0.5.0 добавляет обязательный параметр `group` (логическая группировка:
команда, подсистема, проект).

Программный API:

```java
// v0.4.x
DepHealth dh = DepHealth.builder("my-service", meterRegistry)
    .dependency(...)
    .build();

// v0.5.0
DepHealth dh = DepHealth.builder("my-service", "my-team", meterRegistry)
    .dependency(...)
    .build();
```

Spring Boot YAML:

```yaml
# v0.5.0 — добавьте group
dephealth:
  name: my-service
  group: my-team
  dependencies: ...
```

Альтернатива: переменная окружения `DEPHEALTH_GROUP` (API имеет приоритет).

Валидация: те же правила, что и для `name` — `[a-z][a-z0-9-]*`, 1-63 символа.

---

## v0.4.0 → v0.4.1

### Новое: healthDetails() API

В v0.4.1 добавлен метод `healthDetails()`, возвращающий детальный статус
каждого endpoint-а. Изменений в API нет — чисто аддитивная функция.

```java
Map<String, EndpointStatus> details = dh.healthDetails();

for (var entry : details.entrySet()) {
    EndpointStatus ep = entry.getValue();
    System.out.printf("%s: healthy=%s status=%s detail=%s latency=%s%n",
        entry.getKey(), ep.isHealthy(), ep.getStatus(), ep.getDetail(),
        ep.getLatencyMillis());
}
```

Поля `EndpointStatus`: `getName()`, `getType()`, `getHost()`, `getPort()`,
`isHealthy()` (`Boolean`, `null` = неизвестно), `getStatus()`, `getDetail()`,
`getLatency()`, `getLastCheckedAt()`, `isCritical()`, `getLabels()`.

До завершения первой проверки `isHealthy()` равен `null`, а `getStatus()` — `"unknown"`.

---

## v0.3.x → v0.4.0

### Новые метрики статуса (изменения кода не требуются)

v0.4.0 добавляет две новые автоматически экспортируемые метрики Prometheus:

| Метрика | Тип | Описание |
| --- | --- | --- |
| `app_dependency_status` | Gauge (enum) | Категория статуса: 8 серий на endpoint, ровно одна = 1 |
| `app_dependency_status_detail` | Gauge (info) | Детальная причина сбоя: напр. `http_503`, `auth_error` |

**Изменения кода не требуются** — SDK экспортирует эти метрики автоматически.

#### Влияние на хранилище

Каждый endpoint создаёт 9 дополнительных временных рядов. Для сервиса
с 5 endpoint-ами это добавляет 45 рядов.

#### Новые PromQL-запросы

```promql
# Категория статуса зависимости
app_dependency_status{dependency="postgres-main", status!=""} == 1

# Детальная причина сбоя
app_dependency_status_detail{dependency="postgres-main", detail!=""} == 1

# Алерт на ошибки аутентификации
app_dependency_status{status="auth_error"} == 1
```

---

## v0.1 → v0.2

### Изменения API

| v0.1 | v0.2 | Описание |
| --- | --- | --- |
| `DepHealth.builder(registry)` | `DepHealth.builder("my-service", registry)` | Обязательный аргумент `name` |
| `.critical(true)` (необязателен) | `.critical(true/false)` (обязателен) | Для каждой зависимости |
| нет | `.label("key", "value")` | Произвольные метки |
| `dephealth.name` (нет) | `dephealth.name: my-service` | В application.yml |

### Обязательные изменения

1. Добавьте `name` в builder:

```java
// v0.1
DepHealth dh = DepHealth.builder(meterRegistry)
    .dependency("postgres-main", DependencyType.POSTGRES, d -> d
        .url("postgres://user:pass@pg.svc:5432/mydb")
        .critical(true))
    .build();

// v0.2
DepHealth dh = DepHealth.builder("my-service", meterRegistry)
    .dependency("postgres-main", DependencyType.POSTGRES, d -> d
        .url("postgres://user:pass@pg.svc:5432/mydb")
        .critical(true))
    .build();
```

1. Укажите `.critical()` для каждой зависимости:

```java
// v0.1 — critical необязателен
.dependency("redis-cache", DependencyType.REDIS, d -> d
    .url("redis://redis.svc:6379"))

// v0.2 — critical обязателен
.dependency("redis-cache", DependencyType.REDIS, d -> d
    .url("redis://redis.svc:6379")
    .critical(false))
```

1. Обновите `application.yml` (Spring Boot):

```yaml
# v0.2
dephealth:
  name: my-service
  dependencies:
    redis-cache:
      type: redis
      url: ${REDIS_URL}
      critical: false
```

1. Обновите версию зависимости:

```xml
<version>0.2.2</version>
```

### Новые метки в метриках

```text
# v0.1
app_dependency_health{dependency="postgres-main",type="postgres",host="pg.svc",port="5432"} 1

# v0.2
app_dependency_health{name="my-service",dependency="postgres-main",type="postgres",host="pg.svc",port="5432",critical="yes"} 1
```

Обновите PromQL-запросы и дашборды Grafana, добавив метки `name` и `critical`.

## См. также

- [Начало работы](getting-started.ru.md) — установка и настройка
- [Чекеры](checkers.ru.md) — детали LDAP-чекера
- [Конфигурация](configuration.ru.md) — все опции и значения по умолчанию
- [API Reference](api-reference.ru.md) — полный публичный API
