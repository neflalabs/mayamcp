#!/bin/sh
# Cross-repo parity gate: ngodingmcp's tool contract must stay byte-identical
# to the recorded baseline while both modules build. The sibling repo is
# optional so a standalone clone of the template can still run its own checks.
set -eu

ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
NG=${NGODINGMCP_DIR:-$ROOT/../ngodingmcp}

if [ ! -f "$NG/go.mod" ]; then
  echo "parity-check: ngodingmcp not found at $NG (skipped)"
  exit 0
fi

GOLDEN_B="$NG/internal/mcpserver/testdata/tools.golden.json"
EXPECTED=$(cat "$ROOT/testdata/parity/ngodingmcp.golden.sha256")
ACTUAL=$(sha256sum "$GOLDEN_B" | awk '{print $1}')

if [ "$ACTUAL" != "$EXPECTED" ]; then
  echo "parity-check: ngodingmcp golden changed (want $EXPECTED, got $ACTUAL)" >&2
  exit 1
fi
if ! grep -q 'github.com/neflalabs/mayamcp v' "$NG/go.mod"; then
  echo "parity-check: ngodingmcp does not require the canonical mayamcp path" >&2
  exit 1
fi

(cd "$ROOT" && GOWORK=off go build ./...)
(cd "$NG" && GOWORK=off go build ./...)

echo "parity-check: ngodingmcp golden unchanged; both modules build"
