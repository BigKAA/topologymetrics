*[English version](troubleshooting.md)*

# Устранение неполадок

Типичные проблемы и решения при использовании Java SDK dephealth.

## Пустые метрики / Метрики не экспортируются

**Симптом:** Эндпоинт `/metrics` или `/actuator/prometheus` не содержит
метрик `app_dependency_*`.

**Возможные причины:**

1. **`start()` не вызван.** Метрики регистрируются и обновляются только после
   вызова `dh.start()`. Убедитесь, что `start()` вызывается без ошибок.

   Для Spring Boot: проверьте, что `dephealth-spring-boot-starter` находится
   в classpath — он вызывает `start()` автоматически через `DepHealthLifecycle`.

2. **Неправильный эндпоинт Prometheus.** Для Spring Boot метрики доступны на
   `/actuator/prometheus`, а не на `/metrics`. Убедитесь, что эндпоинт открыт:

   ```yaml
   management:
     endpoints:
       web:
         exposure:
           include: health, prometheus, dependencies
   ```

3. **MeterRegistry не подключён.** Для программного API убедитесь, что один
   и тот же `MeterRegistry` передаётся в `DepHealth.builder()` и в обработчик
   метрик:

   ```java
   var registry = new PrometheusMeterRegistry(PrometheusConfig.DEFAULT);
   var dh = DepHealth.builder("my-service", "my-team", registry)
       // ...
       .build();
   dh.start();

   // Тот же registry для обработчика /metrics
   String metrics = registry.scrape();
   ```

## Все зависимости показывают 0 (unhealthy)

**Симптом:** `app_dependency_health` равен `0` для всех зависимостей.

**Возможные причины:**

1. **Сетевая доступность** — убедитесь, что целевые сервисы доступны из
   контейнера/пода сервиса.

2. **DNS-разрешение** — проверьте, что имена сервисов разрешаются корректно.

3. **Неправильный URL/host/port** — перепроверьте значения конфигурации.

4. **Тайм-аут слишком мал** — по умолчанию 5 сек. Увеличьте для медленных
   зависимостей:

   ```java
   .dependency("slow-db", DependencyType.POSTGRES, d -> d
       .url(System.getenv("DATABASE_URL"))
       .timeout(Duration.ofSeconds(10))
       .critical(true))
   ```

5. **Отладочное логирование** — включите отладку SDK:

   ```yaml
   # Spring Boot
   logging:
     level:
       biz.kryukov.dev.dephealth: DEBUG
   ```

## Высокая латентность проверок PostgreSQL/MySQL

**Симптом:** `app_dependency_latency_seconds` показывает высокие значения
(100 мс+) для проверок баз данных.

**Причина:** Standalone-режим создаёт новое JDBC-соединение при каждой
проверке. Это включает TCP-рукопожатие, согласование TLS и аутентификацию.

**Решение:** Используйте интеграцию с пулом соединений:

```java
// Вместо
.dependency("postgres-main", DependencyType.POSTGRES, d -> d
    .url("postgresql://user:pass@pg.svc:5432/mydb")
    .critical(true))

// Используйте существующий DataSource
.dependency("postgres-main", DependencyType.POSTGRES, d -> d
    .dataSource(dataSource)
    .critical(true))
```

Подробнее — в [Пулы соединений](connection-pools.ru.md).

## gRPC: ошибка DEADLINE_EXCEEDED

**Симптом:** gRPC-проверки завершаются по тайм-ауту или показывают
высокую латентность.

**Возможные причины:**

1. **gRPC-сервис недоступен** по указанному адресу.

2. **Сервис не реализует** `grpc.health.v1.Health/Check` — протокол
   gRPC Health Checking должен быть включён на целевом сервисе.

3. **Используйте `host()` + `port()`**, а не `url()` для gRPC:

   ```java
   .dependency("user-service", DependencyType.GRPC, d -> d
       .host("user.svc")
       .port("9090")
       .critical(true))
   ```

4. **Несовпадение TLS** — если сервис использует TLS, установите `.grpcTls(true)`.

5. **DNS-разрешение в Kubernetes** — используйте FQDN с точкой на конце:

   ```java
   .host("user-service.namespace.svc.cluster.local.")
   ```

## Ошибки Connection Refused

**Симптом:** `app_dependency_status{status="connection_error"}` равен `1`.

**Возможные причины:**

1. **Сервис не запущен** — убедитесь, что целевой сервис работает.

2. **Неправильный host или port** — перепроверьте значения.

3. **Network policies Kubernetes** — убедитесь, что трафик разрешён.

4. **Правила файрвола** — для не-Kubernetes окружений проверьте файрвол.

## Ошибки тайм-аута

**Симптом:** `app_dependency_status{status="timeout"}` равен `1`.

**Возможные причины:**

1. **Тайм-аут по умолчанию слишком мал.** По умолчанию 5 сек. Увеличьте:

   ```java
   // Глобально
   .timeout(Duration.ofSeconds(10))

   // Для конкретной зависимости
   .dependency("slow-service", DependencyType.HTTP, d -> d
       .url("http://slow.svc:8080")
       .timeout(Duration.ofSeconds(10))
       .critical(true))
   ```

2. **Сетевая латентность** — проверьте время отклика целевого сервиса.

3. **Перегрузка целевого сервиса** — сервис может быть слишком медленным.

## Неожиданные ошибки аутентификации

**Симптом:** `app_dependency_status{status="auth_error"}` равен `1`, хотя
учётные данные должны быть верными.

**Возможные причины:**

1. **Учётные данные не установлены или неверны**:

   ```java
   .httpBearerToken(System.getenv("API_TOKEN"))
   .grpcBearerToken(System.getenv("GRPC_TOKEN"))
   ```

2. **Токен истёк** — bearer-токены имеют ограниченный срок действия.

3. **Неправильный метод аутентификации** — некоторые сервисы ожидают
   Basic auth вместо Bearer.

4. **Учётные данные БД** — проверьте корректность в URL:

   ```java
   .url("postgresql://user:password@host:5432/dbname")
   ```

Подробнее — в [Аутентификация](authentication.ru.md).

## AMQP: ошибка подключения к RabbitMQ

**Симптом:** AMQP-чекер не может подключиться.

**Важно**: путь `/` в URL означает vhost `/` (не пустой).

```java
.dependency("rabbitmq", DependencyType.AMQP, d -> d
    .host("rabbitmq.svc")
    .port("5672")
    .amqpUsername("user")
    .amqpPassword("pass")
    .amqpVirtualHost("/")
    .critical(false))
```

## LDAP: ошибки конфигурации

**Симптом:** LDAP-чекер выбрасывает `ConfigurationException` при старте.

**Типичные причины:**

1. **`simple_bind` без учётных данных:**

   ```java
   // Неправильно — нет bindDN и bindPassword
   .ldapCheckMethod("simple_bind")

   // Правильно
   .ldapCheckMethod("simple_bind")
   .ldapBindDN("cn=monitor,dc=corp,dc=com")
   .ldapBindPassword("secret")
   ```

2. **`search` без baseDN:**

   ```java
   // Неправильно — нет baseDN
   .ldapCheckMethod("search")

   // Правильно
   .ldapCheckMethod("search")
   .ldapBaseDN("dc=example,dc=com")
   ```

3. **startTLS с ldaps://** — несовместимы:

   ```java
   // Неправильно — нельзя использовать оба
   .url("ldaps://ldap.svc:636")
   .ldapStartTLS(true)

   // Правильно — используйте одно из двух
   .url("ldaps://ldap.svc:636")        // неявный TLS
   // ИЛИ
   .url("ldap://ldap.svc:389")
   .ldapStartTLS(true)                 // обновление до TLS
   ```

## Произвольные метки не отображаются

**Симптом:** Метки, добавленные через `.label()`, не видны в метриках.

**Возможные причины:**

1. **Недопустимое имя метки.** Должно соответствовать `[a-zA-Z_][a-zA-Z0-9_]*`
   и не быть зарезервированным.

   Зарезервированные: `name`, `group`, `dependency`, `type`, `host`, `port`, `critical`.

2. **Несогласованные метки между зависимостями.** При использовании
   произвольных меток все эндпоинты должны использовать одинаковые
   имена меток.

## health() возвращает пустую Map

**Симптом:** `dh.health()` возвращает пустую карту сразу после `start()`.

**Причина:** Первая проверка ещё не завершилась. Есть начальная задержка
(по умолчанию 5 сек) перед первой проверкой.

**Решение:** Используйте `healthDetails()`. До завершения первой проверки
он возвращает записи с `healthy = null` и `status = "unknown"`:

```java
var details = dh.healthDetails();
details.forEach((key, ep) -> {
    if (ep.isHealthy() == null) {
        System.out.printf("%s: ещё не проверен%n", key);
    } else {
        System.out.printf("%s: healthy=%s%n", key, ep.isHealthy());
    }
});
```

## Spring Boot: метрики не на /actuator/prometheus

**Проверьте:**

1. Зависимость `spring-boot-starter-actuator` присутствует
2. `management.endpoints.web.exposure.include` включает `prometheus`
3. `dephealth-spring-boot-starter` в classpath
4. `io.micrometer:micrometer-registry-prometheus` в classpath

## См. также

- [Начало работы](getting-started.ru.md) — установка и базовая настройка
- [Конфигурация](configuration.ru.md) — все опции, значения по умолчанию и правила валидации
- [Чекеры](checkers.ru.md) — подробное руководство по всем 9 чекерам
- [Метрики](metrics.ru.md) — справочник метрик Prometheus и примеры PromQL
- [Аутентификация](authentication.ru.md) — опции аутентификации
- [Пулы соединений](connection-pools.ru.md) — интеграция с пулами соединений
- [Spring Boot интеграция](spring-boot.ru.md) — детали авто-конфигурации
