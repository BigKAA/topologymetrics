package checks

import (
	"context"
	"database/sql"
	"fmt"
	"net"

	_ "github.com/go-sql-driver/mysql" // MySQL driver

	"github.com/BigKAA/topologymetrics/sdk-go/dephealth"
)

// MySQLOption configures the MySQLChecker.
type MySQLOption func(*MySQLChecker)

// MySQLChecker performs health checks against a MySQL database.
// Supports two modes:
//   - Standalone: creates a new connection per check using DSN built from endpoint
//   - Pool: uses an existing *sql.DB connection pool
type MySQLChecker struct {
	db    *sql.DB // nil = standalone, non-nil = pool mode
	dsn   string  // custom DSN for standalone mode (overrides endpoint-based DSN)
	query string  // health check query
}

// WithMySQLDB sets an existing connection pool for pool mode.
func WithMySQLDB(db *sql.DB) MySQLOption {
	return func(c *MySQLChecker) {
		c.db = db
	}
}

// WithMySQLDSN sets a custom DSN for standalone mode.
// If set, the endpoint host/port are ignored.
func WithMySQLDSN(dsn string) MySQLOption {
	return func(c *MySQLChecker) {
		c.dsn = dsn
	}
}

// WithMySQLQuery sets the health check SQL query (default "SELECT 1").
func WithMySQLQuery(query string) MySQLOption {
	return func(c *MySQLChecker) {
		c.query = query
	}
}

// NewMySQLChecker creates a new MySQL health checker with the given options.
func NewMySQLChecker(opts ...MySQLOption) *MySQLChecker {
	c := &MySQLChecker{
		query: "SELECT 1",
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Check performs a health check against the MySQL endpoint.
// In pool mode, uses the existing *sql.DB. In standalone mode, opens a new connection.
func (c *MySQLChecker) Check(ctx context.Context, endpoint dephealth.Endpoint) error {
	if c.db != nil {
		return c.checkPool(ctx)
	}
	return c.checkStandalone(ctx, endpoint)
}

func (c *MySQLChecker) checkPool(ctx context.Context) error {
	rows, err := c.db.QueryContext(ctx, c.query)
	if err != nil {
		return fmt.Errorf("mysql pool query: %w", err)
	}
	return rows.Close()
}

func (c *MySQLChecker) checkStandalone(ctx context.Context, endpoint dephealth.Endpoint) error {
	dsn := c.dsn
	if dsn == "" {
		dsn = fmt.Sprintf("tcp(%s)/", net.JoinHostPort(endpoint.Host, endpoint.Port))
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("mysql open %s: %w", endpoint.Host, err)
	}
	defer func() { _ = db.Close() }()

	rows, err := db.QueryContext(ctx, c.query)
	if err != nil {
		return fmt.Errorf("mysql query %s: %w", endpoint.Host, err)
	}
	return rows.Close()
}

// Type returns the dependency type for this checker.
func (c *MySQLChecker) Type() string {
	return string(dephealth.TypeMySQL)
}
