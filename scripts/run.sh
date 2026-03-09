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

  ASSET="${BINARY_NAME}_${OS}_${ARCH}"
  URL="https://github.com/${REPO}/releases/latest/download/${ASSET}"

  echo "Downloading ${BINARY_NAME} from ${URL}..." >&2
  mkdir -p "$INSTALL_DIR"
  curl -fsSL "$URL" -o "$BINARY"
  chmod +x "$BINARY"
  echo "Installed ${BINARY_NAME} to ${BINARY}" >&2
fi

exec "$BINARY" "$@"
