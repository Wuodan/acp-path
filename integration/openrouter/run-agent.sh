#!/usr/bin/env bash

set -euo pipefail

agent=$1
root=$(cd "$(dirname "$0")/../.." && pwd)
home=/tmp/acp-path-openrouter-home/$agent
openrouter_model=${OPENROUTER_MODEL:-openrouter/free}

export OPENROUTER_MODEL="$openrouter_model"

rm -rf "$home"
mkdir -p "$home"

case "$agent" in
	codex)
		mkdir -p "$home/.codex"
		cp "$root/testdata/openrouter/codex/config.toml" "$home/.codex/config.toml"
		export CODEX_CONFIG="{\"model\":\"$openrouter_model\"}" HOME="$home"
		exec codex-acp
		;;
	opencode)
		mkdir -p "$home/.config/opencode"
		cp "$root/testdata/openrouter/opencode/opencode.json" "$home/.config/opencode/opencode.json"
		export HOME="$home"
		exec opencode acp
		;;
	vibe)
		export PATH="$HOME/.local/bin:$PATH"
		mkdir -p "$home/.vibe"
		cp "$root/testdata/openrouter/vibe/config.toml" "$home/.vibe/config.toml"
		export HOME="$home" VIBE_ACTIVE_MODEL=openrouter-free
		export VIBE_MODELS="[{\"name\":\"$openrouter_model\",\"provider\":\"openrouter\",\"alias\":\"openrouter-free\"}]"
		exec vibe-acp
		;;
	*)
		echo "unknown agent: $agent" >&2
		exit 2
		;;
esac
