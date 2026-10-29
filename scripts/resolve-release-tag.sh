#!/usr/bin/env bash
# Resolve an action reference to the highest published release tag on the same commit.
#
# Arguments:
#   1. Action reference: a full commit SHA or a tag name.
#   2. Action repository: OWNER/REPOSITORY.
#   3. GitHub API base URL, for example https://api.github.com.
#
# The resolved tag is written to stdout. Diagnostics are written to stderr.

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

resolve_tag_commit() {
  local tag_ref="refs/tags/$1"

  # Annotated tags have a tag-object ref and a peeled commit ref (^{}). Prefer
  # the peeled ref; lightweight tags have only the direct commit ref.
  printf '%s\n' "${tag_refs}" |
    awk -v tag_ref="${tag_ref}" '
      $2 == tag_ref { direct = $1 }
      $2 == tag_ref "^{}" { peeled = $1 }
      END {
        if (peeled != "") {
          print peeled
        } else if (direct != "") {
          print direct
        }
      }
    '
}

tags_for_commit() {
  # tag_refs is already in descending Git version order. Preserve that order
  # so callers try the newest release tag first, without a separate sort.
  printf '%s\n' "${tag_refs}" |
    awk -v sha="$1" '
      $1 == sha {
        tag = $2
        sub("^refs/tags/", "", tag)
        sub("\\^\\{\\}$", "", tag)
        if (!seen[tag]++) {
          print tag
        }
      }
    '
}

release_exists() {
  local tag="$1"
  local status

  status="$(curl "${curl_args[@]}" \
    "${github_api_url}/repos/${action_repository}/releases/tags/${tag}")" ||
    die "Unable to look up release for tag ${tag}."

  case "${status}" in
    200) return 0 ;;
    404) return 1 ;;
    *) die "GitHub API returned ${status} while looking up release for tag ${tag}." ;;
  esac
}

if [[ $# -ne 3 ]]; then
  die "Usage: $0 <action-ref> <owner/repository> <github-api-url>"
fi

readonly action_ref="$1"
readonly action_repository="$2"
readonly github_api_url="${3%/}"
readonly github_token="${GITHUB_TOKEN:-}"

require_command awk curl git

# Fetch once: both reference resolution and candidate selection use the same
# remote snapshot. Git's version sort puts the newest candidate first.
tag_refs="$(git ls-remote --tags --sort=-version:refname \
  "https://github.com/${action_repository}.git")"
readonly tag_refs
[[ -n "${tag_refs}" ]] || die "No tags found in ${action_repository}."

commit_sha="${action_ref}"
if [[ ! "${action_ref}" =~ ^[0-9a-f]{40}$ ]]; then
  commit_sha="$(resolve_tag_commit "${action_ref}")"
  [[ -n "${commit_sha}" ]] ||
    die "Action ref ${action_ref} is not a tag in ${action_repository}."
fi

candidate_tags=()
while IFS= read -r tag; do
  candidate_tags+=("${tag}")
done < <(tags_for_commit "${commit_sha}")

if [[ ${#candidate_tags[@]} -eq 0 ]]; then
  die "No tag points to action commit ${commit_sha}."
fi

curl_args=(
  --silent
  --show-error
  --location
  --retry 3
  --output /dev/null
  --write-out '%{http_code}'
  --header 'Accept: application/vnd.github+json'
)
if [[ -n "${github_token}" ]]; then
  curl_args+=(--header "Authorization: Bearer ${github_token}")
fi

for tag in "${candidate_tags[@]}"; do
  if release_exists "${tag}"; then
    printf '%s\n' "${tag}"
    exit 0
  fi
done

die "No published release matches action ref ${action_ref}."
