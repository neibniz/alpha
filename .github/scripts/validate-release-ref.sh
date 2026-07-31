#!/usr/bin/env bash

set -euo pipefail

component=${RELEASE_COMPONENT:?"RELEASE_COMPONENT is required"}
default_branch=${DEFAULT_BRANCH:-main}
repository=${GITHUB_REPOSITORY:?"GITHUB_REPOSITORY is required"}
tag=${GITHUB_REF_NAME:-}

case "${component}" in
  server | frontend | xray-agent | forwared) ;;
  *)
    printf 'unsupported release component: %s\n' "${component}" >&2
    exit 1
    ;;
esac

if [[ "${GITHUB_EVENT_NAME:-}" != "push" || "${GITHUB_REF_TYPE:-}" != "tag" ]]; then
  printf 'release validation requires a tag push event\n' >&2
  exit 1
fi

release_pattern="^${component}/v(0|[1-9][0-9]*)\\.(0|[1-9][0-9]*)\\.(0|[1-9][0-9]*)$"
if [[ ! "${tag}" =~ ${release_pattern} ]]; then
  printf 'invalid release tag %q; expected %s/vX.Y.Z\n' "${tag}" "${component}" >&2
  exit 1
fi

tag_commit=$(git rev-parse 'HEAD^{commit}')
live_tag_commit=$(
  gh api \
    --method GET \
    "repos/${repository}/commits" \
    -f sha="${tag}" \
    -f per_page=1 \
    --jq '.[0].sha'
)
if [[ "${live_tag_commit}" != "${tag_commit}" ]]; then
  printf 'release tag %s moved from checked-out commit %s to %s\n' \
    "${tag}" "${tag_commit}" "${live_tag_commit}" >&2
  exit 1
fi

comparison=$(
  gh api \
    "repos/${repository}/compare/${tag_commit}...${default_branch}" \
    --jq '.status'
)

case "${comparison}" in
  identical | ahead) ;;
  *)
    printf 'release tag %s points to %s, which is not contained in %s (%s)\n' \
      "${tag}" "${tag_commit}" "${default_branch}" "${comparison}" >&2
    exit 1
    ;;
esac

release_state=$(
  gh release view "${tag}" \
    --repo "${repository}" \
    --json isDraft \
    --jq '.isDraft' 2>/dev/null || true
)
if [[ "${release_state}" == "false" ]]; then
  printf 'release %s is already published; refusing to rebuild immutable artifacts\n' \
    "${tag}" >&2
  exit 1
fi
