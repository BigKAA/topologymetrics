// Package sqldb предоставляет интеграцию dephealth с *sql.DB (PostgreSQL, MySQL).
// Позволяет использовать существующий connection pool сервиса для health-чеков.
package sqldb

import (
	"database/sql"

	"github.com/BigKAA/topologymetrics/dephealth"
	"github.com/BigKAA/topologymetrics/dephealth/checks"
)

// FromDB создаёт Option для мониторинга PostgreSQL через существующий *sql.DB.
// Пользователь обязан передать FromURL или FromParams для определения меток метрик.
func FromDB(name string, db *sql.DB, opts ...dephealth.DependencyOption) dephealth.Option {
	checker := checks.NewPostgresChecker(checks.WithPostgresDB(db))
	return dephealth.AddDependency(name, dephealth.TypePostgres, checker, opts...)
}

// FromMySQLDB создаёт Option для мониторинга MySQL через существующий *sql.DB.
// Пользователь обязан передать FromURL или FromParams для определения меток метрик.
func FromMySQLDB(name string, db *sql.DB, opts ...dephealth.DependencyOption) dephealth.Option {
	checker := checks.NewMySQLChecker(checks.WithMySQLDB(db))
	return dephealth.AddDependency(name, dephealth.TypeMySQL, checker, opts...)
}
