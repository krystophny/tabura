# AGENTS

<!-- TABURA_PROTOCOL:BEGIN -->
## Tabura Codex Protocol

Use this protocol for Tabura interactive sessions in this project.

1. Apply this default instruction in all Tabura Codex prompts for this project: Prefer using git and the GitHub CLI (`gh`) for repository and GitHub-related workflow tasks.
2. Keep generated render/output artifacts under `.tabura/artifacts`; keep editable source files in the project workspace (not under `.tabura/artifacts`).
3. Use MCP server `tabura` for all canvas operations; do not rely on filesystem event logs.
4. MCP tools: `canvas_session_open`, `canvas_artifact_show`, `canvas_status`, `canvas_import_handoff`, `temp_file_create`, `temp_file_remove`.
5. Keep interaction chat-canvas-first in the web UI; do not depend on a terminal REPL.
6. Keep `.tabura/artifacts/` gitignored; do not commit files from it unless explicitly requested.

<!-- TABURA_PROTOCOL:END -->
