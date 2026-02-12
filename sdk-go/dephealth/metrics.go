package dephealth

import (
	"sort"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// Fixed HELP strings from the specification.
const (
	healthHelp  = "Health status of a dependency (1 = healthy, 0 = unhealthy)"
	latencyHelp = "Latency of dependency health check in seconds"
)

// Histogram buckets from the specification.
var defaultLatencyBuckets = []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0, 5.0}

// requiredLabelNames contains the required v2.0 labels (name, dependency, type, host, port, critical).
var requiredLabelNames = []string{"name", "dependency", "type", "host", "port", "critical"}

// MetricsExporter manages Prometheus metrics for dependencies.
type MetricsExporter struct {
	health  *prometheus.GaugeVec
	latency *prometheus.HistogramVec

	// instanceName is the application name (the "name" label).
	instanceName string

	// allLabelNames contains the full set of labels (required + custom),
	// determined at exporter creation time.
	allLabelNames []string
}

// MetricsOption is a functional option for MetricsExporter.
type MetricsOption func(*metricsConfig)

type metricsConfig struct {
	registerer       prometheus.Registerer
	customLabelNames []string
}

// WithMetricsRegisterer sets a custom prometheus.Registerer.
// Defaults to prometheus.DefaultRegisterer.
func WithMetricsRegisterer(r prometheus.Registerer) MetricsOption {
	return func(c *metricsConfig) {
		c.registerer = r
	}
}

// WithCustomLabels sets the list of custom label names.
// Names are validated at exporter creation time. Order: alphabetical (after required labels).
func WithCustomLabels(labels ...string) MetricsOption {
	return func(c *metricsConfig) {
		c.customLabelNames = labels
	}
}

// NewMetricsExporter creates and registers Prometheus metrics.
// instanceName is the application name (the "name" label), added to all metrics.
// Returns an error if registration fails or invalid labels are specified.
func NewMetricsExporter(instanceName string, opts ...MetricsOption) (*MetricsExporter, error) {
	cfg := metricsConfig{
		registerer: prometheus.DefaultRegisterer,
	}
	for _, o := range opts {
		o(&cfg)
	}

	// Build the full set of labels.
	allLabels := make([]string, len(requiredLabelNames))
	copy(allLabels, requiredLabelNames)

	if len(cfg.customLabelNames) > 0 {
		// Validate and sort.
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

// SetHealth updates the app_dependency_health gauge value.
// value: 1 (healthy) or 0 (unhealthy).
func (m *MetricsExporter) SetHealth(dep Dependency, ep Endpoint, value float64) {
	m.health.With(m.labels(dep, ep)).Set(value)
}

// ObserveLatency records the check duration in the histogram.
func (m *MetricsExporter) ObserveLatency(dep Dependency, ep Endpoint, duration time.Duration) {
	m.latency.With(m.labels(dep, ep)).Observe(duration.Seconds())
}

// DeleteMetrics removes metric series for the specified endpoint.
// Used when dynamically removing a dependency.
func (m *MetricsExporter) DeleteMetrics(dep Dependency, ep Endpoint) {
	labels := m.labels(dep, ep)
	m.health.Delete(labels)
	m.latency.Delete(labels)
}

// labels builds a label set from Dependency and Endpoint.
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

	// Custom labels are taken from Endpoint.Labels.
	for _, name := range m.allLabelNames[len(requiredLabelNames):] {
		val, ok := ep.Labels[name]
		if !ok {
			val = ""
		}
		labels[name] = val
	}

	return labels
}

// InvalidLabelError is an error returned when an invalid label is specified.
type InvalidLabelError struct {
	Label string
}

func (e *InvalidLabelError) Error() string {
	return "invalid label name: " + e.Label
}
