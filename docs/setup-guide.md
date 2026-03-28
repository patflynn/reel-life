# Setup guide

This guide covers three ways to deploy reel-life. Pick the one that matches your environment.

## Prerequisites

Before you start, you'll need:

1. An **Anthropic API key** from [console.anthropic.com](https://console.anthropic.com/)
2. A **running Sonarr instance** with API access — see [Sonarr setup](sonarr-setup.md)
3. A **Google Chat webhook URL** — see [Google Chat setup](google-chat-setup.md)

## Method 1: NixOS service

The recommended deployment method for NixOS hosts. This runs reel-life as a hardened systemd service with secrets managed by agenix.

### Step 1: Add reel-life to your flake inputs

In your system flake (`flake.nix`), add reel-life as an input:

```nix
{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    # ... your other inputs ...

    reel-life.url = "github:patflynn/reel-life";
    reel-life.inputs.nixpkgs.follows = "nixpkgs";
  };

  outputs = { self, nixpkgs, reel-life, ... }: {
    nixosConfigurations.classic-laddie = nixpkgs.lib.nixosSystem {
      # ...
      modules = [
        reel-life.nixosModules.default
        ./hosts/classic-laddie/configuration.nix
      ];
    };
  };
}
```

### Step 2: Create agenix secrets

reel-life needs three secrets. The easiest approach is a single environment file containing all of them.

Create the secret source file:

```bash
# Create a temporary file with your secrets (don't commit this)
cat > /tmp/reel-life-env.plain <<'EOF'
ANTHROPIC_API_KEY=sk-ant-api03-your-key-here
SONARR_API_KEY=your-sonarr-api-key
GOOGLE_CHAT_WEBHOOK_URL=https://chat.googleapis.com/v1/spaces/SPACE_ID/messages?key=KEY&token=TOKEN
EOF
```

Encrypt it with agenix:

```bash
cd /path/to/your/secrets/repo

# Add a secrets entry (in secrets.nix)
# "reel-life-env.age".publicKeys = [ your-host-key your-user-key ];

# Encrypt the file
agenix -e reel-life-env.age < /tmp/reel-life-env.plain

# Clean up the plaintext
rm /tmp/reel-life-env.plain
```

Declare the secret in your agenix config:

```nix
# secrets.nix
{
  "reel-life-env.age".publicKeys = [
    "ssh-ed25519 AAAA... classic-laddie"  # host key
    "ssh-ed25519 AAAA... your-user-key"   # your key for editing
  ];
}
```

And in your NixOS configuration:

```nix
age.secrets."reel-life-env" = {
  file = ../secrets/reel-life-env.age;
  # No owner/group needed — reel-life uses DynamicUser
  # and reads the file via systemd's EnvironmentFile directive
};
```

### Step 3: Configure the service

In your host's NixOS configuration:

```nix
{ config, ... }:
{
  services.reel-life = {
    enable = true;
    sonarrUrl = "http://localhost:8989";
    chatBackend = "googlechat";
    monitorInterval = "5m";
    environmentFiles = [
      config.age.secrets."reel-life-env".path
    ];
  };
}
```

The full set of options:

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `enable` | bool | `false` | Enable the reel-life service |
| `package` | package | `pkgs.reel-life` | The reel-life package to use |
| `sonarrUrl` | string | `"http://localhost:8989"` | Sonarr base URL |
| `chatBackend` | string | `"googlechat"` | Chat backend |
| `agentModel` | string | `"claude-sonnet-4-5-20250929"` | Claude model for the agent |
| `agentMaxTokens` | int | `4096` | Max tokens for agent responses |
| `monitorEnabled` | bool | `true` | Enable health monitor |
| `monitorInterval` | string | `"5m"` | Health check polling interval |
| `logLevel` | string | `"info"` | Log level: debug, info, warn, error |
| `logFormat` | string | `"text"` | Log format: text or json |
| `environmentFiles` | list of paths | `[]` | Paths to env files with secrets |

### Step 4: Deploy

```bash
sudo nixos-rebuild switch
```

### Step 5: Verify

Check that the service is running:

```bash
systemctl status reel-life
```

Expected output:

```
● reel-life.service - reel-life media chatops agent
     Loaded: loaded (/etc/systemd/system/reel-life.service; enabled)
     Active: active (running)
```

Watch the logs:

```bash
journalctl -u reel-life -f
```

You should see startup messages followed by periodic health check logs:

```
level=INFO msg="starting reel-life"
level=INFO msg="monitor: checking health"
level=INFO msg="monitor: 0 new issues"
```

### Security hardening

The NixOS module runs with these systemd hardening options enabled automatically:

- `DynamicUser` — no static system user needed
- `NoNewPrivileges` — cannot escalate privileges
- `ProtectSystem=strict` — filesystem is read-only except where needed
- `ProtectHome=true` — no access to home directories
- `PrivateTmp` / `PrivateDevices` — isolated temp and device namespaces
- `ProtectKernelTunables` / `ProtectControlGroups` — kernel protection
- `RestrictSUIDSGID` — cannot create setuid/setgid files

---

## Method 2: Docker

### Step 1: Build the image

```bash
git clone https://github.com/patflynn/reel-life.git
cd reel-life
docker build -t reel-life .
```

The image uses a multi-stage build: Go compilation in an Alpine builder, runtime on `distroless/static-debian12:nonroot`. The final image is minimal and runs as non-root.

### Step 2: Create config.yaml

```bash
cp config.yaml.example config.yaml
```

Edit `config.yaml` to set your Sonarr URL. If Sonarr is running on the Docker host:

```yaml
sonarr:
  base_url: http://host.docker.internal:8989

chat:
  backend: googlechat

monitor:
  enabled: true
  interval: 5m
```

If Sonarr is on the same Docker network, use its container name instead (e.g., `http://sonarr:8989`).

### Step 3: Run

```bash
docker run -d \
  --name reel-life \
  -p 8080:8080 \
  -e ANTHROPIC_API_KEY=sk-ant-api03-your-key-here \
  -e SONARR_API_KEY=your-sonarr-api-key \
  -e GOOGLE_CHAT_WEBHOOK_URL='https://chat.googleapis.com/v1/spaces/SPACE_ID/messages?key=KEY&token=TOKEN' \
  -v $(pwd)/config.yaml:/config.yaml:ro \
  reel-life -config /config.yaml
```

### Step 4: Verify

```bash
# Check container is running
docker ps | grep reel-life

# Check logs
docker logs -f reel-life

# Health check
curl http://localhost:8080/healthz
# Expected: ok
```

---

## Method 3: Binary

### Step 1: Build

```bash
git clone https://github.com/patflynn/reel-life.git
cd reel-life
go build ./cmd/reel-life
```

Or with Nix:

```bash
nix build
# Binary is at ./result/bin/reel-life
```

### Step 2: Create config.yaml

```bash
cp config.yaml.example config.yaml
```

Edit `config.yaml` with your Sonarr URL:

```yaml
sonarr:
  base_url: http://localhost:8989

chat:
  backend: googlechat

monitor:
  enabled: true
  interval: 5m
```

### Step 3: Set environment variables

```bash
export ANTHROPIC_API_KEY=sk-ant-api03-your-key-here
export SONARR_API_KEY=your-sonarr-api-key
export GOOGLE_CHAT_WEBHOOK_URL='https://chat.googleapis.com/v1/spaces/SPACE_ID/messages?key=KEY&token=TOKEN'
```

### Step 4: Run

```bash
./reel-life -config config.yaml
```

The service starts an HTTP server on port 8080 and begins health monitoring immediately.

### Step 5: Verify

In another terminal:

```bash
curl http://localhost:8080/healthz
# Expected: ok
```

Check the process output for health monitoring logs.

---

## Next steps

- [Google Chat setup](google-chat-setup.md) — Create the webhook for your chat space
- [Sonarr setup](sonarr-setup.md) — Configure API access
- [Troubleshooting](troubleshooting.md) — If something isn't working
