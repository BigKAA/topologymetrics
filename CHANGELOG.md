# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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

[0.1.0]: https://github.com/BigKAA/topologymetrics/releases/tag/v0.1.0
