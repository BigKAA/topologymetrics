// Package amqpcheck provides an AMQP health checker for dephealth.
//
// Import this package to register the AMQP checker factory:
//
//	import _ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/amqpcheck"
package amqpcheck

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/BigKAA/topologymetrics/sdk-go/dephealth"
)

func init() {
	dephealth.RegisterCheckerFactory(dephealth.TypeAMQP, NewFromConfig)
}

// Option configures the Checker.
type Option func(*Checker)

// Checker performs health checks by establishing an AMQP connection.
// Only standalone mode is supported: creates a new connection per check.
// The check succeeds if the AMQP connection can be established.
type Checker struct {
	url string // full AMQP URL (overrides endpoint-based URL)
}

// WithURL sets a custom AMQP URL for connections.
// If set, the endpoint host/port are ignored.
func WithURL(url string) Option {
	return func(c *Checker) {
		c.url = url
	}
}

// New creates a new AMQP health checker with the given options.
func New(opts ...Option) *Checker {
	c := &Checker{}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// NewFromConfig creates an AMQP checker from DependencyConfig.
func NewFromConfig(dc *dephealth.DependencyConfig) dephealth.HealthChecker {
	var opts []Option
	if dc.AMQPURL != "" {
		opts = append(opts, WithURL(dc.AMQPURL))
	} else if dc.URL != "" {
		opts = append(opts, WithURL(dc.URL))
	}
	return New(opts...)
}

// Check establishes an AMQP connection and immediately closes it.
// Uses amqp.DialConfig with a bounded dial timeout to prevent goroutine leaks
// when the remote server is unreachable.
func (c *Checker) Check(ctx context.Context, endpoint dephealth.Endpoint) error {
	url := c.url
	if url == "" {
		url = fmt.Sprintf("amqp://guest:guest@%s/", net.JoinHostPort(endpoint.Host, endpoint.Port))
	}

	// Derive dial timeout from context deadline to prevent goroutine leaks.
	dialTimeout := 5 * time.Second
	if deadline, ok := ctx.Deadline(); ok {
		if remaining := time.Until(deadline); remaining < dialTimeout {
			dialTimeout = remaining
		}
	}

	type dialResult struct {
		conn *amqp.Connection
		err  error
	}

	ch := make(chan dialResult, 1)
	go func() {
		conn, err := amqp.DialConfig(url, amqp.Config{
			Dial: func(network, addr string) (net.Conn, error) {
				return net.DialTimeout(network, addr, dialTimeout)
			},
		})
		ch <- dialResult{conn: conn, err: err}
	}()

	select {
	case <-ctx.Done():
		// The goroutine will finish within dialTimeout due to the bounded dialer.
		return fmt.Errorf("amqp dial %s: %w", endpoint.Host, ctx.Err())
	case res := <-ch:
		if res.err != nil {
			return classifyError(res.err, endpoint.Host)
		}
		_ = res.conn.Close()
		return nil
	}
}

// Type returns the dependency type for this checker.
func (c *Checker) Type() string {
	return string(dephealth.TypeAMQP)
}

// classifyError wraps AMQP errors with appropriate classification.
func classifyError(err error, host string) error {
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
