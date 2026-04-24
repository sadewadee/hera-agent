#!/bin/bash
# ============================================================================
# Hera Release Script
#
# Generates changelogs and creates GitHub releases.
#
# Usage:
#   # Preview changelog (dry run)
#   bash scripts/release.sh
#
#   # Create a release
#   bash scripts/release.sh --publish
#
#   # Specify version bump
#   bash scripts/release.sh --bump minor --publish
#
#   # First release (no previous tag)
#   bash scripts/release.sh --bump minor --publish --first-release
# ============================================================================

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$REPO_ROOT"

# ──────────────────────────────────────────────────────────────────────
# Defaults
# ──────────────────────────────────────────────────────────────────────
BUMP=""
PUBLISH=false
FIRST_RELEASE=false
DRY_RUN=true

# ──────────────────────────────────────────────────────────────────────
# Parse arguments
# ──────────────────────────────────────────────────────────────────────
while [[ $# -gt 0 ]]; do
    case "$1" in
        --bump)
            BUMP="$2"
            shift 2
            ;;
        --publish)
            PUBLISH=true
            DRY_RUN=false
            shift
            ;;
        --first-release)
            FIRST_RELEASE=true
            shift
            ;;
        --help|-h)
            echo "Usage: $0 [--bump major|minor|patch] [--publish] [--first-release]"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

# ──────────────────────────────────────────────────────────────────────
# Determine version
# ──────────────────────────────────────────────────────────────────────
if [ "$FIRST_RELEASE" = true ]; then
    PREV_TAG=""
    CURRENT_VERSION="0.1.0"
else
    PREV_TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "")
    if [ -z "$PREV_TAG" ]; then
        echo "No previous tag found. Use --first-release for initial release."
        exit 1
    fi
    CURRENT_VERSION="${PREV_TAG#v}"
fi

# Parse semver components
IFS='.' read -r MAJOR MINOR PATCH <<< "$CURRENT_VERSION"

case "$BUMP" in
    major)
        MAJOR=$((MAJOR + 1))
        MINOR=0
        PATCH=0
        ;;
    minor)
        MINOR=$((MINOR + 1))
        PATCH=0
        ;;
    patch)
        PATCH=$((PATCH + 1))
        ;;
    "")
        # No bump specified, keep current for dry run
        ;;
    *)
        echo "Invalid bump type: $BUMP (use major, minor, or patch)"
        exit 1
        ;;
esac

NEW_VERSION="${MAJOR}.${MINOR}.${PATCH}"
NEW_TAG="v${NEW_VERSION}"
CALVER_TAG="v$(date +%Y.%-m.%-d)"

echo "=== Hera Release ==="
echo "Previous tag:  ${PREV_TAG:-none}"
echo "New version:   ${NEW_VERSION}"
echo "New tag:       ${NEW_TAG}"
echo "CalVer:        ${CALVER_TAG}"
echo ""

# ──────────────────────────────────────────────────────────────────────
# Generate changelog
# ──────────────────────────────────────────────────────────────────────
echo "=== Changelog ==="
echo ""

if [ -n "$PREV_TAG" ]; then
    RANGE="${PREV_TAG}..HEAD"
else
    RANGE="HEAD"
fi

# Categorize commits by conventional commit prefix
declare -A SECTIONS
SECTIONS=(
    ["feat"]="Features"
    ["fix"]="Bug Fixes"
    ["docs"]="Documentation"
    ["refactor"]="Refactoring"
    ["test"]="Tests"
    ["ci"]="CI/CD"
    ["chore"]="Maintenance"
)

for prefix in feat fix docs refactor test ci chore; do
    COMMITS=$(git log "$RANGE" --oneline --grep="^${prefix}" 2>/dev/null || true)
    if [ -n "$COMMITS" ]; then
        echo "### ${SECTIONS[$prefix]}"
        echo "$COMMITS" | while read -r line; do
            echo "- $line"
        done
        echo ""
    fi
done

# Uncategorized commits
OTHER=$(git log "$RANGE" --oneline --invert-grep \
    --grep="^feat" --grep="^fix" --grep="^docs" \
    --grep="^refactor" --grep="^test" --grep="^ci" \
    --grep="^chore" 2>/dev/null || true)
if [ -n "$OTHER" ]; then
    echo "### Other"
    echo "$OTHER" | while read -r line; do
        echo "- $line"
    done
    echo ""
fi

# ──────────────────────────────────────────────────────────────────────
# Build binaries (if publishing)
# ──────────────────────────────────────────────────────────────────────
if [ "$PUBLISH" = true ]; then
    echo "=== Building binaries ==="

    BUILD_DIR="$REPO_ROOT/dist"
    rm -rf "$BUILD_DIR"
    mkdir -p "$BUILD_DIR"

    TARGETS=(
        "linux/amd64"
        "linux/arm64"
        "darwin/amd64"
        "darwin/arm64"
        "windows/amd64"
    )

    for target in "${TARGETS[@]}"; do
        OS="${target%/*}"
        ARCH="${target#*/}"
        EXT=""
        [ "$OS" = "windows" ] && EXT=".exe"

        echo "  Building ${OS}/${ARCH}..."
        GOOS="$OS" GOARCH="$ARCH" go build \
            -ldflags="-s -w -X main.version=${NEW_VERSION}" \
            -o "${BUILD_DIR}/hera_${NEW_VERSION}_${OS}_${ARCH}/hera${EXT}" \
            ./cmd/hera

        # Create archive
        cd "$BUILD_DIR"
        if [ "$OS" = "windows" ]; then
            zip -r "hera_${NEW_VERSION}_${OS}_${ARCH}.zip" \
                "hera_${NEW_VERSION}_${OS}_${ARCH}/"
        else
            tar czf "hera_${NEW_VERSION}_${OS}_${ARCH}.tar.gz" \
                "hera_${NEW_VERSION}_${OS}_${ARCH}/"
        fi
        cd "$REPO_ROOT"
    done

    echo ""
    echo "=== Creating GitHub release ==="

    # Generate checksums
    cd "$BUILD_DIR"
    sha256sum hera_${NEW_VERSION}_*.{tar.gz,zip} > checksums.txt 2>/dev/null || true
    cd "$REPO_ROOT"

    # Tag and push
    git tag -a "$NEW_TAG" -m "Release ${NEW_VERSION}"
    git push origin "$NEW_TAG"

    # Create release with gh
    if command -v gh &>/dev/null; then
        gh release create "$NEW_TAG" \
            --title "Hera ${NEW_VERSION} (${CALVER_TAG})" \
            --generate-notes \
            "${BUILD_DIR}"/hera_${NEW_VERSION}_*.{tar.gz,zip} \
            "${BUILD_DIR}/checksums.txt"
        echo "Release created: ${NEW_TAG}"
    else
        echo "gh CLI not found. Please install it and run:"
        echo "  gh release create ${NEW_TAG} dist/hera_${NEW_VERSION}_*"
    fi
else
    echo "(Dry run -- use --publish to create the release)"
fi
