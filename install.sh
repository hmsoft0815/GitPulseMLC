#!/bin/sh
# Install the latest gitpulse release on Linux or macOS.
#
#   curl -fsSL https://raw.githubusercontent.com/hmsoft0815/GitPulseMLC/main/install.sh | sh
#
# Environment:
# Linux binaries come from the GitHub release, macOS binaries from mlcgo.eu,
# where they are signed and notarized (GitHub only carries Linux and Windows).
#
#   GITPULSE_VERSION   Linux: install this version (e.g. 1.3.0) instead of the latest
#   GITPULSE_DIR       install directory (default: /usr/local/bin if writable, else ~/.local/bin)
set -eu

REPO="hmsoft0815/GitPulseMLC"
MAC_DOWNLOADS="https://mlcgo.eu/downloads/gitpulse"

fail() { echo "install.sh: $*" >&2; exit 1; }

need() { command -v "$1" >/dev/null 2>&1 || fail "$1 is required"; }

fetch() {
	if command -v curl >/dev/null 2>&1; then
		curl -fsSL "$1" -o "$2"
	elif command -v wget >/dev/null 2>&1; then
		wget -qO "$2" "$1"
	else
		fail "curl or wget is required"
	fi
}

detect_platform() {
	case "$(uname -s)" in
		Linux) os=linux ;;
		Darwin) os=darwin ;;
		*) fail "unsupported OS $(uname -s) — download a release from https://github.com/$REPO/releases" ;;
	esac
	case "$(uname -m)" in
		x86_64 | amd64) arch=amd64 ;;
		arm64 | aarch64) arch=arm64 ;;
		*) fail "unsupported architecture $(uname -m)" ;;
	esac
}

latest_version() {
	fetch "https://api.github.com/repos/$REPO/releases/latest" "$tmp/latest.json"
	sed -n 's/.*"tag_name": *"v\{0,1\}\([^"]*\)".*/\1/p' "$tmp/latest.json" | head -n 1
}

# verify FILE SUMSFILE — compare FILE (in $tmp) against its line in SUMSFILE.
verify() {
	expected=$(grep " \*\{0,1\}$1\$" "$tmp/$2" | cut -d ' ' -f 1)
	[ -n "$expected" ] || fail "no checksum for $1"
	if command -v sha256sum >/dev/null 2>&1; then
		actual=$(sha256sum "$tmp/$1" | cut -d ' ' -f 1)
	else
		actual=$(shasum -a 256 "$tmp/$1" | cut -d ' ' -f 1)
	fi
	[ "$expected" = "$actual" ] || fail "checksum mismatch for $1"
}

install_dir() {
	if [ -n "${GITPULSE_DIR:-}" ]; then
		echo "$GITPULSE_DIR"
	elif [ -w /usr/local/bin ]; then
		echo /usr/local/bin
	else
		echo "$HOME/.local/bin"
	fi
}

download_linux() {
	need tar
	version=${GITPULSE_VERSION:-$(latest_version)}
	[ -n "$version" ] || fail "could not determine the latest version"
	archive="gitpulse_${version}_${os}_${arch}.tar.gz"
	base="https://github.com/$REPO/releases/download/v$version"
	echo "Downloading gitpulse $version ($os/$arch) ..."
	fetch "$base/$archive" "$tmp/$archive"
	fetch "$base/checksums.txt" "$tmp/checksums.txt"
	verify "$archive" checksums.txt
	tar -xzf "$tmp/$archive" -C "$tmp" gitpulse
}

download_macos() {
	[ -z "${GITPULSE_VERSION:-}" ] || echo "Note: GITPULSE_VERSION is ignored on macOS — mlcgo.eu serves the latest signed build."
	file="gitpulse-darwin-$arch"
	echo "Downloading signed gitpulse ($os/$arch) from mlcgo.eu ..."
	fetch "$MAC_DOWNLOADS/$file" "$tmp/$file"
	fetch "$MAC_DOWNLOADS/SHA256SUMS" "$tmp/SHA256SUMS"
	verify "$file" SHA256SUMS
	mv "$tmp/$file" "$tmp/gitpulse"
}

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

detect_platform
if [ "$os" = darwin ]; then
	download_macos
else
	download_linux
fi

dir=$(install_dir)
mkdir -p "$dir"
mv "$tmp/gitpulse" "$dir/gitpulse"
chmod +x "$dir/gitpulse"

echo "Installed $("$dir/gitpulse" --version) to $dir/gitpulse"
case ":$PATH:" in
	*":$dir:"*) ;;
	*) echo "Note: $dir is not in your PATH — add it, or call $dir/gitpulse directly." ;;
esac
echo
echo "Next: gitpulse init ~/code    # find your repositories"
echo "      gitpulse                # see what needs attention"
