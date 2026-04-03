# helpy

Helpy is an office-workflow assistant with:

- email (Gmail OAuth + IMAP)
- Google Calendar
- ICS feeds
- spreadsheets (`.xlsx`)
- optional API server, chat, and voice endpoints

## Docs

Canonical docs are in:

- `docs/index.md`

Key docs:

- Go API quickstart: `docs/go-api.md`
- Versioning policy (`v0`): `docs/versioning.md`
- Migration notes: `docs/migration.md`

Public Go API packages live under `pkg/*` and are documented with normal Go tooling (`go doc`, pkg.go.dev style comments/examples).

## Build

```bash
make build
```

## Run Server (v0, localhost-only)

```bash
helpy serve
```

Canonical REST base path:

- `/api/v0/...`

Legacy compatibility alias (temporary):

- `/api/...`

## Voice APIs

Endpoint:

- `POST /api/v0/voice/tts`
- `GET /api/v0/voice/tts/health`
- `GET /api/v0/voice/voices`

STT has been removed from Helpy and is no longer exposed by `/api/v0/voice/*`.
Use VoxType MCP for speech-to-text.

## Run MCP Server (v0, localhost-only)

```bash
helpy mcp-serve --bind 127.0.0.1 --port 8090
```

MCP endpoint:

- `http://127.0.0.1:8090/mcp`

## Codex/Claude MCP Setup

Configure Helpy as its own MCP server key (`helpy`) for both Codex and Claude:

```bash
./scripts/setup-helpy-mcp.sh http://127.0.0.1:8090/mcp
```

Individual scripts:

- `scripts/setup-codex-mcp.sh`
- `scripts/setup-claude-mcp.sh`
- `scripts/setup-helpy-mcp.sh`

Direct dual-server mode:

- configure `helpy` and `tabula` separately in Codex/Claude
- let the assistant orchestrate calls between the two MCP servers

## Dev Hot Reload (Systemd User Units)

Unit templates and install helper:

- `deploy/systemd/user/helpy-mcp.service`
- `deploy/systemd/user/helpy-dev-watch.service`
- `scripts/install-helpy-user-units.sh`

Install/enable:

```bash
./scripts/install-helpy-user-units.sh
```

This keeps the local Helpy MCP daemon running and automatically restarts it when Helpy source/config files change.
