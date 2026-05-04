#!/usr/bin/env bash
set -euo pipefail

SCRIPT_NAME="$(basename "$0")"
SCRIPT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
if [ -f "${SCRIPT_ROOT}/lib/llm_env.sh" ]; then
    # shellcheck source=scripts/lib/llm_env.sh
    source "${SCRIPT_ROOT}/lib/llm_env.sh"
else
    slopshell_llm_env_file() {
        printf '%s' "${SLOPSHELL_LLM_ENV_FILE:-$HOME/.config/slopshell/llm.env}"
    }
    slopshell_load_llm_env() {
        local env_file
        env_file="$(slopshell_llm_env_file)"
        [ -f "$env_file" ] || return 1
        # shellcheck disable=SC1090
        set -a && . "$env_file" && set +a
    }
    slopshell_resolve_intent_llm_url() {
        local value="${SLOPSHELL_INTENT_LLM_URL:-}"
        if [ -n "$value" ]; then
            printf '%s' "$value"
            return 0
        fi
        slopshell_load_llm_env >/dev/null 2>&1 || return 1
        value="${SLOPSHELL_INTENT_LLM_URL:-}"
        [ -n "$value" ] || return 1
        printf '%s' "$value"
    }
    slopshell_resolve_openai_base_url() {
        local value="${SLOPSHELL_CODEX_BASE_URL:-}"
        if [ -n "$value" ]; then
            printf '%s' "$value"
            return 0
        fi
        if slopshell_load_llm_env >/dev/null 2>&1 && [ -n "${SLOPSHELL_CODEX_BASE_URL:-}" ]; then
            printf '%s' "${SLOPSHELL_CODEX_BASE_URL}"
            return 0
        fi
        value="$(slopshell_resolve_intent_llm_url 2>/dev/null || true)"
        [ -n "$value" ] || return 1
        value="${value%/}"
        case "$value" in
            */v1) printf '%s' "$value" ;;
            *) printf '%s/v1' "$value" ;;
        esac
    }
fi
if [ -f "${SCRIPT_ROOT}/lib/python.sh" ]; then
    # shellcheck source=scripts/lib/python.sh
    source "${SCRIPT_ROOT}/lib/python.sh"
else
    slopshell_python_meets_min_version() {
        local candidate="$1"
        local min_major="$2"
        local min_minor="$3"

        [ -x "$candidate" ] || return 1

        "$candidate" - "$min_major" "$min_minor" <<'PY' >/dev/null 2>&1
import sys

min_major = int(sys.argv[1])
min_minor = int(sys.argv[2])
raise SystemExit(0 if sys.version_info >= (min_major, min_minor) else 1)
PY
    }

    slopshell_find_python3() {
        local min_major="${1:-3}"
        local min_minor="${2:-10}"
        local candidate resolved
        local -a candidates=()
        local seen=""

        if [ -n "${SLOPSHELL_PYTHON3_BIN:-}" ]; then
            candidates+=("${SLOPSHELL_PYTHON3_BIN}")
        fi
        if resolved="$(command -v python3 2>/dev/null)"; then
            candidates+=("$resolved")
        fi
        candidates+=(/opt/homebrew/bin/python3 /usr/local/bin/python3 /usr/bin/python3)

        for candidate in "${candidates[@]}"; do
            [ -n "$candidate" ] || continue
            if [ ! -x "$candidate" ] && resolved="$(command -v "$candidate" 2>/dev/null)"; then
                candidate="$resolved"
            fi
            case ":$seen:" in
                *":$candidate:"*) continue ;;
            esac
            seen="${seen:+${seen}:}${candidate}"
            if slopshell_python_meets_min_version "$candidate" "$min_major" "$min_minor"; then
                printf '%s' "$candidate"
                return 0
            fi
        done
        return 1
    }
fi
REPO_OWNER="${SLOPSHELL_REPO_OWNER:-sloppy-org}"
REPO_NAME="${SLOPSHELL_REPO_NAME:-slopshell}"
RELEASE_API_BASE="${SLOPSHELL_RELEASE_API_BASE:-https://api.github.com/repos/${REPO_OWNER}/${REPO_NAME}/releases}"
ASSUME_YES="${SLOPSHELL_ASSUME_YES:-0}"
DRY_RUN="${SLOPSHELL_INSTALL_DRY_RUN:-0}"
SKIP_BROWSER="${SLOPSHELL_INSTALL_SKIP_BROWSER:-0}"
SKIP_STT="${SLOPSHELL_INSTALL_SKIP_STT:-0}"
REQUESTED_VERSION=""
DO_UNINSTALL=0
SLOPSHELL_OS=""
SLOPSHELL_ARCH=""
DATA_ROOT=""
BIN_DIR=""
BIN_PATH=""
PROJECT_DIR=""
WEB_DATA_DIR=""
PIPER_DIR=""
MODEL_DIR=""
VENV_DIR=""
SCRIPT_DIR=""
PIPER_SERVER_SCRIPT=""
LLM_ENV_FILE=""
STT_SETUP_SCRIPT=""
CODEX_PATH=""
PYTHON3_BIN=""

log() {
    printf '[slopshell-install] %s\n' "$*"
}

fail() {
    printf '[slopshell-install] ERROR: %s\n' "$*" >&2
    exit 1
}

run_cmd() {
    if [ "$DRY_RUN" = "1" ]; then
        printf '[slopshell-install] [dry-run]'
        printf ' %q' "$@"
        printf '\n'
        return 0
    fi
    "$@"
}

confirm_default_yes() {
    local prompt="$1"
    if [ "$ASSUME_YES" = "1" ]; then
        log "SLOPSHELL_ASSUME_YES=1 accepted: ${prompt}"
        return 0
    fi
    if [ ! -t 0 ]; then
        log "non-interactive session defaults to yes: ${prompt}"
        return 0
    fi
    local response
    read -r -p "${prompt} [Y/n] " response
    case "$response" in
        "" | [Yy] | [Yy][Ee][Ss]) return 0 ;;
        *) return 1 ;;
    esac
}

have_cmd() {
    command -v "$1" >/dev/null 2>&1
}

resolve_helpy_socket() {
    if [ -n "${SLOPSHELL_HELPY_SOCKET:-}" ]; then
        printf '%s' "$SLOPSHELL_HELPY_SOCKET"
        return 0
    fi
    printf '%%t/sloppy/helpy.sock'
}

voxtype_supports_stt_service() {
    local help_text
    if ! have_cmd "$1"; then
        return 1
    fi
    help_text="$("$1" --help 2>&1 || true)"
    case "$help_text" in
        *"--service"*) return 0 ;;
    esac
    return 1
}

print_help() {
    cat <<USAGE
Usage: ${SCRIPT_NAME} [options]

Options:
  --version <vX.Y.Z>   Install a specific release tag (default: latest)
  --yes                Non-interactive mode (answer yes to prompts)
  --dry-run            Print actions without modifying the system
  --uninstall          Uninstall services and binary
  -h, --help           Show this help

Environment overrides:
  SLOPSHELL_INSTALL_DRY_RUN=1
  SLOPSHELL_INSTALL_SKIP_BROWSER=1
  SLOPSHELL_INSTALL_SKIP_STT=1
  SLOPSHELL_LLM_ENV_FILE=<path>    Override the OpenAI-compatible LLM env file
  SLOPSHELL_INTENT_LLM_URL=<url>   Configure the intent/assistant endpoint
  SLOPSHELL_CODEX_BASE_URL=<url>   Configure the Codex OpenAI-compatible base URL
  SLOPSHELL_REPO_OWNER / SLOPSHELL_REPO_NAME / SLOPSHELL_RELEASE_API_BASE
USAGE
}

parse_args() {
    while [ "$#" -gt 0 ]; do
        case "$1" in
            --version)
                [ "$#" -ge 2 ] || fail "--version requires a value"
                REQUESTED_VERSION="$2"
                shift 2
                ;;
            --yes)
                ASSUME_YES=1
                shift
                ;;
            --dry-run)
                DRY_RUN=1
                shift
                ;;
            --uninstall)
                DO_UNINSTALL=1
                shift
                ;;
            -h | --help)
                print_help
                exit 0
                ;;
            *)
                fail "unknown argument: $1"
                ;;
        esac
    done
}

normalize_version() {
    local raw="$1"
    raw="${raw#v}"
    raw="${raw#V}"
    printf 'v%s' "$raw"
}

resolve_platform() {
    local uname_s uname_m
    uname_s="$(uname -s | tr '[:upper:]' '[:lower:]')"
    uname_m="$(uname -m | tr '[:upper:]' '[:lower:]')"
    case "$uname_s" in
        linux) SLOPSHELL_OS="linux" ;;
        darwin) SLOPSHELL_OS="darwin" ;;
        *) fail "unsupported operating system: ${uname_s}" ;;
    esac
    case "$uname_m" in
        x86_64 | amd64) SLOPSHELL_ARCH="amd64" ;;
        arm64 | aarch64) SLOPSHELL_ARCH="arm64" ;;
        *) fail "unsupported architecture: ${uname_m}" ;;
    esac
}

resolve_paths() {
    local xdg_data
    BIN_DIR="${SLOPSHELL_BIN_DIR:-${HOME}/.local/bin}"
    BIN_PATH="${BIN_DIR}/slopshell"
    if [ "$SLOPSHELL_OS" = "darwin" ]; then
        DATA_ROOT="${SLOPSHELL_DATA_ROOT:-${HOME}/Library/Application Support/slopshell}"
    else
        xdg_data="${XDG_DATA_HOME:-${HOME}/.local/share}"
        DATA_ROOT="${SLOPSHELL_DATA_ROOT:-${xdg_data}/slopshell}"
    fi
    PROJECT_DIR="${SLOPSHELL_PROJECT_DIR:-${DATA_ROOT}/project}"
    WEB_DATA_DIR="${SLOPSHELL_WEB_DATA_DIR:-${DATA_ROOT}/web-data}"
    PIPER_DIR="${DATA_ROOT}/piper-tts"
    MODEL_DIR="${PIPER_DIR}/models"
    VENV_DIR="${PIPER_DIR}/venv"
    SCRIPT_DIR="${DATA_ROOT}/scripts"
    PIPER_SERVER_SCRIPT="${SCRIPT_DIR}/piper_tts_server.py"
    LLM_ENV_FILE="$(slopshell_llm_env_file)"
    STT_SETUP_SCRIPT="${SCRIPT_DIR}/setup-voxtype-stt.sh"
}

require_codex_app_server() {
    CODEX_PATH="$(command -v codex || true)"
    [ -n "$CODEX_PATH" ] || fail "codex app-server is required but codex is not in PATH"
}

require_python_310() {
    PYTHON3_BIN="$(slopshell_find_python3 3 10 || true)"
    [ -n "$PYTHON3_BIN" ] || fail "python3 3.10+ is required"
}

require_base_tools() {
    have_cmd curl || fail "curl is required"
    have_cmd tar || fail "tar is required"
    have_cmd awk || fail "awk is required"
}

install_ffmpeg_linux() {
    local -a sudo_prefix
    sudo_prefix=()
    if [ "$(id -u)" -ne 0 ]; then
        have_cmd sudo || fail "ffmpeg install needs sudo privileges"
        sudo_prefix=(sudo)
    fi
    if have_cmd apt-get; then
        run_cmd "${sudo_prefix[@]}" apt-get update
        run_cmd "${sudo_prefix[@]}" apt-get install -y ffmpeg
        return
    fi
    if have_cmd dnf; then
        run_cmd "${sudo_prefix[@]}" dnf install -y ffmpeg
        return
    fi
    if have_cmd pacman; then
        run_cmd "${sudo_prefix[@]}" pacman -Sy --noconfirm ffmpeg
        return
    fi
    if have_cmd zypper; then
        run_cmd "${sudo_prefix[@]}" zypper --non-interactive install ffmpeg
        return
    fi
    fail "no supported package manager found to install ffmpeg"
}

ensure_ffmpeg() {
    if have_cmd ffmpeg; then
        return
    fi
    if ! confirm_default_yes "ffmpeg is missing. Attempt automatic install?"; then
        fail "ffmpeg is required"
    fi
    if [ "$SLOPSHELL_OS" = "darwin" ]; then
        have_cmd brew || fail "Homebrew is required to install ffmpeg on macOS"
        run_cmd brew install ffmpeg
    else
        install_ffmpeg_linux
    fi
    if [ "$DRY_RUN" = "0" ] && ! have_cmd ffmpeg; then
        fail "ffmpeg installation did not produce ffmpeg in PATH"
    fi
}

release_api_url() {
    if [ -n "$REQUESTED_VERSION" ]; then
        printf '%s/tags/%s' "$RELEASE_API_BASE" "$(normalize_version "$REQUESTED_VERSION")"
        return
    fi
    printf '%s/latest' "$RELEASE_API_BASE"
}

default_dry_run_release_json() {
    local version tag_nov
    version="$(normalize_version "${REQUESTED_VERSION:-0.0.0-test}")"
    tag_nov="${version#v}"
    cat <<JSON
{"tag_name":"${version}","assets":[{"name":"slopshell_${tag_nov}_${SLOPSHELL_OS}_${SLOPSHELL_ARCH}.tar.gz","browser_download_url":"https://example.invalid/slopshell_${tag_nov}_${SLOPSHELL_OS}_${SLOPSHELL_ARCH}.tar.gz"},{"name":"checksums.txt","browser_download_url":"https://example.invalid/checksums.txt"}]}
JSON
}

fetch_release_json() {
    if [ -n "${SLOPSHELL_RELEASE_JSON:-}" ]; then
        printf '%s\n' "$SLOPSHELL_RELEASE_JSON"
        return
    fi
    if [ "$DRY_RUN" = "1" ]; then
        default_dry_run_release_json
        return
    fi
    curl -fsSL "$(release_api_url)"
}

release_field() {
    local field="$1"
    local payload="$2"
    SLOPSHELL_RELEASE_JSON_PAYLOAD="$payload" python3 - "$field" <<'PY'
import json
import os
import sys
field = sys.argv[1]
data = json.loads(os.environ["SLOPSHELL_RELEASE_JSON_PAYLOAD"])
if field == "tag_name":
    value = data.get("tag_name", "")
    if not value:
        raise SystemExit(1)
    print(value)
    raise SystemExit(0)
if field.startswith("asset:"):
    target = field.split(":", 1)[1]
    for asset in data.get("assets", []):
        if asset.get("name") == target:
            print(asset.get("browser_download_url", ""))
            raise SystemExit(0)
    raise SystemExit(1)
raise SystemExit(1)
PY
}

checksum_tool() {
    if have_cmd sha256sum; then
        echo "sha256sum"
        return
    fi
    if have_cmd shasum; then
        echo "shasum"
        return
    fi
    fail "sha256 tool missing (need sha256sum or shasum)"
}

file_sha256() {
    local tool="$1"
    local file="$2"
    if [ "$tool" = "sha256sum" ]; then
        sha256sum "$file" | awk '{print $1}'
        return
    fi
    shasum -a 256 "$file" | awk '{print $1}'
}

download_release_payload() {
    local release_json="$1"
    local tmpdir="$2"
    local tag requested asset_name asset_url checksums_url checksums_file archive_file expected actual tool

    tag="$(release_field tag_name "$release_json")"
    requested="${tag#v}"
    asset_name="slopshell_${requested}_${SLOPSHELL_OS}_${SLOPSHELL_ARCH}.tar.gz"
    asset_url="$(release_field "asset:${asset_name}" "$release_json")" || fail "release missing asset ${asset_name}"
    checksums_url="$(release_field 'asset:checksums.txt' "$release_json")" || fail "release missing checksums.txt"

    archive_file="${tmpdir}/${asset_name}"
    checksums_file="${tmpdir}/checksums.txt"

    if [ "$DRY_RUN" = "1" ]; then
        cat >"${tmpdir}/slopshell" <<'BIN'
#!/usr/bin/env bash
echo "slopshell dry-run binary"
BIN
        chmod +x "${tmpdir}/slopshell"
        if [ -f "scripts/piper_tts_server.py" ]; then
            cp "scripts/piper_tts_server.py" "${tmpdir}/piper_tts_server.py"
        else
            echo "# dry-run piper server" >"${tmpdir}/piper_tts_server.py"
        fi
        if [ -f "scripts/lib/llm_env.sh" ]; then
            mkdir -p "${tmpdir}/scripts/lib"
            cp "scripts/lib/llm_env.sh" "${tmpdir}/scripts/lib/llm_env.sh"
        fi
        if [ -f "scripts/setup-voxtype-stt.sh" ]; then
            cp "scripts/setup-voxtype-stt.sh" "${tmpdir}/setup-voxtype-stt.sh"
        else
            echo "#!/usr/bin/env bash" >"${tmpdir}/setup-voxtype-stt.sh"
        fi
        chmod +x "${tmpdir}/setup-voxtype-stt.sh"
        if [ -f "scripts/build-voxtype-macos.sh" ]; then
            cp "scripts/build-voxtype-macos.sh" "${tmpdir}/build-voxtype-macos.sh"
            chmod +x "${tmpdir}/build-voxtype-macos.sh"
        fi
        if [ -d "deploy/launchd" ]; then
            mkdir -p "${tmpdir}/deploy/launchd"
            cp deploy/launchd/*.plist "${tmpdir}/deploy/launchd/"
        fi
        printf '%s\n' "$tag"
        return
    fi

    curl -fsSL -o "$archive_file" "$asset_url"
    curl -fsSL -o "$checksums_file" "$checksums_url"

    expected="$(awk -v n="$asset_name" '$2 == n {print $1}' "$checksums_file")"
    [ -n "$expected" ] || fail "checksum entry not found for ${asset_name}"

    tool="$(checksum_tool)"
    actual="$(file_sha256 "$tool" "$archive_file")"
    if [ "${actual}" != "${expected}" ]; then
        fail "checksum mismatch for ${asset_name}: got ${actual}, want ${expected}"
    fi

    tar -xzf "$archive_file" -C "$tmpdir"
    [ -x "${tmpdir}/slopshell" ] || fail "slopshell binary missing in archive"
    [ -f "${tmpdir}/scripts/piper_tts_server.py" ] || fail "scripts/piper_tts_server.py missing in archive"
    cp "${tmpdir}/scripts/piper_tts_server.py" "${tmpdir}/piper_tts_server.py"
    if [ -f "${tmpdir}/scripts/setup-voxtype-stt.sh" ]; then
        cp "${tmpdir}/scripts/setup-voxtype-stt.sh" "${tmpdir}/setup-voxtype-stt.sh"
    fi
    if [ -f "${tmpdir}/scripts/build-voxtype-macos.sh" ]; then
        cp "${tmpdir}/scripts/build-voxtype-macos.sh" "${tmpdir}/build-voxtype-macos.sh"
    fi
    printf '%s\n' "$tag"
}

install_binary_payload() {
    local staging_dir="$1"
    run_cmd mkdir -p "$BIN_DIR" "$SCRIPT_DIR"
    run_cmd cp "${staging_dir}/slopshell" "$BIN_PATH"
    run_cmd chmod +x "$BIN_PATH"
    run_cmd cp "${staging_dir}/piper_tts_server.py" "$PIPER_SERVER_SCRIPT"
    if [ -f "${staging_dir}/scripts/lib/llm_env.sh" ]; then
        run_cmd mkdir -p "${SCRIPT_DIR}/lib"
        run_cmd cp "${staging_dir}/scripts/lib/llm_env.sh" "${SCRIPT_DIR}/lib/llm_env.sh"
    fi
    if ! printf ':%s:' "$PATH" | grep -Fq ":${BIN_DIR}:"; then
        log "${BIN_DIR} is not in PATH; add it in your shell profile"
    fi
}

bootstrap_project() {
    run_cmd mkdir -p "$PROJECT_DIR" "$WEB_DATA_DIR"
    if [ "$DRY_RUN" = "1" ]; then
        return
    fi
    "$BIN_PATH" bootstrap --workspace-dir "$PROJECT_DIR" >/dev/null
}

configure_codex_cli() {
    local staging_dir="${1:-}"
    local script_path=""
    local fast_url agentic_url fast_model local_model

    if [ -n "$staging_dir" ] && [ -f "${staging_dir}/scripts/setup-codex-mcp.sh" ]; then
        script_path="${staging_dir}/scripts/setup-codex-mcp.sh"
    elif [ -f "scripts/setup-codex-mcp.sh" ]; then
        script_path="scripts/setup-codex-mcp.sh"
    fi

    if [ -z "$script_path" ]; then
        log "setup-codex-mcp.sh not available; skipping Codex local provider config"
        return
    fi

    slopshell_load_llm_env >/dev/null 2>&1 || true
    fast_url="${SLOPSHELL_CODEX_FAST_URL:-}"
    if [ -z "$fast_url" ]; then
        fast_url="$(slopshell_resolve_openai_base_url 2>/dev/null || printf '%s' 'http://127.0.0.1:8080/v1')"
    fi
    agentic_url="${SLOPSHELL_CODEX_LOCAL_URL:-${SLOPSHELL_CODEX_AGENTIC_URL:-$fast_url}}"
    fast_model="${SLOPSHELL_CODEX_FAST_MODEL:-${SLOPSHELL_INTENT_LLM_MODEL:-qwen}}"
    local_model="${SLOPSHELL_CODEX_LOCAL_MODEL:-${SLOPSHELL_ASSISTANT_LLM_MODEL:-${SLOPSHELL_INTENT_LLM_MODEL:-qwen}}}"

    if [ "$DRY_RUN" = "1" ]; then
        log "[dry-run] configure Codex MCP and local model profiles via ${script_path}"
        return
    fi

    SLOPSHELL_CODEX_FAST_URL="$fast_url" \
    SLOPSHELL_CODEX_FAST_MODEL="$fast_model" \
    SLOPSHELL_CODEX_AGENTIC_URL="$agentic_url" \
    SLOPSHELL_CODEX_LOCAL_URL="$agentic_url" \
    SLOPSHELL_CODEX_LOCAL_MODEL="$local_model" \
    bash "$script_path" >/dev/null
}

install_hotword_assets() {
    local staging_dir="${1:-}"
    local script_path=""

    if [ -n "$staging_dir" ] && [ -f "${staging_dir}/scripts/fetch-hotword-assets.sh" ]; then
        script_path="${staging_dir}/scripts/fetch-hotword-assets.sh"
    elif [ -f "scripts/fetch-hotword-assets.sh" ]; then
        script_path="scripts/fetch-hotword-assets.sh"
    fi

    [ -n "$script_path" ] || fail "fetch-hotword-assets.sh not available"

    if [ "$DRY_RUN" = "1" ]; then
        log "[dry-run] install default hotword assets via ${script_path}"
        return
    fi

    SLOPSHELL_WEB_DATA_DIR="$WEB_DATA_DIR" bash "$script_path"
}

piper_notice() {
    cat <<NOTICE
=== Piper TTS (GPL, runs as HTTP sidecar) ===
Piper TTS will be installed as a local HTTP service.
License: GPL (isolated via HTTP boundary, does not affect Slopshell MIT license)
Voice models: en_GB-alan-medium (MIT-compatible)
NOTICE
}

download_model() {
    local model="$1"
    local subpath="$2"
    local note="$3"
    local hf_base onnx_file json_file
    hf_base="https://huggingface.co/rhasspy/piper-voices/resolve/main"
    onnx_file="${MODEL_DIR}/${model}.onnx"
    json_file="${MODEL_DIR}/${model}.onnx.json"

    if [ -f "$onnx_file" ] && [ -f "$json_file" ]; then
        log "voice model already present: ${model}"
        return
    fi

    log "model notice: ${model}"
    log "${note}"
    log "model card: ${hf_base}/${subpath}/MODEL_CARD"
    if ! confirm_default_yes "Download ${model}?"; then
        log "skipping model ${model}"
        return
    fi

    if [ "$DRY_RUN" = "1" ]; then
        run_cmd mkdir -p "$MODEL_DIR"
        run_cmd touch "$onnx_file" "$json_file"
        return
    fi

    curl -fsSL -o "$onnx_file" "${hf_base}/${subpath}/${model}.onnx"
    curl -fsSL -o "$json_file" "${hf_base}/${subpath}/${model}.onnx.json"
}

setup_piper_tts() {
    piper_notice
    if ! confirm_default_yes "Install Piper TTS?"; then
        log "skipping Piper TTS setup"
        return
    fi

    run_cmd mkdir -p "$MODEL_DIR"
    if [ "$DRY_RUN" = "0" ] && [ ! -x "${VENV_DIR}/bin/python" ]; then
        "$PYTHON3_BIN" -m venv "$VENV_DIR"
    fi
    if [ "$DRY_RUN" = "0" ]; then
        "${VENV_DIR}/bin/python" -m pip install --upgrade pip
        "${VENV_DIR}/bin/python" -m pip install piper-tts fastapi 'uvicorn[standard]'
    fi

    download_model "en_GB-alan-medium" "en/en_GB/alan/medium" "Model card indicates MIT-compatible terms."
    download_model "de_DE-karlsson-low" "de/de_DE/karlsson/low" "Per-model terms are documented in the model card."
}

default_openai_base_url() {
    printf '%s' "${SLOPSHELL_DEFAULT_OPENAI_BASE_URL:-http://127.0.0.1:8080/v1}"
}

default_intent_llm_url() {
    local base_url
    base_url="$(default_openai_base_url)"
    printf '%s' "${base_url%/v1}"
}

write_llm_env_file() {
    local env_dir intent_url intent_model intent_profile intent_profile_options
    local assistant_url assistant_model codex_base_url codex_fast_model codex_local_model
    env_dir="$(dirname "$LLM_ENV_FILE")"
    intent_url="${SLOPSHELL_INTENT_LLM_URL:-$(default_intent_llm_url)}"
    intent_model="${SLOPSHELL_INTENT_LLM_MODEL:-qwen}"
    intent_profile="${SLOPSHELL_INTENT_LLM_PROFILE:-default}"
    intent_profile_options="${SLOPSHELL_INTENT_LLM_PROFILE_OPTIONS:-$intent_profile}"
    assistant_url="${SLOPSHELL_ASSISTANT_LLM_URL:-$intent_url}"
    assistant_model="${SLOPSHELL_ASSISTANT_LLM_MODEL:-$intent_model}"
    codex_base_url="${SLOPSHELL_CODEX_BASE_URL:-}"
    if [ -z "$codex_base_url" ]; then
        codex_base_url="$(slopshell_resolve_openai_base_url 2>/dev/null || printf '%s' "$(default_openai_base_url)")"
    fi
    codex_fast_model="${SLOPSHELL_CODEX_FAST_MODEL:-$intent_model}"
    codex_local_model="${SLOPSHELL_CODEX_LOCAL_MODEL:-$intent_model}"

    run_cmd mkdir -p "$env_dir"
    if [ "$DRY_RUN" = "1" ]; then
        log "[dry-run] write LLM env file to ${LLM_ENV_FILE}"
        return
    fi

    cat >"$LLM_ENV_FILE" <<ENV
SLOPSHELL_INTENT_LLM_URL=${intent_url}
SLOPSHELL_INTENT_LLM_MODEL=${intent_model}
SLOPSHELL_INTENT_LLM_PROFILE=${intent_profile}
SLOPSHELL_INTENT_LLM_PROFILE_OPTIONS=${intent_profile_options}
SLOPSHELL_ASSISTANT_LLM_URL=${assistant_url}
SLOPSHELL_ASSISTANT_LLM_MODEL=${assistant_model}
SLOPSHELL_CODEX_BASE_URL=${codex_base_url}
SLOPSHELL_CODEX_FAST_MODEL=${codex_fast_model}
SLOPSHELL_CODEX_LOCAL_MODEL=${codex_local_model}
ENV

    slopshell_load_llm_env >/dev/null 2>&1 || fail "failed to reload ${LLM_ENV_FILE}"
}

setup_llm_env() {
    local should_write=0

    if [ ! -f "$LLM_ENV_FILE" ]; then
        should_write=1
    elif [ -n "${SLOPSHELL_INTENT_LLM_URL:-}" ] || [ -n "${SLOPSHELL_INTENT_LLM_MODEL:-}" ] || [ -n "${SLOPSHELL_CODEX_BASE_URL:-}" ] || [ -n "${SLOPSHELL_CODEX_FAST_MODEL:-}" ] || [ -n "${SLOPSHELL_CODEX_LOCAL_MODEL:-}" ]; then
        should_write=1
    fi

    if [ "$should_write" = "1" ]; then
        log "writing OpenAI-compatible LLM config to ${LLM_ENV_FILE}"
        write_llm_env_file
        return
    fi

    if slopshell_load_llm_env >/dev/null 2>&1; then
        log "using existing OpenAI-compatible LLM config from ${LLM_ENV_FILE}"
        return
    fi

    log "creating default OpenAI-compatible LLM config at ${LLM_ENV_FILE}"
    write_llm_env_file
}

install_voxtype_stt() {
    if [ "$SKIP_STT" = "1" ]; then
        log "skipping voxtype STT setup due to SLOPSHELL_INSTALL_SKIP_STT=1"
        return
    fi
    cat <<NOTICE
=== Voxtype STT (MIT, runs as HTTP sidecar) ===
voxtype provides local OpenAI-compatible speech-to-text on port 8427.
License: MIT (isolated sidecar process, does not affect Slopshell MIT license)
Model: large-v3-turbo (~1.5 GB download from Hugging Face via voxtype)
NOTICE
    if ! confirm_default_yes "Install voxtype STT sidecar?"; then
        log "skipping voxtype STT setup"
        return
    fi

    if voxtype_supports_stt_service voxtype; then
        log "voxtype already installed with STT service support"
    elif [ "$SLOPSHELL_OS" = "linux" ] && have_cmd pacman; then
        if confirm_default_yes "Install voxtype via AUR (voxtype-bin, fallback voxtype)?"; then
            if have_cmd paru; then
                run_cmd paru -S --noconfirm voxtype-bin || run_cmd paru -S --noconfirm voxtype
            elif have_cmd yay; then
                run_cmd yay -S --noconfirm voxtype-bin || run_cmd yay -S --noconfirm voxtype
            else
                log "no AUR helper found (paru/yay); install voxtype manually"
            fi
        fi
    elif [ "$SLOPSHELL_OS" = "darwin" ]; then
        if have_cmd brew && brew info voxtype >/dev/null 2>&1; then
            if confirm_default_yes "Install voxtype via Homebrew?"; then
                run_cmd brew install voxtype
            fi
        elif have_cmd cargo && have_cmd cmake; then
            log "No Homebrew formula for voxtype; building from source"
            if confirm_default_yes "Build voxtype from source (Rust + cmake)?"; then
                local staging_build="${1:-}"
                local build_script=""
                if [ -n "$staging_build" ] && [ -f "${staging_build}/build-voxtype-macos.sh" ]; then
                    build_script="${staging_build}/build-voxtype-macos.sh"
                elif [ -f "scripts/build-voxtype-macos.sh" ]; then
                    build_script="scripts/build-voxtype-macos.sh"
                fi
                if [ -n "$build_script" ]; then
                    run_cmd bash "$build_script" --yes
                else
                    log "build script not available; build manually:"
                    log "  git clone --branch feature/single-daemon-openai-stt-api https://github.com/peteonrails/voxtype.git"
                    log "  see: https://github.com/sloppy-org/slopshell#voxtype-stt"
                fi
            fi
        else
            log "voxtype not found; to build from source install Rust and cmake:"
            log "  brew install rust cmake"
            log "  then run: scripts/build-voxtype-macos.sh"
        fi
    else
        log "voxtype not found; install voxtype and ensure it is on PATH"
    fi

    if voxtype_supports_stt_service voxtype; then
        if confirm_default_yes "Download voxtype model large-v3-turbo (~1.5 GB)?"; then
            run_cmd voxtype setup --download --model large-v3-turbo --no-post-install
        fi
    else
        log "voxtype with STT service support was not installed; speech-to-text remains unavailable"
    fi

    local staging_stt="${1:-}"
    if [ -n "$staging_stt" ] && [ -f "${staging_stt}/setup-voxtype-stt.sh" ]; then
        run_cmd mkdir -p "$SCRIPT_DIR"
        run_cmd cp "${staging_stt}/setup-voxtype-stt.sh" "$STT_SETUP_SCRIPT"
        run_cmd chmod +x "$STT_SETUP_SCRIPT"
    fi
}

write_systemd_units() {
    local systemd_dir
    systemd_dir="${HOME}/.config/systemd/user"

    if [ "$DRY_RUN" = "1" ]; then
        log "[dry-run] write systemd units under ${systemd_dir}"
        return
    fi

    run_cmd mkdir -p "$systemd_dir"

    cat >"${systemd_dir}/slopshell-codex-app-server.service" <<UNIT
[Unit]
Description=Codex App Server (Slopshell)
After=network.target

[Service]
Type=simple
ExecStart=${CODEX_PATH} app-server --listen ws://127.0.0.1:8787
Restart=on-failure
RestartSec=2

[Install]
WantedBy=default.target
UNIT

    cat >"${systemd_dir}/slopshell-piper-tts.service" <<UNIT
[Unit]
Description=Slopshell Piper TTS
After=network.target

[Service]
Type=simple
Environment=PIPER_MODEL_DIR=${MODEL_DIR}
ExecStart=${VENV_DIR}/bin/uvicorn piper_tts_server:app --app-dir ${SCRIPT_DIR} --host 127.0.0.1 --port 8424
Restart=on-failure
RestartSec=2

[Install]
WantedBy=default.target
UNIT

    local helpy_socket
    local web_host="${SLOPSHELL_WEB_HOST:-127.0.0.1}"
    local web_control_args="--control-socket %t/sloppy/control.sock"
    helpy_socket="$(resolve_helpy_socket)"

    cat >"${systemd_dir}/slopshell-web.service" <<UNIT
[Unit]
Description=Slopshell Web UI
After=network.target slopshell-codex-app-server.service slopshell-piper-tts.service helpy-mcp.service
Wants=slopshell-codex-app-server.service slopshell-piper-tts.service helpy-mcp.service

[Service]
Type=simple
Environment=HOME=%h
Environment=SHELL=/usr/bin/bash
Environment=SLOPSHELL_HELPY_SOCKET=${helpy_socket}
Environment=SLOPSHELL_BRAIN_GTD_SYNC=on
ExecStart=/usr/bin/bash -lc 'set -a; source ~/.config/helpy/env >/dev/null 2>&1 || true; source ${LLM_ENV_FILE} >/dev/null 2>&1 || true; set +a; exec ${BIN_PATH} server --workspace-dir ${PROJECT_DIR} --data-dir ${WEB_DATA_DIR} ${web_control_args} --web-host ${web_host} --web-port 8420 --app-server-url ws://127.0.0.1:8787 --tts-url http://127.0.0.1:8424'
Restart=on-failure
RestartSec=2

[Install]
WantedBy=default.target
UNIT
}

install_services_linux() {
    local -a units
    have_cmd systemctl || fail "systemctl is required for Linux service setup"
    write_systemd_units
    run_cmd systemctl --user daemon-reload
    run_cmd systemctl --user disable --now slopshell.service >/dev/null 2>&1 || true
    run_cmd systemctl --user disable --now slopshell-llm.service slopshell-codex-llm.service >/dev/null 2>&1 || true
    units=(slopshell-codex-app-server.service slopshell-piper-tts.service slopshell-web.service)
    run_cmd systemctl --user enable --now "${units[@]}"
}

substitute_launchd_template() {
    local src="$1" dst="$2"
    local web_host="${SLOPSHELL_WEB_HOST:-127.0.0.1}"
    local voxtype_bin
    local helpy_socket
    voxtype_bin="$(command -v voxtype 2>/dev/null || echo voxtype)"
    helpy_socket="$(resolve_helpy_socket)"
    sed \
        -e "s|@@BIN_PATH@@|${BIN_PATH}|g" \
        -e "s|@@CODEX_PATH@@|${CODEX_PATH}|g" \
        -e "s|@@PROJECT_DIR@@|${PROJECT_DIR}|g" \
        -e "s|@@WEB_DATA_DIR@@|${WEB_DATA_DIR}|g" \
        -e "s|@@SLOPSHELL_WEB_HOST@@|${web_host}|g" \
        -e "s|@@SLOPSHELL_HELPY_SOCKET@@|${helpy_socket}|g" \
        -e "s|@@VENV_DIR@@|${VENV_DIR}|g" \
        -e "s|@@SCRIPT_DIR@@|${SCRIPT_DIR}|g" \
        -e "s|@@PIPER_MODEL_DIR@@|${MODEL_DIR}|g" \
        -e "s|@@STT_SETUP_SCRIPT@@|${STT_SETUP_SCRIPT}|g" \
        -e "s|@@VOXTYPE_BIN@@|${voxtype_bin}|g" \
        "$src" >"$dst"
}

write_launchd_plists() {
    local staging_dir="$1"
    local agent_dir template_dir
    agent_dir="${HOME}/Library/LaunchAgents"
    template_dir="${staging_dir}/deploy/launchd"

    if [ "$DRY_RUN" = "1" ]; then
        log "[dry-run] write launchd plists under ${agent_dir}"
        return
    fi

    run_cmd mkdir -p "$agent_dir"

    [ -d "$template_dir" ] || fail "launchd templates not found in ${template_dir}"

    substitute_launchd_template "${template_dir}/io.slopshell.codex-app-server.plist" "${agent_dir}/io.slopshell.codex-app-server.plist"
    substitute_launchd_template "${template_dir}/io.slopshell.piper-tts.plist" "${agent_dir}/io.slopshell.piper-tts.plist"
    run_cmd launchctl unload "${agent_dir}/io.slopshell.macos-tts.plist" >/dev/null 2>&1 || true
    run_cmd rm -f "${agent_dir}/io.slopshell.macos-tts.plist"

    run_cmd launchctl unload "${agent_dir}/io.slopshell.llm.plist" >/dev/null 2>&1 || true
    run_cmd rm -f "${agent_dir}/io.slopshell.llm.plist" "${agent_dir}/io.slopshell.codex-llm.plist"
    if [ -x "$STT_SETUP_SCRIPT" ]; then
        substitute_launchd_template "${template_dir}/io.slopshell.stt.plist" "${agent_dir}/io.slopshell.stt.plist"
    fi

    substitute_launchd_template "${template_dir}/io.slopshell.web.plist" "${agent_dir}/io.slopshell.web.plist"
}

load_launchd_service() {
    local plist="$1"
    run_cmd launchctl unload "$plist" >/dev/null 2>&1 || true
    run_cmd launchctl load -w "$plist"
}

install_services_macos() {
    local staging_dir="$1"
    local agent_dir
    agent_dir="${HOME}/Library/LaunchAgents"
    write_launchd_plists "$staging_dir"
    load_launchd_service "${agent_dir}/io.slopshell.codex-app-server.plist"
    load_launchd_service "${agent_dir}/io.slopshell.piper-tts.plist"
    if [ -f "${agent_dir}/io.slopshell.stt.plist" ]; then
        load_launchd_service "${agent_dir}/io.slopshell.stt.plist"
    fi
    load_launchd_service "${agent_dir}/io.slopshell.web.plist"
}

open_browser() {
    local url
    url="http://127.0.0.1:8420"
    if [ "$SKIP_BROWSER" = "1" ]; then
        log "skipping browser open due to SLOPSHELL_INSTALL_SKIP_BROWSER=1"
        return
    fi
    if [ "$DRY_RUN" = "1" ]; then
        log "[dry-run] open ${url}"
        return
    fi
    if [ "$SLOPSHELL_OS" = "darwin" ] && have_cmd open; then
        open "$url" >/dev/null 2>&1 || true
        return
    fi
    if [ "$SLOPSHELL_OS" = "linux" ] && have_cmd xdg-open; then
        xdg-open "$url" >/dev/null 2>&1 || true
        return
    fi
    log "open your browser at ${url}"
}

print_summary() {
    local version="$1"
    local tts_summary="Piper (${MODEL_DIR})"
    cat <<SUMMARY

Install complete
  Version:       ${version}
  Binary:        ${BIN_PATH}
  Data root:     ${DATA_ROOT}
  Project dir:   ${PROJECT_DIR}
  TTS backend:   ${tts_summary}
  Service mode:  ${SLOPSHELL_OS}
  Web URL:       http://127.0.0.1:8420
  LLM env file:  ${LLM_ENV_FILE}
  Intent LLM:    ${SLOPSHELL_INTENT_LLM_URL:-off}
SUMMARY
}

disable_lm_studio_login_item() {
    [ "$SLOPSHELL_OS" = "darwin" ] || return 0
    if [ "$DRY_RUN" = "1" ]; then
        log "[dry-run] remove LM Studio from login items"
        return 0
    fi
    osascript <<'APPLESCRIPT' >/dev/null 2>&1 || true
tell application "System Events"
    delete every login item whose name is "LM Studio"
end tell
APPLESCRIPT
}

remove_linux_services() {
    local systemd_dir
    systemd_dir="${HOME}/.config/systemd/user"
    if have_cmd systemctl; then
        run_cmd systemctl --user disable --now \
            slopshell.service \
            slopshell-web.service slopshell-piper-tts.service slopshell-codex-app-server.service \
            slopshell-llm.service slopshell-codex-llm.service >/dev/null 2>&1 || true
        run_cmd systemctl --user daemon-reload >/dev/null 2>&1 || true
    fi
    run_cmd rm -f \
        "${systemd_dir}/slopshell-web.service" \
        "${systemd_dir}/slopshell-piper-tts.service" \
        "${systemd_dir}/slopshell-codex-app-server.service" \
        "${systemd_dir}/slopshell-llm.service" \
        "${systemd_dir}/slopshell-codex-llm.service"
}

remove_macos_services() {
    local agent_dir plist
    agent_dir="${HOME}/Library/LaunchAgents"
    for plist in io.slopshell.web io.slopshell.stt io.slopshell.llm io.slopshell.piper-tts io.slopshell.codex-app-server; do
        run_cmd launchctl unload "${agent_dir}/${plist}.plist" >/dev/null 2>&1 || true
    done
    run_cmd rm -f \
        "${agent_dir}/io.slopshell.web.plist" \
        "${agent_dir}/io.slopshell.stt.plist" \
        "${agent_dir}/io.slopshell.piper-tts.plist" \
        "${agent_dir}/io.slopshell.codex-app-server.plist" \
        "${agent_dir}/io.slopshell.llm.plist" \
        "${agent_dir}/io.slopshell.codex-llm.plist"
}

uninstall_flow() {
    resolve_platform
    resolve_paths
    log "starting uninstall"
    if [ "$SLOPSHELL_OS" = "darwin" ]; then
        remove_macos_services
    else
        remove_linux_services
    fi
    run_cmd rm -f "$BIN_PATH"
    if confirm_default_yes "Remove ${DATA_ROOT} data directory?"; then
        run_cmd rm -rf "$DATA_ROOT"
    fi
    log "uninstall complete"
}

install_flow() {
    local release_json tmpdir installed_tag
    resolve_platform
    resolve_paths
    require_base_tools
    require_codex_app_server
    require_python_310
    ensure_ffmpeg

    tmpdir="$(mktemp -d -t slopshell-install-XXXXXX)"
    trap "rm -rf '$tmpdir'" EXIT

    release_json="$(fetch_release_json)"
    installed_tag="$(download_release_payload "$release_json" "$tmpdir")"
    install_binary_payload "$tmpdir"
    bootstrap_project
    disable_lm_studio_login_item
    setup_piper_tts
    setup_llm_env
    install_voxtype_stt "$tmpdir"
    install_hotword_assets "$tmpdir"
    if [ "$SLOPSHELL_OS" = "darwin" ]; then
        install_services_macos "$tmpdir"
    else
        install_services_linux
    fi
    configure_codex_cli "$tmpdir"
    open_browser
    print_summary "$installed_tag"
}

main() {
    parse_args "$@"
    if [ "$DO_UNINSTALL" = "1" ]; then
        uninstall_flow
        exit 0
    fi
    install_flow
}

main "$@"
