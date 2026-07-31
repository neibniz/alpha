#!/usr/bin/env bash
set -euo pipefail

rm -rf release
mkdir -p release

bun --filter @workspace/ui check
bun --filter @workspace/ui test
bun --filter alpha-admin-web check
bun --filter alpha-admin-web test
bun --filter alpha-admin-web build

admin_bundle_budget=$((800 * 1024))
admin_monaco_bundle_budget=$((4 * 1024 * 1024))
admin_monaco_gzip_budget=$((1024 * 1024))
while IFS= read -r -d '' asset; do
  asset_size=$(wc -c < "${asset}" | tr -d ' ')
  asset_budget=${admin_bundle_budget}
  case $(basename "${asset}") in
    monaco-runtime-*.js)
      asset_budget=${admin_monaco_bundle_budget}
      compressed_size=$(gzip -c "${asset}" | wc -c | tr -d ' ')
      if (( compressed_size > admin_monaco_gzip_budget )); then
        printf 'Lazy Monaco bundle exceeds 1 MiB gzip budget: %s (%s bytes)\n' \
          "${asset}" "${compressed_size}" >&2
        exit 1
      fi
      ;;
  esac
  if (( asset_size > asset_budget )); then
    printf 'Admin bundle exceeds its size budget: %s (%s bytes)\n' \
      "${asset}" "${asset_size}" >&2
    exit 1
  fi
done < <(find apps/admin/dist/static -type f -name '*.js' -print0)

bun --filter alpha-user-next check
bun --filter alpha-user-next lint
bun --filter alpha-user-next test
bun --filter alpha-user-next build

create_archive() {
  local archive=$1
  local source=$2
  if tar --version 2>/dev/null | grep -qi bsdtar; then
    COPYFILE_DISABLE=1 tar --format gnutar -czf "$archive" -C "$source" .
  else
    tar --format=gnu -czf "$archive" -C "$source" .
  fi
}

create_archive release/alpha-admin-web.tar.gz apps/admin/dist

mkdir -p release/user-next/apps/user-next/.next
cp -R apps/user-next/.next/standalone/. release/user-next/
cp -R apps/user-next/.next/static release/user-next/apps/user-next/.next/static
if [[ -d apps/user-next/public ]]; then
  cp -R apps/user-next/public release/user-next/apps/user-next/public
fi
mkdir -p release/user-next/apps/user-next/.next/cache
create_archive release/alpha-user-next.tar.gz release/user-next

for package in release/*.tar.gz; do
  package_name=$(basename "${package}")
  if command -v sha256sum >/dev/null 2>&1; then
    (cd release && sha256sum "${package_name}") > "${package}.sha256"
  else
    (cd release && shasum -a 256 "${package_name}") > "${package}.sha256"
  fi
done
