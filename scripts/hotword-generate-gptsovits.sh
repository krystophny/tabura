#!/usr/bin/env bash
set -euo pipefail

if [[ -z "${SLOPPAD_HOTWORD_GPTSOVITS_COMMAND:-}" ]]; then
  echo "GPT-SoVITS generator is not installed. Set SLOPPAD_HOTWORD_GPTSOVITS_COMMAND to an executable." >&2
  exit 1
fi

exec "${SLOPPAD_HOTWORD_GPTSOVITS_COMMAND}" "$@"

