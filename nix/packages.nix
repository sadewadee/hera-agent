# nix/packages.nix — Hera package built with buildGoModule
{ inputs, ... }: {
  perSystem = { pkgs, system, ... }:
    let
      # Import bundled skills, excluding runtime caches
      bundledSkills = pkgs.lib.cleanSourceWith {
        src = ../skills;
        filter = path: _type:
          !(pkgs.lib.hasInfix "/index-cache/" path);
      };

      runtimeDeps = with pkgs; [
        nodejs ripgrep git openssh ffmpeg
      ];

      runtimePath = pkgs.lib.makeBinPath runtimeDeps;
    in {
      packages.default = pkgs.buildGoModule {
        pname = "hera";
        version = "0.1.0";
        src = ./..;
        vendorHash = null; # Set to the correct hash after first build

        subPackages = [
          "cmd/hera"
          "cmd/hera-agent"
          "cmd/hera-mcp"
          "cmd/hera-acp"
        ];

        nativeBuildInputs = [ pkgs.makeWrapper ];

        postInstall = ''
          mkdir -p $out/share/hera
          cp -r ${bundledSkills} $out/share/hera/skills

          for bin in hera hera-agent hera-mcp hera-acp; do
            if [ -f "$out/bin/$bin" ]; then
              wrapProgram "$out/bin/$bin" \
                --suffix PATH : "${runtimePath}" \
                --set HERA_BUNDLED_SKILLS $out/share/hera/skills
            fi
          done
        '';

        meta = with pkgs.lib; {
          description = "Self-improving multi-platform AI agent";
          homepage = "https://github.com/sadewadee/hera";
          mainProgram = "hera";
          license = licenses.mit;
          platforms = platforms.unix;
        };
      };
    };
}
