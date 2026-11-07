#!/usr/bin/env bash
# Runs the game for an automated check without stealing focus: the window opens on a hidden
# Hyprland special workspace, follows a scripted input, saves a screenshot and quits.
#
#   tools/testrun.sh out.png [frame] [script] [extra game flags...]
#   tools/testrun.sh /tmp/shot.png 200 'right:20,run-right:60,down:40'
set -euo pipefail

out=${1:?usage: testrun.sh out.png [frame] [script] [flags...]}
frame=${2:-120}
script=${3:-wait:1}
shift $(( $# < 3 ? $# : 3 ))

cd "$(dirname "$0")/.."
bin=$(mktemp -d)/game
go build -o "$bin" ./cmd/game

if command -v hyprctl >/dev/null; then
  hyprctl eval 'hl.window_rule({ match = { title = "Oldworld2 test run" }, workspace = "special:oldworld silent", no_initial_focus = true })' >/dev/null
fi

rm -f "$out"
timeout 60 "$bin" -shot "$out" -shot-frame "$frame" -script "$script" "$@"
rm -rf "$(dirname "$bin")"
[ -f "$out" ] && echo "screenshot: $out"
