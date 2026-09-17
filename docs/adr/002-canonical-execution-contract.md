# ADR 002 — Canonical execution contract

- **Date**: 2026-09-14
- **Status**: accepted

## Context

Trilha Spec, Runner and Cloud independently introduced run IDs, attempts, progress phases, repair lineage and result metadata. Their Go modules are released independently, so sharing unreleased Go types through a relative `replace` would make normal builds depend on a sibling checkout.

## Decision

The canonical wire contract is `trilha.execution/v1` and lives in `contracts/execution/v1/schema.json`. The reference Go model lives in package `execution`; the canonical example lives in `execution/testdata/run-v1.json`.

Runner and Cloud keep local transport types so each repository builds independently. Each repository carries an exact copy of the schema and fixture, tests that its types decode the fixture without losing required behavior, and exposes a `make conformance` gate that compares its copies with Trilha Spec when the repositories are checked out as siblings.

A Task is work, a Run is one logical execution of that work, and an Attempt is one provider or corrective pass inside a Run. `retry_of` therefore contains a Run ID such as `run-000006`, never a Task ID.

## Consequences

- Cross-language implementations can validate the JSON contract without importing Go.
- Released modules remain reproducible and do not depend on local filesystem layout.
- Contract changes require a new versioned directory and fixture.
- Cross-repository development must run the conformance gate before release.
