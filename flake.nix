{
  description = "Hera - Self-improving multi-platform AI agent";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
      in
      {
        packages.default = pkgs.buildGoModule {
          pname = "hera";
          version = "0.1.0";
          src = ./.;
          vendorHash = null; # Set to the correct hash after first build
          subPackages = [
            "cmd/hera"
            "cmd/hera-agent"
            "cmd/hera-mcp"
            "cmd/hera-acp"
          ];
          meta = with pkgs.lib; {
            description = "Self-improving multi-platform AI agent";
            homepage = "https://github.com/sadewadee/hera";
            license = licenses.mit;
            mainProgram = "hera";
          };
        };

        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            go
            gopls
            gotools
            go-tools      # staticcheck
            delve         # debugger
            golangci-lint
            sqlite
            nodejs        # for WhatsApp sidecar
          ];

          shellHook = ''
            echo "Hera development shell"
            echo "Go $(go version | awk '{print $3}')"
            export GOPATH=$HOME/go
            export PATH=$GOPATH/bin:$PATH
          '';
        };
      }
    );
}
