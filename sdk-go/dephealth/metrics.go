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

// requiredLabelNames contains the required labels (name, group, dependency, type, host, port, critical).
var requiredLabelNames = []string{"name", "group", "dependency", "type", "host", "port", "critical"}

// MetricsExporter manages Prometheus metrics for dependencies.
type MetricsExporter struct {
	health       *prometheus.GaugeVec
	latency      *prometheus.HistogramVec
	status       *prometheus.GaugeVec
	statusDetail *prometheus.GaugeVec

	// instanceName is the application name (the "name" label).
	instanceName string

	// instanceGroup is the logical group (the "group" label).
	instanceGroup string

	// allLabelNames contains the full set of labels (required + custom),
	// determined at exporter creation time.
	allLabelNames []string

	// cacheMu protects labelCache, prevStatus, and prevDetails.
	cacheMu sync.RWMutex

	// labelCache caches base labels per endpoint to avoid repeated map allocation.
	// The cached maps must not be modified by callers.
	labelCache map[string]prometheus.Labels

	// prevStatus tracks the previous status category per endpoint key.
	// Used to avoid updating all 8 status gauges when status hasn't changed.
	prevStatus map[string]StatusCategory

	// prevDetails tracks the previous detail value per endpoint key.
	// Used to delete the old detail series when the detail changes.
	prevDetails map[string]string
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
// instanceGroup is the logical group (the "group" label), added to all metrics.
// Returns an error if registration fails or invalid labels are specified.
func NewMetricsExporter(instanceName string, instanceGroup string, opts ...MetricsOption) (*MetricsExporter, error) {
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

	// Status metric uses an enum pattern: base labels + "status" dimension.
	statusLabels := make([]string, len(allLabels), len(allLabels)+1)
	copy(statusLabels, allLabels)
	statusLabels = append(statusLabels, "status")

	status := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "app_dependency_status",
		Help: statusHelp,
	}, statusLabels)

	// Status detail metric uses an info pattern: base labels + "detail" dimension.
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
		instanceGroup: instanceGroup,
		allLabelNames: allLabels,
		labelCache:    make(map[string]prometheus.Labels),
		prevStatus:    make(map[string]StatusCategory),
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
// On the first call for an endpoint, all 8 categories are initialized.
// On subsequent calls, only changed categories are updated (delta update).
// If the status hasn't changed since the last call, no gauges are touched.
func (m *MetricsExporter) SetStatus(dep Dependency, ep Endpoint, category StatusCategory) {
	key := endpointKey(dep, ep)

	m.cacheMu.Lock()
	prev, hasPrev := m.prevStatus[key]
	if hasPrev && prev == category {
		m.cacheMu.Unlock()
		return // status unchanged — skip all gauge updates
	}
	m.prevStatus[key] = category
	m.cacheMu.Unlock()

	base := m.labels(dep, ep)

	if hasPrev {
		// Delta update: only touch the two changed categories.
		oldLabels := copyLabels(base)
		oldLabels["status"] = string(prev)
		m.status.With(oldLabels).Set(0)

		newLabels := copyLabels(base)
		newLabels["status"] = string(category)
		m.status.With(newLabels).Set(1)
	} else {
		// First call: initialize all 8 status gauges.
		for _, s := range AllStatusCategories {
			labels := copyLabels(base)
			labels["status"] = string(s)
			if s == category {
				m.status.With(labels).Set(1)
			} else {
				m.status.With(labels).Set(0)
			}
		}
	}
}

// SetStatusDetail updates the app_dependency_status_detail info gauge.
// When the detail changes, the old series is deleted and a new one is created.
// If the detail hasn't changed since the last call, no action is taken.
func (m *MetricsExporter) SetStatusDetail(dep Dependency, ep Endpoint, detail string) {
	key := endpointKey(dep, ep)

	m.cacheMu.Lock()
	prev, hasPrev := m.prevDetails[key]
	if hasPrev && prev == detail {
		m.cacheMu.Unlock()
		return // detail unchanged — skip
	}
	m.prevDetails[key] = detail
	m.cacheMu.Unlock()

	base := m.labels(dep, ep)

	// Delete old series if detail changed.
	if hasPrev {
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

	key := endpointKey(dep, ep)

	// Delete all 8 status series.
	for _, s := range AllStatusCategories {
		labels := copyLabels(base)
		labels["status"] = string(s)
		m.status.Delete(labels)
	}

	// Delete detail series and clean caches.
	m.cacheMu.Lock()
	prev, hasPrev := m.prevDetails[key]
	delete(m.prevDetails, key)
	delete(m.prevStatus, key)
	delete(m.labelCache, key)
	m.cacheMu.Unlock()

	if hasPrev {
		labels := copyLabels(base)
		labels["detail"] = prev
		m.statusDetail.Delete(labels)
	}
}

// labels returns the base label set for the given dependency endpoint.
// Results are cached per endpoint key to avoid repeated map allocation.
// The returned map must not be modified — use copyLabels for mutations.
func (m *MetricsExporter) labels(dep Dependency, ep Endpoint) prometheus.Labels {
	key := endpointKey(dep, ep)

	// Fast path: read lock for cache hit.
	m.cacheMu.RLock()
	if cached, ok := m.labelCache[key]; ok {
		m.cacheMu.RUnlock()
		return cached
	}
	m.cacheMu.RUnlock()

	critical := "no"
	if dep.Critical != nil && *dep.Critical {
		critical = "yes"
	}

	labels := prometheus.Labels{
		"name":       m.instanceName,
		"group":      m.instanceGroup,
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

	m.cacheMu.Lock()
	m.labelCache[key] = labels
	m.cacheMu.Unlock()

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

// Error returns the error message describing the invalid label.
func (e *InvalidLabelError) Error() string {
	return "invalid label name: " + e.Label
}
