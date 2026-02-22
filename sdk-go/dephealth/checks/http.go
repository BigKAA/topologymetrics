package checks

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"maps"
	"net"
	"net/http"

	"github.com/BigKAA/topologymetrics/sdk-go/dephealth"
)

// HTTPOption configures the HTTPChecker.
type HTTPOption func(*HTTPChecker)

// HTTPChecker performs health checks via HTTP GET requests.
// The check succeeds if the final response status code is 2xx.
// Redirects (3xx) are followed automatically.
type HTTPChecker struct {
	healthPath    string
	tlsEnabled    bool
	tlsSkipVerify bool
	headers       map[string]string
}

// WithHealthPath sets the HTTP path for health checks (default "/health").
func WithHealthPath(path string) HTTPOption {
	return func(c *HTTPChecker) {
		c.healthPath = path
	}
}

// WithTLSEnabled enables HTTPS for health check requests.
func WithTLSEnabled(enabled bool) HTTPOption {
	return func(c *HTTPChecker) {
		c.tlsEnabled = enabled
	}
}

// WithHTTPTLSSkipVerify skips TLS certificate verification.
func WithHTTPTLSSkipVerify(skip bool) HTTPOption {
	return func(c *HTTPChecker) {
		c.tlsSkipVerify = skip
	}
}

// WithHeaders sets custom HTTP headers for health check requests.
func WithHeaders(headers map[string]string) HTTPOption {
	return func(c *HTTPChecker) {
		maps.Copy(c.headers, headers)
	}
}

// WithBearerToken sets a Bearer token for HTTP health check requests.
// Adds Authorization: Bearer <token> header.
func WithBearerToken(token string) HTTPOption {
	return func(c *HTTPChecker) {
		c.headers["Authorization"] = "Bearer " + token
	}
}

// WithBasicAuth sets Basic Auth credentials for HTTP health check requests.
// Adds Authorization: Basic <base64(username:password)> header.
func WithBasicAuth(username, password string) HTTPOption {
	return func(c *HTTPChecker) {
		encoded := base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
		c.headers["Authorization"] = "Basic " + encoded
	}
}

// NewHTTPChecker creates a new HTTP health checker with the given options.
func NewHTTPChecker(opts ...HTTPOption) *HTTPChecker {
	c := &HTTPChecker{
		healthPath: "/health",
		headers:    make(map[string]string),
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Check sends an HTTP GET request to the endpoint's health path.
// Returns nil if the response status code is 2xx, or an error otherwise.
func (c *HTTPChecker) Check(ctx context.Context, endpoint dephealth.Endpoint) error {
	scheme := "http"
	if c.tlsEnabled {
		scheme = "https"
	}

	addr := net.JoinHostPort(endpoint.Host, endpoint.Port)
	url := fmt.Sprintf("%s://%s%s", scheme, addr, c.healthPath)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("http create request: %w", err)
	}
	req.Header.Set("User-Agent", "dephealth/"+dephealth.Version)

	// Apply custom headers after User-Agent so they can override it.
	for k, v := range c.headers {
		req.Header.Set(k, v)
	}

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: c.tlsSkipVerify, //nolint:gosec // configurable by user
		},
	}

	client := &http.Client{
		Transport: transport,
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("http request %s: %w", url, err)
	}
	_ = resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// HTTP 401/403 → auth_error.
		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
			return &dephealth.ClassifiedCheckError{
				Category: dephealth.StatusAuthError,
				Detail:   "auth_error",
				Cause:    fmt.Errorf("http status %d from %s", resp.StatusCode, url),
			}
		}
		return &dephealth.ClassifiedCheckError{
			Category: dephealth.StatusUnhealthy,
			Detail:   fmt.Sprintf("http_%d", resp.StatusCode),
			Cause:    fmt.Errorf("http status %d from %s", resp.StatusCode, url),
		}
	}

	return nil
}

// Type returns the dependency type for this checker.
func (c *HTTPChecker) Type() string {
	return string(dephealth.TypeHTTP)
}
