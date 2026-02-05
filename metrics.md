# Мониторинг зависимостей микросервисов: анализ и реализация

## Контекст

Система из 300 микросервисов в Kubernetes. Сбор метрик — VictoriaMetrics, визуализация — Grafana.

**Цели:**

- Отображение зависимостей микросервисов друг от друга
- Автоматическая пометка зависимых сервисов при отказе одного из них
- Учёт количества реплик (частичная деградация — жёлтый цвет)
- Формирование карты зависимостей

**Рассматриваемое предложение:** добавить в каждый микросервис метрику, отражающую состояние соединений с зависимостями. Значение `1` — соединение в порядке, `0` — нет. В метках — информация о том, куда настроено соединение (host, port и т.п.).

---

## Часть 1. Обсуждение предложения

### Позиция архитектора ПО

**Подход жизнеспособен**, но требует стандартизации. 300 сервисов — это масштаб, при котором без единого стандарта неизбежен хаос в именовании метрик и меток.

Ключевые требования к реализации:

1. **Единая библиотека/SDK.** Метрику нельзя отдавать «на откуп» каждой команде. Нужна общая библиотека, которая:
   - стандартизирует имя метрики, набор меток и логику проверки
   - поддерживает типовые проверки (TCP, HTTP, gRPC, database ping, Redis ping, AMQP)
   - читает конфигурацию зависимостей из ConfigMap / переменных окружения

2. **Метрика должна быть не только бинарной.** Бинарное значение `0/1` не отражает деградацию. TCP-соединение может быть установлено, но сервис отвечает с ошибками 5xx. Предлагается расширить модель (см. раздел «Дизайн метрик»).

3. **Используйте логические имена, а не IP-адреса.** В Kubernetes IP подов и сервисов нестабильны. В метках должны быть Kubernetes service names (`payment-svc.payments.svc`) — они стабильны и позволяют строить граф зависимостей.

4. **Кардинальность управляема.** При 300 сервисах и в среднем 5–10 зависимостях на сервис получаем 1500–3000 временных рядов. Для VictoriaMetrics это пренебрежимо малая нагрузка.

### Позиция архитектора службы поддержки

**Подход полезен для эксплуатации**, но с оговорками:

1. **Кто поддерживает список зависимостей?** Если зависимости объявлены в конфигурации каждого сервиса — это ответственность команд разработки. Нужен процесс:
   - обязательное объявление зависимостей при деплое (CI/CD валидация)
   - периодическая сверка объявленных зависимостей с реальным трафиком (по данным трейсинга или service mesh)

2. **Ложные срабатывания.** Кратковременные сетевые проблемы приведут к «мерцанию» метрики. Нужна устойчивость: сглаживание через `avg_over_time`, пороговые значения, задержка перед алертом.

3. **Транзитивные зависимости — ключевая проблема.** Если сервис A зависит от B, а B от C, и C упал, то:
   - B покажет `dependency_health=0` для C
   - A покажет `dependency_health=0` для B (т.к. B деградирован)
   - Но чистый PromQL не умеет рекурсивно обходить граф

   Для определения root cause нужен внешний компонент или статическая карта зависимостей.

4. **Grafana Node Graph** — подходящий инструмент для карты зависимостей, но требует специального формата данных (nodes + edges). Потребуется отдельный datasource или recording rules + transformations.

---

## Часть 2. Плюсы и минусы

### Плюсы

| # | Плюс | Комментарий |
| --- | ------ | ------------- |
| 1 | **Работает с существующим стеком** | VictoriaMetrics + Grafana, не нужна новая инфраструктура |
| 2 | **Каждый сервис знает свои зависимости лучше всех** | Информация из первых рук, а не из наблюдения за трафиком |
| 3 | **Реальное время** | Метрика обновляется каждые 10–30 секунд |
| 4 | **Покрывает все типы зависимостей** | HTTP, gRPC, БД, кэш, очереди, внешние API |
| 5 | **Низкая кардинальность** | ~3000 рядов — ничто для VictoriaMetrics |
| 6 | **Позволяет строить граф зависимостей** | Из метрик можно извлечь пары `service → dependency` |
| 7 | **Учёт реплик** | Разные поды одного сервиса могут показывать разный статус — видна частичная деградация |

### Минусы

| # | Минус | Митигация |
| --- | ------- | --------- |
| 1 | **Требует изменений во всех 300 сервисах** | Общая библиотека/SDK минимизирует объём работ |
| 2 | ~~**Бинарная метрика не отражает деградацию**~~ | ~~Расширить модель~~ — **решено**: метрика принимает значения от 0.0 до 1.0 (доля доступных endpoint-ов) |
| 3 | **Дрейф конфигурации** | Автосверка с реальным трафиком (трейсинг / service mesh) |
| 4 | **Не видит транзитивных зависимостей** | Внешний компонент для вычисления propagation |
| 5 | **Ложные срабатывания при сетевых glitch-ах** | Сглаживание: `avg_over_time`, задержки алертов |
| 6 | **Активные проверки создают дополнительный трафик** | Незначительный: 1 запрос в 15 секунд на зависимость |
| 7 | **Не покрывает зависимости через очереди** | Для async-зависимостей — проверка доступности брокера, а не consumer-а |

---

## Часть 3. Дизайн метрик

### Основная метрика: `app_dependency_health` (per-connection)

Одно имя метрики, **бинарное значение 0/1**, но **отдельный ряд на каждое
соединение** — различаются метками `host` и `port`:

```text
# order-service → payment-service (3 реплики)
app_dependency_health{dependency="payment-service", type="http",
  host="payment-1.payments.svc", port="8080"} 1
app_dependency_health{dependency="payment-service", type="http",
  host="payment-2.payments.svc", port="8080"} 1
app_dependency_health{dependency="payment-service", type="http",
  host="payment-3.payments.svc", port="8080"} 0

# order-service → PostgreSQL кластер (master + 2 реплики)
app_dependency_health{dependency="postgres-main", type="postgres",
  host="pg-master.db.svc", port="5432"} 1
app_dependency_health{dependency="postgres-main", type="postgres",
  host="pg-replica-1.db.svc", port="5432"} 0
app_dependency_health{dependency="postgres-main", type="postgres",
  host="pg-replica-2.db.svc", port="5432"} 1

# order-service → RabbitMQ (все брокеры недоступны)
app_dependency_health{dependency="rabbitmq", type="amqp",
  host="rabbit-0.mq.svc", port="5672"} 0
app_dependency_health{dependency="rabbitmq", type="amqp",
  host="rabbit-1.mq.svc", port="5672"} 0
```

**Метки:**

| Метка | Обяз. | Описание | Пример |
| ------- | :---: | --------- | -------- |
| `dependency` | да | Логическое имя зависимости | `payment-service`, `postgres-main` |
| `type` | да | Тип соединения | `http`, `grpc`, `postgres`, `redis`, `amqp` |
| `host` | да | Адрес endpoint-а | `pg-master.db.svc.cluster.local` |
| `port` | да | Порт endpoint-а | `5432` |

Опциональные метки при необходимости: `role` (master/replica),
`shard` (для Redis Cluster), `vhost` (для RabbitMQ).

> Метка `service`/`job` (имя текущего сервиса) добавляется автоматически
> при scrape через relabeling.

### Почему per-connection, а не pre-computed ratio?

Ранее рассматривался подход с предвычисленным ratio в коде
(`app_dependency_health = 0.67`). **Per-connection подход лучше:**

| Критерий | Pre-computed ratio | Per-connection (выбран) |
| -------- | :--: | :--: |
| Кол-во имён метрик | 5–6 | **2** (health + latency) |
| Сложность кода | Высокая (ratio, counters) | **Низкая** (просто 0/1) |
| Видно, какой endpoint упал | Нужна отдельная метрика | **Да, из коробки** |
| Агрегация в PromQL | Не нужна (предвычислена) | `avg by (dependency)(...)` |
| Идиоматичность Prometheus | Нестандартно | **Стандартный паттерн** (как `up`) |
| Кардинальность | ~19000 (5 метрик) | **~6300** (1 метрика) |

Все агрегаты, которые давали отдельные метрики, легко вычисляются в PromQL:

```promql
# Доля доступных endpoint-ов (эквивалент ratio)
avg by (job, dependency)(app_dependency_health)

# Сколько всего endpoint-ов
count by (job, dependency)(app_dependency_health)

# Сколько доступных (значение 0/1, поэтому sum = count healthy)
sum by (job, dependency)(app_dependency_health)

# Сколько недоступных
count by (job, dependency)(app_dependency_health)
  - sum by (job, dependency)(app_dependency_health)

# Какие именно endpoint-ы недоступны
app_dependency_health{dependency="postgres-main"} == 0
```

### Метрика латентности: `app_dependency_latency_seconds`

```text
# Латентность по каждому endpoint-у (histogram)
app_dependency_latency_seconds_bucket{dependency="postgres-main",
  type="postgres", host="pg-master.db.svc", port="5432", le="0.01"} 145
app_dependency_latency_seconds_bucket{dependency="postgres-main",
  type="postgres", host="pg-master.db.svc", port="5432", le="0.1"} 192
...
```

Те же метки, что у `app_dependency_health` — можно коррелировать
латентность с доступностью для каждого endpoint-а.

### Полная модель метрик (сводка)

| Метрика | Тип | Значение | Метки | Назначение |
| ------- | --- | -------- | ----- | ---------- |
| `app_dependency_health` | Gauge | 0 / 1 | dependency, type, host, port | Статус каждого соединения |
| `app_dependency_latency_seconds` | Histogram | секунды | dependency, type, host, port | Латентность проверки |

Всего **2 имени метрик**. Всё остальное — через PromQL.

**Кардинальность:** 300 сервисов × ~7 зависимостей × ~3 endpoint-а = **~6300 рядов**
для `app_dependency_health`. Для VictoriaMetrics это пренебрежимо мало.

### Конфигурация зависимостей

Зависимости описываются в ConfigMap и монтируются в сервис. Каждая зависимость может иметь **несколько endpoint-ов**:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: order-service-dependencies
data:
  dependencies.yaml: |
    dependencies:
      # Сервис с несколькими репликами
      - name: payment-service
        type: http
        critical: true
        checkInterval: 15s
        timeout: 5s
        endpoints:
          - host: payment-svc.payments.svc.cluster.local
            port: 8080
            healthPath: /health
        # Или auto-discovery через Kubernetes Endpoints API:
        # discovery: kubernetes
        # serviceName: payment-svc
        # namespace: payments

      # Кластер PostgreSQL: master + 2 реплики
      - name: postgres-main
        type: postgres
        critical: true
        checkInterval: 10s
        timeout: 3s
        endpoints:
          - host: pg-master.database.svc.cluster.local
            port: 5432
            role: master
          - host: pg-replica-1.database.svc.cluster.local
            port: 5432
            role: replica
          - host: pg-replica-2.database.svc.cluster.local
            port: 5432
            role: replica

      # Redis кластер с несколькими шардами
      - name: redis-cache
        type: redis
        critical: false
        checkInterval: 10s
        timeout: 2s
        endpoints:
          - host: redis-0.redis.cache.svc.cluster.local
            port: 6379
          - host: redis-1.redis.cache.svc.cluster.local
            port: 6379
          - host: redis-2.redis.cache.svc.cluster.local
            port: 6379

      # Простой сервис с одним endpoint-ом
      - name: notification-service
        type: grpc
        critical: false
        checkInterval: 30s
        timeout: 5s
        endpoints:
          - host: notification-svc.notifications.svc.cluster.local
            port: 9090
```

**Дополнительные возможности конфигурации:**

| Поле | Описание |
| ---- | -------- |
| `critical` | Влияет ли отказ на `readiness` пода |
| `discovery: kubernetes` | Автообнаружение endpoint-ов через Kubernetes Endpoints API вместо статического списка |
| `role` | Роль endpoint-а (master/replica) — попадает в метку метрики |
| `minHealthy` | Минимальное кол-во endpoint-ов, при котором зависимость считается работоспособной (по умолчанию: 1) |

Поле `critical` определяет, влияет ли отказ зависимости на `readiness` пода. Это обеспечивает автоматическое исключение деградированных подов из балансировки. С многосостояльной метрикой можно задавать порог: `readiness` падает, если `app_dependency_health < minHealthy/total`.

---

## Часть 4. Реализация в Grafana

### Дашборд 1: Обзор всех сервисов (Status Grid)

Таблица / Honeycomb с цветовой кодировкой. Цвета отражают **два уровня
деградации**: внутри зависимости (часть endpoint-ов) и между репликами
сервиса (часть подов видит проблему).

Используем recording rule `service:dependency:health` (см. ниже).

| Цвет | Состояние | Условие по recording rule |
| ---- | --------- | ------------------------- |
| Зелёный | Все зависимости OK | `min by (job)(service:dependency:health) == 1` |
| Жёлтый | Частичная деградация | `> 0 AND < 1` |
| Оранжевый | Серьёзная деградация | `> 0 AND <= 0.5` |
| Красный | Полный отказ зависимости | `== 0` |
| Серый | Метрика отсутствует | нет данных |

### Дашборд 2: Детали сервиса

Для конкретного сервиса (переменная `$service`):

- **Таблица зависимостей** — строка на каждую dependency:

  ```promql
  # Доля здоровых endpoint-ов для каждой зависимости
  avg by (dependency, type)(
    app_dependency_health{job="$service"}
  )
  ```

  Рядом — абсолютные числа:

  ```promql
  # Сколько endpoint-ов всего
  count by (dependency)(app_dependency_health{job="$service"})
  # Сколько живых
  sum by (dependency)(app_dependency_health{job="$service"})
  ```

- **Таблица endpoint-ов** — при клике на зависимость:

  ```promql
  # Статус каждого endpoint-а (видно, какой именно узел упал)
  app_dependency_health{job="$service", dependency="$dependency"}
  ```

- **Графики** — `app_dependency_health` и latency по времени
- **Реплики** — статус по `instance` (видно, если проблема у части подов)

### Дашборд 3: Карта зависимостей (Node Graph)

Grafana **Node Graph panel** отображает граф (nodes + edges).

**Recording rules** в VictoriaMetrics для подготовки данных:

```yaml
groups:
  - name: dependency_graph
    interval: 30s
    rules:
      # Здоровье каждой зависимости: avg endpoint-ов, avg реплик
      # Используется в дашбордах и алертах
      - record: service:dependency:health
        expr: >
          avg by (job, dependency, type) (
            avg by (job, dependency, type, instance) (
              app_dependency_health
            )
          )

      # Общее здоровье сервиса (avg по всем зависимостям)
      - record: service:health:avg
        expr: >
          avg by (job) (service:dependency:health)

      # Количество endpoint-ов зависимости
      - record: service:dependency:endpoints:total
        expr: >
          count by (job, dependency) (app_dependency_health)

      # Количество здоровых endpoint-ов
      - record: service:dependency:endpoints:healthy
        expr: >
          sum by (job, dependency) (app_dependency_health)
```

> **Двойная агрегация** в `service:dependency:health`:
> сначала `avg by instance` (внутри каждого пода — доля доступных
> endpoint-ов), затем `avg by job` (между подами).
> Это корректно обрабатывает ситуацию, когда один под видит 3/3
> endpoint-а, а другой — 1/3.

В Grafana Node Graph:

- **Nodes** = уникальные `job` (сервисы)
- **Edges** = пары `job -> dependency` из `service:dependency:health`
- **Цвет узла** = `service:health:avg` (зелёный/жёлтый/красный)
- **Цвет ребра** = `service:dependency:health` (градиент 0..1)
- **Толщина ребра** = `service:dependency:endpoints:total` (больше
  endpoint-ов — толще линия)

> **Ограничение:** при 300 узлах граф перегружен. Решение — фильтрация:
> показывать только проблемные сервисы и их окружение (+/-1 hop),
> или группировка по namespace/domain.

### Дашборд 4: Impact Analysis

При выборе упавшего сервиса — показать blast radius:

```promql
# Все endpoint-ы payment-service, которые кто-то считает недоступными
app_dependency_health{dependency="payment-service"} == 0

# Какие сервисы видят деградацию payment-service
service:dependency:health{dependency="payment-service"} < 1

# Сколько endpoint-ов payment-service недоступно (по сервисам)
service:dependency:endpoints:total{dependency="payment-service"}
  - service:dependency:endpoints:healthy{dependency="payment-service"}
```

Видно и **степень** влияния: полный отказ vs деградация, и по скольким
endpoint-ам.

---

## Часть 5. Алертинг

### Правила алертов

Алерты работают по **recording rules** (агрегированным данным),
а не по сырым метрикам — это стабильнее и устойчивее к флуктуациям
отдельных endpoint-ов.

```yaml
groups:
  - name: dependency_alerts
    rules:
      # ===== Полный отказ =====

      # Все endpoint-ы зависимости недоступны
      - alert: DependencyDown
        expr: service:dependency:health == 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: >-
            {{ $labels.job }}: зависимость {{ $labels.dependency }}
            полностью недоступна

      # ===== Деградация (часть endpoint-ов недоступна) =====

      # Серьёзная: доступно <= 50% endpoint-ов
      - alert: DependencyCriticallyDegraded
        expr: >
          service:dependency:health > 0
          AND
          service:dependency:health <= 0.5
        for: 2m
        labels:
          severity: warning
        annotations:
          summary: >-
            {{ $labels.job }}: {{ $labels.dependency }}
            серьёзно деградирована
            ({{ $value | humanizePercentage }} endpoint-ов доступно)

      # Умеренная: доступно > 50% но не все
      - alert: DependencyDegraded
        expr: >
          service:dependency:health > 0.5
          AND
          service:dependency:health < 1
        for: 5m
        labels:
          severity: info
        annotations:
          summary: >-
            {{ $labels.job }}: {{ $labels.dependency }}
            частично деградирована
            ({{ $value | humanizePercentage }} endpoint-ов доступно)

      # ===== Расхождение между репликами =====

      # Одни поды видят зависимость, другие — нет
      - alert: DependencyPartialOutage
        expr: >
          max by (job, dependency)(
            avg by (job, dependency, instance)(app_dependency_health)
          ) == 1
          AND
          min by (job, dependency)(
            avg by (job, dependency, instance)(app_dependency_health)
          ) == 0
        for: 3m
        labels:
          severity: warning
        annotations:
          summary: >-
            {{ $labels.job }}: часть реплик не видит
            {{ $labels.dependency }} (сетевая проблема?)

      # ===== Латентность =====

      - alert: DependencySlowResponse
        expr: >
          histogram_quantile(0.95,
            sum by (job, dependency, le)(
              rate(app_dependency_latency_seconds_bucket[5m])
            )
          ) > 1.0
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: >-
            {{ $labels.job }}: высокая латентность
            до {{ $labels.dependency }}
```

### Подавление каскадных алертов (Alertmanager Inhibition)

```yaml
# alertmanager.yml
inhibit_rules:
  # Если сервис полностью недоступен — подавить алерты
  # о зависимости от него в других сервисах
  - source_matchers:
      - alertname = "ServiceDown"
    target_matchers:
      - alertname = "DependencyDown"
    equal:
      - dependency  # имя упавшего сервиса = dependency в зависимом

  # Если полный отказ — подавить алерты о деградации
  - source_matchers:
      - alertname = "DependencyDown"
    target_matchers:
      - alertname =~ "DependencyDegraded|DependencyCriticallyDegraded"
    equal:
      - job
      - dependency
```

При падении корневого сервиса алерт приходит один раз
(`DependencyDown`), а не каскад от каждого зависимого.

---

## Часть 6. План внедрения

### Фаза 1: Фундамент (2–4 недели)

1. Разработать общую библиотеку для health-check зависимостей
   - Поддержка типов: HTTP, gRPC, PostgreSQL, MySQL, Redis, AMQP, Kafka
   - Экспорт метрик в формате Prometheus
   - Чтение конфигурации из YAML / ENV
2. Интеграция в 5–10 пилотных сервисов
3. Базовый дашборд в Grafana (таблица статусов)
4. Шаблон ConfigMap для описания зависимостей

### Фаза 2: Масштабирование (4–8 недель)

1. Раскатка библиотеки на все 300 сервисов (постепенно, по командам)
2. Recording rules в VictoriaMetrics
3. Дашборды: детали сервиса + карта зависимостей (Node Graph)
4. Настройка алертов и inhibition rules
5. CI/CD валидация: наличие `dependencies.yaml` при деплое

### Фаза 3: Улучшения (8–12 недель)

1. Сверка объявленных зависимостей с реальным трафиком (через OpenTelemetry / access logs)
2. Автогенерация `dependencies.yaml` из трейсов (предложение, а не замена)
3. Impact analysis дашборд
4. Интеграция со Slack/PagerDuty: при падении сервиса автоматически уведомлять владельцев зависимых сервисов

### Фаза 4: Перспектива

1. Рассмотреть service mesh (Istio/Linkerd) для автоматического сбора метрик зависимостей без изменения кода
2. eBPF-инструменты (Cilium Hubble) для наблюдения за трафиком на уровне ядра
3. Grafana Tempo + Service Map для визуализации зависимостей из трейсов

---

## Часть 7. Альтернативные подходы (сравнение)

| Подход | Изменения в коде | Покрытие | Точность | Стоимость | Транзитивные зависимости |
| ------ | :-: | :-: | :-: | :-: | :-: |
| **Метрика в сервисе (предложение)** | Да, все сервисы | Полное (вкл. БД, кэш) | Высокая | Низкая | Нет (нужен внешний компонент) |
| **Service Mesh (Istio)** | Нет | Только HTTP/gRPC | Средняя | Высокая (sidecar overhead) | Да, из трафика |
| **OpenTelemetry трейсинг** | Да, все сервисы | HTTP/gRPC/DB (с инструментацией) | Высокая | Средняя | Да, из trace spans |
| **eBPF (Cilium Hubble)** | Нет | TCP/UDP трафик | Средняя (нет семантики) | Средняя | Частично |
| **Анализ access logs** | Нет | HTTP/gRPC | Средняя | Низкая | Да, из логов |

**Рекомендация:** предложенный подход с метриками в сервисах — оптимальный стартовый вариант. Он работает с текущим стеком, покрывает все типы зависимостей (включая БД и кэш, которые service mesh не видит) и даёт сервисным командам явный контроль над списком зависимостей. В перспективе его стоит дополнить трейсингом (OpenTelemetry) для автоматической валидации и обнаружения транзитивных зависимостей.

---

## Часть 8. Пример реализации метрики (Go)

Код стал проще — не нужно вычислять ratio, не нужны вспомогательные
метрики. Каждый endpoint — просто `Set(1)` или `Set(0)`.

```go
package depcheck

import (
    "context"
    "sync"
    "time"

    "github.com/prometheus/client_golang/prometheus"
)

var (
    // Статус каждого соединения: 1 = доступен, 0 = недоступен.
    // Отдельный ряд на каждый endpoint (различаются метками).
    dependencyHealth = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "app_dependency_health",
            Help: "Статус соединения с endpoint-ом зависимости (1=OK, 0=FAIL)",
        },
        []string{"dependency", "type", "host", "port"},
    )

    // Латентность проверки (per-endpoint)
    dependencyLatency = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "app_dependency_latency_seconds",
            Help:    "Латентность проверки endpoint-а зависимости",
            Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0, 5.0},
        },
        []string{"dependency", "type", "host", "port"},
    )
)

func init() {
    prometheus.MustRegister(dependencyHealth, dependencyLatency)
}

// Endpoint описывает один endpoint зависимости
type Endpoint struct {
    Host       string `yaml:"host"`
    Port       string `yaml:"port"`
    Role       string `yaml:"role,omitempty"`
    HealthPath string `yaml:"healthPath,omitempty"`
}

// Dependency описывает зависимость (с несколькими endpoint-ами)
type Dependency struct {
    Name          string        `yaml:"name"`
    Type          string        `yaml:"type"`
    Endpoints     []Endpoint    `yaml:"endpoints"`
    CheckInterval time.Duration `yaml:"checkInterval"`
    Timeout       time.Duration `yaml:"timeout"`
    Critical      bool          `yaml:"critical"`
}

// Checker выполняет периодические проверки
type Checker struct {
    deps []Dependency
    mu   sync.RWMutex
}

func NewChecker(deps []Dependency) *Checker {
    return &Checker{deps: deps}
}

// Start запускает проверки для всех зависимостей
func (c *Checker) Start(ctx context.Context) {
    for _, dep := range c.deps {
        go c.runChecks(ctx, dep)
    }
}

func (c *Checker) runChecks(ctx context.Context, dep Dependency) {
    ticker := time.NewTicker(dep.CheckInterval)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            // Проверяем все endpoint-ы зависимости
            for _, ep := range dep.Endpoints {
                checkCtx, cancel := context.WithTimeout(ctx, dep.Timeout)

                start := time.Now()
                ok := c.checkEndpoint(checkCtx, dep.Type, ep)
                elapsed := time.Since(start).Seconds()
                cancel()

                labels := []string{dep.Name, dep.Type, ep.Host, ep.Port}

                dependencyLatency.WithLabelValues(labels...).Observe(elapsed)
                if ok {
                    dependencyHealth.WithLabelValues(labels...).Set(1)
                } else {
                    dependencyHealth.WithLabelValues(labels...).Set(0)
                }
            }
        }
    }
}

func (c *Checker) checkEndpoint(
    ctx context.Context, depType string, ep Endpoint,
) bool {
    switch depType {
    case "http":
        return c.checkHTTP(ctx, ep)
    case "grpc":
        return c.checkGRPC(ctx, ep)
    case "postgres", "mysql":
        return c.checkDB(ctx, ep)
    case "redis":
        return c.checkRedis(ctx, ep)
    default:
        return c.checkTCP(ctx, ep)
    }
}

// checkHTTP, checkGRPC, checkDB, checkRedis, checkTCP — реализации
// ...
```

Обратите внимание: **всего 2 регистрации метрик** вместо 6.
Агрегация полностью на стороне PromQL/recording rules.

---

## Итого

Предложение **добавить метрику зависимостей в микросервисы — обоснованное
и практичное решение**.

Финальная модель: **одна метрика `app_dependency_health`**, бинарная (0/1),
с отдельным рядом на каждое соединение (различаются метками `host`, `port`).
Деградация вычисляется в PromQL через `avg` — это стандартный паттерн
Prometheus, аналогичный встроенной метрике `up`.

Решение работает с текущей инфраструктурой, имеет управляемую кардинальность
(~6300 рядов, 2 имени метрик) и закрывает все поставленные задачи:

- **Карта зависимостей** — строится из пар `service -> dependency`
- **Маркировка проблемных сервисов** — 4 цвета в Grafana через
  recording rules (зелёный/жёлтый/оранжевый/красный)
- **Учёт реплик** — двойная агрегация: `avg by instance` (внутри пода)
  и `avg by job` (между подами) автоматически показывает деградацию
  на обоих уровнях
- **Видно, какой именно endpoint упал** — из коробки, без
  дополнительных метрик
- **Каскадные отказы** — inhibition rules в Alertmanager

Основные риски (изменения в 300 сервисах, дрейф конфигурации,
отсутствие транзитивного анализа) митигируются через общую библиотеку,
CI/CD валидацию и поэтапное внедрение.
