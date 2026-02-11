from __future__ import annotations

import json
from pathlib import Path

import pytest

from tabula.canvas_adapter import CanvasAdapter


def _read_lines(path: Path) -> list[dict[str, object]]:
    rows: list[dict[str, object]] = []
    for line in path.read_text(encoding="utf-8").splitlines():
        line = line.strip()
        if line:
            rows.append(json.loads(line))
    return rows


def test_given_headless_adapter_when_activate_then_status_reports_prompt_and_headless(tmp_path: Path) -> None:
    events = tmp_path / "events.jsonl"
    adapter = CanvasAdapter(project_dir=tmp_path, events_path=events, headless=True, start_canvas=False)

    activation = adapter.canvas_activate(session_id="s1", mode_hint="discussion")
    status = adapter.canvas_status(session_id="s1")

    assert activation["active"] is True
    assert activation["headless"] is True
    assert status["mode"] == "prompt"
    assert status["headless"] is True


def test_given_text_then_clear_when_rendering_then_mode_discussion_then_prompt_and_events_written(tmp_path: Path) -> None:
    events = tmp_path / "events.jsonl"
    adapter = CanvasAdapter(project_dir=tmp_path, events_path=events, headless=True, start_canvas=False)

    text = adapter.canvas_render_text(session_id="s1", title="draft", markdown_or_text="hello")
    clear = adapter.canvas_clear(session_id="s1", reason="done")

    assert text["kind"] == "text_artifact"
    assert text["mode"] == "discussion"
    assert clear["cleared"] is True
    assert clear["mode"] == "prompt"

    lines = _read_lines(events)
    assert [row["kind"] for row in lines] == ["text_artifact", "clear_canvas"]


def test_given_image_and_pdf_when_rendered_then_events_are_valid(tmp_path: Path) -> None:
    image = tmp_path / "img.png"
    image.write_bytes(b"x")
    pdf = tmp_path / "doc.pdf"
    pdf.write_bytes(b"%PDF-1.4")
    events = tmp_path / "events.jsonl"
    adapter = CanvasAdapter(project_dir=tmp_path, events_path=events, headless=True, start_canvas=False)

    img_result = adapter.canvas_render_image(session_id="s1", title="img", path=str(image))
    pdf_result = adapter.canvas_render_pdf(session_id="s1", title="doc", path=str(pdf), page=0)

    assert img_result["kind"] == "image_artifact"
    assert pdf_result["kind"] == "pdf_artifact"
    assert pdf_result["page"] == 0

    lines = _read_lines(events)
    assert [row["kind"] for row in lines] == ["image_artifact", "pdf_artifact"]


def test_given_missing_image_path_when_rendering_then_error(tmp_path: Path) -> None:
    events = tmp_path / "events.jsonl"
    adapter = CanvasAdapter(project_dir=tmp_path, events_path=events, headless=True, start_canvas=False)
    with pytest.raises(ValueError):
        adapter.canvas_render_image(session_id="s1", title="img", path=str(tmp_path / "missing.png"))
