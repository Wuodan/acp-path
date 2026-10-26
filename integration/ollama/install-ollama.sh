#!/usr/bin/env bash

set -euo pipefail

# Ollama needs zstd
apt-get install -y zstd

curl -fsSL https://ollama.com/install.sh | sh
nohup ollama serve >/tmp/ollama.log 2>&1 &

until curl --fail --silent http://127.0.0.1:11434/api/tags >/dev/null; do # DevSkim: ignore DS162092
	sleep 1
done
