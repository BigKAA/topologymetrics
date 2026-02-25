*[English version](README.md)*

# dephealth

SDK для мониторинга зависимостей микросервисов через метрики Prometheus.

## Возможности

- Автоматическая проверка здоровья зависимостей (PostgreSQL, MySQL, Redis, RabbitMQ, Kafka, HTTP, gRPC, TCP, LDAP)
- Экспорт метрик Prometheus: `app_dependency_health` (Gauge 0/1), `app_dependency_latency_seconds` (Histogram), `app_dependency_status` (enum), `app_dependency_status_detail` (info)
- Java 21 LTS, Maven multi-module проект
- Spring Boot starter с auto-configuration и интеграцией actuator
- Поддержка connection pool (предпочтительно) и автономных проверок

## Установка

### Maven

```xml
<dependency>
    <groupId>biz.kryukov.dev</groupId>
    <artifactId>dephealth-core</artifactId>
    <version>0.8.0</version>
</dependency>
```

### Spring Boot

```xml
<dependency>
    <groupId>biz.kryukov.dev</groupId>
    <artifactId>dephealth-spring-boot-starter</artifactId>
    <version>0.8.0</version>
</dependency>
```

## Быстрый старт

```java
import biz.kryukov.dev.dephealth.DepHealth;
import io.micrometer.prometheus.PrometheusConfig;
import io.micrometer.prometheus.PrometheusMeterRegistry;

var registry = new PrometheusMeterRegistry(PrometheusConfig.DEFAULT);

var dh = DepHealth.builder("my-service", "my-team", registry)
    .postgres("db", "postgresql://user:pass@localhost:5432/mydb", true)
    .redis("cache", "redis://localhost:6379", false)
    .build();

dh.start();
// Метрики доступны через Micrometer registry
dh.stop();
```

## Динамические эндпоинты

Добавление, удаление и замена мониторируемых эндпоинтов в рантайме на
работающем экземпляре (v0.6.0+):

```java
import biz.kryukov.dev.dephealth.model.DependencyType;
import biz.kryukov.dev.dephealth.model.Endpoint;
import biz.kryukov.dev.dephealth.checks.HttpHealthChecker;

// После dh.start()...

// Добавить новый эндпоинт
dh.addEndpoint("api-backend", DependencyType.HTTP, true,
    new Endpoint("backend-2.svc", "8080"),
    HttpHealthChecker.builder().build());

// Удалить эндпоинт (отменяет задачу, удаляет метрики)
dh.removeEndpoint("api-backend", "backend-2.svc", "8080");

// Заменить эндпоинт атомарно
dh.updateEndpoint("api-backend", "backend-1.svc", "8080",
    new Endpoint("backend-3.svc", "8080"),
    HttpHealthChecker.builder().build());
```

Подробности в [руководстве по миграции](docs/migration.ru.md#v050--v060).

## Детализация здоровья

```java
var details = dh.healthDetails();
details.forEach((key, ep) ->
    System.out.printf("%s: healthy=%s status=%s latency=%.1fms%n",
        key, ep.isHealthy(), ep.getStatus(), ep.getLatencyMillis()));
```

## Поддерживаемые зависимости

| Тип | Формат URL |
| --- | --- |
| PostgreSQL | `postgresql://user:pass@host:5432/db` |
| MySQL | `mysql://user:pass@host:3306/db` |
| Redis | `redis://host:6379` |
| RabbitMQ | `amqp://user:pass@host:5672/vhost` |
| Kafka | `kafka://host1:9092,host2:9092` |
| HTTP | `http://host:8080/health` |
| gRPC | через `host()` + `port()` |
| TCP | `tcp://host:port` |
| LDAP | `ldap://host:389` или `ldaps://host:636` |

## LDAP-чекер

LDAP-чекер поддерживает четыре метода проверки и несколько режимов TLS:

```java
import biz.kryukov.dev.dephealth.checks.LdapHealthChecker;
import biz.kryukov.dev.dephealth.checks.LdapHealthChecker.CheckMethod;
import biz.kryukov.dev.dephealth.checks.LdapHealthChecker.LdapSearchScope;

// Запрос RootDSE (по умолчанию)
var checker = LdapHealthChecker.builder()
    .checkMethod(CheckMethod.ROOT_DSE)
    .build();

// Простая привязка с учётными данными
var checker = LdapHealthChecker.builder()
    .checkMethod(CheckMethod.SIMPLE_BIND)
    .bindDN("cn=monitor,dc=corp,dc=com")
    .bindPassword("secret")
    .useTLS(true)
    .build();

// Поиск с StartTLS
var checker = LdapHealthChecker.builder()
    .checkMethod(CheckMethod.SEARCH)
    .baseDN("dc=example,dc=com")
    .searchFilter("(objectClass=organizationalUnit)")
    .searchScope(LdapSearchScope.ONE)
    .startTLS(true)
    .build();
```

Методы проверки: `ANONYMOUS_BIND`, `SIMPLE_BIND`, `ROOT_DSE` (по умолчанию), `SEARCH`.

## Аутентификация

HTTP и gRPC чекеры поддерживают Bearer token, Basic Auth и пользовательские заголовки/метаданные:

```java
dh.http("secure-api", "http://api.svc:8080", true,
    HttpHealthChecker.builder().bearerToken("eyJhbG...").build());

dh.grpc("grpc-backend", "backend.svc", "9090", true,
    GrpcHealthChecker.builder().bearerToken("eyJhbG...").build());
```

Все опции описаны в [руководстве по аутентификации](docs/authentication.ru.md).

## Документация

Полная документация доступна в директории [docs/](docs/README.md).

## Лицензия

Apache License 2.0 — см. [LICENSE](https://github.com/BigKAA/topologymetrics/blob/master/LICENSE).
