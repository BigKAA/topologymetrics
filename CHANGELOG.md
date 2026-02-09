# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Java SDK 0.2.2] - 2026-02-09

### Fixed

- **Java SDK**: разрешено создание экземпляра DepHealth без зависимостей (leaf-сервисы)

## [Python SDK 0.2.2] - 2026-02-09

### Fixed

- **Python SDK**: credentials из URL (userinfo) теперь передаются в checkers при автономной проверке

## [Go SDK 0.3.0] - 2026-02-09

### Breaking Changes

- **Go SDK**: module path изменён с `github.com/BigKAA/topologymetrics` на `github.com/BigKAA/topologymetrics/sdk-go` — стандартный подход для Go-модулей в монорепозиториях. Требуется обновление всех import paths. API и поведение SDK не изменились. Подробнее: [руководство по миграции](docs/migration/go.md#миграция-с-v02-на-v03).

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

[Java SDK 0.2.2]: https://github.com/BigKAA/topologymetrics/releases/tag/sdk-java/v0.2.2
[Python SDK 0.2.2]: https://github.com/BigKAA/topologymetrics/releases/tag/sdk-python/v0.2.2
[Go SDK 0.3.0]: https://github.com/BigKAA/topologymetrics/releases/tag/sdk-go/v0.3.0
[0.2.1]: https://github.com/BigKAA/topologymetrics/releases/tag/v0.2.1
[0.2.0]: https://github.com/BigKAA/topologymetrics/releases/tag/v0.2.0
[0.1.0]: https://github.com/BigKAA/topologymetrics/releases/tag/v0.1.0
