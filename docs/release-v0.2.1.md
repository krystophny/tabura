# Release v0.2.1

> **Legal notice:** Slopshell is provided "as is" and "as available" without warranties, and to the maximum extent permitted by applicable law the authors/contributors accept no liability for damages, data loss, or misuse. You are solely responsible for backups, verification, and safe operation. See [`DISCLAIMER.md`](/DISCLAIMER.md).

## Scope

`v0.2.1` consolidates the current app-server-centered runtime, simplifies model switching around Spark-first dialogue with direct per-turn GPT override, and ships the current dialogue/companion UI cleanup.

## Highlights

### Runtime Simplification

- Spark remains the default live dialogue partner on the persistent app-server thread.
- GPT escalation is now a direct per-turn override on that same thread instead of provider parallelism or side delegation paths.
- Gemini grounded-search routing and other abandoned parallel model paths were removed from the active runtime.

### Dialogue and Companion UX

- Dialogue mode again supports explicit canvas artifact rendering when the assistant returns file-backed output.
- The companion surface was flattened to a white canvas, matching the simplified compose and idle surfaces.
- When the companion face is visible in dialogue mode, it now acts as the only activity indicator instead of competing with overlay frame/dot/play cues.

### Runtime Hygiene

- Persistent stale model-switch actions and old delegation surfaces were removed.
- Local workspace/runtime defaults were cleaned up to prefer the actual `slopshell` repository path.
- Version surfaces are aligned across the CLI, web runtime, MCP server, and app-server handshake metadata for `v0.2.1`.

## Verification Scope

This release is intended to be verified against:

- `./scripts/sync-surface.sh --check`
- `go test ./...`
- `./scripts/playwright.sh`
- `scripts/check-version-consistency.sh`

## Traceability

For publication metadata, associate this release with:

- release label: `v0.2.1`
- repository: `https://github.com/krystophny/slopshell`
- exact source revision: tag target commit hash
