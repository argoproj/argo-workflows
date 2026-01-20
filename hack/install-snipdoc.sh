#!/usr/bin/env bash

set -euo pipefail

# Install snipdoc from GitHub releases
# Usage: install-snipdoc.sh <target-path> [version]

if [ $# -lt 1 ]; then
    echo "Usage: $0 <target-path> [version]" >&2
    exit 1
fi

TARGET_PATH="$1"
VERSION="${2:-v0.1.12}"

# Detect OS
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$OS" in
    darwin)
        OS_NAME="macos"
        ;;
    linux)
        OS_NAME="linux"
        ;;
    *)
        echo "Unsupported OS: $OS" >&2
        exit 1
        ;;
esac

# Detect architecture
ARCH=$(uname -m)
case "$ARCH" in
    x86_64)
        ARCH_NAME="x86_64"
        ;;
    aarch64|arm64)
        ARCH_NAME="aarch64"
        ;;
    *)
        echo "Unsupported architecture: $ARCH" >&2
        exit 1
        ;;
esac

# Build download URL
SNIPDOC_TARBALL="snipdoc-${ARCH_NAME}-${OS_NAME}.tar.xz"
DOWNLOAD_URL="https://github.com/kaplanelad/snipdoc/releases/download/${VERSION}/${SNIPDOC_TARBALL}"

echo "Downloading snipdoc from $DOWNLOAD_URL"

# Create temporary directory
TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

# Download and extract
curl -sSfL "$DOWNLOAD_URL" -o "$TMPDIR/snipdoc.tar.xz"
tar -xJf "$TMPDIR/snipdoc.tar.xz" -C "$TMPDIR"

# Find and install the binary
mkdir -p "$(dirname "$TARGET_PATH")"
find "$TMPDIR" -name snipdoc -type f -exec mv {} "$TARGET_PATH" \;
chmod +x "$TARGET_PATH"

echo "✓ snipdoc installed to $TARGET_PATH"
