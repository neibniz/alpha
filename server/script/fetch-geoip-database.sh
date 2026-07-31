#!/bin/sh

set -eu

destination=${1:?"usage: fetch-geoip-database.sh DESTINATION"}
script_dir=$(CDPATH= cd -- "$(dirname "$0")" && pwd)

# shellcheck disable=SC1091
. "$script_dir/../assets/geoip/dbip-city-lite.env"

destination_dir=$(dirname "$destination")
mkdir -p "$destination_dir"
temp_dir=$(mktemp -d "$destination_dir/.alpha-geoip.XXXXXX")
trap 'rm -rf "$temp_dir"' EXIT HUP INT TERM

archive="$temp_dir/dbip-city-lite.mmdb.gz"
database="$temp_dir/dbip-city-lite.mmdb"

curl \
  --fail \
  --location \
  --proto '=https' \
  --tlsv1.2 \
  --retry 5 \
  --retry-all-errors \
  --connect-timeout 15 \
  --max-time 300 \
  --output "$archive" \
  "$ALPHA_GEOIP_DATABASE_URL"

printf '%s  %s\n' "$ALPHA_GEOIP_DATABASE_SHA256" "$archive" | sha256sum -c -
gzip -dc "$archive" > "$database"

database_size=$(wc -c < "$database" | tr -d ' ')
if [ "$database_size" -lt 1048576 ] || [ "$database_size" -gt 167772160 ]; then
  printf 'unexpected GeoIP database size: %s bytes\n' "$database_size" >&2
  exit 1
fi

chmod 0644 "$database"
mv "$database" "$destination"
printf 'GeoIP database %s installed at %s\n' "$ALPHA_GEOIP_DATABASE_VERSION" "$destination"
