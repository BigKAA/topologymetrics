// Package tcpcheck provides a TCP health checker for dephealth.
//
// Import this package to register the TCP checker factory:
//
//	import _ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/tcpcheck"
package tcpcheck

import (
	"context"
	"fmt"
	"net"

	"github.com/BigKAA/topologymetrics/sdk-go/dephealth"
)

func init() {
	dephealth.RegisterCheckerFactory(dephealth.TypeTCP, NewFromConfig)
}

// Checker performs health checks by establishing a TCP connection.
// The check succeeds if a TCP connection can be established within the
// context deadline. No data is sent or received.
type Checker struct{}

// New creates a new TCP health checker.
func New() *Checker {
	return &Checker{}
}

// NewFromConfig creates a TCP checker from DependencyConfig.
func NewFromConfig(_ *dephealth.DependencyConfig) dephealth.HealthChecker {
	return New()
}

// Check establishes a TCP connection to the endpoint and immediately closes it.
// Returns nil if the connection succeeds, or an error otherwise.
func (c *Checker) Check(ctx context.Context, endpoint dephealth.Endpoint) error {
	addr := net.JoinHostPort(endpoint.Host, endpoint.Port)

	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("tcp dial %s: %w", addr, err)
	}
	_ = conn.Close()

	return nil
}

// Type returns the dependency type for this checker.
func (c *Checker) Type() string {
	return string(dephealth.TypeTCP)
}
