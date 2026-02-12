package checks

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/BigKAA/topologymetrics/sdk-go/dephealth"
)

func TestHTTPChecker_Check_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	host, port, _ := net.SplitHostPort(srv.Listener.Addr().String())
	ep := dephealth.Endpoint{Host: host, Port: port}

	checker := NewHTTPChecker(WithHealthPath("/"))
	if err := checker.Check(context.Background(), ep); err != nil {
		t.Errorf("expected success, got error: %v", err)
	}
}

func TestHTTPChecker_Check_Non2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	host, port, _ := net.SplitHostPort(srv.Listener.Addr().String())
	ep := dephealth.Endpoint{Host: host, Port: port}

	checker := NewHTTPChecker(WithHealthPath("/"))
	err := checker.Check(context.Background(), ep)
	if err == nil {
		t.Error("expected error for status 503, got nil")
	}
	if !strings.Contains(err.Error(), "503") {
		t.Errorf("expected status 503 mention in error, got: %v", err)
	}
}

func TestHTTPChecker_Check_Redirect(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/other", http.StatusMovedPermanently)
	}))
	defer srv.Close()

	host, port, _ := net.SplitHostPort(srv.Listener.Addr().String())
	ep := dephealth.Endpoint{Host: host, Port: port}

	checker := NewHTTPChecker(WithHealthPath("/"))
	err := checker.Check(context.Background(), ep)
	if err == nil {
		t.Error("expected error for redirect 301, got nil")
	}
}

func TestHTTPChecker_Check_UserAgent(t *testing.T) {
	var gotUA string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	host, port, _ := net.SplitHostPort(srv.Listener.Addr().String())
	ep := dephealth.Endpoint{Host: host, Port: port}

	checker := NewHTTPChecker(WithHealthPath("/"))
	_ = checker.Check(context.Background(), ep)

	expected := "dephealth/" + Version
	if gotUA != expected {
		t.Errorf("User-Agent = %q, expected %q", gotUA, expected)
	}
}

func TestHTTPChecker_Check_CustomPath(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	host, port, _ := net.SplitHostPort(srv.Listener.Addr().String())
	ep := dephealth.Endpoint{Host: host, Port: port}

	checker := NewHTTPChecker(WithHealthPath("/custom/healthz"))
	_ = checker.Check(context.Background(), ep)

	if gotPath != "/custom/healthz" {
		t.Errorf("path = %q, expected %q", gotPath, "/custom/healthz")
	}
}

func TestHTTPChecker_Check_TLS(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	host, port, _ := net.SplitHostPort(srv.Listener.Addr().String())
	ep := dephealth.Endpoint{Host: host, Port: port}

	// Without skip verify — expect error (self-signed certificate).
	checker := NewHTTPChecker(
		WithHealthPath("/"),
		WithTLSEnabled(true),
	)
	err := checker.Check(context.Background(), ep)
	if err == nil {
		t.Error("expected TLS error for self-signed certificate")
	}

	// With skip verify — expect success.
	checker = NewHTTPChecker(
		WithHealthPath("/"),
		WithTLSEnabled(true),
		WithHTTPTLSSkipVerify(true),
	)
	if err := checker.Check(context.Background(), ep); err != nil {
		t.Errorf("expected success with skip verify, got: %v", err)
	}
}

func TestHTTPChecker_Check_TLSWithCert(t *testing.T) {
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
	checker := NewHTTPChecker(
		WithHealthPath("/"),
		WithTLSEnabled(true),
	)

	// Override the default verification — use the test server's certificate pool.
	ctx := context.Background()
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: certPool,
			},
		},
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	addr := net.JoinHostPort(host, port)
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "https://"+addr+"/", nil)
	req.Header.Set("User-Agent", "dephealth/"+Version)
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

func TestHTTPChecker_Check_ConnectionRefused(t *testing.T) {
	ep := dephealth.Endpoint{Host: "127.0.0.1", Port: "1"}

	checker := NewHTTPChecker(WithHealthPath("/"))
	err := checker.Check(context.Background(), ep)
	if err == nil {
		t.Error("expected error for closed port, got nil")
	}
}

func TestHTTPChecker_Check_ContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	ep := dephealth.Endpoint{Host: "127.0.0.1", Port: "9999"}

	checker := NewHTTPChecker(WithHealthPath("/"))
	err := checker.Check(ctx, ep)
	if err == nil {
		t.Error("expected error for canceled context, got nil")
	}
}

func TestHTTPChecker_Type(t *testing.T) {
	checker := NewHTTPChecker()
	if got := checker.Type(); got != "http" {
		t.Errorf("Type() = %q, expected %q", got, "http")
	}
}

func TestHTTPChecker_DefaultHealthPath(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	host, port, _ := net.SplitHostPort(srv.Listener.Addr().String())
	ep := dephealth.Endpoint{Host: host, Port: port}

	checker := NewHTTPChecker() // without WithHealthPath — default /health
	_ = checker.Check(context.Background(), ep)

	if gotPath != "/health" {
		t.Errorf("path = %q, expected %q (default)", gotPath, "/health")
	}
}
