package dephealth

import (
	"sort"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// Фиксированные HELP-строки из спецификации.
const (
	healthHelp  = "Health status of a dependency (1 = healthy, 0 = unhealthy)"
	latencyHelp = "Latency of dependency health check in seconds"
)

// Бакеты histogram из спецификации.
var defaultLatencyBuckets = []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0, 5.0}

// requiredLabelNames — обязательные метки v2.0 (name, dependency, type, host, port, critical).
var requiredLabelNames = []string{"name", "dependency", "type", "host", "port", "critical"}

// MetricsExporter управляет Prometheus-метриками для зависимостей.
type MetricsExporter struct {
	health  *prometheus.GaugeVec
	latency *prometheus.HistogramVec

	// instanceName — имя приложения (метка name).
	instanceName string

	// allLabelNames содержит полный набор меток (обязательные + произвольные),
	// определяемый при создании exporter.
	allLabelNames []string
}

// MetricsOption — функциональная опция для MetricsExporter.
type MetricsOption func(*metricsConfig)

type metricsConfig struct {
	registerer       prometheus.Registerer
	customLabelNames []string
}

// WithMetricsRegisterer задаёт кастомный prometheus.Registerer.
// По умолчанию используется prometheus.DefaultRegisterer.
func WithMetricsRegisterer(r prometheus.Registerer) MetricsOption {
	return func(c *metricsConfig) {
		c.registerer = r
	}
}

// WithCustomLabels задаёт список произвольных меток (custom labels).
// Имена валидируются при создании exporter. Порядок: алфавитный (после обязательных).
func WithCustomLabels(labels ...string) MetricsOption {
	return func(c *metricsConfig) {
		c.customLabelNames = labels
	}
}

// NewMetricsExporter создаёт и регистрирует Prometheus-метрики.
// instanceName — имя приложения (метка name), добавляется ко всем метрикам.
// Возвращает ошибку, если регистрация не удалась или указаны недопустимые метки.
func NewMetricsExporter(instanceName string, opts ...MetricsOption) (*MetricsExporter, error) {
	cfg := metricsConfig{
		registerer: prometheus.DefaultRegisterer,
	}
	for _, o := range opts {
		o(&cfg)
	}

	// Формируем полный набор меток.
	allLabels := make([]string, len(requiredLabelNames))
	copy(allLabels, requiredLabelNames)

	if len(cfg.customLabelNames) > 0 {
		// Проверяем допустимость и сортируем.
		sorted := make([]string, len(cfg.customLabelNames))
		copy(sorted, cfg.customLabelNames)
		sort.Strings(sorted)

		for _, l := range sorted {
			if err := ValidateLabelName(l); err != nil {
				return nil, err
			}
		}
		allLabels = append(allLabels, sorted...)
	}

	health := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "app_dependency_health",
		Help: healthHelp,
	}, allLabels)

	latency := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "app_dependency_latency_seconds",
		Help:    latencyHelp,
		Buckets: defaultLatencyBuckets,
	}, allLabels)

	if err := cfg.registerer.Register(health); err != nil {
		return nil, err
	}
	if err := cfg.registerer.Register(latency); err != nil {
		return nil, err
	}

	return &MetricsExporter{
		health:        health,
		latency:       latency,
		instanceName:  instanceName,
		allLabelNames: allLabels,
	}, nil
}

// SetHealth обновляет значение gauge app_dependency_health.
// value: 1 (healthy) или 0 (unhealthy).
func (m *MetricsExporter) SetHealth(dep Dependency, ep Endpoint, value float64) {
	m.health.With(m.labels(dep, ep)).Set(value)
}

// ObserveLatency записывает длительность проверки в histogram.
func (m *MetricsExporter) ObserveLatency(dep Dependency, ep Endpoint, duration time.Duration) {
	m.latency.With(m.labels(dep, ep)).Observe(duration.Seconds())
}

// DeleteMetrics удаляет серии метрик для указанного endpoint.
// Используется при динамическом удалении зависимости.
func (m *MetricsExporter) DeleteMetrics(dep Dependency, ep Endpoint) {
	labels := m.labels(dep, ep)
	m.health.Delete(labels)
	m.latency.Delete(labels)
}

// labels формирует набор меток из Dependency и Endpoint.
func (m *MetricsExporter) labels(dep Dependency, ep Endpoint) prometheus.Labels {
	critical := "no"
	if dep.Critical != nil && *dep.Critical {
		critical = "yes"
	}

	labels := prometheus.Labels{
		"name":       m.instanceName,
		"dependency": dep.Name,
		"type":       string(dep.Type),
		"host":       ep.Host,
		"port":       ep.Port,
		"critical":   critical,
	}

	// Произвольные метки берём из Endpoint.Labels.
	for _, name := range m.allLabelNames[len(requiredLabelNames):] {
		val, ok := ep.Labels[name]
		if !ok {
			val = ""
		}
		labels[name] = val
	}

	return labels
}

// InvalidLabelError — ошибка при указании недопустимой метки.
type InvalidLabelError struct {
	Label string
}

func (e *InvalidLabelError) Error() string {
	return "invalid label name: " + e.Label
}
