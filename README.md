# reel-life

An AI-powered chatops agent for media curation and quality control. It connects to media management APIs (Sonarr, Radarr, Prowlarr, Overseerr) and provides both reactive and proactive media management through Google Chat.

## How it works

reel-life uses Claude as an AI agent with a constrained set of tools — it can only interact with your media management APIs, nothing else. When a user sends a message, Claude reasons about what to do and calls the appropriate media API operations.

A background monitor loop periodically polls your media services for health issues (failed downloads, indexer problems, disk space warnings) and sends alerts to your chat space.

## Features

- **Reactive**: Respond to natural language requests — "search for Breaking Bad", "what's downloading?", "check system health"
- **Proactive**: Automatic alerts for health issues, failed downloads, and import errors
- **Constrained**: The AI agent can ONLY call defined media API tools — no filesystem access, no shell, no arbitrary network calls
- **Pluggable**: Chat backend interface makes it easy to add Slack, Discord, or other backends

## Supported services

| Service   | Status |
|-----------|--------|
| Sonarr    | Implemented — search, add, queue, history, health, remove failed |
| Radarr    | Planned |
| Prowlarr  | Planned |
| Overseerr | Planned |

## Configuration

Copy `config.yaml.example` to `config.yaml` and configure your service URLs. All secrets must be provided via environment variables:

| Variable | Description |
|----------|-------------|
| `ANTHROPIC_API_KEY` | Claude API key |
| `SONARR_API_KEY` | Sonarr API key |
| `GOOGLE_CHAT_WEBHOOK_URL` | Google Chat webhook URL |

```yaml
sonarr:
  base_url: http://sonarr:8989

chat:
  backend: googlechat

agent:
  model: claude-sonnet-4-5-20250929
  max_tokens: 4096

monitor:
  enabled: true
  interval: 5m

log:
  level: info
  format: text
```

## Running

### Binary

```bash
export ANTHROPIC_API_KEY=your-key
export SONARR_API_KEY=your-key
export GOOGLE_CHAT_WEBHOOK_URL=https://chat.googleapis.com/v1/spaces/.../messages?key=...&token=...

go build ./cmd/reel-life
./reel-life -config config.yaml
```

### Docker

```bash
docker build -t reel-life .
docker run -p 8080:8080 \
  -e ANTHROPIC_API_KEY=your-key \
  -e SONARR_API_KEY=your-key \
  -e GOOGLE_CHAT_WEBHOOK_URL=your-webhook \
  -v $(pwd)/config.yaml:/config.yaml \
  reel-life -config /config.yaml
```

The container runs as non-root with a minimal distroless base image. It only needs network access to your media services, the Google Chat API, and the Claude API.

## Development

```bash
go test ./...     # run tests
go build ./...    # verify compilation
go vet ./...      # lint
```
