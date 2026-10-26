#!/usr/bin/env bash

set -euo pipefail

agent=$1
root=$(cd "$(dirname "$0")/../.." && pwd)
home=/tmp/acp-path-ollama-home/$agent
ollama_model=${OLLAMA_MODEL:-qwen3:8b}

rm -rf "$home"
mkdir -p "$home"

case "$agent" in
	codex)
		mkdir -p "$home/.codex"
		cp "$root/testdata/ollama/codex/config.toml" "$home/.codex/config.toml"
		export CODEX_CONFIG="{\"model\":\"$ollama_model\"}" HOME="$home"
		exec codex-acp
		;;
	opencode)
		mkdir -p "$home/.config/opencode"
		cp "$root/testdata/ollama/opencode/opencode.json" "$home/.config/opencode/opencode.json"
		export HOME="$home"
		exec opencode acp
		;;
	vibe)
		export PATH="$HOME/.local/bin:$PATH"
		mkdir -p "$home/.vibe"
		cp "$root/testdata/ollama/vibe/config.toml" "$home/.vibe/config.toml"
		export HOME="$home" OLLAMA_API_KEY=ollama VIBE_ACTIVE_MODEL=ollama-local
		export VIBE_MODELS="[{\"name\":\"$ollama_model\",\"provider\":\"ollama\",\"alias\":\"ollama-local\"}]"
		exec vibe-acp
		;;
	*)
		echo "unknown agent: $agent" >&2
		exit 2
		;;
esac
