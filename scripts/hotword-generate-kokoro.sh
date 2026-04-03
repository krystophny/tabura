#!/usr/bin/env bash
set -euo pipefail

if [[ -z "${SLOPPAD_HOTWORD_KOKORO_COMMAND:-}" ]]; then
  echo "Kokoro generator is not installed. Set SLOPPAD_HOTWORD_KOKORO_COMMAND to an executable." >&2
  exit 1
fi

exec "${SLOPPAD_HOTWORD_KOKORO_COMMAND}" "$@"
