# reel-life

AI-powered chatops agent for media curation and quality control.

## Build

```bash
go build ./cmd/reel-life
```

## Test

```bash
go test ./...
```

## Lint

```bash
go vet ./...
```

## Run

```bash
# Required environment variables:
export ANTHROPIC_API_KEY=your-key
export SONARR_API_KEY=your-key
export GOOGLE_CHAT_WEBHOOK_URL=https://chat.googleapis.com/v1/spaces/...

./reel-life -config config.yaml
```

## Architecture

- `internal/config` — YAML config loading with env var overrides for secrets
- `internal/chat` — Chat adapter interface (`Notifier`) with Google Chat webhook implementation
- `internal/sonarr` — Sonarr v3 API client (interface + HTTP implementation)
- `internal/agent` — Claude tool-use agent with constrained Sonarr tools
- `internal/monitor` — Polling loop for proactive health alerts
- `cmd/reel-life` — Entrypoint wiring all components

## Testing conventions

- Use `net/http/httptest` for mock HTTP servers — no mocked interfaces for HTTP clients
- Test tool dispatch functions directly with known inputs
- Monitor tests use short intervals and context cancellation
- All secrets come from env vars; config tests use `t.Setenv`

## Dependencies

- `github.com/anthropics/anthropic-sdk-go` — Claude API client
- `github.com/invopop/jsonschema` — JSON Schema generation for tool definitions
- `gopkg.in/yaml.v3` — Config file parsing
- Standard library `net/http`, `log/slog` for everything else
