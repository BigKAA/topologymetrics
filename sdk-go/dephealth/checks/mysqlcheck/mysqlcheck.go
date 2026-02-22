// Package mysqlcheck provides a MySQL health checker for dephealth.
//
// Import this package to register the MySQL checker factory:
//
//	import _ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/mysqlcheck"
package mysqlcheck

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"net/url"
	"strings"

	_ "github.com/go-sql-driver/mysql" // MySQL driver

	"github.com/BigKAA/topologymetrics/sdk-go/dephealth"
)

func init() {
	dephealth.RegisterCheckerFactory(dephealth.TypeMySQL, NewFromConfig)
}

// Option configures the Checker.
type Option func(*Checker)

// Checker performs health checks against a MySQL database.
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

// New creates a new MySQL health checker with the given options.
func New(opts ...Option) *Checker {
	c := &Checker{
		query: "SELECT 1",
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// NewFromConfig creates a MySQL checker from DependencyConfig.
func NewFromConfig(dc *dephealth.DependencyConfig) dephealth.HealthChecker {
	var opts []Option
	if dc.URL != "" {
		if dsn := URLToDSN(dc.URL); dsn != "" {
			opts = append(opts, WithDSN(dsn))
		}
	}
	if dc.MySQLQuery != "" {
		opts = append(opts, WithQuery(dc.MySQLQuery))
	}
	return New(opts...)
}

// Check performs a health check against the MySQL endpoint.
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
		dsn = fmt.Sprintf("tcp(%s)/", net.JoinHostPort(endpoint.Host, endpoint.Port))
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("mysql open %s: %w", endpoint.Host, err)
	}
	defer func() { _ = db.Close() }()

	rows, err := db.QueryContext(ctx, c.query)
	if err != nil {
		return classifyError(err, endpoint.Host)
	}
	return rows.Close()
}

// classifyError wraps MySQL errors with appropriate classification.
// Detects auth errors via MySQL error code 1045 (Access denied).
func classifyError(err error, target string) error {
	msg := err.Error()
	if strings.Contains(msg, "1045") || strings.Contains(msg, "Access denied") {
		return &dephealth.ClassifiedCheckError{
			Category: dephealth.StatusAuthError,
			Detail:   "auth_error",
			Cause:    fmt.Errorf("mysql %s: %w", target, err),
		}
	}
	return fmt.Errorf("mysql query %s: %w", target, err)
}

// Type returns the dependency type for this checker.
func (c *Checker) Type() string {
	return string(dephealth.TypeMySQL)
}

// URLToDSN converts a mysql:// URL to the go-sql-driver/mysql DSN format.
// mysql://user:pass@host:3306/db → user:pass@tcp(host:3306)/db
// Returns empty string if the URL cannot be parsed.
func URLToDSN(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}

	var userinfo string
	if u.User != nil {
		userinfo = u.User.String()
	}

	host := u.Host // includes port if specified

	path := u.Path // e.g. "/db"

	dsn := userinfo + "@tcp(" + host + ")" + path
	if q := u.RawQuery; q != "" {
		dsn += "?" + q
	}
	return dsn
}
