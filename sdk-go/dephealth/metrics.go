package dephealth

import (
	"sort"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// Fixed HELP strings from the specification.
const (
	healthHelp       = "Health status of a dependency (1 = healthy, 0 = unhealthy)"
	latencyHelp      = "Latency of dependency health check in seconds"
	statusHelp       = "Category of the last check result"
	statusDetailHelp = "Detailed reason of the last check result"
)

// Histogram buckets from the specification.
var defaultLatencyBuckets = []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0, 5.0}

// requiredLabelNames contains the required v2.0 labels (name, dependency, type, host, port, critical).
var requiredLabelNames = []string{"name", "dependency", "type", "host", "port", "critical"}

// MetricsExporter manages Prometheus metrics for dependencies.
type MetricsExporter struct {
	health       *prometheus.GaugeVec
	latency      *prometheus.HistogramVec
	status       *prometheus.GaugeVec
	statusDetail *prometheus.GaugeVec

	// instanceName is the application name (the "name" label).
	instanceName string

	// allLabelNames contains the full set of labels (required + custom),
	// determined at exporter creation time.
	allLabelNames []string

	// prevDetails tracks the previous detail value per endpoint key.
	// Used to delete the old detail series when the detail changes.
	prevDetails map[string]string
	detailMu    sync.Mutex
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

	// Status metric labels: base labels + "status".
	statusLabels := make([]string, len(allLabels), len(allLabels)+1)
	copy(statusLabels, allLabels)
	statusLabels = append(statusLabels, "status")

	status := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "app_dependency_status",
		Help: statusHelp,
	}, statusLabels)

	// Status detail metric labels: base labels + "detail".
	detailLabels := make([]string, len(allLabels), len(allLabels)+1)
	copy(detailLabels, allLabels)
	detailLabels = append(detailLabels, "detail")

	statusDetail := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "app_dependency_status_detail",
		Help: statusDetailHelp,
	}, detailLabels)

	for _, collector := range []prometheus.Collector{health, latency, status, statusDetail} {
		if err := cfg.registerer.Register(collector); err != nil {
			return nil, err
		}
	}

	return &MetricsExporter{
		health:        health,
		latency:       latency,
		status:        status,
		statusDetail:  statusDetail,
		instanceName:  instanceName,
		allLabelNames: allLabels,
		prevDetails:   make(map[string]string),
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

// SetStatus updates the app_dependency_status enum gauge.
// Exactly one of the 8 status values is set to 1, the rest to 0.
func (m *MetricsExporter) SetStatus(dep Dependency, ep Endpoint, category string) {
	base := m.labels(dep, ep)
	for _, s := range AllStatusCategories {
		labels := copyLabels(base)
		labels["status"] = s
		if s == category {
			m.status.With(labels).Set(1)
		} else {
			m.status.With(labels).Set(0)
		}
	}
}

// SetStatusDetail updates the app_dependency_status_detail info gauge.
// When the detail changes, the old series is deleted and a new one is created.
func (m *MetricsExporter) SetStatusDetail(dep Dependency, ep Endpoint, detail string) {
	base := m.labels(dep, ep)
	key := endpointKey(dep, ep)

	m.detailMu.Lock()
	prev, hasPrev := m.prevDetails[key]
	m.prevDetails[key] = detail
	m.detailMu.Unlock()

	// Delete old series if detail changed.
	if hasPrev && prev != detail {
		oldLabels := copyLabels(base)
		oldLabels["detail"] = prev
		m.statusDetail.Delete(oldLabels)
	}

	labels := copyLabels(base)
	labels["detail"] = detail
	m.statusDetail.With(labels).Set(1)
}

// DeleteMetrics removes metric series for the specified endpoint.
// Used when dynamically removing a dependency.
func (m *MetricsExporter) DeleteMetrics(dep Dependency, ep Endpoint) {
	base := m.labels(dep, ep)
	m.health.Delete(base)
	m.latency.Delete(base)

	// Delete all 8 status series.
	for _, s := range AllStatusCategories {
		labels := copyLabels(base)
		labels["status"] = s
		m.status.Delete(labels)
	}

	// Delete detail series.
	key := endpointKey(dep, ep)
	m.detailMu.Lock()
	prev, hasPrev := m.prevDetails[key]
	delete(m.prevDetails, key)
	m.detailMu.Unlock()

	if hasPrev {
		labels := copyLabels(base)
		labels["detail"] = prev
		m.statusDetail.Delete(labels)
	}
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

// copyLabels creates a shallow copy of a prometheus.Labels map.
func copyLabels(src prometheus.Labels) prometheus.Labels {
	dst := make(prometheus.Labels, len(src)+1)
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

// endpointKey returns a unique string key for a dependency endpoint.
func endpointKey(dep Dependency, ep Endpoint) string {
	return dep.Name + ":" + ep.Host + ":" + ep.Port
}

// InvalidLabelError is an error returned when an invalid label is specified.
type InvalidLabelError struct {
	Label string
}

func (e *InvalidLabelError) Error() string {
	return "invalid label name: " + e.Label
}
