#!/usr/bin/env sh
set -eu

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname "$0")" && pwd)
REPO_ROOT=$(CDPATH= cd -- "$SCRIPT_DIR/.." && pwd)

read_env_value() {
  key="$1"
  env_file="$2"
  sed -n "s/^${key}=//p" "$env_file" | tail -n 1
}

DEFAULT_TARGET_VALUE=""

if [ -f "$REPO_ROOT/.env" ]; then
  DEFAULT_TARGET_VALUE=$(read_env_value MCP_PI_DEFAULT_TARGET "$REPO_ROOT/.env")
fi

cd "$REPO_ROOT/tools/mcp-pi"
exec env \
  GOCACHE="${GOCACHE:-/tmp/villa73-mcp-go-build}" \
  MCP_PI_SERVER_CMD="${MCP_PI_SERVER_CMD:-$REPO_ROOT/scripts/run-pi-mcp.sh}" \
  MCP_PI_DEFAULT_TARGET="${MCP_PI_DEFAULT_TARGET:-$DEFAULT_TARGET_VALUE}" \
  go run ./cmd/mcp-pi-tool "$@"
