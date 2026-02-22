package pgcheck

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/BigKAA/topologymetrics/sdk-go/dephealth"
)

func TestChecker_Check_PoolMode(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	mock.ExpectQuery("SELECT 1").WillReturnRows(sqlmock.NewRows([]string{"1"}).AddRow(1))

	checker := New(WithDB(db))
	ep := dephealth.Endpoint{Host: "ignored", Port: "5432"}

	if err := checker.Check(context.Background(), ep); err != nil {
		t.Errorf("expected success in pool mode, got error: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("not all sqlmock expectations were met: %v", err)
	}
}

func TestChecker_Check_PoolMode_Error(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	mock.ExpectQuery("SELECT 1").WillReturnError(context.DeadlineExceeded)

	checker := New(WithDB(db))
	ep := dephealth.Endpoint{Host: "ignored", Port: "5432"}

	if err := checker.Check(context.Background(), ep); err == nil {
		t.Error("expected pool query error, got nil")
	}
}

func TestChecker_Check_PoolMode_CustomQuery(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	mock.ExpectQuery("SELECT version()").WillReturnRows(sqlmock.NewRows([]string{"version"}).AddRow("15.0"))

	checker := New(WithDB(db), WithQuery("SELECT version()"))
	ep := dephealth.Endpoint{Host: "ignored", Port: "5432"}

	if err := checker.Check(context.Background(), ep); err != nil {
		t.Errorf("expected success with custom query, got error: %v", err)
	}
}

func TestChecker_Check_Standalone_ConnectionRefused(t *testing.T) {
	checker := New()
	ep := dephealth.Endpoint{Host: "127.0.0.1", Port: "1"}

	err := checker.Check(context.Background(), ep)
	if err == nil {
		t.Error("expected error for closed port, got nil")
	}
}

func TestChecker_Check_Standalone_ContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	checker := New()
	ep := dephealth.Endpoint{Host: "127.0.0.1", Port: "5432"}

	err := checker.Check(ctx, ep)
	if err == nil {
		t.Error("expected error for canceled context, got nil")
	}
}

func TestChecker_Type(t *testing.T) {
	checker := New()
	if got := checker.Type(); got != "postgres" {
		t.Errorf("Type() = %q, expected %q", got, "postgres")
	}
}

func TestNewFromConfig_URLPassedAsDSN(t *testing.T) {
	dc := &dephealth.DependencyConfig{
		URL: "postgres://user:pass@pg.svc:5432/mydb",
	}
	checker := NewFromConfig(dc)
	pg, ok := checker.(*Checker)
	if !ok {
		t.Fatal("expected *Checker")
	}
	if pg.dsn != dc.URL {
		t.Errorf("dsn = %q, expected %q", pg.dsn, dc.URL)
	}
}
