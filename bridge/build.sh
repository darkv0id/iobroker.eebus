#!/bin/bash

# Build script for eebus-bridge
# Generates binaries for Windows, macOS (Intel & Apple Silicon), and Linux

set -e  # Exit on error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Project paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
OUTPUT_DIR="$PROJECT_ROOT/bin"
CMD_PATH="./cmd/eebus-bridge"

# Binary name
BINARY_NAME="eebus-bridge"

# Get version from git if available
VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

# Build flags
# -s: Omit symbol table and debug information
# -w: Omit DWARF symbol table
LDFLAGS="-s -w -X main.version=$VERSION -X main.buildTime=$BUILD_TIME"

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}  EEBus Bridge - Multi-Platform Build${NC}"
echo -e "${BLUE}========================================${NC}"
echo -e "${GREEN}Version:${NC} $VERSION"
echo -e "${GREEN}Build Time:${NC} $BUILD_TIME"
echo ""

# Create output directory
mkdir -p "$OUTPUT_DIR"

# Function to build for a specific platform
build_platform() {
    local GOOS=$1
    local GOARCH=$2
    local SUFFIX=$3
    local DISPLAY_NAME=$4

    echo -e "${YELLOW}Building for $DISPLAY_NAME...${NC}"

    local OUTPUT_FILE="$OUTPUT_DIR/${BINARY_NAME}${SUFFIX}"

    # Build
    GOOS=$GOOS GOARCH=$GOARCH go build \
        -ldflags "$LDFLAGS" \
        -o "$OUTPUT_FILE" \
        $CMD_PATH

    if [ $? -eq 0 ]; then
        local SIZE=$(du -h "$OUTPUT_FILE" | cut -f1)
        echo -e "${GREEN}✓ Built successfully:${NC} $OUTPUT_FILE ($SIZE)"
    else
        echo -e "${RED}✗ Build failed for $DISPLAY_NAME${NC}"
        return 1
    fi

    echo ""
}

# Navigate to bridge directory
cd "$SCRIPT_DIR"

# Run tests first
if [ "${SKIP_TESTS}" != "true" ]; then
    echo -e "${YELLOW}Running tests...${NC}"
    go test ./... -cover
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✓ All tests passed${NC}"
        echo ""
    else
        echo -e "${RED}✗ Tests failed - aborting build${NC}"
        exit 1
    fi
fi

echo -e "${BLUE}Building binaries...${NC}"
echo ""

# Build for each platform
build_platform "linux"   "amd64" "-linux-amd64"      "Linux (x64)"
build_platform "linux"   "arm64" "-linux-arm64"      "Linux (ARM64)"
build_platform "darwin"  "amd64" "-darwin-amd64"     "macOS (Intel)"
build_platform "darwin"  "arm64" "-darwin-arm64"     "macOS (Apple Silicon)"
build_platform "windows" "amd64" "-windows-amd64.exe" "Windows (x64)"

# Create a symlink to the current platform's binary for development
CURRENT_OS=$(go env GOOS)
CURRENT_ARCH=$(go env GOARCH)
CURRENT_SUFFIX=""

case "$CURRENT_OS" in
    linux)
        CURRENT_SUFFIX="-linux-$CURRENT_ARCH"
        ;;
    darwin)
        CURRENT_SUFFIX="-darwin-$CURRENT_ARCH"
        ;;
    windows)
        CURRENT_SUFFIX="-windows-$CURRENT_ARCH.exe"
        ;;
esac

CURRENT_BINARY="$OUTPUT_DIR/${BINARY_NAME}${CURRENT_SUFFIX}"
DEFAULT_BINARY="$OUTPUT_DIR/${BINARY_NAME}"

if [ -f "$CURRENT_BINARY" ]; then
    echo -e "${YELLOW}Creating symlink for current platform...${NC}"
    ln -sf "$(basename "$CURRENT_BINARY")" "$DEFAULT_BINARY"
    echo -e "${GREEN}✓ Symlink created:${NC} $DEFAULT_BINARY -> $(basename "$CURRENT_BINARY")"
    echo ""
fi

# Summary
echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}  Build Summary${NC}"
echo -e "${BLUE}========================================${NC}"
echo -e "${GREEN}Output directory:${NC} $OUTPUT_DIR"
echo ""
echo -e "${GREEN}Built binaries:${NC}"
ls -lh "$OUTPUT_DIR" | grep -v "^total" | grep -v "^d"
echo ""
echo -e "${GREEN}✓ Build completed successfully!${NC}"
