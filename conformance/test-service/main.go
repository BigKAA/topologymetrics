package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/BigKAA/topologymetrics/dephealth"
	"github.com/BigKAA/topologymetrics/dephealth/contrib/redispool"
	"github.com/BigKAA/topologymetrics/dephealth/contrib/sqldb"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"

	// Регистрация фабрик чекеров
	_ "github.com/BigKAA/topologymetrics/dephealth/checks"

	// PostgreSQL драйвер
	_ "github.com/jackc/pgx/v5/stdlib"
)

// Config хранит конфигурацию conformance test service.
type Config struct {
	Port             string
	PrimaryDBURL     string
	ReplicaDBURL     string
	RedisURL         string
	RabbitMQURL      string
	KafkaHost        string
	KafkaPort        string
	HTTPStubURL      string
	GRPCStubHost     string
	GRPCStubPort     string
	CheckInterval    time.Duration
}

func loadConfig() (*Config, error) {
	cfg := &Config{
		Port:         getEnv("PORT", "8080"),
		PrimaryDBURL: getEnv("PRIMARY_DATABASE_URL", "postgres://dephealth:dephealth-test-pass@postgres-primary.dephealth-conformance.svc:5432/dephealth?sslmode=disable"),
		ReplicaDBURL: getEnv("REPLICA_DATABASE_URL", "postgres://dephealth:dephealth-test-pass@postgres-replica.dephealth-conformance.svc:5432/dephealth?sslmode=disable"),
		RedisURL:     getEnv("REDIS_URL", "redis://redis.dephealth-conformance.svc:6379/0"),
		RabbitMQURL:  getEnv("RABBITMQ_URL", "amqp://dephealth:dephealth-test-pass@rabbitmq.dephealth-conformance.svc:5672/"),
		KafkaHost:    getEnv("KAFKA_HOST", "kafka.dephealth-conformance.svc"),
		KafkaPort:    getEnv("KAFKA_PORT", "9092"),
		HTTPStubURL:  getEnv("HTTP_STUB_URL", "http://http-stub.dephealth-conformance.svc:8080"),
		GRPCStubHost: getEnv("GRPC_STUB_HOST", "grpc-stub.dephealth-conformance.svc"),
		GRPCStubPort: getEnv("GRPC_STUB_PORT", "9090"),
	}

	intervalStr := getEnv("CHECK_INTERVAL", "10s")
	d, err := time.ParseDuration(intervalStr)
	if err != nil {
		return nil, fmt.Errorf("невалидный CHECK_INTERVAL %q: %w", intervalStr, err)
	}
	cfg.CheckInterval = d

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func initDB(ctx context.Context, dsn string, name string, logger *slog.Logger) (*sql.DB, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("ошибка открытия %s: %w", name, err)
	}
	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(5 * time.Minute)

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		logger.Warn("не удалось подключиться к "+name, "error", err)
		// Не fatal — replica может быть недоступна
		return db, nil
	}
	return db, nil
}

func initRedis(ctx context.Context, rawURL string) (*redis.Client, error) {
	opts, err := redis.ParseURL(rawURL)
	if err != nil {
		return nil, fmt.Errorf("невалидный REDIS_URL: %w", err)
	}

	client := redis.NewClient(opts)

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := client.Ping(pingCtx).Err(); err != nil {
		return nil, fmt.Errorf("ошибка подключения к Redis: %w", err)
	}
	return client, nil
}

func initDepHealth(cfg *Config, primaryDB, replicaDB *sql.DB, rdb *redis.Client, logger *slog.Logger) (*dephealth.DepHealth, error) {
	dh, err := dephealth.New(
		dephealth.WithCheckInterval(cfg.CheckInterval),
		dephealth.WithLogger(logger),

		// PostgreSQL primary — через connection pool
		sqldb.FromDB("postgres-primary", primaryDB,
			dephealth.FromURL(cfg.PrimaryDBURL),
			dephealth.Critical(true),
		),

		// PostgreSQL replica — через connection pool
		sqldb.FromDB("postgres-replica", replicaDB,
			dephealth.FromURL(cfg.ReplicaDBURL),
		),

		// Redis — через connection pool (auto host:port)
		redispool.FromClient("redis-cache", rdb,
			dephealth.Critical(true),
		),

		// RabbitMQ — standalone
		dephealth.AMQP("rabbitmq",
			dephealth.FromParams("rabbitmq.dephealth-conformance.svc", "5672"),
			dephealth.WithAMQPURL(cfg.RabbitMQURL),
		),

		// Kafka — standalone
		dephealth.Kafka("kafka-main",
			dephealth.FromParams(cfg.KafkaHost, cfg.KafkaPort),
		),

		// HTTP stub — standalone
		dephealth.HTTP("http-service",
			dephealth.FromURL(cfg.HTTPStubURL),
			dephealth.WithHTTPHealthPath("/health"),
		),

		// gRPC stub — standalone
		dephealth.GRPC("grpc-service",
			dephealth.FromParams(cfg.GRPCStubHost, cfg.GRPCStubPort),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания DepHealth: %w", err)
	}
	return dh, nil
}

// Handlers

func handleIndex() http.HandlerFunc {
	type response struct {
		Service     string `json:"service"`
		Version     string `json:"version"`
		Description string `json:"description"`
	}
	resp := response{
		Service:     "dephealth-conformance-test",
		Version:     "0.1.0",
		Description: "Conformance test service для dephealth SDK (7 зависимостей)",
	}
	data, _ := json.Marshal(resp)

	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(data)
	}
}

func handleHealth() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "OK")
	}
}

func handleDependencies(dh *dephealth.DepHealth) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		health := dh.Health()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(health)
	}
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	cfg, err := loadConfig()
	if err != nil {
		logger.Error("ошибка загрузки конфигурации", "error", err)
		os.Exit(1)
	}
	logger.Info("конфигурация загружена",
		"port", cfg.Port,
		"check_interval", cfg.CheckInterval,
	)

	ctx := context.Background()

	// Инициализация подключений
	primaryDB, err := initDB(ctx, cfg.PrimaryDBURL, "postgres-primary", logger)
	if err != nil {
		logger.Error("ошибка инициализации primary DB", "error", err)
		os.Exit(1)
	}
	defer primaryDB.Close()
	logger.Info("подключение к postgres-primary установлено")

	replicaDB, err := initDB(ctx, cfg.ReplicaDBURL, "postgres-replica", logger)
	if err != nil {
		logger.Error("ошибка инициализации replica DB", "error", err)
		os.Exit(1)
	}
	defer replicaDB.Close()
	logger.Info("подключение к postgres-replica установлено")

	rdb, err := initRedis(ctx, cfg.RedisURL)
	if err != nil {
		logger.Error("ошибка инициализации Redis", "error", err)
		os.Exit(1)
	}
	defer rdb.Close()
	logger.Info("подключение к Redis установлено")

	// Инициализация и запуск dephealth
	dh, err := initDepHealth(cfg, primaryDB, replicaDB, rdb, logger)
	if err != nil {
		logger.Error("ошибка инициализации dephealth", "error", err)
		os.Exit(1)
	}

	if err := dh.Start(ctx); err != nil {
		logger.Error("ошибка запуска dephealth", "error", err)
		os.Exit(1)
	}
	logger.Info("dephealth запущен", "dependencies", 7, "check_interval", cfg.CheckInterval)

	// HTTP-сервер
	mux := http.NewServeMux()
	mux.HandleFunc("/", handleIndex())
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/health", handleHealth())
	mux.HandleFunc("/health/dependencies", handleDependencies(dh))

	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Запуск сервера в горутине
	go func() {
		logger.Info("HTTP-сервер запущен", "addr", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("ошибка HTTP-сервера", "error", err)
			os.Exit(1)
		}
	}()

	// Graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	sig := <-sigCh
	logger.Info("получен сигнал завершения", "signal", sig)

	dh.Stop()
	logger.Info("dephealth остановлен")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("ошибка остановки HTTP-сервера", "error", err)
	}
	logger.Info("HTTP-сервер остановлен, завершение")
}
