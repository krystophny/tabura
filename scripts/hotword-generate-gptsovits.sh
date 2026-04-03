#!/usr/bin/env bash
set -euo pipefail

if [[ -z "${SLOPSHELL_HOTWORD_GPTSOVITS_COMMAND:-}" ]]; then
  echo "GPT-SoVITS generator is not installed. Set SLOPSHELL_HOTWORD_GPTSOVITS_COMMAND to an executable." >&2
  exit 1
fi

exec "${SLOPSHELL_HOTWORD_GPTSOVITS_COMMAND}" "$@"

