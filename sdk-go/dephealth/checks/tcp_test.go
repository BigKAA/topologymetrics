package checks

import (
	"context"
	"net"
	"testing"

	"github.com/BigKAA/topologymetrics/dephealth"
)

func TestTCPChecker_Check_Success(t *testing.T) {
	// Запускаем TCP-сервер на случайном порту.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("не удалось запустить TCP listener: %v", err)
	}
	defer func() { _ = ln.Close() }()

	// Принимаем соединение в горутине.
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		_ = conn.Close()
	}()

	_, port, _ := net.SplitHostPort(ln.Addr().String())
	ep := dephealth.Endpoint{Host: "127.0.0.1", Port: port}

	checker := NewTCPChecker()
	if err := checker.Check(context.Background(), ep); err != nil {
		t.Errorf("ожидали успех, получили ошибку: %v", err)
	}
}

func TestTCPChecker_Check_ConnectionRefused(t *testing.T) {
	// Используем порт, на котором ничего не слушает.
	ep := dephealth.Endpoint{Host: "127.0.0.1", Port: "1"}

	checker := NewTCPChecker()
	err := checker.Check(context.Background(), ep)
	if err == nil {
		t.Error("ожидали ошибку для закрытого порта, получили nil")
	}
}

func TestTCPChecker_Check_ContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	ep := dephealth.Endpoint{Host: "127.0.0.1", Port: "9999"}

	checker := NewTCPChecker()
	err := checker.Check(ctx, ep)
	if err == nil {
		t.Error("ожидали ошибку для отменённого контекста, получили nil")
	}
}

func TestTCPChecker_Type(t *testing.T) {
	checker := NewTCPChecker()
	if got := checker.Type(); got != "tcp" {
		t.Errorf("Type() = %q, ожидали %q", got, "tcp")
	}
}
