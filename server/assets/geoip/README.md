# GeoIP Build Data

Git tracks only `dbip-city-lite.env`, containing a fixed DB-IP City Lite version,
HTTPS source URL and archive SHA-256. Never commit MMDB files or their compressed
archives. Generated data and download caches are ignored by Git and Docker's
source context. Attribution and licensing are in `../../THIRD_PARTY_NOTICES.md`.

The installer downloads, verifies and decompresses the archive during builds.
GitHub Actions caches downloads by the manifest hash; Docker uses a locked
BuildKit cache shared by both target architectures and retains its existing
layer cache. Every reused download is verified. A missing or corrupt cache is
downloaded again; invalid downloads do not replace the existing database.
Container startup never downloads database files.

## Historical Versions And Mirrors

Monthly download URLs can expire. Caches are an optimization, not a permanent
archive. If a pinned URL returns HTTP 404, either update the manifest and notice
to an available version, or provide a mirror serving the **identical archive**:

```sh
docker build --build-arg GEOIP_DATABASE_URL=https://example.com/dbip-city-lite.mmdb.gz .
```

For the standalone installer, use `ALPHA_GEOIP_SOURCE_URL`. Overrides still have
to match the pinned SHA-256. The installer never silently switches to `latest`
or disables verification when a download fails.

## Updating The Version

1. Download the chosen MMDB archive over HTTPS from the official download page:
   https://db-ip.com/db/download/ip-to-city-lite
2. Compare a decompressed temporary copy with DB-IP's published checksums.
3. Update the version, URL and archive SHA-256 in `dbip-city-lite.env`, plus
   `../../THIRD_PARTY_NOTICES.md`. Do not add the downloaded archive to Git.
4. From the `server` directory, verify installation and a real IP lookup:

   ```sh
   sh script/install-geoip-database.sh /tmp/alpha-geoip/dbip-city-lite.mmdb
   ALPHA_TEST_GEOIP_DATABASE=/tmp/alpha-geoip/dbip-city-lite.mmdb go test ./internal/svc ./script
   ```
