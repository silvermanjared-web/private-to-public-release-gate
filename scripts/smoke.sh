#!/bin/bash
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
go test ./internal/gate -run TestRunPassesReviewedExportAndFailsPrivacyBeforeRelease -count=1
echo "release gate smoke test passed"
