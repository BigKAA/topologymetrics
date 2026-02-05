package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

type stubState struct {
	mu      sync.RWMutex
	healthy bool
	delay   time.Duration
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

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	state := &stubState{healthy: true}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	mux := http.NewServeMux()

	// Health endpoint — управляемый
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
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

	// Admin: переключить состояние здоровья
	mux.HandleFunc("POST /admin/toggle", func(w http.ResponseWriter, r *http.Request) {
		healthy, _ := state.isHealthy()
		newState := !healthy
		state.setHealthy(newState)
		logger.Info("toggled health", "healthy", newState)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"healthy": newState})
	})

	// Admin: установить конкретное состояние
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

	// Admin: установить задержку
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

	// Admin: получить текущее состояние
	mux.HandleFunc("GET /admin/status", func(w http.ResponseWriter, r *http.Request) {
		healthy, delay := state.isHealthy()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"healthy":  healthy,
			"delay_ms": delay.Milliseconds(),
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
