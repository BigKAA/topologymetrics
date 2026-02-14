*[English version](metric-contract.md)*

# Контракт метрик

> Версия спецификации: **3.0-draft**
>
> Этот документ является единым источником правды для формата метрик,
> экспортируемых всеми SDK dephealth. Все реализации обязаны следовать этому контракту.
> Соответствие проверяется conformance-тестами.

---

## 1. Общие принципы

- Все метрики экспортируются в формате **Prometheus text exposition format**
  (совместимом с OpenMetrics).
- Endpoint для метрик: `GET /metrics` (или путь, настроенный разработчиком).
- Префикс всех метрик: `app_dependency_`.
- Имена метрик и меток используют только строчные буквы, цифры и символ `_`
  (согласно [Prometheus naming conventions](https://prometheus.io/docs/practices/naming/)).

---

## 2. Метрика здоровья: `app_dependency_health`

### 2.1. Описание

Gauge-метрика, отражающая текущее состояние доступности зависимости.

### 2.2. Свойства

| Свойство | Значение |
| --- | --- |
| Имя | `app_dependency_health` |
| Тип | Gauge |
| Допустимые значения | `1` (доступна), `0` (недоступна) |
| Единица измерения | безразмерная |

### 2.3. Обязательные метки

| Метка | Описание | Правила формирования | Пример |
| --- | --- | --- | --- |
| `name` | Уникальное имя приложения, экспортирующего метрики | Строчные буквы, цифры, `-`. Длина: 1-63 символа. Формат: `[a-z][a-z0-9-]*` | `order-api` |
| `dependency` | Логическое имя зависимости, задаётся разработчиком. Для сервисов с dephealth SDK значение должно совпадать с `name` целевого сервиса | Строчные буквы, цифры, `-`. Длина: 1-63 символа. Формат: `[a-z][a-z0-9-]*` | `payment-api` |
| `type` | Тип соединения / протокол | Одно из: `http`, `grpc`, `tcp`, `postgres`, `mysql`, `redis`, `amqp`, `kafka` | `postgres` |
| `host` | Адрес endpoint-а (hostname или IP) | Как есть из конфигурации. IPv6 без квадратных скобок | `pg-master.db.svc.cluster.local` |
| `port` | Порт endpoint-а | Строка с числом 1-65535. Если порт не указан, используется порт по умолчанию для данного типа | `5432` |
| `critical` | Критичность зависимости для работы приложения | Одно из: `yes` (приложение не работает без зависимости), `no` (деградация допустима). Обязателен, без значения по умолчанию | `yes` |

### 2.4. Произвольные метки (custom labels)

Разработчик может добавлять произвольные метки через `WithLabel(key, value)`.

**Правила**:

- Имя метки: формат `[a-zA-Z_][a-zA-Z0-9_]*` (Prometheus naming conventions).
- Запрещено переопределять обязательные метки: `name`, `dependency`, `type`,
  `host`, `port`, `critical`. При попытке — ошибка конфигурации.
- Если метка не указана, она **не включается** в метрику
  (а не выводится с пустым значением).

**Примеры использования**:

| Метка | Описание | Пример |
| --- | --- | --- |
| `role` | Роль экземпляра в кластере | `primary`, `replica` |
| `shard` | Идентификатор шарда | `shard-01`, `0` |
| `vhost` | Virtual host (для AMQP) | `/`, `production` |
| `env` | Окружение | `production`, `staging` |

### 2.5. Начальное значение

До завершения первой проверки (в период `initialDelay` + первый цикл)
метрика **не экспортируется**. После первой успешной или неуспешной проверки
метрика появляется со значением `1` или `0` соответственно.

**Обоснование**: отсутствие метрики вместо произвольного начального значения
позволяет алертам корректно обрабатывать запуск сервиса через `absent()`.

---

## 3. Метрика латентности: `app_dependency_latency_seconds`

### 3.1. Описание

Histogram-метрика, фиксирующая время выполнения каждой проверки здоровья.

### 3.2. Свойства

| Свойство | Значение |
| --- | --- |
| Имя | `app_dependency_latency_seconds` |
| Тип | Histogram |
| Единица измерения | секунды |
| Бакеты | `0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0, 5.0` |

### 3.3. Метки

Идентичны `app_dependency_health` (обязательные и опциональные).

### 3.4. Что измеряется

Время от начала вызова `HealthChecker.Check()` до получения результата
(успех или ошибка). Включает:

- Установку соединения (если автономный режим)
- Выполнение проверки (SQL-запрос, HTTP-запрос и т.д.)
- Получение ответа

Не включает:

- Время ожидания в очереди планировщика
- Время обработки результата (обновление метрик)

### 3.5. Поведение при ошибке

Латентность записывается **всегда** — как при успешной, так и при неуспешной
проверке. Таймаут приводит к записи значения, равного настроенному `timeout`
(или фактическому времени до срабатывания таймаута).

### 3.6. Начальное значение

Histogram появляется после первой проверки (одновременно с `app_dependency_health`).

---

## 4. Формат вывода `/metrics`

### 4.1. Prometheus text exposition format

SDK экспортирует метрики в стандартном формате:

```text
# HELP app_dependency_health Health status of a dependency (1 = healthy, 0 = unhealthy)
# TYPE app_dependency_health gauge
app_dependency_health{name="order-api",dependency="postgres-main",type="postgres",host="pg-master.db.svc",port="5432",critical="yes"} 1
app_dependency_health{name="order-api",dependency="redis-cache",type="redis",host="redis-0.cache.svc",port="6379",critical="no"} 1
app_dependency_health{name="order-api",dependency="payment-api",type="http",host="payment-svc.payments.svc",port="8080",critical="yes"} 0

# HELP app_dependency_latency_seconds Latency of dependency health check in seconds
# TYPE app_dependency_latency_seconds histogram
app_dependency_latency_seconds_bucket{name="order-api",dependency="postgres-main",type="postgres",host="pg-master.db.svc",port="5432",critical="yes",le="0.001"} 0
app_dependency_latency_seconds_bucket{name="order-api",dependency="postgres-main",type="postgres",host="pg-master.db.svc",port="5432",critical="yes",le="0.005"} 8
app_dependency_latency_seconds_bucket{name="order-api",dependency="postgres-main",type="postgres",host="pg-master.db.svc",port="5432",critical="yes",le="0.01"} 15
app_dependency_latency_seconds_bucket{name="order-api",dependency="postgres-main",type="postgres",host="pg-master.db.svc",port="5432",critical="yes",le="0.05"} 20
app_dependency_latency_seconds_bucket{name="order-api",dependency="postgres-main",type="postgres",host="pg-master.db.svc",port="5432",critical="yes",le="0.1"} 20
app_dependency_latency_seconds_bucket{name="order-api",dependency="postgres-main",type="postgres",host="pg-master.db.svc",port="5432",critical="yes",le="0.5"} 20
app_dependency_latency_seconds_bucket{name="order-api",dependency="postgres-main",type="postgres",host="pg-master.db.svc",port="5432",critical="yes",le="1"} 20
app_dependency_latency_seconds_bucket{name="order-api",dependency="postgres-main",type="postgres",host="pg-master.db.svc",port="5432",critical="yes",le="5"} 20
app_dependency_latency_seconds_bucket{name="order-api",dependency="postgres-main",type="postgres",host="pg-master.db.svc",port="5432",critical="yes",le="+Inf"} 20
app_dependency_latency_seconds_sum{name="order-api",dependency="postgres-main",type="postgres",host="pg-master.db.svc",port="5432",critical="yes"} 0.085
app_dependency_latency_seconds_count{name="order-api",dependency="postgres-main",type="postgres",host="pg-master.db.svc",port="5432",critical="yes"} 20
```

### 4.2. Требования к формату

- Строки `# HELP` и `# TYPE` обязательны для каждой метрики.
- Текст `# HELP` фиксирован (см. примеры выше) и не должен отличаться между SDK.
- Порядок меток: `name`, `dependency`, `type`, `host`, `port`, `critical`,
  затем произвольные в алфавитном порядке.
- Значения меток экранируются согласно Prometheus exposition format:
  символы `\`, `"`, `\n` заменяются на `\\`, `\"`, `\n`.

---

## 5. Поведение при множественных endpoint-ах

Одна зависимость может иметь несколько endpoint-ов (реплики БД, ноды кластера).

### 5.1. Правило: одна метрика на endpoint

Каждый endpoint порождает **отдельную** серию метрик. Агрегация не производится.

**Пример**: PostgreSQL с primary и replica:

```text
app_dependency_health{name="order-api",dependency="postgres-main",type="postgres",host="pg-primary.db.svc",port="5432",critical="yes",role="primary"} 1
app_dependency_health{name="order-api",dependency="postgres-main",type="postgres",host="pg-replica.db.svc",port="5432",critical="yes",role="replica"} 1

app_dependency_latency_seconds_bucket{name="order-api",dependency="postgres-main",type="postgres",host="pg-primary.db.svc",port="5432",critical="yes",role="primary",le="0.005"} 10
app_dependency_latency_seconds_bucket{name="order-api",dependency="postgres-main",type="postgres",host="pg-replica.db.svc",port="5432",critical="yes",role="replica",le="0.005"} 8
```

### 5.2. Обоснование

- Позволяет точно определить, какой именно endpoint недоступен.
- Алертинг может быть настроен на уровне отдельных endpoint-ов
  (например, `DependencyDegraded` при partial failure).
- Агрегация при необходимости выполняется на уровне PromQL:
  `min by (name, dependency) (app_dependency_health{dependency="postgres-main"})`.

### 5.3. Kafka: несколько брокеров

Для Kafka каждый брокер является отдельным endpoint-ом:

```text
app_dependency_health{name="order-api",dependency="kafka-main",type="kafka",host="kafka-0.kafka.svc",port="9092",critical="yes"} 1
app_dependency_health{name="order-api",dependency="kafka-main",type="kafka",host="kafka-1.kafka.svc",port="9092",critical="yes"} 1
app_dependency_health{name="order-api",dependency="kafka-main",type="kafka",host="kafka-2.kafka.svc",port="9092",critical="yes"} 0
```

---

## 6. Примеры типовых конфигураций

### 6.1. Минимальная конфигурация (один сервис, одна зависимость)

```text
# HELP app_dependency_health Health status of a dependency (1 = healthy, 0 = unhealthy)
# TYPE app_dependency_health gauge
app_dependency_health{name="my-service",dependency="redis-cache",type="redis",host="redis.default.svc",port="6379",critical="no"} 1

# HELP app_dependency_latency_seconds Latency of dependency health check in seconds
# TYPE app_dependency_latency_seconds histogram
app_dependency_latency_seconds_bucket{name="my-service",dependency="redis-cache",type="redis",host="redis.default.svc",port="6379",critical="no",le="0.001"} 5
app_dependency_latency_seconds_bucket{name="my-service",dependency="redis-cache",type="redis",host="redis.default.svc",port="6379",critical="no",le="0.005"} 10
app_dependency_latency_seconds_bucket{name="my-service",dependency="redis-cache",type="redis",host="redis.default.svc",port="6379",critical="no",le="0.01"} 10
app_dependency_latency_seconds_bucket{name="my-service",dependency="redis-cache",type="redis",host="redis.default.svc",port="6379",critical="no",le="0.05"} 10
app_dependency_latency_seconds_bucket{name="my-service",dependency="redis-cache",type="redis",host="redis.default.svc",port="6379",critical="no",le="0.1"} 10
app_dependency_latency_seconds_bucket{name="my-service",dependency="redis-cache",type="redis",host="redis.default.svc",port="6379",critical="no",le="0.5"} 10
app_dependency_latency_seconds_bucket{name="my-service",dependency="redis-cache",type="redis",host="redis.default.svc",port="6379",critical="no",le="1"} 10
app_dependency_latency_seconds_bucket{name="my-service",dependency="redis-cache",type="redis",host="redis.default.svc",port="6379",critical="no",le="5"} 10
app_dependency_latency_seconds_bucket{name="my-service",dependency="redis-cache",type="redis",host="redis.default.svc",port="6379",critical="no",le="+Inf"} 10
app_dependency_latency_seconds_sum{name="my-service",dependency="redis-cache",type="redis",host="redis.default.svc",port="6379",critical="no"} 0.025
app_dependency_latency_seconds_count{name="my-service",dependency="redis-cache",type="redis",host="redis.default.svc",port="6379",critical="no"} 10
```

### 6.2. Типичный микросервис (несколько зависимостей разных типов)

```text
# HELP app_dependency_health Health status of a dependency (1 = healthy, 0 = unhealthy)
# TYPE app_dependency_health gauge
app_dependency_health{name="order-api",dependency="postgres-main",type="postgres",host="pg.db.svc",port="5432",critical="yes"} 1
app_dependency_health{name="order-api",dependency="redis-cache",type="redis",host="redis.cache.svc",port="6379",critical="no"} 1
app_dependency_health{name="order-api",dependency="payment-api",type="http",host="payment.payments.svc",port="8080",critical="yes"} 1
app_dependency_health{name="order-api",dependency="auth-api",type="grpc",host="auth.auth.svc",port="9090",critical="yes"} 0
app_dependency_health{name="order-api",dependency="rabbitmq",type="amqp",host="rabbit.mq.svc",port="5672",critical="no"} 1
```

### 6.3. Сервис с AMQP и custom labels

```text
app_dependency_health{name="order-api",dependency="rabbitmq-orders",type="amqp",host="rabbit.mq.svc",port="5672",critical="yes",vhost="orders"} 1
app_dependency_health{name="order-api",dependency="rabbitmq-notifications",type="amqp",host="rabbit.mq.svc",port="5672",critical="no",vhost="notifications"} 1
```

### 6.4. Сервис в состоянии деградации (partial failure)

```text
app_dependency_health{name="order-api",dependency="postgres-main",type="postgres",host="pg-primary.db.svc",port="5432",critical="yes",role="primary"} 1
app_dependency_health{name="order-api",dependency="postgres-main",type="postgres",host="pg-replica-1.db.svc",port="5432",critical="yes",role="replica"} 0
app_dependency_health{name="order-api",dependency="postgres-main",type="postgres",host="pg-replica-2.db.svc",port="5432",critical="yes",role="replica"} 1
```

---

## 7. Полезные PromQL-запросы

Для справки: типичные запросы, которые будут использоваться в Grafana и алертах.

```promql
# Все нездоровые зависимости
app_dependency_health == 0

# Нездоровые зависимости конкретного сервиса (по name)
app_dependency_health{name="order-api"} == 0

# Все нездоровые критичные зависимости
app_dependency_health{critical="yes"} == 0

# Агрегированное здоровье зависимости (хотя бы один endpoint down)
min by (name, dependency) (app_dependency_health) == 0

# P99 латентность проверок за 5 минут
histogram_quantile(0.99, rate(app_dependency_latency_seconds_bucket[5m]))

# Средняя латентность по зависимости
rate(app_dependency_latency_seconds_sum[5m]) / rate(app_dependency_latency_seconds_count[5m])

# Зависимости, которые "мигают" (flapping) — частые переключения
changes(app_dependency_health[15m]) > 4

# Граф зависимостей: все рёбра (name -> dependency)
group by (name, dependency, type, critical) (app_dependency_health)

# Все сервисы, от которых зависит order-api
app_dependency_health{name="order-api"}

# Все сервисы, которые зависят от payment-api
app_dependency_health{dependency="payment-api"}
```

---

## 8. Метрика статуса: `app_dependency_status`

### 8.1. Описание

Gauge-метрика (enum-паттерн), отражающая **категорию** результата последней проверки.
Для каждого endpoint-а всегда экспортируются **все 8 значений** метки `status`.
Ровно одно из них = `1`, остальные 7 = `0`.
Это исключает series churn при смене состояния.

### 8.2. Свойства

| Свойство | Значение |
| --- | --- |
| Имя | `app_dependency_status` |
| Тип | Gauge |
| Текст HELP | `Category of the last check result` |
| Допустимые значения | `1` (активный статус), `0` (неактивный статус) |

### 8.3. Значения статуса

| Значение | Описание | Типичные ситуации |
| --- | --- | --- |
| `ok` | Проверка успешна, зависимость доступна | HTTP 2xx, gRPC SERVING, TCP connected, SQL SELECT 1 OK, Redis PONG |
| `timeout` | Превышен таймаут проверки | Connection timeout, query timeout, gRPC DEADLINE_EXCEEDED, context deadline exceeded |
| `connection_error` | Невозможно установить TCP-соединение | Connection refused (RST), host unreachable, network unreachable, port not listening |
| `dns_error` | Ошибка разрешения DNS-имени | Hostname not found, DNS lookup failure, NXDOMAIN |
| `auth_error` | Ошибка аутентификации/авторизации | Wrong DB credentials, Redis NOAUTH/WRONGPASS, AMQP 403 Access Refused |
| `tls_error` | Ошибка TLS/SSL | Certificate validation failed, TLS handshake error, expired certificate |
| `unhealthy` | Сервис ответил, но сообщает о нездоровом состоянии | HTTP 4xx/5xx, gRPC NOT_SERVING, Kafka no brokers, Redis non-PONG, AMQP connection not open |
| `error` | Прочие неклассифицированные ошибки | Unexpected exceptions, panics, pool exhaustion, query syntax error |

### 8.4. Метки

Те же обязательные и произвольные метки, что и у `app_dependency_health` (разделы 2.3, 2.4),
плюс метка `status` в конце.

Порядок меток: `name`, `dependency`, `type`, `host`, `port`, `critical`,
произвольные метки в алфавитном порядке, `status`.

### 8.5. Начальное значение

До завершения первой проверки метрика **не экспортируется**
(аналогично `app_dependency_health`, см. раздел 2.5).
После первой проверки все 8 серий появляются одновременно.

### 8.6. Пример вывода

Endpoint `pg.svc:5432` доступен (status = ok):

```text
# HELP app_dependency_status Category of the last check result
# TYPE app_dependency_status gauge
app_dependency_status{name="order-api",dependency="postgres-main",type="postgres",host="pg.svc",port="5432",critical="yes",status="ok"} 1
app_dependency_status{name="order-api",dependency="postgres-main",type="postgres",host="pg.svc",port="5432",critical="yes",status="timeout"} 0
app_dependency_status{name="order-api",dependency="postgres-main",type="postgres",host="pg.svc",port="5432",critical="yes",status="connection_error"} 0
app_dependency_status{name="order-api",dependency="postgres-main",type="postgres",host="pg.svc",port="5432",critical="yes",status="dns_error"} 0
app_dependency_status{name="order-api",dependency="postgres-main",type="postgres",host="pg.svc",port="5432",critical="yes",status="auth_error"} 0
app_dependency_status{name="order-api",dependency="postgres-main",type="postgres",host="pg.svc",port="5432",critical="yes",status="tls_error"} 0
app_dependency_status{name="order-api",dependency="postgres-main",type="postgres",host="pg.svc",port="5432",critical="yes",status="unhealthy"} 0
app_dependency_status{name="order-api",dependency="postgres-main",type="postgres",host="pg.svc",port="5432",critical="yes",status="error"} 0
```

---

## 9. Метрика детального статуса: `app_dependency_status_detail`

### 9.1. Описание

Gauge-метрика (info-паттерн), содержащая **детальную причину** результата последней проверки.
Одна серия на endpoint, значение всегда = `1`. Метка `detail` содержит конкретную причину.
При смене причины старая серия удаляется, новая создаётся
(допустимый series churn для info-метрик).

### 9.2. Свойства

| Свойство | Значение |
| --- | --- |
| Имя | `app_dependency_status_detail` |
| Тип | Gauge |
| Текст HELP | `Detailed reason of the last check result` |
| Допустимые значения | `1` (всегда) |

### 9.3. Значения detail по типам чекеров

| Тип чекера | Возможные значения detail |
| --- | --- |
| HTTP | `ok`, `timeout`, `connection_refused`, `dns_error`, `tls_error`, `http_NNN` (конкретный HTTP-код: `http_404`, `http_503` и т.д.), `error` |
| gRPC | `ok`, `timeout`, `connection_refused`, `dns_error`, `tls_error`, `grpc_not_serving`, `grpc_unknown`, `error` |
| TCP | `ok`, `timeout`, `connection_refused`, `dns_error`, `error` |
| PostgreSQL | `ok`, `timeout`, `connection_refused`, `dns_error`, `auth_error`, `tls_error`, `error` |
| MySQL | `ok`, `timeout`, `connection_refused`, `dns_error`, `auth_error`, `tls_error`, `error` |
| Redis | `ok`, `timeout`, `connection_refused`, `dns_error`, `auth_error`, `unhealthy`, `error` |
| AMQP | `ok`, `timeout`, `connection_refused`, `dns_error`, `auth_error`, `tls_error`, `unhealthy`, `error` |
| Kafka | `ok`, `timeout`, `connection_refused`, `dns_error`, `no_brokers`, `error` |

### 9.4. Маппинг detail → status (категория)

Каждое значение `detail` соответствует ровно одной категории `status` (раздел 8.3):

| detail | status |
| --- | --- |
| `ok` | `ok` |
| `timeout` | `timeout` |
| `connection_refused`, `network_unreachable`, `host_unreachable` | `connection_error` |
| `dns_error` | `dns_error` |
| `auth_error` | `auth_error` |
| `tls_error` | `tls_error` |
| `http_NNN`, `grpc_not_serving`, `grpc_unknown`, `unhealthy`, `no_brokers` | `unhealthy` |
| `error`, `pool_exhausted`, `query_error` | `error` |

### 9.5. Метки

Те же обязательные и произвольные метки, что и у `app_dependency_health` (разделы 2.3, 2.4),
плюс метка `detail` в конце.

Порядок меток: `name`, `dependency`, `type`, `host`, `port`, `critical`,
произвольные метки в алфавитном порядке, `detail`.

### 9.6. Начальное значение

До завершения первой проверки метрика **не экспортируется**
(аналогично `app_dependency_health`, см. раздел 2.5).

### 9.7. Пример вывода

`payment-api` вернул HTTP 503:

```text
# HELP app_dependency_status_detail Detailed reason of the last check result
# TYPE app_dependency_status_detail gauge
app_dependency_status_detail{name="order-api",dependency="payment-api",type="http",host="payment.svc",port="8080",critical="yes",detail="http_503"} 1
```

### 9.8. Влияние на хранение

Каждый endpoint порождает:

- `app_dependency_health`: 1 серия
- `app_dependency_latency_seconds`: 10 серий (8 бакетов + sum + count)
- `app_dependency_status`: **8 серий** (по одной на значение status)
- `app_dependency_status_detail`: **1 серия**

Итого: +9 серий на endpoint по сравнению с базовым набором (health + latency).

---

## 10. Расширенные PromQL-запросы

В дополнение к запросам раздела 7, следующие запросы используют новые
метрики status и detail:

```promql
# Текущая категория состояния всех зависимостей
app_dependency_status == 1

# Все зависимости с таймаутом
app_dependency_status{status="timeout"} == 1

# Все зависимости с ошибкой аутентификации (alert-friendly)
app_dependency_status{status="auth_error"} == 1

# Детальная причина по конкретной зависимости
app_dependency_status_detail{name="order-api",dependency="payment-api"}

# Все HTTP 503 ошибки по всему кластеру
app_dependency_status_detail{detail="http_503"}

# Обнаружение flapping (без series churn — работает корректно!)
changes(app_dependency_status{status="ok"}[15m]) > 4

# Распределение по статусам
count by (status) (app_dependency_status == 1)

# Корреляция: unhealthy зависимости с деталями через join
app_dependency_status{status="unhealthy"} == 1
  AND on (name, dependency, type, host, port)
app_dependency_status_detail

# Алерт: критичная зависимость с non-ok статусом более 5 минут
app_dependency_status{status!="ok",critical="yes"} == 1
  AND on (name, dependency, type, host, port)
(app_dependency_status offset 5m {status!="ok"} == 1)
```
