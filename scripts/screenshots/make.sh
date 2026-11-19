#!/usr/bin/env bash
# Copyright (c) 2026 Michael Lechner
# Licensed under the MIT License. See LICENSE file in the project root for full license information.
#
# Renders the product page screenshots from real gitpulse output:
#
#   scripts/screenshots/make.sh BINARY OUTDIR
#
# Builds a demo home (demo.sh) in a temporary directory, runs BINARY there
# with HOME pointing at it — so paths read ~/code/… — and writes:
#
#   terminal-dashboard.svg  gitpulse
#   terminal-add.svg        gitpulse add ~/code (first run)
#   terminal-verbose.svg    gitpulse -v (without dotfiles and invoice-tool)
#   terminal-upstream.svg   fetch third party, --compact, jq one-liner
#   terminal-list.svg       gitpulse ignore old-prototype; gitpulse list
#   terminal-narrow.svg     gitpulse in a 60-column terminal (needs `script`)
#   report-html.png         gitpulse --compact --html (needs chromium)
#
# Nothing goes online: visibility comes from a seeded cache (--offline) and the
# only fetch goes to a local bare repository.
set -euo pipefail

BIN=$(realpath "$1")
OUT=$(realpath -m "$2")
HERE=$(cd "$(dirname "$0")" && pwd)
SVG="python3 $HERE/ansi2svg.py"
REAL_HOME=$HOME

DEMO=$(mktemp -d)
trap 'rm -rf "$DEMO"' EXIT
mkdir -p "$OUT"
bash "$HERE/demo.sh" "$DEMO"

export HOME=$DEMO XDG_CACHE_HOME=$DEMO/.cache CLICOLOR_FORCE=1
cd "$DEMO"
gp() { "$BIN" "$@"; }

# Hide the demo path wherever gitpulse prints an absolute one.
tilde() { sed "s#$DEMO#~#g"; }

{ echo '$ gitpulse add ~/code'; gp add code 2>&1 | tilde; } >add.ansi
$SVG "gitpulse add — first run" "$OUT/terminal-add.svg" <add.ansi

{ echo '$ gitpulse'; gp --offline 2>&1; } >dashboard.ansi
$SVG "gitpulse — terminal" "$OUT/terminal-dashboard.svg" --max-width 84 <dashboard.ansi

{ echo '$ gitpulse -v'; gp --offline -v 2>&1; } >verbose.ansi
$SVG "gitpulse -v — every remote, submodule and branch" "$OUT/terminal-verbose.svg" \
	--max-width 84 --drop-repos dotfiles,invoice-tool --no-summary <verbose.ansi

# Upstream check: the fetch really runs, against the local bare upstream.
W=code/whisper.cpp
url=$(git -C "$W" remote get-url origin)
git -C "$W" remote set-url origin "$DEMO/bare/whisper.git"
gp --offline --json | jq -r '.results[] | select(.third_party) | .path' |
	while read -r dir; do git -C "$dir" fetch --quiet; done
git -C "$W" remote set-url origin "$url"
{
	echo "\$ gitpulse --json | jq -r '.results[] | select(.third_party) | .path' |"
	echo '    while read -r dir; do git -C "$dir" fetch --quiet; done'
	echo
	echo '$ gitpulse --compact'
	# table only: drop the summary block (compared without colour codes)
	gp --offline --compact 2>&1 | perl -ne '($p = $_) =~ s/\e\[[0-9;]*m//g; $s = 1 if $p =~ /^SUMMARY:/; print unless $s || $p =~ /^─{40}$/'
	echo
	echo "\$ gitpulse --json | jq -r '.results[] | select(.third_party and .behind > 0)"
	echo "    | \"\\(.name): \\(.behind) new — \\(.incoming_commits[0])\"'"
	gp --offline --json | jq -r '.results[] | select(.third_party and .behind > 0)
		| "\(.name): \(.behind) new — \(.incoming_commits[0])"'
} >upstream.ansi
$SVG "Third-party checkouts: has upstream moved on?" "$OUT/terminal-upstream.svg" --max-width 86 <upstream.ansi

# A repository that is not in the config yet, for the ignore picture.
git init -q -b main code/old-prototype
echo sketch >code/old-prototype/sketch.txt
git -C code/old-prototype add -A
GIT_AUTHOR_NAME=Demo GIT_AUTHOR_EMAIL=demo@example.com GIT_COMMITTER_NAME=Demo GIT_COMMITTER_EMAIL=demo@example.com \
	git -C code/old-prototype -c commit.gpgsign=false commit -q -m Sketch
{
	echo '$ gitpulse ignore old-prototype'
	(cd code && gp ignore old-prototype 2>&1 | tilde)
	echo
	echo '$ gitpulse list'
	gp list 2>&1 | tilde
} >list.ansi
$SVG "gitpulse ignore / list — managing projects" "$OUT/terminal-list.svg" <list.ansi

if script -qec true /dev/null >/dev/null 2>&1; then
	# 60 columns: the demo table is narrower than most real ones, so this is
	# where it switches to two lines per repository.
	{ echo '$ gitpulse   # 60 columns'; script -qec "stty cols 60; '$BIN' --offline" /dev/null | tr -d '\r'; } >narrow.ansi
	$SVG "gitpulse in a 60-column terminal" "$OUT/terminal-narrow.svg" --no-summary <narrow.ansi
else
	echo "skipped terminal-narrow.svg: needs util-linux script" >&2
fi

CHROME=$(command -v chromium || command -v chromium-browser || command -v google-chrome || true)
if [ -n "$CHROME" ]; then
	# A snap browser only reads non-hidden folders below the real home directory.
	shot=$(mktemp -d "$REAL_HOME/gitpulse-shot.XXXXXX")
	trap 'rm -rf "$DEMO" "$shot"' EXIT
	gp --offline --compact --html >"$shot/report.html"
	HOME=$REAL_HOME "$CHROME" --headless=new --disable-gpu --no-sandbox --hide-scrollbars \
		--window-size=760,480 --screenshot="$shot/report.png" "file://$shot/report.html" >/dev/null 2>&1
	cp "$shot/report.png" "$OUT/report-html.png"
	echo "$OUT/report-html.png"
else
	echo "skipped report-html.png: needs chromium" >&2
fi
