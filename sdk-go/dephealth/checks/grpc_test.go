package checks

import (
	"context"
	"errors"
	"net"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/BigKAA/topologymetrics/sdk-go/dephealth"
)

// testHealthServer implements the gRPC Health service for testing.
type testHealthServer struct {
	healthpb.UnimplementedHealthServer
	status healthpb.HealthCheckResponse_ServingStatus
}

func (s *testHealthServer) Check(_ context.Context, _ *healthpb.HealthCheckRequest) (*healthpb.HealthCheckResponse, error) {
	return &healthpb.HealthCheckResponse{Status: s.status}, nil
}

// testAuthHealthServer requires authorization metadata to respond SERVING.
type testAuthHealthServer struct {
	healthpb.UnimplementedHealthServer
	requiredAuth string // expected value of "authorization" metadata
	gotAuth      string // captured value for assertions
}

func (s *testAuthHealthServer) Check(ctx context.Context, _ *healthpb.HealthCheckRequest) (*healthpb.HealthCheckResponse, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing metadata")
	}
	vals := md.Get("authorization")
	if len(vals) == 0 {
		return nil, status.Error(codes.Unauthenticated, "missing authorization metadata")
	}
	s.gotAuth = vals[0]
	if vals[0] != s.requiredAuth {
		return nil, status.Error(codes.PermissionDenied, "invalid credentials")
	}
	return &healthpb.HealthCheckResponse{Status: healthpb.HealthCheckResponse_SERVING}, nil
}

func startTestGRPCServer(t *testing.T, status healthpb.HealthCheckResponse_ServingStatus) (string, func()) {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start TCP listener: %v", err)
	}

	srv := grpc.NewServer()
	healthpb.RegisterHealthServer(srv, &testHealthServer{status: status})

	go func() {
		_ = srv.Serve(ln)
	}()

	return ln.Addr().String(), func() {
		srv.GracefulStop()
	}
}

func startTestAuthGRPCServer(t *testing.T, requiredAuth string) (string, *testAuthHealthServer, func()) {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start TCP listener: %v", err)
	}

	authSrv := &testAuthHealthServer{requiredAuth: requiredAuth}
	srv := grpc.NewServer()
	healthpb.RegisterHealthServer(srv, authSrv)

	go func() {
		_ = srv.Serve(ln)
	}()

	return ln.Addr().String(), authSrv, func() {
		srv.GracefulStop()
	}
}

func TestGRPCChecker_Check_Serving(t *testing.T) {
	addr, stop := startTestGRPCServer(t, healthpb.HealthCheckResponse_SERVING)
	defer stop()

	host, port, _ := net.SplitHostPort(addr)
	ep := dephealth.Endpoint{Host: host, Port: port}

	checker := NewGRPCChecker()
	if err := checker.Check(context.Background(), ep); err != nil {
		t.Errorf("expected success, got error: %v", err)
	}
}

func TestGRPCChecker_Check_NotServing(t *testing.T) {
	addr, stop := startTestGRPCServer(t, healthpb.HealthCheckResponse_NOT_SERVING)
	defer stop()

	host, port, _ := net.SplitHostPort(addr)
	ep := dephealth.Endpoint{Host: host, Port: port}

	checker := NewGRPCChecker()
	err := checker.Check(context.Background(), ep)
	if err == nil {
		t.Error("expected error for NOT_SERVING, got nil")
	}
}

func TestGRPCChecker_Check_WithServiceName(t *testing.T) {
	addr, stop := startTestGRPCServer(t, healthpb.HealthCheckResponse_SERVING)
	defer stop()

	host, port, _ := net.SplitHostPort(addr)
	ep := dephealth.Endpoint{Host: host, Port: port}

	checker := NewGRPCChecker(WithServiceName("myservice"))
	if err := checker.Check(context.Background(), ep); err != nil {
		t.Errorf("expected success with serviceName, got error: %v", err)
	}
}

func TestGRPCChecker_Check_ConnectionRefused(t *testing.T) {
	ep := dephealth.Endpoint{Host: "127.0.0.1", Port: "1"}

	checker := NewGRPCChecker()
	err := checker.Check(context.Background(), ep)
	if err == nil {
		t.Error("expected error for closed port, got nil")
	}
}

func TestGRPCChecker_Check_ContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	ep := dephealth.Endpoint{Host: "127.0.0.1", Port: "9999"}

	checker := NewGRPCChecker()
	err := checker.Check(ctx, ep)
	if err == nil {
		t.Error("expected error for canceled context, got nil")
	}
}

func TestGRPCChecker_Type(t *testing.T) {
	checker := NewGRPCChecker()
	if got := checker.Type(); got != "grpc" {
		t.Errorf("Type() = %q, expected %q", got, "grpc")
	}
}

func TestGRPCChecker_Check_BearerToken(t *testing.T) {
	addr, authSrv, stop := startTestAuthGRPCServer(t, "Bearer test-token")
	defer stop()

	host, port, _ := net.SplitHostPort(addr)
	ep := dephealth.Endpoint{Host: host, Port: port}

	checker := NewGRPCChecker(WithGRPCBearerToken("test-token"))
	if err := checker.Check(context.Background(), ep); err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if authSrv.gotAuth != "Bearer test-token" {
		t.Errorf("authorization = %q, expected %q", authSrv.gotAuth, "Bearer test-token")
	}
}

func TestGRPCChecker_Check_BasicAuth(t *testing.T) {
	// Base64 of "admin:secret" = "YWRtaW46c2VjcmV0"
	addr, authSrv, stop := startTestAuthGRPCServer(t, "Basic YWRtaW46c2VjcmV0")
	defer stop()

	host, port, _ := net.SplitHostPort(addr)
	ep := dephealth.Endpoint{Host: host, Port: port}

	checker := NewGRPCChecker(WithGRPCBasicAuth("admin", "secret"))
	if err := checker.Check(context.Background(), ep); err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if authSrv.gotAuth != "Basic YWRtaW46c2VjcmV0" {
		t.Errorf("authorization = %q, expected %q", authSrv.gotAuth, "Basic YWRtaW46c2VjcmV0")
	}
}

func TestGRPCChecker_Check_CustomMetadata(t *testing.T) {
	addr, _, stop := startTestAuthGRPCServer(t, "Bearer custom-meta")
	defer stop()

	host, port, _ := net.SplitHostPort(addr)
	ep := dephealth.Endpoint{Host: host, Port: port}

	checker := NewGRPCChecker(WithMetadata(map[string]string{
		"authorization": "Bearer custom-meta",
	}))
	if err := checker.Check(context.Background(), ep); err != nil {
		t.Fatalf("expected success with custom metadata, got error: %v", err)
	}
}

func TestGRPCChecker_Check_Unauthenticated_AuthError(t *testing.T) {
	addr, _, stop := startTestAuthGRPCServer(t, "Bearer required-token")
	defer stop()

	host, port, _ := net.SplitHostPort(addr)
	ep := dephealth.Endpoint{Host: host, Port: port}

	// No auth — server returns UNAUTHENTICATED.
	checker := NewGRPCChecker()
	err := checker.Check(context.Background(), ep)
	if err == nil {
		t.Fatal("expected error for unauthenticated, got nil")
	}

	var ce *dephealth.ClassifiedCheckError
	if !errors.As(err, &ce) {
		t.Fatalf("expected ClassifiedCheckError, got %T: %v", err, err)
	}
	if ce.Category != dephealth.StatusAuthError {
		t.Errorf("Category = %q, expected %q", ce.Category, dephealth.StatusAuthError)
	}
	if ce.Detail != "auth_error" {
		t.Errorf("Detail = %q, expected %q", ce.Detail, "auth_error")
	}
}

func TestGRPCChecker_Check_PermissionDenied_AuthError(t *testing.T) {
	addr, _, stop := startTestAuthGRPCServer(t, "Bearer correct-token")
	defer stop()

	host, port, _ := net.SplitHostPort(addr)
	ep := dephealth.Endpoint{Host: host, Port: port}

	// Wrong token — server returns PERMISSION_DENIED.
	checker := NewGRPCChecker(WithGRPCBearerToken("wrong-token"))
	err := checker.Check(context.Background(), ep)
	if err == nil {
		t.Fatal("expected error for permission denied, got nil")
	}

	var ce *dephealth.ClassifiedCheckError
	if !errors.As(err, &ce) {
		t.Fatalf("expected ClassifiedCheckError, got %T: %v", err, err)
	}
	if ce.Category != dephealth.StatusAuthError {
		t.Errorf("Category = %q, expected %q", ce.Category, dephealth.StatusAuthError)
	}
}

func TestGRPCChecker_Check_NotServing_Unhealthy(t *testing.T) {
	addr, stop := startTestGRPCServer(t, healthpb.HealthCheckResponse_NOT_SERVING)
	defer stop()

	host, port, _ := net.SplitHostPort(addr)
	ep := dephealth.Endpoint{Host: host, Port: port}

	checker := NewGRPCChecker()
	err := checker.Check(context.Background(), ep)
	if err == nil {
		t.Fatal("expected error for NOT_SERVING, got nil")
	}

	var ce *dephealth.ClassifiedCheckError
	if !errors.As(err, &ce) {
		t.Fatalf("expected ClassifiedCheckError, got %T: %v", err, err)
	}
	if ce.Category != dephealth.StatusUnhealthy {
		t.Errorf("Category = %q, expected %q", ce.Category, dephealth.StatusUnhealthy)
	}
	if ce.Detail != "grpc_not_serving" {
		t.Errorf("Detail = %q, expected %q", ce.Detail, "grpc_not_serving")
	}
}
