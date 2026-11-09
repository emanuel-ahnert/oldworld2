#!/usr/bin/env bash
# Builds the Linux release: one executable with the assets inside, which it unpacks to
# ~/.cache/oldworld2 on the first start (internal/assets/bundle.go). It is built with Zig's C
# compiler (installed for this user by mise) against glibc 2.31, so it also runs on older
# distributions than the one it was built on (Ubuntu 20.04, Debian 11 and newer). The window
# libraries are loaded at runtime: the player's own, so linking may skip the newer glibc
# symbols that this machine's copies of them use.
#
#   tools/release/linux.sh [OUT]
set -euo pipefail
cd "$(dirname "$0")/../.."
out=${1:-dist/oldworld2-linux-amd64}
zig_version=0.15.2
target=x86_64-linux-gnu.2.31

mise install "zig@$zig_version" >/dev/null 2>&1
zig="$(mise where "zig@$zig_version")/zig"

bundle=internal/assets/bundle.zip
trap 'rm -f "$bundle"' EXIT
python3 tools/release/bundle.py "$bundle"

mkdir -p "$(dirname "$out")"
# The system's X11, Wayland and xkbcommon headers come after Zig's own glibc headers. No
# undefined behavior sanitizer: Zig turns it on by default, and it would end the game on
# harmless things in the C libraries.
CGO_ENABLED=1 GOOS=linux GOARCH=amd64 \
  CC="$zig cc -target $target" CXX="$zig c++ -target $target" \
  CGO_CFLAGS="-O2 -g0 -fno-sanitize=undefined -Wno-unknown-warning-option -Wno-tautological-compare -idirafter /usr/include" \
  CGO_LDFLAGS="-L/usr/lib -Wl,--allow-shlib-undefined" \
  go build -tags bundle -trimpath -ldflags "-s -w" -o "$out" ./cmd/game
ls -lh "$out"
