*[Русская версия](grafana-dashboards.ru.md)*

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
