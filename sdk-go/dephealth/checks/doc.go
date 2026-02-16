// Package checks provides health checker implementations for various dependency types.
//
// Each checker implements the [dephealth.HealthChecker] interface and supports
// the functional options pattern for configuration.
//
// Supported dependency types:
//   - TCP: basic TCP connectivity check
//   - HTTP: HTTP endpoint health check with configurable path and TLS
//   - gRPC: gRPC Health Checking Protocol (grpc.health.v1)
//   - PostgreSQL: SQL query check (standalone or connection pool)
//   - MySQL: SQL query check (standalone or connection pool)
//   - Redis: PING command check (standalone or connection pool)
//   - AMQP: connection-level check (standalone only)
//   - Kafka: broker metadata request check (standalone only)
package checks

// Version is the SDK version used in User-Agent headers and diagnostics.
const Version = "0.4.2"
