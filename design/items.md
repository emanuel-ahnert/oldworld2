# Items & Economy

> Part of the [Oldworld2 design docs](game-design.md). Items marked `TODO` need a decision; items marked `IDEA` are suggestions, not commitments.

WoW (classic) style itemization: gear comes in rarities from white to epic, scales with item level and rolls random bonus stats. All numbers are starting values, to be tuned.

## 1. Rarity
| Rarity | Color | Bonus stats | Stat budget | Typical sources |
|--------|-------|-------------|-------------|-----------------|
| Common | White | none, base stats only (armor or weapon value) | 1.0× | Shops, chests, elite NPCs |
| Uncommon | Green | 1–2, from a random suffix ("of the Bear") | 1.2× | Elite NPCs, chests, some shops |
| Rare | Blue | 2–3 | 1.45× | Champions, dungeon bosses, rare chests |
| Epic | Purple | 3–4 + one special effect | 1.75× | World bosses, deep dungeons, very rare drops |

- No grey junk items: there are enough white items the player can't use that can be sold instead, so junk would only clutter the bags
- Epic special effects, examples: resist an element (take ½× from Fire), proc on hit (chance to Burn), grant an extra ability, add a type (see [mechanics.md](mechanics.md#6-elements--types))

## 2. Item Level & Stat Budget
- Every piece of gear has an **item level**, equal to the level of the enemy, chest or shop area it came from (so it follows the zone difficulty)
- Using gear needs a minimum character level: `RequiredLevel = max(1, ItemLevel − 3)` (TODO: tune)
- Stat budget per item:

```
Budget = ItemLevel × SlotMod × RarityMod
```

| Slot group | SlotMod |
|------------|--------:|
| Chest, Legs, two-handed weapon | 1.0 |
| Head, Shoulders, one-handed weapon | 0.8 |
| Hands, Waist, Feet, Off hand, Ranged | 0.6 |
| Wrist, Back, Neck, Ring, Trinket | 0.45 |

- The base stat (armor value on armor, Attack or Magic on weapons) takes about half the budget; bonus stats share the rest. Commons put the whole budget into the base stat at 1.0×

## 3. Equipment Slots
WoW classic slots, shown on the character puppet in the inventory screen (see [game-design.md](game-design.md#72-inventory-screen)). Cosmetic slots (shirt, tabard) are left out.

| Puppet side | Slots |
|-------------|-------|
| Left | Head, Neck, Shoulders, Back, Chest, Wrist |
| Right | Hands, Waist, Legs, Feet, Ring ×2, Trinket ×2 |
| Bottom | Main hand, Off hand, Ranged |

| Slot | Notes |
|------|-------|
| Head, Shoulders, Chest, Wrist, Hands, Waist, Legs, Feet | Armor of the class's armor type (see below) |
| Back | Cloak, any class |
| Neck, Ring ×2 | Jewelry, bonus stats only |
| Trinket ×2 | Special effects (e.g. once per battle: heal 20%, Speed up for 2 turns) |
| Main hand | Melee weapon (one- or two-handed) |
| Off hand | Shield, off-hand item (orb, tome) or second weapon; empty with a two-handed weapon |
| Ranged | Bows, crossbows (Hunters), thrown weapons (Rogues), wands (casters). Paladins, Shamans and Druids use a **relic** here instead (libram, totem, idol) that boosts some of their abilities. Warriors leave it empty |

- Melee abilities use the main/off-hand weapon, ranged abilities the ranged slot (see [mechanics.md](mechanics.md#8-melee-vs-ranged))
- Equipped gear is visible on the battle sprite? TODO

### 3.1 Bags
- One bag with **20 slots** from the start. No bigger bags for now
- Equipped items don't take bag slots
- Stackable: consumables (food, potions, cures) stack (TODO: stack size, e.g. 20). Gear never stacks

## 4. Armor & Weapon Types
Like WoW, a class can wear its own armor type and any lighter one, but only gets the full benefit from its own type.

| Armor type | Classes | Feel |
|------------|---------|------|
| Cloth | Mage, Warlock, Priest | Low Defense, high Resistance, often Magic/MP bonuses |
| Leather | Rogue, Druid | Medium Defense, often Speed/crit bonuses |
| Mail | Hunter, Shaman | Medium-high Defense, mixed bonuses |
| Plate | Warrior, Paladin | Highest Defense, low Resistance |

| Class | Melee (main hand) | Off hand | Ranged slot |
|-------|-------------------|----------|-------------|
| Warrior | Swords, axes, maces, spears (1H and 2H), daggers | Shield or second weapon | — (no ranged weapons; Overwhelm handles dodges) |
| Rogue | Daggers, swords, maces | Second weapon | Thrown (for Knife Throw) |
| Hunter | Swords, daggers, axes, spears | Second weapon | Bows, crossbows |
| Mage | Staves, daggers | Orb / tome | Wands |
| Warlock | Staves, daggers | Orb / tome | Wands |
| Priest | Staves, maces, daggers | Orb / tome | Wands |
| Paladin | Swords, maces, axes (1H and 2H) | Shield | Relic (libram) |
| Shaman | Maces, axes, staves, daggers | Shield | Relic (totem) |
| Druid | Staves, maces, daggers | Orb / tome | Relic (idol) |

- Equipping a weapon is not enough: the class needs abilities that use it (see [classes.md](classes.md#4-class-ability-pools))

## 5. Stats & Suffixes
Bonus stats on gear use the combat stats from [mechanics.md](mechanics.md#4-combat-stats), plus crit chance.

Uncommon and better items roll a random suffix that picks their bonus stats (in the spirit of WoW classic's "of the Bear" items):

| Suffix | Bonus stats | Fits |
|--------|-------------|------|
| of the Bear | Attack, HP | Warrior, Paladin |
| of the Owl | Magic, MP | Mage, Priest, Druid |
| of the Fox | Speed, crit chance | Rogue, Hunter |
| of the Turtle | Defense, Resistance | Any |
| of the Serpent | Magic, Speed | Warlock, Shaman |
| of the Boar | Attack, Defense | Warrior, Paladin, Shaman |
| of the Wolf | Attack, Speed | Rogue, Hunter |
| of Vigor | HP, MP | Any |

- Rare and epic items can roll a second suffix stat set or a fixed, hand-named item (e.g. "Frostbite Edge")
- IDEA: drops favor the player's class (about 60% of gear drops are usable armor/weapon types), so loot feels relevant without being guaranteed

## 6. Consumables & Key Items
- **Food:** restores HP/MP, usable only on the adventure map, never in combat
- **Potions:** restore HP or MP, usable in combat (uses the turn)
- **Status cures:** remove Poison, Burn, Freeze, etc.
- **Tomes:** teach a rare ability, used up on learning (see [classes.md](classes.md#32-learning-abilities))
- **Key items:** dungeon keys (quest items later, once quests exist)
- Item use in combat counts as the turn's action

## 7. Loot Sources
| Source | Contents | Rarity odds (common / uncommon / rare / epic) |
|--------|----------|-----------------------------------------------|
| Trash enemy | Money and food; gear rarely (~3% chance) | 80 / 18 / 2 / 0 % |
| Elite enemy (humanoid NPC) | Money, gear (~60% chance), sometimes food | 60 / 32 / 7.5 / 0.5 % |
| Champion | Money, always one piece of gear | 30 / 50 / 18 / 2 % |
| Dungeon boss | Money, 1–2 pieces of gear, sometimes a tome | 0 / 40 / 50 / 10 % |
| World boss | Money, 2 pieces of gear | 0 / 0 / 70 / 30 % |
| Chest | Mostly lower quality items and food | 80 / 18 / 2 / 0 % |
| Shop | Commons always, a few uncommons; restocks over time | 85 / 15 / 0 / 0 % |

- Item pickups lying in the world: food, armor and weapons
- Quality scales with zone difficulty through item level (see [Item Level](#2-item-level--stat-budget))

## 8. Currency & Economy
- Three coin types, WoW style: **bronze**, **silver**, **gold**
- 100 bronze = 1 silver, 100 silver = 1 gold (TODO: confirm ratio)
- Stored internally as a single integer in bronze; shown as `12g 34s 56b`
- Sources: enemy drops, selling items, chests (quest rewards later)
- Sinks: shops, healing?, ability trainers, equipment repair?
- **Death penalty:** 10% of the player's money is lost on every death
- A bank (if added) does not protect money from death loss
- **Sell price** (to shops): `ItemLevel × RarityPrice × SlotMod` bronze, with RarityPrice = 5 / 12 / 30 / 75 for common / uncommon / rare / epic (TODO: tune)
- Shops sell at 4× the sell price
- Durability and repair: TODO
