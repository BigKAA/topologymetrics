# Plan: Go SDK — Split checkers into sub-packages (v0.6.0)

## Goal

Allow users to import only the checkers they need instead of the entire `checks` package.
Reduce binary size and build time for services that use a subset of checkers.

## Current state

- All 8 checkers live in a single `checks` package
- `init()` in `factories.go` registers all 8 factories at once
- `Version` const in `checks/doc.go`, used by `http.go` for User-Agent
- `contrib/redispool` imports `checks` for `NewRedisChecker` + `WithRedisClient`
- `contrib/sqldb` imports `checks` for `NewPostgresChecker`, `NewMySQLChecker`
- External dependencies per checker:
  - TCP, HTTP — stdlib only
  - gRPC — `google.golang.org/grpc`
  - Postgres — `github.com/jackc/pgx/v5`
  - MySQL — `github.com/go-sql-driver/mysql`
  - Redis — `github.com/redis/go-redis/v9`
  - AMQP — `github.com/rabbitmq/amqp091-go`
  - Kafka — `github.com/segmentio/kafka-go`

## Target structure

```
dephealth/
  version.go                    # Version const (moved from checks/doc.go)
  checks/
    doc.go                      # Package doc + blank imports of all sub-packages
    compat.go                   # Type aliases + var wrappers for backward compat
    httpcheck/
      httpcheck.go              # Checker, New(), Options, init()
      httpcheck_test.go
    grpccheck/
      grpccheck.go
      grpccheck_test.go
    tcpcheck/
      tcpcheck.go
      tcpcheck_test.go
    pgcheck/
      pgcheck.go
      pgcheck_test.go
    mysqlcheck/
      mysqlcheck.go
      mysqlcheck_test.go
    redischeck/
      redischeck.go
      redischeck_test.go
    amqpcheck/
      amqpcheck.go
      amqpcheck_test.go
    kafkacheck/
      kafkacheck.go
      kafkacheck_test.go
  contrib/
    redispool/                  # imports checks/redischeck
    sqldb/                      # imports checks/pgcheck + checks/mysqlcheck
```

## Naming conventions

- Package: `httpcheck`, `grpccheck`, `tcpcheck`, `pgcheck`, `mysqlcheck`,
  `redischeck`, `amqpcheck`, `kafkacheck`
- Type: `httpcheck.Checker` (not `HTTPChecker` — package name provides context)
- Constructor: `httpcheck.New()`
- Options: `httpcheck.Option`, `httpcheck.WithHealthPath()`
- Factory: `httpcheck.NewFromConfig()` (exported, was `newHTTPFromConfig`)

## Phases

---

### Phase 1: Prepare core — move Version, create sub-package scaffolding

**Files to modify:**

- [x] Create `dephealth/version.go` — move `Version` const from `checks/doc.go`
- [x] Update `checks/doc.go` — remove `Version` const, keep package doc
- [x] Update `checks/http.go` — `Version` → `dephealth.Version`
- [x] Update `checks/http_test.go` — `Version` → `dephealth.Version`
- [x] Create 8 empty sub-package dirs with placeholder files

**Files created:**

- [x] `dephealth/version.go`
- [x] `dephealth/checks/tcpcheck/tcpcheck.go` (scaffold)
- [x] `dephealth/checks/httpcheck/httpcheck.go` (scaffold)
- [x] `dephealth/checks/grpccheck/grpccheck.go` (scaffold)
- [x] `dephealth/checks/pgcheck/pgcheck.go` (scaffold)
- [x] `dephealth/checks/mysqlcheck/mysqlcheck.go` (scaffold)
- [x] `dephealth/checks/redischeck/redischeck.go` (scaffold)
- [x] `dephealth/checks/amqpcheck/amqpcheck.go` (scaffold)
- [x] `dephealth/checks/kafkacheck/kafkacheck.go` (scaffold)

**Validation:**

- [x] `make build` passes
- [x] `make test` passes
- [x] `make lint` passes

**Status:** done

---

### Phase 2: Migrate TCP and HTTP checkers (no external deps)

Move the two simplest checkers first — they have no external dependencies.

**TCP checker → `checks/tcpcheck/`:**

- [x] Move `TCPChecker`, `NewTCPChecker` from `checks/tcp.go`
- [x] Rename to `Checker`, `New`
- [x] Add `init()` → `dephealth.RegisterCheckerFactory(dephealth.TypeTCP, NewFromConfig)`
- [x] Create `NewFromConfig` (was `newTCPFromConfig` in `factories.go`)
- [x] Move tests from `checks/tcp_test.go` → `checks/tcpcheck/tcpcheck_test.go`

**HTTP checker → `checks/httpcheck/`:**

- [x] Move `HTTPChecker`, `HTTPOption`, `NewHTTPChecker` + all `With*` options
- [x] Rename to `Checker`, `Option`, `New`
- [x] Replace `Version` → `dephealth.Version`
- [x] Add `init()` → `dephealth.RegisterCheckerFactory(dephealth.TypeHTTP, NewFromConfig)`
- [x] Create `NewFromConfig` (was `newHTTPFromConfig` in `factories.go`)
- [x] Move tests from `checks/http_test.go` → `checks/httpcheck/httpcheck_test.go`

**Update `checks/factories.go`:**

- [x] Remove `newTCPFromConfig`, `newHTTPFromConfig` and their `init()` registrations
- [x] Remove TCP/HTTP imports if any become unused

**Delete:**

- [x] `checks/tcp.go`, `checks/http.go`
- [x] `checks/tcp_test.go`, `checks/http_test.go`

**Validation:**

- [x] `make build && make test && make lint`

**Status:** done

---

### Phase 3: Migrate gRPC checker

**gRPC checker → `checks/grpccheck/`:**

- [x] Move `GRPCChecker`, `GRPCOption`, `NewGRPCChecker` + all `With*` options
- [x] Rename to `Checker`, `Option`, `New`
- [x] Add `init()` → `dephealth.RegisterCheckerFactory(dephealth.TypeGRPC, NewFromConfig)`
- [x] Create `NewFromConfig` (was `newGRPCFromConfig`)
- [x] Move tests from `checks/grpc_test.go` → `checks/grpccheck/grpccheck_test.go`

**Update `checks/factories.go`:**

- [x] Remove `newGRPCFromConfig` and its `init()` registration

**Delete:**

- [x] `checks/grpc.go`, `checks/grpc_test.go`

**Validation:**

- [x] `make build && make test && make lint`

**Status:** done

---

### Phase 4: Migrate database checkers (Postgres, MySQL)

**Postgres checker → `checks/pgcheck/`:**

- [x] Move `PostgresChecker`, `PostgresOption`, `NewPostgresChecker` + all `With*`
- [x] Rename to `Checker`, `Option`, `New`
- [x] Add `init()` + `NewFromConfig` (was `newPostgresFromConfig`)
- [x] Move tests

**MySQL checker → `checks/mysqlcheck/`:**

- [x] Move `MySQLChecker`, `MySQLOption`, `NewMySQLChecker` + all `With*`
- [x] Rename to `Checker`, `Option`, `New`
- [x] Add `init()` + `NewFromConfig` (was `newMySQLFromConfig`)
- [x] Move `mysqlURLToDSN` helper into `mysqlcheck` package (as `URLToDSN`)
- [x] Move tests (including `TestMySQLURLToDSN` from `factories_test.go`)

**Update `contrib/sqldb/sqldb.go`:**

- [x] `import "checks"` → `import "checks/pgcheck"` + `import "checks/mysqlcheck"`
- [x] `checks.NewPostgresChecker(checks.WithPostgresDB(db))` → `pgcheck.New(pgcheck.WithDB(db))`
- [x] `checks.NewMySQLChecker(checks.WithMySQLDB(db))` → `mysqlcheck.New(mysqlcheck.WithDB(db))`

**Update `contrib/sqldb/sqldb_test.go`:**

- [x] Blank import `_ "checks"` → `_ "checks/pgcheck"` + `_ "checks/mysqlcheck"`

**Delete:**

- [x] `checks/postgres.go`, `checks/mysql.go`
- [x] `checks/postgres_test.go`, `checks/mysql_test.go`
- [x] Factory-related tests from `checks/factories_test.go` for Postgres/MySQL

**Validation:**

- [x] `make build && make test && make lint`

**Status:** done

---

### Phase 5: Migrate Redis, AMQP, Kafka checkers

**Redis checker → `checks/redischeck/`:**

- [x] Move `RedisChecker`, `RedisOption`, `NewRedisChecker` + all `With*`
- [x] Rename to `Checker`, `Option`, `New`
- [x] Add `init()` + `NewFromConfig` (was `newRedisFromConfig`)
- [x] Move factory logic (password/DB from URL parsing) from `factories.go`
- [x] Move tests (including Redis factory tests from `factories_test.go`)

**Update `contrib/redispool/redispool.go`:**

- [x] `import "checks"` → `import "checks/redischeck"`
- [x] `checks.NewRedisChecker(checks.WithRedisClient(...))` →
  `redischeck.New(redischeck.WithClient(...))`

**Update `contrib/redispool/redispool_test.go`:**

- [x] Blank import `_ "checks"` → `_ "checks/redischeck"`

**AMQP checker → `checks/amqpcheck/`:**

- [x] Move `AMQPChecker`, `AMQPOption`, `NewAMQPChecker` + all `With*`
- [x] Rename to `Checker`, `Option`, `New`
- [x] Add `init()` + `NewFromConfig` (was `newAMQPFromConfig`)
- [x] Move factory logic (AMQP URL fallback) from `factories.go`
- [x] Move tests (including AMQP factory tests from `factories_test.go`)

**Kafka checker → `checks/kafkacheck/`:**

- [x] Move `KafkaChecker`, `NewKafkaChecker`
- [x] Rename to `Checker`, `New`
- [x] Add `init()` + `NewFromConfig` (was `newKafkaFromConfig`)
- [x] Move tests

**Delete:**

- [x] `checks/redis.go`, `checks/amqp.go`, `checks/kafka.go`
- [x] `checks/redis_test.go`, `checks/amqp_test.go`, `checks/kafka_test.go`
- [x] `checks/factories.go` (all factory functions migrated)
- [x] `checks/factories_test.go` (all factory tests migrated)

**Validation:**

- [x] `make build && make test && make lint`

**Status:** done

---

### Phase 6: Backward compatibility layer + cleanup

**Create `checks/doc.go` (new content):**

- [x] Blank imports of all 8 sub-packages

**Create `checks/compat.go`:**

- [x] Type aliases: `type HTTPChecker = httpcheck.Checker`, etc.
- [x] Option aliases: `type HTTPOption = httpcheck.Option`, etc.
- [x] Constructor wrappers: `var NewHTTPChecker = httpcheck.New`, etc.
- [x] Option function aliases: `var WithHealthPath = httpcheck.WithHealthPath`, etc.
- [x] Full list of all exported names from old `checks` package
- [x] All aliases marked `Deprecated` with godoc links to new packages

**Update error message in `dephealth/options.go:329`:**

- [x] Update import hint to mention both `checks` and individual sub-packages

**Update godoc in `dephealth/dephealth.go:27`:**

- [x] Update import example to show both options

**Validation:**

- [x] `make build && make test && make lint`
- [x] Verify backward-compat: all existing test code compiles without changes

**Status:** done

---

### Phase 7: Update conformance test service

**Check and update:**

- [x] `conformance/` test service Go code — only blank import `_ "...checks"`, no changes needed
- [x] Conformance test-service compiles with updated SDK (`go build` OK)

**Validation:**

- [x] Conformance test-service builds successfully
- [x] `make build && make test && make lint` in sdk-go/ — all pass

**Status:** done

---

### Phase 8: Documentation and version bump

**Version bump to v0.6.0:**

- [x] `dephealth/version.go` → `Version = "0.6.0"`
- [x] Only one version reference found in sdk-go/ (version.go)

**Documentation:**

- [x] Update `sdk-go/README.md` — added "Selective Imports" section
- [x] Update `docs/quickstart/go.md` — show both import styles in minimal example
- [x] Update `docs/quickstart/go.ru.md` — same
- [x] Create `docs/migration/v050-to-v060.md` — migration guide (EN)
- [x] Create `docs/migration/v050-to-v060.ru.md` — migration guide (RU)
- [x] Update `CHANGELOG.md` — added `[0.6.0]` section

**Spec update:**

- [x] `spec/specification.md` — language-neutral, no Go-specific import sections, no changes needed

**Validation:**

- [x] `make build && make test && make lint` — all pass
- [x] `markdownlint` on all new/changed .md files — 0 issues

**Status:** done

---

### Phase 9: Merge, tag, release

**Pre-merge checklist:**

- [x] All tests pass (unit + conformance)
- [x] All linters pass
- [x] Backward compat verified (old import style works)
- [x] New selective import verified
- [x] Docs complete (EN + RU)
- [x] CHANGELOG updated

**Actions:**

- Merge to master (or PR — ask user)
- Tag: `sdk-go/v0.6.0`
- GitHub Release: sdk-go/v0.6.0
- Update TODO.md — mark task as done

**Status:** done
