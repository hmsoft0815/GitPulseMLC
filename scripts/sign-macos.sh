#!/usr/bin/env bash
# Sign and notarize macOS binaries on the Mac build host, in place.
#
#   scripts/sign-macos.sh bin/gitpulse-darwin-arm64 ...
#
# Needs MAC_HOST, SIGN_ID and NOTARY_PROFILE (from .release.env via the
# Taskfile, see .release.env.example) and MAC_KEYCHAIN_PASS, the login password
# of the Mac, exported in the shell.
#
# The binaries are copied to the Mac, signed with the Developer ID (hardened
# runtime, secure timestamp), submitted to Apple's notary service in one zip
# and copied back over the originals. A bare Mach-O binary cannot carry a
# stapled ticket — that only works for .app, .dmg and .pkg — so Gatekeeper
# looks the ticket up online on first launch. That is enough for downloads.
set -euo pipefail

[ $# -gt 0 ] || { echo "usage: $0 BINARY ..." >&2; exit 2; }
for v in MAC_HOST SIGN_ID NOTARY_PROFILE; do
	[ -n "${!v:-}" ] || { echo "$v is not set — copy .release.env.example to .release.env and fill it in." >&2; exit 1; }
done
for f in "$@"; do
	[ -f "$f" ] || { echo "not found: $f" >&2; exit 1; }
done
# Without an unlocked keychain codesign fails with "User interaction is not
# allowed" — check before anything is copied.
if [ -z "${MAC_KEYCHAIN_PASS:-}" ]; then
	echo "MAC_KEYCHAIN_PASS is not set — the keychain on $MAC_HOST stays locked." >&2
	echo "  export MAC_KEYCHAIN_PASS='<login password of the Mac>'" >&2
	exit 1
fi

remote_dir=$(ssh "$MAC_HOST" 'mktemp -d /tmp/gitpulse-sign.XXXXXX')
trap 'ssh "$MAC_HOST" "rm -rf $remote_dir"' EXIT

echo "→ Copying $# binaries to $MAC_HOST"
scp -q "$@" "$MAC_HOST:$remote_dir/"

names=()
for f in "$@"; do names+=("$(basename "$f")"); done

# The password travels as the first line of stdin, never on a command line,
# so it does not show up in the process list here or on the Mac.
{
	printf '%s\n' "$MAC_KEYCHAIN_PASS"
	cat <<ENDSSH
set -euo pipefail
cd "$remote_dir"
security unlock-keychain -p "\$KEYCHAIN_PASS" ~/Library/Keychains/login.keychain-db
for f in ${names[*]}; do
	echo "→ Signing \$f"
	codesign --force --options runtime --timestamp --sign "$SIGN_ID" "\$f"
	codesign --verify --strict --verbose=2 "\$f"
done
echo "→ Notarizing (this takes a few minutes)"
mkdir notarize && cp ${names[*]} notarize/
ditto -c -k notarize notarize.zip
xcrun notarytool submit notarize.zip --keychain-profile "$NOTARY_PROFILE" --wait | tee notary.log
grep -q "status: Accepted" notary.log || { echo "notarization was not accepted" >&2; exit 1; }
ENDSSH
} | ssh "$MAC_HOST" "bash -c 'IFS= read -r KEYCHAIN_PASS; export KEYCHAIN_PASS; exec bash -s'"

echo "→ Copying signed binaries back"
for f in "$@"; do
	scp -q "$MAC_HOST:$remote_dir/$(basename "$f")" "$f"
done
echo "✓ Signed and notarized: $*"
