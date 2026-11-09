# Terrain, Zones & Enemies

> Part of the [Oldworld2 design docs](game-design.md). Items marked `TODO` need a decision; items marked `IDEA` are suggestions, not commitments.

## 1. World Structure
- Regions / towns / routes / dungeons: need to be generated (RNG)
- **Endless world:** generated on the fly in chunks as the player moves (TODO: chunk size)
- **Seeded:** the same seed always produces the same world (shareable seeds?)
- **Terrain:** the zone type comes from the zone grid (see [Zone Grid](#22-zone-grid--size)); inside a zone, noise-based height/moisture shapes the terrain: rivers, lakes, cliffs, paths
- **Points of interest** placed procedurally:
  - Settlements (shops, healers, checkpoints, class trainers)
  - Dungeons / caves (separate generated interiors)
  - Shrines (checkpoints, mana shrines), healing springs, libraries, treasure, landmarks
- **Difficulty:** set per zone, see [Zone Difficulty](#21-zone-difficulty)
- Transitions: the overworld is seamless; doors/stairs lead into generated interiors
- Blocked tiles, ledges, water, tall grass (decoration only, no random battles), etc.
- What persists once generated: TODO (opened chests, defeated bosses, changed tiles)

## 2. Zones
Each zone has its own visuals (tileset, props, color palette), enemy NPCs that fit its theme, loot and music.

| Zone | Look & feel | Enemy NPC ideas |
|------|-------------|-----------------|
| Grassland | Meadows, farmland, small forests, rivers | Bandits, wolves, rogue farmhands, hedge wizards |
| Desert | Dunes, oases, sandstone ruins, cacti | Nomad raiders, scorpion riders, sand mages, tomb guardians |
| North (snow) | Snowfields, pine forests, frozen lakes, fjords | Raiders, frost shamans, ice trolls, berserkers |
| Mediterranean | Sunny coast, olive groves, white villages, marble ruins | Pirates, gladiators, oracles, bronze-armored soldiers |
| Unholy | Dark forests, crypts, gothic castles, blood moon | Vampires, thralls, bat swarms, necromancers, cultists |

### 2.1 Zone Difficulty
- The starting zone has **difficulty 1**. Every new zone gets the difficulty of the zone the player came from **+1**, and keeps it once generated
- Implemented as 1 + the zone's ring around the start (the shortest number of zones to walk through), so it doesn't depend on the order zones are visited. A game started in another zone type (new game option) counts rings from that zone instead: it is difficulty 1, and enemies everywhere are shifted by 5 levels per difficulty step
- Enemies, loot (item level) and bosses scale with the zone's difficulty. Enemy levels per difficulty: TODO (e.g. difficulty d → levels 5d−4 to 5d)
- The player can always stay in the current zone and farm XP if the next zone is too hard

### 2.2 Zone Grid & Size
- Zones are laid out on a grid: each zone is one roughly square cell attached to the zone before it
- The usable part of a zone is not a square: barriers (mountains, water, cliffs…) shape it into any outline inside its cell
- Size: think of a WoW zone. Starting guess: a cell of **512×512 tiles** (16×16 chunks of 32×32), about 1.7 minutes to walk across at walking speed (5 tiles/s). (TODO: tune by playtesting)

### 2.3 Zone Placement
Start is a grasslands zone, transitioning into other zones depends on the direction. Each transition has a change to stay in the same zone type, like grasslands leads to grasslands. But there is also a chance to transition to another zone type. Grasslands is in the center, so from its position, it looks like that:
- going north leads to north
- going east leads to unholy
- going west leads to mediterranean
- going south leads to mediterranean or desert

But any of those transitions can still cause another grasslands (can stay in the same type rule).

South of mediterranean is also a possible desert transition and east of unholy may be desert too.

Implemented: the direction from the start zone picks the type (the larger of the two offsets wins). Zones in the first ring stay grassland with 30% chance, the second ring with 15%, farther out 5%. South chooses between Mediterranean and Desert with smooth noise over the zone grid (so deserts form connected regions), with more desert the farther south; east of the unholy lands (3+ zones east) can be desert. **All zone types exist: Grassland, the two deserts, the North, the Mediterranean and the Unholy lands.** Entering a zone shows its name and difficulty.

### 2.3.1 Zone Names
Every zone gets a generated name in the style of its theme, shown with its type and difficulty when the player enters it:
- Grassland: English place names, e.g. "Mallowdale Greens", "Sunlit Lowlands of Ashhollow", "Godwin's Juniperby", "Cedric's Amber Fields"
- Desert (Middle Eastern): e.g. "Wells of Al-Sazirah", "Oasis of Qadun-Surak", "Jazir Wastes"
- Ancient Desert (Egyptian): e.g. "Tombs of Neferemankh", "Eye of Sekhetti the Radiant", "Ptahmose's Barrens"
- Repeats are very rare: every name pattern contains a compound or syllable-built name (tens of thousands to millions of names per theme), and each zone picks the first of 6 candidate names that no zone within 4 zones could also pick. Names never repeat within that 9×9 area; in a test over 31×31 zones, at most 3 of 961 names repeated, always far apart

### 2.3.2 Desert Themes
A desert zone is either the **Middle Eastern Desert** or the **Ancient (Egyptian) Desert**, chosen at random. Smooth noise over the zone grid makes neighboring desert zones usually share a theme.

### 2.3.3 Desert (Middle Eastern, implemented)
- Dark sand with dry grass tufts, light sand paths as roads, small rare oases with sand shores
- Palms grow around oases; boulder fields take the place of forests; zone barriers are boulder walls
- Decoration: skulls, bones, dry sprouts; small rocks as obstacles
- Settlements: domed huts, a domed tower and sandstone houses around the plaza, plus the usual inn
- Battles use a sand-colored background
- Crossings into neighboring zones switch from sand to grass at the zone edge (texture blending at crossings is still to do)

### 2.3.4 Ancient Desert (Egyptian, implemented)
- Same sand, paths and oases as the desert, with fewer boulders: open sand around the monuments
- **Pyramid groups:** two per zone, 60–200 tiles from the settlement: a big pyramid with an entrance, two small pyramids beside it and a pharaoh statue in front. The pyramids are our own pixel art (`assets/generated/egypt.png`, drawn by `cmd/genart`), since the asset pack has none
- The settlement has sandstone houses and a temple, two obelisks and a sphinx by the plaza, plus the inn
- Enemies: mummies, scarab swarms, cobras, and tomb guardians near the pyramids (see Enemies)

### 2.3.5 North (implemented)
- Snow with a few grass tufts poking through, frozen lakes with snowy shores (ice blocks like water), dirt roads with snow along their edges
- Dense forests of snow-laden pines (tall and small), snowy boulders among them; zone barriers are pine walls
- Decoration: frosted ferns, bare twigs, ice crystals; snowy shrubs as small obstacles
- Settlements: timber longhouses and log huts with snow on their roofs, plus the usual inn
- Names in Norse style, e.g. "Ulfheim Fells", "Frozen Reaches of Ravnes", "Sigrun's Hrimdal"
- Music: a slow minor theme (drone, horn, icy bells); a mead-hall tune near the settlement
- Enemies: snow wolves at the forest edges, frost wisps over open snow and frozen lakes, ice trolls deep in the pines, berserkers near roads

### 2.3.6 Mediterranean (implemented)
- Sun-dried golden grass (the grass texture faded and warmed in the shader), dirt roads, lakes
- Open olive groves of gnarled, silvery olive trees, tall dark cypresses and pale limestone rocks; zone barriers are cypress stands
- Decoration: lavender (also the small obstacle), sage, poppies, white flowers, broken marble columns, amphorae
- Settlements: whitewashed houses with flat roofs, blue doors and shutters, a small blue dome and bougainvillea; villas with terracotta roofs and a columned porch; plus the usual inn
- Names in Greek style, e.g. "Olive Groves of Kalinthos", "Serithra of the Muses", "Thalassos Headlands"
- Music: a bright tune in three (lyre, flute, tambourine); a gentle harbor tune near the settlement
- Enemies: giant crabs on the lake shores, wind spirits on the open hills, wild boars in the groves, corsairs (more of them by water) and bronze hoplites near roads

### 2.3.7 Unholy (implemented)
- Dead, withered grass (the grass texture drained in the shader, on a third ground mask) and dark dirt roads, lakes
- Dense dead woods: bare, twisted dead trees, nearly black pines and dark rocks; zone barriers are dark pine and dead tree walls
- Decoration: gravestones and stone crosses, violet mushrooms, thorn bushes (also the small obstacle), skulls and bones
- Settlements: gothic houses of dark stone with steep slate roofs, warm-lit narrow windows and a corner turret; stone crypts; plus the usual inn
- Names in a gothic style, e.g. "Morrowmoor Hollows", "Cursed Woods of Ravenwald", "Count Draven's Grimkeep"
- Music: a slow, dark theme (organ, tolling bell, a heartbeat drum); a mournful waltz near the settlement
- Enemies: dire wolves in the dead woods, bat swarms on the open moors, cultists near roads, and lone vampires far from the settlement

### 2.4 Zone Borders
zones do not share large open borders. They are separated by natural barriers (mountains, cliffs, deep water, dense forest, chasms, rivers) and connected only at a few small crossing points where a road leads from one zone to the next (pass, bridge, tunnel, gate)
- Number of crossings per border: 1-3 (1: 50% chance, 2: 30% chance, 3: 20% chance)
- Implemented (grassland): the barrier is a band of solid forest about 10 tiles wide (wobbling by a few tiles), crossings are 3-tile gaps kept at least 48 tiles from zone corners, and lakes stay out of the band
- **Transitions:** textures blend into each other around crossings as a visual transition effect

### 2.5 Zone Element Tendencies (IDEA)
Each zone favors certain elements (see [mechanics.md](mechanics.md#6-elements--types)).

| Zone | Common enemy types |
|------|--------------------|
| Grassland | Physical, Nature, Air |
| Desert | Fire, Nature |
| North (snow) | Ice, Water |
| Mediterranean | Water, Air, Holy |
| Unholy | Dark, Dark/Fire, Dark/Ice |

### 2.6 Reachability
- Forests, bushes and water can enclose holes. Small holes are fine, but nothing may be trapped in one
- When a zone is first needed, its whole terrain is generated (in parallel) and the largest connected walkable area becomes the zone's **main area**. Enemies and the player only spawn there
- Forest interiors are capped at 75% tree density so they stay porous; only zone barriers are solid walls

### 2.7 Roads
Roads guide the player to the main objectives: settlements and the crossings to the next zones.
- Every crossing has a road to the zone's settlement: straight in from the crossing for 16 tiles, then winding (midpoint-displaced curves) to the hub. Roads continue on the other side of a crossing into the next zone
- Roads are about 3 tiles wide, drawn with the dirt path tiles of the floor tileset (autotiled like the shoreline). Trees and bushes keep 2 tiles away, lakes 4 tiles
- No enemy groups within 5 tiles of a road; bandit camps sit within 24 tiles of one (see [Enemy Placement](#5-enemy-placement))
- **Shrines:** every road longer than 120 tiles gets a checkpoint shrine about halfway, 4–6 tiles beside the road (at least 50 tiles from the settlement). Trees, bushes and lakes keep clear of it, and no enemy groups spawn within 12 tiles
- Planned: side roads to points of interest (dungeons, camps), bridges over rivers

### 2.8 Settlements
- One settlement per zone (the hub): at the world origin in the start zone, otherwise within ±64 tiles of the zone center
- A dirt plaza (radius 5) where the roads meet, the **inn** and up to four houses around it. Grassland uses the orange-roofed houses; the beige sandstone and red brick houses fit the desert better. Buildings search for free spots near the plaza, so they never stand on a road
- No trees or bushes within 24 tiles, no lakes within 28 tiles, no enemies within 40 tiles of the hub, and no elites within 100 tiles (trash takes their place), so a new character can level up on trash before meeting the first elite
- **Inn = checkpoint:** face the inn door and press Space to fully heal and make it your respawn point. A new game starts in front of the start zone's inn
- Planned: shops, class trainers, villagers with one-liners, more buildings in bigger settlements

## 3. Enemies
- Enemy NPCs themed per zone (see [Zones](#2-zones))
- Like in WoW, enemies are built from a **creature** plus an optional **class**:
  - The **creature** (human, ogre, wolf, troll, bat swarm…) sets base stats, creature types and traits (e.g. Flying, see [mechanics.md](mechanics.md#8-melee-vs-ranged))
  - The **class** adds its types and a few abilities from that class's pool (see [classes.md](classes.md)). An Ogre Mage is an ogre with Mage abilities; a bandit is a human Rogue
  - Not every creature has a class: most ogres are plain ogres with a few creature abilities (Club Smash, Roar). Beasts usually have no class
- Higher-level enemies know more abilities from their class pool
- Scaling: stats and ability sets grow with zone difficulty
- Aggro behavior and speed per creature: see [mechanics.md](mechanics.md#31-aggro--movement)

### 3.1 Enemy Tiers
The world is mostly filled with easy **trash** enemies to farm; gear mostly comes from **elite** humanoid NPCs. HP and damage are compared to a player of the same level (starting values, TODO: tune).

| Tier | Examples | HP | Damage | Abilities | XP | Loot |
|------|----------|---:|-------:|-----------|---:|------|
| Trash | Wolves, boars, goblins, bats, spiders (creatures without a class) | ~35% | ~50% | 1–2 creature abilities | 1× | Money and food, gear rarely |
| Elite | Bandits, hedge wizards, cultists (humanoids with a class) | 60–80% | ~100% | 2–3 class abilities | 2× | Main source of gear |
| Champion | Elites or big creatures with a random modifier ("Burning", "Swift", "Shielded", "Winged", "Thorned") | ~100% | ~110% | 3–4 + the modifier | 3× | Always gear, better rarity |
| Boss | Dungeon and world bosses | 200%+ | ~120% | Own ability set | 5× | Several pieces of gear, tomes |

- Trash enemies spawn at the lower end of the zone's level range, elites and champions at the upper end

### 3.3 Enemies
Level 1 values; they grow per level the same way as player stats ([classes.md](classes.md#11-base-stats)). Player reference at level 1: 85–120 HP, Speed 10.

| Enemy | Tier | Types | HP | Attack | Magic | Defense | Resistance | Speed | Aggro | Move | Leash |
|-------|------|-------|---:|-------:|------:|--------:|-----------:|------:|-------|------|------:|
| Wolf | Trash | Physical | 50 | 8 | 2 | 10 | 8 | 12 | Aggressive | Fast | 8–15 s |
| Boar | Trash | Physical, Nature | 60 | 9 | 2 | 12 | 6 | 8 | Defensive | Normal | 5–8 s |
| Goblin | Trash | Physical, Dark | 45 | 9 | 2 | 8 | 7 | 12 | Normal | Fast | 6–12 s |
| Bandit (Rogue) | Elite | Physical, Dark | 80 | 13 | 4 | 9 | 8 | 11 | Normal | Normal | 10–15 s |

| Enemy | Abilities |
|-------|-----------|
| Wolf | Bite (Physical, melee, Power 40), Howl (Attack +1 stage) |
| Boar | Gore (Physical, melee, Power 45), Tusk Charge (Nature, melee, Power 55, goes first) |
| Goblin | Club (Physical, melee, Power 40), Sling Stone (Physical, ranged, Power 30, 30%: Speed −1 stage) |
| Bandit | Quick Stab (Power 45, high crit), Poisoned Blade (Poison), Smoke Screen (Smoke for 3 rounds) |

| Enemy (desert) | Tier | Types | HP | Attack | Magic | Defense | Resistance | Speed | Aggro | Move | Leash |
|-------|------|-------|---:|-------:|------:|--------:|-----------:|------:|-------|------|------:|
| Sand Snake | Trash | Nature | 45 | 9 | 3 | 9 | 8 | 12 | Normal | Fast | 6–12 s |
| Scorpion | Trash | Physical, Nature | 55 | 10 | 2 | 13 | 6 | 9 | Defensive | Normal | 5–8 s |
| Ember Spirit | Trash | Fire | 40 | 4 | 11 | 7 | 11 | 12 | Passive | Normal | 5–10 s |
| Nomad Raider | Elite | Physical, Fire | 85 | 14 | 6 | 10 | 8 | 11 | Normal | Normal | 10–15 s |

| Enemy (desert) | Abilities |
|-------|-----------|
| Sand Snake | Venom Bite (Nature, Power 35, 30% Poison), Coil (Defense +1 stage) |
| Scorpion | Pincer (Physical, Power 40), Sting (Nature, Power 30, 50% Poison) |
| Ember Spirit | Ember (Fire spell, Power 40, 20% Burn), Flare (Fire area spell, Power 35) |
| Nomad Raider | Scimitar Slash (Physical, Power 50), Fire Brand (Fire melee, Power 35, 30% Burn), Sand Veil (Smoke) |

| Enemy (ancient desert) | Tier | Types | HP | Attack | Magic | Defense | Resistance | Speed | Aggro | Move | Leash |
|-------|------|-------|---:|-------:|------:|--------:|-----------:|------:|-------|------|------:|
| Mummy | Trash | Physical, Dark | 60 | 10 | 6 | 12 | 6 | 6 | Normal | Slow | 10–15 s |
| Scarab | Trash | Physical, Nature | 40 | 9 | 2 | 12 | 7 | 13 | Aggressive | Fast | 6–10 s |
| Cobra | Trash | Nature | 45 | 9 | 4 | 8 | 9 | 12 | Normal | Normal | 6–12 s |
| Tomb Guardian | Elite | Physical, Holy | 95 | 14 | 10 | 14 | 10 | 8 | Normal | Slow | 8–12 s |

| Enemy (ancient desert) | Abilities |
|-------|-----------|
| Mummy | Wrap (Physical, Power 40, 30% Speed down), Ancient Curse (Dark spell, Power 35, 50% Attack down) |
| Scarab | Mandible (Physical, Power 35), Carapace (Defense +1 stage); comes in swarms of 2–3 |
| Cobra | Venom Spit (Nature, ranged, Power 40, 30% Poison), Hood Flare (enemy Attack −1 stage) |
| Tomb Guardian | Sun Strike (Holy melee, Power 50), Judgment Ray (Holy spell, Power 45), Sacred Ward (Defense and Resistance +1) |

- Ancient desert placement: scarabs and cobras everywhere, mummies on open sand; tomb guardians (and more mummies) within 30 tiles of a pyramid
- Desert placement: scorpions in boulder fields, snakes and ember spirits on open sand, nomad raider groups near roads (rarer than in grassland because the desert has many roads)
- Enemy levels follow the zone difficulty: difficulty d → levels 5d−4 to 5d, trash at the low end, elites at the high end, rising with the distance from the settlement
- Rough check at level 1 (Affinity included): a Warrior kills a wolf in about 2 Heavy Strikes (~26 damage each, 20 MP) and takes ~14 per Bite (about 8 bites to lose); a Mage needs 2–3 Fire Bolts (~26 each) and takes ~19 per Bite (about 4–5 bites to lose)
- Generated bosses: TODO (templates + random abilities?)
- Enemy AI behavior: TODO (random, weighted, rule-based patterns)

| Enemy (north) | Tier | Types | HP | Attack | Magic | Defense | Resistance | Speed | Aggro | Move | Leash |
|-------|------|-------|---:|-------:|------:|--------:|-----------:|------:|-------|------|------:|
| Snow Wolf | Trash | Ice | 50 | 9 | 3 | 9 | 9 | 12 | Aggressive | Fast | 8–15 s |
| Frost Wisp | Trash | Ice | 40 | 4 | 11 | 7 | 11 | 11 | Passive | Normal | 5–10 s |
| Ice Troll | Trash | Physical, Ice | 75 | 12 | 4 | 13 | 6 | 5 | Normal | Slow | 8–12 s |
| Berserker | Elite | Physical, Ice | 90 | 15 | 4 | 8 | 8 | 11 | Normal | Normal | 10–15 s |

| Enemy (north) | Abilities |
|-------|-----------|
| Snow Wolf | Frost Bite (Ice, Power 40, 20% Speed down), Howl (Attack +1 stage); packs of 2–3 |
| Frost Wisp | Ice Shard (Ice spell, Power 40, 15% Freeze), Chill Wind (Ice area spell, Power 35); flies |
| Ice Troll | Troll Smash (Physical, Power 60), Hurl Ice (Ice, ranged, Power 40), Frozen Hide (Defense +1) |
| Berserker | Cleave (Physical, Power 55), Berserk (Attack +2, Defense −1 stages), Frost Axe (Ice, Power 40, 15% Freeze) |

| Enemy (mediterranean) | Tier | Types | HP | Attack | Magic | Defense | Resistance | Speed | Aggro | Move | Leash |
|-------|------|-------|---:|-------:|------:|--------:|-----------:|------:|-------|------|------:|
| Giant Crab | Trash | Water, Physical | 60 | 10 | 4 | 15 | 6 | 7 | Defensive | Normal | 5–8 s |
| Wind Spirit | Trash | Air | 40 | 4 | 11 | 7 | 10 | 14 | Passive | Fast | 5–10 s |
| Corsair | Elite | Physical, Water | 85 | 14 | 5 | 9 | 8 | 12 | Normal | Normal | 10–15 s |
| Hoplite | Elite | Physical, Holy | 95 | 13 | 6 | 15 | 9 | 9 | Normal | Normal | 8–12 s |

| Enemy (mediterranean) | Abilities |
|-------|-----------|
| Giant Crab | Pinch (Physical, Power 45), Bubble Spray (Water spell, Power 35, 30% Speed down), Hard Shell (Defense +2 stages) |
| Wind Spirit | Gust (Air spell, Power 40), Whirlwind (Air area spell, Power 35, 20% Speed down); flies |
| Corsair | Cutlass Slash (Physical, Power 50), Pistol Shot (Physical, ranged, Power 60, priority +1, once per battle), Smoke Flare (Smoke for 2 rounds) |
| Hoplite | Spear Thrust (Physical, Power 55), Shield Wall (Defense +2 stages), Sun Spear (Holy melee, Power 45) |

| Enemy (unholy) | Tier | Types | HP | Attack | Magic | Defense | Resistance | Speed | Aggro | Move | Leash |
|-------|------|-------|---:|-------:|------:|--------:|-----------:|------:|-------|------|------:|
| Bat Swarm | Trash | Dark | 40 | 9 | 4 | 7 | 8 | 14 | Aggressive | Fast | 6–10 s |
| Dire Wolf | Trash | Dark, Physical | 58 | 10 | 3 | 10 | 8 | 12 | Aggressive | Fast | 8–15 s |
| Cultist | Elite | Dark, Fire | 80 | 8 | 14 | 8 | 11 | 10 | Normal | Normal | 10–15 s |
| Vampire | Elite | Dark | 100 | 14 | 12 | 10 | 11 | 12 | Normal | Normal | 10–15 s |

| Enemy (unholy) | Abilities |
|-------|-----------|
| Bat Swarm | Bat Bites (Physical, Power 35, 20% crit), Screech (enemy Attack −1 stage); flies |
| Dire Wolf | Savage Bite (Physical, Power 45, 25% Bleed), Dark Howl (Attack +1 stage) |
| Cultist | Shadow Bolt (Dark spell, Power 50), Fire Brand (Fire melee, Power 35, 30% Burn), Dark Pact (Attack and Magic +1 stage) |
| Vampire | Life Drain (Dark spell, Power 30, Drain at half strength: 1/16 max HP per round), Vampire Claw (Physical, Power 55), Mesmerize (Dread for 2 rounds, once per battle); alone |

### 3.2 Enemy-Only Classes
Extra classes that are not playable. They have their own ability pools, often mixed from player classes.

| Class | Types | Zone | Style |
|-------|-------|------|-------|
| Vampire | Dark | Unholy | Drains life, Bat Form (gains Flying), calls thralls that join the chase |
| Necromancer | Dark, Ice | Unholy | Curses, bone shields, raises a defeated member of its group once |
| Berserker | Physical, Ice | North | Hits harder the lower its HP, can't be Stunned while enraged |
| Corsair | Physical, Water | Mediterranean | Pistol and cutlass, Smoke flares, steals money on hit |
| Oracle | Holy, Air | Mediterranean | Foresight (dodges the next attack), shields, lightning |
| Sand Shaper | Fire, Nature | Desert | Sandstorm (Smoke for both sides), scorching area spells |

## 4. Encounter Tables
- Encounter tables per zone, scaled by zone difficulty: TODO
- No random battles; every fight is a visible NPC (see [mechanics.md](mechanics.md#3-encounters--chases))

## 5. Enemy Placement
Enemies are **not spread evenly** across a zone. Placement follows reasons (habitats, camps, roads), with randomness on top so no two zones look the same.

- **Habitats:** each creature lists the terrain it lives in and how it groups up

| Creature (Grassland examples) | Habitat | Group |
|-------------------------------|---------|-------|
| Wolves | Forests and forest edges | Packs of 2–4 |
| Boars | Meadows, farmland edges | Alone or pairs |
| Goblins | Camps by ponds and streams | Groups of 2–3 |
| Bats | Caves, ruins | Swarms of 2–3 |
| Bandits (elite) | Open land within 24 tiles of a road, away from the start | 2–3 per group |
| Hedge wizards (elite) | Lone towers, stone circles | Alone |

- **Camps and points of interest:** elites and champions sit at camps, ruins and dens, which act as small danger spots
- **Safe areas:** no hostile spawns near settlements and checkpoints, and no elites within 100 tiles of a settlement; roads are quieter, except for ambushes near camps
- **Density:** grows with distance from roads and settlements, highest in remote corners and near zone barriers
- **Randomness:** seeded noise varies density per habitat, group sizes are rolled, and groups keep a minimum distance from each other (Poisson-disc style) so they don't pile up. A few enemies wander instead of staying at their spot
