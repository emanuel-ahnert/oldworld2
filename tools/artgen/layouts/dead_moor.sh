#!/usr/bin/env bash
# Draws dead_moor.png, the layout of the unholy lands' battle background (bg_unholy in
# battle.json), to the battle camera (internal/game/scene.go): the horizon a quarter of the way
# down (row 148 of 576), open misty ground below it where the fighters stand (the enemy at
# about x 640, y 288, the hero lower left), big dead trunks only at the left and right edges,
# gravestones away from the fighters, a red moon in a violet sky over a low line of dead trees.
#
#   tools/artgen/layouts/dead_moor.sh
set -euo pipefail
cd "$(dirname "$0")"
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
magick -size 1024x576 gradient:'#3a2a4a'-'#8a6a8c' \
  -fill '#c03838' -draw "circle 600,70 600,118" \
  -fill '#3c3444' -draw "polygon 0,148 60,128 120,140 200,118 260,138 360,124 440,140 520,122 600,140 700,120 780,138 860,124 940,140 1024,126 1024,150 0,150" \
  -fill '#6e6474' -draw "rectangle 0,148 1024,576" \
  -fill '#7c7280' -draw "ellipse 520,250 360,40 0,360" -draw "ellipse 380,420 420,60 0,360" \
  -fill '#5e5664' -draw "ellipse 700,340 200,26 0,360" -draw "ellipse 200,520 260,40 0,360" \
  -fill '#2a2230' -draw "polygon 0,576 0,0 70,0 110,120 90,300 120,576" -draw "polygon 1024,576 1024,0 950,0 900,140 930,320 900,576" \
  -fill '#2a2230' -draw "polygon 70,40 260,0 300,10 110,80" -draw "polygon 950,40 760,0 720,12 910,90" \
  -fill '#8c8492' -draw "roundrectangle 190,190 216,232 8,8" -draw "roundrectangle 236,200 258,236 7,7" \
  -draw "rectangle 820,180 832,226" -draw "rectangle 808,192 844,202" -draw "roundrectangle 160,330 196,384 10,10" \
  "$tmp/flat.png"
# Texture for the painting to pick up: noise over everything, and short dark strokes of dead
# grass scattered over the ground, denser toward the bottom.
strokes=()
for i in $(seq 0 140); do
  x=$(( (i * 7919) % 1024 )); y=$(( 160 + (i * 104729) % 416 ))
  h=$(( 6 + (y - 150) / 18 ))
  strokes+=(-draw "line $x,$y $((x + (i % 5) - 2)),$((y - h))")
done
magick "$tmp/flat.png" -attenuate 0.6 +noise Gaussian -blur 0x1 -stroke '#4c4452' -strokewidth 2 "${strokes[@]}" dead_moor.png
echo "wrote $(pwd)/dead_moor.png"
