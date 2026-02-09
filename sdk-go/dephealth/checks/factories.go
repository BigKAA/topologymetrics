package checks

import (
	"net/url"
	"strconv"
	"strings"

	"github.com/BigKAA/topologymetrics/dephealth"
)

func init() {
	dephealth.RegisterCheckerFactory(dephealth.TypeHTTP, newHTTPFromConfig)
	dephealth.RegisterCheckerFactory(dephealth.TypeGRPC, newGRPCFromConfig)
	dephealth.RegisterCheckerFactory(dephealth.TypeTCP, newTCPFromConfig)
	dephealth.RegisterCheckerFactory(dephealth.TypePostgres, newPostgresFromConfig)
	dephealth.RegisterCheckerFactory(dephealth.TypeMySQL, newMySQLFromConfig)
	dephealth.RegisterCheckerFactory(dephealth.TypeRedis, newRedisFromConfig)
	dephealth.RegisterCheckerFactory(dephealth.TypeAMQP, newAMQPFromConfig)
	dephealth.RegisterCheckerFactory(dephealth.TypeKafka, newKafkaFromConfig)
}

func newHTTPFromConfig(dc *dephealth.DependencyConfig) dephealth.HealthChecker {
	var opts []HTTPOption
	if dc.HTTPHealthPath != "" {
		opts = append(opts, WithHealthPath(dc.HTTPHealthPath))
	}
	if dc.HTTPTLS != nil {
		opts = append(opts, WithTLSEnabled(*dc.HTTPTLS))
	}
	if dc.HTTPTLSSkipVerify != nil {
		opts = append(opts, WithHTTPTLSSkipVerify(*dc.HTTPTLSSkipVerify))
	}
	return NewHTTPChecker(opts...)
}

func newGRPCFromConfig(dc *dephealth.DependencyConfig) dephealth.HealthChecker {
	var opts []GRPCOption
	if dc.GRPCServiceName != "" {
		opts = append(opts, WithServiceName(dc.GRPCServiceName))
	}
	if dc.GRPCTLS != nil {
		opts = append(opts, WithGRPCTLS(*dc.GRPCTLS))
	}
	if dc.GRPCTLSSkipVerify != nil {
		opts = append(opts, WithGRPCTLSSkipVerify(*dc.GRPCTLSSkipVerify))
	}
	return NewGRPCChecker(opts...)
}

func newTCPFromConfig(_ *dephealth.DependencyConfig) dephealth.HealthChecker {
	return NewTCPChecker()
}

func newPostgresFromConfig(dc *dephealth.DependencyConfig) dephealth.HealthChecker {
	var opts []PostgresOption
	if dc.URL != "" {
		opts = append(opts, WithPostgresDSN(dc.URL))
	}
	if dc.PostgresQuery != "" {
		opts = append(opts, WithPostgresQuery(dc.PostgresQuery))
	}
	return NewPostgresChecker(opts...)
}

func newMySQLFromConfig(dc *dephealth.DependencyConfig) dephealth.HealthChecker {
	var opts []MySQLOption
	if dc.URL != "" {
		if dsn := mysqlURLToDSN(dc.URL); dsn != "" {
			opts = append(opts, WithMySQLDSN(dsn))
		}
	}
	if dc.MySQLQuery != "" {
		opts = append(opts, WithMySQLQuery(dc.MySQLQuery))
	}
	return NewMySQLChecker(opts...)
}

func newRedisFromConfig(dc *dephealth.DependencyConfig) dephealth.HealthChecker {
	var opts []RedisOption
	if dc.RedisPassword != "" {
		opts = append(opts, WithRedisPassword(dc.RedisPassword))
	}
	if dc.RedisDB != nil {
		opts = append(opts, WithRedisDB(*dc.RedisDB))
	}
	// Извлечь password и db из URL, если явные опции не заданы.
	if dc.URL != "" {
		u, err := url.Parse(dc.URL)
		if err == nil && u != nil && u.User != nil && dc.RedisPassword == "" {
			if p, ok := u.User.Password(); ok {
				opts = append(opts, WithRedisPassword(p))
			}
		}
		if err == nil && u != nil && dc.RedisDB == nil {
			dbStr := strings.TrimPrefix(u.Path, "/")
			if db, parseErr := strconv.Atoi(dbStr); parseErr == nil {
				opts = append(opts, WithRedisDB(db))
			}
		}
	}
	return NewRedisChecker(opts...)
}

func newAMQPFromConfig(dc *dephealth.DependencyConfig) dephealth.HealthChecker {
	var opts []AMQPOption
	if dc.AMQPURL != "" {
		opts = append(opts, WithAMQPURL(dc.AMQPURL))
	} else if dc.URL != "" {
		opts = append(opts, WithAMQPURL(dc.URL))
	}
	return NewAMQPChecker(opts...)
}

func newKafkaFromConfig(_ *dephealth.DependencyConfig) dephealth.HealthChecker {
	return NewKafkaChecker()
}

// mysqlURLToDSN converts a mysql:// URL to the go-sql-driver/mysql DSN format.
// mysql://user:pass@host:3306/db → user:pass@tcp(host:3306)/db
// Returns empty string if the URL cannot be parsed.
func mysqlURLToDSN(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}

	var userinfo string
	if u.User != nil {
		userinfo = u.User.String()
	}

	host := u.Host // includes port if specified

	path := u.Path // e.g. "/db"

	dsn := userinfo + "@tcp(" + host + ")" + path
	if q := u.RawQuery; q != "" {
		dsn += "?" + q
	}
	return dsn
}
