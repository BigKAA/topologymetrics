// Package pgcheck provides a PostgreSQL health checker for dephealth.
//
// Import this package to register the PostgreSQL checker factory:
//
//	import _ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/pgcheck"
package pgcheck

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"strings"

	_ "github.com/jackc/pgx/v5/stdlib" // PostgreSQL driver

	"github.com/BigKAA/topologymetrics/sdk-go/dephealth"
)

func init() {
	dephealth.RegisterCheckerFactory(dephealth.TypePostgres, NewFromConfig)
}

// Option configures the Checker.
type Option func(*Checker)

// Checker performs health checks against a PostgreSQL database.
// Supports two modes:
//   - Standalone: creates a new connection per check using DSN built from endpoint
//   - Pool: uses an existing *sql.DB connection pool
type Checker struct {
	db    *sql.DB // nil = standalone, non-nil = pool mode
	dsn   string  // custom DSN for standalone mode (overrides endpoint-based DSN)
	query string  // health check query
}

// WithDB sets an existing connection pool for pool mode.
func WithDB(db *sql.DB) Option {
	return func(c *Checker) {
		c.db = db
	}
}

// WithDSN sets a custom DSN for standalone mode.
// If set, the endpoint host/port are ignored.
func WithDSN(dsn string) Option {
	return func(c *Checker) {
		c.dsn = dsn
	}
}

// WithQuery sets the health check SQL query (default "SELECT 1").
func WithQuery(query string) Option {
	return func(c *Checker) {
		c.query = query
	}
}

// New creates a new PostgreSQL health checker with the given options.
func New(opts ...Option) *Checker {
	c := &Checker{
		query: "SELECT 1",
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// NewFromConfig creates a PostgreSQL checker from DependencyConfig.
func NewFromConfig(dc *dephealth.DependencyConfig) dephealth.HealthChecker {
	var opts []Option
	if dc.URL != "" {
		opts = append(opts, WithDSN(dc.URL))
	}
	if dc.PostgresQuery != "" {
		opts = append(opts, WithQuery(dc.PostgresQuery))
	}
	return New(opts...)
}

// Check performs a health check against the PostgreSQL endpoint.
// In pool mode, uses the existing *sql.DB. In standalone mode, opens a new connection.
func (c *Checker) Check(ctx context.Context, endpoint dephealth.Endpoint) error {
	if c.db != nil {
		return c.checkPool(ctx)
	}
	return c.checkStandalone(ctx, endpoint)
}

func (c *Checker) checkPool(ctx context.Context) error {
	rows, err := c.db.QueryContext(ctx, c.query)
	if err != nil {
		return classifyError(err, "pool")
	}
	return rows.Close()
}

func (c *Checker) checkStandalone(ctx context.Context, endpoint dephealth.Endpoint) error {
	dsn := c.dsn
	if dsn == "" {
		dsn = fmt.Sprintf("postgres://%s/postgres", net.JoinHostPort(endpoint.Host, endpoint.Port))
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("postgres open %s: %w", endpoint.Host, err)
	}
	defer func() { _ = db.Close() }()

	rows, err := db.QueryContext(ctx, c.query)
	if err != nil {
		return classifyError(err, endpoint.Host)
	}
	return rows.Close()
}

// classifyError wraps PostgreSQL errors with appropriate classification.
// Detects auth errors via SQLSTATE codes 28000/28P01 in the error message.
func classifyError(err error, target string) error {
	msg := err.Error()
	if strings.Contains(msg, "28000") || strings.Contains(msg, "28P01") ||
		strings.Contains(msg, "password authentication failed") {
		return &dephealth.ClassifiedCheckError{
			Category: dephealth.StatusAuthError,
			Detail:   "auth_error",
			Cause:    fmt.Errorf("postgres %s: %w", target, err),
		}
	}
	return fmt.Errorf("postgres query %s: %w", target, err)
}

// Type returns the dependency type for this checker.
func (c *Checker) Type() string {
	return string(dephealth.TypePostgres)
}
