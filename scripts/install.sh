#!/bin/sh
set -eu

REPO=${1:-${ACP_REPO:-podopodo/abs}}
VERSION=${ACP_VERSION:-latest}
INSTALL_DIR=${ACP_INSTALL_DIR:-$HOME/.local/bin}

if [ -z "$REPO" ]; then
  echo "usage: install.sh [OWNER/REPO]" >&2
  echo "or set ACP_REPO=OWNER/REPO (default podopodo/abs)" >&2
  exit 2
fi

case "$(uname -s)" in
  Linux) os=linux ;;
  Darwin) os=darwin ;;
  *) echo "unsupported operating system: $(uname -s)" >&2; exit 1 ;;
esac
case "$(uname -m)" in
  x86_64|amd64) arch=amd64 ;;
  arm64|aarch64) arch=arm64 ;;
  *) echo "unsupported architecture: $(uname -m)" >&2; exit 1 ;;
esac

asset="acp_${os}_${arch}.tar.gz"
if [ "$VERSION" = latest ]; then
  base="https://github.com/${REPO}/releases/latest/download"
else
  base="https://github.com/${REPO}/releases/download/${VERSION}"
fi

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT HUP INT TERM
curl -fL "$base/$asset" -o "$tmp/$asset"
curl -fL "$base/checksums.txt" -o "$tmp/checksums.txt"
(
  cd "$tmp"
  if command -v sha256sum >/dev/null 2>&1; then
    grep "  $asset$" checksums.txt | sha256sum -c -
  else
    expected=$(grep "  $asset$" checksums.txt | awk '{print $1}')
    actual=$(shasum -a 256 "$asset" | awk '{print $1}')
    [ "$expected" = "$actual" ] || { echo "checksum mismatch" >&2; exit 1; }
  fi
  tar -xzf "$asset" acp
)
mkdir -p "$INSTALL_DIR"
install -m 0755 "$tmp/acp" "$INSTALL_DIR/acp"
echo "Installed $INSTALL_DIR/acp"
"$INSTALL_DIR/acp" version
