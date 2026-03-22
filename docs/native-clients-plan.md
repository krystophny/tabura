# Native Clients Plan

> **Legal notice:** Tabura is provided "as is" and "as available" without warranties, and to the maximum extent permitted by applicable law the authors/contributors accept no liability for damages, data loss, or misuse. You are solely responsible for backups, verification, and safe operation. See [`DISCLAIMER.md`](/DISCLAIMER.md).

## Architecture Decision

Tabura's mobile direction is server-driven thin native clients.

Business logic lives in the Go server. Native clients stay focused on platform
I/O, low-latency capture, native rendering, and background/runtime integration.
That keeps behavior fixes centralized in `internal/web/` and `internal/mcp/`
instead of splitting product logic across multiple frontends.

Each native client owns three responsibilities:

- **Capture**: audio PCM, ink strokes, taps, and gestures.
- **Render**: structured chat/canvas output rendered with native surfaces.
- **Platform services**: background audio, push notifications, wake locks, and
  e-ink refresh hooks where applicable.

## Current Server Anchors

The current server already exposes the foundations required by thin native
clients:

- `internal/web/chat_ws.go` for chat websocket turns.
- `internal/web/server_relay.go` for canvas relay and file-backed canvas
  transport.
- `internal/web/mdns.go` for loopback-safe mDNS advertisement of the runtime.
- `internal/web/push.go` for push registration and relay plumbing.
- `tests/playwright/canvas.spec.ts` for render-protocol coverage.

## Platform Surfaces

The shipped native surfaces match the thin-client split.

### iOS

- `platforms/ios/TaburaIOS/TaburaInkCaptureView.swift` uses `PencilKit` for ink
  capture.
- `platforms/ios/TaburaIOS/TaburaAudioCapture.swift` owns microphone capture.
- `platforms/ios/TaburaIOS/TaburaCanvasTransport.swift` and
  `platforms/ios/TaburaIOS/TaburaChatTransport.swift` connect to the server.
- `platforms/ios/TaburaIOS/TaburaServerDiscovery.swift` handles `_tabura._tcp`
  discovery.

### Android and Boox

- `platforms/android/app/src/main/kotlin/com/tabura/android/TaburaInkSurfaceView.kt`
  uses `MotionEventPredictor` for low-latency stylus capture.
- `platforms/android/app/src/main/kotlin/com/tabura/android/TaburaBooxInkSurfaceView.kt`
  uses `TouchHelper.create` and raw drawing for Onyx Boox devices.
- `platforms/android/app/src/main/kotlin/com/tabura/android/TaburaAudioCaptureService.kt`
  owns background microphone capture.
- `platforms/android/app/src/main/kotlin/com/tabura/android/TaburaCanvasTransport.kt`
  and
  `platforms/android/app/src/main/kotlin/com/tabura/android/TaburaChatTransport.kt`
  connect to the server.
- `platforms/android/app/src/main/kotlin/com/tabura/android/TaburaServerDiscovery.kt`
  handles `_tabura._tcp` discovery.

### Web

- `internal/web/static/app-runtime-ui.ts` toggles `black-screen` dialogue mode.
- `internal/web/static/companion.css` defines the black-screen runtime surface.
- `tests/playwright/live-dialogue-companion.spec.ts` covers black-screen
  dialogue behavior.

## Ink Latency Targets

The current runtime keeps the original latency targets as design guidance:

| Platform | Target | Technique |
| --- | --- | --- |
| iOS + Apple Pencil | ~9ms | `PencilKit` with native prediction |
| Android + stylus | ~4ms | `MotionEventPredictor` with the Ink stack |
| Onyx Boox e-ink | ~100ms | raw drawing via `TouchHelper` |
| Web + stylus | ~10ms | browser ink path with delegated presentation where available |

These are target envelopes, not CI-enforced benchmarks. The concrete shipped
techniques are anchored in the platform files above.

## Delivery Status

The original implementation slices for the native-client push are complete:

- `#632` server-side render protocol
- `#633` native iOS client
- `#634` native Android client
- `#635` Onyx Boox support
- `#636` web ink rewrite
- `#637` black-screen dialogue mode
- `#638` mDNS advertisement and push relay

Issue `#639` remains useful as the umbrella record, but the actual plan now
lives in this canonical repo document instead of a missing private plan path.
