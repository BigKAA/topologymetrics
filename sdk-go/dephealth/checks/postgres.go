package checks

import (
	"context"
	"database/sql"
	"fmt"
	"net"

	_ "github.com/jackc/pgx/v5/stdlib" // PostgreSQL driver

	"github.com/company/dephealth/dephealth"
)

// PostgresOption configures the PostgresChecker.
type PostgresOption func(*PostgresChecker)

// PostgresChecker performs health checks against a PostgreSQL database.
// Supports two modes:
//   - Standalone: creates a new connection per check using DSN built from endpoint
//   - Pool: uses an existing *sql.DB connection pool
type PostgresChecker struct {
	db    *sql.DB // nil = standalone, non-nil = pool mode
	dsn   string  // custom DSN for standalone mode (overrides endpoint-based DSN)
	query string  // health check query
}

// WithPostgresDB sets an existing connection pool for pool mode.
func WithPostgresDB(db *sql.DB) PostgresOption {
	return func(c *PostgresChecker) {
		c.db = db
	}
}

// WithPostgresDSN sets a custom DSN for standalone mode.
// If set, the endpoint host/port are ignored.
func WithPostgresDSN(dsn string) PostgresOption {
	return func(c *PostgresChecker) {
		c.dsn = dsn
	}
}

// WithPostgresQuery sets the health check SQL query (default "SELECT 1").
func WithPostgresQuery(query string) PostgresOption {
	return func(c *PostgresChecker) {
		c.query = query
	}
}

// NewPostgresChecker creates a new PostgreSQL health checker with the given options.
func NewPostgresChecker(opts ...PostgresOption) *PostgresChecker {
	c := &PostgresChecker{
		query: "SELECT 1",
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Check performs a health check against the PostgreSQL endpoint.
// In pool mode, uses the existing *sql.DB. In standalone mode, opens a new connection.
func (c *PostgresChecker) Check(ctx context.Context, endpoint dephealth.Endpoint) error {
	if c.db != nil {
		return c.checkPool(ctx)
	}
	return c.checkStandalone(ctx, endpoint)
}

func (c *PostgresChecker) checkPool(ctx context.Context) error {
	rows, err := c.db.QueryContext(ctx, c.query)
	if err != nil {
		return fmt.Errorf("postgres pool query: %w", err)
	}
	return rows.Close()
}

func (c *PostgresChecker) checkStandalone(ctx context.Context, endpoint dephealth.Endpoint) error {
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
		return fmt.Errorf("postgres query %s: %w", endpoint.Host, err)
	}
	return rows.Close()
}

// Type returns the dependency type for this checker.
func (c *PostgresChecker) Type() string {
	return string(dephealth.TypePostgres)
}
