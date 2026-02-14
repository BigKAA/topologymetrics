package checks

import (
	"context"
	"fmt"
	"net"

	"github.com/segmentio/kafka-go"

	"github.com/BigKAA/topologymetrics/sdk-go/dephealth"
)

// KafkaChecker performs health checks against a Kafka broker.
// Connects to the broker, requests broker metadata, and closes the connection.
// Only standalone mode is supported.
type KafkaChecker struct{}

// NewKafkaChecker creates a new Kafka health checker.
func NewKafkaChecker() *KafkaChecker {
	return &KafkaChecker{}
}

// Check connects to the Kafka broker, requests metadata, and closes.
// Returns nil if the broker responds with metadata.
func (c *KafkaChecker) Check(ctx context.Context, endpoint dephealth.Endpoint) error {
	addr := net.JoinHostPort(endpoint.Host, endpoint.Port)

	dialer := &kafka.Dialer{}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("kafka dial %s: %w", addr, err)
	}
	defer func() { _ = conn.Close() }()

	brokers, err := conn.Brokers()
	if err != nil {
		return fmt.Errorf("kafka brokers %s: %w", addr, err)
	}
	if len(brokers) == 0 {
		return &dephealth.ClassifiedCheckError{
			Category: dephealth.StatusUnhealthy,
			Detail:   "no_brokers",
			Cause:    fmt.Errorf("kafka %s: no brokers in metadata response", addr),
		}
	}

	return nil
}

// Type returns the dependency type for this checker.
func (c *KafkaChecker) Type() string {
	return string(dephealth.TypeKafka)
}
