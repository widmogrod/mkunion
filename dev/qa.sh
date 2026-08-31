#!/usr/bin/env bash
# Full QA for the project. Run it before you finish any piece of work:
#
#   dev/qa.sh          # codegen + short tests + CRAP gate
#   QA_FULL=1 dev/qa.sh # same, but the full test suite (needs dev/bootstrap.sh services)
#
# The CRAP gate (go tool mkcrap) fails when any function is both complex
# and untested. No exceptions.
set -euo pipefail
cd "$(dirname "$0")/.."

echo "==> generating code (mkunion)"
go tool mkunion watch -g ./...

if [ "${QA_FULL:-0}" = "1" ]; then
    echo "==> go test (full) with coverage"
    go test -coverprofile=coverage.out ./...
else
    echo "==> go test (short) with coverage"
    go test -short -coverprofile=coverage.out ./...
fi

echo "==> CRAP gate (complexity vs coverage)"
go tool mkcrap

echo "==> QA OK"
