#!/usr/bin/env bash

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
