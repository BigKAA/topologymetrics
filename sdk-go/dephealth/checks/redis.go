package checks

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/BigKAA/topologymetrics/sdk-go/dephealth"
)

// RedisOption configures the RedisChecker.
type RedisOption func(*RedisChecker)

// RedisChecker performs health checks against Redis using the PING command.
// Supports two modes:
//   - Standalone: creates a new redis client per check
//   - Pool: uses an existing redis.Cmdable (Client, ClusterClient, etc.)
type RedisChecker struct {
	client   redis.Cmdable // nil = standalone, non-nil = pool mode
	password string        // password for standalone mode
	db       int           // database number for standalone mode
}

// WithRedisClient sets an existing Redis client for pool mode.
func WithRedisClient(client redis.Cmdable) RedisOption {
	return func(c *RedisChecker) {
		c.client = client
	}
}

// WithRedisPassword sets the password for standalone mode connections.
func WithRedisPassword(password string) RedisOption {
	return func(c *RedisChecker) {
		c.password = password
	}
}

// WithRedisDB sets the database number for standalone mode connections.
func WithRedisDB(db int) RedisOption {
	return func(c *RedisChecker) {
		c.db = db
	}
}

// NewRedisChecker creates a new Redis health checker with the given options.
func NewRedisChecker(opts ...RedisOption) *RedisChecker {
	c := &RedisChecker{}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Check performs a PING command against the Redis endpoint.
// In pool mode, uses the existing client. In standalone mode, creates a new client.
func (c *RedisChecker) Check(ctx context.Context, endpoint dephealth.Endpoint) error {
	if c.client != nil {
		return c.checkPool(ctx)
	}
	return c.checkStandalone(ctx, endpoint)
}

func (c *RedisChecker) checkPool(ctx context.Context) error {
	if err := c.client.Ping(ctx).Err(); err != nil {
		return classifyRedisError(err, "pool")
	}
	return nil
}

func (c *RedisChecker) checkStandalone(ctx context.Context, endpoint dephealth.Endpoint) error {
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
		return classifyRedisError(err, addr)
	}
	return nil
}

// classifyRedisError wraps Redis errors with appropriate classification.
func classifyRedisError(err error, target string) error {
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
func (c *RedisChecker) Type() string {
	return string(dephealth.TypeRedis)
}
