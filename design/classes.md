# Classes & Spells

> Part of the [Oldworld2 design docs](game-design.md). Items marked `TODO` need a decision; items marked `IDEA` are suggestions, not commitments.

## 1. Classes
The main character and NPCs have classes, inspired by World of Warcraft (classic). A class sets the character's affinities (the elements that get the Affinity bonus; heroes themselves are always Physical-typed, see [mechanics.md](mechanics.md#6-elements--types)), stat focus, ability pool and the armor and weapons they can use (see [items.md](items.md#4-armor--weapon-types)). Enemies use the same classes, often with fewer or different abilities, and there are extra enemy-only classes (see [world.md](world.md#3-enemies)).

- The player picks a class at character creation; it is fixed. Name, appearance: TODO
- Every class starts with 100 HP and 150 MP at level 1 (see [mechanics.md](mechanics.md#4-combat-stats)), changed by the class modifiers below

| Class | Affinities | Stat focus / role | Class modifiers | Starting abilities |
|-------|-------|-------------------|-----------------|--------------------|
| Warrior | Physical | High HP, Attack, Defense | 2× passive HP regeneration on the map | **Heavy Strike** (Physical, melee), **Battle Cry** (self: Attack up), **Punch** (Physical, melee; fallback attack) |
| Rogue | Physical, Dark | High Speed and crit chance, poisons, disables | | **Quick Stab** (Physical, melee, high crit, priority +1), **Smoke Veil** (self: Smoke for 2 turns, enemy can't use ranged attacks or targeted spells on you), **Jab** (fallback) |
| Hunter | Physical, Nature | Attack and Speed, ranged, traps | | **Steady Shot** (Physical, ranged), **Venom Arrow** (Nature, ranged, Poison), **Kick** (Physical, melee; fallback, so Smoke doesn't leave only Wait) |
| Mage | Ice, Water, Fire | Highest Magic, low HP and Defense, tricky defenses | 1.5× MP | **Ice Bolt** (Ice, targeted spell, Speed down chance), **Fire Bolt** (Fire, targeted spell; fallback attack, see below) |
| Warlock | Dark | Magic, highest damage over time, HP drain | | **Drain Life** (Dark, targeted spell, Drain: 1/8 max HP per round and heals the Warlock for the same), **Wither** (Dark, targeted spell, 1/16 max HP per round), **Hex Bolt** (Dark, targeted spell; fallback) |
| Priest | Holy | Magic and Resistance, healing | | **Smite** (Holy, targeted spell), **Mend** (self heal), **Light Spark** (Holy, targeted spell; fallback) |
| Paladin | Physical, Holy | Highest Defense, self-heals, lower damage to make up for it | Higher Defense, lower Attack/Magic (see Base Stats) | **Righteous Strike** (Holy, melee), **Holy Seal** (self: physical hits deal extra Holy damage), **Hammer Blow** (Physical, melee; fallback) |
| Shaman | Air, Fire, Water | Balanced Attack and Magic, elemental shocks, totems | | **Storm Bolt** (Air, targeted spell), **Flame Shock** (Fire, targeted spell, Burn), **Zap** (Air, targeted spell; fallback) |
| Druid | Nature | Magic, healing over time, shapeshifting | | **Thorn Lash** (Nature, targeted spell), **Rejuvenate** (self: Regen), **Bramble** (Nature, targeted spell; fallback) |

### 1.1 Base Stats
First proposal, to be tuned by playtesting. Level 1 values:

| Class | HP | MP | Attack | Magic | Defense | Resistance | Speed |
|-------|---:|---:|-------:|------:|--------:|-----------:|------:|
| Warrior | 120 | 150 | 14 | 4 | 13 | 8 | 10 |
| Rogue | 95 | 150 | 14 | 5 | 9 | 8 | **11** |
| Hunter | 100 | 150 | 13 | 6 | 10 | 9 | 10 |
| Mage | 85 | 225 | 4 | 16 | 7 | 12 | 10 |
| Warlock | 95 | 150 | 4 | 14 | 8 | 11 | 10 |
| Priest | 95 | 150 | 4 | 13 | 8 | 13 | 10 |
| Paladin | 115 | 150 | 11 | 8 | 15 | 11 | 10 |
| Shaman | 105 | 150 | 11 | 11 | 11 | 10 | 10 |
| Druid | 100 | 150 | 9 | 13 | 10 | 11 | 10 |

- **Growth per level:** `HP × (1 + 0.09 × (Level − 1))`, every other stat except MP `× (1 + 0.1 × (Level − 1))`. All classes grow at the same rate, so the ratios (and the Rogue's Speed lead) stay the same. MP doesn't grow
- Speed is the same for everyone except the Rogue, who is 1 faster
- Paladin: highest Defense, lower Attack than the Warrior. Mage: 1.5× MP, highest Magic, lowest HP

### 1.2 Ability Numbers
Starting and level-up abilities of every class (the level is when it's learned; Start = at level 1). Power is used in the damage formula ([mechanics.md](mechanics.md#5-damage-formula)). Trainer and tome abilities (section 4) aren't in the game yet. First proposal, to be tuned by playtesting.

| Class | Ability | Level | Power | MP | Notes |
|-------|---------|------:|------:|---:|-------|
| Warrior | Heavy Strike | Start | 50 | 20 | Main attack |
| Warrior | Punch | Start | 30 | 0 | Basic move (fallback attack): free and always available |
| Warrior | Battle Cry | Start | — | 20 | Attack +1 stage |
| Warrior | Overwhelm | 2 | 100 | 25 | Only after a dodge or miss; can't be dodged |
| Warrior | Enrage | 3 | — | — | Passive; always on critical hits taken |
| Warrior | Charge | 6 | 50 | 0 | Priority +1, first round only, once per battle, stuns for that round |
| Warrior | Second Wind | 8 | — | — | Passive: 30% (crit: 100%) to start Regen when taking a hit: 1/16 HP at the end of each of the next 3 rounds; a new trigger restarts it |
| Warrior | Rend | 10 | 20 | 20 | Bleed 1/16 for 10 rounds |
| Mage | Ice Bolt | Start | 55 | 25 | 30% chance: Speed −1 stage |
| Mage | Fire Bolt | Start | 40 | 0 | Fallback attack |
| Mage | Frost Ring | 2 | 45 | 30 | Area spell, 20% chance to Freeze 1 round |
| Mage | Mana Shield | 3 | — | 30 | 5 rounds, priority +2 |
| Mage | Blink | 6 | — | 20 | Evade the next attack |
| Mage | Tidal Splash | 8 | 50 | 25 | Water, targeted spell |
| Rogue | Quick Stab | Start | 45 | 20 | Priority +1, 25% crit chance |
| Rogue | Smoke Veil | Start | — | 20 | Smoke for 2 rounds |
| Rogue | Jab | Start | 30 | 0 | Fallback attack |
| Rogue | Knife Throw | 2 | 45 | 20 | Physical, ranged |
| Rogue | Poisoned Blade | 3 | 20 | 20 | Nature, melee, Poison |
| Rogue | Cheap Shot | 6 | 10 | 25 | Stuns for 2 rounds, once per battle |
| Rogue | Improved Sprint | 6 | — | — | Passive, on the map: sprinting uses up stamina at only 20% of the normal rate (learned along with Cheap Shot) |
| Rogue | Cheat Death | 8 | — | — | Passive: once per battle, an attack that would kill the rogue leaves it at 10% HP instead (damage over time can still kill; learned along with Gut Slash) |
| Rogue | Prey on the Weak | 8 | — | — | Passive: +10% critical hit chance (on top of the base 6.25%) for each level the rogue is above the enemy, up to 100%; no penalty against higher-level enemies |
| Rogue | Poison Resistance | 10 | — | — | Passive: poison does 75% less damage to the rogue (its damage per round, at least 1); other damage over time is unchanged |
| Shaman | Magic Wrath Totem | 4 | — | — | Passive: at the start of every battle the shaman puts down a totem that raises Magic by 1 stage for the battle (learned along with Stone Shock) |
| Shaman | Healing Stream Totem | 4 | — | — | Passive: at the start of every battle the shaman also puts down a totem that heals 1/16 of max HP at the end of every round. Both totems are announced in one message |
| Rogue | Gut Slash | 8 | 75 | 30 | High power |
| Hunter | Steady Shot | Start | 50 | 20 | Physical, ranged |
| Hunter | Venom Arrow | Start | 25 | 20 | Nature, ranged, Poison |
| Hunter | Kick | Start | 30 | 0 | Fallback attack (melee, so Smoke can't stop it) |
| Hunter | Mark Prey | 2 | — | 20 | Ranged; target's Defense −1 stage |
| Hunter | Concussive Shot | 3 | 35 | 20 | Ranged; target's Speed −1 stage |
| Hunter | Volley | 6 | 45 | 30 | Nature, area (physical, so Silence doesn't stop it) |
| Hunter | Savage Strike | 8 | 55 | 20 | Physical, melee |
| Hunter | First Aid | 10 | — | 30 | Heal 30% of max HP |
| Warlock | Drain Life | Start | 20 | 30 | Drain for 5 rounds: 1/8 of the target's max HP per round, healing the Warlock as much |
| Warlock | Wither | Start | 20 | 20 | Wither: 1/16 max HP per round, 10 rounds |
| Warlock | Hex Bolt | Start | 40 | 0 | Fallback attack |
| Warlock | Hellfire Rain | 2 | 45 | 30 | Fire, area spell (early, so a bandit's Smoke Screen can't shut the Warlock down) |
| Warlock | Curse of Weakness | 3 | — | 20 | Targeted spell; target's Attack −1 stage |
| Warlock | Demon Armor | 4 | — | — | Passive: regenerates 1/16 max HP every round |
| Warlock | Dread | 6 | — | 25 | 3 rounds: the target loses its turn half of the time |
| Priest | Smite | Start | 50 | 20 | Holy, targeted spell |
| Priest | Mend | Start | — | 25 | Heal 25% of max HP |
| Priest | Light Spark | Start | 40 | 0 | Fallback attack |
| Priest | Holy Burst | 2 | 45 | 30 | Holy, area spell (early, against Smoke) |
| Priest | Ward of Light | 3 | — | 25 | Shield: absorbs damage up to 25% of max HP |
| Priest | Renewal | 4 | — | 25 | Regen for 8 rounds |
| Priest | Mind Pain | 6 | 20 | 20 | Dark; Pain: 1/16 max HP per round, 10 rounds |
| Paladin | Righteous Strike | Start | 50 | 20 | Holy, melee |
| Paladin | Holy Seal | Start | — | 20 | 5 rounds: every physical hit adds a Holy hit of power 20 |
| Paladin | Hammer Blow | Start | 30 | 0 | Fallback attack |
| Paladin | Judgment Hammer | 2 | 30 | 25 | Holy, melee, priority +1, once per battle; stuns for a random 1–3 rounds |
| Paladin | Devotion | 4 | — | 20 | Defense +1 stage |
| Paladin | Healing Light | 6 | — | 30 | Heal 30% of max HP |
| Shaman | Storm Bolt | Start | 50 | 20 | Air, targeted spell |
| Shaman | Flame Shock | Start | 25 | 20 | Fire, targeted spell, Burn |
| Shaman | Zap | Start | 40 | 0 | Fallback attack |
| Shaman | Chain Lightning | 2 | 45 | 30 | Air, area spell (early, against Smoke) |
| Shaman | Healing Wave | 3 | — | 30 | Heal 30% of max HP |
| Shaman | Stone Shock | 4 | 40 | 25 | Nature; 30% chance to Silence for 2 rounds |
| Shaman | Frost Shock | 6 | 45 | 25 | Ice; target's Speed −1 stage |
| Druid | Thorn Lash | Start | 50 | 20 | Nature, targeted spell |
| Druid | Rejuvenate | Start | — | 25 | Regen for 8 rounds |
| Druid | Bramble | Start | 40 | 0 | Fallback attack |
| Druid | Insect Swarm | 2 | 30 | 30 | Nature, area spell (early, against Smoke); Swarm: 1/16 max HP per round, 10 rounds; 30% chance: Attack −1 stage |
| Druid | Grasping Roots | 3 | 30 | 20 | Nature; target's Speed −1 stage |
| Druid | Moonbeam | 4 | 50 | 25 | Holy, targeted spell |
| Druid | Thorns | 4 | — | — | Passive: a melee attack that hits the druid hurts the attacker for 1/16 of the druid's max HP (at least 1); ranged attacks and spells don't |
| Druid | Bear Form | 6 | — | 30 | Once per battle, for the rest of it: Defense +2 stages and heal 25%; attacks become melee (Smoke and Silence can't stop them) and hit Defense with the Druid's Magic |

- Every caster learns an area spell at level 2 (Frost Ring, Hellfire Rain, Holy Burst, Chain Lightning, Insect Swarm): area spells get through Smoke, so a bandit's Smoke Screen can't leave them with only Wait
- Every class has a 0-MP fallback attack (Punch, Fire Bolt, Jab, Kick, Hex Bolt, Light Spark, Hammer Blow, Zap, Bramble), so running out of MP never leaves only Wait. All other abilities cost 20–30 MP

## 2. Class Matchups
Heroes are always Physical-typed, so every class takes 1× from every element: no weaknesses, resistances or immunities. Class elements only decide which abilities get the Affinity bonus (see [mechanics.md](mechanics.md#63-multi-type-balancing)).

## 3. Abilities & Spells
- Categories: damage, healing, buffs, debuffs, status effects (incl. damage over time and drains, see [mechanics.md](mechanics.md#7-status-effects)), utility
- Each ability has: name, element, category (physical/magical), cost (around 20–30 MP), power, delivery (melee / ranged / targeted spell / area spell / self, see [mechanics.md](mechanics.md#8-melee-vs-ranged)), uses per battle (if limited), cooldown?, effect. No accuracy value: abilities always hit unless a mechanic says otherwise
- Every ability belongs to a class. NPCs use the same class ability pools as the main character
- **Passive** abilities (e.g. Enrage) are always on and don't take an active slot

### 3.1 Ability Loadout
- Characters start with 2 abilities and learn more over time
- Up to **4 active abilities** can be used in battle. **Wait** (do nothing) is always offered on top and doesn't count toward the 4
- **Fallback attacks** (the basic move of each class: the Warrior's Punch, the Mage's Fire Bolt…) cost no MP, don't take a slot either, and are always available. The battle menu has three rows: the 4 ability slots in the top two (an empty slot shows "-"), and the basic move and Wait in the bottom row
- The active 4 are chosen out of combat in the **abilities menu** (key N, see [game-design.md](game-design.md#71-controls))

### 3.2 Learning Abilities
1. **Starting abilities:** every character starts with 2 abilities that fit their class
2. **Level-up:** some class abilities are learned automatically at fixed levels (e.g. 3, 6, 10, then roughly every 5 levels; TODO: tune)
3. **Trainers:** advanced class abilities are bought from class trainers in settlements. Each needs a minimum level and costs money. A settlement only has trainers for 1–2 classes, so finding your own trainer is part of exploring
4. **External sources:** rare and powerful abilities are not taught by anyone. They come from **tomes** found in dungeon chests, **libraries** in ruins or large settlements, or boss drops. A tome is used up when its ability is learned
   - IDEA: rare tomes can teach an ability from another class or element, as a way to cover your own weaknesses

## 4. Class Ability Pools
Level-up abilities come automatically, trainer abilities cost money, tome abilities must be found.

### Warrior
| Source | Abilities |
|--------|-----------|
| Level-up | **Overwhelm** at level 2 (after the enemy dodges or the Warrior misses, it becomes usable for the next turn; very high power and can't be dodged; early, because flying enemies dodge half of the Warrior's attacks), **Enrage** at level 3 (passive: when taking damage, 50% chance to enrage, always on critical hits; +40% damage dealt for 4 rounds; Thorns recoil can trigger it too), **Charge** at level 6 (Physical, melee, Power 50, priority +1, free; only in the first round and once per battle; stuns the target for that round), **Second Wind** at level 8 (passive: when the Warrior takes damage, 30% chance to start Regen for 3 rounds, 1/16 of max HP at the end of each, always on critical hits taken; a new trigger restarts it rather than stacking), Rend (Bleed) at level 10 |
| Trainer | Shield Slam (Stun chance), Armor Break (Defense down), Finishing Blow (big damage vs. enemies under 20% HP), Berserk (Attack up, Defense down) |
| Tome | Unbroken (once per battle: survive a lethal hit with 1 HP) |

### Rogue
| Source | Abilities |
|--------|-----------|
| Level-up | **Knife Throw** at level 2 (Physical, ranged, uses the thrown weapon; the answer to anti-melee traits), Poisoned Blade at level 3 (Nature, melee, Poison), **Cheap Shot** at level 6 (Stun for 2 rounds, once per battle), Gut Slash at level 8 (high power) |
| Trainer | **Blind** (always hits, disables the enemy for 3 turns, damage breaks it; combine with Wait to let enemy buffs run out), Shadow Strike (Dark, melee, crit up), Evade (dodge the next attack) |
| Tome | Death Mark (next hit on the target deals double damage) |

### Hunter
| Source | Abilities |
|--------|-----------|
| Level-up | Mark Prey at level 2 (Defense down), **Concussive Shot** at level 3 (Speed down), **Volley** at level 6 (Nature, ranged, hits an area), Savage Strike at level 8 (Physical, melee: lets Hunters fight with swords, daggers, axes, spears), **First Aid** at level 10 (self: heal 30% of max HP) |
| Trainer | **Freezing Trap** (always hits, Freeze for 2 rounds, damage breaks it), **Feign Death** (evade one attack, twice per battle), Hawk Focus (Attack up), Twin Shot (two hits) |
| Tome | Call of the Wild (a spirit beast attacks alongside you for 3 turns) |

### Mage
| Source | Abilities |
|--------|-----------|
| Level-up | **Frost Ring** at level 2 (Ice, area spell, Freeze chance; early so Smoke can't shut the Mage down), **Mana Shield** at level 3 (priority +2: 50% of damage taken is taken from MP instead of HP), **Blink** at level 6 (evade the opponent's next attack), **Tidal Splash** at level 8 (Water, targeted spell) |
| Trainer | **Time Warp** (+1 Speed stage at the start of each of the next 3 rounds; using Blink during that time still gains the stack, so Time Warp → Blink is a free Speed stage), Flame Burst (Fire, high power), Hailstorm (Ice, area spell, multi-hit), Mutate (turns the enemy into a harmless critter for 2 turns) |
| Tome | Meteor (Fire, very high power, charges for 1 turn) |

### Warlock
The Warlock's identity is the strongest damage over time and HP drain in the game (see [mechanics.md](mechanics.md#7-status-effects)).

| Source | Abilities |
|--------|-----------|
| Level-up | Hellfire Rain at level 2 (Fire, area spell), Curse of Weakness at level 3 (Attack down), **Demon Armor** at level 4 (passive: regenerate 1/16 max HP every round), Dread at level 6 (enemy may skip a turn) |
| Trainer | Shadow Bolt (Dark, targeted spell, direct damage), Hellfire Brand (Fire, Burn), Blood Pact (trade HP for MP) |
| Tome | Soul Harvest (restores MP if it defeats the enemy) |

### Priest
| Source | Abilities |
|--------|-----------|
| Level-up | Holy Burst at level 2 (Holy, area spell), Ward of Light at level 3 (Shield), Renewal at level 4 (Regen), Mind Pain at level 6 (Dark, 1/16 max HP per round) |
| Trainer | Holy Fire (Holy, Burn), Purify (remove status effects), Greater Mend (big heal) |
| Tome | Last Grace (once per battle: survive a lethal hit and heal 30%) |

### Paladin
The Paladin gets higher Defense in general, so their damage is lower.

| Source | Abilities |
|--------|-----------|
| Level-up | **Judgment Hammer** at level 2 (always stuns for a random 1–3 rounds, once per battle; TODO: balance), Devotion at level 4 (Defense up), Healing Light at level 6 (heal) |
| Trainer | **Consecration** (Holy, area spell, 1/16 max HP per round, like a weaker Drain Life without the heal), Exorcism (Holy, high power, extra damage vs. Dark), Cleanse (remove status effects) |
| Tome | **Divine Shield** (immune to all attacks for 2 turns, once per battle; while it is active the Paladin can only cast heals) |

### Shaman
| Source | Abilities |
|--------|-----------|
| Level-up | Chain Lightning at level 2 (Air, area spell), Healing Wave at level 3 (Water, heal), Stone Shock at level 4 (Nature, Silence chance), Frost Shock at level 6 (Ice, Speed down) |
| Trainer | Storm Lash (Air, hits twice), Totem of Gales (Speed up for 3 turns), Stoneskin Totem (Defense up for 3 turns) |
| Tome | Thunderstorm (Air, area spell, very high power, Stun chance) |

### Druid
| Source | Abilities |
|--------|-----------|
| Level-up | Insect Swarm at level 2 (Nature, area spell, 1/16 max HP per round, Attack down), Grasping Roots at level 3 (Nature, Speed down), Moonbeam at level 4 (Holy, targeted spell), Bear Form at level 6 (Defense and HP up, abilities become physical) |
| Trainer | **Cat Form** (Speed and crit up, abilities become physical), Healing Touch (big heal) |
| Tome | Tranquility (full heal, once per battle) |
