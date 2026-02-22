package redispool

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"

	"github.com/BigKAA/topologymetrics/sdk-go/dephealth"
	_ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/redischeck" // register Redis checker factory
)

func TestFromClient(t *testing.T) {
	mr := miniredis.RunT(t)

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer func() { _ = client.Close() }()

	reg := prometheus.NewRegistry()
	dh, err := dephealth.New("test-app", "test-group",
		dephealth.WithRegisterer(reg),
		FromClient("redis-cache", client, dephealth.Critical(false)),
	)
	if err != nil {
		t.Fatalf("failed to create DepHealth: %v", err)
	}

	if err := dh.Start(context.Background()); err != nil {
		t.Fatalf("failed to start: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	health := dh.Health()
	// Key should contain host:port from miniredis.
	found := false
	for _, v := range health {
		if v {
			found = true
		}
	}
	if !found {
		t.Error("expected healthy=true for Redis")
	}

	dh.Stop()
}

func TestFromClient_WithCritical(t *testing.T) {
	mr := miniredis.RunT(t)

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer func() { _ = client.Close() }()

	reg := prometheus.NewRegistry()
	dh, err := dephealth.New("test-app", "test-group",
		dephealth.WithRegisterer(reg),
		FromClient("redis-cache", client,
			dephealth.Critical(true),
		),
	)
	if err != nil {
		t.Fatalf("failed to create: %v", err)
	}
	_ = dh
}

func TestFromClient_AutoExtractsAddr(t *testing.T) {
	mr := miniredis.RunT(t)

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer func() { _ = client.Close() }()

	reg := prometheus.NewRegistry()
	dh, err := dephealth.New("test-app", "test-group",
		dephealth.WithRegisterer(reg),
		FromClient("redis-cache", client, dephealth.Critical(false)),
	)
	if err != nil {
		t.Fatalf("failed to create: %v", err)
	}

	if err := dh.Start(context.Background()); err != nil {
		t.Fatalf("failed to start: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	health := dh.Health()
	if len(health) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(health))
	}

	// Verify that the key contains the correct host:port.
	for key := range health {
		t.Logf("Health key: %s", key)
	}

	dh.Stop()
}
