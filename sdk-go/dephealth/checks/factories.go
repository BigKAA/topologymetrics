package checks

import "github.com/company/dephealth/dephealth"

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
	if dc.PostgresQuery != "" {
		opts = append(opts, WithPostgresQuery(dc.PostgresQuery))
	}
	return NewPostgresChecker(opts...)
}

func newMySQLFromConfig(dc *dephealth.DependencyConfig) dephealth.HealthChecker {
	var opts []MySQLOption
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
	return NewRedisChecker(opts...)
}

func newAMQPFromConfig(dc *dephealth.DependencyConfig) dephealth.HealthChecker {
	var opts []AMQPOption
	if dc.AMQPURL != "" {
		opts = append(opts, WithAMQPURL(dc.AMQPURL))
	}
	return NewAMQPChecker(opts...)
}

func newKafkaFromConfig(_ *dephealth.DependencyConfig) dephealth.HealthChecker {
	return NewKafkaChecker()
}
