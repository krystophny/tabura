#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

"$SCRIPT_DIR/setup-codex-mcp.sh"
"$SCRIPT_DIR/setup-claude-mcp.sh"
"$SCRIPT_DIR/setup-gemini-mcp.sh"

echo "Configured Codex, Claude and Gemini to use local stdio MCP servers: sloppy, helpy"
