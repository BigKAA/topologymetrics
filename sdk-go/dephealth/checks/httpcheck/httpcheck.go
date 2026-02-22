// Package httpcheck provides an HTTP health checker for dephealth.
//
// Import this package to register the HTTP checker factory:
//
//	import _ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/httpcheck"
package httpcheck

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

func init() {
	dephealth.RegisterCheckerFactory(dephealth.TypeHTTP, NewFromConfig)
}

// Option configures the Checker.
type Option func(*Checker)

// Checker performs health checks via HTTP GET requests.
// The check succeeds if the final response status code is 2xx.
// Redirects (3xx) are followed automatically.
type Checker struct {
	healthPath    string
	tlsEnabled    bool
	tlsSkipVerify bool
	headers       map[string]string
}

// WithHealthPath sets the HTTP path for health checks (default "/health").
func WithHealthPath(path string) Option {
	return func(c *Checker) {
		c.healthPath = path
	}
}

// WithTLSEnabled enables HTTPS for health check requests.
func WithTLSEnabled(enabled bool) Option {
	return func(c *Checker) {
		c.tlsEnabled = enabled
	}
}

// WithTLSSkipVerify skips TLS certificate verification.
func WithTLSSkipVerify(skip bool) Option {
	return func(c *Checker) {
		c.tlsSkipVerify = skip
	}
}

// WithHeaders sets custom HTTP headers for health check requests.
func WithHeaders(headers map[string]string) Option {
	return func(c *Checker) {
		maps.Copy(c.headers, headers)
	}
}

// WithBearerToken sets a Bearer token for HTTP health check requests.
// Adds Authorization: Bearer <token> header.
func WithBearerToken(token string) Option {
	return func(c *Checker) {
		c.headers["Authorization"] = "Bearer " + token
	}
}

// WithBasicAuth sets Basic Auth credentials for HTTP health check requests.
// Adds Authorization: Basic <base64(username:password)> header.
func WithBasicAuth(username, password string) Option {
	return func(c *Checker) {
		encoded := base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
		c.headers["Authorization"] = "Basic " + encoded
	}
}

// New creates a new HTTP health checker with the given options.
func New(opts ...Option) *Checker {
	c := &Checker{
		healthPath: "/health",
		headers:    make(map[string]string),
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// NewFromConfig creates an HTTP checker from DependencyConfig.
func NewFromConfig(dc *dephealth.DependencyConfig) dephealth.HealthChecker {
	var opts []Option
	if dc.HTTPHealthPath != "" {
		opts = append(opts, WithHealthPath(dc.HTTPHealthPath))
	}
	if dc.HTTPTLS != nil {
		opts = append(opts, WithTLSEnabled(*dc.HTTPTLS))
	}
	if dc.HTTPTLSSkipVerify != nil {
		opts = append(opts, WithTLSSkipVerify(*dc.HTTPTLSSkipVerify))
	}
	if len(dc.HTTPHeaders) > 0 {
		opts = append(opts, WithHeaders(dc.HTTPHeaders))
	}
	if dc.HTTPBearerToken != "" {
		opts = append(opts, WithBearerToken(dc.HTTPBearerToken))
	}
	if dc.HTTPBasicUser != "" {
		opts = append(opts, WithBasicAuth(dc.HTTPBasicUser, dc.HTTPBasicPass))
	}
	return New(opts...)
}

// Check sends an HTTP GET request to the endpoint's health path.
// Returns nil if the response status code is 2xx, or an error otherwise.
func (c *Checker) Check(ctx context.Context, endpoint dephealth.Endpoint) error {
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
func (c *Checker) Type() string {
	return string(dephealth.TypeHTTP)
}
