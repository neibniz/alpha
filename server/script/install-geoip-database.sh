#!/bin/sh

set -eu

destination=${1:?"usage: install-geoip-database.sh DESTINATION"}
script_dir=$(CDPATH='' cd -- "$(dirname "$0")" && pwd)

# shellcheck disable=SC1091
. "$script_dir/../assets/geoip/dbip-city-lite.env"

archive="$script_dir/../assets/geoip/dbip-city-lite.mmdb.gz"
if [ ! -f "$archive" ]; then
  printf 'bundled GeoIP archive is missing: %s; restore it from the repository\n' "$archive" >&2
  exit 1
fi

if ! printf '%s  %s\n' "$ALPHA_GEOIP_DATABASE_SHA256" "$archive" | sha256sum -c -; then
  printf 'bundled GeoIP archive checksum mismatch (version %s)\n' "$ALPHA_GEOIP_DATABASE_VERSION" >&2
  exit 1
fi

destination_dir=$(dirname "$destination")
mkdir -p "$destination_dir"
temp_dir=$(mktemp -d "$destination_dir/.alpha-geoip.XXXXXX")
trap 'rm -rf "$temp_dir"' EXIT
trap 'exit 1' HUP INT TERM
database="$temp_dir/dbip-city-lite.mmdb"

gzip -dc "$archive" > "$database"
database_size=$(wc -c < "$database" | tr -d ' ')
if [ "$database_size" -lt 1048576 ] || [ "$database_size" -gt 167772160 ]; then
  printf 'unexpected GeoIP database size: %s bytes\n' "$database_size" >&2
  exit 1
fi

chmod 0644 "$database"
mv "$database" "$destination"
printf 'Bundled GeoIP database %s installed at %s (offline)\n' "$ALPHA_GEOIP_DATABASE_VERSION" "$destination"
