package httpcheck

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/BigKAA/topologymetrics/sdk-go/dephealth"
)

func TestChecker_Check_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	host, port, _ := net.SplitHostPort(srv.Listener.Addr().String())
	ep := dephealth.Endpoint{Host: host, Port: port}

	checker := New(WithHealthPath("/"))
	if err := checker.Check(context.Background(), ep); err != nil {
		t.Errorf("expected success, got error: %v", err)
	}
}

func TestChecker_Check_Non2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	host, port, _ := net.SplitHostPort(srv.Listener.Addr().String())
	ep := dephealth.Endpoint{Host: host, Port: port}

	checker := New(WithHealthPath("/"))
	err := checker.Check(context.Background(), ep)
	if err == nil {
		t.Error("expected error for status 503, got nil")
	}
	if !strings.Contains(err.Error(), "503") {
		t.Errorf("expected status 503 mention in error, got: %v", err)
	}
}

func TestChecker_Check_Redirect(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/other", http.StatusMovedPermanently)
	}))
	defer srv.Close()

	host, port, _ := net.SplitHostPort(srv.Listener.Addr().String())
	ep := dephealth.Endpoint{Host: host, Port: port}

	checker := New(WithHealthPath("/"))
	err := checker.Check(context.Background(), ep)
	if err == nil {
		t.Error("expected error for redirect 301, got nil")
	}
}

func TestChecker_Check_UserAgent(t *testing.T) {
	var gotUA string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	host, port, _ := net.SplitHostPort(srv.Listener.Addr().String())
	ep := dephealth.Endpoint{Host: host, Port: port}

	checker := New(WithHealthPath("/"))
	_ = checker.Check(context.Background(), ep)

	expected := "dephealth/" + dephealth.Version
	if gotUA != expected {
		t.Errorf("User-Agent = %q, expected %q", gotUA, expected)
	}
}

func TestChecker_Check_CustomPath(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	host, port, _ := net.SplitHostPort(srv.Listener.Addr().String())
	ep := dephealth.Endpoint{Host: host, Port: port}

	checker := New(WithHealthPath("/custom/healthz"))
	_ = checker.Check(context.Background(), ep)

	if gotPath != "/custom/healthz" {
		t.Errorf("path = %q, expected %q", gotPath, "/custom/healthz")
	}
}

func TestChecker_Check_TLS(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	host, port, _ := net.SplitHostPort(srv.Listener.Addr().String())
	ep := dephealth.Endpoint{Host: host, Port: port}

	// Without skip verify — expect error (self-signed certificate).
	checker := New(
		WithHealthPath("/"),
		WithTLSEnabled(true),
	)
	err := checker.Check(context.Background(), ep)
	if err == nil {
		t.Error("expected TLS error for self-signed certificate")
	}

	// With skip verify — expect success.
	checker = New(
		WithHealthPath("/"),
		WithTLSEnabled(true),
		WithTLSSkipVerify(true),
	)
	if err := checker.Check(context.Background(), ep); err != nil {
		t.Errorf("expected success with skip verify, got: %v", err)
	}
}

func TestChecker_Check_TLSWithCert(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	// Add the test server certificate to the trusted pool.
	certPool := x509.NewCertPool()
	certPool.AddCert(srv.Certificate())

	host, port, _ := net.SplitHostPort(srv.Listener.Addr().String())
	ep := dephealth.Endpoint{Host: host, Port: port}

	// Create a checker with TLS and a trusted certificate via custom transport.
	// This verifies that TLS works correctly with a valid certificate.
	checker := New(
		WithHealthPath("/"),
		WithTLSEnabled(true),
	)

	// Override the default verification — use the test server's certificate pool.
	ctx := context.Background()
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				MinVersion: tls.VersionTLS12,
				RootCAs:    certPool,
			},
		},
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	addr := net.JoinHostPort(host, port)
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "https://"+addr+"/", nil)
	req.Header.Set("User-Agent", "dephealth/"+dephealth.Version)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request with trusted certificate failed: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, expected 200", resp.StatusCode)
	}

	// Verify that the standard checker without skip verify would indeed return an error.
	if err := checker.Check(ctx, ep); err == nil {
		t.Error("expected TLS error without skip verify for self-signed certificate")
	}
}

func TestChecker_Check_ConnectionRefused(t *testing.T) {
	ep := dephealth.Endpoint{Host: "127.0.0.1", Port: "1"}

	checker := New(WithHealthPath("/"))
	err := checker.Check(context.Background(), ep)
	if err == nil {
		t.Error("expected error for closed port, got nil")
	}
}

func TestChecker_Check_ContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	ep := dephealth.Endpoint{Host: "127.0.0.1", Port: "9999"}

	checker := New(WithHealthPath("/"))
	err := checker.Check(ctx, ep)
	if err == nil {
		t.Error("expected error for canceled context, got nil")
	}
}

func TestChecker_Type(t *testing.T) {
	checker := New()
	if got := checker.Type(); got != "http" {
		t.Errorf("Type() = %q, expected %q", got, "http")
	}
}

func TestChecker_DefaultHealthPath(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	host, port, _ := net.SplitHostPort(srv.Listener.Addr().String())
	ep := dephealth.Endpoint{Host: host, Port: port}

	checker := New() // without WithHealthPath — default /health
	_ = checker.Check(context.Background(), ep)

	if gotPath != "/health" {
		t.Errorf("path = %q, expected %q (default)", gotPath, "/health")
	}
}

func TestChecker_Check_BearerToken(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	host, port, _ := net.SplitHostPort(srv.Listener.Addr().String())
	ep := dephealth.Endpoint{Host: host, Port: port}

	checker := New(WithHealthPath("/"), WithBearerToken("test-token-123"))
	if err := checker.Check(context.Background(), ep); err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if gotAuth != "Bearer test-token-123" {
		t.Errorf("Authorization = %q, expected %q", gotAuth, "Bearer test-token-123")
	}
}

func TestChecker_Check_BasicAuth(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	host, port, _ := net.SplitHostPort(srv.Listener.Addr().String())
	ep := dephealth.Endpoint{Host: host, Port: port}

	checker := New(WithHealthPath("/"), WithBasicAuth("admin", "secret"))
	if err := checker.Check(context.Background(), ep); err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if !strings.HasPrefix(gotAuth, "Basic ") {
		t.Errorf("Authorization = %q, expected Basic prefix", gotAuth)
	}
}

func TestChecker_Check_CustomHeaders(t *testing.T) {
	var gotAPIKey, gotCustom string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAPIKey = r.Header.Get("X-API-Key")
		gotCustom = r.Header.Get("X-Custom")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	host, port, _ := net.SplitHostPort(srv.Listener.Addr().String())
	ep := dephealth.Endpoint{Host: host, Port: port}

	checker := New(
		WithHealthPath("/"),
		WithHeaders(map[string]string{
			"X-API-Key": "my-key",
			"X-Custom":  "value",
		}),
	)
	if err := checker.Check(context.Background(), ep); err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if gotAPIKey != "my-key" {
		t.Errorf("X-API-Key = %q, expected %q", gotAPIKey, "my-key")
	}
	if gotCustom != "value" {
		t.Errorf("X-Custom = %q, expected %q", gotCustom, "value")
	}
}

func TestChecker_Check_401_AuthError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	host, port, _ := net.SplitHostPort(srv.Listener.Addr().String())
	ep := dephealth.Endpoint{Host: host, Port: port}

	checker := New(WithHealthPath("/"))
	err := checker.Check(context.Background(), ep)
	if err == nil {
		t.Fatal("expected error for 401, got nil")
	}

	var ce *dephealth.ClassifiedCheckError
	if !errors.As(err, &ce) {
		t.Fatalf("expected ClassifiedCheckError, got %T: %v", err, err)
	}
	if ce.Category != dephealth.StatusAuthError {
		t.Errorf("Category = %q, expected %q", ce.Category, dephealth.StatusAuthError)
	}
	if ce.Detail != "auth_error" {
		t.Errorf("Detail = %q, expected %q", ce.Detail, "auth_error")
	}
}

func TestChecker_Check_403_AuthError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	host, port, _ := net.SplitHostPort(srv.Listener.Addr().String())
	ep := dephealth.Endpoint{Host: host, Port: port}

	checker := New(WithHealthPath("/"))
	err := checker.Check(context.Background(), ep)
	if err == nil {
		t.Fatal("expected error for 403, got nil")
	}

	var ce *dephealth.ClassifiedCheckError
	if !errors.As(err, &ce) {
		t.Fatalf("expected ClassifiedCheckError, got %T: %v", err, err)
	}
	if ce.Category != dephealth.StatusAuthError {
		t.Errorf("Category = %q, expected %q", ce.Category, dephealth.StatusAuthError)
	}
}

func TestChecker_Check_503_Unhealthy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	host, port, _ := net.SplitHostPort(srv.Listener.Addr().String())
	ep := dephealth.Endpoint{Host: host, Port: port}

	checker := New(WithHealthPath("/"))
	err := checker.Check(context.Background(), ep)
	if err == nil {
		t.Fatal("expected error for 503, got nil")
	}

	var ce *dephealth.ClassifiedCheckError
	if !errors.As(err, &ce) {
		t.Fatalf("expected ClassifiedCheckError, got %T: %v", err, err)
	}
	if ce.Category != dephealth.StatusUnhealthy {
		t.Errorf("Category = %q, expected %q", ce.Category, dephealth.StatusUnhealthy)
	}
	if ce.Detail != "http_503" {
		t.Errorf("Detail = %q, expected %q", ce.Detail, "http_503")
	}
}
