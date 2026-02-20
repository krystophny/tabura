# Tabula Architecture (Go)

Tabula is a Go-first MCP canvas/runtime stack with a browser UI.

## Components

- `cmd/tabula/main.go`
  - CLI entrypoint and subcommand dispatch.
- `internal/mcp/server.go`
  - MCP JSON-RPC server (`initialize`, `tools/list`, `tools/call`).
- `internal/canvas/adapter.go`
  - Canvas/session/annotation state and tool implementations.
- `internal/serve/app.go`
  - Local MCP HTTP daemon (`/mcp`, `/ws/canvas`, `/files/*`).
- `internal/web/server.go`
  - Web UI backend (auth, host/session APIs, terminal/canvas websockets).
- `internal/ptyd/app.go`
  - PTY daemon to keep terminal sessions alive across web restarts.
- `internal/pty/*.go`
  - Local/PTYD PTY transport implementations.
- `internal/store/store.go`
  - SQLite persistence for web auth/hosts/session mappings.
- `internal/protocol/bootstrap.go`
  - Project bootstrap (`.tabula/*`, protocol files, gitignore wiring).

## Data Flow

1. Assistant calls MCP tools through `tabula mcp-server` (stdio) or `tabula serve` (HTTP MCP).
2. MCP calls resolve in `internal/canvas/adapter.go`.
3. Canvas events are broadcast over websocket to the browser canvas UI.
4. Web terminal traffic is handled via local PTY or PTYD transport.

## Cross-Server Handoff Flow (Helpy -> Tabula)

1. Producer server (for example Helpy) resolves source data and creates handoff via `handoff.create`.
2. Consumer server (Tabula) imports via `canvas_import_handoff` with `handoff_id` and `producer_mcp_url`.
3. Tabula validates kind with `handoff.peek`, then consumes with `handoff.consume`.
4. Imported payload is rendered as a canvas artifact and tagged with handoff metadata.

Design rule:

- Producer owns source-system access (email, file, sheet, etc.).
- Tabula consumes opaque handoff IDs and does not need direct source access.
- Controller should hand off to canvas instead of dumping payload in terminal/chat when UI testing is intended.

## Runtime Modes

- CLI stdio MCP: `tabula mcp-server`
- Local MCP HTTP daemon: `tabula serve`
- Browser UI: `tabula web`
- Desktop canvas browser mode: `tabula canvas` (opens `/canvas`)
- PTY daemon: `tabula ptyd`

Default local integration addresses:

- Tabula web: `http://localhost:8420`
- Tabula MCP: `http://127.0.0.1:9420/mcp`
- Helpy MCP: `http://127.0.0.1:8090/mcp`
- Local session id: `local`
