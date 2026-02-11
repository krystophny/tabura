#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PYTHON_BIN="python3"

ensure_jsonschema() {
  if "${PYTHON_BIN}" - <<'PY' >/dev/null 2>&1
import importlib.util
import sys
sys.exit(0 if importlib.util.find_spec("jsonschema") else 1)
PY
  then
    return 0
  fi

  local venv_dir="${ROOT_DIR}/.cache/design-system-validator/.venv"
  if [[ ! -x "${venv_dir}/bin/python" ]]; then
    python3 -m venv "${venv_dir}"
  fi

  "${venv_dir}/bin/python" -m pip install --quiet --upgrade pip
  "${venv_dir}/bin/python" -m pip install --quiet "jsonschema==4.23.0"
  PYTHON_BIN="${venv_dir}/bin/python"
}

ensure_jsonschema
exec "${PYTHON_BIN}" "${ROOT_DIR}/scripts/validate-design-system.py" "$@"
