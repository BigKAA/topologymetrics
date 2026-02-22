package checks

import (
	"net/url"
	"strconv"
	"strings"

	"github.com/BigKAA/topologymetrics/sdk-go/dephealth"
)

func init() {
	dephealth.RegisterCheckerFactory(dephealth.TypeRedis, newRedisFromConfig)
	dephealth.RegisterCheckerFactory(dephealth.TypeAMQP, newAMQPFromConfig)
	dephealth.RegisterCheckerFactory(dephealth.TypeKafka, newKafkaFromConfig)
}

func newRedisFromConfig(dc *dephealth.DependencyConfig) dephealth.HealthChecker {
	var opts []RedisOption
	if dc.RedisPassword != "" {
		opts = append(opts, WithRedisPassword(dc.RedisPassword))
	}
	if dc.RedisDB != nil {
		opts = append(opts, WithRedisDB(*dc.RedisDB))
	}
	// Extract password and db from URL if explicit options are not set.
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
