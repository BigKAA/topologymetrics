package dephealth

import (
	"encoding/json"
	"time"
)

// EndpointStatus represents the detailed health check state for a single endpoint.
// It is returned by HealthDetails() and contains all 11 fields defined in the specification.
type EndpointStatus struct {
	Healthy       *bool             `json:"healthy"`
	Status        StatusCategory    `json:"status"`
	Detail        string            `json:"detail"`
	Latency       time.Duration     `json:"-"`
	Type          DependencyType    `json:"type"`
	Name          string            `json:"name"`
	Host          string            `json:"host"`
	Port          string            `json:"port"`
	Critical      bool              `json:"critical"`
	LastCheckedAt time.Time         `json:"last_checked_at"`
	Labels        map[string]string `json:"labels"`
}

// LatencyMillis returns the latency in milliseconds as a float64.
func (es EndpointStatus) LatencyMillis() float64 {
	return float64(es.Latency.Nanoseconds()) / 1e6
}

// endpointStatusJSON is the JSON representation of EndpointStatus.
type endpointStatusJSON struct {
	Healthy       *bool             `json:"healthy"`
	Status        StatusCategory    `json:"status"`
	Detail        string            `json:"detail"`
	LatencyMs     float64           `json:"latency_ms"`
	Type          DependencyType    `json:"type"`
	Name          string            `json:"name"`
	Host          string            `json:"host"`
	Port          string            `json:"port"`
	Critical      bool              `json:"critical"`
	LastCheckedAt *time.Time        `json:"last_checked_at"`
	Labels        map[string]string `json:"labels"`
}

// MarshalJSON implements custom JSON marshaling.
// Latency is serialized as latency_ms (milliseconds float).
// LastCheckedAt is serialized as null when zero (before first check).
func (es EndpointStatus) MarshalJSON() ([]byte, error) {
	j := endpointStatusJSON{
		Healthy:   es.Healthy,
		Status:    es.Status,
		Detail:    es.Detail,
		LatencyMs: es.LatencyMillis(),
		Type:      es.Type,
		Name:      es.Name,
		Host:      es.Host,
		Port:      es.Port,
		Critical:  es.Critical,
		Labels:    es.Labels,
	}
	if !es.LastCheckedAt.IsZero() {
		t := es.LastCheckedAt.UTC()
		j.LastCheckedAt = &t
	}
	return json.Marshal(j)
}

// UnmarshalJSON implements custom JSON unmarshaling.
func (es *EndpointStatus) UnmarshalJSON(data []byte) error {
	var j endpointStatusJSON
	if err := json.Unmarshal(data, &j); err != nil {
		return err
	}
	es.Healthy = j.Healthy
	es.Status = j.Status
	es.Detail = j.Detail
	es.Latency = time.Duration(j.LatencyMs * 1e6)
	es.Type = j.Type
	es.Name = j.Name
	es.Host = j.Host
	es.Port = j.Port
	es.Critical = j.Critical
	es.Labels = j.Labels
	if j.LastCheckedAt != nil {
		es.LastCheckedAt = *j.LastCheckedAt
	}
	return nil
}
