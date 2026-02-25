// Example: PostgreSQL pool integration with contrib/sqldb and a standalone Redis check.
package main

import (
	"context"
	"database/sql"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/BigKAA/topologymetrics/sdk-go/dephealth"
	_ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/redischeck"
	"github.com/BigKAA/topologymetrics/sdk-go/dephealth/contrib/sqldb"
)

func main() {
	// Open a real PostgreSQL connection pool.
	db, err := sql.Open("pgx", "postgres://user:pass@pg-primary.db:5432/mydb?sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Create dephealth: PostgreSQL via pool, Redis via standalone check.
	dh, err := dephealth.New("order-service", "backend",
		dephealth.WithLogger(slog.Default()),

		// PostgreSQL — reuses the existing *sql.DB connection pool.
		sqldb.FromDB("postgres-main", db,
			dephealth.FromParams("pg-primary.db", "5432"),
			dephealth.Critical(true),
		),

		// Redis — standalone check (new connection each time).
		dephealth.Redis("redis-cache",
			dephealth.FromURL("redis://redis.cache:6379"),
			dephealth.Critical(false),
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

	http.Handle("/metrics", promhttp.Handler())
	go func() {
		if err := http.ListenAndServe(":9090", nil); err != nil {
			log.Fatal(err)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	dh.Stop()
}
