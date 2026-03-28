# Troubleshooting

Common issues and how to fix them.

## Service won't start

### Missing environment variables

reel-life requires three environment variables. If any are missing, it will exit on startup.

**Symptoms:**

```
level=ERROR msg="ANTHROPIC_API_KEY environment variable is required"
```

Or a validation error about missing `sonarr.api_key` or `chat.webhook_url`.

**Fix:** Make sure all three variables are set:

```bash
export ANTHROPIC_API_KEY=your-key
export SONARR_API_KEY=your-key
export GOOGLE_CHAT_WEBHOOK_URL=your-webhook-url
```

For NixOS deployments, check that your `environmentFiles` path is correct and the agenix secret has been decrypted:

```bash
# Check the secret file exists and is readable by the service
sudo ls -la /run/agenix/reel-life-env

# Check the service's environment
sudo systemctl show reel-life | grep EnvironmentFile
```

### Config file not found

**Symptoms:**

```
level=ERROR msg="failed to load config" error="open config.yaml: no such file or directory"
```

**Fix:** Either create `config.yaml` from the example or specify the path:

```bash
./reel-life -config /path/to/config.yaml
```

For Docker, make sure the volume mount is correct:

```bash
docker run -v $(pwd)/config.yaml:/config.yaml:ro reel-life -config /config.yaml
```

The NixOS module generates the config file automatically — you shouldn't see this error with the NixOS deployment.

---

## Connection refused to Sonarr

**Symptoms:**

```
level=ERROR msg="monitor: health check failed" error="Get \"http://localhost:8989/api/v3/health\": dial tcp 127.0.0.1:8989: connect: connection refused"
```

**Possible causes:**

1. **Sonarr isn't running**: Check with `systemctl status sonarr` or `docker ps | grep sonarr`
2. **Wrong port**: Sonarr's default is 8989, but it can be configured differently. Check Sonarr's config.
3. **Wrong host**: If reel-life runs in Docker and Sonarr is on the host, use `host.docker.internal` instead of `localhost`. If both are in Docker, use the container name.
4. **Firewall**: On the same host this is unlikely, but for remote Sonarr instances check that the port is open.

**Debugging:**

```bash
# From the machine running reel-life:
curl -s http://localhost:8989/api/v3/system/status -H "X-Api-Key: YOUR_KEY"

# If that fails, check if the port is listening:
ss -tlnp | grep 8989
```

---

## Authentication failures

**Symptoms:**

```
level=ERROR msg="sonarr API error" status=401
```

**Fix:** The Sonarr API key is wrong. To verify:

```bash
curl -s -o /dev/null -w "%{http_code}" \
  -H "X-Api-Key: YOUR_KEY" \
  http://localhost:8989/api/v3/health
```

- `200` = key is correct
- `401` = key is wrong

Get the correct key from Sonarr UI: **Settings** → **General** → **API Key**.

For NixOS with agenix, check that the decrypted secret contains the correct value:

```bash
# This shows the decrypted content (be careful with secrets in terminal history)
sudo cat /run/agenix/reel-life-env | grep SONARR_API_KEY
```

---

## Google Chat webhook errors

### 401/403 errors

The webhook URL is invalid or has been revoked.

**Fix:** Create a new webhook in the Google Chat space settings. See [Google Chat setup](google-chat-setup.md#step-2-create-an-incoming-webhook).

### 404 errors

The space may have been deleted or the URL is malformed.

**Fix:** Verify the URL is complete — it should include both `key=` and `token=` query parameters:

```
https://chat.googleapis.com/v1/spaces/SPACE_ID/messages?key=AIza...&token=abc...
```

### Messages not appearing

Test the webhook directly:

```bash
curl -X POST "$GOOGLE_CHAT_WEBHOOK_URL" \
  -H 'Content-Type: application/json; charset=UTF-8' \
  -d '{"text": "test message"}'
```

If the curl succeeds but reel-life messages don't appear, check that:
- The `GOOGLE_CHAT_WEBHOOK_URL` environment variable is set correctly for the reel-life process
- There are no extra quotes or whitespace in the URL

---

## Health monitor not sending alerts

**Possible causes:**

1. **Monitor is disabled**: Check your config has `monitor.enabled: true`
2. **No health issues**: The monitor only sends alerts when Sonarr reports problems. If Sonarr is healthy, there's nothing to alert about.
3. **Issues already reported**: The monitor deduplicates — it only alerts on *new* issues. If the issue was already reported and hasn't resolved, it won't send again. Restart reel-life to reset the deduplication state.
4. **Webhook errors**: Check the logs for webhook-related errors (see above).

**Debugging:**

```bash
# Watch the monitor logs
journalctl -u reel-life -f | grep monitor

# You should see periodic messages like:
# level=INFO msg="monitor: checking health"
# level=INFO msg="monitor: 0 new issues"
```

If you don't see any monitor log lines, the monitor may not be starting. Check for startup errors.

---

## Reading the logs

reel-life uses structured logging via Go's `slog` package.

### Log levels

| Level | What it shows |
|-------|--------------|
| `debug` | Detailed internal state, API request/response details |
| `info` | Normal operations: startup, health checks, messages sent |
| `warn` | Recoverable issues: transient API errors, unexpected responses |
| `error` | Failures: can't reach Sonarr, webhook errors, config problems |

Set the level in config:

```yaml
log:
  level: debug    # show everything while debugging
```

Or for the NixOS module:

```nix
services.reel-life.logLevel = "debug";
```

### Log format

Two formats are available:

**text** (default) — human-readable, good for `journalctl`:
```
level=INFO msg="monitor: checking health"
level=INFO msg="monitor: 0 new issues"
```

**json** — machine-parseable, good for log aggregation:
```json
{"level":"INFO","msg":"monitor: checking health"}
```

### Viewing logs

```bash
# NixOS / systemd
journalctl -u reel-life -f              # follow live
journalctl -u reel-life --since "1h ago" # last hour
journalctl -u reel-life -p err          # errors only

# Docker
docker logs -f reel-life
docker logs --since 1h reel-life

# Binary
# Logs go to stderr by default
./reel-life -config config.yaml 2>&1 | tee reel-life.log
```

---

## Claude API errors

### 401 from Anthropic

The `ANTHROPIC_API_KEY` is invalid or expired.

**Fix:** Check your key at [console.anthropic.com](https://console.anthropic.com/). Generate a new one if needed.

### Rate limiting

If you see `429` errors, you're hitting Anthropic's rate limits. This is unlikely with normal usage but could happen if many users are sending requests simultaneously.

**Fix:** The agent will retry automatically on transient errors. If it persists, check your Anthropic plan's rate limits.

### Model not available

If you configured a model that doesn't exist or isn't available on your plan:

```
level=ERROR msg="agent error" error="model not found: ..."
```

**Fix:** Use a valid model name. The default `claude-sonnet-4-5-20250929` should work for all plans. Check available models in the [Anthropic docs](https://docs.anthropic.com/en/docs/about-claude/models).
