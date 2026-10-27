#!/bin/sh

set -eu

destination=${1:?"usage: install-geoip-database.sh DESTINATION"}
script_dir=$(CDPATH='' cd -- "$(dirname "$0")" && pwd)

# shellcheck disable=SC1091
. "$script_dir/../assets/geoip/dbip-city-lite.env"

cache_dir=${ALPHA_GEOIP_CACHE_DIR:-"$script_dir/../.cache/geoip"}
source_url=${ALPHA_GEOIP_SOURCE_URL:-"$ALPHA_GEOIP_DATABASE_URL"}
cache_file="$cache_dir/dbip-city-lite-$ALPHA_GEOIP_DATABASE_VERSION-$ALPHA_GEOIP_DATABASE_SHA256.mmdb.gz"
download_dir=''
temp_dir=''
cleanup() {
  if [ -n "$download_dir" ]; then rm -rf "$download_dir"; fi
  if [ -n "$temp_dir" ]; then rm -rf "$temp_dir"; fi
}
trap cleanup EXIT
trap 'exit 1' HUP INT TERM

verify_archive() {
  printf '%s  %s\n' "$ALPHA_GEOIP_DATABASE_SHA256" "$1" | sha256sum -c - >/dev/null 2>&1
}

mkdir -p "$cache_dir"
archive="$cache_file"
if [ -f "$cache_file" ] && verify_archive "$cache_file"; then
  printf 'Using verified GeoIP cache for %s\n' "$ALPHA_GEOIP_DATABASE_VERSION"
else
  if [ -f "$cache_file" ]; then
    printf 'Ignoring invalid GeoIP cache for %s\n' "$ALPHA_GEOIP_DATABASE_VERSION" >&2
  fi
  download_dir=$(mktemp -d "$cache_dir/.alpha-geoip-download.XXXXXX")
  archive="$download_dir/dbip-city-lite.mmdb.gz"
  if curl --fail --location --proto '=https' --proto-redir '=https' --tlsv1.2 \
    --retry 3 --retry-connrefused --connect-timeout 15 --max-time 300 \
    --output "$archive" "$source_url"; then
    :
  else
    status=$?
    printf 'GeoIP download failed for pinned version %s (curl exit %s). If its monthly URL returns 404, update the version and checksum or set ALPHA_GEOIP_SOURCE_URL to a mirror of the same archive.\n' "$ALPHA_GEOIP_DATABASE_VERSION" "$status" >&2
    exit "$status"
  fi
  if ! verify_archive "$archive"; then
    printf 'GeoIP archive checksum mismatch (version %s)\n' "$ALPHA_GEOIP_DATABASE_VERSION" >&2
    exit 1
  fi
fi

destination_dir=$(dirname "$destination")
mkdir -p "$destination_dir"
temp_dir=$(mktemp -d "$destination_dir/.alpha-geoip.XXXXXX")
database="$temp_dir/dbip-city-lite.mmdb"

gzip -dc "$archive" > "$database"
database_size=$(wc -c < "$database" | tr -d ' ')
if [ "$database_size" -lt 1048576 ] || [ "$database_size" -gt 167772160 ]; then
  printf 'unexpected GeoIP database size: %s bytes\n' "$database_size" >&2
  exit 1
fi

chmod 0644 "$database"
if [ -n "$download_dir" ]; then
  chmod 0644 "$archive"
  mv "$archive" "$cache_file"
fi
mv "$database" "$destination"
printf 'Verified GeoIP database %s installed at %s\n' "$ALPHA_GEOIP_DATABASE_VERSION" "$destination"
