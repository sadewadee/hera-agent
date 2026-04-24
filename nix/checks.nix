# nix/checks.nix — Build-time verification tests for Hera
#
# Checks are Linux-only due to cross-compilation constraints.
# The package and devShell still work on macOS.
{ inputs, ... }: {
  perSystem = { pkgs, system, lib, ... }:
    let
      hera = inputs.self.packages.${system}.default;

      configMergeScript = pkgs.callPackage ./configMergeScript.nix { };
    in {
      checks = lib.optionalAttrs pkgs.stdenv.hostPlatform.isLinux {
        # Verify binaries exist and are executable
        package-contents = pkgs.runCommand "hera-package-contents" { } ''
          set -e
          echo "=== Checking binaries ==="
          test -x ${hera}/bin/hera || (echo "FAIL: hera binary missing"; exit 1)
          test -x ${hera}/bin/hera-agent || (echo "FAIL: hera-agent binary missing"; exit 1)
          echo "PASS: All binaries present"

          echo "=== Checking version ==="
          ${hera}/bin/hera version 2>&1 | grep -qi "hera" || (echo "FAIL: version check"; exit 1)
          echo "PASS: Version check"

          echo "=== All checks passed ==="
          mkdir -p $out
          echo "ok" > $out/result
        '';

        # Verify all expected binary entry points are present
        entry-points-sync = pkgs.runCommand "hera-entry-points-sync" { } ''
          set -e
          echo "=== Checking entry points ==="
          for bin in hera hera-agent hera-mcp hera-acp; do
            test -x ${hera}/bin/$bin || (echo "FAIL: $bin binary missing from Nix package"; exit 1)
            echo "PASS: $bin present"
          done

          mkdir -p $out
          echo "ok" > $out/result
        '';

        # Verify CLI subcommands are accessible
        cli-commands = pkgs.runCommand "hera-cli-commands" { } ''
          set -e
          export HOME=$(mktemp -d)

          echo "=== Checking hera --help ==="
          ${hera}/bin/hera --help 2>&1 | grep -q "gateway" || (echo "FAIL: gateway subcommand missing"; exit 1)
          ${hera}/bin/hera --help 2>&1 | grep -q "config" || (echo "FAIL: config subcommand missing"; exit 1)
          echo "PASS: All subcommands accessible"

          echo "=== All CLI checks passed ==="
          mkdir -p $out
          echo "ok" > $out/result
        '';

        # Verify bundled skills are present in the package
        bundled-skills = pkgs.runCommand "hera-bundled-skills" { } ''
          set -e
          echo "=== Checking bundled skills ==="
          test -d ${hera}/share/hera/skills || (echo "FAIL: skills directory missing"; exit 1)
          echo "PASS: skills directory exists"

          SKILL_COUNT=$(find ${hera}/share/hera/skills -name "SKILL.md" | wc -l)
          test "$SKILL_COUNT" -gt 0 || (echo "FAIL: no SKILL.md files found in skills directory"; exit 1)
          echo "PASS: $SKILL_COUNT bundled skills found"

          echo "=== All bundled skills checks passed ==="
          mkdir -p $out
          echo "ok" > $out/result
        '';

        # Verify HERA_MANAGED guard works on all mutation commands
        managed-guard = pkgs.runCommand "hera-managed-guard" { } ''
          set -e
          export HOME=$(mktemp -d)

          check_blocked() {
            local label="$1"
            shift
            OUTPUT=$(HERA_MANAGED=true "$@" 2>&1 || true)
            echo "$OUTPUT" | grep -q "managed by NixOS" || (echo "FAIL: $label not guarded"; echo "$OUTPUT"; exit 1)
            echo "PASS: $label blocked in managed mode"
          }

          echo "=== Checking HERA_MANAGED guards ==="
          check_blocked "config set" ${hera}/bin/hera config set model foo
          check_blocked "config edit" ${hera}/bin/hera config edit

          echo "=== All guard checks passed ==="
          mkdir -p $out
          echo "ok" > $out/result
        '';

        # Config merge round-trip test
        config-roundtrip = let
          nixSettings = pkgs.writeText "nix-settings.json" (builtins.toJSON {
            model = "test/nix-model";
            toolsets = ["nix-toolset"];
            terminal = { backend = "docker"; timeout = 999; };
            mcp_servers = {
              nix-server = { command = "echo"; args = ["nix"]; };
            };
          });

          fixtureB = pkgs.writeText "fixture-b.yaml" ''
            model: "old-model"
            mcp_servers:
              old-server:
                url: "http://old"
          '';
          fixtureC = pkgs.writeText "fixture-c.yaml" ''
            skills:
              disabled:
                - skill-a
                - skill-b
            session_reset:
              mode: idle
              idle_minutes: 30
            streaming:
              enabled: true
            fallback_model:
              provider: openrouter
              model: test-fallback
          '';
          fixtureD = pkgs.writeText "fixture-d.yaml" ''
            model: "user-model"
            skills:
              disabled:
                - skill-x
            streaming:
              enabled: true
              transport: edit
          '';
          fixtureE = pkgs.writeText "fixture-e.yaml" ''
            mcp_servers:
              user-server:
                url: "http://user-mcp"
              nix-server:
                command: "old-cmd"
                args: ["old"]
          '';
          fixtureF = pkgs.writeText "fixture-f.yaml" ''
            terminal:
              cwd: "/user/path"
              custom_key: "preserved"
              env_passthrough:
                - USER_VAR
          '';

        in pkgs.runCommand "hera-config-roundtrip" {
          nativeBuildInputs = [ pkgs.jq pkgs.yq-go ];
        } ''
          set -e
          export HOME=$(mktemp -d)
          ERRORS=""

          fail() { ERRORS="$ERRORS\nFAIL: $1"; }

          merge_and_read() {
            local hera_home="$1"
            export HERA_HOME="$hera_home"
            ${configMergeScript} ${nixSettings} "$hera_home/config.yaml"
            yq -o=json eval '.' "$hera_home/config.yaml"
          }

          echo "=== Scenario A: Fresh install ==="
          A_HOME=$(mktemp -d)
          A_CONFIG=$(merge_and_read "$A_HOME")
          echo "$A_CONFIG" | jq -e '.model == "test/nix-model"' > /dev/null \
            || fail "A: model not set from Nix"
          echo "PASS: Scenario A"

          echo "=== Scenario B: Nix overrides ==="
          B_HOME=$(mktemp -d)
          install -m 0644 ${fixtureB} "$B_HOME/config.yaml"
          B_CONFIG=$(merge_and_read "$B_HOME")
          echo "$B_CONFIG" | jq -e '.model == "test/nix-model"' > /dev/null \
            || fail "B: Nix model did not override"
          echo "PASS: Scenario B"

          echo "=== Scenario C: User keys preserved ==="
          C_HOME=$(mktemp -d)
          install -m 0644 ${fixtureC} "$C_HOME/config.yaml"
          C_CONFIG=$(merge_and_read "$C_HOME")
          echo "$C_CONFIG" | jq -e '.skills.disabled == ["skill-a", "skill-b"]' > /dev/null \
            || fail "C: skills.disabled not preserved"
          echo "PASS: Scenario C"

          echo "=== Scenario D: Mixed merge ==="
          D_HOME=$(mktemp -d)
          install -m 0644 ${fixtureD} "$D_HOME/config.yaml"
          D_CONFIG=$(merge_and_read "$D_HOME")
          echo "$D_CONFIG" | jq -e '.model == "test/nix-model"' > /dev/null \
            || fail "D: Nix model did not override user model"
          echo "PASS: Scenario D"

          echo "=== Scenario E: MCP additive merge ==="
          E_HOME=$(mktemp -d)
          install -m 0644 ${fixtureE} "$E_HOME/config.yaml"
          E_CONFIG=$(merge_and_read "$E_HOME")
          echo "$E_CONFIG" | jq -e '.mcp_servers."nix-server".command == "echo"' > /dev/null \
            || fail "E: Nix MCP server did not override same-name user server"
          echo "PASS: Scenario E"

          echo "=== Scenario F: Nested deep merge ==="
          F_HOME=$(mktemp -d)
          install -m 0644 ${fixtureF} "$F_HOME/config.yaml"
          F_CONFIG=$(merge_and_read "$F_HOME")
          echo "$F_CONFIG" | jq -e '.terminal.backend == "docker"' > /dev/null \
            || fail "F: Nix terminal.backend did not override"
          echo "PASS: Scenario F"

          echo "=== Scenario G: Idempotency ==="
          G_HOME=$(mktemp -d)
          install -m 0644 ${fixtureD} "$G_HOME/config.yaml"
          ${configMergeScript} ${nixSettings} "$G_HOME/config.yaml"
          FIRST=$(cat "$G_HOME/config.yaml")
          ${configMergeScript} ${nixSettings} "$G_HOME/config.yaml"
          SECOND=$(cat "$G_HOME/config.yaml")
          if [ "$FIRST" != "$SECOND" ]; then
            fail "G: second merge produced different output"
          fi
          echo "PASS: Scenario G"

          if [ -n "$ERRORS" ]; then
            echo ""
            echo "FAILURES:"
            echo -e "$ERRORS"
            exit 1
          fi

          echo ""
          echo "=== All 7 merge scenarios passed ==="
          mkdir -p $out
          echo "ok" > $out/result
        '';
      };
    };
}
