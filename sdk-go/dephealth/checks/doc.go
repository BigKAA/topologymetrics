// Package checks registers all built-in health checkers.
//
// Importing this package registers factories for all supported dependency types.
// For selective imports (to reduce binary size), import individual sub-packages:
//
//	import _ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/httpcheck"
//	import _ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/pgcheck"
//
// This package also provides backward-compatible type aliases and constructor
// wrappers so that existing code using checks.HTTPChecker, checks.NewHTTPChecker, etc.
// continues to compile without changes.
package checks

import (
	_ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/amqpcheck"
	_ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/grpccheck"
	_ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/httpcheck"
	_ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/kafkacheck"
	_ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/ldapcheck"
	_ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/mysqlcheck"
	_ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/pgcheck"
	_ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/redischeck"
	_ "github.com/BigKAA/topologymetrics/sdk-go/dephealth/checks/tcpcheck"
)
