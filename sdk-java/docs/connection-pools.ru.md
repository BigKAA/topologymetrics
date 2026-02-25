*[English version](connection-pools.md)*

# Интеграция с пулами соединений

dephealth поддерживает два режима проверки зависимостей:

- **Standalone-режим** — SDK создаёт новое соединение для каждой проверки
- **Pool-режим** — SDK использует существующий пул соединений сервиса

Pool-режим предпочтителен, так как отражает реальную способность сервиса
работать с зависимостью. Если пул исчерпан, проверка это обнаружит.

## Standalone vs Pool

| Аспект | Standalone | Pool |
| --- | --- | --- |
| Соединение | Новое на каждую проверку | Из существующего пула |
| Отражает реальное здоровье | Частично | Да |
| Настройка | Простая — только URL | Требует передачи объекта пула |
| Внешние зависимости | Нет (использует драйвер чекера) | Драйвер приложения |
| Обнаруживает исчерпание пула | Нет | Да |

## PostgreSQL через DataSource

Передайте существующий `DataSource` (HikariCP, Tomcat JDBC и т.д.)
в builder зависимости:

```java
import javax.sql.DataSource;

DataSource dataSource = ...; // существующий пул из приложения

var dh = DepHealth.builder("my-service", "my-team", registry)
    .dependency("postgres-main", DependencyType.POSTGRES, d -> d
        .dataSource(dataSource)
        .critical(true))
    .build();
```

При использовании `dataSource()` хост и порт извлекаются из метаданных
DataSource. Можно указать явно при необходимости:

```java
.dependency("postgres-main", DependencyType.POSTGRES, d -> d
    .dataSource(dataSource)
    .host("pg.svc")
    .port("5432")
    .critical(true))
```

Чекер выполняет `SELECT 1` (по умолчанию) на соединении из пула.
Свой запрос через `.dbQuery()`:

```java
.dependency("postgres-main", DependencyType.POSTGRES, d -> d
    .dataSource(dataSource)
    .dbQuery("SELECT 1 FROM pg_stat_activity LIMIT 1")
    .critical(true))
```

## MySQL через DataSource

Аналогично PostgreSQL — передайте `DataSource`:

```java
DataSource dataSource = ...; // HikariCP, Tomcat JDBC и т.д.

var dh = DepHealth.builder("my-service", "my-team", registry)
    .dependency("mysql-main", DependencyType.MYSQL, d -> d
        .dataSource(dataSource)
        .critical(true))
    .build();
```

## Redis через JedisPool

Передайте существующий `JedisPool`:

```java
import redis.clients.jedis.JedisPool;

JedisPool jedisPool = ...; // существующий пул из приложения

var dh = DepHealth.builder("my-service", "my-team", registry)
    .dependency("redis-cache", DependencyType.REDIS, d -> d
        .jedisPool(jedisPool)
        .critical(false))
    .build();
```

Чекер берёт соединение `Jedis` из пула, отправляет `PING` и возвращает
его. Хост и порт извлекаются из конфигурации пула.

## LDAP через LDAPConnection

Передайте существующее соединение UnboundID `LDAPConnection`:

```java
import com.unboundid.ldap.sdk.LDAPConnection;

LDAPConnection conn = ...; // существующее соединение

var dh = DepHealth.builder("my-service", "my-team", registry)
    .dependency("directory", DependencyType.LDAP, d -> d
        .ldapConnection(conn)
        .ldapCheckMethod("root_dse")
        .critical(true))
    .build();
```

При использовании существующего соединения настройки TLS (startTLS, useTLS)
управляются самим соединением, а не чекером.

## Прямое создание чекеров с пулом

Для большего контроля можно создать чекеры напрямую с объектами пулов
и зарегистрировать их через `addEndpoint()`:

### PostgreSQL

```java
import biz.kryukov.dev.dephealth.checks.PostgresHealthChecker;
import biz.kryukov.dev.dephealth.model.Endpoint;

DataSource dataSource = ...;

var checker = PostgresHealthChecker.builder()
    .dataSource(dataSource)
    .query("SELECT 1")
    .build();

// После dh.start()
dh.addEndpoint("postgres-main", DependencyType.POSTGRES, true,
    new Endpoint("pg.svc", "5432"),
    checker);
```

### Redis

```java
import biz.kryukov.dev.dephealth.checks.RedisHealthChecker;

JedisPool jedisPool = ...;

var checker = RedisHealthChecker.builder()
    .jedisPool(jedisPool)
    .build();

dh.addEndpoint("redis-cache", DependencyType.REDIS, false,
    new Endpoint("redis.svc", "6379"),
    checker);
```

## Standalone vs Pool: когда что использовать

| Сценарий | Рекомендация |
| --- | --- |
| Стандартная настройка, один пул на зависимость | Pool через `dataSource()` / `jedisPool()` |
| Нет существующего пула (внешние сервисы) | Standalone через `url()` |
| HTTP и gRPC сервисы | Только standalone (пул не нужен) |
| Свой запрос проверки | Pool с `.dbQuery()` |
| LDAP с управляемым жизненным циклом соединения | Pool через `.ldapConnection()` |

## Полный пример: смешанные режимы

```java
import biz.kryukov.dev.dephealth.DepHealth;
import biz.kryukov.dev.dephealth.model.DependencyType;
import io.micrometer.prometheus.PrometheusConfig;
import io.micrometer.prometheus.PrometheusMeterRegistry;
import javax.sql.DataSource;
import redis.clients.jedis.JedisPool;

public class Main {
    public static void main(String[] args) {
        var registry = new PrometheusMeterRegistry(PrometheusConfig.DEFAULT);

        // Существующие пулы соединений
        DataSource dataSource = ...; // HikariCP
        JedisPool jedisPool = ...;

        var dh = DepHealth.builder("my-service", "my-team", registry)
            // Pool-режим — PostgreSQL
            .dependency("postgres-main", DependencyType.POSTGRES, d -> d
                .dataSource(dataSource)
                .critical(true))

            // Pool-режим — Redis
            .dependency("redis-cache", DependencyType.REDIS, d -> d
                .jedisPool(jedisPool)
                .critical(false))

            // Standalone-режим — HTTP (пул не нужен)
            .dependency("payment-api", DependencyType.HTTP, d -> d
                .url("http://payment.svc:8080")
                .critical(true))

            .build();

        dh.start();
        Runtime.getRuntime().addShutdownHook(new Thread(dh::stop));
    }
}
```

## Интеграция с пулами в Spring Boot

При использовании Spring Boot можно определить свой бин `DepHealth`,
использующий авто-сконфигурированные пулы соединений:

```java
@Configuration
public class DepHealthConfig {
    @Bean
    public DepHealth depHealth(MeterRegistry registry, DataSource dataSource) {
        return DepHealth.builder("my-service", "my-team", registry)
            .dependency("postgres-main", DependencyType.POSTGRES, d -> d
                .dataSource(dataSource)
                .critical(true))
            .dependency("auth-service", DependencyType.HTTP, d -> d
                .url("http://auth.svc:8080")
                .critical(true))
            .build();
    }
}
```

Это перекрывает YAML-конфигурацию, но lifecycle management и actuator
integration из стартера продолжают работать.

## См. также

- [Чекеры](checkers.ru.md) — все детали чекеров включая опции пулов
- [Spring Boot интеграция](spring-boot.ru.md) — авто-конфигурация с пулами
- [Конфигурация](configuration.ru.md) — опции DataSource и JedisPool
- [API Reference](api-reference.ru.md) — справочник методов builder
- [Устранение неполадок](troubleshooting.ru.md) — типичные проблемы и решения
