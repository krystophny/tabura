from __future__ import annotations

import io
import json
from pathlib import Path

from tabula.canvas_adapter import CanvasAdapter
from tabula.mcp_server import TabulaMcpServer, read_message


def _call(server: TabulaMcpServer, request: dict[str, object]) -> dict[str, object]:
    out = io.BytesIO()
    server.output_stream = out
    server.handle_message(request)
    out.seek(0)
    response = read_message(out)
    assert response is not None
    return response


def test_initialize_returns_server_capabilities(tmp_path: Path) -> None:
    adapter = CanvasAdapter(project_dir=tmp_path, events_path=tmp_path / "events.jsonl", headless=True, start_canvas=False)
    server = TabulaMcpServer(adapter, input_stream=io.BytesIO(), output_stream=io.BytesIO())

    response = _call(server, {"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": {}})
    assert response["id"] == 1
    result = response["result"]
    assert result["serverInfo"]["name"] == "tabula-canvas"
    assert "tools" in result["capabilities"]


def test_tools_list_exposes_canvas_tools(tmp_path: Path) -> None:
    adapter = CanvasAdapter(project_dir=tmp_path, events_path=tmp_path / "events.jsonl", headless=True, start_canvas=False)
    server = TabulaMcpServer(adapter, input_stream=io.BytesIO(), output_stream=io.BytesIO())

    response = _call(server, {"jsonrpc": "2.0", "id": 2, "method": "tools/list", "params": {}})
    names = [item["name"] for item in response["result"]["tools"]]
    assert "canvas_activate" in names
    assert "canvas_render_text" in names
    assert "canvas_render_image" in names
    assert "canvas_render_pdf" in names
    assert "canvas_clear" in names
    assert "canvas_status" in names


def test_tools_call_render_text_writes_event_and_returns_structured_content(tmp_path: Path) -> None:
    events = tmp_path / "events.jsonl"
    adapter = CanvasAdapter(project_dir=tmp_path, events_path=events, headless=True, start_canvas=False)
    server = TabulaMcpServer(adapter, input_stream=io.BytesIO(), output_stream=io.BytesIO())

    response = _call(
        server,
        {
            "jsonrpc": "2.0",
            "id": 3,
            "method": "tools/call",
            "params": {
                "name": "canvas_render_text",
                "arguments": {
                    "session_id": "s1",
                    "title": "draft",
                    "markdown_or_text": "hello",
                },
            },
        },
    )
    result = response["result"]
    assert result["isError"] is False
    assert result["structuredContent"]["kind"] == "text_artifact"
    assert events.exists()
    lines = [json.loads(line) for line in events.read_text(encoding="utf-8").splitlines() if line.strip()]
    assert len(lines) == 1
    assert lines[0]["kind"] == "text_artifact"


def test_tools_call_unknown_tool_returns_error_payload(tmp_path: Path) -> None:
    adapter = CanvasAdapter(project_dir=tmp_path, events_path=tmp_path / "events.jsonl", headless=True, start_canvas=False)
    server = TabulaMcpServer(adapter, input_stream=io.BytesIO(), output_stream=io.BytesIO())

    response = _call(
        server,
        {
            "jsonrpc": "2.0",
            "id": 4,
            "method": "tools/call",
            "params": {"name": "unknown_tool", "arguments": {}},
        },
    )
    assert response["result"]["isError"] is True
    assert "unknown tool" in response["result"]["structuredContent"]["error"]


def test_unknown_rpc_method_returns_jsonrpc_error(tmp_path: Path) -> None:
    adapter = CanvasAdapter(project_dir=tmp_path, events_path=tmp_path / "events.jsonl", headless=True, start_canvas=False)
    server = TabulaMcpServer(adapter, input_stream=io.BytesIO(), output_stream=io.BytesIO())

    response = _call(server, {"jsonrpc": "2.0", "id": 5, "method": "does/not/exist", "params": {}})
    assert response["id"] == 5
    assert response["error"]["code"] == -32601
