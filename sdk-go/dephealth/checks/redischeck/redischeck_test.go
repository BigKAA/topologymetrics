package redischeck

import (
	"context"
	"fmt"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"github.com/BigKAA/topologymetrics/sdk-go/dephealth"
)

func TestChecker_Check_PoolMode(t *testing.T) {
	mr := miniredis.RunT(t)

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer func() { _ = client.Close() }()

	checker := New(WithClient(client))
	ep := dephealth.Endpoint{Host: "ignored", Port: "6379"}

	if err := checker.Check(context.Background(), ep); err != nil {
		t.Errorf("expected success in pool mode, got error: %v", err)
	}
}

func TestChecker_Check_Standalone(t *testing.T) {
	mr := miniredis.RunT(t)

	checker := New()
	ep := dephealth.Endpoint{Host: mr.Host(), Port: mr.Port()}

	if err := checker.Check(context.Background(), ep); err != nil {
		t.Errorf("expected success in standalone mode, got error: %v", err)
	}
}

func TestChecker_Check_Standalone_WithPassword(t *testing.T) {
	mr := miniredis.RunT(t)
	mr.RequireAuth("secret")

	checker := New(WithPassword("secret"))
	ep := dephealth.Endpoint{Host: mr.Host(), Port: mr.Port()}

	if err := checker.Check(context.Background(), ep); err != nil {
		t.Errorf("expected success with password, got error: %v", err)
	}
}

func TestChecker_Check_Standalone_WrongPassword(t *testing.T) {
	mr := miniredis.RunT(t)
	mr.RequireAuth("secret")

	checker := New(WithPassword("wrong"))
	ep := dephealth.Endpoint{Host: mr.Host(), Port: mr.Port()}

	if err := checker.Check(context.Background(), ep); err == nil {
		t.Error("expected error for wrong password, got nil")
	}
}

func TestChecker_Check_Standalone_WithDB(t *testing.T) {
	mr := miniredis.RunT(t)

	checker := New(WithDB(2))
	ep := dephealth.Endpoint{Host: mr.Host(), Port: mr.Port()}

	if err := checker.Check(context.Background(), ep); err != nil {
		t.Errorf("expected success with DB=2, got error: %v", err)
	}
}

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
	ep := dephealth.Endpoint{Host: "127.0.0.1", Port: "6379"}

	err := checker.Check(ctx, ep)
	if err == nil {
		t.Error("expected error for canceled context, got nil")
	}
}

func TestChecker_Type(t *testing.T) {
	checker := New()
	if got := checker.Type(); got != "redis" {
		t.Errorf("Type() = %q, expected %q", got, "redis")
	}
}

func TestNewFromConfig_PasswordFromURL(t *testing.T) {
	mr := miniredis.RunT(t)
	mr.RequireAuth("secret")

	dc := &dephealth.DependencyConfig{
		URL: fmt.Sprintf("redis://:secret@%s:%s/0", mr.Host(), mr.Port()),
	}
	checker := NewFromConfig(dc)
	ep := dephealth.Endpoint{Host: mr.Host(), Port: mr.Port()}

	if err := checker.Check(context.Background(), ep); err != nil {
		t.Errorf("expected success with password from URL, got error: %v", err)
	}
}

func TestNewFromConfig_ExplicitPasswordOverridesURL(t *testing.T) {
	mr := miniredis.RunT(t)
	mr.RequireAuth("correct")

	dc := &dephealth.DependencyConfig{
		URL:           fmt.Sprintf("redis://:wrong@%s:%s/0", mr.Host(), mr.Port()),
		RedisPassword: "correct",
	}
	checker := NewFromConfig(dc)
	ep := dephealth.Endpoint{Host: mr.Host(), Port: mr.Port()}

	if err := checker.Check(context.Background(), ep); err != nil {
		t.Errorf("expected success with explicit password, got error: %v", err)
	}
}

func TestNewFromConfig_DBFromURL(t *testing.T) {
	dc := &dephealth.DependencyConfig{
		URL: "redis://localhost:6379/3",
	}
	checker := NewFromConfig(dc)
	rc, ok := checker.(*Checker)
	if !ok {
		t.Fatal("expected *Checker")
	}
	if rc.db != 3 {
		t.Errorf("db = %d, expected 3", rc.db)
	}
}
