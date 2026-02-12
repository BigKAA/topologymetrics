package checks

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"

	"github.com/BigKAA/topologymetrics/sdk-go/dephealth"
)

// GRPCOption configures the GRPCChecker.
type GRPCOption func(*GRPCChecker)

// GRPCChecker performs health checks using the gRPC Health Checking Protocol.
// Each check creates a new connection, calls Health/Check, and closes the connection.
// The check succeeds if the response status is SERVING.
type GRPCChecker struct {
	serviceName   string
	tlsEnabled    bool
	tlsSkipVerify bool
}

// WithServiceName sets the gRPC service name for health checks.
// An empty string checks the overall server health.
func WithServiceName(name string) GRPCOption {
	return func(c *GRPCChecker) {
		c.serviceName = name
	}
}

// WithGRPCTLS enables TLS for gRPC connections.
func WithGRPCTLS(enabled bool) GRPCOption {
	return func(c *GRPCChecker) {
		c.tlsEnabled = enabled
	}
}

// WithGRPCTLSSkipVerify skips TLS certificate verification for gRPC connections.
func WithGRPCTLSSkipVerify(skip bool) GRPCOption {
	return func(c *GRPCChecker) {
		c.tlsSkipVerify = skip
	}
}

// NewGRPCChecker creates a new gRPC health checker with the given options.
func NewGRPCChecker(opts ...GRPCOption) *GRPCChecker {
	c := &GRPCChecker{}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Check performs a gRPC health check against the endpoint.
// Creates a new connection, sends Health/Check request, and closes.
// Returns nil if the service status is SERVING.
func (c *GRPCChecker) Check(ctx context.Context, endpoint dephealth.Endpoint) error {
	addr := net.JoinHostPort(endpoint.Host, endpoint.Port)

	var transportCreds grpc.DialOption
	if c.tlsEnabled {
		tlsCfg := &tls.Config{
			InsecureSkipVerify: c.tlsSkipVerify, //nolint:gosec // configurable by user
		}
		transportCreds = grpc.WithTransportCredentials(credentials.NewTLS(tlsCfg))
	} else {
		transportCreds = grpc.WithTransportCredentials(insecure.NewCredentials())
	}

	conn, err := grpc.NewClient("passthrough:///"+addr, transportCreds)
	if err != nil {
		return fmt.Errorf("grpc new client %s: %w", addr, err)
	}
	defer func() { _ = conn.Close() }()

	client := healthpb.NewHealthClient(conn)
	resp, err := client.Check(ctx, &healthpb.HealthCheckRequest{
		Service: c.serviceName,
	})
	if err != nil {
		return fmt.Errorf("grpc health check %s: %w", addr, err)
	}

	if resp.GetStatus() != healthpb.HealthCheckResponse_SERVING {
		return fmt.Errorf("grpc health status %s from %s", resp.GetStatus(), addr)
	}

	return nil
}

// Type returns the dependency type for this checker.
func (c *GRPCChecker) Type() string {
	return string(dephealth.TypeGRPC)
}
