package checks

import (
	"context"
	"net"
	"testing"

	"github.com/BigKAA/topologymetrics/sdk-go/dephealth"
)

func TestTCPChecker_Check_Success(t *testing.T) {
	// Start a TCP server on a random port.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start TCP listener: %v", err)
	}
	defer func() { _ = ln.Close() }()

	// Accept connections in a goroutine.
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		_ = conn.Close()
	}()

	_, port, _ := net.SplitHostPort(ln.Addr().String())
	ep := dephealth.Endpoint{Host: "127.0.0.1", Port: port}

	checker := NewTCPChecker()
	if err := checker.Check(context.Background(), ep); err != nil {
		t.Errorf("expected success, got error: %v", err)
	}
}

func TestTCPChecker_Check_ConnectionRefused(t *testing.T) {
	// Use a port where nothing is listening.
	ep := dephealth.Endpoint{Host: "127.0.0.1", Port: "1"}

	checker := NewTCPChecker()
	err := checker.Check(context.Background(), ep)
	if err == nil {
		t.Error("expected error for closed port, got nil")
	}
}

func TestTCPChecker_Check_ContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	ep := dephealth.Endpoint{Host: "127.0.0.1", Port: "9999"}

	checker := NewTCPChecker()
	err := checker.Check(ctx, ep)
	if err == nil {
		t.Error("expected error for canceled context, got nil")
	}
}

func TestTCPChecker_Type(t *testing.T) {
	checker := NewTCPChecker()
	if got := checker.Type(); got != "tcp" {
		t.Errorf("Type() = %q, expected %q", got, "tcp")
	}
}
