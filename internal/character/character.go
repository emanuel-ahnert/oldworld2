// Package character holds the main character's progression: class, level, XP, the global
// HP/MP and the ability loadout (see design/classes.md and design/mechanics.md).
package character

import (
	"math"
	"slices"
	"strings"

	"oldworld2/internal/battle"
	"oldworld2/internal/creature"
)

// Class is a playable class. The order is stored in save files: only append.
type Class int

const (
	Warrior Class = iota
	Mage
	Rogue
	Hunter
	Warlock
	Priest
	Paladin
	Shaman
	Druid
	NumClasses
)

// ClassInfo describes a class (design/classes.md, Base Stats and Ability Numbers).
type ClassInfo struct {
	Name        string
	Affinity    []battle.Element // elements the class gets the Affinity bonus on; heroes are always Physical-typed
	Base        battle.Stats     // level 1
	Start       []string
	Learn       map[int]string // level → ability learned automatically
	Perks       map[int]string // level → a passive learned automatically too, e.g. one for the map
	HPRegen     float64        // multiplier for passive HP regeneration on the map
	Description string         // one line for the new-game screen
}

var classes = map[Class]ClassInfo{
	Warrior: {
		Name:        "Warrior",
		Affinity:    []battle.Element{battle.Physical},
		Base:        battle.Stats{MaxHP: 120, MaxMP: 150, Attack: 14, Magic: 4, Defense: 13, Resistance: 8, Speed: 10},
		Start:       []string{"heavy_strike", "battle_cry", "punch"}, // Punch is the fallback attack
		Learn:       map[int]string{2: "overwhelm", 3: "enrage", 6: "charge", 8: "second_wind", 10: "rend"},
		HPRegen:     2,
		Description: "Tough melee fighter who recovers fast.",
	},
	Mage: {
		Name:        "Mage",
		Affinity:    []battle.Element{battle.Ice, battle.Water, battle.Fire},
		Base:        battle.Stats{MaxHP: 85, MaxMP: 225, Attack: 4, Magic: 16, Defense: 7, Resistance: 12, Speed: 10},
		Start:       []string{"ice_bolt", "fire_bolt"},                                                // Fire Bolt is the fallback attack
		Learn:       map[int]string{2: "frost_ring", 3: "mana_shield", 6: "blink", 8: "tidal_splash"}, // early area spell against Smoke
		HPRegen:     1,
		Description: "Fragile caster of ice, water and fire.",
	},
	Rogue: {
		Name:        "Rogue",
		Affinity:    []battle.Element{battle.Physical, battle.Dark},
		Base:        battle.Stats{MaxHP: 95, MaxMP: 150, Attack: 14, Magic: 5, Defense: 9, Resistance: 8, Speed: 11},
		Start:       []string{"rogue_quick_stab", "rogue_smoke_veil", "jab"},
		Learn:       map[int]string{2: "knife_throw", 3: "rogue_poisoned_blade", 6: "kidney_shot", 8: "gut_slash"},
		Perks:       map[int]string{6: "improved_sprint"},
		HPRegen:     1,
		Description: "Fast striker with poisons and smoke.",
	},
	Hunter: {
		Name:        "Hunter",
		Affinity:    []battle.Element{battle.Physical, battle.Nature},
		Base:        battle.Stats{MaxHP: 100, MaxMP: 150, Attack: 13, Magic: 6, Defense: 10, Resistance: 9, Speed: 10},
		Start:       []string{"steady_shot", "venom_arrow", "kick"},
		Learn:       map[int]string{2: "mark_prey", 3: "concussive_shot", 6: "volley", 8: "savage_strike", 10: "first_aid"},
		HPRegen:     1,
		Description: "Archer who wears enemies down from afar.",
	},
	Warlock: {
		Name:        "Warlock",
		Affinity:    []battle.Element{battle.Dark},
		Base:        battle.Stats{MaxHP: 95, MaxMP: 150, Attack: 4, Magic: 14, Defense: 8, Resistance: 11, Speed: 10},
		Start:       []string{"drain_life", "wither", "hex_bolt"},
		Learn:       map[int]string{2: "hellfire_rain", 3: "curse_of_weakness", 4: "demon_armor", 6: "dread"},
		HPRegen:     1,
		Description: "Dark caster: curses, drains and decay.",
	},
	Priest: {
		Name:        "Priest",
		Affinity:    []battle.Element{battle.Holy},
		Base:        battle.Stats{MaxHP: 95, MaxMP: 150, Attack: 4, Magic: 13, Defense: 8, Resistance: 13, Speed: 10},
		Start:       []string{"smite", "mend", "light_spark"},
		Learn:       map[int]string{2: "holy_burst", 3: "ward_of_light", 4: "renewal", 6: "mind_pain"},
		HPRegen:     1,
		Description: "Holy caster who heals and shields.",
	},
	Paladin: {
		Name:        "Paladin",
		Affinity:    []battle.Element{battle.Physical, battle.Holy},
		Base:        battle.Stats{MaxHP: 115, MaxMP: 150, Attack: 11, Magic: 8, Defense: 15, Resistance: 11, Speed: 10},
		Start:       []string{"righteous_strike", "holy_seal", "hammer_blow"},
		Learn:       map[int]string{2: "judgment_hammer", 4: "devotion", 6: "healing_light"},
		HPRegen:     1,
		Description: "Holy knight: the toughest defense.",
	},
	Shaman: {
		Name:        "Shaman",
		Affinity:    []battle.Element{battle.Air, battle.Fire, battle.Water},
		Base:        battle.Stats{MaxHP: 105, MaxMP: 150, Attack: 11, Magic: 11, Defense: 11, Resistance: 10, Speed: 10},
		Start:       []string{"storm_bolt", "flame_shock", "zap"},
		Learn:       map[int]string{2: "chain_lightning", 3: "healing_wave", 4: "stone_shock", 6: "frost_shock"},
		HPRegen:     1,
		Description: "Caller of storm, flame and water.",
	},
	Druid: {
		Name:        "Druid",
		Affinity:    []battle.Element{battle.Nature},
		Base:        battle.Stats{MaxHP: 100, MaxMP: 150, Attack: 9, Magic: 13, Defense: 10, Resistance: 11, Speed: 10},
		Start:       []string{"thorn_lash", "rejuvenate", "bramble"},
		Learn:       map[int]string{2: "insect_swarm", 3: "grasping_roots", 4: "moonbeam", 6: "bear_form"},
		HPRegen:     1,
		Description: "Nature caster: thorns, regrowth, bear form.",
	},
}

// Info returns the description of a class.
func (c Class) Info() ClassInfo { return classes[c] }

// ParseClass maps a name like "mage" to a class.
func ParseClass(name string) (Class, bool) {
	for c := Class(0); c < NumClasses; c++ {
		if strings.EqualFold(name, c.Info().Name) {
			return c, true
		}
	}
	return 0, false
}

// MaxActive is how many regular abilities can be used in battle. Wait, passives and fallback
// attacks don't count toward it.
const MaxActive = 4

// Passive regeneration on the adventure map, per second (design/mechanics.md, HP & MP).
const (
	hpRegenPerSecond = 0.10 / 60
	mpRegenPerSecond = 0.50 / 60
)

// Character is the main character.
type Character struct {
	Class  Class
	Level  int
	XP     int
	HP, MP float64 // global: the same on the map and in battle
	Money  int     // in bronze
	Known  []string
	Active []string // up to MaxActive regular abilities (not passives or fallback attacks)
}

// New creates a level 1 character with full HP and MP.
func New(c Class) *Character {
	ch := &Character{Class: c, Level: 1}
	for _, id := range c.Info().Start {
		ch.learn(id)
	}
	s := ch.Stats()
	ch.HP, ch.MP = float64(s.MaxHP), float64(s.MaxMP)
	return ch
}

// Stats returns the stats at the current level: HP grows 9% and the other stats 10% of the
// level 1 value per level; MP doesn't grow (design/classes.md, Base Stats).
func (c *Character) Stats() battle.Stats {
	return grow(c.Class.Info().Base, c.Level)
}

func grow(b battle.Stats, level int) battle.Stats {
	l := float64(level - 1)
	g := func(v int) int { return int(math.Round(float64(v) * (1 + 0.1*l))) }
	return battle.Stats{
		MaxHP:  int(math.Round(float64(b.MaxHP) * (1 + 0.09*l))),
		MaxMP:  b.MaxMP,
		Attack: g(b.Attack), Magic: g(b.Magic),
		Defense: g(b.Defense), Resistance: g(b.Resistance), Speed: g(b.Speed),
	}
}

func (c *Character) learn(id string) {
	c.Known = append(c.Known, id)
	if a := battle.Get(id); a != nil && !a.Passive && !a.Fallback && len(c.Active) < MaxActive {
		c.Active = append(c.Active, id)
	}
}

// StaminaUse is how fast sprinting uses up stamina (endurance) on the map: 1 normally, less
// with Improved Sprint.
func (c *Character) StaminaUse() float64 {
	if slices.Contains(c.Known, "improved_sprint") {
		return improvedSprintUse
	}
	return 1
}

const improvedSprintUse = 0.2

// Slottable reports whether an ability can be put into an active slot: a known regular
// ability (not a passive or fallback attack).
func (c *Character) Slottable(id string) bool {
	a := battle.Get(id)
	return a != nil && !a.Passive && !a.Fallback && slices.Contains(c.Known, id)
}

// Equip puts an ability into the first free slot. It reports false if the slots are full or
// the ability can't be slotted.
func (c *Character) Equip(id string) bool {
	if !c.Slottable(id) || slices.Contains(c.Active, id) || len(c.Active) >= MaxActive {
		return false
	}
	c.Active = append(c.Active, id)
	return true
}

// Replace puts an ability into slot i, replacing what was there.
func (c *Character) Replace(i int, id string) bool {
	if !c.Slottable(id) || slices.Contains(c.Active, id) || i < 0 || i >= len(c.Active) {
		return false
	}
	c.Active[i] = id
	return true
}

// Unequip takes the ability out of slot i.
func (c *Character) Unequip(i int) {
	if i >= 0 && i < len(c.Active) {
		c.Active = slices.Delete(c.Active, i, i+1)
	}
}

// Combatant builds the battle side of the character from its current state.
func (c *Character) Combatant() *battle.Combatant {
	info := c.Class.Info()
	abilities := append([]string(nil), c.Active...)
	for _, id := range c.Known {
		if a := battle.Get(id); a != nil && (a.Passive || a.Fallback) {
			abilities = append(abilities, id)
		}
	}
	return &battle.Combatant{
		Name: info.Name, You: true, Level: c.Level, Types: []battle.Element{battle.Physical}, Affinity: info.Affinity, Base: c.Stats(),
		HP: int(math.Ceil(c.HP)), MP: int(math.Floor(c.MP)), Abilities: abilities,
	}
}

// Regenerate restores HP and MP for one frame on the adventure map.
func (c *Character) Regenerate() {
	s := c.Stats()
	c.HP = math.Min(float64(s.MaxHP), c.HP+float64(s.MaxHP)*hpRegenPerSecond/60*c.Class.Info().HPRegen)
	c.MP = math.Min(float64(s.MaxMP), c.MP+float64(s.MaxMP)*mpRegenPerSecond/60)
}

// Regeneration while resting at a camp fire, as a share of the maximum per second.
const (
	campHPPerSecond = 0.10
	campMPPerSecond = 0.20
)

// RegenerateAtCamp restores HP (10% per second) and MP (20% per second) for one frame at a
// camp fire.
func (c *Character) RegenerateAtCamp() {
	s := c.Stats()
	c.HP = math.Min(float64(s.MaxHP), c.HP+float64(s.MaxHP)*campHPPerSecond/60)
	c.MP = math.Min(float64(s.MaxMP), c.MP+float64(s.MaxMP)*campMPPerSecond/60)
}

// Heal fully restores HP and MP (checkpoints, respawn).
func (c *Character) Heal() {
	s := c.Stats()
	c.HP, c.MP = float64(s.MaxHP), float64(s.MaxMP)
}

// XPToNext is the XP needed to go from the current level to the next
// (design/mechanics.md, XP Curve).
func XPToNext(level int) int {
	kills := math.Round(2 + 2*math.Log2(float64(level)))
	return int(kills) * 10 * level
}

// EnemyXP is the XP for defeating an enemy.
func EnemyXP(enemyLevel, playerLevel int, tier string) int {
	tierMod := 1.0
	if tier == "elite" {
		tierMod = 2
	}
	levelMod := math.Min(1.5, math.Max(0.1, 1+0.1*float64(enemyLevel-playerLevel)))
	return max(1, int(math.Round(10*float64(enemyLevel)*tierMod*levelMod)))
}

// LevelUp describes one level gained.
type LevelUp struct {
	Level   int
	Learned []string // ability IDs learned at this level
}

// GainXP adds XP and returns the levels gained.
func (c *Character) GainXP(xp int) []LevelUp {
	var ups []LevelUp
	c.XP += xp
	for c.XP >= XPToNext(c.Level) {
		c.XP -= XPToNext(c.Level)
		c.Level++
		up := LevelUp{Level: c.Level}
		info := c.Class.Info()
		for _, learned := range []map[int]string{info.Learn, info.Perks} {
			if id, ok := learned[c.Level]; ok {
				c.learn(id)
				up.Learned = append(up.Learned, id)
			}
		}
		ups = append(ups, up)
	}
	return ups
}

// Defeated applies the death penalty: lose 10% of the money, respawn fully healed
// (design/mechanics.md, Win / Lose).
func (c *Character) Defeated() (lost int) {
	lost = c.Money / 10
	c.Money -= lost
	c.Heal()
	return lost
}

// Enemy builds a battle combatant for a creature at a level. Enemy HP and stats grow like
// player stats.
func Enemy(k creature.Kind, level int) *battle.Combatant {
	info := creature.Get(k)
	s := grow(info.Stats, level)
	return &battle.Combatant{
		Name: info.Name, Level: level, Types: info.Types, Base: s,
		HP: s.MaxHP, MP: s.MaxMP, Abilities: info.Abilities, Flying: info.Flying,
	}
}
