# Tech Stack — Oldworld2

> Status: draft scaffold. Items marked `TODO` need a decision; items marked `IDEA` are suggestions, not commitments.

## 1. Language
- **Go** (version: TODO, e.g. latest stable)
- Module name: TODO (e.g. `github.com/<user>/oldworld`)

## 2. Game Engine / Framework
- **raylib** through `github.com/gen2brain/raylib-go` (zlib license; uses cgo): 3D models with skeletal animation (glTF), shaders, render textures, input
- Started on Ebitengine (2D); switched when the art moved to 3D (see game-design.md, Art & Audio). The game logic (world, battles, characters, saving) didn't change
- Packages: `internal/r3d` draws the 3D scenes (toon and outline shaders, ground, characters from parts, held items, animation players); `internal/gfx` is a small 2D layer on raylib for the pixel UI (images, sub-images, scaling, tinting, rectangles); `internal/game` holds the scenes, menus and input

## 3. Supporting Libraries
| Concern | Candidate | Notes |
|---------|-----------|-------|
| Noise | IDEA: `github.com/ojrac/opensimplex-go` or own implementation | Terrain height, moisture, temperature → zone maps |
| Random | stdlib `math/rand/v2` (PCG) | Seeded, deterministic per chunk |
| Prefabs (optional) | Tiled editor + `github.com/lafriks/go-tiled` | Handcrafted building/room pieces stitched into generated maps |
| Text / fonts | own 8×8 bitmap font (`tools/font/makefont.py` → `assets/generated/ui/font8x8.png`) via `internal/gfx` | Dialogue, menus |
| Audio | raylib audio | Music (ogg, streamed, crossfaded) and SFX (wav), made by `tools/audio/makeaudio.py` into `assets/generated/audio`; see `internal/audio` |
| UI widgets | custom | Menus, battle UI |
| Input | raylib (keyboard, gamepad) | Hold durations tracked in `internal/game/input.go` |
| Data files | JSON (stdlib `encoding/json`) | Content (abilities, classes, enemies, items) as data, not code. Separate from save files |
| Save files | SQLite via `modernc.org/sqlite` (pure Go, no cgo) | One autosave file; partial updates per part (position, character, inventory, chunk diffs) in small transactions, WAL mode. See game-design.md 8.1 |
| Testing | stdlib `testing` | Focus on battle logic, damage formulas, generation determinism |

## 4. Architecture

### 4.1 Scene / State Management
- Scene stack or state machine: `TitleScene`, `OverworldScene`, `BattleScene`, `MenuScene`, `DialogueScene`
- Transitions: overworld → battle (pass encounter data) → back to overworld (pass results)

### 4.2 Overworld
- Tile map rendering with camera (only visible chunks drawn)
- Grid movement + collision layer
- Entity system for player, NPCs, triggers (TODO: plain structs vs. ECS)

### 4.3 Procedural Generation
- **Chunk-based infinite world:** world split into fixed-size chunks of 32x32 tiles, addressed by `(cx, cy)`. A zone cell is 16x16 chunks (starting guess)
- **Streaming:** generate/load chunks around the player, unload distant ones (TODO: view radius)
- **Determinism:** each chunk's RNG is seeded from `hash(worldSeed, cx, cy)` so generation is independent of visit order
- **Zone layer (region level):** zones can't come from per-tile noise alone, because they need closed borders with only a few crossings. Instead, zones are cells on a coarse square grid (see [world.md](world.md#22-zone-grid--size)), each assigned a zone type and difficulty, generated before chunks. Barriers inside the cell give each zone an irregular usable shape
  - Each border between two regions becomes an impassable barrier (mountains, cliffs, water, ...)
  - Crossing points on each border are chosen deterministically from `hash(worldSeed, regionA, regionB)`, so both sides agree no matter which is generated first
  - Roads connect crossings to settlements inside each region
  - Texture blending is applied in a narrow band around crossings
- **Pipeline per chunk:** look up region/zone → barriers + crossings + roads → noise layers (height, moisture) for terrain inside the zone → base tiles → features (trees, rocks, water) → points of interest → entity/encounter spawns
- **Cross-chunk structures** (settlements, rivers): placed at region level, then rasterized into the chunks they overlap
- **Interiors / dungeons:** separate generator (rooms + corridors, cellular automata caves, or BSP), seeded from entrance location
- **Background generation:** generate chunks in goroutines ahead of the player to avoid frame hitches
- **Persistence:** generated chunks are never saved in full; only player-caused changes (diffs) are stored per chunk

### 4.4 Battle Engine
- Keep battle logic **pure and decoupled from rendering** (testable, deterministic with seeded RNG)
- Core types: `Combatant`, `Stats`, `Ability`, `Effect`, `StatusEffect`, `BattleState`, `Action`
- Turn resolver → event list → renderer animates events

### 4.5 Data-Driven Content
- Zones, generation parameters, abilities, classes, enemies, item bases/suffixes and encounter tables defined in data files (quest templates later)
- Loaded at startup; validated with tests

## 5. Proposed Project Layout
```
oldworld/
├── cmd/
│   └── game/            # main.go entry point
├── internal/
│   ├── game/            # top-level Game struct, scene manager
│   ├── scene/           # title, overworld, battle, menu scenes
│   ├── world/           # chunks, tiles, collision, camera, entities, chunk streaming
│   ├── procgen/         # noise, zones, terrain, POIs, dungeons (pure, deterministic)
│   ├── battle/          # pure battle logic (no rendering)
│   ├── data/            # loading/validating content files
│   ├── ui/              # menus, dialogue box, HUD widgets
│   ├── audio/           # music/SFX helpers
│   └── save/            # save/load
├── assets/
│   ├── tilesets/        # one tileset per zone (grassland, desert, north, mediterranean, unholy)
│   ├── prefabs/         # optional Tiled .tmx pieces (buildings, rooms)
│   ├── sprites/
│   ├── fonts/
│   ├── audio/
│   └── data/            # zones.json, abilities.json, enemies.json, items.json ...
├── go.mod
└── README.md
```

## 6. Rendering
- Internal resolution: 320x180 (about 20x11 tiles visible), scaled up pixel-perfect by whole numbers (6x on 1080p). TODO: confirm by playtesting
- Tile size: 16 px per tile in world positions (1 tile = 1 m in 3D)
- Target FPS / TPS: 60

## 7. Asset Pipeline & Tools
- Map editor: Tiled (only for optional prefabs; the world itself is generated)
- Debug tooling: `go run ./cmd/mapshot -seed 1 -out map.png` renders a part of the generated world to a PNG without opening a window
- Sprite editor: TODO (Aseprite, LibreSprite, Piskel)
- Embedding assets: `embed` package for release builds?
- No third-party 2D art or audio: the font, music and sound effects are our own (see game-design.md)
- 3D models: made in Blender by scripts in `tools/models` (scenery, all creatures, weapons and shields, the male body and the outfits, the heroes' modelled after the concept art in `art/concept`). Still from the Quaternius packs (CC0, see game-design.md 6.0.2): the animations, and the female body and outfits
- Art generation: stable-diffusion.cpp (Vulkan backend) with SDXL (Lightning); pipeline scripts in `tools/` (see game-design.md, How the art is made)

### 7.1 Running & Testing
- Run the game from the repository root: `go run ./cmd/game` opens the title screen (continue or set up a new game). Command line flags for testing skip it: `-new` starts a new game right away, with `-seed`, `-class warrior|mage`, `-level N` (start at a higher level), `-zone north` (start in the nearest zone of a type: grassland, desert, ancient_desert, north, mediterranean, unholy) and `-at X,Y` (start at a tile); `-title` shows the title screen even in test runs; `-save FILE` uses another save file; `-assets`
- VS Code: Run and Debug view → "Game" (`.vscode/launch.json`, needs the Go extension and Delve). The map preview tool runs from the terminal: `go run ./cmd/mapshot -seed 1 -out map.png`
- Test runs (`-shot`/`-script`) never touch the autosave unless `-save` is given
- Unit tests: `go test ./...` (world generation determinism, chunk borders, water autotiling, player movement)
- Automated visual checks: `tools/testrun.sh out.png [frame] [script]` builds the game, opens it unfocused on a hidden Hyprland special workspace, plays a scripted input (e.g. `right:20,run-down:40`), saves a screenshot and quits. Test runs disable vsync, because a hidden window gets no frame callbacks
- Linux graphics: under native Wayland with display scaling, raylib (GLFW) draws into only part of the window, so `cmd/game` runs through XWayland (it unsets `WAYLAND_DISPLAY`)
- 3D models: after unpacking the Quaternius packs into `assets/quaternius/`, run `python3 tools/models/prepare.py` once (light copies of the models, see game-design.md 6.0.2; it also merges the second set of skin weights the female ranger outfit uses, which raylib ignores). All creatures are made, not downloaded: `blender -b --python tools/models/scarab.py -- assets/generated/creatures/Scarab.gltf [preview_dir]`, likewise `mummy.py -- .../Mummy.gltf` and `snake.py -- cobra .../Cobra.gltf` / `snake.py -- sandsnake .../SandSnake.gltf`, `scorpion.py -- .../Scorpion.gltf`, `guardian.py -- .../TombGuardian.gltf`, `ember.py -- .../EmberSpirit.gltf`, `beasts.py -- wolf .../Wolf.gltf` / `beasts.py -- snowwolf .../SnowWolf.gltf` / `beasts.py -- boar .../Boar.gltf`, `goblin.py -- .../Goblin.gltf`, `troll.py -- .../IceTroll.gltf`, `crab.py -- .../GiantCrab.gltf`, `bats.py -- .../BatSwarm.gltf`, `beasts.py -- direwolf .../DireWolf.gltf`, `ember.py -- .../FrostWisp.gltf "" frost` / `ember.py -- .../WindSpirit.gltf "" wind`. All weapons (steel sword, scimitar, heater, paladin and hoplite shields, spear, two axes, dagger, bow, war hammer, staffs) come from `blender -b --python tools/models/weapons.py -- assets/generated/props`. The map scenery (trees, palms, bushes, plants, flowers, grass, rocks, bones) comes from `blender -b --python tools/models/nature.py -- assets/generated/nature [preview_dir]`. Human bodies, hair and outfit pieces: `blender -b --python tools/models/human.py -- <UAL1_Standard.glb> assets/generated/humans [preview_dir]`; check them in the animation poses with `tools/models/poses.py`

## 8. Build & Distribution
- Targets: Linux first, Windows (one .exe cross-built on Linux with Zig's C compiler: `tools/release/windows.sh`), Web/WASM optional later
- Build tooling: `go build`, Makefile / Taskfile: TODO
- CI: TODO (GitHub Actions: build + test + lint)
- Linting: `gofmt`, `go vet`, `golangci-lint`

## 9. Conventions
- Code style: standard Go (`gofmt`), small packages, no global mutable state where avoidable
- Error handling: TODO
- Logging: `log/slog`
- Version control: git, branch strategy TODO

## 10. Open Questions
- ECS or simple structs?
- Streaming radius?
- Zone cell size in tiles (and how it relates to chunk size)?
- TODO
