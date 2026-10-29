#!/usr/bin/env bash
# Download selected release assets and verify each against the release's
# SHA256SUMS manifest. Requested filenames are read from REQUESTED_FILES as
# whitespace-delimited values; duplicate filenames are ignored.

set -euo pipefail

die() {
  echo "error: $*" >&2
  exit 1
}

require_command() {
  local command
  for command; do
    command -v "${command}" >/dev/null 2>&1 ||
      die "Required command is unavailable: ${command}"
  done
}

sha256_file() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{ print $1 }'
  elif command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$1" | awk '{ print $1 }'
  else
    die "Required command is unavailable: sha256sum or shasum"
  fi
}

download_asset() {
  local asset="$1"
  local output_path="$2"

  curl "${curl_args[@]}" \
    --output "${output_path}" \
    "${github_server_url}/${action_repository}/releases/download/${release_tag}/${asset}"
}

if [[ $# -ne 4 ]]; then
  die "Usage: $0 <release-tag> <owner/repository> <github-server-url> <destination>"
fi

readonly release_tag="$1"
readonly action_repository="$2"
readonly github_server_url="${3%/}"
readonly destination="$4"
readonly requested_files="${REQUESTED_FILES:-}"
readonly github_token="${GITHUB_TOKEN:-}"

require_command awk chmod curl mktemp mkdir mv rm

if [[ -z "${requested_files}" ]]; then
  die "REQUESTED_FILES must contain at least one release asset filename."
fi

# Shell word splitting is intentional here: the action input defines assets as
# whitespace-delimited values. Disable pathname expansion so '*' remains data.
set -f
assets=()
for asset in ${requested_files}; do
  duplicate=false
  for existing_asset in "${assets[@]}"; do
    if [[ "${asset}" == "${existing_asset}" ]]; then
      duplicate=true
      break
    fi
  done
  if [[ "${duplicate}" == false ]]; then
    assets+=("${asset}")
  fi
done
set +f

if [[ ${#assets[@]} -eq 0 ]]; then
  die "REQUESTED_FILES must contain at least one release asset filename."
fi

mkdir -p "${destination}"
staging_directory="$(mktemp -d "${destination%/}/.acp-wsl-download.XXXXXX")"
cleanup() {
  rm -rf "${staging_directory}"
}
trap cleanup EXIT

curl_args=(
  --fail
  --silent
  --show-error
  --location
  --retry 3
)
if [[ -n "${github_token}" ]]; then
  curl_args+=(--header "Authorization: Bearer ${github_token}")
fi

readonly manifest_path="${staging_directory}/SHA256SUMS"
download_asset SHA256SUMS "${manifest_path}"

manifest_requested=false
for asset in "${assets[@]}"; do
  # SHA256SUMS is already downloaded for verification. It is not listed inside
  # itself, so retain it for the caller only when explicitly requested.
  if [[ "${asset}" == SHA256SUMS ]]; then
    manifest_requested=true
    continue
  fi

  # The manifest is the authority for allowed asset names and checksums. The
  # release workflow writes names as ./<asset>, hence the exact lookup below.
  expected_sha="$(awk -v path="./${asset}" '$2 == path { print $1; exit }' \
    "${manifest_path}")"
  if [[ -z "${expected_sha}" ]]; then
    die "Release ${release_tag} has no SHA256SUMS entry for ${asset}."
  fi

  downloaded_path="${staging_directory}/${asset}"
  download_asset "${asset}" "${downloaded_path}"

  actual_sha="$(sha256_file "${downloaded_path}")"
  if [[ "${actual_sha}" != "${expected_sha}" ]]; then
    die "SHA-256 verification failed for ${asset}."
  fi

  mv -f "${downloaded_path}" "${destination}/${asset}"
  if [[ "${asset}" == acp-wsl*-linux-* ]]; then
    chmod +x "${destination}/${asset}"
  fi
done

if [[ "${manifest_requested}" == true ]]; then
  mv -f "${manifest_path}" "${destination}/SHA256SUMS"
fi
