// Package redispool предоставляет интеграцию dephealth с *redis.Client.
// Автоматически извлекает host:port из клиента для меток метрик.
package redispool

import (
	"net"

	"github.com/redis/go-redis/v9"

	"github.com/BigKAA/topologymetrics/dephealth"
	"github.com/BigKAA/topologymetrics/dephealth/checks"
)

// FromClient создаёт Option для мониторинга Redis через существующий *redis.Client.
// Host и port извлекаются автоматически из client.Options().Addr.
// Дополнительные DependencyOption (Critical, CheckInterval и т.д.) можно передать.
func FromClient(name string, client *redis.Client, opts ...dephealth.DependencyOption) dephealth.Option {
	checker := checks.NewRedisChecker(checks.WithRedisClient(client))

	// Извлекаем host:port из клиента.
	addr := client.Options().Addr
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		// Если не удалось разобрать — используем addr как host.
		host = addr
		port = "6379"
	}

	// Добавляем FromParams в начало опций (пользователь может переопределить).
	allOpts := make([]dephealth.DependencyOption, 0, len(opts)+1)
	allOpts = append(allOpts, dephealth.FromParams(host, port))
	allOpts = append(allOpts, opts...)

	return dephealth.AddDependency(name, dephealth.TypeRedis, checker, allOpts...)
}
