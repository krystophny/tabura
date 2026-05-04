#!/usr/bin/env bash
set -euo pipefail

CONFIG_PATH="${CODEX_CONFIG_PATH:-$HOME/.codex/config.toml}"
PLATFORM="$(uname -s)"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=scripts/lib/llm_env.sh
source "${SCRIPT_DIR}/lib/llm_env.sh"
DEFAULT_CODEX_URL="${SLOPSHELL_CODEX_BASE_URL:-$(slopshell_resolve_openai_base_url 2>/dev/null || printf 'http://127.0.0.1:8080/v1')}"
FAST_URL="${SLOPSHELL_CODEX_FAST_URL:-${DEFAULT_CODEX_URL}}"
FAST_MODEL="${SLOPSHELL_CODEX_FAST_MODEL:-qwen}"
SLOPTOOLS_BIN="${SLOPSHELL_SLOPTOOLS_BIN:-$HOME/.local/bin/sloptools}"
HELPY_BIN="${SLOPSHELL_HELPY_BIN:-$HOME/.local/bin/helpy}"
SLOPPY_PROJECT_DIR="${SLOPSHELL_SLOPPY_PROJECT_DIR:-$HOME}"
SLOPPY_DATA_DIR="${SLOPSHELL_SLOPPY_DATA_DIR:-$HOME/.local/share/sloppy}"
LOCAL_URL="${SLOPSHELL_CODEX_LOCAL_URL:-${DEFAULT_CODEX_URL}}"
LOCAL_MODEL="${SLOPSHELL_CODEX_LOCAL_MODEL:-qwen}"
LOCAL_PROVIDER_NAME="OpenAI-compatible local"
FAST_PROVIDER_NAME="OpenAI-compatible fast"
MCP_MARKER_BEGIN="# BEGIN SLOPSHELL MCP"
MCP_MARKER_END="# END SLOPSHELL MCP"
MODELS_MARKER_BEGIN="# BEGIN SLOPSHELL LOCAL MODELS"
MODELS_MARKER_END="# END SLOPSHELL LOCAL MODELS"

mkdir -p "$(dirname "$CONFIG_PATH")"
if [[ -f "$CONFIG_PATH" ]]; then
  cp "$CONFIG_PATH" "$CONFIG_PATH.bak.$(date +%Y%m%d%H%M%S)"
fi

TMP_BASE="$(mktemp)"
TMP_OUT="$(mktemp)"
cleanup() {
  rm -f "$TMP_BASE" "$TMP_OUT"
}
trap cleanup EXIT

strip_block() {
  local input="$1"
  local output="$2"
  local begin="$3"
  local end="$4"
  awk -v begin="$begin" -v end="$end" '
    $0 == begin { in_block = 1; next }
    $0 == end { in_block = 0; next }
    !in_block { print }
  ' "$input" >"$output"
}

strip_tables() {
  local input="$1"
  local output="$2"
  local table_pattern="$3"
  awk -v table_pattern="$table_pattern" '
    /^\[[^]]+\]$/ {
      table = $0
      gsub(/^\[/, "", table)
      gsub(/\]$/, "", table)
      skip = (table ~ table_pattern)
    }
    !skip && $0 !~ /^# (BEGIN|END) sloppy\/helpy stdio MCP servers/ { print }
  ' "$input" >"$output"
}

if [[ -f "$CONFIG_PATH" ]]; then
  strip_block "$CONFIG_PATH" "$TMP_BASE.mcp" "$MCP_MARKER_BEGIN" "$MCP_MARKER_END"
  strip_block "$TMP_BASE.mcp" "$TMP_BASE.models" "$MODELS_MARKER_BEGIN" "$MODELS_MARKER_END"
  strip_tables \
    "$TMP_BASE.models" \
    "$TMP_BASE" \
    '^(mcp_servers[.](sloppy|helpy|sloptools|slopshell)|model_providers[.](local|fast)|profiles[.](local|fast))$'
  rm -f "$TMP_BASE.mcp" "$TMP_BASE.models"
else
  : >"$TMP_BASE"
fi

{
  cat "$TMP_BASE"
  if [[ -s "$TMP_BASE" ]]; then
    echo
  fi
  echo "$MCP_MARKER_BEGIN"
  echo "[mcp_servers.sloppy]"
  printf 'command = "%s"\n' "$SLOPTOOLS_BIN"
  printf 'args = ["mcp-server", "--project-dir", "%s", "--data-dir", "%s"]\n' "$SLOPPY_PROJECT_DIR" "$SLOPPY_DATA_DIR"
  echo
  echo "[mcp_servers.helpy]"
  printf 'command = "%s"\n' "$HELPY_BIN"
  echo 'args = ["mcp-stdio"]'
  echo "$MCP_MARKER_END"
  echo
  echo "$MODELS_MARKER_BEGIN"
  echo "[model_providers.local]"
  printf 'name = "%s"\n' "$LOCAL_PROVIDER_NAME"
  printf 'base_url = "%s"\n' "$LOCAL_URL"
  echo 'wire_api = "responses"'
  echo
  echo "[model_providers.fast]"
  printf 'name = "%s"\n' "$FAST_PROVIDER_NAME"
  printf 'base_url = "%s"\n' "$FAST_URL"
  echo 'wire_api = "responses"'
  echo
  echo "[profiles.local]"
  echo 'model_provider = "local"'
  printf 'model = "%s"\n' "$LOCAL_MODEL"
  echo 'model_reasoning_effort = "high"'
  echo
  echo "[profiles.fast]"
  echo 'model_provider = "fast"'
  printf 'model = "%s"\n' "$FAST_MODEL"
  echo 'model_reasoning_effort = "minimal"'
  echo "$MODELS_MARKER_END"
  echo
} >"$TMP_OUT"

mv "$TMP_OUT" "$CONFIG_PATH"
echo "updated $CONFIG_PATH"
echo "server keys: mcp_servers.sloppy, mcp_servers.helpy"
echo "profile keys: local, fast"
