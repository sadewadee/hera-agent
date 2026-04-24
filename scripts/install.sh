#!/usr/bin/env bash
set -euo pipefail

# Hera AI Agent - Install Script
# Supports: Linux (amd64, arm64), macOS (amd64, arm64), Termux (arm64, arm)

REPO_URL="https://github.com/sadewadee/hera.git"
INSTALL_PREFIX="${PREFIX:-$HOME/.local}"
CONFIG_DIR="$HOME/.config/hera"
GO_VERSION="1.22.5"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

info()    { echo -e "${BLUE}[INFO]${NC} $*"; }
success() { echo -e "${GREEN}[OK]${NC} $*"; }
warn()    { echo -e "${YELLOW}[WARN]${NC} $*"; }
error()   { echo -e "${RED}[ERROR]${NC} $*"; exit 1; }

usage() {
    cat <<EOF
Hera AI Agent Installer

Usage: $0 [OPTIONS]

Options:
  --prefix <path>   Install prefix (default: $HOME/.local)
  --help            Show this help message

The installer will:
  1. Detect your platform and architecture
  2. Check/install Go (if needed)
  3. Clone or update the Hera repository
  4. Build all binaries
  5. Install to \$PREFIX/bin
  6. Create default config directory
EOF
    exit 0
}

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --prefix)
            INSTALL_PREFIX="$2"
            shift 2
            ;;
        --help|-h)
            usage
            ;;
        *)
            error "Unknown option: $1"
            ;;
    esac
done

detect_platform() {
    local os arch

    case "$(uname -s)" in
        Linux)
            if [ -n "${TERMUX_VERSION:-}" ] || [ -d "/data/data/com.termux" ]; then
                os="termux"
            else
                os="linux"
            fi
            ;;
        Darwin)
            os="darwin"
            ;;
        *)
            error "Unsupported operating system: $(uname -s)"
            ;;
    esac

    case "$(uname -m)" in
        x86_64|amd64)   arch="amd64" ;;
        aarch64|arm64)   arch="arm64" ;;
        armv7l|armhf)    arch="arm" ;;
        *)               error "Unsupported architecture: $(uname -m)" ;;
    esac

    echo "$os" "$arch"
}

check_go() {
    if command -v go &>/dev/null; then
        local go_ver
        go_ver=$(go version | grep -oP 'go\K[0-9]+\.[0-9]+' || echo "0.0")
        info "Go found: $(go version)"
        return 0
    fi
    return 1
}

install_go() {
    local os="$1" arch="$2"

    if [ "$os" = "termux" ]; then
        info "Installing Go via Termux package manager..."
        pkg install -y golang
        return
    fi

    if [ "$os" = "darwin" ] && command -v brew &>/dev/null; then
        info "Installing Go via Homebrew..."
        brew install go
        return
    fi

    # Download from go.dev
    local go_os="$os"
    if [ "$os" = "termux" ]; then
        go_os="linux"
    fi

    local tarball="go${GO_VERSION}.${go_os}-${arch}.tar.gz"
    local url="https://go.dev/dl/${tarball}"

    info "Downloading Go ${GO_VERSION} from ${url}..."
    local tmpdir
    tmpdir=$(mktemp -d)
    trap "rm -rf $tmpdir" EXIT

    if command -v curl &>/dev/null; then
        curl -fsSL "$url" -o "$tmpdir/$tarball"
    elif command -v wget &>/dev/null; then
        wget -q "$url" -O "$tmpdir/$tarball"
    else
        error "Neither curl nor wget found. Install one and retry."
    fi

    info "Installing Go to /usr/local/go..."
    if [ -w /usr/local ]; then
        rm -rf /usr/local/go
        tar -C /usr/local -xzf "$tmpdir/$tarball"
    else
        sudo rm -rf /usr/local/go
        sudo tar -C /usr/local -xzf "$tmpdir/$tarball"
    fi

    export PATH="/usr/local/go/bin:$PATH"
    success "Go ${GO_VERSION} installed"
}

clone_or_update() {
    local target_dir="$1"

    if [ -d "$target_dir/.git" ]; then
        info "Updating existing repository..."
        cd "$target_dir"
        git pull --ff-only || warn "Could not pull latest changes"
    elif [ -d "$target_dir" ]; then
        info "Directory exists but is not a git repo. Building from existing source..."
    else
        info "Cloning Hera repository..."
        git clone "$REPO_URL" "$target_dir"
    fi
}

build_binaries() {
    local src_dir="$1"
    cd "$src_dir"

    info "Building Hera binaries..."
    mkdir -p bin

    go build -o bin/hera ./cmd/hera
    success "Built bin/hera"

    go build -o bin/hera-agent ./cmd/hera-agent
    success "Built bin/hera-agent"

    go build -o bin/hera-mcp ./cmd/hera-mcp
    success "Built bin/hera-mcp"

    go build -o bin/hera-acp ./cmd/hera-acp
    success "Built bin/hera-acp"
}

install_binaries() {
    local src_dir="$1" prefix="$2"
    local bin_dir="$prefix/bin"

    mkdir -p "$bin_dir"

    for bin in hera hera-agent hera-mcp hera-acp; do
        if [ -f "$src_dir/bin/$bin" ]; then
            cp "$src_dir/bin/$bin" "$bin_dir/$bin"
            chmod +x "$bin_dir/$bin"
            success "Installed $bin -> $bin_dir/$bin"
        fi
    done
}

setup_config() {
    local src_dir="$1"

    mkdir -p "$CONFIG_DIR"

    if [ ! -f "$CONFIG_DIR/config.yaml" ] && [ -f "$src_dir/configs/hera.example.yaml" ]; then
        cp "$src_dir/configs/hera.example.yaml" "$CONFIG_DIR/config.yaml"
        success "Created default config at $CONFIG_DIR/config.yaml"
    else
        info "Config already exists at $CONFIG_DIR/config.yaml"
    fi

    # Create data directories.
    mkdir -p "$CONFIG_DIR/skills"
    mkdir -p "$CONFIG_DIR/data"
}

# Main
echo ""
echo "=============================="
echo "  Hera AI Agent Installer"
echo "=============================="
echo ""

read -r PLATFORM ARCH <<< "$(detect_platform)"
info "Platform: $PLATFORM ($ARCH)"
info "Install prefix: $INSTALL_PREFIX"
echo ""

# Step 1: Check/install Go
if ! check_go; then
    warn "Go not found. Installing..."
    install_go "$PLATFORM" "$ARCH"
    check_go || error "Go installation failed"
fi
echo ""

# Step 2: Determine source directory
SRC_DIR="$(cd "$(dirname "$0")/.." && pwd)"
if [ ! -f "$SRC_DIR/go.mod" ]; then
    # Not running from inside the repo; clone to a temp location.
    SRC_DIR="$HOME/hera"
    clone_or_update "$SRC_DIR"
fi
info "Source directory: $SRC_DIR"
echo ""

# Step 3: Build
build_binaries "$SRC_DIR"
echo ""

# Step 4: Install
install_binaries "$SRC_DIR" "$INSTALL_PREFIX"
echo ""

# Step 5: Setup config
setup_config "$SRC_DIR"
echo ""

# Step 6: PATH check
if [[ ":$PATH:" != *":$INSTALL_PREFIX/bin:"* ]]; then
    warn "$INSTALL_PREFIX/bin is not in your PATH."
    echo ""
    echo "Add it to your shell profile:"
    echo "  echo 'export PATH=\"$INSTALL_PREFIX/bin:\$PATH\"' >> ~/.bashrc"
    echo ""
fi

echo ""
success "Hera installation complete!"
echo ""
echo "Next steps:"
echo "  1. Run 'hera setup' to configure your LLM provider"
echo "  2. Run 'hera chat' to start chatting"
echo "  3. Run 'hera doctor' to check system health"
echo ""
