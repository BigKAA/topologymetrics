package dephealth

// Status category constants used for app_dependency_status metric.
const (
	StatusOK              = "ok"
	StatusTimeout         = "timeout"
	StatusConnectionError = "connection_error"
	StatusDNSError        = "dns_error"
	StatusAuthError       = "auth_error"
	StatusTLSError        = "tls_error"
	StatusUnhealthy       = "unhealthy"
	StatusError           = "error"
)

// AllStatusCategories contains all possible status category values.
// Used to initialize all 8 series of the enum-pattern gauge.
var AllStatusCategories = []string{
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
	Category string // One of Status* constants.
	Detail   string // Specific detail value (e.g. "http_503", "grpc_not_serving").
}

// ClassifiedError is an interface for errors that carry status classification.
// Health checkers can return errors implementing this interface to provide
// precise status category and detail for the app_dependency_status metrics.
type ClassifiedError interface {
	error
	StatusCategory() string
	StatusDetail() string
}

// ClassifiedCheckError is a concrete error type that implements ClassifiedError.
// Checkers use this to return classified errors.
type ClassifiedCheckError struct {
	Category string
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
func (e *ClassifiedCheckError) StatusCategory() string {
	return e.Category
}

// StatusDetail returns the detail value for this error.
func (e *ClassifiedCheckError) StatusDetail() string {
	return e.Detail
}
