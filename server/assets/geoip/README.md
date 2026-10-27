# Bundled GeoIP Snapshot

`dbip-city-lite.mmdb.gz` is the original DB-IP City Lite MMDB download for the
version recorded in `dbip-city-lite.env`. It is distributed under CC BY 4.0;
see `../../THIRD_PARTY_NOTICES.md` for attribution and licensing.

Container builds verify the archive SHA-256 and decompress it with networking
disabled. Neither image builds nor container startup fetch the database from
DB-IP. Keep this archive in Git, not Git LFS: a source checkout must contain the
real archive without requiring a separate LFS download.

The source URL in the manifest is provenance only. Monthly download URLs can
expire and must not be used as the sole source for rebuilding a release.

## Updating The Snapshot

1. Download the chosen MMDB archive over HTTPS from the official page:
   https://db-ip.com/db/download/ip-to-city-lite
2. Decompress a temporary copy and compare it with the checksums published by
   DB-IP. For the October 2026 MMDB, the uncompressed SHA-1 is
   `df17bd24390108ed1d24dcb7718dc60f6b9abd4f` and MD5 is
   `5966907981d3e82f338e823d1b83e002`.
3. Replace `dbip-city-lite.mmdb.gz` with the unmodified archive. Record its
   SHA-256, source URL and version in `dbip-city-lite.env`, and update the notice.
4. Run the installer and test a real database lookup before committing:

   ```sh
   sh script/install-geoip-database.sh /tmp/alpha-geoip/dbip-city-lite.mmdb
   ALPHA_TEST_GEOIP_DATABASE=/tmp/alpha-geoip/dbip-city-lite.mmdb go test ./internal/svc ./script
   ```

Run these commands from the `server` directory. The archive is about 57 MiB;
avoid unnecessary snapshot churn because each update adds to Git history.
