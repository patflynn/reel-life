# Sonarr setup

reel-life connects to Sonarr's v3 API to manage TV series. This guide covers how to get your API key, verify connectivity, and understand what reel-life can do with Sonarr.

## Step 1: Find your Sonarr API key

1. Open the Sonarr web UI (usually `http://localhost:8989`)
2. Go to **Settings** → **General**
3. Under **Security**, find the **API Key** field
4. Copy the API key

The key is a 32-character hex string like `a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4`.

## Step 2: Verify API access

Test that you can reach the Sonarr API from the machine where reel-life will run:

```bash
curl -s -H "X-Api-Key: YOUR_API_KEY" http://localhost:8989/api/v3/health | head
```

Expected output (JSON array — may be empty if there are no health issues):

```json
[]
```

Or with health issues present:

```json
[
  {
    "source": "IndexerStatusCheck",
    "type": "warning",
    "message": "Indexers unavailable due to failures: Example Indexer"
  }
]
```

If you get `connection refused`, check that:
- Sonarr is running: `systemctl status sonarr` or check Docker
- The port is correct (default is 8989)
- There's no firewall blocking the connection
- If reel-life runs on a different host, use that host's IP or hostname instead of `localhost`

If you get `401 Unauthorized`, double-check the API key.

## Step 3: Determine the base URL

The Sonarr base URL depends on where reel-life runs relative to Sonarr:

| Deployment | Sonarr URL |
|-----------|------------|
| Same host (binary or NixOS service) | `http://localhost:8989` |
| Docker (Sonarr on host) | `http://host.docker.internal:8989` |
| Docker (same Docker network) | `http://sonarr:8989` |
| Remote host | `http://sonarr.example.com:8989` |

If Sonarr is behind a reverse proxy with a URL base (e.g., `/sonarr`), include it:

```
http://localhost:8989/sonarr
```

## Step 4: Configure reel-life

### Config file

Set the base URL in `config.yaml`:

```yaml
sonarr:
  base_url: http://localhost:8989
```

The base URL can also be set via the `SONARR_URL` environment variable, which overrides the config file.

### API key

The API key is always provided via environment variable:

```bash
export SONARR_API_KEY=a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4
```

For NixOS deployments, include it in your agenix environment file. See the [setup guide](setup-guide.md#step-2-create-agenix-secrets).

## What reel-life can do with Sonarr

The Claude agent has access to these Sonarr operations:

### search_series

Search for TV series by name. Returns matching series with title, year, overview, TVDB ID, and monitoring status. Use this to find series before adding them.

### add_series

Add a TV series to Sonarr for monitoring and automatic downloading. Requires the TVDB ID (from search results), a quality profile ID, and a root folder path. The agent will typically search first, then add based on the results.

### get_queue

Show the current download queue — active and pending downloads with their status, progress, and any issues. Useful for checking what's currently downloading or stuck.

### get_history

Show recent download history: completed imports, failed downloads, and other events. Accepts a page size parameter (default 20).

### check_health

Check Sonarr system health for warnings and errors. This covers indexer connectivity, download client status, disk space, update availability, and more. This is also what the background monitor polls automatically.

### remove_failed

Remove a failed download from the queue by its queue item ID. Can optionally blocklist the release to prevent Sonarr from re-downloading the same problematic file.

## Background health monitoring

When `monitor.enabled` is `true` (the default), reel-life polls `check_health` on the configured interval and sends alerts to Google Chat when new issues appear. It tracks which issues have already been reported to avoid duplicate alerts, and logs when issues resolve.

All health alerts are grouped into a single thread in your Google Chat space to keep notifications organized.
