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

// labelNames — порядок обязательных меток (dependency, type, host, port).
var labelNames = []string{"dependency", "type", "host", "port"}

// allowedOptionalLabels — допустимые опциональные метки.
var allowedOptionalLabels = map[string]bool{
	"role":  true,
	"shard": true,
	"vhost": true,
}

// MetricsExporter управляет Prometheus-метриками для зависимостей.
type MetricsExporter struct {
	health  *prometheus.GaugeVec
	latency *prometheus.HistogramVec

	// allLabelNames содержит полный набор меток (обязательные + опциональные),
	// определяемый при создании exporter.
	allLabelNames []string
}

// MetricsOption — функциональная опция для MetricsExporter.
type MetricsOption func(*metricsConfig)

type metricsConfig struct {
	registerer     prometheus.Registerer
	optionalLabels []string
}

// WithRegisterer задаёт кастомный prometheus.Registerer.
// По умолчанию используется prometheus.DefaultRegisterer.
func WithRegisterer(r prometheus.Registerer) MetricsOption {
	return func(c *metricsConfig) {
		c.registerer = r
	}
}

// WithOptionalLabels задаёт список опциональных меток (role, shard, vhost).
// Метки должны быть из допустимого набора. Порядок: алфавитный (после обязательных).
func WithOptionalLabels(labels ...string) MetricsOption {
	return func(c *metricsConfig) {
		c.optionalLabels = labels
	}
}

// NewMetricsExporter создаёт и регистрирует Prometheus-метрики.
// Возвращает ошибку, если регистрация не удалась или указаны недопустимые метки.
func NewMetricsExporter(opts ...MetricsOption) (*MetricsExporter, error) {
	cfg := metricsConfig{
		registerer: prometheus.DefaultRegisterer,
	}
	for _, o := range opts {
		o(&cfg)
	}

	// Формируем полный набор меток.
	allLabels := make([]string, len(labelNames))
	copy(allLabels, labelNames)

	if len(cfg.optionalLabels) > 0 {
		// Проверяем допустимость и сортируем.
		sorted := make([]string, len(cfg.optionalLabels))
		copy(sorted, cfg.optionalLabels)
		sort.Strings(sorted)

		for _, l := range sorted {
			if !allowedOptionalLabels[l] {
				return nil, &InvalidLabelError{Label: l}
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
	labels := prometheus.Labels{
		"dependency": dep.Name,
		"type":       string(dep.Type),
		"host":       ep.Host,
		"port":       ep.Port,
	}

	// Опциональные метки берём из Endpoint.Metadata.
	for _, name := range m.allLabelNames[len(labelNames):] {
		val, ok := ep.Metadata[name]
		if !ok {
			val = ""
		}
		labels[name] = val
	}

	return labels
}

// InvalidLabelError — ошибка при указании недопустимой опциональной метки.
type InvalidLabelError struct {
	Label string
}

func (e *InvalidLabelError) Error() string {
	return "invalid optional label: " + e.Label
}
