// Package redispool provides dephealth integration with *redis.Client.
// It automatically extracts host:port from the client for metric labels.
package redispool

import (
	"net"

	"github.com/redis/go-redis/v9"

	"github.com/BigKAA/topologymetrics/sdk-go/dephealth"
	"github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/redischeck"
)

// FromClient creates an Option for monitoring Redis via an existing *redis.Client.
// Host and port are automatically extracted from client.Options().Addr.
// Additional DependencyOption values (Critical, CheckInterval, etc.) can be provided.
func FromClient(name string, client *redis.Client, opts ...dephealth.DependencyOption) dephealth.Option {
	checker := redischeck.New(redischeck.WithClient(client))

	// Extract host:port from the client.
	addr := client.Options().Addr
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		// If parsing fails, use addr as host.
		host = addr
		port = "6379"
	}

	// Prepend FromParams to options (user can override).
	allOpts := make([]dephealth.DependencyOption, 0, len(opts)+1)
	allOpts = append(allOpts, dephealth.FromParams(host, port))
	allOpts = append(allOpts, opts...)

	return dephealth.AddDependency(name, dephealth.TypeRedis, checker, allOpts...)
}
