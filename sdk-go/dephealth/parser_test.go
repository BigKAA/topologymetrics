package dephealth

import (
	"testing"
)

func TestParseURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		want    []ParsedConnection
		wantErr bool
	}{
		// PostgreSQL
		{
			name: "postgres with port",
			url:  "postgres://app:pass@pg.svc:5432/orders",
			want: []ParsedConnection{{Host: "pg.svc", Port: "5432", ConnType: TypePostgres}},
		},
		{
			name: "postgres without port",
			url:  "postgres://app:pass@pg.svc/orders",
			want: []ParsedConnection{{Host: "pg.svc", Port: "5432", ConnType: TypePostgres}},
		},
		{
			name: "postgresql scheme",
			url:  "postgresql://app@pg.svc:5432/db",
			want: []ParsedConnection{{Host: "pg.svc", Port: "5432", ConnType: TypePostgres}},
		},

		// Redis
		{
			name: "redis with port",
			url:  "redis://redis.svc:6379/0",
			want: []ParsedConnection{{Host: "redis.svc", Port: "6379", ConnType: TypeRedis}},
		},
		{
			name: "redis without port",
			url:  "redis://redis.svc",
			want: []ParsedConnection{{Host: "redis.svc", Port: "6379", ConnType: TypeRedis}},
		},
		{
			name: "rediss (TLS)",
			url:  "rediss://redis.svc:6380/0",
			want: []ParsedConnection{{Host: "redis.svc", Port: "6380", ConnType: TypeRedis}},
		},

		// HTTP
		{
			name: "http with port",
			url:  "http://payment.svc:8080/health",
			want: []ParsedConnection{{Host: "payment.svc", Port: "8080", ConnType: TypeHTTP}},
		},
		{
			name: "https without port",
			url:  "https://payment.svc/health",
			want: []ParsedConnection{{Host: "payment.svc", Port: "443", ConnType: TypeHTTP}},
		},

		// gRPC
		{
			name: "grpc",
			url:  "grpc://auth.svc:9090",
			want: []ParsedConnection{{Host: "auth.svc", Port: "9090", ConnType: TypeGRPC}},
		},

		// AMQP
		{
			name: "amqp with vhost",
			url:  "amqp://user:pass@rabbit.svc:5672/orders",
			want: []ParsedConnection{{Host: "rabbit.svc", Port: "5672", ConnType: TypeAMQP}},
		},
		{
			name: "amqps (TLS)",
			url:  "amqps://user:pass@rabbit.svc/prod",
			want: []ParsedConnection{{Host: "rabbit.svc", Port: "5671", ConnType: TypeAMQP}},
		},

		// Kafka
		{
			name: "kafka single broker",
			url:  "kafka://broker-0.svc:9092",
			want: []ParsedConnection{{Host: "broker-0.svc", Port: "9092", ConnType: TypeKafka}},
		},
		{
			name: "kafka multi-broker",
			url:  "kafka://broker-0:9092,broker-1:9092,broker-2:9092",
			want: []ParsedConnection{
				{Host: "broker-0", Port: "9092", ConnType: TypeKafka},
				{Host: "broker-1", Port: "9092", ConnType: TypeKafka},
				{Host: "broker-2", Port: "9092", ConnType: TypeKafka},
			},
		},

		// MySQL
		{
			name: "mysql",
			url:  "mysql://app:pass@mysql.svc:3306/orders",
			want: []ParsedConnection{{Host: "mysql.svc", Port: "3306", ConnType: TypeMySQL}},
		},

		// IPv6
		{
			name: "postgres IPv6",
			url:  "postgres://[::1]:5432/db",
			want: []ParsedConnection{{Host: "::1", Port: "5432", ConnType: TypePostgres}},
		},
		{
			name: "postgres IPv6 full",
			url:  "postgres://[2001:db8::1]:5432/db",
			want: []ParsedConnection{{Host: "2001:db8::1", Port: "5432", ConnType: TypePostgres}},
		},

		// Edge cases
		{
			name: "kafka multi-broker without port",
			url:  "kafka://broker-0,broker-1,broker-2",
			want: []ParsedConnection{
				{Host: "broker-0", Port: "9092", ConnType: TypeKafka},
				{Host: "broker-1", Port: "9092", ConnType: TypeKafka},
				{Host: "broker-2", Port: "9092", ConnType: TypeKafka},
			},
		},
		{
			name: "http default port 80",
			url:  "http://payment.svc/health",
			want: []ParsedConnection{{Host: "payment.svc", Port: "80", ConnType: TypeHTTP}},
		},
		{
			name: "grpc default port",
			url:  "grpc://auth.svc",
			want: []ParsedConnection{{Host: "auth.svc", Port: "443", ConnType: TypeGRPC}},
		},

		// Errors
		{name: "empty URL", url: "", wantErr: true},
		{name: "no scheme", url: "pg.svc:5432/db", wantErr: true},
		{name: "unsupported scheme", url: "ftp://host:21", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseURL(%q) error = %v, wantErr %v", tt.url, err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}

			if len(got) != len(tt.want) {
				t.Fatalf("ParseURL(%q) returned %d results, want %d", tt.url, len(got), len(tt.want))
			}

			for i, w := range tt.want {
				g := got[i]
				if g.Host != w.Host || g.Port != w.Port || g.ConnType != w.ConnType {
					t.Errorf("ParseURL(%q)[%d] = {%q, %q, %q}, want {%q, %q, %q}",
						tt.url, i, g.Host, g.Port, g.ConnType, w.Host, w.Port, w.ConnType)
				}
			}
		})
	}
}

func TestParseConnectionString(t *testing.T) {
	tests := []struct {
		name     string
		connStr  string
		wantHost string
		wantPort string
		wantErr  bool
	}{
		{
			name:     "Host and Port",
			connStr:  "Host=pg.svc;Port=5432;Database=orders",
			wantHost: "pg.svc",
			wantPort: "5432",
		},
		{
			name:     "Server with comma port (SQL Server)",
			connStr:  "Server=pg.svc,5432;Database=orders",
			wantHost: "pg.svc",
			wantPort: "5432",
		},
		{
			name:     "Host with colon port",
			connStr:  "Host=pg.svc:5432;Database=orders",
			wantHost: "pg.svc",
			wantPort: "5432",
		},
		{
			name:     "Data Source key",
			connStr:  "Data Source=pg.svc;Port=5432",
			wantHost: "pg.svc",
			wantPort: "5432",
		},
		{
			name:     "Address key",
			connStr:  "Address=pg.svc;Port=5432",
			wantHost: "pg.svc",
			wantPort: "5432",
		},
		{
			name:     "case insensitive keys",
			connStr:  "HOST=pg.svc;PORT=5432",
			wantHost: "pg.svc",
			wantPort: "5432",
		},
		{
			name:     "host only, no port",
			connStr:  "Host=pg.svc;Database=orders",
			wantHost: "pg.svc",
			wantPort: "",
		},
		{
			name:     "IPv6 in brackets",
			connStr:  "Host=[::1];Port=5432",
			wantHost: "::1",
			wantPort: "5432",
		},

		// Errors
		{name: "empty", connStr: "", wantErr: true},
		{name: "no host key", connStr: "Database=orders;Port=5432", wantErr: true},
		{name: "invalid port", connStr: "Host=pg.svc;Port=abc", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host, port, err := ParseConnectionString(tt.connStr)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseConnectionString(%q) error = %v, wantErr %v", tt.connStr, err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if host != tt.wantHost || port != tt.wantPort {
				t.Errorf("ParseConnectionString(%q) = (%q, %q), want (%q, %q)",
					tt.connStr, host, port, tt.wantHost, tt.wantPort)
			}
		})
	}
}

func TestParseJDBC(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		want    ParsedConnection
		wantErr bool
	}{
		{
			name: "postgresql with port",
			url:  "jdbc:postgresql://pg.svc:5432/orders",
			want: ParsedConnection{Host: "pg.svc", Port: "5432", ConnType: TypePostgres},
		},
		{
			name: "postgresql without port",
			url:  "jdbc:postgresql://pg.svc/orders",
			want: ParsedConnection{Host: "pg.svc", Port: "5432", ConnType: TypePostgres},
		},
		{
			name: "mysql",
			url:  "jdbc:mysql://mysql.svc:3306/orders",
			want: ParsedConnection{Host: "mysql.svc", Port: "3306", ConnType: TypeMySQL},
		},

		// Errors
		{name: "empty", url: "", wantErr: true},
		{name: "no jdbc prefix", url: "postgresql://pg.svc:5432/db", wantErr: true},
		{name: "unsupported subprotocol", url: "jdbc:oracle://host:1521/db", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseJDBC(tt.url)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseJDBC(%q) error = %v, wantErr %v", tt.url, err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if len(got) != 1 {
				t.Fatalf("ParseJDBC(%q) returned %d results, want 1", tt.url, len(got))
			}
			g := got[0]
			if g.Host != tt.want.Host || g.Port != tt.want.Port || g.ConnType != tt.want.ConnType {
				t.Errorf("ParseJDBC(%q) = {%q, %q, %q}, want {%q, %q, %q}",
					tt.url, g.Host, g.Port, g.ConnType, tt.want.Host, tt.want.Port, tt.want.ConnType)
			}
		})
	}
}

func TestParseParams(t *testing.T) {
	tests := []struct {
		name     string
		host     string
		port     string
		wantHost string
		wantPort string
		wantErr  bool
	}{
		{"valid", "pg.svc", "5432", "pg.svc", "5432", false},
		{"ipv6", "::1", "5432", "::1", "5432", false},
		{"ipv6 brackets", "[::1]", "5432", "::1", "5432", false},

		{"empty host", "", "5432", "", "", true},
		{"empty port", "pg.svc", "", "", "", true},
		{"invalid port", "pg.svc", "abc", "", "", true},
		{"port zero", "pg.svc", "0", "", "", true},
		{"port too high", "pg.svc", "70000", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ep, err := ParseParams(tt.host, tt.port)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseParams(%q, %q) error = %v, wantErr %v", tt.host, tt.port, err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if ep.Host != tt.wantHost || ep.Port != tt.wantPort {
				t.Errorf("ParseParams(%q, %q) = {%q, %q}, want {%q, %q}",
					tt.host, tt.port, ep.Host, ep.Port, tt.wantHost, tt.wantPort)
			}
		})
	}
}
