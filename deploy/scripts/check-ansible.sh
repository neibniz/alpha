#!/usr/bin/env sh
set -eu

repo_root=$(CDPATH='' cd -- "$(dirname "$0")/.." && pwd)
cd "$repo_root"

if ! command -v ansible-inventory >/dev/null 2>&1; then
  printf '%s\n' 'ansible-inventory is required' >&2
  exit 1
fi

if ! command -v ansible-playbook >/dev/null 2>&1; then
  printf '%s\n' 'ansible-playbook is required' >&2
  exit 1
fi

if ! command -v ansible-config >/dev/null 2>&1; then
  printf '%s\n' 'ansible-config is required' >&2
  exit 1
fi

if ! command -v ansible-lint >/dev/null 2>&1; then
  printf '%s\n' 'ansible-lint is required' >&2
  exit 1
fi

if ! ansible-config dump -t connection ssh --only-changed | grep -q 'pipelining.* = True'; then
  printf '%s\n' 'SSH pipelining must be enabled in ansible.cfg' >&2
  exit 1
fi

ansible-inventory --list >/dev/null

for playbook in playbooks/*.yml; do
  ansible-playbook --syntax-check "$playbook"
done

ansible-lint

for script in lan-docker/*.sh scripts/*.sh; do
  sh -n "$script"
done

if command -v shellcheck >/dev/null 2>&1; then
  shellcheck lan-docker/*.sh scripts/*.sh
fi

ansible-playbook tests/render-templates.yml

python -c \
  'import pathlib, sys, yaml; [yaml.safe_load(pathlib.Path(p).read_text()) for p in sys.argv[1:]]' \
  .ansible/rendered/*.yaml

if command -v docker >/dev/null 2>&1 && docker compose version >/dev/null 2>&1; then
  for compose_file in .ansible/rendered/*.compose.yml; do
    docker compose -f "$compose_file" config --quiet
  done

  docker compose \
    --env-file lan-docker/.env.example \
    -f lan-docker/docker-compose.yml \
    config \
    --quiet

  docker run --rm \
    --network none \
    -v "$repo_root/.ansible/rendered/Caddyfile:/etc/caddy/Caddyfile:ro" \
    -v "$repo_root/.ansible/rendered/sites:/etc/caddy/sites:ro" \
    caddy:2.11.4-alpine@sha256:5f5c8640aae01df9654968d946d8f1a56c497f1dd5c5cda4cf95ab7c14d58648 \
    caddy validate --config /etc/caddy/Caddyfile
fi
