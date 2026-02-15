package dephealth

import (
	"errors"
	"testing"
)

func TestAllStatusCategories_Count(t *testing.T) {
	if len(AllStatusCategories) != 8 {
		t.Errorf("expected 8 status categories, got %d", len(AllStatusCategories))
	}
}

func TestAllStatusCategories_Values(t *testing.T) {
	expected := map[StatusCategory]bool{
		StatusOK: true, StatusTimeout: true, StatusConnectionError: true,
		StatusDNSError: true, StatusAuthError: true, StatusTLSError: true,
		StatusUnhealthy: true, StatusError: true,
	}
	for _, s := range AllStatusCategories {
		if !expected[s] {
			t.Errorf("unexpected status category: %q", s)
		}
	}
}

func TestClassifiedCheckError_Implements_ClassifiedError(t *testing.T) {
	var _ ClassifiedError = &ClassifiedCheckError{}
}

func TestClassifiedCheckError_Error_WithCause(t *testing.T) {
	cause := errors.New("connection refused")
	e := &ClassifiedCheckError{
		Category: StatusConnectionError,
		Detail:   "connection_refused",
		Cause:    cause,
	}
	if e.Error() != "connection refused" {
		t.Errorf("expected cause message, got %q", e.Error())
	}
}

func TestClassifiedCheckError_Error_WithoutCause(t *testing.T) {
	e := &ClassifiedCheckError{
		Category: StatusUnhealthy,
		Detail:   "http_503",
	}
	if e.Error() != "http_503" {
		t.Errorf("expected detail as message, got %q", e.Error())
	}
}

func TestClassifiedCheckError_Unwrap(t *testing.T) {
	cause := errors.New("original")
	e := &ClassifiedCheckError{Category: StatusError, Detail: "error", Cause: cause}
	if !errors.Is(e, cause) {
		t.Error("Unwrap should allow errors.Is to match the cause")
	}
}

func TestClassifiedCheckError_StatusCategory(t *testing.T) {
	e := &ClassifiedCheckError{Category: StatusAuthError, Detail: "auth_error"}
	if e.StatusCategory() != StatusAuthError {
		t.Errorf("expected %q, got %q", StatusAuthError, e.StatusCategory())
	}
}

func TestClassifiedCheckError_StatusDetail(t *testing.T) {
	e := &ClassifiedCheckError{Category: StatusUnhealthy, Detail: "grpc_not_serving"}
	if e.StatusDetail() != "grpc_not_serving" {
		t.Errorf("expected %q, got %q", "grpc_not_serving", e.StatusDetail())
	}
}

func TestClassifiedCheckError_ErrorsAs(t *testing.T) {
	e := &ClassifiedCheckError{Category: StatusTimeout, Detail: "timeout"}
	wrapped := errors.Join(errors.New("wrapper"), e)

	var ce ClassifiedError
	if !errors.As(wrapped, &ce) {
		t.Error("errors.As should find ClassifiedError in wrapped error")
	}
	if ce.StatusCategory() != StatusTimeout {
		t.Errorf("expected %q, got %q", StatusTimeout, ce.StatusCategory())
	}
}
