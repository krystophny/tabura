# Release v0.1.7

> **Legal notice:** Slopshell is provided "as is" and "as available" without warranties, and to the maximum extent permitted by applicable law the authors/contributors accept no liability for damages, data loss, or misuse. You are solely responsible for backups, verification, and safe operation. See [`DISCLAIMER.md`](/DISCLAIMER.md).

## Scope

`v0.1.7` focuses on lower-latency voice interaction and a stronger local coordinator path, with Silero VAD v5 replacing legacy energy-threshold VAD and broad test coverage updates across chat, intent/delegation, and UI integration paths.

## Highlights

### Silero VAD v5 Integration

- Replaced energy-based browser VAD with Silero VAD v5 flow.
- Added model asset fetch script and runtime wiring for VAD assets.
- Updated frontend voice-capture behavior to use the new detector path.

### Intent/Delegation and Runtime Improvements

- Consolidated local coordinator behavior for intent and delegation routing.
- Added runtime preference handling updates in web/API surface.
- Updated service defaults and install scripts for the local model/runtime stack.

### Test and Quality Coverage

- Expanded unit tests for intent planning, action policy, model profiles, participant/session handling, and web runtime preferences.
- Added/updated Playwright and E2E suites for app load, STT/TTS paths, websocket flows, and voice roundtrip/system checks.
- Added CI test-report workflow refinements.

## Documentation and Protocol Alignment

- Refreshed architecture/spec/interface docs for current runtime behavior.
- Updated handoff protocol docs and related whitepapers.
- Kept legal/disclaimer notice consistently visible across primary docs.

## Traceability

For publication metadata, associate this release with:

- release label: `v0.1.7`
- repository: `https://github.com/sloppy-org/slopshell`
- exact source revision: tag target commit hash
