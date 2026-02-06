package main

import (
	"encoding/json"
	"log/slog"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

type stubState struct {
	mu           sync.RWMutex
	healthy      bool
	healthServer *health.Server
}

func (s *stubState) setHealthy(v bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.healthy = v
	if v {
		s.healthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	} else {
		s.healthServer.SetServingStatus("", healthpb.HealthCheckResponse_NOT_SERVING)
	}
}

func (s *stubState) isHealthy() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.healthy
}

func main() {
	grpcPort := os.Getenv("GRPC_PORT")
	if grpcPort == "" {
		grpcPort = "9090"
	}
	adminPort := os.Getenv("ADMIN_PORT")
	if adminPort == "" {
		adminPort = "8080"
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	// gRPC Health Server
	healthServer := health.NewServer()
	healthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)

	state := &stubState{
		healthy:      true,
		healthServer: healthServer,
	}

	grpcServer := grpc.NewServer()
	healthpb.RegisterHealthServer(grpcServer, healthServer)

	// Запуск gRPC сервера
	go func() {
		lis, err := net.Listen("tcp", ":"+grpcPort)
		if err != nil {
			logger.Error("failed to listen grpc", "error", err)
			os.Exit(1)
		}
		logger.Info("starting grpc server", "port", grpcPort)
		if err := grpcServer.Serve(lis); err != nil {
			logger.Error("grpc server failed", "error", err)
			os.Exit(1)
		}
	}()

	// Admin HTTP API
	mux := http.NewServeMux()

	mux.HandleFunc("POST /admin/toggle", func(w http.ResponseWriter, r *http.Request) {
		newState := !state.isHealthy()
		state.setHealthy(newState)
		logger.Info("toggled health", "healthy", newState)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"healthy": newState})
	})

	mux.HandleFunc("PUT /admin/health", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Healthy bool `json:"healthy"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		state.setHealthy(req.Healthy)
		logger.Info("set health", "healthy", req.Healthy)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"healthy": req.Healthy})
	})

	mux.HandleFunc("GET /admin/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"healthy": state.isHealthy()})
	})

	adminAddr := ":" + adminPort
	logger.Info("starting admin http server", "addr", adminAddr)

	adminServer := &http.Server{
		Addr:              adminAddr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}
	if err := adminServer.ListenAndServe(); err != nil {
		logger.Error("admin server failed", "error", err)
		os.Exit(1)
	}
}
