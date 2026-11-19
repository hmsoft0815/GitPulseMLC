#!/usr/bin/env bash
# Copyright (c) 2026 Michael Lechner
# Licensed under the MIT License. See LICENSE file in the project root for full license information.
#
# Builds the demo home for the product page screenshots: seven repositories
# below $1/code in the states the pictures show, a config in
# $1/.gitpulsemlc/repos.ini and a visibility cache, so gitpulse runs fully
# offline. Remotes point at fake hosts; the upstreams are bare repositories
# in $1/bare.
#
#   api-server    gitlab, private, 2 commits not pushed
#   blog          github, public, a branch untouched since August (stale)
#   docs-site     github, private, submodule .mlcai one commit ahead (sub-ptr)
#   dotfiles      no remote, clean
#   invoice-tool  LAN server, one stash
#   webshop       github, public, second remote on the LAN, dirty files
#   whisper.cpp   third party, patched locally, 14 new commits upstream
set -euo pipefail

D=$1
C=$D/code B=$D/bare
mkdir -p "$C" "$B" "$D/.gitpulsemlc" "$D/.cache/gitpulse"

export GIT_AUTHOR_NAME=Demo GIT_AUTHOR_EMAIL=demo@example.com
export GIT_COMMITTER_NAME=Demo GIT_COMMITTER_EMAIL=demo@example.com
GIT_AUTHOR_DATE=$(date -d '-2 days' -Iseconds 2>/dev/null || date -v-2d +%Y-%m-%dT%H:%M:%S)
export GIT_AUTHOR_DATE GIT_COMMITTER_DATE=$GIT_AUTHOR_DATE

g() { git -c init.defaultBranch=main -c protocol.file.allow=always -c commit.gpgsign=false "$@"; }
commit() { echo "$2" >>"$1/$2.txt"; g -C "$1" add -A; g -C "$1" commit -q -m "$3"; }
upstreamed() { # bare upstream + clone with one pushed commit
	g init -q --bare "$B/$1.git"
	g clone -q "$B/$1.git" "$C/$1" 2>/dev/null
	commit "$C/$1" init "Initial commit"
	g -C "$C/$1" push -q -u origin main
}

upstreamed api-server
commit "$C/api-server" limit "Add rate limiting"
commit "$C/api-server" retry "Return 429 with Retry-After"
g -C "$C/api-server" remote set-url origin git@gitlab.com:me-team/api-server.git

upstreamed blog
g -C "$C/blog" checkout -q -b draft-redesign
echo x >"$C/blog/theme.css"
g -C "$C/blog" add -A
GIT_AUTHOR_DATE=2026-08-01T10:00:00 GIT_COMMITTER_DATE=2026-08-01T10:00:00 g -C "$C/blog" commit -q -m "WIP redesign"
g -C "$C/blog" checkout -q main
g -C "$C/blog" remote set-url origin https://github.com/me/blog.git

g init -q --bare "$B/mlcai.git"
g clone -q "$B/mlcai.git" "$D/mlcai-seed" 2>/dev/null
commit "$D/mlcai-seed" PLAN "Plan"
g -C "$D/mlcai-seed" push -q -u origin main
upstreamed docs-site
g -C "$C/docs-site" submodule add -q "$B/mlcai.git" .mlcai
g -C "$C/docs-site" commit -q -m "Add .mlcai"
g -C "$C/docs-site" push -q
commit "$C/docs-site/.mlcai" WORKLOG "Worklog"
g -C "$C/docs-site/.mlcai" push -q origin HEAD:main
g -C "$C/docs-site/.mlcai" fetch -q
g -C "$C/docs-site" remote set-url origin https://github.com/me/docs-site.git

g init -q "$C/dotfiles"
commit "$C/dotfiles" bashrc "bashrc"

upstreamed invoice-tool
echo wip >"$C/invoice-tool/pdf.go"
g -C "$C/invoice-tool" add -A
g -C "$C/invoice-tool" stash -q
g -C "$C/invoice-tool" remote set-url origin nas.local:/volume1/repos/invoice-tool.git

upstreamed webshop
g -C "$C/webshop" remote add nas nas.local:/volume1/repos/webshop.git
echo "a{}" >"$C/webshop/cart.css"
echo "package shop" >"$C/webshop/cart.go"
echo "package shop" >"$C/webshop/order.go"
g -C "$C/webshop" remote set-url origin https://github.com/me/webshop.git

g init -q --bare "$B/whisper.git"
g clone -q "$B/whisper.git" "$D/whisper-up" 2>/dev/null
commit "$D/whisper-up" README "Initial commit"
g -C "$D/whisper-up" push -q -u origin main
g clone -q "$B/whisper.git" "$C/whisper.cpp" 2>/dev/null
for i in $(seq 1 13); do commit "$D/whisper-up" "f$i" "ggml: speed up kernel $i"; done
commit "$D/whisper-up" vad "Add voice activity detection to the server"
g -C "$D/whisper-up" push -q
g -C "$C/whisper.cpp" fetch -q # as after a fetch: the 14 commits are known
echo "// patched" >"$C/whisper.cpp/main.cpp"
echo "// patched" >"$C/whisper.cpp/server.cpp"
g -C "$C/whisper.cpp" remote set-url origin https://github.com/ggerganov/whisper.cpp.git

cat >"$D/.gitpulsemlc/repos.ini" <<EOF
[general]
replace_path_prefix = $D
replace_with        = ~

[settings]
own_remotes = github.com/me, gitlab.com/me-team

[projects]
EOF

# Visibility as the online check would find it; gitpulse runs with --offline.
now=$(date -u +%Y-%m-%dT%H:%M:%SZ)
q='/info/refs?service=git-upload-pack'
cat >"$D/.cache/gitpulse/visibility.json" <<EOF
{
  "https://github.com/me/blog.git$q": {"visibility": "public", "checked_at": "$now"},
  "https://github.com/me/webshop.git$q": {"visibility": "public", "checked_at": "$now"},
  "https://github.com/me/docs-site.git$q": {"visibility": "private", "checked_at": "$now"},
  "https://gitlab.com/me-team/api-server.git$q": {"visibility": "private", "checked_at": "$now"}
}
EOF
