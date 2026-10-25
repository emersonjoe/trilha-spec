#!/bin/sh
# Compares this repository's execution contract (adr/002) with the copies the
# sibling repositories carry. A sibling that is not checked out next to this
# one is skipped; a sibling that is present must match byte for byte. Exits 0
# only when every present sibling matches, and stops at the first difference.
set -eu

ROOT=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
FIXTURE="$ROOT/execution/testdata/run-v1.json"
NEEDS_INPUT="$ROOT/execution/testdata/run-v1-needs-input.json"
SCHEMA="$ROOT/contracts/execution/v1/schema.json"

compared=0
for repo in trilha-runner trilha-cloud; do
  target=$(CDPATH= cd -- "$ROOT/../$repo" 2>/dev/null && pwd || true)
  if [ -z "$target" ]; then
    printf 'skipped: %s not present (expected at %s)\n' "$repo" "$ROOT/../$repo"
    continue
  fi
  cmp "$FIXTURE" "$target/testdata/contracts/execution-v1.json"
  cmp "$NEEDS_INPUT" "$target/testdata/contracts/execution-v1-needs-input.json"
  cmp "$SCHEMA" "$target/testdata/contracts/execution-v1.schema.json"
  printf '%s execution contract matches v1\n' "$repo"
  compared=$((compared + 1))
done

if [ "$compared" -eq 0 ]; then
  printf 'skipped: no sibling checkout present; nothing compared\n'
fi
