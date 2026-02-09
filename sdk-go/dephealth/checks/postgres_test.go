package checks

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/BigKAA/topologymetrics/sdk-go/dephealth"
)

func TestPostgresChecker_Check_PoolMode(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("не удалось создать sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	mock.ExpectQuery("SELECT 1").WillReturnRows(sqlmock.NewRows([]string{"1"}).AddRow(1))

	checker := NewPostgresChecker(WithPostgresDB(db))
	ep := dephealth.Endpoint{Host: "ignored", Port: "5432"}

	if err := checker.Check(context.Background(), ep); err != nil {
		t.Errorf("ожидали успех в pool mode, получили ошибку: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("не все ожидания sqlmock выполнены: %v", err)
	}
}

func TestPostgresChecker_Check_PoolMode_Error(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("не удалось создать sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	mock.ExpectQuery("SELECT 1").WillReturnError(context.DeadlineExceeded)

	checker := NewPostgresChecker(WithPostgresDB(db))
	ep := dephealth.Endpoint{Host: "ignored", Port: "5432"}

	if err := checker.Check(context.Background(), ep); err == nil {
		t.Error("ожидали ошибку pool query, получили nil")
	}
}

func TestPostgresChecker_Check_PoolMode_CustomQuery(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("не удалось создать sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	mock.ExpectQuery("SELECT version()").WillReturnRows(sqlmock.NewRows([]string{"version"}).AddRow("15.0"))

	checker := NewPostgresChecker(WithPostgresDB(db), WithPostgresQuery("SELECT version()"))
	ep := dephealth.Endpoint{Host: "ignored", Port: "5432"}

	if err := checker.Check(context.Background(), ep); err != nil {
		t.Errorf("ожидали успех с custom query, получили ошибку: %v", err)
	}
}

func TestPostgresChecker_Check_Standalone_ConnectionRefused(t *testing.T) {
	checker := NewPostgresChecker()
	ep := dephealth.Endpoint{Host: "127.0.0.1", Port: "1"}

	err := checker.Check(context.Background(), ep)
	if err == nil {
		t.Error("ожидали ошибку для закрытого порта, получили nil")
	}
}

func TestPostgresChecker_Check_Standalone_ContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	checker := NewPostgresChecker()
	ep := dephealth.Endpoint{Host: "127.0.0.1", Port: "5432"}

	err := checker.Check(ctx, ep)
	if err == nil {
		t.Error("ожидали ошибку для отменённого контекста, получили nil")
	}
}

func TestPostgresChecker_Type(t *testing.T) {
	checker := NewPostgresChecker()
	if got := checker.Type(); got != "postgres" {
		t.Errorf("Type() = %q, ожидали %q", got, "postgres")
	}
}
