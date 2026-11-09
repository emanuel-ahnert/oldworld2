#!/usr/bin/env bash
# Retakes the README screenshots in docs/screenshots/: test runs (tools/testrun.sh, no focus,
# no sound) at fixed tiles of world seed 1, each saved at a chosen frame, then scaled down to
# JPEG. Battles are seeded, so the same frame shows the same moment. Scripted battles always
# use the first usable ability, so -moves lists the one to show first. -zone sets the home
# zone, one zone away from the fight, so the enemies are levels 6 to 10 instead of 1 to 5.
#
#   tools/screenshots.sh          all of them
#   tools/screenshots.sh mage     only those whose name contains "mage"
#
# Needs ImageMagick (magick).
set -euo pipefail
cd "$(dirname "$0")/.."
out=docs/screenshots
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
mkdir -p "$out"

# name frame script flags...
shot() {
  local name=$1 frame=$2 script=$3
  shift 3
  [[ $name == *"${filter:-}"* ]] || return 0
  tools/testrun.sh "$tmp/$name.png" "$frame" "$script" -new "$@" >/dev/null
  magick "$tmp/$name.png" -resize 1280x -quality 85 "$out/$name.jpg"
  echo "$out/$name.jpg"
}
filter=${1:-}

fight='up:10,wait:900' # step into the enemy north of the start tile, then let the battle run

# Battles: every class, every zone.
shot battle-mage-ice-block 290 "$fight" -class mage -level 10 -moves ice_block,frostfire_bolt,frost_ring,blink -zone north -at -13,45
shot battle-shaman-totems 425 "$fight" -class shaman -level 7 -moves chain_lightning,storm_bolt,flame_shock,healing_wave -zone grassland -at -540,-530
shot battle-rogue-cheap-shot 265 "$fight" -class rogue -level 9 -moves cheap_shot,rogue_quick_stab,rogue_smoke_veil,rogue_poisoned_blade -zone grassland -at -546,-530
shot battle-paladin-judgment 265 "$fight" -class paladin -level 5 -moves judgment_hammer,righteous_strike,holy_seal,devotion -zone grassland -at 539,23
shot battle-priest-holy-burst 380 "$fight" -class priest -level 6 -moves holy_burst,smite,mend,ward_of_light -zone grassland -at 500,-74
shot battle-druid-moonbeam 265 "$fight" -class druid -level 12 -moves moonbeam,thorn_lash,insect_swarm,bear_form -zone desert -at -1069,1517
shot battle-warrior-charge 235 "$fight" -class warrior -level 8 -moves charge,heavy_strike,battle_cry,overwhelm -zone desert -at -572,526
shot battle-warlock-hellfire 395 "$fight" -class warlock -level 7 -moves hellfire_rain,drain_life,wither,curse_of_weakness -zone desert -at -548,454
shot battle-hunter-volley 270 "$fight" -class hunter -level 8 -moves volley,steady_shot,mark_prey,concussive_shot -zone grassland -at -21,405

# The adventure map, with the zone's name shown on arrival.
shot map-grassland 20 'wait:30' -class paladin -level 2 -at 2,-6
shot map-desert 20 'wait:30' -class rogue -level 4 -zone desert -at 12,434
shot map-ancient-desert 20 'wait:30' -class druid -level 12 -zone ancient_desert -at -1012,1426
