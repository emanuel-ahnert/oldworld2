# Game Design Document — Oldworld2

> Status: draft scaffold. Items marked `TODO` need a decision; items marked `IDEA` are suggestions, not commitments.

This is the overview. Details live in separate files:

| Document | Contents |
|----------|----------|
| [mechanics.md](mechanics.md) | Battle system, damage formula, elements & types, status effects, HP/MP, checkpoints & death, leveling & XP |
| [world.md](world.md) | World generation, zones, zone placement & borders, enemies, encounter tables |
| [classes.md](classes.md) | Classes, class matchups, abilities & how they are learned, ability pools |
| [items.md](items.md) | Itemization (white to epic), equipment slots, armor & weapon types, consumables, loot, currency & economy |
| [tech-stack.md](tech-stack.md) | Language, engine, architecture, project layout, running & testing |

## 1. Vision

- **One-line pitch:** A top-down, turn-based RPG where you explore an overworld and fight encounters in a separate battle view. A single main character fights enemy NPCs directly with spells and abilities (no captured creatures like in Pokemon, no party).
- **Inspirations:** Creature-collector RPGs (overworld + battle screen loop), classic JRPGs, roguelikes / procedural exploration games, World of Warcraft (classic) for classes and itemization.
- **Target player:** kinda competent player, game starts easy as a warmup and gets a bit more difficult over time.
- **Platform(s):** Linux desktop first, Windows later, web optional later via WASM
- **Session length / total playtime target:** infinite, random generated world
- **Death model:** one persistent world. On defeat the main character respawns at the last checkpoint (Pokemon style) and loses money. More death penalties may come later.

## 2. Core Gameplay Loop

1. Explore the generated overworld (move tile by tile, discover zones, settlements, dungeons, items).
2. Get caught by an enemy: visible NPCs and monsters notice the player and chase them (no random battles).
3. Switch to battle view and fight turn by turn with abilities.
4. Earn rewards (XP, items, money, new abilities).
5. Get stronger and push further out into harder territory; repeat endlessly.

### 2.1 Endless Progression
- The goal for now is a game with endless play: explore further, fight harder enemies, get stronger
- **Difficulty** goes up by one with every zone the player moves outward (see [world.md](world.md#21-zone-difficulty)). Staying in the current zone to farm XP is always possible when the next zone is too hard
- Long-term motivation beyond that: TODO later
  - IDEA: milestones/achievements, rare zones, legendary abilities to discover
  - IDEA: optional long-term goals (e.g. hunt generated world bosses, map completion)
- Power curve: how to avoid the player outscaling everything (or being outscaled): TODO

## 3. Main Character
- Single playable character, no party or companions
- Class chosen at character creation (see [classes.md](classes.md)). Name, appearance: TODO

## 4. Overworld

### 4.1 Movement & Camera
- Grid-based, tile-locked movement (4 directions), like Pokemon FireRed: a step always ends on a tile, so movement stays easy to control
- Camera follows player; 16×16 px tiles, about 20×11 tiles visible (see tech-stack.md rendering)
- Running / sprinting: sprinting for about 10 seconds, then the stamina is exhausted and needs to regenerate. It regenerates twice as fast when standing still. When exhausted you can still walk at normal speed
- Speed numbers (starting guesses): see [mechanics.md](mechanics.md#31-aggro--movement)

### 4.2 Interaction
- NPC dialogue: just a simple one liner for each NPC, similar to Pokemon
- NPC dialogue is template-based / generated, since there's no fixed script
- Signs
- chests: contain mostly lower quality items and food (see [items.md](items.md#7-loot-sources))
- item pickups: food, armor and weapons
- Shops, inns / healing points, class trainers (found in generated settlements)
- Saving: autosave only, no manual saves (see [Save System](#8-save-system))

## 5. Story & Setting
**Postponed.** Quests, lore and story are skipped for now and come later. No fixed main quest; the game is about exploring and fighting.
- Parked ideas for later: a setting that explains the endless world (e.g. a shifting/broken realm), procedural quests from settlements ("defeat X", "fetch Y from dungeon Z", "escort"), rare handcrafted events/landmarks, tone

## 6. Art & Audio Direction
**Decided: low-poly 3D with toon shading** (light in three flat bands, ink outlines), on the adventure map and in battle, in front of painted comic backgrounds in battle. The art first went the 2D route (comic images generated with Stable Diffusion, see 6.0.5); the switch to 3D came because generated 2D characters can't be made consistent across directions, poses and gear.

### 6.0.1 Why 3D
- **Consistency:** one model per character works from every direction and in every pose; animations come from one shared skeleton
- **Gear:** equipment is a matter of swapping or attaching meshes (see 6.0.3), and shows on the map and in battle alike
- **Perspective and scale** fit automatically under the map camera; nothing has to be drawn per viewing angle

### 6.0.2 Models
- **Source: Quaternius packs (CC0)** in `assets/quaternius/` (not in git, ~1 GB; unpacked from the free "Standard" downloads): Universal Base Characters, Modular Character Outfits – Fantasy, Universal Animation Library (43 humanoid animations). The map scenery (`tools/models/nature.py`) and all weapons and the shield (`tools/models/weapons.py`) are made in Blender instead; the Stylized Nature MegaKit and Fantasy Props MegaKit are no longer used. The pack only has peasant and ranger outfits, so the nine classes differ by outfit, hair, weapon and colors; outfit tints can recolor a part instead of only darkening it, Ultimate Animated Animals, Ultimate Monsters, Farm Animals (pig as the boar; ships only FBX and .blend, converted with `tools/models/convert.py` in Blender). Ultimate Monsters' "Big" models (goblin, yeti, dino, skull orc…) stretch their arms when raylib animates them; Blender re-exports of them (`monsters/Big/glTF-blender/`) work. Where no pack model fits, creatures are built from simple shapes and rigged and animated by Blender scripts in `tools/models/` (shared helpers in `rigkit.py`), writing to `assets/generated/creatures/`: the scarab (`scarab.py`), the cobra and the sand snake (`snake.py`, a smoothly bending body; the cobra has a hood), the mummy (`mummy.py`, with a generated bandage texture), the scorpion (`scorpion.py`), the tomb guardian (`guardian.py`, a golden jackal-headed statue with a sun staff) and the ember spirit (`ember.py`, a living flame)
- `tools/models/prepare.py` writes light copies of the models (`*.lite.gltf`): only the base color texture, at most 1024 px. The packs' normal and roughness maps aren't used by the toon shader and made loading a character take seconds
- **Humanoids** (classes, bandits, raiders) are built from parts on one skeleton: the base body, hair, and outfit pieces per equipment slot (`tools/models/human.py`). Enemies are built from the same pieces with other colors and options
- **Creatures:** every creature is built from simple shapes in Blender by a script in `tools/models` (wolf and boar: `beasts.py`; goblin: `goblin.py`; the desert creatures: one script each), rigged with one bone per part and animated by keyframes in the script. The same model in other colors, sizes and effects can serve several kinds (as WoW reuses creature models)
- **Buildings** are built in code from simple shapes (`internal/game/buildings.go`): walls, gable roofs (one slab plus gable-end triangles, generated prism meshes), domes, pyramids, on the footprint the world generator gives them. Their surfaces are generated textures (plaster, wood, roof tiles, stone, sandstone, mud brick; `tools/artgen/buildings.json`), normalized to grey so each part's tint sets the color, mapped by position (box mapping, since the shapes have no useful UVs) and darkened near the ground. The free packs have no houses; the Medieval Village MegaKit (CC0, modular walls and roofs) is an option
- Objects without a 3D version yet (sphinx, pharaoh statue) show their pixel sprite as a billboard

### 6.0.3 Gear
- **Armor slots** map to outfit parts: head (hood), shoulders (pauldrons), chest (body), hands (arms), legs, feet. Equipping an item swaps the part; every animation works with every combination
- **Weapons and shields** are models attached to bones: the weapon to the right hand, the shield to the left forearm. Their position follows the animation each frame
- **More looks per slot:** colors per part in the shader (materials, quality levels); new base color textures for the same meshes (could be generated with the art pipeline); more outfit packs on the same skeleton
- Planned: the shield moves to the back while walking on the map

### 6.0.4 Map and Battle
- **Map:** one tile is one meter. The camera looks down at about 55° from the south and follows the player. Ground: seamless textures (grass, dirt, water, sand, sand path) blended per tile through a smoothed, noise-roughened mask with ink lines along roads and water. Trees, bushes, rocks, flowers and grass tufts are pack models placed where the world generator puts objects, with a stable random variant, size and rotation per tile. Tall objects in front of the player turn see-through (dithered)
- **Battle:** the hero from behind on the left, the enemy further back on the right, both in 3D in front of the zone's painted background. Animations: stance, attack (or cast for the Mage), hit reaction, death; the hit target flashes. The pixel UI (unit frames, message box, menu) stays on top

### 6.0.5 Generated art (Stable Diffusion)
- **Still used for:** battle backgrounds per zone type, ground and building textures, and later illustrations and textures for gear
- stable-diffusion.cpp running locally (Vulkan), **SDXL** (SDXL-Lightning), handled by Claude. Pipeline in `tools/artgen/` with prompt files, candidates per seed (`--try`), cleanup (seamless textures, cut-outs with BiRefNet, MIT), and the prompt and settings kept next to every image
- Only models whose licenses allow commercial use of the output (checked before download)
- **Look:** hand-inked fantasy illustration: clean dark linework, matte colors in a muted earthy palette, soft cel shading with fine ink hatching. No gloss (no comic LoRA: it made everything glossy)
- **Battle backgrounds** are close-up scenes with free ground where the fighters stand and a high back edge of the ground, made from a rough layout picture (img2img). Objects at the fighters' depth keep their real size
- **Ground textures** need close-up prompts (one repeat covers a few meters) and are made seamless after generation; sd-cli's circular mode breaks SDXL-Lightning images
- ControlNet (OpenPose, xinsir, Apache-2.0) works with a local fix to ggml's Vulkan backend (`tools/artgen/patches/`); it was used for character poses before the switch to 3D

### 6.0.6 Order of work
1. ~~Art pipeline, style, battle backgrounds~~ (done)
2. ~~Switch to 3D: map, battle, buildings, characters from parts~~ (done in the first version)
3. Gear: items swap parts and held models; the shield on the back while walking
4. More creature models and recolors per zone; ~~desert and ancient desert battle backgrounds~~ (done: oasis with palms and a red rock; pyramids and a carved temple wall; both keep the middle open); monuments in 3D
5. Restyle the pixel UI to fit the look

- **Sound and music: our own**, synthesized by `tools/audio/makeaudio.py` (numpy, encoded with ffmpeg): 31 sound effects and 9 tracks (title, grassland, village, desert, oasis, ancient desert, fight, elite fight, defeat), each from a fixed seed. The UI font is our own too (`tools/font/makefont.py`).
- Not used: paid image APIs and scripting web UIs (Grok)
- Distinct palette and look per zone

### 6.1 Music
- At least one theme per zone, matching the zone's character
- **Battle music** is separate from the zone music: fast and driving like the Pokemon battle themes, but it still fits the zone a bit (same instruments or a variation of the zone's melody). One battle theme per zone, plus IDEA: special themes for elites and bosses
- Implemented with our own tracks (`tools/audio/makeaudio.py`, played by `internal/audio`, picked in `internal/game/sound.go`): title; Grassland "grassland", near its settlement "village"; Desert "desert" and Ancient Desert "ancient", near their settlements "oasis"; battles "fight", elites "fight_elite"; a defeat plays "sad"
- Sound effects, also synthesized by `tools/audio/makeaudio.py`: menu cursor, confirm and back; the "!" when an enemy spots the player; abilities by element or delivery, hits, critical hits, misses, stat changes, statuses (not Stun), heals; coins, level-ups; lighting and crackling camp fires; healing at inns and shrines. Test runs never open the audio device; `-mute` turns audio off

| Zone | Mood | Reference / style |
|------|------|-------------------|
| Grassland | A bit adventurous, upbeat, setting out on a journey | Like Pokemon's "Road to Fuchsia" route theme |
| Desert | Middle Eastern vibe, hot and wide | Middle Eastern scales, oud, hand drums |
| North (snow) | Cold, sparse, Nordic | Low drones, Nordic folk instruments, choir |
| Mediterranean | Sunny, warm, relaxed | Mandolin, guitar, light folk melody |
| Unholy | Dark, gothic, eerie | Organ, harpsichord, low strings |

- SFX style: TODO

## 7. UI / UX
- Main menu, pause menu, inventory, character screen, abilities menu, map
- Overworld HUD: HP bar, MP bar, stamina (while sprinting), money
- Money display (gold/silver/bronze) in HUD and inventory
- Item names colored by rarity (white, green, blue, purple)
- Dialogue box style

### 7.1 Controls
| Key | Adventure map | Battle | Menus / inventory |
|-----|---------------|--------|-------------------|
| WASD / arrow keys | Move | Move selection | Move the cursor |
| Space | Interact (talk, open chests, use shrines) | Confirm | Equip / unequip the selected item |
| B | Open inventory | Back / cancel | Close inventory |
| N | Open abilities menu | — | Close abilities menu |
| Escape | Open main menu | — | Back / close |
| Shift (hold) | Run (sprint) | — | — |
| C | Camp: light a fire and rest (moving breaks camp) | — | — |

- Space is a dedicated interact key and does nothing else on the map; running is on Shift

| Gamepad | Same as |
|---------|---------|
| D-pad | WASD / arrow keys |
| A | Space |
| B | B |
| X | Escape |
| Y (hold) | Shift (run) |
| Select (Back/View) | N |
| Left shoulder | C (camp) |

- Mouse works in menus too (click to select, click to equip/unequip)

### 7.2 Inventory Screen
- **B** opens the inventory: the **bags** on the right, the **character puppet** on the left
- The puppet shows the equipment slots WoW classic style (see [items.md](items.md#3-equipment-slots)), including a dedicated ranged slot
- Equip and unequip with the mouse, or with keyboard/gamepad: a cursor moves over bag and puppet slots, Space equips the selected bag item (into its matching slot) or unequips the selected puppet item (back into the bag)
- Bag size: 20 slots (see [items.md](items.md#31-bags))

### 7.3 Abilities Menu
- **N** (gamepad: Select) opens the abilities menu on the adventure map: the **4 active slots** on the left, all other known abilities on the right, details of the selected ability at the bottom. N, B or Escape closes it
- Arrow keys / WASD move the cursor, left/right switch sides. Space on a known ability puts it into a free slot; if all slots are full, the player then picks the slot to replace (B cancels). Space on a slot takes the ability out
- Passive abilities (e.g. Enrage) and fallback attacks (Punch, Fire Bolt) are listed but always available; they never take a slot
- When a level-up teaches an ability while all slots are full, the reward screen points to the menu
- See [classes.md](classes.md#31-ability-loadout)

### 7.4 Title Screen and Main Menu
- **Title screen** when the game starts: Continue (only if a save exists), New game, Demo, Quit
- **Demo:** the game plays itself with a random class in a random zone, unsaved; Escape returns to the title. A bot walks by simple rules (rest at a camp fire when HP fall below 70% or MP below half, and before taking on a group of enemies until it has more the bigger the group; go for the nearest enemy it can beat, or wait for one that is already chasing it; run from chasers when hurt or when it can't beat them; skip enemies 5 or more levels below it; otherwise explore, preferring roads and zones fit for its level; once its level suits a harder zone next door, it travels to the nearest crossing into it, along the roads where it can) and picks battle moves by playing each one out on copies of the battle (`battle.PlanPlayerMove`, no trained model). It runs endlessly with the same character: after a defeat it carries on from the checkpoint like a player would
- **New game:** choose the class (left/right; shows the sprite, a one-line description, types and starting abilities), the start zone (Grassland, Desert, Ancient Desert or North: the game starts in front of the inn of the nearest zone of that type, and that zone is difficulty 1) and type a world seed, or leave it empty for a random world. If a save exists, starting asks for confirmation, because the new game replaces it. A short loading screen shows while the start zone is prepared
- **Main menu in the game:** Escape (gamepad X) on the adventure map: Resume, Abilities, Controls (key reference), Save and return to title, Save and quit. B or Escape closes it

## 8. Save System
- **One save, the autosave.** No save slots and no save menu; the player never manages saves. They can quit at any time and continue from where they were the next time they start the game
- What gets saved: world seed, player position, last checkpoint, main character (current HP/MP, stats, level, abilities, equipment), inventory, money (quest state once quests exist)
- World changes are stored as diffs on top of the seed (only chunks the player modified), not the whole world

### 8.1 Write Strategy
Saves must not spam the disk (avoid needless SSD writes), and they only write what changed:
- **Partial updates:** the save is split into independent parts (player position, character, inventory, each chunk diff, world meta). Each part has a dirty flag and only dirty parts are written
- **Event saves** for important changes, written right away: battle won or lost, item gained/equipped/sold, level-up, ability learned, checkpoint activated, entering a new zone
- **Position only:** while the player just walks around, only the position is saved, and at most every ~30 seconds (and only if it changed)
- **On quit** (and when the window is closed): write all dirty parts
- **Format:** a single SQLite file (one table per part, chunk diffs keyed by chunk coordinates). Each save is one small transaction that updates only the changed rows, and a crash mid-write can't corrupt the save (see [tech-stack.md](tech-stack.md))
- **Implemented:** `~/.local/share/oldworld2/save.db` (or `$XDG_DATA_HOME`). Parts: world seed, player (position, facing, checkpoint), character (class, level, XP, HP, MP, money, abilities), defeated enemies. Events that save right away: battle end, checkpoint, closing the abilities menu. Position and HP/MP are saved at most every 30 seconds when they changed, and everything that changed on quit
- The title screen offers Continue; "New game" (or `-new` on the command line) replaces the save with a new world

## 9. Scope & Milestones
- **Demo (done):** one full-sized Grassland zone to test gameplay, zone size, NPC placement, graphics and sound. It grew into three zone types (Grassland, Desert, Ancient Desert) and all nine classes, and the game has left demo mode
  - Engine: Go with raylib (see [tech-stack.md](tech-stack.md))
  - Player movement, running with stamina, camera
  - Classes: Warrior and Mage (melee vs. spells); since then all nine classes are playable (see [classes.md](classes.md))
  - 3–4 enemy types: trash (wolves, boars, goblins; the demo's bat was dropped, as no fitting model exists) and one elite (bandit Rogue), placed by the placement rules (see [world.md](world.md#5-enemy-placement))
  - Aggro, chasing, leash, grace period
  - Battles with the 4-slot loadout + Wait, damage formula, types, a few status effects
  - XP and level-ups, death with respawn at a checkpoint and money loss
  - Autosave
  - Not in the demo: items, shops, trainers, settlements, dungeons
- **Vertical slice:** all five zones, generated settlements + one dungeon type, zone difficulty scaling, loot, shops, saving
- **Full game:** TODO

## 10. Open Questions
Decided topics moved into the docs: difficulty per zone and zone grid ([world.md](world.md#2-zones)), chases and back-to-back fights ([mechanics.md](mechanics.md#3-encounters--chases)).

Still open, to be handled later:
- **Enemy respawn:** do defeated enemy NPCs and monsters come back, and when? (e.g. after a timer, when the chunk unloads, never for bosses)
- **Long-term motivation:** what keeps players going besides endless exploration
- **Handcrafted content:** comes later together with story and quests. Probably no handcrafted maps
- **Class balance:** solved type extremes by making heroes Physical-typed (see [mechanics.md](mechanics.md#63-multi-type-balancing)). Still open: Judgment Hammer's random 1–3 round stun
- **Art style:** decided: comic style on the map and in battle, generated with Stable Diffusion (see [Art & Audio](#6-art--audio-direction)). Open: how well rig parts and seamless tiles work out in practice
- **Stat numbers:** base Attack/Magic/Defense/Resistance/Speed per class, growth per level, ability power values
- **Zone size** in tiles: starting guess 512×512, to be judged by playtesting
