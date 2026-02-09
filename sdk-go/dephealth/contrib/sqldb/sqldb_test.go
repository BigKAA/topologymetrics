package sqldb

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/BigKAA/topologymetrics/sdk-go/dephealth"
	_ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks" // регистрация фабрик
)

func TestFromDB(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("ошибка создания sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	// Ожидаем SELECT 1 при health check.
	mock.ExpectQuery("SELECT 1").WillReturnRows(sqlmock.NewRows([]string{"1"}).AddRow(1))

	reg := prometheus.NewRegistry()
	dh, err := dephealth.New("test-app",
		dephealth.WithRegisterer(reg),
		FromDB("pg-main", db,
			dephealth.FromParams("pg.svc", "5432"),
			dephealth.Critical(true),
		),
	)
	if err != nil {
		t.Fatalf("ошибка создания DepHealth: %v", err)
	}
	_ = dh
}

func TestFromDB_MissingAddr(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("ошибка создания sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	reg := prometheus.NewRegistry()
	_, err = dephealth.New("test-app",
		dephealth.WithRegisterer(reg),
		FromDB("pg-main", db, dephealth.Critical(true)),
	)
	if err == nil {
		t.Fatal("ожидали ошибку при отсутствии адреса")
	}
}

func TestFromMySQLDB(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("ошибка создания sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	mock.ExpectQuery("SELECT 1").WillReturnRows(sqlmock.NewRows([]string{"1"}).AddRow(1))

	reg := prometheus.NewRegistry()
	dh, err := dephealth.New("test-app",
		dephealth.WithRegisterer(reg),
		FromMySQLDB("mysql-main", db,
			dephealth.FromParams("mysql.svc", "3306"),
			dephealth.Critical(true),
		),
	)
	if err != nil {
		t.Fatalf("ошибка создания DepHealth: %v", err)
	}
	_ = dh
}
