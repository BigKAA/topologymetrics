package dephealth

import (
	"context"
	"errors"
	"fmt"
	"net"
	"syscall"
	"testing"
)

func TestClassifyError_Nil(t *testing.T) {
	r := classifyError(nil)
	if r.Category != StatusOK || r.Detail != "ok" {
		t.Errorf("nil error: expected ok/ok, got %s/%s", r.Category, r.Detail)
	}
}

func TestClassifyError_ClassifiedError(t *testing.T) {
	err := &ClassifiedCheckError{
		Category: StatusUnhealthy,
		Detail:   "http_503",
		Cause:    errors.New("HTTP 503"),
	}
	r := classifyError(err)
	if r.Category != StatusUnhealthy || r.Detail != "http_503" {
		t.Errorf("expected unhealthy/http_503, got %s/%s", r.Category, r.Detail)
	}
}

func TestClassifyError_WrappedClassifiedError(t *testing.T) {
	inner := &ClassifiedCheckError{
		Category: StatusAuthError,
		Detail:   "auth_error",
	}
	err := fmt.Errorf("check failed: %w", inner)
	r := classifyError(err)
	if r.Category != StatusAuthError || r.Detail != "auth_error" {
		t.Errorf("expected auth_error/auth_error, got %s/%s", r.Category, r.Detail)
	}
}

func TestClassifyError_SentinelTimeout(t *testing.T) {
	r := classifyError(ErrTimeout)
	if r.Category != StatusTimeout || r.Detail != "timeout" {
		t.Errorf("expected timeout/timeout, got %s/%s", r.Category, r.Detail)
	}
}

func TestClassifyError_SentinelConnectionRefused(t *testing.T) {
	r := classifyError(ErrConnectionRefused)
	if r.Category != StatusConnectionError || r.Detail != "connection_refused" {
		t.Errorf("expected connection_error/connection_refused, got %s/%s", r.Category, r.Detail)
	}
}

func TestClassifyError_SentinelUnhealthy(t *testing.T) {
	r := classifyError(ErrUnhealthy)
	if r.Category != StatusUnhealthy || r.Detail != "unhealthy" {
		t.Errorf("expected unhealthy/unhealthy, got %s/%s", r.Category, r.Detail)
	}
}

func TestClassifyError_WrappedSentinel(t *testing.T) {
	err := fmt.Errorf("postgres: %w", ErrConnectionRefused)
	r := classifyError(err)
	if r.Category != StatusConnectionError {
		t.Errorf("expected connection_error, got %s", r.Category)
	}
}

func TestClassifyError_DeadlineExceeded(t *testing.T) {
	r := classifyError(context.DeadlineExceeded)
	if r.Category != StatusTimeout || r.Detail != "timeout" {
		t.Errorf("expected timeout/timeout, got %s/%s", r.Category, r.Detail)
	}
}

func TestClassifyError_WrappedDeadlineExceeded(t *testing.T) {
	err := fmt.Errorf("query: %w", context.DeadlineExceeded)
	r := classifyError(err)
	if r.Category != StatusTimeout {
		t.Errorf("expected timeout, got %s", r.Category)
	}
}

func TestClassifyError_DNSError(t *testing.T) {
	dnsErr := &net.DNSError{
		Err:  "no such host",
		Name: "unknown.svc",
	}
	r := classifyError(dnsErr)
	if r.Category != StatusDNSError || r.Detail != "dns_error" {
		t.Errorf("expected dns_error/dns_error, got %s/%s", r.Category, r.Detail)
	}
}

func TestClassifyError_WrappedDNSError(t *testing.T) {
	dnsErr := &net.DNSError{Err: "no such host", Name: "test.svc"}
	err := fmt.Errorf("dial: %w", dnsErr)
	r := classifyError(err)
	if r.Category != StatusDNSError {
		t.Errorf("expected dns_error, got %s", r.Category)
	}
}

func TestClassifyError_ConnectionRefused_OpError(t *testing.T) {
	opErr := &net.OpError{
		Op:  "dial",
		Net: "tcp",
		Err: syscall.ECONNREFUSED,
	}
	r := classifyError(opErr)
	if r.Category != StatusConnectionError || r.Detail != "connection_refused" {
		t.Errorf("expected connection_error/connection_refused, got %s/%s", r.Category, r.Detail)
	}
}

func TestClassifyError_TLSError_ByMessage(t *testing.T) {
	err := errors.New("tls: failed to verify certificate")
	r := classifyError(err)
	if r.Category != StatusTLSError || r.Detail != "tls_error" {
		t.Errorf("expected tls_error/tls_error, got %s/%s", r.Category, r.Detail)
	}
}

func TestClassifyError_X509Error_ByMessage(t *testing.T) {
	err := errors.New("x509: certificate signed by unknown authority")
	r := classifyError(err)
	if r.Category != StatusTLSError {
		t.Errorf("expected tls_error, got %s", r.Category)
	}
}

func TestClassifyError_Fallback(t *testing.T) {
	err := errors.New("some unknown error")
	r := classifyError(err)
	if r.Category != StatusError || r.Detail != "error" {
		t.Errorf("expected error/error, got %s/%s", r.Category, r.Detail)
	}
}

func TestClassifyError_ClassifiedError_TakesPriority(t *testing.T) {
	// ClassifiedError should take priority over platform detection.
	inner := &ClassifiedCheckError{
		Category: StatusAuthError,
		Detail:   "auth_error",
		Cause:    context.DeadlineExceeded, // Would otherwise be classified as timeout.
	}
	r := classifyError(inner)
	if r.Category != StatusAuthError {
		t.Errorf("ClassifiedError should take priority, got %s", r.Category)
	}
}
