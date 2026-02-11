# Tabula

Tabula is a Linux desktop monolith for fast local AI-assisted coding iteration.

## v1 Architecture

- Desktop UI: React + xterm.js (`apps/desktop`)
- Desktop runtime: Tauri + Rust PTY manager + SQLite (`apps/desktop/src-tauri`)
- Shared domain logic: TypeScript package (`packages/domain`)
- Implementation-independent UX source: `docs/design-system`

## Quick Start

```bash
npm install
npm run validate:design-system
npm run test:domain
npm run dev:desktop
```

## Quality Gates

- Design system strict traceability: `npm run validate:design-system`
- Domain unit tests: `npm run test:domain`
