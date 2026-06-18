#!/bin/sh
# claudectx installer.
#
#   curl -fsSL https://raw.githubusercontent.com/RaphaelNeumann/claudectx/main/install.sh | sh
#
# Environment overrides:
#   VERSION=v0.1.0      install a specific release tag (default: latest)
#   BINDIR=/usr/local/bin   install directory (default: $HOME/.local/bin)
set -eu

REPO="RaphaelNeumann/claudectx"
BINARY="claudectx"
BINDIR="${BINDIR:-$HOME/.local/bin}"

info() { printf '%s\n' "claudectx-install: $*"; }
err() {
	printf '%s\n' "claudectx-install: error: $*" >&2
	exit 1
}
have() { command -v "$1" >/dev/null 2>&1; }

# --- downloader ---------------------------------------------------------------
if have curl; then
	dl() { curl -fsSL -o "$2" "$1"; }
	redirect_url() { curl -fsSLI -o /dev/null -w '%{url_effective}' "$1"; }
elif have wget; then
	dl() { wget -qO "$2" "$1"; }
	redirect_url() { wget -q -S -O /dev/null "$1" 2>&1 | sed -n 's/.*[Ll]ocation: //p' | tr -d '\r' | tail -1; }
else
	err "need curl or wget"
fi

# --- detect platform ----------------------------------------------------------
os=$(uname -s)
case "$os" in
Darwin) os=darwin ;;
Linux) os=linux ;;
*) err "unsupported OS: $os (macOS only for now)" ;;
esac

arch=$(uname -m)
case "$arch" in
x86_64 | amd64) arch=amd64 ;;
arm64 | aarch64) arch=arm64 ;;
*) err "unsupported architecture: $arch" ;;
esac

# --- resolve version ----------------------------------------------------------
VERSION="${VERSION:-}"
if [ -z "$VERSION" ]; then
	# Follow the /releases/latest redirect to read the tag — no API token, no jq.
	VERSION=$(redirect_url "https://github.com/$REPO/releases/latest" | sed -E 's#.*/tag/##')
fi
case "$VERSION" in
"" | */* | http*) err "could not determine a release tag (are there releases yet?); set VERSION=vX.Y.Z" ;;
esac

# goreleaser strips a leading "v" from archive names ({{ .Version }}).
ver=$(printf '%s' "$VERSION" | sed 's/^v//')
archive="${BINARY}_${ver}_${os}_${arch}.tar.gz"
base="https://github.com/$REPO/releases/download/$VERSION"

info "installing $BINARY $VERSION ($os/$arch)"

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT INT TERM

# --- download -----------------------------------------------------------------
dl "$base/$archive" "$tmp/$archive" ||
	err "download failed: $base/$archive (no prebuilt binary for $os/$arch — macOS only for now)"

# --- verify checksum (best-effort) --------------------------------------------
if dl "$base/checksums.txt" "$tmp/checksums.txt" 2>/dev/null; then
	want=$(awk -v f="$archive" '$2 == f {print $1}' "$tmp/checksums.txt")
	if [ -n "$want" ]; then
		if have shasum; then
			got=$(shasum -a 256 "$tmp/$archive" | awk '{print $1}')
		elif have sha256sum; then
			got=$(sha256sum "$tmp/$archive" | awk '{print $1}')
		else
			got=""
		fi
		if [ -n "$got" ] && [ "$got" != "$want" ]; then
			err "checksum mismatch for $archive"
		fi
		[ -n "$got" ] && info "checksum verified"
	fi
fi

# --- extract ------------------------------------------------------------------
tar -xzf "$tmp/$archive" -C "$tmp"
[ -f "$tmp/$BINARY" ] || err "binary '$BINARY' not found in archive"
chmod +x "$tmp/$BINARY"

# --- install ------------------------------------------------------------------
mkdir -p "$BINDIR"
if mv "$tmp/$BINARY" "$BINDIR/$BINARY" 2>/dev/null; then
	:
elif have sudo; then
	info "writing to $BINDIR requires elevated permissions"
	sudo mv "$tmp/$BINARY" "$BINDIR/$BINARY"
else
	err "cannot write to $BINDIR; re-run with BINDIR set to a writable directory"
fi
info "installed $BINARY to $BINDIR/$BINARY"

# --- PATH hint ----------------------------------------------------------------
case ":$PATH:" in
*":$BINDIR:"*) ;;
*) info "note: $BINDIR is not on your PATH — add it: export PATH=\"$BINDIR:\$PATH\"" ;;
esac

info "done — run '$BINARY' to get started"
