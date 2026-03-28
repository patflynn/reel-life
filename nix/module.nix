{ config, lib, pkgs, ... }:
let
  cfg = config.services.reel-life;
in
{
  options.services.reel-life = {
    enable = lib.mkEnableOption "reel-life media chatops agent";

    package = lib.mkOption {
      type = lib.types.package;
      default = pkgs.reel-life;
      description = "The reel-life package to use";
    };

    sonarrUrl = lib.mkOption {
      type = lib.types.str;
      default = "http://localhost:8989";
      description = "Sonarr base URL";
    };

    chatBackend = lib.mkOption {
      type = lib.types.str;
      default = "googlechat";
      description = "Chat backend to use";
    };

    agentModel = lib.mkOption {
      type = lib.types.str;
      default = "claude-sonnet-4-5-20250929";
      description = "Claude model to use for the agent";
    };

    agentMaxTokens = lib.mkOption {
      type = lib.types.int;
      default = 4096;
      description = "Maximum tokens for agent responses";
    };

    monitorEnabled = lib.mkOption {
      type = lib.types.bool;
      default = true;
      description = "Whether to enable the health monitor";
    };

    monitorInterval = lib.mkOption {
      type = lib.types.str;
      default = "5m";
      description = "Polling interval for the health monitor";
    };

    logLevel = lib.mkOption {
      type = lib.types.str;
      default = "info";
      description = "Log level (debug, info, warn, error)";
    };

    logFormat = lib.mkOption {
      type = lib.types.str;
      default = "text";
      description = "Log format (text, json)";
    };

    environmentFiles = lib.mkOption {
      type = lib.types.listOf lib.types.path;
      default = [ ];
      description = "Files containing environment variables (for secrets like API keys)";
    };
  };

  config = lib.mkIf cfg.enable {
    environment.etc."reel-life/config.yaml".text = ''
      sonarr:
        base_url: ${cfg.sonarrUrl}
      chat:
        backend: ${cfg.chatBackend}
      agent:
        model: ${cfg.agentModel}
        max_tokens: ${toString cfg.agentMaxTokens}
      monitor:
        enabled: ${lib.boolToString cfg.monitorEnabled}
        interval: ${cfg.monitorInterval}
      log:
        level: ${cfg.logLevel}
        format: ${cfg.logFormat}
    '';

    systemd.services.reel-life = {
      description = "reel-life media chatops agent";
      after = [ "network-online.target" ];
      wants = [ "network-online.target" ];
      wantedBy = [ "multi-user.target" ];

      serviceConfig = {
        Type = "simple";
        ExecStart = "${cfg.package}/bin/reel-life -config /etc/reel-life/config.yaml";
        Restart = "on-failure";
        RestartSec = 10;

        # Security hardening
        DynamicUser = true;
        NoNewPrivileges = true;
        ProtectSystem = "strict";
        ProtectHome = true;
        PrivateTmp = true;
        PrivateDevices = true;
        ProtectKernelTunables = true;
        ProtectControlGroups = true;
        RestrictSUIDSGID = true;

        # Secrets via environment files (e.g. agenix paths)
        EnvironmentFile = cfg.environmentFiles;
      };
    };
  };
}
