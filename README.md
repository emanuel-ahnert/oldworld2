# Oldworld2

A top-down, turn-based RPG in Go. You explore an endless, procedurally generated overworld
tile by tile; enemies you can see notice you and give chase, and a fight switches to a
separate battle view where your one hero fights them turn by turn with spells and
abilities. Win XP and money, learn abilities, and push outward into harder zones.

The world is 3D (raylib) with a comic look: toon shading and ink outlines, painted battle
backgrounds, our own pixel font, music and sounds.

## Features

- **Nine classes:** Warrior, Mage, Rogue, Hunter, Warlock, Priest, Paladin, Shaman and
  Druid, each with its own types, stats and abilities learned by level.
- **An endless seeded world** of zones laid out on a grid: Grassland at the center, the
  North, the Mediterranean, the Unholy lands, the Middle Eastern Desert and the Ancient
  (Egyptian) Desert, each with its
  own ground, scenery, settlements, music, names and enemies. Difficulty rises by one per
  zone outward.
- **Settlements** with an inn (rest and checkpoint), roads between zones, shrines along
  them.
- **Battles** with elements and types, physical and magical abilities, status effects,
  stat stages, priority moves and elite enemies.
- **Autosave** to a local database; on defeat you wake up at the last checkpoint.

## Running

Needs Go (see `go.mod`) and a C compiler, since raylib is used through cgo
([raylib-go](https://github.com/gen2brain/raylib-go); on Linux also the X11 and OpenGL
development headers it lists).

The heroes' animations, and the female body and outfits, still come from free
[Quaternius](https://quaternius.com) packs (CC0), which are not in the repository. Download
the "Standard" versions of **Universal Animation Library**, **Universal Base Characters**
and **Modular Character Outfits – Fantasy**, unpack them into `assets/quaternius/`, then run
once:

    python3 tools/models/prepare.py

Then, from the repository root:

    go run ./cmd/game

This opens the title screen: continue the saved game, or set up a new one (class, start
zone, world seed). Saves go to `~/.local/share/oldworld2/save.db`.

Useful flags for testing: `-new` starts a new game right away, with `-class rogue`,
`-zone north`, `-seed 42`, `-level 6` or `-at X,Y`; `-mute` turns the sound off. See
`go run ./cmd/game -h` for all of them.

## Controls

| Key | On the map | In battle | In menus |
|-----|------------|-----------|----------|
| WASD / arrow keys | Move | Choose | Move the cursor |
| Shift (hold) | Run | | |
| Space | Interact (inn, shrines) | Confirm | Confirm |
| B | | Back | |
| N | Abilities menu | | Close |
| C | Camp and rest | | |
| Escape | Main menu | | Back |

A gamepad works too: the D-pad moves, A is Space.

## Development

    go test ./...        # unit tests: world generation, battles, saving, movement…
    go vet ./...
    go run ./cmd/mapshot -seed 1 -out map.png           # the generated world as a map image
    tools/testrun.sh out.png 120 'right:20,run-down:40'  # scripted run, saves a screenshot

`tools/testrun.sh` opens the game without stealing focus (on a hidden Hyprland workspace),
plays a scripted input and saves a screenshot.

    tools/release/windows.sh    # the Windows release, built on Linux: dist/oldworld2-windows-amd64.exe

The release is one .exe with the assets inside. It unpacks them to `%LOCALAPPDATA%\oldworld2`
on the first start and saves to `%AppData%\oldworld2\save.db`. The build needs the Quaternius
packs and [mise](https://mise.jdx.dev), which installs the Zig C cross compiler for raylib. If
the game starts loading another pack model, add it to `tools/release/pack_models.txt` (a test
checks this).

Almost all art is made by scripts in this repository:

- **3D models** (bodies, outfits, creatures, weapons, scenery) by Blender scripts in
  `tools/models` (`blender -b --python tools/models/human.py -- …`), writing to
  `assets/generated/`. The heroes and human enemies are modelled after the concept art in
  `art/concept`.
- **Buildings** are built in code from simple shapes (`internal/game/buildings.go`).
- **Textures, battle backgrounds and concept art** are generated locally with
  stable-diffusion.cpp (SDXL Lightning) by `tools/artgen/artgen.py` from the prompt files
  next to it.
- **Music and sound effects** are synthesized by `tools/audio/makeaudio.py`; the pixel font
  is drawn by `tools/font/makefont.py`.

## Project layout

| Path | Contents |
|------|----------|
| `cmd/game` | The game |
| `cmd/mapshot` | Renders part of the world to a PNG, without a window |
| `internal/game` | Game states: title, map, battles, menus, HUD, the 3D scene, sound |
| `internal/battle` | The battle rules: abilities, types, damage, status effects |
| `internal/character` | Classes, levels and the hero |
| `internal/creature` | Enemy kinds: stats, abilities, behavior |
| `internal/procgen` | World generation: zones, terrain, settlements, roads, enemy spawns, names |
| `internal/world` | The loaded world: chunks, the player, enemies moving and chasing |
| `internal/r3d` | 3D rendering on raylib: toon shading, outlines, characters, the ground |
| `internal/save` | The save file |
| `design/` | The design documents, starting with [game-design.md](design/game-design.md) |
| `tools/` | The asset scripts (models, art, audio, font), the test runner and the release build |
| `art/concept` | Concept art the models follow |
