package amqpcheck

import (
	"context"
	"testing"
	"time"

	"github.com/BigKAA/topologymetrics/sdk-go/dephealth"
)

func TestChecker_Check_ConnectionRefused(t *testing.T) {
	checker := New()
	ep := dephealth.Endpoint{Host: "127.0.0.1", Port: "1"}

	err := checker.Check(context.Background(), ep)
	if err == nil {
		t.Error("expected error for closed port, got nil")
	}
}

func TestChecker_Check_ContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	checker := New()
	ep := dephealth.Endpoint{Host: "127.0.0.1", Port: "5672"}

	err := checker.Check(ctx, ep)
	if err == nil {
		t.Error("expected error for canceled context, got nil")
	}
}

func TestChecker_Check_Timeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Connect to a non-existent host to trigger a timeout.
	checker := New()
	ep := dephealth.Endpoint{Host: "192.0.2.1", Port: "5672"} // TEST-NET-1, non-routable

	err := checker.Check(ctx, ep)
	if err == nil {
		t.Error("expected timeout error, got nil")
	}
}

func TestChecker_Check_WithURL(t *testing.T) {
	// Verify that the URL is used instead of endpoint.
	checker := New(WithURL("amqp://guest:guest@127.0.0.1:1/"))
	ep := dephealth.Endpoint{Host: "ignored", Port: "5672"}

	err := checker.Check(context.Background(), ep)
	if err == nil {
		t.Error("expected error (connection refused via URL), got nil")
	}
}

func TestChecker_Type(t *testing.T) {
	checker := New()
	if got := checker.Type(); got != "amqp" {
		t.Errorf("Type() = %q, expected %q", got, "amqp")
	}
}

func TestNewFromConfig_URLPassedAsAMQPURL(t *testing.T) {
	dc := &dephealth.DependencyConfig{
		URL: "amqp://user:pass@rabbit.svc:5672/orders",
	}
	checker := NewFromConfig(dc)
	ac, ok := checker.(*Checker)
	if !ok {
		t.Fatal("expected *Checker")
	}
	if ac.url != dc.URL {
		t.Errorf("url = %q, expected %q", ac.url, dc.URL)
	}
}

func TestNewFromConfig_ExplicitAMQPURLHasPriority(t *testing.T) {
	dc := &dephealth.DependencyConfig{
		URL:     "amqp://user:pass@rabbit.svc:5672/orders",
		AMQPURL: "amqp://admin:admin@other.svc:5672/",
	}
	checker := NewFromConfig(dc)
	ac, ok := checker.(*Checker)
	if !ok {
		t.Fatal("expected *Checker")
	}
	if ac.url != dc.AMQPURL {
		t.Errorf("url = %q, expected %q (explicit AMQPURL)", ac.url, dc.AMQPURL)
	}
}
