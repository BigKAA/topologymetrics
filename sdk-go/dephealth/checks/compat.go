package checks

// Backward-compatible type aliases, constructor wrappers, and option aliases.
// These allow existing code that uses checks.HTTPChecker, checks.NewHTTPChecker, etc.
// to compile without changes after the sub-package migration.
//
// Deprecated: use individual sub-packages (httpcheck, grpccheck, etc.) directly.

import (
	"github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/amqpcheck"
	"github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/grpccheck"
	"github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/httpcheck"
	"github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/kafkacheck"
	"github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/mysqlcheck"
	"github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/pgcheck"
	"github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/redischeck"
	"github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/tcpcheck"
)

// --- TCP ---

// TCPChecker is an alias for [tcpcheck.Checker].
//
// Deprecated: use [tcpcheck.Checker].
type TCPChecker = tcpcheck.Checker

// NewTCPChecker is a wrapper for [tcpcheck.New].
//
// Deprecated: use [tcpcheck.New].
var NewTCPChecker = tcpcheck.New

// --- HTTP ---

// HTTPChecker is an alias for [httpcheck.Checker].
//
// Deprecated: use [httpcheck.Checker].
type HTTPChecker = httpcheck.Checker

// HTTPOption is an alias for [httpcheck.Option].
//
// Deprecated: use [httpcheck.Option].
type HTTPOption = httpcheck.Option

// NewHTTPChecker is a wrapper for [httpcheck.New].
//
// Deprecated: use [httpcheck.New].
var NewHTTPChecker = httpcheck.New

// WithHealthPath is a wrapper for [httpcheck.WithHealthPath].
//
// Deprecated: use [httpcheck.WithHealthPath].
var WithHealthPath = httpcheck.WithHealthPath

// WithTLSEnabled is a wrapper for [httpcheck.WithTLSEnabled].
//
// Deprecated: use [httpcheck.WithTLSEnabled].
var WithTLSEnabled = httpcheck.WithTLSEnabled

// WithHTTPTLSSkipVerify is a wrapper for [httpcheck.WithTLSSkipVerify].
//
// Deprecated: use [httpcheck.WithTLSSkipVerify].
var WithHTTPTLSSkipVerify = httpcheck.WithTLSSkipVerify

// WithHeaders is a wrapper for [httpcheck.WithHeaders].
//
// Deprecated: use [httpcheck.WithHeaders].
var WithHeaders = httpcheck.WithHeaders

// WithBearerToken is a wrapper for [httpcheck.WithBearerToken].
//
// Deprecated: use [httpcheck.WithBearerToken].
var WithBearerToken = httpcheck.WithBearerToken

// WithBasicAuth is a wrapper for [httpcheck.WithBasicAuth].
//
// Deprecated: use [httpcheck.WithBasicAuth].
var WithBasicAuth = httpcheck.WithBasicAuth

// --- gRPC ---

// GRPCChecker is an alias for [grpccheck.Checker].
//
// Deprecated: use [grpccheck.Checker].
type GRPCChecker = grpccheck.Checker

// GRPCOption is an alias for [grpccheck.Option].
//
// Deprecated: use [grpccheck.Option].
type GRPCOption = grpccheck.Option

// NewGRPCChecker is a wrapper for [grpccheck.New].
//
// Deprecated: use [grpccheck.New].
var NewGRPCChecker = grpccheck.New

// WithServiceName is a wrapper for [grpccheck.WithServiceName].
//
// Deprecated: use [grpccheck.WithServiceName].
var WithServiceName = grpccheck.WithServiceName

// WithGRPCTLS is a wrapper for [grpccheck.WithTLS].
//
// Deprecated: use [grpccheck.WithTLS].
var WithGRPCTLS = grpccheck.WithTLS

// WithGRPCTLSSkipVerify is a wrapper for [grpccheck.WithTLSSkipVerify].
//
// Deprecated: use [grpccheck.WithTLSSkipVerify].
var WithGRPCTLSSkipVerify = grpccheck.WithTLSSkipVerify

// WithMetadata is a wrapper for [grpccheck.WithMetadata].
//
// Deprecated: use [grpccheck.WithMetadata].
var WithMetadata = grpccheck.WithMetadata

// WithGRPCBearerToken is a wrapper for [grpccheck.WithBearerToken].
//
// Deprecated: use [grpccheck.WithBearerToken].
var WithGRPCBearerToken = grpccheck.WithBearerToken

// WithGRPCBasicAuth is a wrapper for [grpccheck.WithBasicAuth].
//
// Deprecated: use [grpccheck.WithBasicAuth].
var WithGRPCBasicAuth = grpccheck.WithBasicAuth

// --- PostgreSQL ---

// PostgresChecker is an alias for [pgcheck.Checker].
//
// Deprecated: use [pgcheck.Checker].
type PostgresChecker = pgcheck.Checker

// PostgresOption is an alias for [pgcheck.Option].
//
// Deprecated: use [pgcheck.Option].
type PostgresOption = pgcheck.Option

// NewPostgresChecker is a wrapper for [pgcheck.New].
//
// Deprecated: use [pgcheck.New].
var NewPostgresChecker = pgcheck.New

// WithPostgresDB is a wrapper for [pgcheck.WithDB].
//
// Deprecated: use [pgcheck.WithDB].
var WithPostgresDB = pgcheck.WithDB

// WithPostgresDSN is a wrapper for [pgcheck.WithDSN].
//
// Deprecated: use [pgcheck.WithDSN].
var WithPostgresDSN = pgcheck.WithDSN

// WithPostgresQuery is a wrapper for [pgcheck.WithQuery].
//
// Deprecated: use [pgcheck.WithQuery].
var WithPostgresQuery = pgcheck.WithQuery

// --- MySQL ---

// MySQLChecker is an alias for [mysqlcheck.Checker].
//
// Deprecated: use [mysqlcheck.Checker].
type MySQLChecker = mysqlcheck.Checker

// MySQLOption is an alias for [mysqlcheck.Option].
//
// Deprecated: use [mysqlcheck.Option].
type MySQLOption = mysqlcheck.Option

// NewMySQLChecker is a wrapper for [mysqlcheck.New].
//
// Deprecated: use [mysqlcheck.New].
var NewMySQLChecker = mysqlcheck.New

// WithMySQLDB is a wrapper for [mysqlcheck.WithDB].
//
// Deprecated: use [mysqlcheck.WithDB].
var WithMySQLDB = mysqlcheck.WithDB

// WithMySQLDSN is a wrapper for [mysqlcheck.WithDSN].
//
// Deprecated: use [mysqlcheck.WithDSN].
var WithMySQLDSN = mysqlcheck.WithDSN

// WithMySQLQuery is a wrapper for [mysqlcheck.WithQuery].
//
// Deprecated: use [mysqlcheck.WithQuery].
var WithMySQLQuery = mysqlcheck.WithQuery

// --- Redis ---

// RedisChecker is an alias for [redischeck.Checker].
//
// Deprecated: use [redischeck.Checker].
type RedisChecker = redischeck.Checker

// RedisOption is an alias for [redischeck.Option].
//
// Deprecated: use [redischeck.Option].
type RedisOption = redischeck.Option

// NewRedisChecker is a wrapper for [redischeck.New].
//
// Deprecated: use [redischeck.New].
var NewRedisChecker = redischeck.New

// WithRedisClient is a wrapper for [redischeck.WithClient].
//
// Deprecated: use [redischeck.WithClient].
var WithRedisClient = redischeck.WithClient

// WithRedisPassword is a wrapper for [redischeck.WithPassword].
//
// Deprecated: use [redischeck.WithPassword].
var WithRedisPassword = redischeck.WithPassword

// WithRedisDB is a wrapper for [redischeck.WithDB].
//
// Deprecated: use [redischeck.WithDB].
var WithRedisDB = redischeck.WithDB

// --- AMQP ---

// AMQPChecker is an alias for [amqpcheck.Checker].
//
// Deprecated: use [amqpcheck.Checker].
type AMQPChecker = amqpcheck.Checker

// AMQPOption is an alias for [amqpcheck.Option].
//
// Deprecated: use [amqpcheck.Option].
type AMQPOption = amqpcheck.Option

// NewAMQPChecker is a wrapper for [amqpcheck.New].
//
// Deprecated: use [amqpcheck.New].
var NewAMQPChecker = amqpcheck.New

// WithAMQPURL is a wrapper for [amqpcheck.WithURL].
//
// Deprecated: use [amqpcheck.WithURL].
var WithAMQPURL = amqpcheck.WithURL

// --- Kafka ---

// KafkaChecker is an alias for [kafkacheck.Checker].
//
// Deprecated: use [kafkacheck.Checker].
type KafkaChecker = kafkacheck.Checker

// NewKafkaChecker is a wrapper for [kafkacheck.New].
//
// Deprecated: use [kafkacheck.New].
var NewKafkaChecker = kafkacheck.New
