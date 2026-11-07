// Package creature defines the enemy types: how they look, how they move and how
// aggressive they are (see design/world.md, Enemies, and design/mechanics.md,
// Aggro & Movement).
package creature

import "oldworld2/internal/battle"

// Kind identifies a creature type.
type Kind int

const (
	Wolf Kind = iota
	Boar
	Goblin
	Bandit
	SandSnake
	Scorpion
	EmberSpirit
	NomadRaider
	Mummy
	Scarab
	Cobra
	TombGuardian
	SnowWolf
	FrostWisp
	IceTroll
	Berserker
	GiantCrab
	WindSpirit
	Corsair
	Hoplite
	BatSwarm
	DireWolf
	Cultist
	Vampire
	numKinds
)

// Aggro ranges in tiles.
const (
	Defensive  = 1.0 // only fights when the player is right next to it
	Passive    = 4.0
	Normal     = 6.5
	Aggressive = 10.5
)

// Movement speeds as frames per tile at 60 TPS.
const (
	Slow     = 18
	Medium   = 9
	Fast     = 7
	VeryFast = 5
)

// Info describes a creature type.
type Info struct {
	Name       string
	Tier       string  // trash or elite, see design/world.md Enemy Tiers
	Aggro      float64 // tiles
	StepFrames int
	LeashMin   int // seconds a chase lasts at least
	LeashMax   int // seconds a chase lasts at most
	GroupMin   int
	GroupMax   int
	Flying     bool

	// Battle stats at level 1; they grow per level like player stats (design/world.md).
	Types     []battle.Element
	Stats     battle.Stats
	Abilities []string
}

var infos = [numKinds]Info{
	Wolf: {
		Name: "Wolf", Tier: "trash",
		Aggro: Aggressive, StepFrames: Fast, LeashMin: 8, LeashMax: 15,
		GroupMin: 2, GroupMax: 3,
		Types:     []battle.Element{battle.Physical},
		Stats:     battle.Stats{MaxHP: 50, Attack: 8, Magic: 2, Defense: 10, Resistance: 8, Speed: 12},
		Abilities: []string{"bite", "howl"},
	},
	Boar: {
		Name: "Boar", Tier: "trash",
		Aggro: Defensive, StepFrames: Medium, LeashMin: 5, LeashMax: 8,
		GroupMin: 1, GroupMax: 2,
		Types:     []battle.Element{battle.Physical, battle.Nature},
		Stats:     battle.Stats{MaxHP: 60, Attack: 9, Magic: 2, Defense: 12, Resistance: 6, Speed: 8},
		Abilities: []string{"gore", "tusk_charge"},
	},
	Goblin: {
		Name: "Goblin", Tier: "trash",
		Aggro: Normal, StepFrames: Fast, LeashMin: 6, LeashMax: 12,
		GroupMin: 2, GroupMax: 3,
		Types:     []battle.Element{battle.Physical, battle.Dark},
		Stats:     battle.Stats{MaxHP: 45, Attack: 9, Magic: 2, Defense: 8, Resistance: 7, Speed: 12},
		Abilities: []string{"club", "sling_stone"},
	},
	Bandit: {
		Name: "Bandit", Tier: "elite",
		Aggro: Normal, StepFrames: Medium, LeashMin: 10, LeashMax: 15,
		GroupMin: 2, GroupMax: 3,
		Types:     []battle.Element{battle.Physical, battle.Dark},
		Stats:     battle.Stats{MaxHP: 80, MaxMP: 100, Attack: 13, Magic: 4, Defense: 9, Resistance: 8, Speed: 11},
		Abilities: []string{"quick_stab", "poisoned_blade", "smoke_screen"},
	},

	// Desert (design/world.md, Zones: nomad raiders, scorpions, fire spirits).
	SandSnake: {
		Name: "Sand Snake", Tier: "trash",
		Aggro: Normal, StepFrames: Fast, LeashMin: 6, LeashMax: 12,
		GroupMin: 1, GroupMax: 2,
		Types:     []battle.Element{battle.Nature},
		Stats:     battle.Stats{MaxHP: 45, Attack: 9, Magic: 3, Defense: 9, Resistance: 8, Speed: 12},
		Abilities: []string{"venom_bite", "coil"},
	},
	Scorpion: {
		Name: "Scorpion", Tier: "trash",
		Aggro: Defensive, StepFrames: Medium, LeashMin: 5, LeashMax: 8,
		GroupMin: 1, GroupMax: 2,
		Types:     []battle.Element{battle.Physical, battle.Nature},
		Stats:     battle.Stats{MaxHP: 55, Attack: 10, Magic: 2, Defense: 13, Resistance: 6, Speed: 9},
		Abilities: []string{"pincer", "sting"},
	},
	EmberSpirit: {
		Name: "Ember Spirit", Tier: "trash",
		Aggro: Passive, StepFrames: Medium, LeashMin: 5, LeashMax: 10,
		GroupMin: 1, GroupMax: 2,
		Types:     []battle.Element{battle.Fire},
		Stats:     battle.Stats{MaxHP: 40, MaxMP: 60, Attack: 4, Magic: 11, Defense: 7, Resistance: 11, Speed: 12},
		Abilities: []string{"ember", "flare"},
	},
	NomadRaider: {
		Name: "Nomad Raider", Tier: "elite",
		Aggro: Normal, StepFrames: Medium, LeashMin: 10, LeashMax: 15,
		GroupMin: 2, GroupMax: 3,
		Types:     []battle.Element{battle.Physical, battle.Fire},
		Stats:     battle.Stats{MaxHP: 85, MaxMP: 100, Attack: 14, Magic: 6, Defense: 10, Resistance: 8, Speed: 11},
		Abilities: []string{"scimitar_slash", "fire_brand", "sand_veil"},
	},

	// Ancient (Egyptian) desert.
	Mummy: {
		Name: "Mummy", Tier: "trash",
		Aggro: Normal, StepFrames: Slow, LeashMin: 10, LeashMax: 15,
		GroupMin: 1, GroupMax: 2,
		Types:     []battle.Element{battle.Physical, battle.Dark},
		Stats:     battle.Stats{MaxHP: 60, MaxMP: 60, Attack: 10, Magic: 6, Defense: 12, Resistance: 6, Speed: 6},
		Abilities: []string{"wrap", "mummy_curse"},
	},
	Scarab: {
		Name: "Scarab", Tier: "trash",
		Aggro: Aggressive, StepFrames: Fast, LeashMin: 6, LeashMax: 10,
		GroupMin: 2, GroupMax: 3,
		Types:     []battle.Element{battle.Physical, battle.Nature},
		Stats:     battle.Stats{MaxHP: 40, Attack: 9, Magic: 2, Defense: 12, Resistance: 7, Speed: 13},
		Abilities: []string{"mandible", "carapace"},
	},
	Cobra: {
		Name: "Cobra", Tier: "trash",
		Aggro: Normal, StepFrames: Medium, LeashMin: 6, LeashMax: 12,
		GroupMin: 1, GroupMax: 1,
		Types:     []battle.Element{battle.Nature},
		Stats:     battle.Stats{MaxHP: 45, Attack: 9, Magic: 4, Defense: 8, Resistance: 9, Speed: 12},
		Abilities: []string{"venom_spit", "hood_flare"},
	},
	TombGuardian: {
		Name: "Tomb Guardian", Tier: "elite",
		Aggro: Normal, StepFrames: Slow, LeashMin: 8, LeashMax: 12,
		GroupMin: 1, GroupMax: 2,
		Types:     []battle.Element{battle.Physical, battle.Holy},
		Stats:     battle.Stats{MaxHP: 95, MaxMP: 100, Attack: 14, Magic: 10, Defense: 14, Resistance: 10, Speed: 8},
		Abilities: []string{"sun_strike", "judgment_ray", "sacred_ward"},
	},

	// The North.
	SnowWolf: {
		Name: "Snow Wolf", Tier: "trash",
		Aggro: Aggressive, StepFrames: Fast, LeashMin: 8, LeashMax: 15,
		GroupMin: 2, GroupMax: 3,
		Types:     []battle.Element{battle.Ice},
		Stats:     battle.Stats{MaxHP: 50, MaxMP: 60, Attack: 9, Magic: 3, Defense: 9, Resistance: 9, Speed: 12},
		Abilities: []string{"frost_bite", "howl"},
	},
	FrostWisp: {
		Name: "Frost Wisp", Tier: "trash",
		Aggro: Passive, StepFrames: Medium, LeashMin: 5, LeashMax: 10,
		GroupMin: 1, GroupMax: 2, Flying: true,
		Types:     []battle.Element{battle.Ice},
		Stats:     battle.Stats{MaxHP: 40, MaxMP: 60, Attack: 4, Magic: 11, Defense: 7, Resistance: 11, Speed: 11},
		Abilities: []string{"ice_shard", "chill_wind"},
	},
	IceTroll: {
		Name: "Ice Troll", Tier: "trash",
		Aggro: Normal, StepFrames: Slow, LeashMin: 8, LeashMax: 12,
		GroupMin: 1, GroupMax: 1,
		Types:     []battle.Element{battle.Physical, battle.Ice},
		Stats:     battle.Stats{MaxHP: 75, MaxMP: 60, Attack: 12, Magic: 4, Defense: 13, Resistance: 6, Speed: 5},
		Abilities: []string{"troll_smash", "hurl_ice", "frozen_hide"},
	},
	Berserker: {
		Name: "Berserker", Tier: "elite",
		Aggro: Normal, StepFrames: Medium, LeashMin: 10, LeashMax: 15,
		GroupMin: 1, GroupMax: 2,
		Types:     []battle.Element{battle.Physical, battle.Ice},
		Stats:     battle.Stats{MaxHP: 90, MaxMP: 100, Attack: 15, Magic: 4, Defense: 8, Resistance: 8, Speed: 11},
		Abilities: []string{"cleave", "berserk", "frost_axe"},
	},

	// The Mediterranean.
	GiantCrab: {
		Name: "Giant Crab", Tier: "trash",
		Aggro: Defensive, StepFrames: Medium, LeashMin: 5, LeashMax: 8,
		GroupMin: 1, GroupMax: 2,
		Types:     []battle.Element{battle.Water, battle.Physical},
		Stats:     battle.Stats{MaxHP: 60, MaxMP: 60, Attack: 10, Magic: 4, Defense: 15, Resistance: 6, Speed: 7},
		Abilities: []string{"pinch", "bubble_spray", "hard_shell"},
	},
	WindSpirit: {
		Name: "Wind Spirit", Tier: "trash",
		Aggro: Passive, StepFrames: Fast, LeashMin: 5, LeashMax: 10,
		GroupMin: 1, GroupMax: 2, Flying: true,
		Types:     []battle.Element{battle.Air},
		Stats:     battle.Stats{MaxHP: 40, MaxMP: 60, Attack: 4, Magic: 11, Defense: 7, Resistance: 10, Speed: 14},
		Abilities: []string{"gust", "whirlwind"},
	},
	Corsair: {
		Name: "Corsair", Tier: "elite",
		Aggro: Normal, StepFrames: Medium, LeashMin: 10, LeashMax: 15,
		GroupMin: 2, GroupMax: 3,
		Types:     []battle.Element{battle.Physical, battle.Water},
		Stats:     battle.Stats{MaxHP: 85, MaxMP: 100, Attack: 14, Magic: 5, Defense: 9, Resistance: 8, Speed: 12},
		Abilities: []string{"cutlass_slash", "pistol_shot", "smoke_flare"},
	},
	Hoplite: {
		Name: "Hoplite", Tier: "elite",
		Aggro: Normal, StepFrames: Medium, LeashMin: 8, LeashMax: 12,
		GroupMin: 1, GroupMax: 2,
		Types:     []battle.Element{battle.Physical, battle.Holy},
		Stats:     battle.Stats{MaxHP: 95, MaxMP: 100, Attack: 13, Magic: 6, Defense: 15, Resistance: 9, Speed: 9},
		Abilities: []string{"spear_thrust", "shield_wall", "sun_spear"},
	},

	// The unholy lands.
	BatSwarm: {
		Name: "Bat Swarm", Tier: "trash",
		Aggro: Aggressive, StepFrames: Fast, LeashMin: 6, LeashMax: 10,
		GroupMin: 1, GroupMax: 2, Flying: true,
		Types:     []battle.Element{battle.Dark},
		Stats:     battle.Stats{MaxHP: 40, MaxMP: 60, Attack: 9, Magic: 4, Defense: 7, Resistance: 8, Speed: 14},
		Abilities: []string{"bat_bites", "screech"},
	},
	DireWolf: {
		Name: "Dire Wolf", Tier: "trash",
		Aggro: Aggressive, StepFrames: Fast, LeashMin: 8, LeashMax: 15,
		GroupMin: 1, GroupMax: 3,
		Types:     []battle.Element{battle.Dark, battle.Physical},
		Stats:     battle.Stats{MaxHP: 58, MaxMP: 60, Attack: 10, Magic: 3, Defense: 10, Resistance: 8, Speed: 12},
		Abilities: []string{"savage_bite", "dark_howl"},
	},
	Cultist: {
		Name: "Cultist", Tier: "elite",
		Aggro: Normal, StepFrames: Medium, LeashMin: 10, LeashMax: 15,
		GroupMin: 2, GroupMax: 3,
		Types:     []battle.Element{battle.Dark, battle.Fire},
		Stats:     battle.Stats{MaxHP: 80, MaxMP: 100, Attack: 8, Magic: 14, Defense: 8, Resistance: 11, Speed: 10},
		Abilities: []string{"shadow_bolt", "fire_brand", "dark_pact"},
	},
	Vampire: {
		Name: "Vampire", Tier: "elite",
		Aggro: Normal, StepFrames: Medium, LeashMin: 10, LeashMax: 15,
		GroupMin: 1, GroupMax: 1,
		Types:     []battle.Element{battle.Dark},
		Stats:     battle.Stats{MaxHP: 100, MaxMP: 100, Attack: 14, Magic: 12, Defense: 10, Resistance: 11, Speed: 12},
		Abilities: []string{"life_drain", "vampire_claw", "mesmerize"},
	},
}

// Get returns the description of a creature type.
func Get(k Kind) Info { return infos[k] }

// All lists every creature type.
func All() []Kind {
	out := make([]Kind, numKinds)
	for i := range out {
		out[i] = Kind(i)
	}
	return out
}
