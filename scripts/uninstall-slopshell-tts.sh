#!/usr/bin/env bash
set -euo pipefail

# Uninstall Chatterbox TTS server.
# Stops the service, removes systemd unit, deletes all data.

TTS_DIR="${HOME}/.local/share/slopshell-tts"
UNIT_DST="${HOME}/.config/systemd/user"

echo "=== Slopshell TTS Uninstall ==="

# Stop and disable the service
if systemctl --user is-active slopshell-tts.service >/dev/null 2>&1; then
    echo "Stopping slopshell-tts.service..."
    systemctl --user stop slopshell-tts.service
fi
if systemctl --user is-enabled slopshell-tts.service >/dev/null 2>&1; then
    echo "Disabling slopshell-tts.service..."
    systemctl --user disable slopshell-tts.service
fi

# Remove systemd unit
if [ -f "${UNIT_DST}/slopshell-tts.service" ]; then
    rm "${UNIT_DST}/slopshell-tts.service"
    systemctl --user daemon-reload
    echo "Removed systemd unit."
fi

# Remove HuggingFace cached model weights
HF_CACHE="${HOME}/.cache/huggingface/hub"
for d in "$HF_CACHE"/models--ResembleAI--chatterbox*; do
    if [ -d "$d" ]; then
        echo "Removing cached model weights: $d"
        rm -rf "$d"
    fi
done

# Remove TTS data directory (server, venv, reference voice)
if [ -d "$TTS_DIR" ]; then
    SIZE=$(du -sh "$TTS_DIR" 2>/dev/null | cut -f1)
    echo "Removing TTS directory ($SIZE): $TTS_DIR"
    rm -rf "$TTS_DIR"
else
    echo "TTS directory not found: $TTS_DIR"
fi

echo ""
echo "Slopshell TTS uninstalled."
echo "To reinstall: bash scripts/setup-slopshell-tts.sh"
