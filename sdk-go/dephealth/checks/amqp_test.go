package checks

import (
	"context"
	"testing"
	"time"

	"github.com/BigKAA/topologymetrics/sdk-go/dephealth"
)

func TestAMQPChecker_Check_ConnectionRefused(t *testing.T) {
	checker := NewAMQPChecker()
	ep := dephealth.Endpoint{Host: "127.0.0.1", Port: "1"}

	err := checker.Check(context.Background(), ep)
	if err == nil {
		t.Error("expected error for closed port, got nil")
	}
}

func TestAMQPChecker_Check_ContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	checker := NewAMQPChecker()
	ep := dephealth.Endpoint{Host: "127.0.0.1", Port: "5672"}

	err := checker.Check(ctx, ep)
	if err == nil {
		t.Error("expected error for canceled context, got nil")
	}
}

func TestAMQPChecker_Check_Timeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Connect to a non-existent host to trigger a timeout.
	checker := NewAMQPChecker()
	ep := dephealth.Endpoint{Host: "192.0.2.1", Port: "5672"} // TEST-NET-1, non-routable

	err := checker.Check(ctx, ep)
	if err == nil {
		t.Error("expected timeout error, got nil")
	}
}

func TestAMQPChecker_Check_WithURL(t *testing.T) {
	// Verify that the URL is used instead of endpoint.
	checker := NewAMQPChecker(WithAMQPURL("amqp://guest:guest@127.0.0.1:1/"))
	ep := dephealth.Endpoint{Host: "ignored", Port: "5672"}

	err := checker.Check(context.Background(), ep)
	if err == nil {
		t.Error("expected error (connection refused via URL), got nil")
	}
}

func TestAMQPChecker_Type(t *testing.T) {
	checker := NewAMQPChecker()
	if got := checker.Type(); got != "amqp" {
		t.Errorf("Type() = %q, expected %q", got, "amqp")
	}
}
