package checks

import (
	"context"
	"fmt"
	"testing"

	"github.com/alicebob/miniredis/v2"

	"github.com/BigKAA/topologymetrics/dephealth"
)

func TestPostgresFactory_URLPassedAsDSN(t *testing.T) {
	dc := &dephealth.DependencyConfig{
		URL: "postgres://user:pass@pg.svc:5432/mydb",
	}
	checker := newPostgresFromConfig(dc)
	pg, ok := checker.(*PostgresChecker)
	if !ok {
		t.Fatal("ожидали *PostgresChecker")
	}
	if pg.dsn != dc.URL {
		t.Errorf("dsn = %q, ожидали %q", pg.dsn, dc.URL)
	}
}

func TestMySQLFactory_URLConvertedToDSN(t *testing.T) {
	dc := &dephealth.DependencyConfig{
		URL: "mysql://user:pass@mysql.svc:3306/mydb",
	}
	checker := newMySQLFromConfig(dc)
	my, ok := checker.(*MySQLChecker)
	if !ok {
		t.Fatal("ожидали *MySQLChecker")
	}
	want := "user:pass@tcp(mysql.svc:3306)/mydb"
	if my.dsn != want {
		t.Errorf("dsn = %q, ожидали %q", my.dsn, want)
	}
}

func TestRedisFactory_PasswordFromURL(t *testing.T) {
	mr := miniredis.RunT(t)
	mr.RequireAuth("secret")

	dc := &dephealth.DependencyConfig{
		URL: fmt.Sprintf("redis://:secret@%s:%s/0", mr.Host(), mr.Port()),
	}
	checker := newRedisFromConfig(dc)
	ep := dephealth.Endpoint{Host: mr.Host(), Port: mr.Port()}

	if err := checker.Check(context.Background(), ep); err != nil {
		t.Errorf("ожидали успех с password из URL, получили ошибку: %v", err)
	}
}

func TestRedisFactory_ExplicitPasswordOverridesURL(t *testing.T) {
	mr := miniredis.RunT(t)
	mr.RequireAuth("correct")

	dc := &dephealth.DependencyConfig{
		URL:           fmt.Sprintf("redis://:wrong@%s:%s/0", mr.Host(), mr.Port()),
		RedisPassword: "correct",
	}
	checker := newRedisFromConfig(dc)
	ep := dephealth.Endpoint{Host: mr.Host(), Port: mr.Port()}

	if err := checker.Check(context.Background(), ep); err != nil {
		t.Errorf("ожидали успех с явным password, получили ошибку: %v", err)
	}
}

func TestRedisFactory_DBFromURL(t *testing.T) {
	dc := &dephealth.DependencyConfig{
		URL: "redis://localhost:6379/3",
	}
	checker := newRedisFromConfig(dc)
	rc, ok := checker.(*RedisChecker)
	if !ok {
		t.Fatal("ожидали *RedisChecker")
	}
	if rc.db != 3 {
		t.Errorf("db = %d, ожидали 3", rc.db)
	}
}

func TestAMQPFactory_URLPassedAsAMQPURL(t *testing.T) {
	dc := &dephealth.DependencyConfig{
		URL: "amqp://user:pass@rabbit.svc:5672/orders",
	}
	checker := newAMQPFromConfig(dc)
	ac, ok := checker.(*AMQPChecker)
	if !ok {
		t.Fatal("ожидали *AMQPChecker")
	}
	if ac.url != dc.URL {
		t.Errorf("url = %q, ожидали %q", ac.url, dc.URL)
	}
}

func TestAMQPFactory_ExplicitAMQPURLHasPriority(t *testing.T) {
	dc := &dephealth.DependencyConfig{
		URL:     "amqp://user:pass@rabbit.svc:5672/orders",
		AMQPURL: "amqp://admin:admin@other.svc:5672/",
	}
	checker := newAMQPFromConfig(dc)
	ac, ok := checker.(*AMQPChecker)
	if !ok {
		t.Fatal("ожидали *AMQPChecker")
	}
	if ac.url != dc.AMQPURL {
		t.Errorf("url = %q, ожидали %q (explicit AMQPURL)", ac.url, dc.AMQPURL)
	}
}

func TestMySQLURLToDSN(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want string
	}{
		{
			name: "полный URL",
			url:  "mysql://user:pass@host:3306/db",
			want: "user:pass@tcp(host:3306)/db",
		},
		{
			name: "без пароля",
			url:  "mysql://user@host:3306/db",
			want: "user@tcp(host:3306)/db",
		},
		{
			name: "без credentials",
			url:  "mysql://host:3306/db",
			want: "@tcp(host:3306)/db",
		},
		{
			name: "с query параметрами",
			url:  "mysql://user:pass@host:3306/db?charset=utf8mb4",
			want: "user:pass@tcp(host:3306)/db?charset=utf8mb4",
		},
		{
			name: "без базы данных",
			url:  "mysql://user:pass@host:3306",
			want: "user:pass@tcp(host:3306)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mysqlURLToDSN(tt.url)
			if got != tt.want {
				t.Errorf("mysqlURLToDSN(%q) = %q, ожидали %q", tt.url, got, tt.want)
			}
		})
	}
}
