# Tabula Desktop Monolith v1

## Overview

Tabula v1 is a Linux desktop application built with Tauri + React.
The runtime is local-only and monolithic:

- UI: React + xterm.js
- Core runtime: Tauri (Rust) with local PTY session manager
- Storage: local SQLite at `~/.tabula/state.db`
- Artifact files: `~/.tabula/artifacts/<turn-id>/`

No network backend is required for v1.

## Product Boundaries

In scope:

- Explicit mode tuple: `context(global|project)` + `cognitive(dialogue|plan|execution|review)`
- Composer prompt submission routed to local PTY sessions (`codex`, `shell`)
- Terminal drawer as secondary surface with session continuity
- Artifact bundle v1: canonical text + optional diff patch
- Warning guardrails as advisory only (non-blocking)

Out of scope for v1:

- Voice capture/STT/TTS
- SSH remote execution and credential management
- Android/web delivery surface
- Hard enforcement of plan locks/global write approvals

## Runtime Interfaces

Tauri commands:

- `start_session(profile, cwd?)`
- `send_input(session_id, data)`
- `resize_session(session_id, cols, rows)`
- `stop_session(session_id)`
- `set_mode(context_mode, cognitive_mode)`
- `submit_prompt(session_id, prompt, mode, plan_lock_hash?)`
- `list_artifacts(turn_id?)`
- `load_artifact(artifact_id)`
- `list_warnings(turn_id?)`

Tauri events:

- `term_output`
- `term_status`
- `mode_changed`
- `warning_emitted`
- `turn_completed`

## Persistence Model

SQLite tables:

- `sessions`
- `session_events`
- `turns`
- `artifacts`
- `warnings`
- `app_state`

Artifact filesystem:

- `canonical-text.md` is required for each turn
- `latest.patch` is optional and generated when diff-like content is detected

## Guardrail Behavior (v1)

Warnings are generated and persisted but never block execution.

- Project execution without plan lock -> `plan_lock_missing` warning
- Global prompts with risky write/exec intent -> `global_write_risk` warning

## Quality Gates

Required checks:

- `scripts/design-system-quality-gate.sh --strict-traceability`
- Domain unit tests
- Desktop frontend tests (when added)
- Desktop runtime tests (Rust)

## Development Loop

- Install workspace deps from repo root: `npm install`
- Run design-system validator: `npm run validate:design-system`
- Run domain tests: `npm run test:domain`
- Launch desktop app: `npm run dev:desktop`
