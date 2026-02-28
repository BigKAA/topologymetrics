package dephealth

import (
	"context"
	"errors"
)

var (
	// ErrTimeout indicates that a health check exceeded its deadline.
	ErrTimeout = errors.New("health check timeout")
	// ErrConnectionRefused indicates that the connection to the dependency was refused.
	ErrConnectionRefused = errors.New("connection refused")
	// ErrUnhealthy indicates that the dependency reported an unhealthy status.
	ErrUnhealthy = errors.New("dependency unhealthy")
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
