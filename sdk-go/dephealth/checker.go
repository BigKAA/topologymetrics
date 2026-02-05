package dephealth

import (
	"context"
	"errors"
)

// Sentinel errors for health check failures.
var (
	ErrTimeout           = errors.New("health check timeout")
	ErrConnectionRefused = errors.New("connection refused")
	ErrUnhealthy         = errors.New("dependency unhealthy")
)

// HealthChecker is the interface for dependency health checks.
// Each dependency type (HTTP, gRPC, TCP, Postgres, etc.) implements this interface.
type HealthChecker interface {
	// Check performs a health check against the given endpoint.
	// Returns nil if the endpoint is healthy, or an error describing the failure.
	// The context carries the timeout deadline.
	Check(ctx context.Context, endpoint Endpoint) error

	// Type returns the dependency type this checker handles (e.g. "http", "postgres").
	Type() string
}
