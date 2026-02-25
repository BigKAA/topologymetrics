# Java SDK Documentation

## Overview

dephealth Java SDK provides dependency health monitoring for Java
microservices via Prometheus metrics. It supports both programmatic API
and Spring Boot auto-configuration.

**Current version:** 0.8.0 | **Java:** 21 LTS | **Spring Boot:** 3.x

## Documentation

| Document | Description |
| --- | --- |
| [Getting Started](getting-started.md) | Installation, basic setup, and first example |
| [API Reference](api-reference.md) | Complete reference of all public classes and interfaces |
| [Configuration](configuration.md) | All options, defaults, and environment variables |
| [Health Checkers](checkers.md) | All 9 built-in checkers with examples |
| [Spring Boot Integration](spring-boot.md) | Auto-configuration, actuator, properties |
| [Prometheus Metrics](metrics.md) | Metrics reference and PromQL examples |
| [Authentication](authentication.md) | Auth options for HTTP, gRPC, and database checkers |
| [Connection Pools](connection-pools.md) | Integration with DataSource, JedisPool, LDAPConnection |
| [Troubleshooting](troubleshooting.md) | Common issues and solutions |
| [Migration Guide](migration.md) | Version upgrade instructions |
| [Code Style](code-style.md) | Java code conventions for this project |
| [Examples](examples/) | Complete runnable examples |

## Quick Links

- [Javadoc](../dephealth-core/target/site/apidocs/) (generated via `make docs`)
- [Specification](../../spec/) — cross-SDK metric contracts and behavior
- [Grafana Dashboards](../../docs/grafana-dashboards.md) — dashboard configuration
- [Alert Rules](../../docs/alerting/alert-rules.md) — alerting configuration
