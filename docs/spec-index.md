# Spec Index (Go)

## CLI and Command Surface

- `cmd/tabula/main.go`

## MCP Protocol / Tools

- `internal/mcp/server.go`
- `internal/canvas/adapter.go`
- `internal/canvas/events.go`

Notable MCP tools for cross-server integration:

- `handoff.create|peek|consume|revoke|status` (producer side, see `../helpy`)
- `canvas_import_handoff` (consumer side in Tabula)

## HTTP MCP Daemon

- `internal/serve/app.go`

## Web UI Backend

- `internal/web/server.go`
- `internal/store/store.go`
- `internal/pty/pty.go`
- `internal/pty/ptyd_transport.go`

Notable UI routing behavior:

- `/canvas` redirects to desktop canvas mode (`/?desktop=1`)
- local browser session id is `local`

## PTY Daemon

- `internal/ptyd/app.go`

## Bootstrap / Protocol Files

- `internal/protocol/bootstrap.go`

## Browser UI Assets

- `internal/web/static/index.html`
- `internal/web/static/app.js`
- `internal/web/static/canvas.js`
- `internal/web/static/terminal.js`
- `internal/web/static/style.css`
