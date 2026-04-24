# nix/configMergeScript.nix — Deep-merge Nix settings into existing config.yaml
#
# Used by the NixOS module activation script and by checks.nix tests.
# Nix keys override; user-added keys (skills, streaming, etc.) are preserved.
#
# Go-native implementation using yq instead of Python/PyYAML.
{ pkgs }:
pkgs.writeShellScript "hera-config-merge" ''
  set -euo pipefail

  NIX_JSON="$1"
  CONFIG_PATH="$2"

  # If config doesn't exist yet, convert JSON to YAML directly
  if [ ! -f "$CONFIG_PATH" ]; then
    ${pkgs.yq-go}/bin/yq -P eval '.' "$NIX_JSON" > "$CONFIG_PATH"
    exit 0
  fi

  # Deep-merge: Nix settings override existing keys, user-only keys preserved
  # yq's * operator performs deep merge (right side wins for conflicts)
  MERGED=$(${pkgs.yq-go}/bin/yq eval-all 'select(fileIndex == 0) * select(fileIndex == 1)' \
    "$CONFIG_PATH" <(${pkgs.yq-go}/bin/yq -P eval '.' "$NIX_JSON"))

  echo "$MERGED" > "$CONFIG_PATH"
''
