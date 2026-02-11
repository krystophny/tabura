# Tabula Design System Source of Truth

This folder is the canonical, implementation-independent UX source for Tabula v1.

Canonical precedence:

1. BDD features (`features/*.feature`)
2. Behavior contracts (`contracts/*.json`)
3. Artifact bundle contracts (`artifacts/*.bundle.json`)
4. Interaction flows (`flows/*.json`)
5. Platform mappings (`platform-mappings/*.mapping.json`)
6. Tokens (`tokens/*.json`)

Diagrams are optional and non-canonical in v1.

## Process

1. Update feature files first.
2. Update contracts/flows/bundles/mappings/tokens/traceability.
3. Run `./scripts/design-system-quality-gate.sh --strict-traceability`.
4. Implement only after validation passes.
