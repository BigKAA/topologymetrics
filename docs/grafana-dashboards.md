[English](#english) | [Русский](#russian)

---

<a id="english"></a>

# dephealth Grafana Dashboards

A set of 5 dashboards for monitoring the health of microservice dependencies. Dashboards are linked to each other via links and drill-down transitions, forming a unified navigation system from a general overview to the details of a specific connection.

## Overview

| Name | Description | UID | Path |
| --- | --- | --- | --- |
| Service List | Main overview: all services and their dependencies | `dephealth-service-list` | `/d/dephealth-service-list/` |
| Services Status | System state over time: timeline, charts, heatmap | `dephealth-services-status` | `/d/dephealth-services-status/` |
| Service Status | Detailed dependency status for a selected service | `dephealth-service-status` | `/d/dephealth-service-status/` |
| Links Status | Table of all connections with metrics | `dephealth-links-status` | `/d/dephealth-links-status/` |
| Link Status | Detailed information about a specific connection | `dephealth-link-status` | `/d/dephealth-link-status/` |

## Navigation Between Dashboards

```text
Service List (overview)
  |
  +---> Services Status (timeline of all services)
  |       |
  |       +---> Service Status (single service)
  |               |
  |               +---> Link Status (single connection)
  |
  +---> Links Status (table of all connections)
          |
          +---> Link Status (single connection)
```

Transitions are made via links in the dashboard header and clickable values in tables.

## 1. Service List

**Purpose**: main overview of all services and their dependencies. Entry point to the monitoring system.

![Service List](images/dashboard-service-list.png)

### Panels

- **OK / Degraded / Broken** (stat) -- three service counters by state:
  - OK (green) -- all service dependencies are available
  - Degraded (orange) -- some dependencies are unavailable
  - Broken (red) -- all dependencies are unavailable
- **Service state (timeline)** -- state-timeline of problematic services only over the selected period. Values: OK=2, Degraded=1, Broken=0. Clicking a service navigates to Service Status
- **Services and dependencies** (table) -- full list of dependencies with columns:
  - Service -- service name (link to Service Status)
  - Dependency -- dependency name (link to Link Status)
  - Type -- dependency type (http, grpc, postgres, redis, etc.)
  - Host, Port -- dependency address
  - Status -- UP (green) / DOWN (red)
  - P99 (sec) -- 99th percentile latency (color: green < 100ms, yellow < 1s, red >= 1s)
  - Flapping -- number of state changes in 15 minutes (color: green < 4, orange < 8, red >= 8)

### Filters

- **Datasource** -- Prometheus/VictoriaMetrics data source
- **Namespace** -- Kubernetes namespace filter (multi-select)
- **Type** -- dependency type filter (multi-select)
- **Dependency** -- dependency name filter (multi-select)
- **Host** -- host filter (multi-select)

### Navigation

- Header links: Services Status, Links Status
- Click on service in table: navigate to Service Status
- Click on dependency in table: navigate to Link Status

## 2. Services Status

**Purpose**: system state over a time period -- timeline of all services, health charts, heatmap, and P99 latency.

![Services Status](images/dashboard-services-status.png)

### Panels

- **OK / Degraded / Broken** (stat) -- service counters by state (same as Service List)
- **Service state (timeline)** -- state-timeline of problematic services. Clicking a service navigates to Service Status
- **Dependency health** (timeseries) -- app_dependency_health chart for each dependency (0/1). Click navigates to Link Status. Legend as table
- **Latency (heatmap)** -- heatmap of check latency distribution
- **P99 latency by service** (timeseries) -- 99th percentile latency by service. Thresholds: > 1s -- warning (yellow), > 5s -- critical (red). Click navigates to Service Status. Legend: Mean, Max, Last

### Filters

- **Datasource** -- data source
- **Namespace** -- Kubernetes namespace (multi-select)
- **Type** -- dependency type (multi-select)

### Navigation

- Header links: Service List, Links Status
- Click on service in timeline: navigate to Service Status
- Click on dependency in health chart: navigate to Link Status
- Click on service in P99 chart: navigate to Service Status

## 3. Service Status

**Purpose**: detailed dependency status for a selected service -- current state, SLA, endpoint table, timeline, and latency.

![Service Status](images/dashboard-service-status.png)

### Panels

- **Dependency status** (stat) -- current state of each service dependency: UP/DOWN with color indication. Click navigates to Link Status
- **SLA / Availability for period** (stat) -- availability percentage of each dependency for the selected period. Thresholds: < 95% -- red, 95-99% -- orange, >= 99% -- green
- **Endpoint table** (table) -- all service dependencies with columns: Dependency (link to Link Status), Type, Host, Port, Status, P99 (sec), Avg (sec), Flapping
- **app_dependency_health (timeline)** -- state-timeline of each dependency health (UP/DOWN). Click navigates to Link Status
- **Check latency (heatmap)** -- heatmap of latency distribution
- **P99 latency by dependency** (timeseries) -- 99th percentile by each dependency. Legend: Mean, Max, Last
- **Average latency by dependency** (timeseries) -- average latency by each dependency. Legend: Mean, Max, Last

### Filters

- **Datasource** -- data source
- **Service** -- specific service selection (single-select)

### Navigation

- Header links: Service List, Links Status
- Click on dependency stat panel: navigate to Link Status
- Click on dependency in table: navigate to Link Status
- Click on dependency in timeline: navigate to Link Status

## 4. Links Status

**Purpose**: unified table of all connections in the system with status, latency, and flapping metrics.

![Links Status](images/dashboard-links-status.png)

### Panels

- **All connections** (table) -- complete table of all connections with columns:
  - Dependency -- dependency name (link to Link Status)
  - Type -- dependency type
  - Host -- host address
  - Port -- port number
  - Service -- service name (link to Service Status)
  - Status -- UP (green) / DOWN (red)
  - P99 (sec) -- 99th percentile latency
  - Avg (sec) -- average latency
  - Flapping -- number of state changes in 15 minutes

Table is sorted by status (DOWN first), all columns are filterable.

### Filters

- **Datasource** -- data source
- **Namespace** -- Kubernetes namespace (multi-select)
- **Type** -- dependency type (multi-select)
- **Host** -- host filter (multi-select)

### Navigation

- Header links: Service List, Services Status
- Click on dependency: navigate to Link Status
- Click on service: navigate to Service Status

## 5. Link Status

**Purpose**: detailed information about a specific connection -- status, SLA, flapping, timeline, heatmap, and latency for each service using this connection.

![Link Status](images/dashboard-link-status.png)

### Panels

- **Current status** (stat) -- UP/DOWN for each service using this dependency
- **SLA / Availability** (stat) -- availability percentage for the selected period by each service. Thresholds: < 95% -- red, 95-99% -- orange, >= 99% -- green
- **Flapping (15 min)** (stat) -- number of state changes in 15 minutes by each service. Thresholds: < 4 -- green, 4-8 -- orange, >= 8 -- red
- **Connection health (timeline)** -- state-timeline UP/DOWN for each service
- **Latency (heatmap)** -- heatmap of check latency distribution
- **P99 latency** (timeseries) -- 99th percentile by each service. Thresholds: > 100ms -- yellow, > 1s -- red. Legend: Mean, Max, Last
- **Average latency** (timeseries) -- average latency by each service. Legend: Mean, Max, Last

### Annotations

State change annotations are enabled on this dashboard -- red markers on charts at the moments of UP/DOWN transitions.

### Filters

- **Datasource** -- data source
- **Dependency** -- dependency name (single-select)
- **Host** -- host address (single-select)
- **Port** -- port number (single-select)

### Navigation

- Header links: Service List, Links Status

## Deployment and Updates

Dashboards are shipped as part of the `dephealth-monitoring` Helm chart and are automatically provisioned into Grafana via ConfigMap.

### Updating Dashboards

```bash
# Update Helm release
helm upgrade dephealth-monitoring deploy/helm/dephealth-monitoring/ \
  -f deploy/helm/dephealth-monitoring/values-homelab.yaml \
  -n dephealth-monitoring

# Restart Grafana to apply updated ConfigMaps
kubectl rollout restart deployment/grafana -n dephealth-monitoring
```

### Access

- **URL**: `grafana.rootUrl` value from values (default `http://grafana.dephealth.local`)
- **Login**: `grafana.adminUser` value (default `admin`)
- **Password**: `grafana.adminPassword` value (default `dephealth`)

### Dashboard Source Files Location

```text
deploy/helm/dephealth-monitoring/dashboards/
  service-list.json
  services-status.json
  service-status.json
  links-status.json
  link-status.json
```

---

<a id="russian"></a>

# Grafana дашборды dephealth

Набор из 5 дашбордов для мониторинга состояния зависимостей микросервисов. Дашборды связаны между собой ссылками и drill-down переходами, образуя единую систему навигации от общего обзора до деталей конкретного соединения.

## Обзор

| Название | Описание | UID | Путь |
| --- | --- | --- | --- |
| Service List | Главный обзор: все сервисы и их зависимости | `dephealth-service-list` | `/d/dephealth-service-list/` |
| Services Status | Состояние системы во времени: timeline, графики, heatmap | `dephealth-services-status` | `/d/dephealth-services-status/` |
| Service Status | Детальный статус зависимостей выбранного сервиса | `dephealth-service-status` | `/d/dephealth-service-status/` |
| Links Status | Таблица всех соединений с метриками | `dephealth-links-status` | `/d/dephealth-links-status/` |
| Link Status | Подробная информация о конкретном соединении | `dephealth-link-status` | `/d/dephealth-link-status/` |

## Навигация между дашбордами

```text
Service List (обзор)
  |
  +---> Services Status (timeline всех сервисов)
  |       |
  |       +---> Service Status (один сервис)
  |               |
  |               +---> Link Status (одно соединение)
  |
  +---> Links Status (таблица всех соединений)
          |
          +---> Link Status (одно соединение)
```

Переходы осуществляются через ссылки в заголовке дашборда и кликабельные значения в таблицах.

## 1. Service List

**Назначение**: главный обзор всех сервисов и их зависимостей. Точка входа в систему мониторинга.

![Service List](images/dashboard-service-list.png)

### Панели

- **OK / Degraded / Broken** (stat) -- три счётчика сервисов по состоянию:
  - OK (зелёный) -- все зависимости сервиса доступны
  - Degraded (оранжевый) -- часть зависимостей недоступна
  - Broken (красный) -- все зависимости недоступны
- **Состояние сервисов (timeline)** -- state-timeline только проблемных сервисов за выбранный период. Значения: OK=2, Degraded=1, Broken=0. Клик по сервису ведёт на Service Status
- **Сервисы и зависимости** (таблица) -- полный список зависимостей с колонками:
  - Сервис -- имя сервиса (ссылка на Service Status)
  - Зависимость -- имя зависимости (ссылка на Link Status)
  - Тип -- тип зависимости (http, grpc, postgres, redis и т.д.)
  - Хост, Порт -- адрес зависимости
  - Статус -- UP (зелёный) / DOWN (красный)
  - P99 (сек) -- 99-й перцентиль латентности (цвет: зелёный < 100ms, жёлтый < 1s, красный >= 1s)
  - Flapping -- количество смен состояния за 15 минут (цвет: зелёный < 4, оранжевый < 8, красный >= 8)

### Фильтры

- **Datasource** -- источник данных Prometheus/VictoriaMetrics
- **Namespace** -- фильтр по Kubernetes namespace (multi-select)
- **Тип** -- фильтр по типу зависимости (multi-select)
- **Зависимость** -- фильтр по имени зависимости (multi-select)
- **Хост** -- фильтр по хосту (multi-select)

### Навигация

- Ссылки в заголовке: Services Status, Links Status
- Клик по сервису в таблице: переход на Service Status
- Клик по зависимости в таблице: переход на Link Status

## 2. Services Status

**Назначение**: состояние всей системы на промежутке времени -- timeline всех сервисов, графики здоровья, heatmap и P99 латентности.

![Services Status](images/dashboard-services-status.png)

### Панели

- **OK / Degraded / Broken** (stat) -- счётчики сервисов по состоянию (аналогично Service List)
- **Состояние сервисов (timeline)** -- state-timeline проблемных сервисов. Клик по сервису ведёт на Service Status
- **Здоровье зависимостей** (timeseries) -- график app_dependency_health по каждой зависимости (0/1). Клик ведёт на Link Status. Легенда в виде таблицы
- **Латентность (heatmap)** -- тепловая карта распределения латентности проверок
- **P99 латентность по сервисам** (timeseries) -- 99-й перцентиль латентности по сервисам. Пороги: > 1s -- warning (жёлтый), > 5s -- critical (красный). Клик ведёт на Service Status. Легенда: Mean, Max, Last

### Фильтры

- **Datasource** -- источник данных
- **Namespace** -- Kubernetes namespace (multi-select)
- **Тип** -- тип зависимости (multi-select)

### Навигация

- Ссылки в заголовке: Service List, Links Status
- Клик по сервису в timeline: переход на Service Status
- Клик по зависимости в графике здоровья: переход на Link Status
- Клик по сервису в P99 графике: переход на Service Status

## 3. Service Status

**Назначение**: детальный статус зависимостей выбранного сервиса -- текущее состояние, SLA, таблица endpoint-ов, timeline и латентность.

![Service Status](images/dashboard-service-status.png)

### Панели

- **Статус зависимостей** (stat) -- текущее состояние каждой зависимости сервиса: UP/DOWN с цветовой индикацией. Клик ведёт на Link Status
- **SLA / Доступность за период** (stat) -- процент доступности каждой зависимости за выбранный период. Пороги: < 95% -- красный, 95-99% -- оранжевый, >= 99% -- зелёный
- **Таблица endpoint-ов** (table) -- все зависимости сервиса с колонками: Зависимость (ссылка на Link Status), Тип, Хост, Порт, Статус, P99 (сек), Avg (сек), Flapping
- **app_dependency_health (timeline)** -- state-timeline здоровья каждой зависимости (UP/DOWN). Клик ведёт на Link Status
- **Латентность проверок (heatmap)** -- тепловая карта распределения латентности
- **P99 латентность по зависимости** (timeseries) -- 99-й перцентиль по каждой зависимости. Легенда: Mean, Max, Last
- **Средняя латентность по зависимости** (timeseries) -- средняя латентность по каждой зависимости. Легенда: Mean, Max, Last

### Фильтры

- **Datasource** -- источник данных
- **Сервис** -- выбор конкретного сервиса (single-select)

### Навигация

- Ссылки в заголовке: Service List, Links Status
- Клик по stat-панели зависимости: переход на Link Status
- Клик по зависимости в таблице: переход на Link Status
- Клик по зависимости в timeline: переход на Link Status

## 4. Links Status

**Назначение**: единая таблица всех соединений в системе с метриками статуса, латентности и flapping.

![Links Status](images/dashboard-links-status.png)

### Панели

- **Все соединения** (table) -- полная таблица всех соединений с колонками:
  - Зависимость -- имя зависимости (ссылка на Link Status)
  - Тип -- тип зависимости
  - Хост -- адрес хоста
  - Порт -- номер порта
  - Сервис -- имя сервиса (ссылка на Service Status)
  - Статус -- UP (зелёный) / DOWN (красный)
  - P99 (сек) -- 99-й перцентиль латентности
  - Avg (сек) -- средняя латентность
  - Flapping -- количество смен состояния за 15 минут

Таблица отсортирована по статусу (DOWN сверху), все колонки фильтруемые.

### Фильтры

- **Datasource** -- источник данных
- **Namespace** -- Kubernetes namespace (multi-select)
- **Тип** -- тип зависимости (multi-select)
- **Хост** -- фильтр по хосту (multi-select)

### Навигация

- Ссылки в заголовке: Service List, Services Status
- Клик по зависимости: переход на Link Status
- Клик по сервису: переход на Service Status

## 5. Link Status

**Назначение**: подробная информация о конкретном соединении -- статус, SLA, flapping, timeline, heatmap и латентность по каждому сервису, использующему это соединение.

![Link Status](images/dashboard-link-status.png)

### Панели

- **Текущий статус** (stat) -- UP/DOWN по каждому сервису, использующему эту зависимость
- **SLA / Доступность** (stat) -- процент доступности за выбранный период по каждому сервису. Пороги: < 95% -- красный, 95-99% -- оранжевый, >= 99% -- зелёный
- **Flapping (15 мин)** (stat) -- количество смен состояния за 15 минут по каждому сервису. Пороги: < 4 -- зелёный, 4-8 -- оранжевый, >= 8 -- красный
- **Здоровье соединения (timeline)** -- state-timeline UP/DOWN по каждому сервису
- **Латентность (heatmap)** -- тепловая карта распределения латентности проверок
- **P99 латентность** (timeseries) -- 99-й перцентиль по каждому сервису. Пороги: > 100ms -- жёлтый, > 1s -- красный. Легенда: Mean, Max, Last
- **Средняя латентность** (timeseries) -- средняя латентность по каждому сервису. Легенда: Mean, Max, Last

### Аннотации

На дашборде включены аннотации изменения состояния -- красные маркеры на графиках в моменты смены UP/DOWN.

### Фильтры

- **Datasource** -- источник данных
- **Зависимость** -- имя зависимости (single-select)
- **Хост** -- адрес хоста (single-select)
- **Порт** -- номер порта (single-select)

### Навигация

- Ссылки в заголовке: Service List, Links Status

## Деплой и обновление

Дашборды поставляются как часть Helm chart `dephealth-monitoring` и автоматически провижонятся в Grafana через ConfigMap.

### Обновление дашбордов

```bash
# Обновить Helm release
helm upgrade dephealth-monitoring deploy/helm/dephealth-monitoring/ \
  -f deploy/helm/dephealth-monitoring/values-homelab.yaml \
  -n dephealth-monitoring

# Перезапустить Grafana для применения обновлённых ConfigMap
kubectl rollout restart deployment/grafana -n dephealth-monitoring
```

### Доступ

- **URL**: значение `grafana.rootUrl` из values (по умолчанию `http://grafana.dephealth.local`)
- **Логин**: значение `grafana.adminUser` (по умолчанию `admin`)
- **Пароль**: значение `grafana.adminPassword` (по умолчанию `dephealth`)

### Расположение исходников дашбордов

```text
deploy/helm/dephealth-monitoring/dashboards/
  service-list.json
  services-status.json
  service-status.json
  links-status.json
  link-status.json
```
