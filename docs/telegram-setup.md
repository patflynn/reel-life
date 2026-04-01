# Telegram setup

This guide walks you through setting up reel-life with Telegram as the chat backend.

## 1. Create a bot via @BotFather

1. Open Telegram and search for [@BotFather](https://t.me/botfather).
2. Send `/newbot` and follow the prompts to choose a name and username.
3. BotFather will give you an API token like `123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11`. Save this — it's your `TELEGRAM_BOT_TOKEN`.

## 2. Find your Telegram user ID

Your user ID is needed for the allowlist so only you can interact with the bot.

1. Search for [@userinfobot](https://t.me/userinfobot) on Telegram.
2. Send it any message — it will reply with your user ID (a numeric value like `123456789`).

Alternatively, forward a message from yourself to [@JsonDumpBot](https://t.me/JsonDumpBot) and look for the `from.id` field.

## 3. Configure reel-life

Update your `config.yaml`:

```yaml
chat:
  backend: telegram
  telegram_chat_id: 0              # 0 = auto-capture from first message you send
  telegram_allowed_users: [123456789]  # your Telegram user ID
```

Set the environment variable:

```bash
export TELEGRAM_BOT_TOKEN=123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11
```

The bot token should always be provided via environment variable, never in the config file.

### Chat ID

If you set `telegram_chat_id: 0`, the bot will automatically capture the chat ID from the first message it receives. This is the simplest setup for personal use.

If you want proactive monitor alerts to work immediately (before sending your first message), set the chat ID explicitly. You can find it by sending a message to your bot and checking the logs — the auto-captured ID will be logged.

## 4. Test the connection

1. Start reel-life:

```bash
export ANTHROPIC_API_KEY=your-key
export SONARR_API_KEY=your-key
export TELEGRAM_BOT_TOKEN=your-bot-token
./reel-life -config config.yaml
```

2. Open a chat with your bot in Telegram.
3. Send `/start` — the bot should reply with a help message.
4. Try a command: `search for Breaking Bad` or `/ask what's downloading?`

## NixOS configuration

```nix
services.reel-life = {
  enable = true;
  chatBackend = "telegram";
  chatTelegramAllowedUsers = [ 123456789 ];
  # chatTelegramChatID = 0;  # default: auto-capture
  environmentFiles = [ config.age.secrets.reel-life.path ];
};
```

Your agenix secret file should contain:

```
ANTHROPIC_API_KEY=sk-ant-...
SONARR_API_KEY=...
TELEGRAM_BOT_TOKEN=123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11
```
