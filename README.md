# reel-life

An AI-powered chatops agent for media curation and quality control. reel-life connects Claude to your media management stack (starting with Sonarr) and communicates through Google Chat. It can respond to natural language requests like "search for Breaking Bad" or "what's in the download queue?", and it proactively monitors your services for health issues — alerting you when something goes wrong.

Claude operates as a constrained agent: it can only call a defined set of media API tools. No filesystem access, no shell commands, no arbitrary network calls. It reasons about what to do, calls the right Sonarr endpoint, and reports back.

## Architecture

```
┌─────────────┐     webhook      ┌──────────────┐     API calls     ┌─────────┐
│ Google Chat  │ ───────────────→ │  reel-life   │ ───────────────→  │ Sonarr  │
│   Space      │ ←─────────────── │              │ ←───────────────  │  (v3)   │
└─────────────┘   notifications   │  ┌────────┐  │   JSON responses  └─────────┘
                                  │  │ Claude │  │
                                  │  │ Agent  │  │
                                  │  └────────┘  │
                                  │  ┌────────┐  │
                                  │  │Monitor │──┘ polls health every 5m
                                  │  │ Loop   │
                                  │  └────────┘
                                  └──────────────┘
```

The **agent** handles reactive requests — a user asks something in Google Chat, Claude figures out which Sonarr tool to call, and sends the result back. The **monitor** handles proactive alerts — it polls Sonarr health on a schedule, deduplicates issues, and notifies your chat space when something new breaks (or resolves).

## Features

- **Reactive**: Natural language requests — "search for Breaking Bad", "what's downloading?", "check system health"
- **Proactive**: Automatic alerts for health issues, failed downloads, and indexer problems
- **Constrained**: The AI agent can ONLY call defined media API tools — no filesystem, no shell, no arbitrary network
- **Pluggable**: Chat backend interface supports additional backends (Slack, Discord, etc.)

## Supported services

| Service   | Status |
|-----------|--------|
| Sonarr    | Implemented — search, add, queue, history, health, remove failed |
| Radarr    | Planned |
| Prowlarr  | Planned |
| Overseerr | Planned |

## Agent capabilities

The Claude agent has access to these Sonarr tools:

| Tool | What it does |
|------|-------------|
| `search_series` | Search for TV series by name |
| `add_series` | Add a series to Sonarr for monitoring and downloading |
| `get_queue` | Show active and pending downloads |
| `get_history` | Show recent download history (completed, failed, imported) |
| `check_health` | Check Sonarr system health for warnings and errors |
| `remove_failed` | Remove a failed download from the queue, optionally blocklisting |

## Prerequisites

- **Anthropic API key** — [console.anthropic.com](https://console.anthropic.com/)
- **Running Sonarr instance** with API access enabled (v3 API)
- **Google Chat space** with an incoming webhook URL
- **For NixOS deployment**: NixOS with flakes enabled, [agenix](https://github.com/ryantm/agenix) for secrets management

## Quick start

There are three ways to deploy reel-life. See the [setup guide](docs/setup-guide.md) for detailed instructions.

### NixOS service (recommended for NixOS hosts)

Add to your flake inputs, import the module, configure the service, and deploy with `nixos-rebuild switch`. Secrets are managed with agenix. See [full NixOS setup instructions](docs/setup-guide.md#method-1-nixos-service).

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

The container runs as non-root with a minimal distroless base image.

### Binary

```bash
go build ./cmd/reel-life
export ANTHROPIC_API_KEY=your-key
export SONARR_API_KEY=your-key
export GOOGLE_CHAT_WEBHOOK_URL=https://chat.googleapis.com/v1/spaces/.../messages?key=...&token=...
./reel-life -config config.yaml
```

## Configuration

Copy `config.yaml.example` to `config.yaml`:

```yaml
sonarr:
  base_url: http://localhost:8989

chat:
  backend: googlechat

agent:
  model: claude-sonnet-4-5-20250929
  max_tokens: 4096

monitor:
  enabled: true
  interval: 5m

log:
  level: info     # debug, info, warn, error
  format: text    # text or json
```

Secrets are always provided via environment variables — never put API keys in config.yaml:

| Variable | Description |
|----------|-------------|
| `ANTHROPIC_API_KEY` | Claude API key (required) |
| `SONARR_API_KEY` | Sonarr API key (required) |
| `GOOGLE_CHAT_WEBHOOK_URL` | Google Chat webhook URL (required) |

The `SONARR_URL` environment variable can also override `sonarr.base_url` from the config file.

## Documentation

- [Setup guide](docs/setup-guide.md) — Detailed deployment instructions for all three methods
- [Google Chat setup](docs/google-chat-setup.md) — Creating and configuring the Google Chat webhook
- [Sonarr setup](docs/sonarr-setup.md) — Connecting reel-life to your Sonarr instance
- [Troubleshooting](docs/troubleshooting.md) — Common issues and how to fix them

## Development

```bash
go test ./...     # run tests
go build ./...    # verify compilation
go vet ./...      # lint
```

Uses a Nix flake for development dependencies:

```bash
nix develop       # enter dev shell with go, gopls, golangci-lint
```
