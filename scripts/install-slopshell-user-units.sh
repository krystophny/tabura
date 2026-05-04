#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PLATFORM="$(uname -s)"
# shellcheck source=scripts/lib/llm_env.sh
source "${REPO_ROOT}/scripts/lib/llm_env.sh"

log() { printf '[slopshell-units] %s\n' "$*"; }
fail() { printf '[slopshell-units] ERROR: %s\n' "$*" >&2; exit 1; }

resolve_intent_llm_url() {
  slopshell_resolve_intent_llm_url 2>/dev/null || printf '%s' "http://127.0.0.1:8080"
}

confirm_default_yes() {
    local prompt="$1"
    if [ ! -t 0 ]; then return 0; fi
    local response
    read -r -p "$prompt [Y/n] " response
    case "$response" in
        "" | [Yy] | [Yy][Ee][Ss]) return 0 ;;
        *) return 1 ;;
    esac
}

REUSE_LLM_URL=""
CODEX_PATH=""
VOXTYPE_PATH=""
BIN_PATH=""
WEB_DATA_DIR=""

resolve_helpy_socket() {
  if [ -n "${SLOPSHELL_HELPY_SOCKET:-}" ]; then
    printf '%s' "$SLOPSHELL_HELPY_SOCKET"
    return 0
  fi
  if [ "$PLATFORM" = "Darwin" ]; then
    printf '%s' "$HOME/Library/Caches/sloppy/helpy.sock"
    return 0
  fi
  printf '%%t/sloppy/helpy.sock'
}

voxtype_supports_stt_service() {
  local help_text
  if ! command -v "$1" >/dev/null 2>&1; then
    return 1
  fi
  help_text="$("$1" --help 2>&1 || true)"
  case "$help_text" in
    *"--service"*) return 0 ;;
  esac
  return 1
}

install_sls_binary() {
  local sls_bin_dir sls_bin_path
  sls_bin_dir="${SLOPSHELL_BIN_DIR:-${HOME}/.local/bin}"
  sls_bin_path="${sls_bin_dir}/sls"
  log "Building sls terminal client -> ${sls_bin_path}"
  mkdir -p "$sls_bin_dir"
  if ! (cd "$REPO_ROOT" && go build -o "$sls_bin_path" ./cmd/sls); then
    fail "go build failed for sls"
  fi
  if ! printf ':%s:' "$PATH" | grep -Fq ":${sls_bin_dir}:"; then
    log "${sls_bin_dir} is not in PATH; add it to your shell profile to use sls"
  fi
}

install_slopshell_binary() {
  local bin_dir
  bin_dir="${SLOPSHELL_BIN_DIR:-${HOME}/.local/bin}"
  BIN_PATH="${bin_dir}/slopshell"
  log "Building slopshell binary -> ${BIN_PATH}"
  mkdir -p "$bin_dir"
  if ! (cd "$REPO_ROOT" && go build -o "$BIN_PATH" ./cmd/slopshell); then
    fail "go build failed for slopshell"
  fi
  if ! printf ':%s:' "$PATH" | grep -Fq ":${bin_dir}:"; then
    log "${bin_dir} is not in PATH; add it to your shell profile to use slopshell"
  fi
}

configure_codex_cli() {
  local fast_url agentic_url
  fast_url="${SLOPSHELL_CODEX_FAST_URL:-${REUSE_LLM_URL}/v1}"
  agentic_url="${SLOPSHELL_CODEX_LOCAL_URL:-${REUSE_LLM_URL}/v1}"

  SLOPSHELL_CODEX_FAST_URL="$fast_url" \
  SLOPSHELL_CODEX_LOCAL_URL="$agentic_url" \
  SLOPSHELL_CODEX_AGENTIC_URL="$agentic_url" \
  SLOPSHELL_CODEX_FAST_MODEL="${SLOPSHELL_CODEX_FAST_MODEL:-qwen}" \
  SLOPSHELL_CODEX_LOCAL_MODEL="${SLOPSHELL_CODEX_LOCAL_MODEL:-qwen}" \
  "$REPO_ROOT/scripts/setup-codex-mcp.sh"
}

install_hotword_assets() {
  local script_path="${REPO_ROOT}/scripts/fetch-hotword-assets.sh"
  [ -x "$script_path" ] || fail "hotword asset bootstrap missing: $script_path"
  SLOPSHELL_WEB_DATA_DIR="$WEB_DATA_DIR" "$script_path"
}

# --- Platform detection ---

case "$PLATFORM" in
  Linux)  ;;
  Darwin) ;;
  *)      fail "unsupported platform: $PLATFORM" ;;
esac

# --- Resolve runtime URLs and verify prerequisites ---

REUSE_LLM_URL="$(resolve_intent_llm_url)"
log "Intent/app assistant route -> ${REUSE_LLM_URL}"

HAVE_VOXTYPE=1
HAVE_VOXTYPE_STT_SERVICE=1

if ! command -v codex >/dev/null 2>&1; then
  if [ "$PLATFORM" = "Darwin" ]; then
    fail "codex not in PATH. Install: npm install -g @openai/codex"
  else
    fail "codex not in PATH. Install @openai/codex"
  fi
fi

if ! command -v voxtype >/dev/null 2>&1; then
  HAVE_VOXTYPE=0
  HAVE_VOXTYPE_STT_SERVICE=0
  if [ "$PLATFORM" = "Darwin" ]; then
    log "WARNING: voxtype not in PATH. Build from source: scripts/build-voxtype-macos.sh"
  else
    fail "voxtype not in PATH. Install voxtype"
  fi
elif ! voxtype_supports_stt_service voxtype; then
  HAVE_VOXTYPE_STT_SERVICE=0
  log "WARNING: installed voxtype does not expose the local STT service flags; skipping slopshell-stt.service"
fi

if [ "$PLATFORM" = "Darwin" ]; then
  command -v go >/dev/null 2>&1 || fail "go not in PATH. Install: brew install go"
fi

# --- Linux: systemd install ---

install_linux() {
  local unit_src="$REPO_ROOT/deploy/systemd/user"
  local unit_dst="$HOME/.config/systemd/user"
  local effective_llm_url="${REUSE_LLM_URL:-http://127.0.0.1:8080}"
  local helpy_socket
  local web_host="${SLOPSHELL_WEB_HOST:-127.0.0.1}"
  local -a core_units=(
    slopshell-codex-app-server.service
    slopshell-piper-tts.service
    slopshell-web.service
  )
  local -a optional_units=()
  if [ "$HAVE_VOXTYPE_STT_SERVICE" = "1" ]; then
    core_units+=(slopshell-stt.service)
  fi

  install_slopshell_binary
  install_sls_binary
  CODEX_PATH="$(command -v codex 2>/dev/null || true)"
  [ -n "$CODEX_PATH" ] || fail "codex not in PATH. Install @openai/codex"
  VOXTYPE_PATH="$(command -v voxtype 2>/dev/null || true)"
  [ -n "$VOXTYPE_PATH" ] || fail "voxtype not in PATH. Install voxtype"
  SLOPSHELL_ASSUME_YES=1 "$REPO_ROOT/scripts/setup-slopshell-piper-tts.sh"
  helpy_socket="$(resolve_helpy_socket)"
  mkdir -p "$unit_dst"
  for f in "$unit_src"/*.service; do
    local base
    base="$(basename "$f")"
    if [ "$base" = "slopshell-llm.service" ] || [ "$base" = "slopshell-codex-llm.service" ]; then
      continue
    fi
    sed -e "s|@@REPO_ROOT@@|${REPO_ROOT}|g" \
        -e "s|@@BIN_PATH@@|${BIN_PATH}|g" \
        -e "s|@@CODEX_PATH@@|${CODEX_PATH}|g" \
        -e "s|@@VOXTYPE_BIN@@|${VOXTYPE_PATH}|g" \
        -e "s|@@SLOPSHELL_WEB_HOST@@|${web_host}|g" \
        -e "s|@@SLOPSHELL_INTENT_LLM_URL@@|${effective_llm_url}|g" \
        -e "s|@@SLOPSHELL_HELPY_SOCKET@@|${helpy_socket}|g" \
        "$f" > "$unit_dst/$base"
  done
  rm -f "$unit_dst/slopshell-llm.service" "$unit_dst/slopshell-codex-llm.service"
  systemctl --user daemon-reload

  # Disable legacy units
  systemctl --user disable --now \
    slopshell.service \
    slopshell-dev-watch.path \
    slopshell-mcp.service \
    slopshell-voxtype-mcp.service \
    slopshell-ptt.service \
    tabura.service \
    tabura-web.service \
    tabura-mcp.service \
    tabura-llm.service \
    tabura-stt.service \
    tabura-piper-tts.service \
    tabura-codex-app-server.service \
    helpy-mcp.service \
    sloptools.service \
    voxtype.service \
    >/dev/null 2>&1 || true
  systemctl --user disable --now slopshell-llm.service slopshell-codex-llm.service >/dev/null 2>&1 || true
  if [ "$HAVE_VOXTYPE_STT_SERVICE" != "1" ]; then
    systemctl --user disable --now slopshell-stt.service >/dev/null 2>&1 || true
  fi

  # Enable and start all services
  local units=("${core_units[@]}" "${optional_units[@]}")

  systemctl --user enable --now "${units[@]}"
  log "Enabled: ${units[*]}"

  # Verify all core services are running. Optional helpers are best-effort.
  sleep 3
  local failed=()
  local optional_failed=()
  local unit
  for unit in "${core_units[@]}"; do
    if ! systemctl --user is-active "$unit" >/dev/null 2>&1; then
      failed+=("$unit")
    fi
  done

  for unit in "${optional_units[@]}"; do
    if ! systemctl --user is-active "$unit" >/dev/null 2>&1; then
      optional_failed+=("$unit")
    fi
  done

  if ((${#optional_failed[@]} > 0)); then
    log "Optional services inactive: ${optional_failed[*]}"
    for unit in "${optional_failed[@]}"; do
      systemctl --user status "$unit" --no-pager -n 10 2>&1 || true
    done
  fi

  if ((${#failed[@]} > 0)); then
    log "FAILED services: ${failed[*]}"
    for unit in "${failed[@]}"; do
      systemctl --user status "$unit" --no-pager -n 10 2>&1 || true
    done
    fail "Not all services started"
  fi

  log "All services running"
}

# --- macOS: launchd helpers ---

launchctl_available() {
  local probe="/tmp/slopshell-launchctl-probe.plist"
  cat > "$probe" <<'PLIST'
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key>
  <string>io.slopshell.probe</string>
  <key>ProgramArguments</key>
  <array><string>/usr/bin/true</string></array>
</dict>
</plist>
PLIST
  if launchctl load "$probe" >/dev/null 2>&1; then
    launchctl unload "$probe" >/dev/null 2>&1 || true
    rm -f "$probe"
    return 0
  fi
  rm -f "$probe"
  return 1
}

# --- macOS: launchd install ---

install_macos() {
  local plist_src="$REPO_ROOT/deploy/launchd"
  local plist_dst="$HOME/Library/LaunchAgents"
  local data_root="$HOME/Library/Application Support/slopshell"
  local piper_model_dir piper_venv_dir
  local effective_llm_url="${REUSE_LLM_URL:-http://127.0.0.1:8080}"
  local web_host="${SLOPSHELL_WEB_HOST:-127.0.0.1}"
  local helpy_socket

  [ -d "$plist_src" ] || fail "launchd templates not found: $plist_src"

  # Build Go binary for dev use
  log "Building slopshell binary"
  if ! (cd "$REPO_ROOT" && go build -o "$REPO_ROOT/slopshell" ./cmd/slopshell); then
    fail "go build failed"
  fi
  install_sls_binary

  BIN_PATH="$REPO_ROOT/slopshell"
  CODEX_PATH="$(command -v codex 2>/dev/null || true)"
  VOXTYPE_PATH="$(command -v voxtype 2>/dev/null || echo voxtype)"
  WEB_DATA_DIR="${data_root}/web-data"
  helpy_socket="$(resolve_helpy_socket)"
  piper_model_dir="${HOME}/.local/share/slopshell-piper-tts/models"
  piper_venv_dir="${HOME}/.local/share/slopshell-piper-tts/venv"

  mkdir -p "$plist_dst" "$WEB_DATA_DIR"
  install_hotword_assets
  launchctl unload "$plist_dst/io.slopshell.llm.plist" >/dev/null 2>&1 || true
  launchctl unload "$plist_dst/io.slopshell.codex-llm.plist" >/dev/null 2>&1 || true
  rm -f "$plist_dst/io.slopshell.llm.plist" "$plist_dst/io.slopshell.codex-llm.plist"
  launchctl unload "$plist_dst/io.sloptools.mcp.plist" >/dev/null 2>&1 || true
  launchctl unload "$plist_dst/io.slopshell.piper-tts.plist" >/dev/null 2>&1 || true
  launchctl unload "$plist_dst/io.slopshell.macos-tts.plist" >/dev/null 2>&1 || true
  launchctl unload "$plist_dst/io.slopshell.codex-app-server.plist" >/dev/null 2>&1 || true
  launchctl unload "$plist_dst/io.tabura.web.plist" >/dev/null 2>&1 || true
  launchctl unload "$plist_dst/io.tabura.stt.plist" >/dev/null 2>&1 || true
  launchctl unload "$plist_dst/io.tabura.llm.plist" >/dev/null 2>&1 || true
  launchctl unload "$plist_dst/io.tabura.piper-tts.plist" >/dev/null 2>&1 || true
  launchctl unload "$plist_dst/io.tabura.codex-app-server.plist" >/dev/null 2>&1 || true
  launchctl unload "$plist_dst/io.tabura.mcp.plist" >/dev/null 2>&1 || true
  rm -f "$plist_dst/io.sloptools.mcp.plist" "$plist_dst/io.slopshell.piper-tts.plist" "$plist_dst/io.slopshell.macos-tts.plist" "$plist_dst/io.slopshell.codex-app-server.plist" "$plist_dst"/io.tabura.*.plist

  # Determine which agents to install
  local agents=(codex-app-server piper-tts web)
  if [ "$HAVE_VOXTYPE_STT_SERVICE" = "1" ]; then
    agents+=(stt)
  fi

  # Install plist files (always, even if launchctl is unavailable)
  local src dst
  for name in "${agents[@]}"; do
    src="$plist_src/io.slopshell.${name}.plist"
    dst="$plist_dst/io.slopshell.${name}.plist"
    if [ ! -f "$src" ]; then
      log "WARNING: template missing: $src"
      continue
    fi
    sed \
      -e "s|@@BIN_PATH@@|${BIN_PATH}|g" \
      -e "s|@@CODEX_PATH@@|${CODEX_PATH}|g" \
      -e "s|@@PROJECT_DIR@@|${REPO_ROOT}|g" \
      -e "s|@@WEB_DATA_DIR@@|${WEB_DATA_DIR}|g" \
      -e "s|@@SLOPSHELL_WEB_HOST@@|${web_host}|g" \
      -e "s|@@VENV_DIR@@|${piper_venv_dir}|g" \
      -e "s|@@SCRIPT_DIR@@|${REPO_ROOT}/scripts|g" \
      -e "s|@@PIPER_MODEL_DIR@@|${piper_model_dir}|g" \
      -e "s|@@STT_SETUP_SCRIPT@@|${REPO_ROOT}/scripts/setup-voxtype-stt.sh|g" \
      -e "s|@@VOXTYPE_BIN@@|${VOXTYPE_PATH}|g" \
      -e "s|@@SLOPSHELL_INTENT_LLM_URL@@|${effective_llm_url}|g" \
      -e "s|@@SLOPSHELL_HELPY_SOCKET@@|${helpy_socket}|g" \
      "$src" > "$dst"
    log "Installed plist: $dst"
  done

  # Activate services
  if launchctl_available; then
    activate_launchd "${agents[@]}"
  else
    log "launchctl unavailable (SSH/tmux session); starting services directly"
    activate_direct "${agents[@]}"
  fi
}

activate_launchd() {
  local plist_dst="$HOME/Library/LaunchAgents"
  local dst
  for name in "$@"; do
    dst="$plist_dst/io.slopshell.${name}.plist"
    launchctl unload "$dst" >/dev/null 2>&1 || true
    launchctl load -w "$dst"
    log "Loaded: io.slopshell.${name}"
  done

  sleep 3
  local failed=()
  for name in "$@"; do
    if ! launchctl list "io.slopshell.${name}" >/dev/null 2>&1; then
      failed+=("io.slopshell.${name}")
    fi
  done

  if ((${#failed[@]} > 0)); then
    log "FAILED agents: ${failed[*]}"
    fail "Not all agents started"
  fi

  log "All agents running (launchd)"
}

activate_direct() {
  local pidfile="/tmp/slopshell-pids.txt"
  local web_host="${SLOPSHELL_WEB_HOST:-127.0.0.1}"
  local effective_llm_url="${REUSE_LLM_URL:-http://127.0.0.1:8080}"
  : > "$pidfile"

  for name in "$@"; do
    local logfile="/tmp/slopshell-${name}.log"
    case "$name" in
      codex-app-server)
        nohup "$CODEX_PATH" app-server --listen ws://127.0.0.1:8787 \
          >"$logfile" 2>&1 &
        ;;
      piper-tts)
        PIPER_MODEL_DIR="${HOME}/.local/share/slopshell-piper-tts/models" \
        nohup "${HOME}/.local/share/slopshell-piper-tts/venv/bin/uvicorn" piper_tts_server:app \
          --app-dir "$REPO_ROOT/scripts" --host 127.0.0.1 --port 8424 \
          >"$logfile" 2>&1 &
        ;;
      web)
        SLOPSHELL_INTENT_LLM_URL="$effective_llm_url" \
        SLOPSHELL_ASSISTANT_MODE=local \
        SLOPSHELL_INTENT_LLM_MODEL=qwen \
        SLOPSHELL_ASSISTANT_LLM_MODEL=qwen \
        SLOPSHELL_INTENT_LLM_PROFILE=qwen3.6-35b-a3b-q4 \
        SLOPSHELL_INTENT_LLM_PROFILE_OPTIONS=qwen3.6-35b-a3b-q4 \
        nohup "$BIN_PATH" server \
          --workspace-dir "$REPO_ROOT" --data-dir "$WEB_DATA_DIR" \
          --control-socket "$HOME/Library/Caches/sloppy/control.sock" \
          --web-host "$web_host" --web-port 8420 \
          --tts-url http://127.0.0.1:8424 \
          >"$logfile" 2>&1 &
        ;;
      stt)
        SLOPSHELL_STT_LANGUAGE=de,en SLOPSHELL_STT_MODEL=large-v3-turbo \
        nohup "$REPO_ROOT/scripts/setup-voxtype-stt.sh" \
          >"$logfile" 2>&1 &
        ;;
    esac
    echo "$! io.slopshell.${name}" >> "$pidfile"
    log "Started: io.slopshell.${name} (pid $!)"
  done

  sleep 3
  local failed=()
  local pid label
  while read -r pid label; do
    if ! kill -0 "$pid" 2>/dev/null; then
      failed+=("$label")
      log "FAILED: $label (pid $pid) — see /tmp/slopshell-${label#io.slopshell.}.log"
    fi
  done < "$pidfile"

  if ((${#failed[@]} > 0)); then
    fail "Not all services started"
  fi

  log "All services running (direct); PIDs in $pidfile"
  log "Stop all: awk '{print \$1}' $pidfile | xargs kill"
}

# --- Main ---

if [ "$PLATFORM" = "Darwin" ]; then
  install_macos
else
  install_linux
fi

configure_codex_cli
