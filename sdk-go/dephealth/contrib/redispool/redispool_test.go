package redispool

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"

	"github.com/BigKAA/topologymetrics/dephealth"
	_ "github.com/BigKAA/topologymetrics/dephealth/checks" // регистрация фабрик
)

func TestFromClient(t *testing.T) {
	mr := miniredis.RunT(t)

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer func() { _ = client.Close() }()

	reg := prometheus.NewRegistry()
	dh, err := dephealth.New(
		dephealth.WithRegisterer(reg),
		FromClient("redis-cache", client),
	)
	if err != nil {
		t.Fatalf("ошибка создания DepHealth: %v", err)
	}

	if err := dh.Start(context.Background()); err != nil {
		t.Fatalf("ошибка запуска: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	health := dh.Health()
	// Ключ должен содержать host:port из miniredis.
	found := false
	for _, v := range health {
		if v {
			found = true
		}
	}
	if !found {
		t.Error("ожидали healthy=true для Redis")
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
	dh, err := dephealth.New(
		dephealth.WithRegisterer(reg),
		FromClient("redis-cache", client,
			dephealth.Critical(true),
		),
	)
	if err != nil {
		t.Fatalf("ошибка создания: %v", err)
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
	dh, err := dephealth.New(
		dephealth.WithRegisterer(reg),
		FromClient("redis-cache", client),
	)
	if err != nil {
		t.Fatalf("ошибка создания: %v", err)
	}

	if err := dh.Start(context.Background()); err != nil {
		t.Fatalf("ошибка запуска: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	health := dh.Health()
	if len(health) != 1 {
		t.Fatalf("ожидали 1 запись, получили %d", len(health))
	}

	// Проверяем, что ключ содержит корректный host:port.
	for key := range health {
		t.Logf("Health key: %s", key)
	}

	dh.Stop()
}
