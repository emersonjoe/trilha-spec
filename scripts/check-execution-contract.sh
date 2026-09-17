#!/bin/sh
set -eu

ROOT=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
FIXTURE="$ROOT/execution/testdata/run-v1.json"
SCHEMA="$ROOT/contracts/execution/v1/schema.json"

for repo in trilha-runner trilha-cloud; do
  target=$(CDPATH= cd -- "$ROOT/../$repo" 2>/dev/null && pwd || true)
  if [ -z "$target" ]; then
    printf 'skip %s: sibling checkout not found\n' "$repo"
    continue
  fi
  cmp "$FIXTURE" "$target/testdata/contracts/execution-v1.json"
  cmp "$SCHEMA" "$target/testdata/contracts/execution-v1.schema.json"
  printf '%s execution contract matches v1\n' "$repo"
done
