# AGENTS

<!-- TABULA_PROTOCOL:BEGIN -->
## Tabula Codex Protocol

Use this protocol for Tabula interactive sessions in this project.

1. Read extra instructions from `.tabula/prompt-injection.txt` and apply them.
2. Keep generated render/output artifacts under `.tabula/artifacts`; keep editable source files in the project workspace (not under `.tabula/artifacts`).
3. Use MCP server `tabula-canvas` for all canvas operations; do not rely on filesystem event logs.
4. MCP tools: `canvas_activate`, `canvas_render_text`, `canvas_render_image`, `canvas_render_pdf`, `canvas_clear`, `canvas_status`, `canvas_selection`, `canvas_history`.
5. Keep interaction terminal-first; do not replace the terminal with a custom REPL.
6. Keep `.tabula/artifacts/` gitignored; do not commit files from it unless explicitly requested.

<!-- TABULA_PROTOCOL:END -->
