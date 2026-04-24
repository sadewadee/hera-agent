# nix/nixosModules.nix — NixOS module for hera
#
# Two modes:
#   container.enable = false (default) -> native systemd service
#   container.enable = true            -> OCI container (persistent writable layer)
#
# Usage:
#   services.hera = {
#     enable = true;
#     settings.model = "anthropic/claude-sonnet-4";
#     environmentFiles = [ config.sops.secrets."hera/env".path ];
#   };
#
{ inputs, ... }: {
  flake.nixosModules.default = { config, lib, pkgs, ... }:

  let
    cfg = config.services.hera;
    hera = inputs.self.packages.${pkgs.system}.default;

    # Deep-merge config type
    deepConfigType = lib.types.mkOptionType {
      name = "hera-config-attrs";
      description = "Hera YAML config (attrset), merged deeply via lib.recursiveUpdate.";
      check = builtins.isAttrs;
      merge = _loc: defs: lib.foldl' lib.recursiveUpdate { } (map (d: d.value) defs);
    };

    # Generate config.yaml from Nix attrset (YAML is a superset of JSON)
    configJson = builtins.toJSON cfg.settings;
    generatedConfigFile = pkgs.writeText "hera-config.yaml" configJson;
    configFile = if cfg.configFile != null then cfg.configFile else generatedConfigFile;

    configMergeScript = pkgs.callPackage ./configMergeScript.nix { };

    # Generate .env from non-secret environment attrset
    envFileContent = lib.concatStringsSep "\n" (
      lib.mapAttrsToList (k: v: "${k}=${v}") cfg.environment
    );

    # Build documents derivation
    documentDerivation = pkgs.runCommand "hera-documents" { } (
      ''
        mkdir -p $out
      '' + lib.concatStringsSep "\n" (
        lib.mapAttrsToList (name: value:
          if builtins.isPath value || lib.isStorePath value
          then "cp ${value} $out/${name}"
          else "cat > $out/${name} <<'HERA_DOC_EOF'\n${value}\nHERA_DOC_EOF"
        ) cfg.documents
      )
    );

    containerName = "hera";
    containerDataDir = "/data";
    containerHomeDir = "/home/hera";

    containerBin = if cfg.container.backend == "docker"
      then "${pkgs.docker}/bin/docker"
      else "${pkgs.podman}/bin/podman";

  in {
    options.services.hera = {
      enable = lib.mkEnableOption "Hera AI agent";

      package = lib.mkOption {
        type = lib.types.package;
        default = hera;
        description = "The Hera package to use.";
      };

      settings = lib.mkOption {
        type = deepConfigType;
        default = { };
        description = "Hera configuration (merged into config.yaml).";
      };

      configFile = lib.mkOption {
        type = lib.types.nullOr lib.types.path;
        default = null;
        description = "Path to a pre-built config.yaml. Overrides settings.";
      };

      environment = lib.mkOption {
        type = lib.types.attrsOf lib.types.str;
        default = { };
        description = "Non-secret environment variables for Hera.";
      };

      environmentFiles = lib.mkOption {
        type = lib.types.listOf lib.types.path;
        default = [ ];
        description = "Paths to env files (e.g. sops secrets) loaded at runtime.";
      };

      documents = lib.mkOption {
        type = lib.types.attrsOf (lib.types.either lib.types.str lib.types.path);
        default = { };
        description = "Files to place in HERA_HOME/documents/.";
      };

      stateDir = lib.mkOption {
        type = lib.types.str;
        default = "/var/lib/hera";
        description = "State directory for Hera data.";
      };

      user = lib.mkOption {
        type = lib.types.str;
        default = "hera";
        description = "User to run Hera as.";
      };

      group = lib.mkOption {
        type = lib.types.str;
        default = "hera";
        description = "Group to run Hera as.";
      };

      container = {
        enable = lib.mkOption {
          type = lib.types.bool;
          default = false;
          description = "Run Hera in an OCI container.";
        };

        backend = lib.mkOption {
          type = lib.types.enum [ "docker" "podman" ];
          default = "podman";
          description = "Container runtime backend.";
        };

        image = lib.mkOption {
          type = lib.types.str;
          default = "ubuntu:24.04";
          description = "Base image for the container.";
        };
      };
    };

    config = lib.mkIf cfg.enable {
      users.users.${cfg.user} = {
        isSystemUser = true;
        group = cfg.group;
        home = cfg.stateDir;
        createHome = true;
      };
      users.groups.${cfg.group} = { };

      systemd.services.hera = {
        description = "Hera AI Agent";
        wantedBy = [ "multi-user.target" ];
        after = [ "network.target" ];

        environment = {
          HERA_HOME = cfg.stateDir;
          HERA_MANAGED = "true";
        } // cfg.environment;

        serviceConfig = {
          Type = "simple";
          User = cfg.user;
          Group = cfg.group;
          ExecStartPre = let
            preScript = pkgs.writeShellScript "hera-pre" ''
              set -euo pipefail
              mkdir -p "${cfg.stateDir}"

              # Merge Nix settings into config.yaml (preserves user keys)
              ${configMergeScript} ${configFile} "${cfg.stateDir}/config.yaml"

              # Write non-secret .env
              cat > "${cfg.stateDir}/.env" <<'EOF'
              ${envFileContent}
              EOF

              # Link documents
              ${lib.optionalString (cfg.documents != { }) ''
                mkdir -p "${cfg.stateDir}/documents"
                for f in ${documentDerivation}/*; do
                  ln -sf "$f" "${cfg.stateDir}/documents/$(basename "$f")"
                done
              ''}
            '';
          in "+${preScript}";

          ExecStart = "${cfg.package}/bin/hera-agent";
          Restart = "on-failure";
          RestartSec = 30;

          # Hardening
          ProtectSystem = "strict";
          ProtectHome = "read-only";
          ReadWritePaths = [ cfg.stateDir ];
          PrivateTmp = true;
          NoNewPrivileges = true;
        } // lib.optionalAttrs (cfg.environmentFiles != [ ]) {
          EnvironmentFile = cfg.environmentFiles;
        };
      };
    };
  };
}
