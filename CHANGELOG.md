# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.6.0] - 2026-02-22 (Go SDK only)

Split the `checks` package into individual sub-packages so users can import
only the checkers they need, reducing binary size and build time.

### Added

- **Go SDK**: Selective checker imports via sub-packages (`httpcheck`, `pgcheck`,
  `grpccheck`, `tcpcheck`, `mysqlcheck`, `redischeck`, `amqpcheck`, `kafkacheck`)
- **Go SDK**: Each sub-package self-registers its factory via `init()`
- **Go SDK**: Backward-compatible type aliases and constructor wrappers in
  `checks/compat.go` (marked `Deprecated`)
- Migration guide from v0.5.0 to v0.6.0

### Changed

- **Go SDK**: `checks` package now re-exports all sub-packages via blank imports
- **Go SDK**: `Version` constant moved from `checks/doc.go` to `dephealth/version.go`
- **Go SDK**: `contrib/sqldb` and `contrib/redispool` updated to use sub-packages
- **Go SDK**: Error message for missing checker factory now mentions both import styles

## [0.5.0] - 2026-02-18

Mandatory `group` label: logical grouping of services (team, subsystem, project),
independent of Kubernetes namespace. Enables dephealth-ui to group services by
logical membership rather than physical location.

### Breaking Changes

- **All SDKs**: Added mandatory `group` parameter to service configuration
  - Go: `New(name, group, opts...)` (was `New(name, opts...)`)
  - Java: `builder(name, group, registry)` (was `builder(name, registry)`)
  - Python: `DependencyHealth(name, group, *specs)` (was `DependencyHealth(name, *specs)`)
  - C#: `CreateBuilder(name, group)` (was `CreateBuilder(name)`)
- **All SDKs**: `group` label added to all Prometheus metrics
  - Label order: `name, group, dependency, type, host, port, critical, [custom]`
- **All SDKs**: Missing `group` causes a fail-fast error with clear message
  referencing both API parameter and `DEPHEALTH_GROUP` environment variable

### Added

- `group` label in all four metrics: `app_dependency_health`,
  `app_dependency_latency_seconds`, `app_dependency_status`,
  `app_dependency_status_detail`
- `DEPHEALTH_GROUP` environment variable (same precedence as `DEPHEALTH_NAME`:
  API > env var)
- `group` validation: same rules as `name` — `[a-z][a-z0-9-]*`, 1-63 chars
- Migration guide from v0.4.2 to v0.5.0
- Conformance scenario `group-label` validating `group` label presence and value
- Java Spring Boot: `dephealth.group` property in application.yml

## [0.4.2] - 2026-02-16

Authentication support for HTTP and gRPC health checkers across all SDKs.
Per-dependency auth configuration with Bearer token, Basic Auth, and
custom headers/metadata.

### Added

#### All SDKs

- HTTP checker: Bearer token (`WithHTTPBearerToken` / `http_bearer_token` /
  `HttpBearerToken`), Basic Auth (`WithHTTPBasicAuth` / `http_basic_auth` /
  `HttpBasicAuth`), custom headers (`WithHTTPHeaders` / `headers` /
  `HttpHeaders`).
- gRPC checker: Bearer token (`WithGRPCBearerToken` / `grpc_bearer_token` /
  `GrpcBearerToken`), custom metadata (`WithGRPCMetadata` / `metadata` /
  `GrpcMetadata`).
- Conflict validation: only one auth method per dependency (Bearer OR
  Basic Auth OR custom Authorization header). Error at creation time.
- Auth error classification: HTTP 401/403 and gRPC UNAUTHENTICATED/
  PERMISSION_DENIED mapped to `status="auth_error"`, `detail="auth_error"`.

#### Java Spring Boot Starter

- YAML properties: `http-bearer-token`, `http-basic-username`,
  `http-basic-password`, `http-headers`, `grpc-bearer-token`,
  `grpc-metadata` for per-dependency auth configuration.

#### Specification

- `spec/check-behavior.md` sections 4.1 and 4.2: auth parameters,
  validation rules, error classification for HTTP and gRPC checkers.
- `spec/config-contract.md`: auth configuration examples.

#### Conformance

- 4 new auth scenarios: `auth-http-bearer`, `auth-http-basic`,
  `auth-http-header`, `auth-grpc` (total: 13 scenarios).
- HTTP and gRPC test stubs with dynamic auth configuration via admin API.
- All 4 SDKs pass 13/13 scenarios (325 checks each).

## [0.4.1] - 2026-02-15

Programmatic health details: new `HealthDetails()` API across all SDKs,
returning detailed endpoint status (category, detail, latency, labels)
without relying on Prometheus metrics parsing.

### Added

#### All SDKs

- `HealthDetails()` / `healthDetails()` / `health_details()` method returning
  `EndpointStatus` for each monitored endpoint with 11 fields: dependency,
  type, host, port, healthy, status, detail, latency, last_checked_at,
  critical, labels.
- `EndpointStatus` type: immutable value object with JSON serialization
  support (Go: custom MarshalJSON, Java: immutable class, Python: frozen
  dataclass with `to_dict()`, C#: sealed class with System.Text.Json).
- `StatusCategory.Unknown` / `STATUS_UNKNOWN` — new status for endpoints
  that have not been checked yet.

#### Specification

- `spec/check-behavior.md` section 8: Programmatic Health Details API contract

#### Conformance

- New `health-details` scenario (37 checks per SDK): endpoint exists,
  structure, types, consistency with /metrics, status values, expected values.
- All 4 SDKs pass 9/9 scenarios (36 tests total).

## [0.4.0] - 2026-02-14

Status metrics: two new Prometheus metrics for diagnosing **why** a dependency
is unavailable — without logs, directly from metrics. All SDKs unified at
version 0.4.0.

### Added

#### All SDKs

- `app_dependency_status` gauge (enum pattern) — 8 status categories per endpoint:
  `ok`, `timeout`, `connection_error`, `dns_error`, `auth_error`, `tls_error`,
  `unhealthy`, `error`. All 8 series always exported; exactly one = 1, rest = 0.
  No series churn on state changes.
- `app_dependency_status_detail` gauge (info pattern) — detailed reason per
  endpoint (e.g. `http_503`, `grpc_not_serving`, `auth_error`, `no_brokers`).
  Old series deleted on change (acceptable churn).
- Error classification architecture: `ClassifiedError` interface (Go),
  `status_category`/`detail` on `CheckError` (Python), typed `CheckException`
  hierarchy (Java), virtual properties on `DepHealthException` (C#).
- Typed exceptions for DNS, TLS, and auth errors in all checkers.
- `ErrorClassifier` — platform-level error detection (DNS, TLS, timeout,
  connection refused) as fallback for unclassified errors.

#### Specification

- `spec/metric-contract.md` sections 8-9: full contract for status and detail metrics
- `spec/check-behavior.md` section 6.2: error classification rules

### Changed

- All SDK versions unified at 0.4.0 (previously Go 0.3.1, Java 0.2.2,
  Python 0.2.2, C# 0.2.1)
- C# license corrected from MIT to Apache 2.0 in Directory.Build.props

## [Java SDK 0.2.2] - 2026-02-09

### Fixed

- **Java SDK**: разрешено создание экземпляра DepHealth без зависимостей (leaf-сервисы)

## [Python SDK 0.2.2] - 2026-02-09

### Fixed

- **Python SDK**: credentials из URL (userinfo) теперь передаются в checkers при автономной проверке

## [Go SDK 0.3.0] - 2026-02-09

### Breaking Changes

- **Go SDK**: module path изменён с `github.com/BigKAA/topologymetrics` на `github.com/BigKAA/topologymetrics/sdk-go` — стандартный подход для Go-модулей в монорепозиториях. Требуется обновление всех import paths. API и поведение SDK не изменились. Подробнее: [руководство по миграции](docs/migration/go.ru.md#миграция-с-v02-на-v03).

## [0.2.1] - 2026-02-09

### Fixed

- **Go SDK**: HTTP-чекер теперь следует HTTP-редиректам (3xx) вместо ошибки
- **Java SDK**: HTTP-чекер теперь следует HTTP-редиректам (3xx) вместо ошибки
- **Java SDK**: обновлён User-Agent с 0.1.0 до 0.2.1
- **C# SDK**: обновлён User-Agent с 0.1.0 до 0.2.1
- **Go SDK**: обновлена константа Version с 0.1.0 до 0.2.1
- **Спецификация**: обновлена таблица edge cases — редиректы следуются, ожидается финальный 2xx

### Changed

- **Документация**: обновлены скриншоты Grafana дашбордов

## [0.2.0] - 2026-02-09

Dependency topology: обязательная идентификация приложений (`name`), критичность
зависимостей (`critical`), произвольные метки (`WithLabel`). Эти изменения позволяют
строить полный граф зависимостей микросервисов и фильтровать по критичности.

### Breaking Changes

- **Все SDK**: первый параметр `name` при создании экземпляра DepHealth стал обязательным
- **Все SDK**: параметр `critical` для каждой зависимости стал обязательным (без значения по умолчанию)
- **Go SDK**: `Endpoint.Metadata` переименован в `Endpoint.Labels`
- **Go SDK**: удалён `WithOptionalLabels()`, `allowedOptionalLabels`
- **Метрики**: добавлены обязательные метки `name` и `critical` — порядок меток изменился

### Added

#### Спецификация

- `spec/metric-contract.md` v2.0-draft: обязательные метки `name`, `critical`, произвольные `WithLabel`
- `spec/config-contract.md` v2.0-draft: `DEPHEALTH_NAME`, `DEPHEALTH_<DEP>_CRITICAL`, `DEPHEALTH_<DEP>_LABEL_<KEY>`

#### Go SDK

- `New(name, ...)`: обязательный `name` приложения
- `Critical(bool)`: обязательная критичность зависимости
- `WithLabel(key, value)`: произвольные метки
- Env vars: `DEPHEALTH_NAME`, `DEPHEALTH_<DEP>_CRITICAL`, `DEPHEALTH_<DEP>_LABEL_<KEY>`

#### Java SDK

- `DepHealth.builder(name, registry)`: обязательный `name`
- `.critical(bool)`: обязательная критичность
- `.label(key, value)`: произвольные метки
- Spring Boot: `dephealth.name` в application.yml, `critical` и `labels` для зависимостей
- Env vars: `DEPHEALTH_NAME`, `DEPHEALTH_<DEP>_CRITICAL`, `DEPHEALTH_<DEP>_LABEL_<KEY>`

#### Python SDK

- `DependencyHealth(name, ...)`: обязательный `name`
- `critical` обязателен для всех фабрик
- `labels={"key": "value"}`: произвольные метки
- `dephealth_lifespan(name, ...)`: обязательный `name`
- Env vars: `DEPHEALTH_NAME`, `DEPHEALTH_<DEP>_CRITICAL`, `DEPHEALTH_<DEP>_LABEL_<KEY>`

#### C# SDK

- `CreateBuilder(name)` / `AddDepHealth(name, ...)`: обязательный `name`
- `.Critical(bool)`: обязательная критичность
- `.Label(key, value)`: произвольные метки
- Env vars: `DEPHEALTH_NAME`, `DEPHEALTH_<DEP>_CRITICAL`, `DEPHEALTH_<DEP>_LABEL_<KEY>`

#### Conformance

- Обновлены все 8 сценариев: проверка `name`, `critical`, custom labels
- Обновлены все conformance-сервисы (Go, Java, Python, C#)

## [0.1.0] - 2026-02-07

First public release. Four native SDKs sharing a common specification, with conformance tests
verifying cross-language compatibility.

### Added

#### Go SDK (`sdk-go/`)

- Core abstractions: `Dependency`, `Endpoint`, `HealthChecker` interface
- Connection config parser: URL, connection string, JDBC, explicit params
- 8 health checkers: HTTP, gRPC, TCP, PostgreSQL, MySQL, Redis, AMQP, Kafka
- Check scheduler with configurable interval, timeout, failure/success thresholds
- Prometheus metrics exporter: `app_dependency_health` (Gauge) and `app_dependency_latency_seconds` (Histogram)
- Public API with functional options pattern (`WithDependency`, `WithChecker`, `WithInterval`, etc.)
- contrib/ packages: `pgxcheck`, `redischeck`, `sqlcheck` for connection pool integration
- Test service (`test-services/go-service/`) with 4 dependencies

#### Python SDK (`sdk-python/`)

- Async architecture built on `asyncio`
- 8 health checkers matching Go SDK specification
- Connection pool support (asyncpg, redis, aiomysql)
- FastAPI integration: `DepHealthMiddleware`, `dephealth_lifespan`, `/health/dependencies` endpoint
- Prometheus metrics via `prometheus-client`
- Optional dependencies via extras: `dephealth[postgres,redis,fastapi,all]`
- 79 unit tests
- Test service (`test-services/python-service/`) with FastAPI

#### Java SDK (`sdk-java/`)

- Java 21 LTS, Maven multi-module project
- Core module (`dephealth-core`): model, parser, 8 checkers, Micrometer metrics, scheduler
- Spring Boot starter (`dephealth-spring-boot-starter`): auto-configuration, actuator integration
- URL credential extraction for simplified configuration
- 151 unit tests (JUnit 5 + Mockito)
- Lint: Checkstyle (Google-based) + SpotBugs
- Test service (`test-services/java-service/`) with Spring Boot

#### C# SDK (`sdk-csharp/`)

- .NET 8 LTS, C# 12
- Core library (`DepHealth.Core`): model, parser, 8 checkers, prometheus-net metrics, scheduler
- ASP.NET Core integration (`DepHealth.AspNetCore`): hosted service, health checks, middleware
- Entity Framework integration (`DepHealth.EntityFramework`)
- 97 unit tests (xUnit + Moq)
- Test service (`test-services/csharp-service/`) with ASP.NET Core

#### Conformance Testing

- YAML-based test scenarios: 8 scenarios covering all dependency types and edge cases
- Conformance runner (`conformance/runner/`): automated verification with Prometheus metrics parsing
- Cross-language verification script (`cross_verify.py`)
- Language-specific conformance services (Go, Python, Java, C#)
- Result: 4 SDKs x 8 scenarios = 32/32 passed

#### Infrastructure

- Helm charts: `dephealth-infra`, `dephealth-services`, `dephealth-monitoring`, `dephealth-conformance`
- Docker-based development: Makefiles for build, test, lint, format per SDK
- docker-compose for local development environment
- Grafana dashboards (5 dashboards) for dependency monitoring
- VictoriaMetrics + Alertmanager alerting rules
- Container registry: Harbor with proxy cache configuration

#### Documentation

- Specification (`spec/`): metric contract, behavior, configuration
- Quickstart guides for Go, Python, Java, C#
- Migration guide for Go
- SDK comparison table
- CONTRIBUTING.md with development workflow

[0.5.0]: https://github.com/BigKAA/topologymetrics/releases/tag/v0.5.0
[0.4.2]: https://github.com/BigKAA/topologymetrics/releases/tag/v0.4.2
[0.4.1]: https://github.com/BigKAA/topologymetrics/releases/tag/v0.4.1
[0.4.0]: https://github.com/BigKAA/topologymetrics/releases/tag/v0.4.0
[Java SDK 0.2.2]: https://github.com/BigKAA/topologymetrics/releases/tag/sdk-java/v0.2.2
[Python SDK 0.2.2]: https://github.com/BigKAA/topologymetrics/releases/tag/sdk-python/v0.2.2
[Go SDK 0.3.0]: https://github.com/BigKAA/topologymetrics/releases/tag/sdk-go/v0.3.0
[0.2.1]: https://github.com/BigKAA/topologymetrics/releases/tag/v0.2.1
[0.2.0]: https://github.com/BigKAA/topologymetrics/releases/tag/v0.2.0
[0.1.0]: https://github.com/BigKAA/topologymetrics/releases/tag/v0.1.0
