#!/usr/bin/env bash
# Makes snow_clearing.png, the layout of the North's battle background (bg_north in
# battle.json), from clearing.png: the green ground below the forest band turns to snow with
# pale blue shadows, then rough shapes give the ground something to paint: drift shadows,
# grey rocks poking through, tufts of dry grass and a fallen branch.
#
#   tools/artgen/layouts/snow_clearing.sh
set -euo pipefail
cd "$(dirname "$0")"
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

magick clearing.png -depth 8 "$tmp/base.png"
# The ground: green and bright, below the dark band of the forest (y > 92).
magick "$tmp/base.png" -blur 0x3 -fx "(g>r*1.2 && g>0.33 && j>92)?1:0" -colorspace gray -blur 0x1.5 -threshold 50% "$tmp/mask.png"
magick "$tmp/base.png" -blur 0x3 -colorspace gray -level 25%,65% +level-colors '#b4c6e0','#f6f8fc' "$tmp/snow.png"
magick "$tmp/base.png" "$tmp/snow.png" "$tmp/mask.png" -composite "$tmp/snowy.png"

drift='#a9bcd8'
rock='#7c8088'
grass='#b89a5a'
magick "$tmp/snowy.png" \
  -fill "$drift" -draw "ellipse 250,330 170,14 0,360" -draw "ellipse 620,300 120,10 0,360" \
  -draw "ellipse 470,430 200,16 0,360" -draw "ellipse 150,470 130,12 0,360" -draw "ellipse 700,500 160,14 0,360" \
  -draw "ellipse 420,540 230,18 0,360" \
  -fill "$rock" -draw "ellipse 120,400 26,14 0,360" -draw "ellipse 560,370 18,10 0,360" \
  -draw "ellipse 330,505 30,16 0,360" -draw "ellipse 760,440 22,12 0,360" -draw "ellipse 600,555 34,18 0,360" \
  -fill '#f6f8fc' -draw "ellipse 120,392 22,6 0,360" -draw "ellipse 330,496 25,7 0,360" -draw "ellipse 600,545 28,8 0,360" \
  -stroke "$grass" -strokewidth 3 \
  -draw "line 200,420 196,398" -draw "line 206,420 210,396" -draw "line 212,421 220,402" \
  -draw "line 480,350 476,330" -draw "line 486,350 490,328" \
  -draw "line 690,395 686,372" -draw "line 696,396 702,374" -draw "line 702,396 710,380" \
  -draw "line 60,520 56,496" -draw "line 66,520 70,498" \
  -draw "line 520,480 516,458" -draw "line 526,480 532,460" \
  -stroke '#5e4632' -strokewidth 6 -draw "line 250,560 420,575" -strokewidth 3 -draw "line 330,566 350,548" -draw "line 380,571 404,556" \
  snow_clearing.png
echo "wrote $(pwd)/snow_clearing.png"
