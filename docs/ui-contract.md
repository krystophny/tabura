# UI Contract

Tabura now treats shared UI design as a first-class source of truth, not just an after-the-fact test artifact.

## Layered Source Of Truth

- Component contract:
  `internal/web/static/tabura-circle-contract.ts`
- Interaction flows:
  `tests/flows/`
- Target mapping per platform:
  `tests/flows/targets.cjs`

Each platform may render with its own native toolkit, but it must preserve the same semantic contract. The web runtime is not the source of truth because it is HTML; it is the source of truth only where it implements the shared contract.

## Document Navigation Contract

Canvas document navigation is also part of the shared UI contract across web, iOS, and Android.

- The canvas is no-scroll for artifact reading.
- Document artifacts are paginated into discrete page units.
- Short horizontal swipe means `next_page` or `previous_page`, with automatic rollover to the next or previous artifact only at document boundaries.
- Long-held horizontal swipe means `next_artifact` or `previous_artifact` immediately, even if more pages remain in the current artifact.
- Horizontal wheel and trackpad gestures follow the short-swipe rule.
- Inbox item swipe and edge-panel swipe must share the same horizontal-intent thresholds and axis-dominance semantics, even when the resulting action differs.

## Tabura Circle Contract

The Tabura Circle contract currently defines:

- stable segment ids for `dialogue`, `meeting`, `silent`, `prompt`, `text_note`, `pointer`, `highlight`, and `ink`
- icon-only rendering with accessible labels and tooltips
- a corner enum of `top_left`, `top_right`, `bottom_left`, `bottom_right`
- corner-aware quarter-fan geometry computed from one anchor and one set of polar layout tuples
- local per-device persistence for circle placement
- bug reporting as a top-panel action instead of a second floating control

## Platform Rule

Web, iOS, and Android must share:

- ids and accessibility identifiers
- icon meaning
- active/inactive state semantics
- corner placement semantics
- interaction flows
- paged canvas navigation semantics

Web, iOS, and Android do not need to share:

- DOM structure
- CSS layout primitives
- widget classes
- animation implementation details
- gesture-recognizer classes or event APIs

Native clients should use toolkit-native implementations that realize the shared contract instead of recreating browser markup inside native shells.
