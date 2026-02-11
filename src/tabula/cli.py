from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path

from .events import EventValidationError, event_schema, parse_event_line
from .mcp_server import run_mcp_stdio_server
from .protocol import bootstrap_project


def _build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(prog="tabula")
    sub = parser.add_subparsers(dest="command", required=True)

    p_canvas = sub.add_parser("canvas", help="launch canvas window")
    p_canvas.add_argument("--events", type=Path, default=Path(".tabula/canvas-events.jsonl"))
    p_canvas.add_argument("--poll-ms", type=int, default=250)

    p_check = sub.add_parser("check-events", help="validate JSONL event file")
    p_check.add_argument("--events", type=Path, required=True)

    sub.add_parser("schema", help="print JSON schema")

    p_bootstrap = sub.add_parser("bootstrap", help="initialize tabula protocol files")
    p_bootstrap.add_argument("--project-dir", type=Path, default=Path("."))

    p_mcp = sub.add_parser("mcp-server", help="run tabula-canvas MCP server over stdio")
    p_mcp.add_argument("--project-dir", type=Path, default=Path("."))
    p_mcp.add_argument("--events", type=Path, default=None)
    p_mcp.add_argument("--headless", action="store_true")
    p_mcp.add_argument("--no-canvas", action="store_true")
    p_mcp.add_argument("--poll-ms", type=int, default=250)
    return parser


def _cmd_canvas(events: Path, poll_ms: int) -> int:
    try:
        from .window import run_canvas
    except ModuleNotFoundError:
        print(
            "PySide6 is required for 'tabula canvas'. Install with: python -m pip install -e .[gui]",
            file=sys.stderr,
        )
        return 2
    return run_canvas(events, poll_interval_ms=poll_ms)


def _cmd_check_events(events: Path) -> int:
    if not events.exists():
        print(f"event file does not exist: {events}", file=sys.stderr)
        return 1

    errors: list[str] = []
    for line_no, raw in enumerate(events.read_text(encoding="utf-8").splitlines(), start=1):
        if not raw.strip():
            continue
        try:
            parse_event_line(raw, line_no=line_no, base_dir=events.parent)
        except EventValidationError as exc:
            errors.append(str(exc))

    if errors:
        print("event validation failed:", file=sys.stderr)
        for err in errors:
            print(f"- {err}", file=sys.stderr)
        return 1

    print("event validation passed")
    return 0


def _cmd_schema() -> int:
    print(json.dumps(event_schema(), indent=2, sort_keys=True))
    return 0


def _cmd_bootstrap(project_dir: Path) -> int:
    try:
        result = bootstrap_project(project_dir)
    except RuntimeError as exc:
        print(str(exc), file=sys.stderr)
        return 1

    print(f"project prepared: {result.paths.project_dir}")
    print(f"agents protocol: {result.paths.agents_path}")
    print(f"mcp config snippet: {result.paths.mcp_config_path}")
    if result.git_initialized:
        print("git initialized")
    return 0


def _cmd_mcp_server(project_dir: Path, events: Path | None, headless: bool, no_canvas: bool, poll_ms: int) -> int:
    try:
        bootstrap = bootstrap_project(project_dir)
    except RuntimeError as exc:
        print(str(exc), file=sys.stderr)
        return 1

    if events is None:
        events_path = bootstrap.paths.events_path
    else:
        events_path = events if events.is_absolute() else bootstrap.paths.project_dir / events
        events_path = events_path.resolve()

    return run_mcp_stdio_server(
        project_dir=bootstrap.paths.project_dir,
        events_path=events_path,
        headless=headless,
        poll_interval_ms=poll_ms,
        start_canvas=not no_canvas,
    )


def main(argv: list[str] | None = None) -> int:
    parser = _build_parser()
    raw_argv = list(sys.argv[1:] if argv is None else argv)
    if not raw_argv:
        parser.print_help(sys.stderr)
        return 2

    args = parser.parse_args(raw_argv)
    if args.command == "canvas":
        return _cmd_canvas(args.events, args.poll_ms)
    if args.command == "check-events":
        return _cmd_check_events(args.events)
    if args.command == "schema":
        return _cmd_schema()
    if args.command == "bootstrap":
        return _cmd_bootstrap(args.project_dir)
    if args.command == "mcp-server":
        return _cmd_mcp_server(args.project_dir, args.events, args.headless, args.no_canvas, args.poll_ms)

    parser.error("unknown command")
    return 2


if __name__ == "__main__":
    raise SystemExit(main())
