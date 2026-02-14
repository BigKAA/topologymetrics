package checks

import (
	"context"
	"fmt"
	"net"
	"strings"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/BigKAA/topologymetrics/sdk-go/dephealth"
)

// AMQPOption configures the AMQPChecker.
type AMQPOption func(*AMQPChecker)

// AMQPChecker performs health checks by establishing an AMQP connection.
// Only standalone mode is supported: creates a new connection per check.
// The check succeeds if the AMQP connection can be established.
type AMQPChecker struct {
	url string // full AMQP URL (overrides endpoint-based URL)
}

// WithAMQPURL sets a custom AMQP URL for connections.
// If set, the endpoint host/port are ignored.
func WithAMQPURL(url string) AMQPOption {
	return func(c *AMQPChecker) {
		c.url = url
	}
}

// NewAMQPChecker creates a new AMQP health checker with the given options.
func NewAMQPChecker(opts ...AMQPOption) *AMQPChecker {
	c := &AMQPChecker{}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Check establishes an AMQP connection and immediately closes it.
// Uses context for cancellation/timeout via a goroutine wrapper
// since amqp091-go does not natively support context.
func (c *AMQPChecker) Check(ctx context.Context, endpoint dephealth.Endpoint) error {
	url := c.url
	if url == "" {
		url = fmt.Sprintf("amqp://guest:guest@%s/", net.JoinHostPort(endpoint.Host, endpoint.Port))
	}

	type dialResult struct {
		conn *amqp.Connection
		err  error
	}

	ch := make(chan dialResult, 1)
	go func() {
		conn, err := amqp.Dial(url)
		ch <- dialResult{conn: conn, err: err}
	}()

	select {
	case <-ctx.Done():
		return fmt.Errorf("amqp dial %s: %w", endpoint.Host, ctx.Err())
	case res := <-ch:
		if res.err != nil {
			return classifyAMQPError(res.err, endpoint.Host)
		}
		_ = res.conn.Close()
		return nil
	}
}

// Type returns the dependency type for this checker.
func (c *AMQPChecker) Type() string {
	return string(dephealth.TypeAMQP)
}

// classifyAMQPError wraps AMQP errors with appropriate classification.
func classifyAMQPError(err error, host string) error {
	msg := err.Error()
	// AMQP 403 ACCESS_REFUSED indicates authentication/authorization failure.
	if strings.Contains(msg, "403") || strings.Contains(msg, "ACCESS_REFUSED") {
		return &dephealth.ClassifiedCheckError{
			Category: dephealth.StatusAuthError,
			Detail:   "auth_error",
			Cause:    fmt.Errorf("amqp dial %s: %w", host, err),
		}
	}
	return fmt.Errorf("amqp dial %s: %w", host, err)
}
