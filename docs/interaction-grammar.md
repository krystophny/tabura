# Interaction Grammar

This document is the canonical product reference for Tabura's interaction model.

If code, UI copy, storage, tests, or follow-on design notes disagree with this document, the implementation is wrong and should be changed. The document is not a compatibility promise for legacy shapes.

## Authoritative Ontology

Tabura has exactly five primary product nouns:

- **Workspace** — a real directory the user works in. Always filesystem-grounded. Composed via symlinks and environment variables, not virtual abstractions. Thin record: path, name, sphere, active flag, chat config. Archivable as a self-contained unit.
- **Artifact** — curated content shown on canvas. Not every file is an artifact; artifacts are created lazily when interacted with, synced from external systems, or explicitly captured. Has kind, file path or URL, and metadata. Non-file artifacts (email, issue) can be materialized as real files in a workspace.
- **Item** — an open loop requiring attention. Tracked in the inbox. Has state (inbox/waiting/someday/done), an optional artifact, an optional actor, and a workspace where it is tracked. Items do not always have artifacts; bare tasks like "call Bob" are items without artifacts.
- **Actor** — a human or agent responsible for progress.
- **Label** — a flat organizational tag. Attached to items, artifacts, and workspaces. The only cross-cutting grouping mechanism. Workspace labels cascade: querying a label includes items and artifacts in workspaces carrying that label.

Project is not a product concept. Session and message are transport or storage details, not user-facing ontology.

### Key invariants

1. The canvas shows artifacts, always. Bare items (no artifact) show in the sidebar only.
2. An item may have an artifact. The common case (email, PR, document) does. Bare tasks do not. No synthetic canvas content is generated for bare items.
3. A workspace is a directory. No virtual workspaces. Composition via filesystem tools.
4. Labels are the only cross-cutting grouping mechanism. No ProjectID anywhere.
5. Sphere (work/private) stays as a dedicated field, not a label.

### Materialization

Non-file artifacts linked to a workspace can be materialized as real files: email to .eml, GitHub issue to .md, external task to .md. This enables workspace archival: tar the directory and get a self-contained unit.

## Authoritative Live Model

Tabura exposes exactly two live runtime modes:

- Dialogue
- Meeting

Anything else is either a temporary implementation detail or a bug in naming, UX, or state modeling.

## Canonical Action Semantics

All shipped actions must reduce to this grammar:

- Open / Show
- Annotate / Capture
- Compose
- Bundle / Review
- Dispatch / Execute
- Track as Item
- Delegate to Actor

These actions may appear in different UI contexts, but they must not change meaning across surfaces.

## Allowed Tool Modalities

Tabura may accept input through these modalities:

- tap-to-voice
- right-click and type
- keyboard direct input
- ink or stylus
- live Dialogue
- live Meeting
- narrow intake surfaces

These are input paths into the same system, not alternate product models.

## Rules for Auxiliary Surfaces

Auxiliary surfaces are allowed only when all of the following are true:

- The surface is narrower than the main workspace.
- The surface exists to make one job materially faster.
- The surface writes back into the same Workspace / Artifact / Item / Actor / Label ontology.
- The surface does not create its own action grammar.
- The surface does not create a parallel runtime shell, inbox, review system, or workspace universe.

If a surface cannot satisfy all of those constraints, it does not belong in the product.

## Rules for New Artifact Kinds

New artifact kinds are allowed only when all of the following are true:

- The artifact still participates in the same ontology.
- The artifact can be opened, annotated, composed around, reviewed, dispatched, tracked, or delegated without inventing a separate grammar.
- Any artifact-specific affordance remains a narrow specialization of the canonical actions above.
- The artifact does not require its own naming scheme for live modes, state, or workflow concepts.

Artifact kinds may add presentation or extraction details. They must not define a second product.

## Design Lineage

The intellectual foundations for this interaction model are documented in `design-lineage.md`. The core claim, supported by six decades of computing research and cognitive science, is that the workspace (not the app) should be the unit of design, and that action is situated (not pre-planned), which is why annotation-accumulation beats immediate-dispatch.
