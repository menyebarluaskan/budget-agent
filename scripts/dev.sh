#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")/.."

mkdir -p .local/data .local/receipts

export APP_ADDR="${APP_ADDR:-127.0.0.1:8080}"
export DATABASE_PATH="${DATABASE_PATH:-$PWD/.local/data/budget.db}"
export RECEIPTS_DIR="${RECEIPTS_DIR:-$PWD/.local/receipts}"
export BASE_URL="${BASE_URL:-http://127.0.0.1:8080}"
export AUTH_MODE="${AUTH_MODE:-dev}"
export TOKEN_PEPPER="${TOKEN_PEPPER:-local-development-only}"
export LOG_LEVEL="${LOG_LEVEL:-debug}"

pids=()

cleanup() {
	for pid in "${pids[@]}"; do
		kill "$pid" 2>/dev/null || true
	done
	wait 2>/dev/null || true
}

trap cleanup EXIT INT TERM

go run ./cmd/budget-agent &
pids+=("$!")

(cd web && pnpm dev --host 127.0.0.1) &
pids+=("$!")

wait -n "${pids[@]}"
