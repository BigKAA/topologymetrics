package checks

import (
	"context"
	"net"
	"testing"

	"google.golang.org/grpc"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"

	"github.com/company/dephealth/dephealth"
)

// testHealthServer implements the gRPC Health service for testing.
type testHealthServer struct {
	healthpb.UnimplementedHealthServer
	status healthpb.HealthCheckResponse_ServingStatus
}

func (s *testHealthServer) Check(_ context.Context, _ *healthpb.HealthCheckRequest) (*healthpb.HealthCheckResponse, error) {
	return &healthpb.HealthCheckResponse{Status: s.status}, nil
}

func startTestGRPCServer(t *testing.T, status healthpb.HealthCheckResponse_ServingStatus) (string, func()) {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("не удалось запустить TCP listener: %v", err)
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

func TestGRPCChecker_Check_Serving(t *testing.T) {
	addr, stop := startTestGRPCServer(t, healthpb.HealthCheckResponse_SERVING)
	defer stop()

	host, port, _ := net.SplitHostPort(addr)
	ep := dephealth.Endpoint{Host: host, Port: port}

	checker := NewGRPCChecker()
	if err := checker.Check(context.Background(), ep); err != nil {
		t.Errorf("ожидали успех, получили ошибку: %v", err)
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
		t.Error("ожидали ошибку для NOT_SERVING, получили nil")
	}
}

func TestGRPCChecker_Check_WithServiceName(t *testing.T) {
	addr, stop := startTestGRPCServer(t, healthpb.HealthCheckResponse_SERVING)
	defer stop()

	host, port, _ := net.SplitHostPort(addr)
	ep := dephealth.Endpoint{Host: host, Port: port}

	checker := NewGRPCChecker(WithServiceName("myservice"))
	if err := checker.Check(context.Background(), ep); err != nil {
		t.Errorf("ожидали успех с serviceName, получили ошибку: %v", err)
	}
}

func TestGRPCChecker_Check_ConnectionRefused(t *testing.T) {
	ep := dephealth.Endpoint{Host: "127.0.0.1", Port: "1"}

	checker := NewGRPCChecker()
	err := checker.Check(context.Background(), ep)
	if err == nil {
		t.Error("ожидали ошибку для закрытого порта, получили nil")
	}
}

func TestGRPCChecker_Check_ContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	ep := dephealth.Endpoint{Host: "127.0.0.1", Port: "9999"}

	checker := NewGRPCChecker()
	err := checker.Check(ctx, ep)
	if err == nil {
		t.Error("ожидали ошибку для отменённого контекста, получили nil")
	}
}

func TestGRPCChecker_Type(t *testing.T) {
	checker := NewGRPCChecker()
	if got := checker.Type(); got != "grpc" {
		t.Errorf("Type() = %q, ожидали %q", got, "grpc")
	}
}
