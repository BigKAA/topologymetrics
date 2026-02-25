// Example: basic HTTP dependency monitoring with Prometheus metrics export.
package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/BigKAA/topologymetrics/sdk-go/dephealth"
	_ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/httpcheck"
)

func main() {
	// Create dephealth instance with a single HTTP dependency.
	dh, err := dephealth.New("my-service", "backend",
		dephealth.WithLogger(slog.Default()),
		dephealth.HTTP("payment-api",
			dephealth.FromURL("https://payment.internal:8443/health"),
			dephealth.Critical(true),
		),
	)
	if err != nil {
		log.Fatal(err)
	}

	// Start health checks.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := dh.Start(ctx); err != nil {
		log.Fatal(err)
	}

	// Expose Prometheus metrics on /metrics.
	http.Handle("/metrics", promhttp.Handler())
	go func() {
		if err := http.ListenAndServe(":9090", nil); err != nil {
			log.Fatal(err)
		}
	}()

	// Wait for shutdown signal.
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	dh.Stop()
}
