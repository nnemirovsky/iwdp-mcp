#!/usr/bin/env bash
set -euo pipefail

REPO="nnemirovsky/iwdp-mcp"
BINARY_NAME="iwdp-mcp"

# Determine install location: prefer plugin root, fall back to script dir.
if [ -n "${CLAUDE_PLUGIN_ROOT:-}" ]; then
  INSTALL_DIR="${CLAUDE_PLUGIN_ROOT}/bin"
else
  INSTALL_DIR="$(cd "$(dirname "$0")" && pwd)/../bin"
fi

BINARY="${INSTALL_DIR}/${BINARY_NAME}"

if [ ! -f "$BINARY" ]; then
  OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
  ARCH="$(uname -m)"
  case "$ARCH" in
    x86_64)  ARCH="amd64" ;;
    aarch64) ARCH="arm64" ;;
  esac

  # Resolve latest version tag (e.g. "v0.1.0" → "0.1.0")
  VERSION="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed -E 's/.*"v?([^"]+)".*/\1/')"
  ASSET="${BINARY_NAME}_${VERSION}_${OS}_${ARCH}"
  URL="https://github.com/${REPO}/releases/download/v${VERSION}/${ASSET}"

  echo "Downloading ${BINARY_NAME} v${VERSION} from ${URL}..." >&2
  mkdir -p "$INSTALL_DIR"
  curl -fsSL "$URL" -o "$BINARY"
  chmod +x "$BINARY"
  echo "Installed ${BINARY_NAME} v${VERSION} to ${BINARY}" >&2
fi

exec "$BINARY" "$@"
