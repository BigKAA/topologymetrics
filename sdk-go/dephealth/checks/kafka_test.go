package checks

import (
	"context"
	"testing"
	"time"

	"github.com/BigKAA/topologymetrics/sdk-go/dephealth"
)

func TestKafkaChecker_Check_ConnectionRefused(t *testing.T) {
	checker := NewKafkaChecker()
	ep := dephealth.Endpoint{Host: "127.0.0.1", Port: "1"}

	err := checker.Check(context.Background(), ep)
	if err == nil {
		t.Error("expected error for closed port, got nil")
	}
}

func TestKafkaChecker_Check_ContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	checker := NewKafkaChecker()
	ep := dephealth.Endpoint{Host: "127.0.0.1", Port: "9092"}

	err := checker.Check(ctx, ep)
	if err == nil {
		t.Error("expected error for canceled context, got nil")
	}
}

func TestKafkaChecker_Check_Timeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	checker := NewKafkaChecker()
	ep := dephealth.Endpoint{Host: "192.0.2.1", Port: "9092"} // TEST-NET-1

	err := checker.Check(ctx, ep)
	if err == nil {
		t.Error("expected timeout error, got nil")
	}
}

func TestKafkaChecker_Type(t *testing.T) {
	checker := NewKafkaChecker()
	if got := checker.Type(); got != "kafka" {
		t.Errorf("Type() = %q, expected %q", got, "kafka")
	}
}
