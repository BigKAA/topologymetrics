package dephealth

import (
	"context"
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
)

// DepHealth — главная точка входа SDK.
// Объединяет MetricsExporter и Scheduler, предоставляя удобный API.
type DepHealth struct {
	scheduler *Scheduler
	metrics   *MetricsExporter
}

// New создаёт экземпляр DepHealth из функциональных опций.
// Для использования встроенных фабрик (HTTP, Postgres, Redis и т.д.)
// необходимо импортировать пакет checks:
//
//	import _ "github.com/company/dephealth/dephealth/checks"
func New(opts ...Option) (*DepHealth, error) {
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

	// Создаём MetricsExporter.
	var metricsOpts []MetricsOption
	if cfg.registerer != nil {
		metricsOpts = append(metricsOpts, WithMetricsRegisterer(cfg.registerer))
	}
	metrics, err := NewMetricsExporter(metricsOpts...)
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
