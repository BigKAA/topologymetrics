package dephealth

// StatusCategory is a typed alias for status category string values.
// It wraps the status constants used by the error classification system
// and the app_dependency_status metric.
type StatusCategory string

// Status category constants used for app_dependency_status metric
// and the HealthDetails() API.
const (
	StatusOK              StatusCategory = "ok"
	StatusTimeout         StatusCategory = "timeout"
	StatusConnectionError StatusCategory = "connection_error"
	StatusDNSError        StatusCategory = "dns_error"
	StatusAuthError       StatusCategory = "auth_error"
	StatusTLSError        StatusCategory = "tls_error"
	StatusUnhealthy       StatusCategory = "unhealthy"
	StatusError           StatusCategory = "error"
	StatusUnknown         StatusCategory = "unknown"
)

// AllStatusCategories contains all possible status category values
// used for metrics (excludes StatusUnknown which is only for HealthDetails API).
// Used to initialize all 8 series of the enum-pattern gauge.
var AllStatusCategories = []StatusCategory{
	StatusOK,
	StatusTimeout,
	StatusConnectionError,
	StatusDNSError,
	StatusAuthError,
	StatusTLSError,
	StatusUnhealthy,
	StatusError,
}

// CheckResult holds the classification of a health check outcome.
type CheckResult struct {
	Category StatusCategory // One of Status* constants.
	Detail   string         // Specific detail value (e.g. "http_503", "grpc_not_serving").
}

// ClassifiedError is an interface for errors that carry status classification.
// Health checkers can return errors implementing this interface to provide
// precise status category and detail for the app_dependency_status metrics.
type ClassifiedError interface {
	error
	StatusCategory() StatusCategory
	StatusDetail() string
}

// ClassifiedCheckError is a concrete error type that implements ClassifiedError.
// Checkers use this to return classified errors.
type ClassifiedCheckError struct {
	Category StatusCategory
	Detail   string
	Cause    error
}

func (e *ClassifiedCheckError) Error() string {
	if e.Cause != nil {
		return e.Cause.Error()
	}
	return e.Detail
}

func (e *ClassifiedCheckError) Unwrap() error {
	return e.Cause
}

// StatusCategory returns the status category for this error.
func (e *ClassifiedCheckError) StatusCategory() StatusCategory {
	return e.Category
}

// StatusDetail returns the detail value for this error.
func (e *ClassifiedCheckError) StatusDetail() string {
	return e.Detail
}
