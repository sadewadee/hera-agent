# nix/devShell.nix — Development shell for Hera (Go)
{ inputs, ... }: {
  perSystem = { pkgs, ... }:
    let
      go = pkgs.go_1_22;
    in {
      devShells.default = pkgs.mkShell {
        packages = with pkgs; [
          go gopls gotools go-tools delve golangci-lint
          sqlite nodejs ripgrep git openssh ffmpeg
        ];

        shellHook = ''
          echo "Hera development shell"
          echo "Go $(go version | awk '{print $3}')"
          export GOPATH=$HOME/go
          export PATH=$GOPATH/bin:$PATH
          echo "Ready. Run 'go run ./cmd/hera' to start."
        '';
      };
    };
}
