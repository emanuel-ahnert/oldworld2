#!/usr/bin/env bash
# Draws olive_grove.png, the layout of the Mediterranean's battle background
# (bg_mediterranean in battle.json): a blue sky, the sea on a high horizon with a hill, wide
# open golden ground for the fighters, an olive tree framing the left edge and a broken
# marble column with fallen blocks on the right, a few shadow patches on the ground.
#
#   tools/artgen/layouts/olive_grove.sh
set -euo pipefail
cd "$(dirname "$0")"
magick -size 1024x576 gradient:'#6aa6dc'-'#bcdcf0' \
  -fill '#2f6fa8' -draw "rectangle 0,196 1024,214" \
  -fill '#9aa86a' -draw "ellipse 760,214 300,40 180,360" \
  -fill '#d8c27a' -draw "rectangle 0,214 1024,576" \
  -fill '#c9ae62' -draw "ellipse 300,330 240,26 0,360" -draw "ellipse 700,420 280,30 0,360" -draw "ellipse 160,520 260,34 0,360" \
  -fill '#e6d494' -draw "ellipse 560,300 170,18 0,360" -draw "ellipse 420,480 220,26 0,360" \
  -fill '#7a6e5c' -draw "polygon 60,576 110,330 150,240 190,250 170,340 200,576" \
  -fill '#8c9a68' -draw "ellipse 120,120 210,110 0,360" -draw "ellipse 260,70 140,70 0,360" \
  -fill '#7c8a5c' -draw "ellipse 60,200 120,60 0,360" \
  -fill '#ece8de' -draw "rectangle 880,150 960,500" -draw "rectangle 860,500 990,540" \
  -fill '#d6d0c2' -draw "polygon 880,150 900,120 920,150 940,110 960,150" \
  -fill '#e2ddd0' -draw "ellipse 800,520 50,26 0,360" -draw "rectangle 760,540 860,566" \
  olive_grove.png
echo "wrote $(pwd)/olive_grove.png"
