package checks

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"

	"github.com/BigKAA/topologymetrics/dephealth"
)

// HTTPOption configures the HTTPChecker.
type HTTPOption func(*HTTPChecker)

// HTTPChecker performs health checks via HTTP GET requests.
// The check succeeds if the response status code is 2xx.
// Redirects are not followed (3xx is considered a failure).
type HTTPChecker struct {
	healthPath    string
	tlsEnabled    bool
	tlsSkipVerify bool
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

// NewHTTPChecker creates a new HTTP health checker with the given options.
func NewHTTPChecker(opts ...HTTPOption) *HTTPChecker {
	c := &HTTPChecker{
		healthPath: "/health",
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
	req.Header.Set("User-Agent", "dephealth/"+Version)

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: c.tlsSkipVerify, //nolint:gosec // configurable by user
		},
	}

	client := &http.Client{
		Transport: transport,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("http request %s: %w", url, err)
	}
	_ = resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("http status %d from %s", resp.StatusCode, url)
	}

	return nil
}

// Type returns the dependency type for this checker.
func (c *HTTPChecker) Type() string {
	return string(dephealth.TypeHTTP)
}
