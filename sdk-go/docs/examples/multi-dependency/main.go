// Example: multiple dependencies with pool integration, custom labels,
// a Kubernetes readiness probe, and a JSON health details endpoint.
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"

	"github.com/BigKAA/topologymetrics/sdk-go/dephealth"
	_ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/grpccheck"
	_ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/httpcheck"
	_ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/kafkacheck"
	"github.com/BigKAA/topologymetrics/sdk-go/dephealth/contrib/redispool"
	"github.com/BigKAA/topologymetrics/sdk-go/dephealth/contrib/sqldb"
)

func main() {
	// Open PostgreSQL pool.
	db, err := sql.Open("pgx", "postgres://app:secret@pg.db:5432/orders?sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Open Redis client.
	rdb := redis.NewClient(&redis.Options{
		Addr:     "redis.cache:6379",
		Password: "redis-pass",
		DB:       0,
	})
	defer rdb.Close()

	// Build dephealth with multiple dependencies and global options.
	dh, err := dephealth.New("order-service", "backend",
		dephealth.WithLogger(slog.Default()),
		dephealth.WithCheckInterval(10*time.Second),
		dephealth.WithTimeout(3*time.Second),

		// PostgreSQL — pool integration.
		sqldb.FromDB("postgres-main", db,
			dephealth.FromParams("pg.db", "5432"),
			dephealth.Critical(true),
			dephealth.WithLabel("env", "production"),
		),

		// Redis — pool integration, auto-extracts host:port from client.
		redispool.FromClient("redis-cache", rdb,
			dephealth.Critical(false),
			dephealth.WithLabel("env", "production"),
		),

		// HTTP with Bearer auth.
		dephealth.HTTP("auth-service",
			dephealth.FromURL("https://auth.internal:8443"),
			dephealth.WithHTTPHealthPath("/healthz"),
			dephealth.WithHTTPBearerToken("my-service-token"),
			dephealth.Critical(true),
			dephealth.WithLabel("env", "production"),
		),

		// gRPC dependency.
		dephealth.GRPC("recommendation-grpc",
			dephealth.FromParams("recommend.internal", "9090"),
			dephealth.WithGRPCServiceName("recommendation.v1.Recommender"),
			dephealth.Critical(false),
			dephealth.WithLabel("env", "production"),
		),

		// Kafka brokers (parsed from URL with multiple hosts).
		dephealth.Kafka("events-kafka",
			dephealth.FromURL("kafka://kafka-0.broker:9092,kafka-1.broker:9092,kafka-2.broker:9092"),
			dephealth.Critical(true),
			dephealth.WithLabel("env", "production"),
		),
	)
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := dh.Start(ctx); err != nil {
		log.Fatal(err)
	}

	// Prometheus metrics.
	http.Handle("/metrics", promhttp.Handler())

	// Kubernetes readiness probe: returns 200 if all critical deps are healthy.
	http.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		health := dh.Health()
		for _, ok := range health {
			if !ok {
				w.WriteHeader(http.StatusServiceUnavailable)
				w.Write([]byte("not ready"))
				return
			}
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	// Debug endpoint: detailed JSON health status.
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		details := dh.HealthDetails()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(details)
	})

	go func() {
		if err := http.ListenAndServe(":8080", nil); err != nil {
			log.Fatal(err)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	dh.Stop()
}
