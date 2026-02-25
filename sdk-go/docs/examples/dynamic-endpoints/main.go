// Example: dynamic endpoint management via a REST API.
// Endpoints can be added, removed, and updated at runtime.
package main

import (
	"context"
	"encoding/json"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/BigKAA/topologymetrics/sdk-go/dephealth"
	"github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/httpcheck"
)

func main() {
	// Start with one static HTTP dependency.
	dh, err := dephealth.New("gateway", "platform",
		dephealth.WithLogger(slog.Default()),
		dephealth.HTTP("users-api",
			dephealth.FromURL("http://users.internal:8080"),
			dephealth.Critical(true),
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

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	// GET /health — current health status.
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(dh.HealthDetails())
	})

	// POST /endpoints — add a new monitored endpoint.
	// Body: {"name": "billing-api", "host": "billing.internal", "port": "8080", "critical": true}
	mux.HandleFunc("/endpoints", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			Name     string `json:"name"`
			Host     string `json:"host"`
			Port     string `json:"port"`
			Critical bool   `json:"critical"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		ep := dephealth.Endpoint{Host: req.Host, Port: req.Port}
		checker := httpcheck.New()

		if err := dh.AddEndpoint(req.Name, dephealth.TypeHTTP, req.Critical, ep, checker); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"status": "added"})
	})

	// DELETE /endpoints?name=billing-api&host=billing.internal&port=8080
	mux.HandleFunc("/endpoints/delete", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		name := r.URL.Query().Get("name")
		host := r.URL.Query().Get("host")
		port := r.URL.Query().Get("port")

		if err := dh.RemoveEndpoint(name, host, port); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "removed"})
	})

	// PUT /endpoints — update an existing endpoint's target.
	// Body: {"name": "billing-api", "old_host": "billing.internal", "old_port": "8080",
	//        "new_host": "billing-v2.internal", "new_port": "8080"}
	mux.HandleFunc("/endpoints/update", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			Name    string `json:"name"`
			OldHost string `json:"old_host"`
			OldPort string `json:"old_port"`
			NewHost string `json:"new_host"`
			NewPort string `json:"new_port"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		newEp := dephealth.Endpoint{Host: req.NewHost, Port: req.NewPort}
		checker := httpcheck.New()

		if err := dh.UpdateEndpoint(req.Name, req.OldHost, req.OldPort, newEp, checker); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
	})

	go func() {
		if err := http.ListenAndServe(":8080", mux); err != nil {
			log.Fatal(err)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	dh.Stop()
}
