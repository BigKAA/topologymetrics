*[English version](spring-boot.md)*

# Spring Boot интеграция

`dephealth-spring-boot-starter` обеспечивает авто-конфигурацию для
приложений Spring Boot 3.x. Стартер автоматически создает, настраивает,
запускает и останавливает экземпляр `DepHealth` на основе свойств
`application.yml`.

## Установка

### Maven

```xml
<dependency>
    <groupId>biz.kryukov.dev</groupId>
    <artifactId>dephealth-spring-boot-starter</artifactId>
    <version>0.8.0</version>
</dependency>
```

### Gradle

```groovy
implementation 'biz.kryukov.dev:dephealth-spring-boot-starter:0.8.0'
```

Стартер транзитивно подключает `dephealth-core`, поэтому добавлять его
отдельно не нужно.

## Авто-конфигурация

Стартер автоматически регистрирует следующие бины при запуске
приложения:

| Бин | Описание |
| --- | --- |
| `DepHealth` | Основной экземпляр для проверки зависимостей. Создается из свойств `dephealth.*`, если вы не определили свой `@Bean` |
| `DepHealthLifecycle` | Реализует Spring-интерфейс `Lifecycle` — запускает `DepHealth` при старте приложения и останавливает при завершении |
| `DepHealthIndicator` | Интегрируется в эндпоинт Spring Boot Actuator `/actuator/health` |
| `DependenciesEndpoint` | Кастомный actuator-эндпоинт `/actuator/dependencies`, возвращающий статусы зависимостей |

Авто-конфигурация условна:

- `@ConditionalOnClass(DepHealth.class)` — стартер активируется только
  при наличии `dephealth-core` в classpath.
- `@ConditionalOnMissingBean(DepHealth.class)` — если вы определили
  собственный бин `DepHealth`, авто-сконфигурированный пропускается.

## Свойства конфигурации

Все свойства используют префикс `dephealth`. Ниже приведен полный
пример YAML со всеми поддерживаемыми параметрами:

```yaml
dephealth:
  name: my-service
  group: my-team
  interval: 15s
  timeout: 5s
  dependencies:
    postgres-main:
      type: postgres
      url: ${DATABASE_URL}
      critical: true
      labels:
        role: primary
    redis-cache:
      type: redis
      url: ${REDIS_URL}
      critical: false
    auth-service:
      type: http
      url: http://auth.svc:8080
      health-path: /healthz
      critical: true
      http-bearer-token: ${API_TOKEN}
    user-service:
      type: grpc
      host: user.svc
      port: "9090"
      critical: false
    rabbitmq:
      type: amqp
      host: rabbitmq.svc
      port: "5672"
      amqp-username: user
      amqp-password: pass
      virtual-host: /
      critical: false
    kafka:
      type: kafka
      host: kafka.svc
      port: "9092"
      critical: false
    directory:
      type: ldap
      url: ldap://ldap.svc:389
      critical: true
      ldap-check-method: root_dse
```

## Справочник свойств

### Глобальные свойства

| Свойство | Тип | По умолчанию | Описание |
| --- | --- | --- | --- |
| `dephealth.name` | `String` | — | Имя приложения (обязательно). Запасной вариант — переменная окружения `DEPHEALTH_NAME` |
| `dephealth.group` | `String` | — | Группа приложения (обязательно). Запасной вариант — переменная окружения `DEPHEALTH_GROUP` |
| `dephealth.interval` | `Duration` | `15s` | Глобальный интервал проверки |
| `dephealth.timeout` | `Duration` | `5s` | Глобальный таймаут проверки |

### Свойства зависимостей

Каждая зависимость определяется в `dephealth.dependencies.<name>`.

#### Общие

| Свойство | Тип | По умолчанию | Описание |
| --- | --- | --- | --- |
| `.type` | `String` | — | Тип зависимости: `http`, `grpc`, `tcp`, `postgres`, `mysql`, `redis`, `amqp`, `kafka`, `ldap` (обязательно) |
| `.url` | `String` | — | URL подключения (альтернатива host/port) |
| `.host` | `String` | — | Имя хоста (альтернатива url) |
| `.port` | `String` | — | Порт (альтернатива url) |
| `.critical` | `boolean` | — | Является ли зависимость критичной (обязательно) |
| `.interval` | `Duration` | глобальный | Переопределение интервала проверки для конкретной зависимости |
| `.timeout` | `Duration` | глобальный | Переопределение таймаута проверки для конкретной зависимости |
| `.labels` | `Map<String, String>` | — | Пользовательские метки, добавляемые к метрикам Prometheus |
| `.tls` | `boolean` | `false` | Включить TLS для соединения |
| `.tls-skip-verify` | `boolean` | `false` | Пропустить проверку TLS-сертификата |

#### HTTP

| Свойство | Тип | По умолчанию | Описание |
| --- | --- | --- | --- |
| `.health-path` | `String` | `/` | Путь для HTTP-проверки |
| `.http-headers` | `Map<String, String>` | — | Пользовательские HTTP-заголовки |
| `.http-bearer-token` | `String` | — | Bearer-токен для заголовка Authorization |
| `.http-basic-username` | `String` | — | Имя пользователя HTTP Basic auth |
| `.http-basic-password` | `String` | — | Пароль HTTP Basic auth |

#### gRPC

| Свойство | Тип | По умолчанию | Описание |
| --- | --- | --- | --- |
| `.service-name` | `String` | `""` | Имя сервиса для gRPC health check |
| `.grpc-metadata` | `Map<String, String>` | — | Пользовательские gRPC metadata-заголовки |
| `.grpc-bearer-token` | `String` | — | Bearer-токен для gRPC call credentials |
| `.grpc-basic-username` | `String` | — | Имя пользователя gRPC Basic auth |
| `.grpc-basic-password` | `String` | — | Пароль gRPC Basic auth |

#### Базы данных (Postgres, MySQL)

| Свойство | Тип | По умолчанию | Описание |
| --- | --- | --- | --- |
| `.username` | `String` | — | Имя пользователя БД |
| `.password` | `String` | — | Пароль БД |
| `.database` | `String` | — | Имя базы данных |
| `.query` | `String` | `SELECT 1` | Пользовательский SQL-запрос для проверки |

#### Redis

| Свойство | Тип | По умолчанию | Описание |
| --- | --- | --- | --- |
| `.redis-password` | `String` | — | Пароль Redis |
| `.redis-db` | `int` | `0` | Номер базы данных Redis |

#### AMQP

| Свойство | Тип | По умолчанию | Описание |
| --- | --- | --- | --- |
| `.amqp-url` | `String` | — | AMQP URL подключения (альтернатива host/port) |
| `.amqp-username` | `String` | `guest` | Имя пользователя AMQP |
| `.amqp-password` | `String` | `guest` | Пароль AMQP |
| `.virtual-host` | `String` | `/` | Виртуальный хост AMQP |

#### LDAP

| Свойство | Тип | По умолчанию | Описание |
| --- | --- | --- | --- |
| `.ldap-check-method` | `String` | `root_dse` | Метод проверки: `root_dse`, `bind`, `search` |
| `.ldap-bind-dn` | `String` | — | Bind DN для методов `bind` или `search` |
| `.ldap-bind-password` | `String` | — | Пароль для bind |
| `.ldap-base-dn` | `String` | — | Base DN для метода `search` |
| `.ldap-search-filter` | `String` | `(objectClass=*)` | LDAP-фильтр поиска |
| `.ldap-search-scope` | `String` | `base` | Область поиска: `base`, `one`, `sub` |
| `.ldap-start-tls` | `boolean` | `false` | Использовать расширение STARTTLS |
| `.ldap-tls-skip-verify` | `boolean` | `false` | Пропустить проверку TLS-сертификата для LDAP |

## Actuator-эндпоинты

### `/actuator/dependencies`

Кастомный эндпоинт, возвращающий карту статусов всех зависимостей.

**Запрос:**

```bash
curl -s http://localhost:8080/actuator/dependencies | jq .
```

**Ответ:**

```json
{
  "postgres-main:pg.svc:5432": true,
  "redis-cache:redis.svc:6379": true,
  "auth-service:auth.svc:8080": true,
  "user-service:user.svc:9090": false,
  "rabbitmq:rabbitmq.svc:5672": true,
  "kafka:kafka.svc:9092": true,
  "directory:ldap.svc:389": true
}
```

### `/actuator/health`

`DepHealthIndicator` интегрируется в стандартный health-эндпоинт
Spring Boot. Индикатор сообщает `UP`, когда все **критичные**
зависимости здоровы, и `DOWN` в противном случае.

**Запрос:**

```bash
curl -s http://localhost:8080/actuator/health | jq .
```

**Ответ:**

```json
{
  "status": "UP",
  "components": {
    "dephealth": {
      "status": "UP",
      "details": {
        "postgres-main:pg.svc:5432": "UP",
        "redis-cache:redis.svc:6379": "UP",
        "auth-service:auth.svc:8080": "UP",
        "user-service:user.svc:9090": "DOWN",
        "rabbitmq:rabbitmq.svc:5672": "UP",
        "kafka:kafka.svc:9092": "UP",
        "directory:ldap.svc:389": "UP"
      }
    }
  }
}
```

> Примечание: `user-service` имеет статус DOWN, но общий статус — UP,
> поскольку эта зависимость не отмечена как `critical`.

### `/actuator/prometheus`

Стандартный эндпоинт метрик Prometheus. Метрики DepHealth экспортируются
автоматически через Micrometer `MeterRegistry`.

**Запрос:**

```bash
curl -s http://localhost:8080/actuator/prometheus | grep app_dependency
```

**Ответ:**

```text
app_dependency_health{name="my-service",group="my-team",dependency="postgres-main",type="postgres",host="pg.svc",port="5432",critical="yes"} 1
app_dependency_health{name="my-service",group="my-team",dependency="redis-cache",type="redis",host="redis.svc",port="6379",critical="no"} 1
app_dependency_latency_seconds_bucket{name="my-service",group="my-team",dependency="postgres-main",type="postgres",host="pg.svc",port="5432",critical="yes",le="0.01"} 42
```

## Настройка Actuator

Убедитесь, что необходимые эндпоинты открыты в `application.yml`:

```yaml
management:
  endpoints:
    web:
      exposure:
        include: health, prometheus, dependencies
```

Для отображения полных деталей health (включая компонент `dephealth`):

```yaml
management:
  endpoint:
    health:
      show-details: always
  endpoints:
    web:
      exposure:
        include: health, prometheus, dependencies
```

## Собственный бин DepHealth

Для переопределения авто-конфигурации — например, для интеграции с
существующим пулом соединений — определите собственный бин `DepHealth`:

```java
@Configuration
public class DepHealthConfig {

    @Bean
    public DepHealth depHealth(MeterRegistry registry, DataSource dataSource) {
        return DepHealth.builder("my-service", "my-team", registry)
            .dependency("postgres-main", DependencyType.POSTGRES, d -> d
                .dataSource(dataSource)
                .critical(true))
            .build();
    }
}
```

При наличии пользовательского `@Bean` стартер пропускает создание
собственного `DepHealth`, но по-прежнему регистрирует
`DepHealthLifecycle`, `DepHealthIndicator` и `DependenciesEndpoint`
вокруг вашего бина.

## Переменные окружения

Spring Boot нативно поддерживает подстановки `${VAR_NAME}` в YAML-файлах.
Используйте это для хранения чувствительных значений вне конфигурации:

```yaml
dephealth:
  name: ${DEPHEALTH_NAME:my-service}
  group: ${DEPHEALTH_GROUP:my-team}
  dependencies:
    postgres-main:
      type: postgres
      url: ${DATABASE_URL}
      username: ${DB_USERNAME}
      password: ${DB_PASSWORD}
      critical: true
    auth-service:
      type: http
      url: ${AUTH_SERVICE_URL:http://auth.svc:8080}
      http-bearer-token: ${API_TOKEN}
      critical: true
```

Синтаксис `${VAR:default}` задает значение по умолчанию, которое
используется при отсутствии переменной окружения.

## Смотрите также

- [Начало работы](getting-started.ru.md) — установка и первый пример
- [Конфигурация](configuration.ru.md) — все опции, значения по умолчанию и переменные окружения
- [Пулы соединений](connection-pools.ru.md) — интеграция с DataSource, JedisPool и LDAPConnection
- [Устранение неполадок](troubleshooting.ru.md) — типичные проблемы и решения
