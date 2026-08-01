#!/bin/bash
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
BUILD_DIR="$(mktemp -d)"
trap 'rm -rf "$BUILD_DIR"' EXIT
if [[ -n "$(gofmt -l .)" ]]; then
  gofmt -l . >&2
  exit 1
fi
go test ./...
go vet ./...
go build -trimpath -o "$BUILD_DIR/release-gate" ./cmd/release-gate
echo "private-to-public release gate validation passed"
