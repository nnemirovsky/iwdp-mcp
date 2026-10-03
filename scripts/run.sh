#!/usr/bin/env bash
# Launches the iwdp-mcp server, downloading its release binary on first run.
#
# The binary is pinned to this plugin's own version and checked against the
# release's checksums file before it is ever run.
set -euo pipefail

REPO="nnemirovsky/iwdp-mcp"
BINARY_NAME="iwdp-mcp"

PLUGIN_ROOT="${CLAUDE_PLUGIN_ROOT:-$(cd "$(dirname "$0")/.." && pwd)}"

# Release tags match the plugin version, so a plugin version always runs the
# binary built from the same commit.
VERSION="$(sed -nE 's/^[[:space:]]*"version":[[:space:]]*"([^"]+)".*/\1/p' "${PLUGIN_ROOT}/.claude-plugin/plugin.json" | head -1)"
if [ -z "$VERSION" ]; then
  echo "iwdp-mcp: cannot read the plugin version from ${PLUGIN_ROOT}/.claude-plugin/plugin.json" >&2
  exit 1
fi

INSTALL_DIR="${PLUGIN_ROOT}/bin"
BINARY="${INSTALL_DIR}/${BINARY_NAME}-${VERSION}"

if [ ! -x "$BINARY" ]; then
  OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
  case "$OS" in
    darwin|linux) ;;
    *) echo "iwdp-mcp: no release binary for ${OS}" >&2; exit 1 ;;
  esac
  ARCH="$(uname -m)"
  case "$ARCH" in
    x86_64|amd64)  ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *) echo "iwdp-mcp: no release binary for ${ARCH}" >&2; exit 1 ;;
  esac

  ASSET="${BINARY_NAME}_${VERSION}_${OS}_${ARCH}"
  BASE="https://github.com/${REPO}/releases/download/v${VERSION}"

  mkdir -p "$INSTALL_DIR"
  TMP="$(mktemp "${INSTALL_DIR}/.${ASSET}.XXXXXX")"
  trap 'rm -f "$TMP"' EXIT

  echo "Downloading ${ASSET} from ${BASE}..." >&2
  curl -fsSL "${BASE}/${ASSET}" -o "$TMP"

  EXPECTED="$(curl -fsSL "${BASE}/${BINARY_NAME}_${VERSION}_checksums.txt" | awk -v f="$ASSET" '$2 == f { print $1 }')"
  if command -v sha256sum >/dev/null 2>&1; then
    ACTUAL="$(sha256sum "$TMP" | awk '{ print $1 }')"
  else
    ACTUAL="$(shasum -a 256 "$TMP" | awk '{ print $1 }')"
  fi
  if [ -z "$EXPECTED" ] || [ "$EXPECTED" != "$ACTUAL" ]; then
    echo "iwdp-mcp: checksum mismatch for ${ASSET}, refusing to run it" >&2
    exit 1
  fi

  chmod 755 "$TMP"
  mv "$TMP" "$BINARY"
  trap - EXIT
  echo "Installed ${ASSET} to ${BINARY}" >&2
fi

exec "$BINARY" "$@"
