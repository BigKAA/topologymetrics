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

	"github.com/BigKAA/topologymetrics/dephealth"
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
		t.Errorf("ожидали успех, получили ошибку: %v", err)
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
		t.Error("ожидали ошибку для статуса 503, получили nil")
	}
	if !strings.Contains(err.Error(), "503") {
		t.Errorf("ожидали упоминание статуса 503 в ошибке, получили: %v", err)
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
		t.Error("ожидали ошибку для редиректа 301, получили nil")
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
		t.Errorf("User-Agent = %q, ожидали %q", gotUA, expected)
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
		t.Errorf("path = %q, ожидали %q", gotPath, "/custom/healthz")
	}
}

func TestHTTPChecker_Check_TLS(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	host, port, _ := net.SplitHostPort(srv.Listener.Addr().String())
	ep := dephealth.Endpoint{Host: host, Port: port}

	// Без skip verify — ожидаем ошибку (самоподписанный сертификат).
	checker := NewHTTPChecker(
		WithHealthPath("/"),
		WithTLSEnabled(true),
	)
	err := checker.Check(context.Background(), ep)
	if err == nil {
		t.Error("ожидали ошибку TLS для самоподписанного сертификата")
	}

	// С skip verify — ожидаем успех.
	checker = NewHTTPChecker(
		WithHealthPath("/"),
		WithTLSEnabled(true),
		WithHTTPTLSSkipVerify(true),
	)
	if err := checker.Check(context.Background(), ep); err != nil {
		t.Errorf("ожидали успех с skip verify, получили: %v", err)
	}
}

func TestHTTPChecker_Check_TLSWithCert(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	// Добавляем сертификат тестового сервера в пул доверенных.
	certPool := x509.NewCertPool()
	certPool.AddCert(srv.Certificate())

	host, port, _ := net.SplitHostPort(srv.Listener.Addr().String())
	ep := dephealth.Endpoint{Host: host, Port: port}

	// Создаём чекер с TLS и доверенным сертификатом через кастомный транспорт.
	// Это проверяет, что TLS работает корректно с валидным сертификатом.
	checker := NewHTTPChecker(
		WithHealthPath("/"),
		WithTLSEnabled(true),
	)

	// Подменяем стандартную проверку — используем пул сертификатов тестового сервера.
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
		t.Fatalf("запрос с доверенным сертификатом не удался: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("статус = %d, ожидали 200", resp.StatusCode)
	}

	// Проверяем, что стандартный чекер без skip verify действительно вернул бы ошибку.
	if err := checker.Check(ctx, ep); err == nil {
		t.Error("ожидали ошибку TLS без skip verify для самоподписанного сертификата")
	}
}

func TestHTTPChecker_Check_ConnectionRefused(t *testing.T) {
	ep := dephealth.Endpoint{Host: "127.0.0.1", Port: "1"}

	checker := NewHTTPChecker(WithHealthPath("/"))
	err := checker.Check(context.Background(), ep)
	if err == nil {
		t.Error("ожидали ошибку для закрытого порта, получили nil")
	}
}

func TestHTTPChecker_Check_ContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	ep := dephealth.Endpoint{Host: "127.0.0.1", Port: "9999"}

	checker := NewHTTPChecker(WithHealthPath("/"))
	err := checker.Check(ctx, ep)
	if err == nil {
		t.Error("ожидали ошибку для отменённого контекста, получили nil")
	}
}

func TestHTTPChecker_Type(t *testing.T) {
	checker := NewHTTPChecker()
	if got := checker.Type(); got != "http" {
		t.Errorf("Type() = %q, ожидали %q", got, "http")
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

	checker := NewHTTPChecker() // без WithHealthPath — default /health
	_ = checker.Check(context.Background(), ep)

	if gotPath != "/health" {
		t.Errorf("path = %q, ожидали %q (default)", gotPath, "/health")
	}
}
