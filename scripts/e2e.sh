#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
TARGET_PORT="${TARGET_PORT:-18081}"
PUBLIC_PORT="${PUBLIC_PORT:-18080}"
CLIENT_PORT="${CLIENT_PORT:-19090}"
PAIR_TIMEOUT="${PAIR_TIMEOUT:-10s}"

SERVER_LOG="${ROOT_DIR}/.server.log"
CLIENT_LOG="${ROOT_DIR}/.client.log"
TARGET_LOG="${ROOT_DIR}/.target.log"

cleanup() {
  set +e
  if [[ -n "${CLIENT_PID:-}" ]]; then kill "${CLIENT_PID}" 2>/dev/null; fi
  if [[ -n "${SERVER_PID:-}" ]]; then kill "${SERVER_PID}" 2>/dev/null; fi
  if [[ -n "${TARGET_PID:-}" ]]; then kill "${TARGET_PID}" 2>/dev/null; fi
}
trap cleanup EXIT

cd "${ROOT_DIR}"
go build ./...

python3 -m http.server "${TARGET_PORT}" >"${TARGET_LOG}" 2>&1 &
TARGET_PID=$!

sleep 0.5

go run ./cmd/reverse-tunnel server \
  --listen-a ":${PUBLIC_PORT}" \
  --listen-b ":${CLIENT_PORT}" \
  --pair-timeout "${PAIR_TIMEOUT}" >"${SERVER_LOG}" 2>&1 &
SERVER_PID=$!

sleep 0.5

go run ./cmd/reverse-tunnel client \
  --server "127.0.0.1:${CLIENT_PORT}" \
  --target "127.0.0.1:${TARGET_PORT}" >"${CLIENT_LOG}" 2>&1 &
CLIENT_PID=$!

sleep 1

if curl -fsS "http://127.0.0.1:${PUBLIC_PORT}" >/dev/null; then
  echo "E2E OK: proxy chain is working (public:${PUBLIC_PORT} -> target:${TARGET_PORT})"
else
  echo "E2E FAILED"
  echo "==== server log ===="
  cat "${SERVER_LOG}" || true
  echo "==== client log ===="
  cat "${CLIENT_LOG}" || true
  exit 1
fi
