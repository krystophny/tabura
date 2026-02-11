#!/usr/bin/env python3
from __future__ import annotations

import argparse
import json
import re
import sys
from pathlib import Path
from typing import Any

try:
    import jsonschema
except ImportError as exc:  # pragma: no cover
    raise SystemExit("jsonschema is required. Run ./scripts/design-system-quality-gate.sh") from exc

ROOT = Path(__file__).resolve().parents[1]
DESIGN_ROOT = ROOT / "docs" / "design-system"
SCHEMA_ROOT = DESIGN_ROOT / "schemas"
FEATURES_ROOT = DESIGN_ROOT / "features"

SCENARIO_ID_RE = re.compile(r"^@SCN-[A-Z0-9-]+$")
REQUIRED_CLASS_TAGS = {"@happy_path", "@safety_guardrail", "@failure_recovery"}

REQUIRED_DOCS = [
    DESIGN_ROOT / "README.md",
    DESIGN_ROOT / "governance.md",
    DESIGN_ROOT / "changelog.md",
    DESIGN_ROOT / "principles.md",
    DESIGN_ROOT / "information-architecture.md",
    DESIGN_ROOT / "accessibility.md",
    DESIGN_ROOT / "signoff-checklist.md",
    FEATURES_ROOT / "README.md",
    DESIGN_ROOT / "contracts" / "README.md",
    DESIGN_ROOT / "flows" / "README.md",
    DESIGN_ROOT / "tokens" / "README.md",
    DESIGN_ROOT / "artifacts" / "README.md",
    DESIGN_ROOT / "platform-mappings" / "README.md",
    DESIGN_ROOT / "traceability" / "README.md",
    SCHEMA_ROOT / "README.md",
]

PATTERN_TO_SCHEMA = {
    "tokens/*.json": "DesignTokenSet.schema.json",
    "contracts/*.json": "BehaviorContract.schema.json",
    "flows/*.json": "InteractionFlow.schema.json",
    "artifacts/*.bundle.json": "ArtifactBundle.schema.json",
    "platform-mappings/*.mapping.json": "PlatformMapping.schema.json",
    "traceability/feature-traceability.json": "FeatureTraceability.schema.json",
}


def load_json(path: Path) -> Any:
    with path.open("r", encoding="utf-8") as f:
        return json.load(f)


def validate_file(instance: Path, schema: Path) -> list[str]:
    obj = load_json(instance)
    sch = load_json(schema)
    validator_cls = jsonschema.validators.validator_for(sch)
    validator_cls.check_schema(sch)
    validator = validator_cls(sch)
    errors = sorted(validator.iter_errors(obj), key=lambda e: list(e.absolute_path))
    out: list[str] = []
    for e in errors:
        ptr = "/".join(str(p) for p in e.absolute_path) or "<root>"
        out.append(f"{instance}: {ptr}: {e.message}")
    return out


def check_required_files() -> list[str]:
    missing = [p for p in REQUIRED_DOCS if not p.exists()]
    return [f"Missing required artifact: {p}" for p in missing]


def gather_pairs() -> tuple[list[tuple[Path, Path]], list[str]]:
    errors: list[str] = []
    pairs: list[tuple[Path, Path]] = []
    for pattern, schema_name in PATTERN_TO_SCHEMA.items():
        matches = sorted(DESIGN_ROOT.glob(pattern))
        if not matches:
            errors.append(f"No files matched required pattern '{pattern}'")
            continue
        schema = SCHEMA_ROOT / schema_name
        if not schema.exists():
            errors.append(f"Missing schema {schema}")
            continue
        for m in matches:
            pairs.append((m, schema))
    return pairs, errors


def parse_features() -> tuple[dict[str, dict[str, str]], list[str]]:
    errors: list[str] = []
    scenarios: dict[str, dict[str, str]] = {}

    files = sorted(FEATURES_ROOT.glob("*.feature"))
    if not files:
        return scenarios, [f"No feature files found under {FEATURES_ROOT}"]

    for feature in files:
        pending_tags: list[str] = []
        class_counts = {k: 0 for k in REQUIRED_CLASS_TAGS}
        scenario_count = 0

        lines = feature.read_text(encoding="utf-8").splitlines()
        for line_no, raw in enumerate(lines, start=1):
            stripped = raw.strip()
            if not stripped or stripped.startswith("#"):
                continue
            if stripped.startswith("@"):
                pending_tags.extend(stripped.split())
                continue
            if stripped.startswith("Scenario"):
                scenario_count += 1
                tags = set(pending_tags)
                pending_tags = []

                scn_ids = [t for t in tags if SCENARIO_ID_RE.match(t)]
                if len(scn_ids) != 1:
                    errors.append(f"{feature}:{line_no}: scenario must have exactly one @SCN-* tag")
                    continue
                scn_id = scn_ids[0]
                if scn_id in scenarios:
                    errors.append(f"Duplicate scenario id {scn_id} in {feature}:{line_no}")
                    continue

                class_tags = [t for t in tags if t in REQUIRED_CLASS_TAGS]
                if len(class_tags) != 1:
                    errors.append(
                        f"{feature}:{line_no}: scenario {scn_id} must have exactly one class tag from "
                        f"{', '.join(sorted(REQUIRED_CLASS_TAGS))}"
                    )
                else:
                    class_counts[class_tags[0]] += 1

                scenarios[scn_id] = {
                    "feature": feature.name,
                    "line": str(line_no),
                }

        if scenario_count == 0:
            errors.append(f"{feature}: file has no scenarios")

        missing_classes = [tag for tag, count in class_counts.items() if count == 0]
        if missing_classes:
            errors.append(
                f"{feature}: missing required scenario class tags: {', '.join(sorted(missing_classes))}"
            )

    return scenarios, errors


def collect_contract_ids() -> tuple[set[str], dict[str, set[str]], list[str]]:
    ids: set[str] = set()
    scenarios_by_contract: dict[str, set[str]] = {}
    errors: list[str] = []

    for path in sorted((DESIGN_ROOT / "contracts").glob("*.json")):
        data = load_json(path)
        if not isinstance(data, dict):
            errors.append(f"{path}: expected object")
            continue
        cid = data.get("contractId")
        scenario_refs = data.get("scenarioRefs", [])
        if not isinstance(cid, str) or not cid:
            errors.append(f"{path}: missing contractId")
            continue
        if cid in ids:
            errors.append(f"Duplicate contractId: {cid}")
        ids.add(cid)
        refs: set[str] = set()
        if isinstance(scenario_refs, list):
            for ref in scenario_refs:
                if isinstance(ref, str):
                    refs.add(ref)
        scenarios_by_contract[cid] = refs

    return ids, scenarios_by_contract, errors


def collect_flow_ids() -> tuple[set[str], dict[str, set[str]], list[str]]:
    ids: set[str] = set()
    scenarios_by_flow: dict[str, set[str]] = {}
    errors: list[str] = []

    for path in sorted((DESIGN_ROOT / "flows").glob("*.json")):
        data = load_json(path)
        if not isinstance(data, dict):
            errors.append(f"{path}: expected object")
            continue
        fid = data.get("flowId")
        scenario_refs = data.get("scenarioRefs", [])
        if not isinstance(fid, str) or not fid:
            errors.append(f"{path}: missing flowId")
            continue
        if fid in ids:
            errors.append(f"Duplicate flowId: {fid}")
        ids.add(fid)
        refs: set[str] = set()
        if isinstance(scenario_refs, list):
            for ref in scenario_refs:
                if isinstance(ref, str):
                    refs.add(ref)
        scenarios_by_flow[fid] = refs

    return ids, scenarios_by_flow, errors


def collect_mapping_ids() -> tuple[set[str], list[str]]:
    ids: set[str] = set()
    errors: list[str] = []
    for path in sorted((DESIGN_ROOT / "platform-mappings").glob("*.mapping.json")):
        data = load_json(path)
        mid = data.get("mappingId") if isinstance(data, dict) else None
        if not isinstance(mid, str) or not mid:
            errors.append(f"{path}: missing mappingId")
            continue
        if mid in ids:
            errors.append(f"Duplicate mappingId: {mid}")
        ids.add(mid)
    return ids, errors


def collect_bundle_ids() -> tuple[set[str], list[str]]:
    ids: set[str] = set()
    errors: list[str] = []
    for path in sorted((DESIGN_ROOT / "artifacts").glob("*.bundle.json")):
        data = load_json(path)
        bid = data.get("bundleId") if isinstance(data, dict) else None
        if not isinstance(bid, str) or not bid:
            errors.append(f"{path}: missing bundleId")
            continue
        if bid in ids:
            errors.append(f"Duplicate bundleId: {bid}")
        ids.add(bid)

        artifacts = data.get("artifacts", []) if isinstance(data, dict) else []
        if not isinstance(artifacts, list) or not artifacts:
            errors.append(f"{path}: artifacts must be non-empty")
            continue
        has_text = any(isinstance(a, dict) and a.get("artifactType") == "text_response" for a in artifacts)
        if not has_text:
            errors.append(f"{path}: bundle must include at least one text_response artifact")

    return ids, errors


def check_traceability(
    scenario_map: dict[str, dict[str, str]],
    contract_ids: set[str],
    flow_ids: set[str],
    mapping_ids: set[str],
    bundle_ids: set[str],
    strict: bool,
    contract_scenarios: dict[str, set[str]],
    flow_scenarios: dict[str, set[str]],
) -> list[str]:
    errors: list[str] = []
    path = DESIGN_ROOT / "traceability" / "feature-traceability.json"
    data = load_json(path)
    if not isinstance(data, dict):
        return [f"{path}: expected object"]

    entries = data.get("scenarios")
    if not isinstance(entries, list):
        return [f"{path}: scenarios must be array"]

    seen: set[str] = set()
    traced: set[str] = set()

    for i, entry in enumerate(entries):
        if not isinstance(entry, dict):
            errors.append(f"{path}: scenarios[{i}] must be object")
            continue

        scn = entry.get("scenarioId")
        feature = entry.get("feature")
        contract_refs = entry.get("contractRefs")
        flow_refs = entry.get("flowRefs")
        mapping_refs = entry.get("mappingRefs")
        bundle_refs = entry.get("bundleRefs")

        if not isinstance(scn, str) or scn not in scenario_map:
            errors.append(f"{path}: unknown scenarioId '{scn}'")
            continue

        if scn in seen:
            errors.append(f"{path}: duplicate scenarioId '{scn}'")
            continue
        seen.add(scn)
        traced.add(scn)

        if feature != scenario_map[scn]["feature"]:
            errors.append(
                f"{path}: scenario '{scn}' feature mismatch; expected '{scenario_map[scn]['feature']}', got '{feature}'"
            )

        if not isinstance(contract_refs, list) or not contract_refs:
            errors.append(f"{path}: scenario '{scn}' missing contractRefs")
        else:
            for ref in contract_refs:
                if ref not in contract_ids:
                    errors.append(f"{path}: scenario '{scn}' unknown contract ref '{ref}'")

        if not isinstance(flow_refs, list) or not flow_refs:
            errors.append(f"{path}: scenario '{scn}' missing flowRefs")
        else:
            for ref in flow_refs:
                if ref not in flow_ids:
                    errors.append(f"{path}: scenario '{scn}' unknown flow ref '{ref}'")

        if not isinstance(mapping_refs, list) or not mapping_refs:
            errors.append(f"{path}: scenario '{scn}' missing mappingRefs")
        else:
            for ref in mapping_refs:
                if ref not in mapping_ids:
                    errors.append(f"{path}: scenario '{scn}' unknown mapping ref '{ref}'")

        if not isinstance(bundle_refs, list) or not bundle_refs:
            errors.append(f"{path}: scenario '{scn}' missing bundleRefs")
        else:
            for ref in bundle_refs:
                if ref not in bundle_ids:
                    errors.append(f"{path}: scenario '{scn}' unknown bundle ref '{ref}'")

    if strict:
        missing = sorted(set(scenario_map) - traced)
        for scn in missing:
            errors.append(f"{path}: strict traceability missing scenario '{scn}'")

        for scn in sorted(scenario_map):
            in_contract = any(scn in refs for refs in contract_scenarios.values())
            in_flow = any(scn in refs for refs in flow_scenarios.values())
            if not in_contract:
                errors.append(f"Strict traceability: scenario '{scn}' not linked by any contract")
            if not in_flow:
                errors.append(f"Strict traceability: scenario '{scn}' not linked by any flow")

    return errors


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Validate Tabula design-system artifacts")
    parser.add_argument("--strict-traceability", action="store_true", help="require full scenario traceability")
    parser.add_argument("--require-approved", action="store_true", help="require Implementation Gate approved")
    return parser.parse_args()


def check_signoff(require_approved: bool) -> list[str]:
    if not require_approved:
        return []
    checklist = DESIGN_ROOT / "signoff-checklist.md"
    text = checklist.read_text(encoding="utf-8") if checklist.exists() else ""
    if "Implementation Gate: `APPROVED`" not in text:
        return [
            "Design signoff is not approved. Expected 'Implementation Gate: `APPROVED`' in docs/design-system/signoff-checklist.md"
        ]
    return []


def main() -> int:
    args = parse_args()
    errors: list[str] = []

    errors.extend(check_required_files())
    errors.extend(check_signoff(args.require_approved))

    pairs, gather_errors = gather_pairs()
    errors.extend(gather_errors)
    for instance, schema in pairs:
        errors.extend(validate_file(instance, schema))

    scenario_map, feature_errors = parse_features()
    errors.extend(feature_errors)

    contract_ids, contract_scenarios, contract_errors = collect_contract_ids()
    errors.extend(contract_errors)

    flow_ids, flow_scenarios, flow_errors = collect_flow_ids()
    errors.extend(flow_errors)

    mapping_ids, mapping_errors = collect_mapping_ids()
    errors.extend(mapping_errors)

    bundle_ids, bundle_errors = collect_bundle_ids()
    errors.extend(bundle_errors)

    errors.extend(
        check_traceability(
            scenario_map=scenario_map,
            contract_ids=contract_ids,
            flow_ids=flow_ids,
            mapping_ids=mapping_ids,
            bundle_ids=bundle_ids,
            strict=args.strict_traceability,
            contract_scenarios=contract_scenarios,
            flow_scenarios=flow_scenarios,
        )
    )

    if errors:
        print("Design-system validation failed:", file=sys.stderr)
        for e in errors:
            print(f"- {e}", file=sys.stderr)
        return 1

    print("Design-system validation passed.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
