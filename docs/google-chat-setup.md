# Google Chat setup

reel-life sends messages to Google Chat via incoming webhooks. This guide walks you through creating a webhook and configuring reel-life to use it.

## Step 1: Create or choose a Google Chat space

You can use an existing space or create a new one:

1. Open [Google Chat](https://chat.google.com/)
2. Click **New chat** → **Create a space**
3. Name it something like "Media Alerts" or "reel-life"
4. Choose who has access (your workspace or specific people)

## Step 2: Create an incoming webhook

1. Open your Google Chat space
2. Click the space name at the top → **Apps & integrations** (or **Manage webhooks** in older UI)
3. Click **Add webhooks**
4. Give the webhook a name: `reel-life`
5. Optionally set an avatar URL
6. Click **Save**
7. Copy the webhook URL — it looks like this:

```
https://chat.googleapis.com/v1/spaces/AAAA_BBBB/messages?key=AIza...&token=abc123...
```

Keep this URL secret. Anyone with the URL can post messages to your space.

## Step 3: Configure reel-life with the webhook URL

### Option A: Environment variable (binary or Docker)

```bash
export GOOGLE_CHAT_WEBHOOK_URL='https://chat.googleapis.com/v1/spaces/AAAA_BBBB/messages?key=AIza...&token=abc123...'
```

### Option B: Agenix secret (NixOS)

Include the URL in your `reel-life-env.age` secret file alongside your other secrets:

```
ANTHROPIC_API_KEY=sk-ant-api03-your-key
SONARR_API_KEY=your-sonarr-key
GOOGLE_CHAT_WEBHOOK_URL=https://chat.googleapis.com/v1/spaces/AAAA_BBBB/messages?key=AIza...&token=abc123...
```

See the [setup guide](setup-guide.md#step-2-create-agenix-secrets) for full agenix instructions.

## Step 4: Test the webhook

Before starting reel-life, verify the webhook works by sending a test message with curl:

```bash
curl -X POST \
  'https://chat.googleapis.com/v1/spaces/AAAA_BBBB/messages?key=AIza...&token=abc123...' \
  -H 'Content-Type: application/json; charset=UTF-8' \
  -d '{"text": "Hello from reel-life setup test!"}'
```

You should see the message appear in your Google Chat space within a few seconds. The response from the API looks like:

```json
{
  "name": "spaces/AAAA_BBBB/messages/...",
  "text": "Hello from reel-life setup test!",
  ...
}
```

If you get an error, double-check that you copied the full webhook URL including the `key` and `token` parameters.

## How reel-life uses the webhook

reel-life sends two types of messages:

**Health alerts** (from the monitor): These are posted as threaded messages with a `sonarr-health` thread key. All health alerts are grouped into the same thread in your space, keeping your main chat clean.

Example alert:
```
⚠️ Sonarr Health Alert

• [warning] IndexerStatusCheck: Indexers unavailable due to failures
• [error] DownloadClientCheck: No download client is available
```

**Agent responses** (from user requests): When the bidirectional webhook is implemented, agent responses will be sent as replies to the requesting message.

## Bidirectional webhooks (not yet implemented)

Currently, reel-life can only **send** messages to Google Chat. Receiving messages from users (so they can ask questions like "search for Breaking Bad") requires setting up a Google Chat app, which involves:

- Creating a Google Cloud project
- Enabling the Google Chat API
- Configuring an HTTP endpoint for Google Chat to call
- Publishing the app to your workspace

This is on the roadmap but not yet implemented. For now, the monitor's proactive health alerts are the primary interaction mode.

## Troubleshooting

**"401 Unauthorized" or "403 Forbidden"**: The webhook URL is invalid or expired. Create a new webhook in the space settings.

**"404 Not Found"**: The space may have been deleted, or the URL is malformed. Check that the full URL was copied correctly.

**Messages not appearing**: Check that the webhook is still listed in the space's Apps & integrations settings. Webhooks can be removed by space admins.

**Rate limits**: Google Chat webhooks have a rate limit of roughly 1 message per second per webhook. The reel-life monitor sends at most one message per health check interval (default 5 minutes), so this shouldn't be an issue.
