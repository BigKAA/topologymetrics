# Documentation Audit Fix Plan

## Summary

Fix all documentation inconsistencies found during the audit:
LDAP missing from ~20 locations, HELP string mismatches, missing `group` label,
conformance test count contradictions, outdated labels in sdk-architecture.md,
non-existent Django reference, missing LDAP section in Go SDK checkers.md.

## Phase 1: Add LDAP to spec and top-level docs

Files:
- `spec/metric-contract.md` + `spec/metric-contract.ru.md` — add `ldap` to `type` label, add LDAP row to detail values table
- `README.md` + `README.ru.md` — add LDAP to architecture diagram and "Supported Check Types" table
- `CLAUDE.md` — add `ldap` to type label list and Health Checkers description

## Phase 2: Add LDAP to docs/ directory files

Files:
- `docs/specification.md` + `docs/specification.ru.md` — add `ldap` to type label, check types table, auto-detection, default ports
- `docs/comparison.md` + `docs/comparison.ru.md` — add LDAP row to checkers table, fix `group` label, fix HELP strings, fix conformance count

## Phase 3: Fix sdk-architecture.md

- Add `ldap` to type label values
- Add LDAP row to check types table
- Add LDAP to architecture diagram (also add missing MySQL)
- Add LdapChecker to HealthChecker implementations list
- Add ldap checker files to all 4 SDK internal structures
- Fix required labels (add `name`, `group`, `critical`)
- Mark Django integration as planned/future (or remove)

## Phase 4: Fix Go SDK checkers.md

- Change "8 built-in" to "9 built-in"
- Add full LDAP section (matching Java/Python/C# pattern)

## Phase 5: Fix conformance test counts

- Verify actual number in conformance/ directory
- Update docs/comparison.md and docs/specification.md to match README (14 scenarios)

---

## Status

- [x] Phase 1
- [x] Phase 2
- [x] Phase 3
- [x] Phase 4
- [x] Phase 5
