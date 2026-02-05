package checks

import (
	"context"
	"testing"
	"time"

	"github.com/company/dephealth/dephealth"
)

func TestAMQPChecker_Check_ConnectionRefused(t *testing.T) {
	checker := NewAMQPChecker()
	ep := dephealth.Endpoint{Host: "127.0.0.1", Port: "1"}

	err := checker.Check(context.Background(), ep)
	if err == nil {
		t.Error("ожидали ошибку для закрытого порта, получили nil")
	}
}

func TestAMQPChecker_Check_ContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	checker := NewAMQPChecker()
	ep := dephealth.Endpoint{Host: "127.0.0.1", Port: "5672"}

	err := checker.Check(ctx, ep)
	if err == nil {
		t.Error("ожидали ошибку для отменённого контекста, получили nil")
	}
}

func TestAMQPChecker_Check_Timeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Подключаемся к несуществующему хосту, чтобы сработал таймаут.
	checker := NewAMQPChecker()
	ep := dephealth.Endpoint{Host: "192.0.2.1", Port: "5672"} // TEST-NET-1, не маршрутизируется

	err := checker.Check(ctx, ep)
	if err == nil {
		t.Error("ожидали ошибку таймаута, получили nil")
	}
}

func TestAMQPChecker_Check_WithURL(t *testing.T) {
	// Проверяем, что URL используется вместо endpoint.
	checker := NewAMQPChecker(WithAMQPURL("amqp://guest:guest@127.0.0.1:1/"))
	ep := dephealth.Endpoint{Host: "ignored", Port: "5672"}

	err := checker.Check(context.Background(), ep)
	if err == nil {
		t.Error("ожидали ошибку (connection refused по URL), получили nil")
	}
}

func TestAMQPChecker_Type(t *testing.T) {
	checker := NewAMQPChecker()
	if got := checker.Type(); got != "amqp" {
		t.Errorf("Type() = %q, ожидали %q", got, "amqp")
	}
}
