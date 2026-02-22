// Package sqldb provides dephealth integration with *sql.DB (PostgreSQL, MySQL).
// It allows using an existing service connection pool for health checks.
package sqldb

import (
	"database/sql"

	"github.com/BigKAA/topologymetrics/sdk-go/dephealth"
	"github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/mysqlcheck"
	"github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/pgcheck"
)

// FromDB creates an Option for monitoring PostgreSQL via an existing *sql.DB.
// The user must provide FromURL or FromParams to determine metric labels.
func FromDB(name string, db *sql.DB, opts ...dephealth.DependencyOption) dephealth.Option {
	checker := pgcheck.New(pgcheck.WithDB(db))
	return dephealth.AddDependency(name, dephealth.TypePostgres, checker, opts...)
}

// FromMySQLDB creates an Option for monitoring MySQL via an existing *sql.DB.
// The user must provide FromURL or FromParams to determine metric labels.
func FromMySQLDB(name string, db *sql.DB, opts ...dephealth.DependencyOption) dephealth.Option {
	checker := mysqlcheck.New(mysqlcheck.WithDB(db))
	return dephealth.AddDependency(name, dephealth.TypeMySQL, checker, opts...)
}
