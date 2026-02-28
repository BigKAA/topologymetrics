package dephealth

// StatusCategory is a typed alias for status category string values.
// It wraps the status constants used by the error classification system
// and the app_dependency_status metric.
type StatusCategory string

const (
	// StatusOK means the health check succeeded.
	StatusOK StatusCategory = "ok"
	// StatusTimeout means the health check exceeded its deadline.
	StatusTimeout StatusCategory = "timeout"
	// StatusConnectionError means the connection to the dependency failed.
	StatusConnectionError StatusCategory = "connection_error"
	// StatusDNSError means DNS resolution failed for the dependency host.
	StatusDNSError StatusCategory = "dns_error"
	// StatusAuthError means authentication with the dependency failed.
	StatusAuthError StatusCategory = "auth_error"
	// StatusTLSError means a TLS handshake or certificate error occurred.
	StatusTLSError StatusCategory = "tls_error"
	// StatusUnhealthy means the dependency responded but reported unhealthy status.
	StatusUnhealthy StatusCategory = "unhealthy"
	// StatusError means an unclassified error occurred during the health check.
	StatusError StatusCategory = "error"
	// StatusUnknown is used only for HealthDetails API before the first check completes.
	StatusUnknown StatusCategory = "unknown"
)

// AllStatusCategories contains the 8 status categories used for the
// app_dependency_status enum-pattern gauge. StatusUnknown is excluded
// as it is only used for the HealthDetails API.
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

// Error returns the error message, delegating to Cause if present.
func (e *ClassifiedCheckError) Error() string {
	if e.Cause != nil {
		return e.Cause.Error()
	}
	return e.Detail
}

// Unwrap returns the underlying cause for use with errors.Is/As.
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
