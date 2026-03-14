#!/usr/bin/env sh
set -eu

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname "$0")" && pwd)
REPO_ROOT=$(CDPATH= cd -- "$SCRIPT_DIR/.." && pwd)

read_env_value() {
  key="$1"
  env_file="$2"
  sed -n "s/^${key}=//p" "$env_file" | tail -n 1
}

HOME_SERVER_VALUE=""
HOME_SERVER_USER_VALUE=""
HOME_SERVER_DIR1_VALUE=""
HOME_SERVER_DIR2_VALUE=""

if [ -f "$REPO_ROOT/.env" ]; then
  HOME_SERVER_VALUE=$(read_env_value HOME_SERVER "$REPO_ROOT/.env")
  HOME_SERVER_USER_VALUE=$(read_env_value HOME_SERVER_USER "$REPO_ROOT/.env")
  HOME_SERVER_DIR1_VALUE=$(read_env_value HOME_SERVER_DIR "$REPO_ROOT/.env")
  HOME_SERVER_DIR2_VALUE=$(read_env_value HOME_SERVER_DIR2 "$REPO_ROOT/.env")
fi

if [ "${MCP_PI_SSH_TARGET:-}" = "" ] && [ "$HOME_SERVER_VALUE" != "" ]; then
  if [ "$HOME_SERVER_USER_VALUE" != "" ]; then
    export MCP_PI_SSH_TARGET="${HOME_SERVER_USER_VALUE}@${HOME_SERVER_VALUE}"
  else
    export MCP_PI_SSH_TARGET="${HOME_SERVER_VALUE}"
  fi
fi

cd "$REPO_ROOT/tools/mcp-pi"
exec env \
  GOCACHE="${GOCACHE:-/tmp/villa73-mcp-go-build}" \
  MCP_PI_ALLOWED_PROJECT_DIR1="${MCP_PI_ALLOWED_PROJECT_DIR1:-$HOME_SERVER_DIR1_VALUE}" \
  MCP_PI_ALLOWED_PROJECT_DIR2="${MCP_PI_ALLOWED_PROJECT_DIR2:-$HOME_SERVER_DIR2_VALUE}" \
  go run .
