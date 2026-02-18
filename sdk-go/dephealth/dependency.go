// Package dephealth provides SDK for monitoring dependencies of microservices.
// Each dependency is checked periodically and exposes Prometheus metrics
// (app_dependency_health gauge and app_dependency_latency_seconds histogram).
package dephealth

import (
	"fmt"
	"regexp"
	"time"
)

// DependencyType represents the type of a dependency connection.
type DependencyType string

const (
	TypeHTTP     DependencyType = "http"
	TypeGRPC     DependencyType = "grpc"
	TypeTCP      DependencyType = "tcp"
	TypePostgres DependencyType = "postgres"
	TypeMySQL    DependencyType = "mysql"
	TypeRedis    DependencyType = "redis"
	TypeAMQP     DependencyType = "amqp"
	TypeKafka    DependencyType = "kafka"
)

// ValidTypes contains all valid dependency types.
var ValidTypes = map[DependencyType]bool{
	TypeHTTP:     true,
	TypeGRPC:     true,
	TypeTCP:      true,
	TypePostgres: true,
	TypeMySQL:    true,
	TypeRedis:    true,
	TypeAMQP:     true,
	TypeKafka:    true,
}

// Default values from specification.
const (
	DefaultCheckInterval    = 15 * time.Second
	DefaultTimeout          = 5 * time.Second
	DefaultInitialDelay     = 5 * time.Second
	DefaultFailureThreshold = 1
	DefaultSuccessThreshold = 1

	MinCheckInterval = 1 * time.Second
	MaxCheckInterval = 10 * time.Minute
	MinTimeout       = 100 * time.Millisecond
	MaxTimeout       = 30 * time.Second
	MinInitialDelay  = 0
	MaxInitialDelay  = 5 * time.Minute
	MinThreshold     = 1
	MaxThreshold     = 10
)

// namePattern validates dependency names: lowercase letters, digits, hyphens.
var namePattern = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

// labelNamePattern validates custom label names per Prometheus naming conventions.
var labelNamePattern = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// reservedLabels contains label names that cannot be used as custom labels.
var reservedLabels = map[string]bool{
	"name":       true,
	"group":      true,
	"dependency": true,
	"type":       true,
	"host":       true,
	"port":       true,
	"critical":   true,
}

const (
	minNameLen = 1
	maxNameLen = 63
)

// CheckConfig holds parameters for health check scheduling.
type CheckConfig struct {
	Interval         time.Duration
	Timeout          time.Duration
	InitialDelay     time.Duration
	FailureThreshold int
	SuccessThreshold int
}

// DefaultCheckConfig returns CheckConfig with default values from specification.
func DefaultCheckConfig() CheckConfig {
	return CheckConfig{
		Interval:         DefaultCheckInterval,
		Timeout:          DefaultTimeout,
		InitialDelay:     DefaultInitialDelay,
		FailureThreshold: DefaultFailureThreshold,
		SuccessThreshold: DefaultSuccessThreshold,
	}
}

// Validate checks that config values are within allowed ranges.
func (c CheckConfig) Validate() error {
	if c.Interval < MinCheckInterval || c.Interval > MaxCheckInterval {
		return fmt.Errorf("checkInterval %s out of range [%s, %s]", c.Interval, MinCheckInterval, MaxCheckInterval)
	}
	if c.Timeout < MinTimeout || c.Timeout > MaxTimeout {
		return fmt.Errorf("timeout %s out of range [%s, %s]", c.Timeout, MinTimeout, MaxTimeout)
	}
	if c.Timeout >= c.Interval {
		return fmt.Errorf("timeout %s must be less than checkInterval %s", c.Timeout, c.Interval)
	}
	if c.InitialDelay < MinInitialDelay || c.InitialDelay > MaxInitialDelay {
		return fmt.Errorf("initialDelay %s out of range [%v, %s]", c.InitialDelay, MinInitialDelay, MaxInitialDelay)
	}
	if c.FailureThreshold < MinThreshold || c.FailureThreshold > MaxThreshold {
		return fmt.Errorf("failureThreshold %d out of range [%d, %d]", c.FailureThreshold, MinThreshold, MaxThreshold)
	}
	if c.SuccessThreshold < MinThreshold || c.SuccessThreshold > MaxThreshold {
		return fmt.Errorf("successThreshold %d out of range [%d, %d]", c.SuccessThreshold, MinThreshold, MaxThreshold)
	}
	return nil
}

// Endpoint represents a single network endpoint of a dependency.
type Endpoint struct {
	Host   string
	Port   string
	Labels map[string]string // Custom labels via WithLabel.
}

// Dependency describes a monitored dependency.
type Dependency struct {
	Name      string
	Type      DependencyType
	Critical  *bool // nil = not set (validation error)
	Endpoints []Endpoint
	Config    CheckConfig
}

// ValidateName checks that a dependency name follows the naming rules.
func ValidateName(name string) error {
	if len(name) < minNameLen || len(name) > maxNameLen {
		return fmt.Errorf("invalid dependency name %q: length must be %d-%d", name, minNameLen, maxNameLen)
	}
	if !namePattern.MatchString(name) {
		return fmt.Errorf("invalid dependency name %q: must match [a-z][a-z0-9-]*", name)
	}
	return nil
}

// ValidateLabelName checks that a custom label name is valid.
func ValidateLabelName(name string) error {
	if !labelNamePattern.MatchString(name) {
		return fmt.Errorf("invalid label name: %q", name)
	}
	if reservedLabels[name] {
		return fmt.Errorf("reserved label: %q", name)
	}
	return nil
}

// ValidateLabels checks that all custom labels have valid names.
func ValidateLabels(labels map[string]string) error {
	for k := range labels {
		if err := ValidateLabelName(k); err != nil {
			return err
		}
	}
	return nil
}

// BoolToYesNo converts a bool to "yes"/"no" for the critical label.
func BoolToYesNo(v bool) string {
	if v {
		return "yes"
	}
	return "no"
}

// Validate checks that the dependency configuration is valid.
func (d Dependency) Validate() error {
	if err := ValidateName(d.Name); err != nil {
		return err
	}
	if !ValidTypes[d.Type] {
		return fmt.Errorf("unknown dependency type %q", d.Type)
	}
	if d.Critical == nil {
		return fmt.Errorf("missing critical for dependency %q", d.Name)
	}
	if len(d.Endpoints) == 0 {
		return fmt.Errorf("dependency %q has no endpoints", d.Name)
	}
	for i, ep := range d.Endpoints {
		if ep.Host == "" {
			return fmt.Errorf("missing host for dependency %q endpoint %d", d.Name, i)
		}
		if ep.Port == "" {
			return fmt.Errorf("missing port for dependency %q endpoint %d", d.Name, i)
		}
		if err := ValidateLabels(ep.Labels); err != nil {
			return fmt.Errorf("dependency %q endpoint %d: %w", d.Name, i, err)
		}
	}
	return d.Config.Validate()
}
