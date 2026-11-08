# Game Mechanics

> Part of the [Oldworld2 design docs](game-design.md). Items marked `TODO` need a decision; items marked `IDEA` are suggestions, not commitments.

## 1. Battle View
- Separate screen; layout: pokemon style view and slide in, background image that matches the zone art
- UI elements: player HP/MP bars, enemy info, action menu, battle log

## 2. Turn Structure
- Turn order: highest Speed moves first every turn (Pokemon style); ties are random
- **Priority** (like Pokemon move priority): some abilities have a priority of +1 or +2 and always go before abilities with a lower priority, regardless of Speed. Speed only decides within the same priority. Examples: Mana Shield +2, Quick Stab +1, Tusk Charge +1
- Always 1v1: main character vs. one enemy
- Several enemies are never fought at once; a group engages one after another (see [Encounters & Chases](#3-encounters--chases))
- **Actions per turn:** one of the 4 active abilities (see [classes.md](classes.md#31-ability-loadout)), **Wait**, or an Item
- **Wait** is a fifth combat move everyone always has: do nothing this turn. Useful to let an enemy's buffs run out while it is disabled (e.g. the Rogue waiting out Blind)
- **Limited uses:** some strong abilities can only be used once (or twice) per battle; the limit resets after the battle. Some can only be used in the first round (openers like Charge)

## 3. Encounters & Chases
- Encounter types: visible roaming enemies and generated bosses / champions. **No random battles** in tall grass; fast, aggressive enemies with a large aggro range take that role, since they can't be avoided
- Encounter tables per zone: see [world.md](world.md#4-encounter-tables)
- **Chasing:** NPCs that notice the player chase them on the adventure map. If they catch up (they are faster, or the player doesn't run), the battle starts
- **Groups:** NPCs can form a group and attack together. They still fight one at a time: when the first is defeated the player is back on the map, and the next one tries to engage
- **Escaping:** after beating one, the player can outrun the rest to get away. There is no special gauntlet mode and no extra rewards for beating a whole group; back-to-back fights simply happen when several NPCs catch you. HP/MP carry over (they are global, see [HP & MP](#10-hp--mp))
- **Grace period:** after a battle ends, the next NPC can't engage for a random 1–2 seconds, so the player has a moment to start running. Chasers keep two tiles away during it (backing off if closer) instead of standing nose to the player, and close in once it ends
- Flee rules: there is no flee from a running battle, but we might introduce skills that allow that

### 3.1 Aggro & Movement
Each creature has its own aggro behavior and movement speed. All numbers are starting guesses, to be found through playtesting.

| Behavior | Aggro range | Notes |
|----------|------------:|-------|
| Defensive | 1 tile | Never attacks on its own; only fights when the player comes right next to it |
| Passive | 4 tiles | Only notices the player up close |
| Normal | 6.5 tiles | |
| Aggressive | 10.5 tiles | Spots the player from afar |

Reference: Pokemon FireRed, where a walking step takes 16 frames (≈ 3.75 tiles/s) and running is twice as fast. That felt slow in 3D, so all speeds are about 30% faster than that, and the aggro ranges 30% larger. Movement is tile-locked like in FireRed (a step always finishes on a tile), which keeps 4-direction movement easy to control. Speeds are set as time per step so they are easy to tune.

| Mover | Time per tile | Speed (tiles/s) | Notes |
|-------|--------------:|----------------:|-------|
| Player walking | 12 frames | 5 | On a road: 10 frames, 6 tiles/s |
| Player running | 6 frames | 10 | On a road: 5 frames, 12 tiles/s. | Hold Shift / Y. Stamina lasts ~10 s, refills in ~20 s (twice as fast standing still). Without stamina the player walks. Rogues from level 6 (Improved Sprint): stamina lasts ~50 s |
| Slow NPC | 18 frames | ≈ 3.3 | Can always be walked away from |
| Normal NPC | 9 frames | ≈ 6.7 | Faster than walking, slower than running |
| Fast NPC | 7 frames | ≈ 8.6 | Catches up once stamina runs out |
| Very fast NPC | 5 frames | 12 | Faster than running; can't be avoided once it chases |

- **Giving up (leash):** each chase lasts a random **5–15 seconds**, then the NPC gives up and walks back to its spot. Each creature has its own range inside that (e.g. wolves 8–15 s, boars 5–8 s), see [world.md](world.md#33-enemies)
- Each creature has one behavior and one speed (e.g. wolves: aggressive + fast; boars: defensive + normal; bandits: normal + normal), see [world.md](world.md#3-enemies)

## 4. Combat Stats
| Stat | Purpose |
|------|---------|
| HP | Health; 0 = defeated |
| MP | Resource for abilities |
| Attack | Physical damage |
| Magic | Spell damage |
| Defense | Physical damage reduction |
| Resistance | Magic damage reduction |
| Speed | Turn order |

- **Base values at level 1:** 100 HP and 150 MP. Classes modify these (e.g. Mages have 1.5× MP); per-class numbers and growth per level in [classes.md](classes.md#11-base-stats)
- Abilities cost around **20–30 MP** per use. MP does not grow with level (gear can add MP), so the number of casts per full MP bar stays about the same

### 4.1 Stat Stages
Buffs and debuffs work like Pokemon stat stages: each step raises or lowers a stat by one **stage**, from −6 to +6. Stages reset when the battle ends.

| Stage | −3 | −2 | −1 | 0 | +1 | +2 | +3 |
|-------|----|----|----|---|----|----|----|
| Multiplier | 0.4× | 0.5× | 0.67× | 1× | 1.5× | 2× | 2.5× |

- Formula: `+n → (2 + n) / 2`, `−n → 2 / (2 + n)`
- "Attack up" means +1 stage unless the ability says otherwise
- Example: the Mage's Time Warp gives +1 Speed stage at the start of each of the next 3 rounds (up to +3)

## 5. Damage Formula
Pokemon-style formula (starting values, to be tuned):

```
Base   = ((2 × Level / 5 + 2) × Power × Atk / Def) / 50 + 2
Damage = max(1, floor(3 × Base × Affinity × Element × Crit × Random × Other))
```

| Term | Meaning |
|------|---------|
| 3 | Scale factor, because our HP starts at 100 (Pokemon's starts much lower) |
| Level | Attacker's level |
| Power | Ability power (e.g. basic strike 40, strong spell 90) |
| Atk / Def | Attacker's **Attack** vs. defender's **Defense** for physical abilities; attacker's **Magic** vs. defender's **Resistance** for magical abilities. Stat stages apply |
| Affinity | 1.5 if the ability's element matches one of the attacker's types, otherwise 1.0 (like Pokemon's same-type bonus) |
| Element | Product of the chart multipliers against each of the defender's types (see [Elements & Types](#6-elements--types)) |
| Crit | 1.5 on a critical hit, otherwise 1.0. Base crit chance 6.25% (1/16) |
| Random | Uniform roll between 0.85 and 1.00 |
| Other | Enrage (+40%), Burn (halves physical damage dealt), Mana Shield, equipment modifiers, etc. |

- Every damaging ability has a **category** (physical or magical) that picks the stat pair, and an **element** that picks the chart row. A "Flame Blade" can be a physical Fire ability; a "Fireball" is a magical Fire ability. The Physical element is always the physical category
- An Element multiplier of 0 means immune: the attack deals 0 damage (the `max(1, …)` floor does not apply) and the battle log shows "It had no effect"
- Status-only abilities skip the formula but still check the element chart for immunity
- **No random misses:** attacks always hit by default, because randomly missing feels frustrating. Misses only come from specific, visible mechanics (Flying, and evade abilities like Blink, Feign Death, Evade, see [Melee vs. Ranged](#8-melee-vs-ranged)). When a miss chance exists it is high (around 50%), so it feels like a clear penalty to play around, not bad luck
- Example: level 10 Mage (Magic 30) casts Fire Bolt (Fire, magical, Power 60) at an Ice-type enemy with Resistance 20:
  `Base = (6 × 60 × 30/20) / 50 + 2 = 12.8` → × 3 = 38.4 → × 1.5 (Affinity: Fire is a Mage type) → × 2 (Fire vs. Ice) → **98–115 damage**, against an elite with about 145 HP at that level

## 6. Elements & Types
- **Elements (8):** Physical, Fire, Water, Ice, Nature (earth, plants, poison), Air (wind & lightning), Holy, Dark
- Every element is both an attack element (for abilities) and a type (for characters)
- Like Pokemon, every enemy has **one or more types**. Player classes don't: heroes are always **Physical-typed** and take 1× from every element. Their class elements (see [classes.md](classes.md)) only grant the Affinity bonus
- Multiple types multiply: a Fire attack against an Ice/Dark enemy deals 2 × 2 = 4×; a Fire attack against a Water/Nature enemy deals 0.5 × 0.5 = 0.25×
- Physical is neutral: Physical attacks deal 1× against every type, and the Physical type takes 1× from every element. Having the Physical type only grants the Affinity bonus on Physical abilities
- IDEA: rare armor or accessories can add another type (and with it Affinity bonuses and new weaknesses)

### 6.1 Element Chart
Rows = attacking element, columns = defender's type. Empty cell = 1×.

| Attack ↓ / Defender → | Physical | Fire | Water | Ice | Nature | Air | Holy | Dark |
|------------------------|:--------:|:----:|:-----:|:---:|:------:|:---:|:----:|:----:|
| **Physical**           |          |      |       |     |        |     |      |      |
| **Fire**               |          | ½    | ½     | 2   | ½      |     |      | 2    |
| **Water**              |          | 2    | ½     |     | 2      |     |      |      |
| **Ice**                |          | ½    | ½     | ½   | 2      | 2   |      |      |
| **Nature**             |          | 2    |       | 2   |        | 0   |      |      |
| **Air**                |          |      | 2     |     | ½      | ½   |      |      |
| **Holy**               |          |      |       |     |        |     | ½    | 2    |
| **Dark**               |          |      |       |     |        |     | 2    | ½    |

Flavor: fire melts ice and burns the undead; water douses fire and floods the land; ice freezes the ground and grounds flyers; nature smothers fire and cracks ice with roots and stone, but can't reach the sky; lightning conducts through water; Holy and Dark are mutually destructive; Physical is plain force and neutral to everything.

### 6.2 Weaknesses & Resistances per Type
| Type | Weak to (2×) | Resists (½×) | Immune to (0×) | Notes |
|------|--------------|--------------|----------------|-------|
| Physical | — | — | — | Neutral |
| Fire | Water, Nature | Fire, Ice | — | |
| Water | Air | Fire, Water, Ice | — | Defensively strong, one weakness |
| Ice | Fire, Nature | Ice | — | Offensive type: hits Nature and Air hard |
| Nature | Water, Ice | Fire, Air | — | |
| Air | Ice | Air | Nature | |
| Holy | Dark | Holy | — | |
| Dark | Fire, Holy | Dark | — | Main Unholy type |

### 6.3 Multi-Type Balancing
With 2–3 types per class the multipliers stacked up to extremes (a Mage took 4× from Nature, a Shaman was immune to it), which made some classes too hard. Decision: heroes are always Physical-typed and take neutral damage from everything; class elements are kept as **affinities** (the 1.5× bonus on matching abilities). Enemies keep their types, so the player still benefits from hitting weaknesses.

## 7. Status Effects
Pokemon style: damage and healing over time are a **fraction of the target's max HP**, not absolute numbers, so there are no ranks of the same effect and they stay relevant at every level.

| Effect | What it does | Broken by damage |
|--------|--------------|:----------------:|
| Drain | Deals **1/8** max HP per round and heals the caster for the same amount (Drain Life). The strongest damage over time in the game; the Warlock's signature | — |
| Burn | 1/16 max HP per round; also halves physical damage dealt | — |
| Poison | 1/16 max HP per round | — |
| Bleed | 1/16 max HP per round (e.g. Rend) | — |
| Other DoTs | 1/16 max HP per round unless the ability says otherwise (Wither, Mind Pain, Consecration, Insect Swarm) | — |
| Regen | Heals 1/16 max HP per round | — |
| Stun | Skips turns for the ability's duration | No |
| Freeze | Skips turns for the ability's duration | Yes |
| Blind | Disables the target: skips turns for the ability's duration | Yes |
| Dread | Each turn, 50% chance to lose it (Dread) | No |
| Silence | Can't use spells (targeted and area spells that aren't physical) | No |
| Shield | Absorbs damage up to 25% of max HP (Ward of Light), then breaks | — |
| Evade | The next attack against the character misses (Blink, Feign Death, Evade) | — |
| Immune | Takes no damage (Divine Shield) | — |
| Smoke | Ranged attacks and targeted spells can't target the character (see [Melee vs. Ranged](#8-melee-vs-ranged)) | — |

- **Cap:** no damage over time deals more than 1/8 max HP per round
- **Stacking:** the same effect doesn't stack; reapplying it resets its duration. Different effects do stack (Drain Life + Wither + Poison)
- **Duration:** set per ability. Defaults: 1/16 effects (DoTs and Regen) last **10 rounds**, which is usually the rest of the battle, so they are worth using; Drain lasts 5 rounds because it is twice as strong and heals (TODO: tune)
- All status effects end when the battle ends
- IDEA: Sleep (like Blind, from Mutate), Lifesteal on hit (fits Unholy enemies)

## 8. Melee vs. Ranged
Every ability has a **delivery** tag that decides what it can get through:

| Delivery | Examples | Notes |
|----------|----------|-------|
| Melee | Heavy Strike, Quick Stab, Savage Strike | Uses the main-hand (and off-hand) weapon |
| Ranged | Steady Shot, Volley, Knife Throw | Uses the weapon in the ranged slot (see [items.md](items.md#3-equipment-slots)). No ammo |
| Targeted spell | Fire Bolt, Shadow Bolt, Smite | A projectile or beam aimed at the enemy |
| Area spell | Hailstorm, Hellfire Rain, Consecration, Thunderstorm | Hits a whole area, so it doesn't need a clear target. Still one enemy, since battles are 1v1 |
| Self | Battle Cry, Mend | Affects the caster only |

- A class needs abilities for a weapon type to attack with it (e.g. Hunters have Savage Strike for swords/daggers/spears, Rogues have Knife Throw)
- Enemy traits (like Pokemon's contact moves triggering Rough Skin or Static) make some enemies punish or block one delivery, so the player has a reason to switch:

| Trait | Effect | Example enemies |
|-------|--------|-----------------|
| Flying | 50% chance to dodge melee attacks. Ranged and spells hit normally | Bats, harpies, vampire bat swarms |
| Thorns | Melee attackers take recoil damage (e.g. 15% of the damage they dealt) | Cactus golems, briar beasts |
| Burning aura | Melee attackers take Fire damage and may get Burn | Fire elementals |
| Smoke | Ranged attacks and targeted spells can't be used against this enemy. Melee and area spells still hit | Rogues (Smoke Veil, 2 rounds), bandits (Smoke Screen, 3 rounds) |

- Champion modifiers can add traits too (e.g. "Winged", "Thorned", "Smoke-cloaked")
- How classes deal with these:
  - Rogues switch to **Knife Throw** against anti-melee traits
  - Warriors have **Overwhelm**, which turns dodges and misses into a big hit; recoil from Thorns triggers their **Enrage**
  - Casters use area spells against Smoke; ranged classes shoot Flying enemies
- IDEA: shield wall / deflect (ranged deals ½ damage), burrowed (only melee can hit until revealed)
- IDEA: opening shot (ranged gets a free first strike at the start of a battle)
- IDEA: weapon profiles (melee weapons hit harder, ranged weapons crit more often)

## 9. Win / Lose, Checkpoints & Death
- Win: enemy defeated → rewards screen (XP, money, loot)
- Lose: main character's HP reaches 0 → respawn at the last checkpoint, fully healed, lose 10% of money (see [items.md](items.md#8-currency--economy))
- **Checkpoints:** the inn in every settlement, plus shrines along the roads (see [world.md](world.md#27-roads))
  - Activated by facing the inn door or the shrine and pressing Space; also fully heals the main character
  - The last visited checkpoint is where the main character respawns after defeat

## 10. HP & MP
- HP and MP are **global**: the same values in the overworld and in battle. Damage taken and mana spent in a fight stay after it ends
- Both are always visible on the overworld HUD while traveling
- Ways to restore:
  - **Food** (consumable items), usable only on the adventure map, never in combat
  - **Abilities** such as healing spells, usable in the overworld and in battle
  - **World objects** like a mana shrine or a healing spring, placed procedurally. Usable once, then they recharge for 5 minutes before they can be used again
  - **Checkpoints** fully restore both
  - **Camp:** press C (gamepad: left shoulder) while standing still to light a camp fire and rest. HP regenerates at **10%** and MP at **20%** of the maximum per second until the player moves again. Enemies can still find a resting player
- **Passive regeneration**, only on the adventure map (never in combat, since combat is turn-based):
  - MP: 50% of max MP per minute (starting value, to be tuned)
  - HP: 10% of max HP per minute (starting value, to be tuned). Warriors regenerate HP twice as fast

## 11. Leveling & XP
- Uncapped levels, fitting the endless world (TODO: confirm)
- XP comes fast early and slows down over time: level 2 after defeating 2 enemies, level 3 after 4 more, then the number of kills per level grows more and more slowly
- Stat growth per level depends on the class (see [classes.md](classes.md))
- Skill trees / talent points: TODO

### 11.1 XP Curve
```
KillsNeeded(L) = round(2 + 2 × log2(L))        # same-level trash kills to go from level L to L+1
EnemyXP        = 10 × EnemyLevel × TierMod × LevelMod
XPToNext(L)    = KillsNeeded(L) × 10 × L
LevelMod       = clamp(1 + 0.1 × (EnemyLevel − PlayerLevel), 0.1, 1.5)
TierMod        = trash 1, elite 2, champion 3, boss 5   (see world.md enemy tiers)
```

| Level | Same-level trash kills to next level | XP to next level |
|------:|-------------------------------------:|-----------------:|
| 1 | 2 | 20 |
| 2 | 4 | 80 |
| 3 | 5 | 150 |
| 4 | 6 | 240 |
| 5 | 7 | 350 |
| 8 | 8 | 640 |
| 10 | 9 | 900 |
| 20 | 11 | 2 200 |
| 50 | 13 | 6 500 |
| 100 | 15 | 15 000 |

- Fighting enemies above your level speeds this up (up to 1.5× XP at +5 levels). Farming much weaker enemies is barely worth it (down to 0.1× XP at −9 levels or more)
- The kill counts assume same-level trash enemies (e.g. wolves). Elite humanoid NPCs give twice as much
- Since zone difficulty rises with every zone outward, the player naturally runs into higher-level enemies as they push outward
