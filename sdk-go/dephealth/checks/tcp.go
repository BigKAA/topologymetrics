package checks

import (
	"context"
	"fmt"
	"net"

	"github.com/company/dephealth/dephealth"
)

// TCPChecker performs health checks by establishing a TCP connection.
// The check succeeds if a TCP connection can be established within the
// context deadline. No data is sent or received.
type TCPChecker struct{}

// NewTCPChecker creates a new TCP health checker.
func NewTCPChecker() *TCPChecker {
	return &TCPChecker{}
}

// Check establishes a TCP connection to the endpoint and immediately closes it.
// Returns nil if the connection succeeds, or an error otherwise.
func (c *TCPChecker) Check(ctx context.Context, endpoint dephealth.Endpoint) error {
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
func (c *TCPChecker) Type() string {
	return string(dephealth.TypeTCP)
}
