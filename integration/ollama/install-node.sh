#!/usr/bin/env bash

set -euo pipefail

node_version=v24.20.0

if [[ ! -x /usr/local/bin/node ]]; then
	archive=$(mktemp)
	trap 'rm -f "$archive"' EXIT
	curl -fsSL "https://nodejs.org/dist/$node_version/node-$node_version-linux-x64.tar.xz" -o "$archive"
	tar -xJf "$archive" --strip-components=1 -C /usr/local
fi
