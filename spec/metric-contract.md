# Контракт метрик

> Версия спецификации: **1.0-draft**
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
| `dependency` | Логическое имя зависимости, задаётся разработчиком | Строчные буквы, цифры, `-`. Длина: 1-63 символа. Формат: `[a-z][a-z0-9-]*` | `postgres-main` |
| `type` | Тип соединения / протокол | Одно из: `http`, `grpc`, `tcp`, `postgres`, `mysql`, `redis`, `amqp`, `kafka` | `postgres` |
| `host` | Адрес endpoint-а (hostname или IP) | Как есть из конфигурации. IPv6 без квадратных скобок | `pg-master.db.svc.cluster.local` |
| `port` | Порт endpoint-а | Строка с числом 1-65535. Если порт не указан, используется порт по умолчанию для данного типа | `5432` |

### 2.4. Опциональные метки

| Метка | Описание | Пример |
| --- | --- | --- |
| `role` | Роль экземпляра в кластере | `primary`, `replica`, `master`, `slave` |
| `shard` | Идентификатор шарда | `shard-01`, `0` |
| `vhost` | Virtual host (для AMQP) | `/`, `production` |

Опциональные метки добавляются только при явном указании разработчиком
через поле `metadata` структуры `Endpoint`. Если метка не указана,
она **не включается** в метрику (а не выводится с пустым значением).

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
app_dependency_health{dependency="postgres-main",type="postgres",host="pg-master.db.svc",port="5432"} 1
app_dependency_health{dependency="redis-cache",type="redis",host="redis-0.cache.svc",port="6379"} 1
app_dependency_health{dependency="payment-service",type="http",host="payment-svc.payments.svc",port="8080"} 0

# HELP app_dependency_latency_seconds Latency of dependency health check in seconds
# TYPE app_dependency_latency_seconds histogram
app_dependency_latency_seconds_bucket{dependency="postgres-main",type="postgres",host="pg-master.db.svc",port="5432",le="0.001"} 0
app_dependency_latency_seconds_bucket{dependency="postgres-main",type="postgres",host="pg-master.db.svc",port="5432",le="0.005"} 8
app_dependency_latency_seconds_bucket{dependency="postgres-main",type="postgres",host="pg-master.db.svc",port="5432",le="0.01"} 15
app_dependency_latency_seconds_bucket{dependency="postgres-main",type="postgres",host="pg-master.db.svc",port="5432",le="0.05"} 20
app_dependency_latency_seconds_bucket{dependency="postgres-main",type="postgres",host="pg-master.db.svc",port="5432",le="0.1"} 20
app_dependency_latency_seconds_bucket{dependency="postgres-main",type="postgres",host="pg-master.db.svc",port="5432",le="0.5"} 20
app_dependency_latency_seconds_bucket{dependency="postgres-main",type="postgres",host="pg-master.db.svc",port="5432",le="1"} 20
app_dependency_latency_seconds_bucket{dependency="postgres-main",type="postgres",host="pg-master.db.svc",port="5432",le="5"} 20
app_dependency_latency_seconds_bucket{dependency="postgres-main",type="postgres",host="pg-master.db.svc",port="5432",le="+Inf"} 20
app_dependency_latency_seconds_sum{dependency="postgres-main",type="postgres",host="pg-master.db.svc",port="5432"} 0.085
app_dependency_latency_seconds_count{dependency="postgres-main",type="postgres",host="pg-master.db.svc",port="5432"} 20
```

### 4.2. Требования к формату

- Строки `# HELP` и `# TYPE` обязательны для каждой метрики.
- Текст `# HELP` фиксирован (см. примеры выше) и не должен отличаться между SDK.
- Порядок меток: `dependency`, `type`, `host`, `port`, затем опциональные
  в алфавитном порядке.
- Значения меток экранируются согласно Prometheus exposition format:
  символы `\`, `"`, `\n` заменяются на `\\`, `\"`, `\n`.

---

## 5. Поведение при множественных endpoint-ах

Одна зависимость может иметь несколько endpoint-ов (реплики БД, ноды кластера).

### 5.1. Правило: одна метрика на endpoint

Каждый endpoint порождает **отдельную** серию метрик. Агрегация не производится.

**Пример**: PostgreSQL с primary и replica:

```text
app_dependency_health{dependency="postgres-main",type="postgres",host="pg-primary.db.svc",port="5432",role="primary"} 1
app_dependency_health{dependency="postgres-main",type="postgres",host="pg-replica.db.svc",port="5432",role="replica"} 1

app_dependency_latency_seconds_bucket{dependency="postgres-main",type="postgres",host="pg-primary.db.svc",port="5432",role="primary",le="0.005"} 10
app_dependency_latency_seconds_bucket{dependency="postgres-main",type="postgres",host="pg-replica.db.svc",port="5432",role="replica",le="0.005"} 8
```

### 5.2. Обоснование

- Позволяет точно определить, какой именно endpoint недоступен.
- Алертинг может быть настроен на уровне отдельных endpoint-ов
  (например, `DependencyDegraded` при partial failure).
- Агрегация при необходимости выполняется на уровне PromQL:
  `min by (dependency) (app_dependency_health{dependency="postgres-main"})`.

### 5.3. Kafka: несколько брокеров

Для Kafka каждый брокер является отдельным endpoint-ом:

```text
app_dependency_health{dependency="kafka-main",type="kafka",host="kafka-0.kafka.svc",port="9092"} 1
app_dependency_health{dependency="kafka-main",type="kafka",host="kafka-1.kafka.svc",port="9092"} 1
app_dependency_health{dependency="kafka-main",type="kafka",host="kafka-2.kafka.svc",port="9092"} 0
```

---

## 6. Примеры типовых конфигураций

### 6.1. Минимальная конфигурация (один сервис, одна зависимость)

```text
# HELP app_dependency_health Health status of a dependency (1 = healthy, 0 = unhealthy)
# TYPE app_dependency_health gauge
app_dependency_health{dependency="redis-cache",type="redis",host="redis.default.svc",port="6379"} 1

# HELP app_dependency_latency_seconds Latency of dependency health check in seconds
# TYPE app_dependency_latency_seconds histogram
app_dependency_latency_seconds_bucket{dependency="redis-cache",type="redis",host="redis.default.svc",port="6379",le="0.001"} 5
app_dependency_latency_seconds_bucket{dependency="redis-cache",type="redis",host="redis.default.svc",port="6379",le="0.005"} 10
app_dependency_latency_seconds_bucket{dependency="redis-cache",type="redis",host="redis.default.svc",port="6379",le="0.01"} 10
app_dependency_latency_seconds_bucket{dependency="redis-cache",type="redis",host="redis.default.svc",port="6379",le="0.05"} 10
app_dependency_latency_seconds_bucket{dependency="redis-cache",type="redis",host="redis.default.svc",port="6379",le="0.1"} 10
app_dependency_latency_seconds_bucket{dependency="redis-cache",type="redis",host="redis.default.svc",port="6379",le="0.5"} 10
app_dependency_latency_seconds_bucket{dependency="redis-cache",type="redis",host="redis.default.svc",port="6379",le="1"} 10
app_dependency_latency_seconds_bucket{dependency="redis-cache",type="redis",host="redis.default.svc",port="6379",le="5"} 10
app_dependency_latency_seconds_bucket{dependency="redis-cache",type="redis",host="redis.default.svc",port="6379",le="+Inf"} 10
app_dependency_latency_seconds_sum{dependency="redis-cache",type="redis",host="redis.default.svc",port="6379"} 0.025
app_dependency_latency_seconds_count{dependency="redis-cache",type="redis",host="redis.default.svc",port="6379"} 10
```

### 6.2. Типичный микросервис (несколько зависимостей разных типов)

```text
# HELP app_dependency_health Health status of a dependency (1 = healthy, 0 = unhealthy)
# TYPE app_dependency_health gauge
app_dependency_health{dependency="postgres-main",type="postgres",host="pg.db.svc",port="5432"} 1
app_dependency_health{dependency="redis-cache",type="redis",host="redis.cache.svc",port="6379"} 1
app_dependency_health{dependency="payment-service",type="http",host="payment.payments.svc",port="8080"} 1
app_dependency_health{dependency="auth-service",type="grpc",host="auth.auth.svc",port="9090"} 0
app_dependency_health{dependency="rabbitmq",type="amqp",host="rabbit.mq.svc",port="5672"} 1
```

### 6.3. Сервис с AMQP и vhost

```text
app_dependency_health{dependency="rabbitmq-orders",type="amqp",host="rabbit.mq.svc",port="5672",vhost="orders"} 1
app_dependency_health{dependency="rabbitmq-notifications",type="amqp",host="rabbit.mq.svc",port="5672",vhost="notifications"} 1
```

### 6.4. Сервис в состоянии деградации (partial failure)

```text
app_dependency_health{dependency="postgres-main",type="postgres",host="pg-primary.db.svc",port="5432",role="primary"} 1
app_dependency_health{dependency="postgres-main",type="postgres",host="pg-replica-1.db.svc",port="5432",role="replica"} 0
app_dependency_health{dependency="postgres-main",type="postgres",host="pg-replica-2.db.svc",port="5432",role="replica"} 1
```

---

## 7. Полезные PromQL-запросы

Для справки: типичные запросы, которые будут использоваться в Grafana и алертах.

```promql
# Все нездоровые зависимости
app_dependency_health == 0

# Нездоровые зависимости конкретного сервиса (по job/instance)
app_dependency_health{job="my-service"} == 0

# Агрегированное здоровье зависимости (хотя бы один endpoint down)
min by (dependency) (app_dependency_health) == 0

# P99 латентность проверок за 5 минут
histogram_quantile(0.99, rate(app_dependency_latency_seconds_bucket[5m]))

# Средняя латентность по зависимости
rate(app_dependency_latency_seconds_sum[5m]) / rate(app_dependency_latency_seconds_count[5m])

# Зависимости, которые "мигают" (flapping) — частые переключения
changes(app_dependency_health[15m]) > 4
```
