package checks

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"github.com/company/dephealth/dephealth"
)

func TestRedisChecker_Check_PoolMode(t *testing.T) {
	mr := miniredis.RunT(t)

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer func() { _ = client.Close() }()

	checker := NewRedisChecker(WithRedisClient(client))
	ep := dephealth.Endpoint{Host: "ignored", Port: "6379"}

	if err := checker.Check(context.Background(), ep); err != nil {
		t.Errorf("ожидали успех в pool mode, получили ошибку: %v", err)
	}
}

func TestRedisChecker_Check_Standalone(t *testing.T) {
	mr := miniredis.RunT(t)

	checker := NewRedisChecker()
	ep := dephealth.Endpoint{Host: mr.Host(), Port: mr.Port()}

	if err := checker.Check(context.Background(), ep); err != nil {
		t.Errorf("ожидали успех в standalone mode, получили ошибку: %v", err)
	}
}

func TestRedisChecker_Check_Standalone_WithPassword(t *testing.T) {
	mr := miniredis.RunT(t)
	mr.RequireAuth("secret")

	checker := NewRedisChecker(WithRedisPassword("secret"))
	ep := dephealth.Endpoint{Host: mr.Host(), Port: mr.Port()}

	if err := checker.Check(context.Background(), ep); err != nil {
		t.Errorf("ожидали успех с паролем, получили ошибку: %v", err)
	}
}

func TestRedisChecker_Check_Standalone_WrongPassword(t *testing.T) {
	mr := miniredis.RunT(t)
	mr.RequireAuth("secret")

	checker := NewRedisChecker(WithRedisPassword("wrong"))
	ep := dephealth.Endpoint{Host: mr.Host(), Port: mr.Port()}

	if err := checker.Check(context.Background(), ep); err == nil {
		t.Error("ожидали ошибку для неверного пароля, получили nil")
	}
}

func TestRedisChecker_Check_Standalone_WithDB(t *testing.T) {
	mr := miniredis.RunT(t)

	checker := NewRedisChecker(WithRedisDB(2))
	ep := dephealth.Endpoint{Host: mr.Host(), Port: mr.Port()}

	if err := checker.Check(context.Background(), ep); err != nil {
		t.Errorf("ожидали успех с DB=2, получили ошибку: %v", err)
	}
}

func TestRedisChecker_Check_ConnectionRefused(t *testing.T) {
	checker := NewRedisChecker()
	ep := dephealth.Endpoint{Host: "127.0.0.1", Port: "1"}

	err := checker.Check(context.Background(), ep)
	if err == nil {
		t.Error("ожидали ошибку для закрытого порта, получили nil")
	}
}

func TestRedisChecker_Check_ContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	checker := NewRedisChecker()
	ep := dephealth.Endpoint{Host: "127.0.0.1", Port: "6379"}

	err := checker.Check(ctx, ep)
	if err == nil {
		t.Error("ожидали ошибку для отменённого контекста, получили nil")
	}
}

func TestRedisChecker_Type(t *testing.T) {
	checker := NewRedisChecker()
	if got := checker.Type(); got != "redis" {
		t.Errorf("Type() = %q, ожидали %q", got, "redis")
	}
}
