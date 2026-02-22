package checks

import (
	"context"
	"fmt"
	"testing"

	"github.com/alicebob/miniredis/v2"

	"github.com/BigKAA/topologymetrics/sdk-go/dephealth"
)

func TestRedisFactory_PasswordFromURL(t *testing.T) {
	mr := miniredis.RunT(t)
	mr.RequireAuth("secret")

	dc := &dephealth.DependencyConfig{
		URL: fmt.Sprintf("redis://:secret@%s:%s/0", mr.Host(), mr.Port()),
	}
	checker := newRedisFromConfig(dc)
	ep := dephealth.Endpoint{Host: mr.Host(), Port: mr.Port()}

	if err := checker.Check(context.Background(), ep); err != nil {
		t.Errorf("expected success with password from URL, got error: %v", err)
	}
}

func TestRedisFactory_ExplicitPasswordOverridesURL(t *testing.T) {
	mr := miniredis.RunT(t)
	mr.RequireAuth("correct")

	dc := &dephealth.DependencyConfig{
		URL:           fmt.Sprintf("redis://:wrong@%s:%s/0", mr.Host(), mr.Port()),
		RedisPassword: "correct",
	}
	checker := newRedisFromConfig(dc)
	ep := dephealth.Endpoint{Host: mr.Host(), Port: mr.Port()}

	if err := checker.Check(context.Background(), ep); err != nil {
		t.Errorf("expected success with explicit password, got error: %v", err)
	}
}

func TestRedisFactory_DBFromURL(t *testing.T) {
	dc := &dephealth.DependencyConfig{
		URL: "redis://localhost:6379/3",
	}
	checker := newRedisFromConfig(dc)
	rc, ok := checker.(*RedisChecker)
	if !ok {
		t.Fatal("expected *RedisChecker")
	}
	if rc.db != 3 {
		t.Errorf("db = %d, expected 3", rc.db)
	}
}

func TestAMQPFactory_URLPassedAsAMQPURL(t *testing.T) {
	dc := &dephealth.DependencyConfig{
		URL: "amqp://user:pass@rabbit.svc:5672/orders",
	}
	checker := newAMQPFromConfig(dc)
	ac, ok := checker.(*AMQPChecker)
	if !ok {
		t.Fatal("expected *AMQPChecker")
	}
	if ac.url != dc.URL {
		t.Errorf("url = %q, expected %q", ac.url, dc.URL)
	}
}

func TestAMQPFactory_ExplicitAMQPURLHasPriority(t *testing.T) {
	dc := &dephealth.DependencyConfig{
		URL:     "amqp://user:pass@rabbit.svc:5672/orders",
		AMQPURL: "amqp://admin:admin@other.svc:5672/",
	}
	checker := newAMQPFromConfig(dc)
	ac, ok := checker.(*AMQPChecker)
	if !ok {
		t.Fatal("expected *AMQPChecker")
	}
	if ac.url != dc.AMQPURL {
		t.Errorf("url = %q, expected %q (explicit AMQPURL)", ac.url, dc.AMQPURL)
	}
}
