package dephealth

import (
	"context"
	"fmt"
	"os"
	"sort"

	"github.com/prometheus/client_golang/prometheus"
)

// DepHealth is the main SDK entry point.
// It combines MetricsExporter and Scheduler, providing a convenient API.
type DepHealth struct {
	scheduler *Scheduler
	metrics   *MetricsExporter
}

// New creates a DepHealth instance from functional options.
// name is the unique application name (the "name" label in metrics).
// group is the logical group (the "group" label in metrics).
// If name is empty, it attempts to read from DEPHEALTH_NAME env var.
// If group is empty, it attempts to read from DEPHEALTH_GROUP env var.
// To use built-in checker factories (HTTP, Postgres, Redis, etc.),
// import all checkers at once:
//
//	import _ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks"
//
// Or import only what you need to reduce binary size:
//
//	import _ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/httpcheck"
//	import _ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/pgcheck"
func New(name string, group string, opts ...Option) (*DepHealth, error) {
	// name: API > env var > error.
	if name == "" {
		name = os.Getenv("DEPHEALTH_NAME")
	}
	if name == "" {
		return nil, fmt.Errorf("dephealth: missing name: pass as first argument or set DEPHEALTH_NAME")
	}
	if err := ValidateName(name); err != nil {
		return nil, fmt.Errorf("dephealth: invalid name: %w", err)
	}

	// group: API > env var > error.
	if group == "" {
		group = os.Getenv("DEPHEALTH_GROUP")
	}
	if group == "" {
		return nil, fmt.Errorf("dephealth: missing group: pass as second argument or set DEPHEALTH_GROUP")
	}
	if err := ValidateName(group); err != nil {
		return nil, fmt.Errorf("dephealth: invalid group: %w", err)
	}

	cfg := config{
		registerer: prometheus.DefaultRegisterer,
	}

	for _, o := range opts {
		if err := o(&cfg); err != nil {
			return nil, fmt.Errorf("dephealth: %w", err)
		}
	}

	// Collect all custom label keys from all endpoints.
	customLabelKeys := collectCustomLabelKeys(cfg.entries)

	// Create MetricsExporter.
	var metricsOpts []MetricsOption
	if cfg.registerer != nil {
		metricsOpts = append(metricsOpts, WithMetricsRegisterer(cfg.registerer))
	}
	if len(customLabelKeys) > 0 {
		metricsOpts = append(metricsOpts, WithCustomLabels(customLabelKeys...))
	}
	metrics, err := NewMetricsExporter(name, group, metricsOpts...)
	if err != nil {
		return nil, fmt.Errorf("dephealth: metrics: %w", err)
	}

	// Build global check config: user overrides > defaults.
	globalCfg := DefaultCheckConfig()
	if cfg.interval > 0 {
		globalCfg.Interval = cfg.interval
	}
	if cfg.timeout > 0 {
		globalCfg.Timeout = cfg.timeout
	}

	// Create Scheduler.
	var schedOpts []SchedulerOption
	if cfg.logger != nil {
		schedOpts = append(schedOpts, WithSchedulerLogger(cfg.logger))
	}
	schedOpts = append(schedOpts, WithGlobalCheckConfig(globalCfg))
	sched := NewScheduler(metrics, schedOpts...)

	// Register all dependencies.
	for _, entry := range cfg.entries {
		sched.deps = append(sched.deps, scheduledDep(entry))
	}

	return &DepHealth{
		scheduler: sched,
		metrics:   metrics,
	}, nil
}

// collectCustomLabelKeys collects unique custom label keys from all endpoints.
func collectCustomLabelKeys(entries []dependencyEntry) []string {
	keys := make(map[string]bool)
	for _, entry := range entries {
		for _, ep := range entry.dep.Endpoints {
			for k := range ep.Labels {
				keys[k] = true
			}
		}
	}
	if len(keys) == 0 {
		return nil
	}
	result := make([]string, 0, len(keys))
	for k := range keys {
		result = append(result, k)
	}
	sort.Strings(result)
	return result
}

// Start launches periodic health checks for all dependencies.
func (dh *DepHealth) Start(ctx context.Context) error {
	return dh.scheduler.Start(ctx)
}

// Stop stops all health checks and waits for goroutines to finish.
func (dh *DepHealth) Stop() {
	dh.scheduler.Stop()
}

// Health returns the current health state of all endpoints.
// Key is "dependency:host:port", value is true/false.
func (dh *DepHealth) Health() map[string]bool {
	return dh.scheduler.Health()
}

// HealthDetails returns the detailed health state of all endpoints.
// Key is "dependency:host:port", value is EndpointStatus with 11 fields.
// Unlike Health(), UNKNOWN endpoints (before first check) are included.
func (dh *DepHealth) HealthDetails() map[string]EndpointStatus {
	return dh.scheduler.HealthDetails()
}
