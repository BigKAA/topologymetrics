// Package redischeck provides a Redis health checker for dephealth.
//
// Import this package to register the Redis checker factory:
//
//	import _ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/redischeck"
package redischeck

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/BigKAA/topologymetrics/sdk-go/dephealth"
)

func init() {
	dephealth.RegisterCheckerFactory(dephealth.TypeRedis, NewFromConfig)
}

// Option configures the Checker.
type Option func(*Checker)

// Checker performs health checks against Redis using the PING command.
// Supports two modes:
//   - Standalone: creates a new redis client per check
//   - Pool: uses an existing redis.Cmdable (Client, ClusterClient, etc.)
type Checker struct {
	client   redis.Cmdable // nil = standalone, non-nil = pool mode
	password string        // password for standalone mode
	db       int           // database number for standalone mode
}

// WithClient sets an existing Redis client for pool mode.
func WithClient(client redis.Cmdable) Option {
	return func(c *Checker) {
		c.client = client
	}
}

// WithPassword sets the password for standalone mode connections.
func WithPassword(password string) Option {
	return func(c *Checker) {
		c.password = password
	}
}

// WithDB sets the database number for standalone mode connections.
func WithDB(db int) Option {
	return func(c *Checker) {
		c.db = db
	}
}

// New creates a new Redis health checker with the given options.
func New(opts ...Option) *Checker {
	c := &Checker{}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// NewFromConfig creates a Redis checker from DependencyConfig.
func NewFromConfig(dc *dephealth.DependencyConfig) dephealth.HealthChecker {
	var opts []Option
	if dc.RedisPassword != "" {
		opts = append(opts, WithPassword(dc.RedisPassword))
	}
	if dc.RedisDB != nil {
		opts = append(opts, WithDB(*dc.RedisDB))
	}
	// Extract password and db from URL if explicit options are not set.
	if dc.URL != "" {
		u, err := url.Parse(dc.URL)
		if err == nil && u != nil && u.User != nil && dc.RedisPassword == "" {
			if p, ok := u.User.Password(); ok {
				opts = append(opts, WithPassword(p))
			}
		}
		if err == nil && u != nil && dc.RedisDB == nil {
			dbStr := strings.TrimPrefix(u.Path, "/")
			if db, parseErr := strconv.Atoi(dbStr); parseErr == nil {
				opts = append(opts, WithDB(db))
			}
		}
	}
	return New(opts...)
}

// Check performs a PING command against the Redis endpoint.
// In pool mode, uses the existing client. In standalone mode, creates a new client.
func (c *Checker) Check(ctx context.Context, endpoint dephealth.Endpoint) error {
	if c.client != nil {
		return c.checkPool(ctx)
	}
	return c.checkStandalone(ctx, endpoint)
}

func (c *Checker) checkPool(ctx context.Context) error {
	if err := c.client.Ping(ctx).Err(); err != nil {
		return classifyError(err, "pool")
	}
	return nil
}

func (c *Checker) checkStandalone(ctx context.Context, endpoint dephealth.Endpoint) error {
	addr := net.JoinHostPort(endpoint.Host, endpoint.Port)
	client := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     c.password,
		DB:           c.db,
		MaxRetries:   0,               // Single attempt; retries are handled by the check scheduler.
		DialTimeout:  3 * time.Second, // Shorter than the check timeout to get a classifiable net error.
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})
	defer func() { _ = client.Close() }()

	if err := client.Ping(ctx).Err(); err != nil {
		return classifyError(err, addr)
	}
	return nil
}

// classifyError wraps Redis errors with appropriate classification.
func classifyError(err error, target string) error {
	msg := err.Error()

	// Auth errors.
	if strings.Contains(msg, "NOAUTH") || strings.Contains(msg, "WRONGPASS") {
		return &dephealth.ClassifiedCheckError{
			Category: dephealth.StatusAuthError,
			Detail:   "auth_error",
			Cause:    fmt.Errorf("redis %s: %w", target, err),
		}
	}

	// Connection refused — go-redis wraps net.OpError; detect via error chain.
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		if errors.Is(opErr.Err, syscall.ECONNREFUSED) {
			return &dephealth.ClassifiedCheckError{
				Category: dephealth.StatusConnectionError,
				Detail:   "connection_refused",
				Cause:    fmt.Errorf("redis %s: %w", target, err),
			}
		}
		// Dial/connect timeout (e.g., k8s service with no endpoints).
		if opErr.Timeout() {
			return &dephealth.ClassifiedCheckError{
				Category: dephealth.StatusConnectionError,
				Detail:   "connection_refused",
				Cause:    fmt.Errorf("redis %s: %w", target, err),
			}
		}
	}

	// Message-based fallback for connection refused.
	if strings.Contains(msg, "connection refused") {
		return &dephealth.ClassifiedCheckError{
			Category: dephealth.StatusConnectionError,
			Detail:   "connection_refused",
			Cause:    fmt.Errorf("redis %s: %w", target, err),
		}
	}

	// Context deadline exceeded — the check scheduler's timeout fired before
	// go-redis could complete the dial. In Kubernetes, when a Service has no
	// endpoints (e.g. scaled to 0), TCP SYN hangs until the context deadline.
	// This is effectively a connection error, not an application-level timeout.
	if errors.Is(err, context.DeadlineExceeded) {
		return &dephealth.ClassifiedCheckError{
			Category: dephealth.StatusConnectionError,
			Detail:   "connection_refused",
			Cause:    fmt.Errorf("redis %s: %w", target, err),
		}
	}

	return fmt.Errorf("redis ping %s: %w", target, err)
}

// Type returns the dependency type for this checker.
func (c *Checker) Type() string {
	return string(dephealth.TypeRedis)
}
