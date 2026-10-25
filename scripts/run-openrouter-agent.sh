#!/usr/bin/env bash
set -euo pipefail
bridge=$1
agent=$2
root=$(cd "$(dirname "$0")/.." && pwd)
home=/tmp/acp-path-openrouter-home/$agent
rm -rf "$home"
mkdir -p "$home"
case "$agent" in
  codex) mkdir -p "$home/.codex"; cp "$root/testdata/openrouter/codex/config.toml" "$home/.codex/config.toml"; export HOME="$home"; exec "$bridge" -- codex-acp ;;
  opencode) mkdir -p "$home/.config/opencode"; cp "$root/testdata/openrouter/opencode/opencode.json" "$home/.config/opencode/opencode.json"; export HOME="$home"; exec "$bridge" -- opencode acp ;;
  vibe) export PATH="$HOME/.local/bin:$PATH"; mkdir -p "$home/.vibe"; cp "$root/testdata/openrouter/vibe/config.toml" "$home/.vibe/config.toml"; export HOME="$home" VIBE_ACTIVE_MODEL=openrouter-free; exec "$bridge" -- vibe-acp ;;
esac
