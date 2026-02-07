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
	"strconv"
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

// Config хранит конфигурацию сервиса из переменных окружения.
type Config struct {
	Port          string
	DatabaseURL   string
	RedisURL      string
	HTTPStubURL   string
	GRPCStubHost  string
	GRPCStubPort  string
	CheckInterval time.Duration
}

func loadConfig() (*Config, error) {
	cfg := &Config{
		Port:         getEnv("PORT", "8080"),
		DatabaseURL:  getEnv("DATABASE_URL", "postgres://dephealth:dephealth-test-pass@postgres:5432/dephealth?sslmode=disable"),
		RedisURL:     getEnv("REDIS_URL", "redis://redis:6379/0"),
		HTTPStubURL:  getEnv("HTTP_STUB_URL", "http://http-stub:8080"),
		GRPCStubHost: getEnv("GRPC_STUB_HOST", "grpc-stub"),
		GRPCStubPort: getEnv("GRPC_STUB_PORT", "9090"),
	}

	intervalStr := getEnv("CHECK_INTERVAL", "10")
	d, err := parseDuration(intervalStr)
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

// parseDuration парсит строку длительности. Принимает Go-формат ("10s", "1m")
// или число секунд ("10", "15.5").
func parseDuration(s string) (time.Duration, error) {
	if d, err := time.ParseDuration(s); err == nil {
		return d, nil
	}
	sec, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, fmt.Errorf("ожидается число секунд или Go duration: %q", s)
	}
	return time.Duration(sec * float64(time.Second)), nil
}

func initDB(ctx context.Context, dsn string) (*sql.DB, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("ошибка открытия БД: %w", err)
	}
	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(5 * time.Minute)

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		return nil, fmt.Errorf("ошибка подключения к БД: %w", err)
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

func initDepHealth(cfg *Config, db *sql.DB, rdb *redis.Client, logger *slog.Logger) (*dephealth.DepHealth, error) {
	dh, err := dephealth.New(
		dephealth.WithCheckInterval(cfg.CheckInterval),
		dephealth.WithLogger(logger),

		// PostgreSQL — через connection pool сервиса
		sqldb.FromDB("postgres", db,
			dephealth.FromURL(cfg.DatabaseURL),
			dephealth.Critical(true),
		),

		// Redis — через connection pool сервиса (auto host:port)
		redispool.FromClient("redis", rdb,
			dephealth.Critical(true),
		),

		// HTTP stub — standalone check
		dephealth.HTTP("http-stub",
			dephealth.FromURL(cfg.HTTPStubURL),
			dephealth.WithHTTPHealthPath("/health"),
		),

		// gRPC stub — standalone check
		dephealth.GRPC("grpc-stub",
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
		Service:     "dephealth-test-go",
		Version:     "0.1.0",
		Description: "Тестовый сервис для демонстрации dephealth SDK",
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
	db, err := initDB(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("ошибка инициализации БД", "error", err)
		os.Exit(1)
	}
	defer db.Close()
	logger.Info("подключение к PostgreSQL установлено")

	rdb, err := initRedis(ctx, cfg.RedisURL)
	if err != nil {
		logger.Error("ошибка инициализации Redis", "error", err)
		os.Exit(1)
	}
	defer rdb.Close()
	logger.Info("подключение к Redis установлено")

	// Инициализация и запуск dephealth
	dh, err := initDepHealth(cfg, db, rdb, logger)
	if err != nil {
		logger.Error("ошибка инициализации dephealth", "error", err)
		os.Exit(1)
	}

	if err := dh.Start(ctx); err != nil {
		logger.Error("ошибка запуска dephealth", "error", err)
		os.Exit(1)
	}
	logger.Info("dephealth запущен", "check_interval", cfg.CheckInterval)

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

	// Остановка dephealth (отменяет все проверки)
	dh.Stop()
	logger.Info("dephealth остановлен")

	// Остановка HTTP-сервера
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("ошибка остановки HTTP-сервера", "error", err)
	}
	logger.Info("HTTP-сервер остановлен, завершение")
}
