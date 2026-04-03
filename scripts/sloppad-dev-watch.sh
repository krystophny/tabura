#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
INTERVAL="${SLOPPAD_DEV_WATCH_INTERVAL:-0.7}"

snapshot_all() {
  (
    cd "$REPO_ROOT"
    {
      if [[ "$(uname)" == "Darwin" ]]; then
        find cmd internal deploy/launchd scripts -type f \
          \( -name '*.go' -o -name '*.js' -o -name '*.css' -o -name '*.html' -o -name '*.json' -o -name '*.plist' -o -name '*.sh' \) \
          -exec stat -f '%N	%m	%z' {} + 2>/dev/null
      else
        find cmd internal deploy/systemd/user scripts -type f \
          \( -name '*.go' -o -name '*.js' -o -name '*.css' -o -name '*.html' -o -name '*.json' -o -name '*.service' -o -name '*.sh' \) \
          -printf '%p\t%T@\t%s\n' 2>/dev/null
      fi
      for f in go.mod go.sum; do
        if [[ -f "$f" ]]; then
          if [[ "$(uname)" == "Darwin" ]]; then
            stat -f '%N	%m	%z' "$f"
          else
            stat -c '%n	%Y	%s' "$f"
          fi
        fi
      done
    } | sort
  )
}

needs_ptyd_restart() {
  local changed_file
  while IFS= read -r changed_file; do
    [[ -z "$changed_file" ]] && continue
    if [[ "$changed_file" == internal/pty/* ]] || [[ "$changed_file" == internal/ptyd/* ]] || [[ "$changed_file" == deploy/systemd/user/sloppad-ptyd.service ]]; then
      return 0
    fi
  done
  return 1
}

PREV="$(mktemp)"
NEXT="$(mktemp)"
cleanup() {
  rm -f "$PREV" "$NEXT"
}
trap cleanup EXIT

snapshot_all >"$PREV"

while true; do
  sleep "$INTERVAL"
  snapshot_all >"$NEXT"
  if ! cmp -s "$PREV" "$NEXT"; then
    CHANGED_PATHS="$(
      diff --old-line-format='%L' --new-line-format='%L' --unchanged-line-format='' "$PREV" "$NEXT" \
        | awk -F '\t' 'NF { print $1 }' \
        | sort -u
    )"
    if needs_ptyd_restart <<<"$CHANGED_PATHS"; then
      "$REPO_ROOT/scripts/sloppad-dev-restart.sh" --include-ptyd
    else
      "$REPO_ROOT/scripts/sloppad-dev-restart.sh"
    fi
    cp "$NEXT" "$PREV"
  fi
done
