package dephealth

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"strings"
	"syscall"
)

// classifyError determines the CheckResult for a health check outcome.
// It follows the classification chain defined in the specification:
// 1. ClassifiedError interface
// 2. Sentinel errors
// 3. Platform error detection
// 4. Fallback → error/error
func classifyError(err error) CheckResult {
	if err == nil {
		return CheckResult{Category: StatusOK, Detail: StatusOK}
	}

	// 1. ClassifiedError interface — highest priority.
	var ce ClassifiedError
	if errors.As(err, &ce) {
		return CheckResult{Category: ce.StatusCategory(), Detail: ce.StatusDetail()}
	}

	// 2. Sentinel errors.
	if errors.Is(err, ErrTimeout) {
		return CheckResult{Category: StatusTimeout, Detail: "timeout"}
	}
	if errors.Is(err, ErrConnectionRefused) {
		return CheckResult{Category: StatusConnectionError, Detail: "connection_refused"}
	}
	if errors.Is(err, ErrUnhealthy) {
		return CheckResult{Category: StatusUnhealthy, Detail: "unhealthy"}
	}

	// 3. Platform error detection.

	// context.DeadlineExceeded — timeout.
	if errors.Is(err, context.DeadlineExceeded) {
		return CheckResult{Category: StatusTimeout, Detail: "timeout"}
	}

	// DNS errors.
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return CheckResult{Category: StatusDNSError, Detail: "dns_error"}
	}

	// Connection refused / network unreachable via OpError.
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		if isSyscallConnectionRefused(opErr.Err) {
			return CheckResult{Category: StatusConnectionError, Detail: "connection_refused"}
		}
		// Timeout inside OpError.
		if opErr.Timeout() {
			return CheckResult{Category: StatusTimeout, Detail: "timeout"}
		}
	}

	// TLS errors.
	var certErr *tls.CertificateVerificationError
	if errors.As(err, &certErr) {
		return CheckResult{Category: StatusTLSError, Detail: "tls_error"}
	}
	// Also detect TLS errors by common error message patterns.
	if isTLSError(err) {
		return CheckResult{Category: StatusTLSError, Detail: "tls_error"}
	}

	// 4. Fallback.
	return CheckResult{Category: StatusError, Detail: "error"}
}

// isSyscallConnectionRefused checks if the error is ECONNREFUSED.
func isSyscallConnectionRefused(err error) bool {
	var sysErr *syscall.Errno
	if errors.As(err, &sysErr) {
		return *sysErr == syscall.ECONNREFUSED
	}
	// Also check for direct syscall.Errno value.
	if errors.Is(err, syscall.ECONNREFUSED) {
		return true
	}
	return false
}

// isTLSError checks if the error message indicates a TLS error.
func isTLSError(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "tls:") ||
		strings.Contains(msg, "x509:") ||
		strings.Contains(msg, "certificate")
}
