#!/usr/bin/env bash
# Builds the Windows release on Linux: one .exe with the assets inside, which it unpacks to the
# user's cache folder on the first start (internal/assets/bundle.go). raylib is C, so this needs
# a C cross compiler: Zig's, installed for this user by mise.
#
#   tools/release/windows.sh [OUT.exe]
set -euo pipefail
cd "$(dirname "$0")/../.."
out=${1:-dist/oldworld2-windows-amd64.exe}
zig_version=0.15.2

mise install "zig@$zig_version" >/dev/null 2>&1
zig="$(mise where "zig@$zig_version")/zig"

bundle=internal/assets/bundle.zip
trap 'rm -f "$bundle"' EXIT
python3 tools/release/bundle.py "$bundle"

mkdir -p "$(dirname "$out")"
# No undefined behavior sanitizer: Zig turns it on by default, and it would end the game on
# harmless things in the C libraries. Zig ignores Go's -H windowsgui, so the linker gets told
# directly that there's no console window.
CGO_ENABLED=1 GOOS=windows GOARCH=amd64 \
  CC="$zig cc -target x86_64-windows-gnu" CXX="$zig c++ -target x86_64-windows-gnu" \
  CGO_CFLAGS="-O2 -g0 -fno-sanitize=undefined -Wno-unknown-warning-option -Wno-tautological-compare" \
  go build -tags bundle -trimpath -ldflags "-s -w -H windowsgui -extldflags=-Wl,--subsystem,windows" -o "$out" ./cmd/game
ls -lh "$out"
