// Package grpccheck provides a gRPC health checker for dephealth.
//
// Import this package to register the gRPC checker factory:
//
//	import _ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/grpccheck"
package grpccheck

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"maps"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/BigKAA/topologymetrics/sdk-go/dephealth"
)

func init() {
	dephealth.RegisterCheckerFactory(dephealth.TypeGRPC, NewFromConfig)
}

// Option configures the Checker.
type Option func(*Checker)

// Checker performs health checks using the gRPC Health Checking Protocol.
// Each check creates a new connection, calls Health/Check, and closes the connection.
// The check succeeds if the response status is SERVING.
type Checker struct {
	serviceName   string
	tlsEnabled    bool
	tlsSkipVerify bool
	metadata      map[string]string
}

// WithServiceName sets the gRPC service name for health checks.
// An empty string checks the overall server health.
func WithServiceName(name string) Option {
	return func(c *Checker) {
		c.serviceName = name
	}
}

// WithTLS enables TLS for gRPC connections.
func WithTLS(enabled bool) Option {
	return func(c *Checker) {
		c.tlsEnabled = enabled
	}
}

// WithTLSSkipVerify skips TLS certificate verification for gRPC connections.
func WithTLSSkipVerify(skip bool) Option {
	return func(c *Checker) {
		c.tlsSkipVerify = skip
	}
}

// WithMetadata sets custom gRPC metadata for health check calls.
func WithMetadata(md map[string]string) Option {
	return func(c *Checker) {
		maps.Copy(c.metadata, md)
	}
}

// WithBearerToken sets a Bearer token for gRPC health check calls.
// Adds authorization: Bearer <token> metadata.
func WithBearerToken(token string) Option {
	return func(c *Checker) {
		c.metadata["authorization"] = "Bearer " + token
	}
}

// WithBasicAuth sets Basic Auth credentials for gRPC health check calls.
// Adds authorization: Basic <base64(username:password)> metadata.
func WithBasicAuth(username, password string) Option {
	return func(c *Checker) {
		encoded := base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
		c.metadata["authorization"] = "Basic " + encoded
	}
}

// New creates a new gRPC health checker with the given options.
func New(opts ...Option) *Checker {
	c := &Checker{
		metadata: make(map[string]string),
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// NewFromConfig creates a gRPC checker from DependencyConfig.
func NewFromConfig(dc *dephealth.DependencyConfig) dephealth.HealthChecker {
	var opts []Option
	if dc.GRPCServiceName != "" {
		opts = append(opts, WithServiceName(dc.GRPCServiceName))
	}
	if dc.GRPCTLS != nil {
		opts = append(opts, WithTLS(*dc.GRPCTLS))
	}
	if dc.GRPCTLSSkipVerify != nil {
		opts = append(opts, WithTLSSkipVerify(*dc.GRPCTLSSkipVerify))
	}
	if len(dc.GRPCMetadata) > 0 {
		opts = append(opts, WithMetadata(dc.GRPCMetadata))
	}
	if dc.GRPCBearerToken != "" {
		opts = append(opts, WithBearerToken(dc.GRPCBearerToken))
	}
	if dc.GRPCBasicUser != "" {
		opts = append(opts, WithBasicAuth(dc.GRPCBasicUser, dc.GRPCBasicPass))
	}
	return New(opts...)
}

// Check performs a gRPC health check against the endpoint.
// Creates a new connection, sends Health/Check request, and closes.
// Returns nil if the service status is SERVING.
func (c *Checker) Check(ctx context.Context, endpoint dephealth.Endpoint) error {
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

	// Attach metadata if configured.
	callCtx := ctx
	if len(c.metadata) > 0 {
		md := metadata.New(nil)
		for k, v := range c.metadata {
			md.Set(k, v)
		}
		callCtx = metadata.NewOutgoingContext(ctx, md)
	}

	client := healthpb.NewHealthClient(conn)
	resp, err := client.Check(callCtx, &healthpb.HealthCheckRequest{
		Service: c.serviceName,
	})
	if err != nil {
		// Classify UNAUTHENTICATED and PERMISSION_DENIED as auth_error.
		if s, ok := status.FromError(err); ok {
			if s.Code() == codes.Unauthenticated || s.Code() == codes.PermissionDenied {
				return &dephealth.ClassifiedCheckError{
					Category: dephealth.StatusAuthError,
					Detail:   "auth_error",
					Cause:    fmt.Errorf("grpc health check %s: %w", addr, err),
				}
			}
		}
		return fmt.Errorf("grpc health check %s: %w", addr, err)
	}

	if resp.GetStatus() != healthpb.HealthCheckResponse_SERVING {
		detail := "grpc_unknown"
		if resp.GetStatus() == healthpb.HealthCheckResponse_NOT_SERVING {
			detail = "grpc_not_serving"
		}
		return &dephealth.ClassifiedCheckError{
			Category: dephealth.StatusUnhealthy,
			Detail:   detail,
			Cause:    fmt.Errorf("grpc health status %s from %s", resp.GetStatus(), addr),
		}
	}

	return nil
}

// Type returns the dependency type for this checker.
func (c *Checker) Type() string {
	return string(dephealth.TypeGRPC)
}
