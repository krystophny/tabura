#!/usr/bin/env bash
set -euo pipefail

if [[ -z "${SLOPSHELL_HOTWORD_KOKORO_COMMAND:-}" ]]; then
  echo "Kokoro generator is not installed. Set SLOPSHELL_HOTWORD_KOKORO_COMMAND to an executable." >&2
  exit 1
fi

exec "${SLOPSHELL_HOTWORD_KOKORO_COMMAND}" "$@"
