package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// authMode defines the authentication mode for the gRPC service.
type authMode int

const (
	authNone   authMode = iota
	authBearer          // Require "authorization: Bearer <token>" metadata
)

type authConfig struct {
	mode  authMode
	token string // bearer token
}

type stubState struct {
	mu           sync.RWMutex
	healthy      bool
	healthServer *health.Server
	auth         authConfig
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

func (s *stubState) setAuth(cfg authConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.auth = cfg
}

func (s *stubState) getAuth() authConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.auth
}

// authInterceptor creates a unary server interceptor that checks metadata
// for authentication before forwarding to the handler.
func authInterceptor(state *stubState) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		auth := state.getAuth()

		if auth.mode == authNone {
			return handler(ctx, req)
		}

		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "missing metadata")
		}

		switch auth.mode {
		case authBearer:
			values := md.Get("authorization")
			if len(values) == 0 {
				return nil, status.Error(codes.Unauthenticated, "missing authorization metadata")
			}
			authVal := values[0]
			if !strings.HasPrefix(authVal, "Bearer ") {
				return nil, status.Error(codes.PermissionDenied, "invalid authorization format")
			}
			if strings.TrimPrefix(authVal, "Bearer ") != auth.token {
				return nil, status.Error(codes.PermissionDenied, "invalid token")
			}
		}

		return handler(ctx, req)
	}
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

	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(authInterceptor(state)),
	)
	healthpb.RegisterHealthServer(grpcServer, healthServer)

	// Start gRPC server
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

	// Admin: configure bearer token auth for gRPC
	mux.HandleFunc("PUT /admin/auth/bearer", func(w http.ResponseWriter, r *http.Request) {
		token := r.URL.Query().Get("token")
		if token == "" {
			http.Error(w, "token parameter required", http.StatusBadRequest)
			return
		}
		state.setAuth(authConfig{mode: authBearer, token: token})
		logger.Info("set auth mode", "mode", "bearer")

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"auth": "bearer"})
	})

	// Admin: disable auth
	mux.HandleFunc("DELETE /admin/auth", func(w http.ResponseWriter, r *http.Request) {
		state.setAuth(authConfig{mode: authNone})
		logger.Info("disabled auth")

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"auth": "none"})
	})

	mux.HandleFunc("GET /admin/status", func(w http.ResponseWriter, r *http.Request) {
		auth := state.getAuth()
		authModeStr := "none"
		if auth.mode == authBearer {
			authModeStr = "bearer"
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"healthy": state.isHealthy(),
			"auth":    authModeStr,
		})
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
