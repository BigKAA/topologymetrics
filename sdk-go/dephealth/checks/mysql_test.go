package checks

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/BigKAA/topologymetrics/sdk-go/dephealth"
)

func TestMySQLChecker_Check_PoolMode(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	mock.ExpectQuery("SELECT 1").WillReturnRows(sqlmock.NewRows([]string{"1"}).AddRow(1))

	checker := NewMySQLChecker(WithMySQLDB(db))
	ep := dephealth.Endpoint{Host: "ignored", Port: "3306"}

	if err := checker.Check(context.Background(), ep); err != nil {
		t.Errorf("expected success in pool mode, got error: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("not all sqlmock expectations were met: %v", err)
	}
}

func TestMySQLChecker_Check_PoolMode_Error(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	mock.ExpectQuery("SELECT 1").WillReturnError(context.DeadlineExceeded)

	checker := NewMySQLChecker(WithMySQLDB(db))
	ep := dephealth.Endpoint{Host: "ignored", Port: "3306"}

	if err := checker.Check(context.Background(), ep); err == nil {
		t.Error("expected pool query error, got nil")
	}
}

func TestMySQLChecker_Check_PoolMode_CustomQuery(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	mock.ExpectQuery("SELECT version()").WillReturnRows(sqlmock.NewRows([]string{"version"}).AddRow("8.0"))

	checker := NewMySQLChecker(WithMySQLDB(db), WithMySQLQuery("SELECT version()"))
	ep := dephealth.Endpoint{Host: "ignored", Port: "3306"}

	if err := checker.Check(context.Background(), ep); err != nil {
		t.Errorf("expected success with custom query, got error: %v", err)
	}
}

func TestMySQLChecker_Check_Standalone_ConnectionRefused(t *testing.T) {
	checker := NewMySQLChecker()
	ep := dephealth.Endpoint{Host: "127.0.0.1", Port: "1"}

	err := checker.Check(context.Background(), ep)
	if err == nil {
		t.Error("expected error for closed port, got nil")
	}
}

func TestMySQLChecker_Check_Standalone_ContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	checker := NewMySQLChecker()
	ep := dephealth.Endpoint{Host: "127.0.0.1", Port: "3306"}

	err := checker.Check(ctx, ep)
	if err == nil {
		t.Error("expected error for canceled context, got nil")
	}
}

func TestMySQLChecker_Type(t *testing.T) {
	checker := NewMySQLChecker()
	if got := checker.Type(); got != "mysql" {
		t.Errorf("Type() = %q, expected %q", got, "mysql")
	}
}
