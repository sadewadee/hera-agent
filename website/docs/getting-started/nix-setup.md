# Nix / NixOS Setup

Hera includes a `flake.nix` for reproducible builds and development environments. No CGO is required — the binary is fully static.

## Quick run (no install)

```bash
nix run github:sadewadee/hera
```

## Install into your Nix profile

```bash
nix profile install github:sadewadee/hera
```

This installs `hera`, `hera-agent`, `hera-mcp`, and `hera-acp` into your profile.

## Development shell

The flake provides a dev shell with Go, gopls, staticcheck, Delve, and other tooling:

```bash
git clone https://github.com/sadewadee/hera.git
cd hera
nix develop
```

Inside the shell you get:

- `go` — Go toolchain
- `gopls` — Go Language Server
- `staticcheck` — static analysis
- `delve` — debugger
- `gotools` — goimports, godoc, etc.

## Build locally from the flake

```bash
nix build
./result/bin/hera --version
```

## NixOS system configuration

Add Hera to your NixOS `configuration.nix`:

```nix
{ config, pkgs, ... }:
let
  hera = (builtins.getFlake "github:sadewadee/hera").packages.${pkgs.system}.default;
in
{
  environment.systemPackages = [ hera ];
}
```

Or using a flake-based NixOS config (`flake.nix`):

```nix
{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    hera.url = "github:sadewadee/hera";
  };

  outputs = { nixpkgs, hera, ... }: {
    nixosConfigurations.myhost = nixpkgs.lib.nixosSystem {
      modules = [
        ({ pkgs, ... }: {
          environment.systemPackages = [
            hera.packages.${pkgs.system}.default
          ];
        })
      ];
    };
  };
}
```

## Home Manager

```nix
{ pkgs, ... }:
let
  hera = (builtins.getFlake "github:sadewadee/hera").packages.${pkgs.system}.default;
in
{
  home.packages = [ hera ];
}
```

## Vendor hash

On first local build you need to provide the correct vendor hash. Run:

```bash
nix build 2>&1 | grep "got:"
```

Copy the hash and update `vendorHash` in `flake.nix`:

```nix
packages.default = pkgs.buildGoModule {
  # ...
  vendorHash = "sha256-AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=";
};
```

## Running Hera as a NixOS service

You can define a systemd service for the headless agent:

```nix
systemd.services.hera-agent = {
  description = "Hera AI Agent (headless)";
  after = [ "network.target" ];
  wantedBy = [ "multi-user.target" ];
  environment = {
    OPENAI_API_KEY = "sk-..."; # better: use sops-nix for secrets
    HERA_AGENT_DEFAULT_PROVIDER = "openai";
  };
  serviceConfig = {
    ExecStart = "${hera}/bin/hera-agent";
    Restart = "on-failure";
    User = "hera";
    StateDirectory = "hera";
    WorkingDirectory = "/var/lib/hera";
  };
};

users.users.hera = {
  isSystemUser = true;
  group = "hera";
};
users.groups.hera = {};
```
