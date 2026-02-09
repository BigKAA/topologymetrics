package dephealth

import (
	"context"
	"fmt"
	"os"
	"sort"

	"github.com/prometheus/client_golang/prometheus"
)

// DepHealth — главная точка входа SDK.
// Объединяет MetricsExporter и Scheduler, предоставляя удобный API.
type DepHealth struct {
	scheduler *Scheduler
	metrics   *MetricsExporter
}

// New создаёт экземпляр DepHealth из функциональных опций.
// name — уникальное имя приложения (метка name в метриках).
// Если name пустой, пытается прочитать из DEPHEALTH_NAME.
// Для использования встроенных фабрик (HTTP, Postgres, Redis и т.д.)
// необходимо импортировать пакет checks:
//
//	import _ "github.com/BigKAA/topologymetrics/dephealth/checks"
func New(name string, opts ...Option) (*DepHealth, error) {
	// API > env var.
	if name == "" {
		name = os.Getenv("DEPHEALTH_NAME")
	}
	if name == "" {
		return nil, fmt.Errorf("dephealth: missing name")
	}
	if err := ValidateName(name); err != nil {
		return nil, fmt.Errorf("dephealth: invalid name: %w", err)
	}

	cfg := config{
		registerer: prometheus.DefaultRegisterer,
	}

	for _, o := range opts {
		if err := o(&cfg); err != nil {
			return nil, fmt.Errorf("dephealth: %w", err)
		}
	}

	if len(cfg.entries) == 0 {
		return nil, fmt.Errorf("dephealth: no dependencies configured")
	}

	// Собираем все custom label keys из всех endpoints.
	customLabelKeys := collectCustomLabelKeys(cfg.entries)

	// Создаём MetricsExporter.
	var metricsOpts []MetricsOption
	if cfg.registerer != nil {
		metricsOpts = append(metricsOpts, WithMetricsRegisterer(cfg.registerer))
	}
	if len(customLabelKeys) > 0 {
		metricsOpts = append(metricsOpts, WithCustomLabels(customLabelKeys...))
	}
	metrics, err := NewMetricsExporter(name, metricsOpts...)
	if err != nil {
		return nil, fmt.Errorf("dephealth: metrics: %w", err)
	}

	// Создаём Scheduler.
	var schedOpts []SchedulerOption
	if cfg.logger != nil {
		schedOpts = append(schedOpts, WithSchedulerLogger(cfg.logger))
	}
	sched := NewScheduler(metrics, schedOpts...)

	// Регистрируем все зависимости.
	for _, entry := range cfg.entries {
		sched.deps = append(sched.deps, scheduledDep(entry))
	}

	return &DepHealth{
		scheduler: sched,
		metrics:   metrics,
	}, nil
}

// collectCustomLabelKeys собирает уникальные custom label keys из всех endpoints.
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

// Start запускает периодические проверки всех зависимостей.
func (dh *DepHealth) Start(ctx context.Context) error {
	return dh.scheduler.Start(ctx)
}

// Stop останавливает все проверки и ожидает завершения горутин.
func (dh *DepHealth) Stop() {
	dh.scheduler.Stop()
}

// Health возвращает текущее состояние всех endpoint-ов.
// Ключ — "dependency:host:port", значение — true/false.
func (dh *DepHealth) Health() map[string]bool {
	return dh.scheduler.Health()
}
