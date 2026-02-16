package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// authMode defines the authentication mode for the health endpoint.
type authMode int

const (
	authNone   authMode = iota
	authBearer          // Require "Authorization: Bearer <token>"
	authBasic           // Require "Authorization: Basic <base64(user:pass)>"
	authHeader          // Require a specific header with a specific value
)

type authConfig struct {
	mode        authMode
	token       string // bearer token
	username    string // basic auth username
	password    string // basic auth password
	headerName  string // required header name
	headerValue string // required header value
}

type stubState struct {
	mu      sync.RWMutex
	healthy bool
	delay   time.Duration
	auth    authConfig
}

func (s *stubState) isHealthy() (bool, time.Duration) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.healthy, s.delay
}

func (s *stubState) setHealthy(v bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.healthy = v
}

func (s *stubState) setDelay(d time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.delay = d
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

// checkAuth validates the request against the current auth config.
// Returns 0 if OK, or an HTTP status code (401/403) if auth fails.
func (s *stubState) checkAuth(r *http.Request) int {
	auth := s.getAuth()

	switch auth.mode {
	case authNone:
		return 0
	case authBearer:
		header := r.Header.Get("Authorization")
		if header == "" {
			return http.StatusUnauthorized
		}
		if !strings.HasPrefix(header, "Bearer ") {
			return http.StatusForbidden
		}
		if strings.TrimPrefix(header, "Bearer ") != auth.token {
			return http.StatusForbidden
		}
		return 0
	case authBasic:
		header := r.Header.Get("Authorization")
		if header == "" {
			return http.StatusUnauthorized
		}
		if !strings.HasPrefix(header, "Basic ") {
			return http.StatusForbidden
		}
		expected := base64.StdEncoding.EncodeToString(
			[]byte(auth.username + ":" + auth.password),
		)
		if strings.TrimPrefix(header, "Basic ") != expected {
			return http.StatusForbidden
		}
		return 0
	case authHeader:
		val := r.Header.Get(auth.headerName)
		if val == "" {
			return http.StatusUnauthorized
		}
		if val != auth.headerValue {
			return http.StatusForbidden
		}
		return 0
	}
	return 0
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	state := &stubState{healthy: true}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	mux := http.NewServeMux()

	// Health endpoint with auth checking
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		// Check authentication first
		if code := state.checkAuth(r); code != 0 {
			w.WriteHeader(code)
			fmt.Fprintf(w, `{"error":"unauthorized","code":%d}`, code)
			return
		}

		healthy, delay := state.isHealthy()

		if delay > 0 {
			time.Sleep(delay)
		}

		if healthy {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{"status":"healthy"}`)
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprint(w, `{"status":"unhealthy"}`)
		}
	})

	// Admin: toggle health state
	mux.HandleFunc("POST /admin/toggle", func(w http.ResponseWriter, r *http.Request) {
		healthy, _ := state.isHealthy()
		newState := !healthy
		state.setHealthy(newState)
		logger.Info("toggled health", "healthy", newState)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"healthy": newState})
	})

	// Admin: set specific health state
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

	// Admin: set delay
	mux.HandleFunc("PUT /admin/delay", func(w http.ResponseWriter, r *http.Request) {
		msStr := r.URL.Query().Get("ms")
		ms, err := strconv.Atoi(msStr)
		if err != nil || ms < 0 {
			http.Error(w, "invalid ms parameter", http.StatusBadRequest)
			return
		}
		d := time.Duration(ms) * time.Millisecond
		state.setDelay(d)
		logger.Info("set delay", "ms", ms)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]int{"delay_ms": ms})
	})

	// Admin: configure bearer token auth
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

	// Admin: configure basic auth
	mux.HandleFunc("PUT /admin/auth/basic", func(w http.ResponseWriter, r *http.Request) {
		username := r.URL.Query().Get("username")
		password := r.URL.Query().Get("password")
		if username == "" || password == "" {
			http.Error(w, "username and password parameters required", http.StatusBadRequest)
			return
		}
		state.setAuth(authConfig{mode: authBasic, username: username, password: password})
		logger.Info("set auth mode", "mode", "basic")

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"auth": "basic"})
	})

	// Admin: configure required header auth
	mux.HandleFunc("PUT /admin/auth/header", func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Query().Get("name")
		value := r.URL.Query().Get("value")
		if name == "" || value == "" {
			http.Error(w, "name and value parameters required", http.StatusBadRequest)
			return
		}
		state.setAuth(authConfig{mode: authHeader, headerName: name, headerValue: value})
		logger.Info("set auth mode", "mode", "header", "header_name", name)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"auth": "header", "header_name": name})
	})

	// Admin: disable auth
	mux.HandleFunc("DELETE /admin/auth", func(w http.ResponseWriter, r *http.Request) {
		state.setAuth(authConfig{mode: authNone})
		logger.Info("disabled auth")

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"auth": "none"})
	})

	// Admin: get current state
	mux.HandleFunc("GET /admin/status", func(w http.ResponseWriter, r *http.Request) {
		healthy, delay := state.isHealthy()
		auth := state.getAuth()
		authMode := "none"
		switch auth.mode {
		case authBearer:
			authMode = "bearer"
		case authBasic:
			authMode = "basic"
		case authHeader:
			authMode = "header"
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"healthy":  healthy,
			"delay_ms": delay.Milliseconds(),
			"auth":     authMode,
		})
	})

	addr := ":" + port
	logger.Info("starting http-stub", "addr", addr)

	server := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}
	if err := server.ListenAndServe(); err != nil {
		logger.Error("server failed", "error", err)
		os.Exit(1)
	}
}
