package battle

// Category picks the stat pair used for damage.
type Category int

const (
	PhysicalCat Category = iota // Attack vs. Defense
	MagicalCat                  // Magic vs. Resistance
	StatusCat                   // no damage
)

// Delivery decides what an ability can get through (see design/mechanics.md, Melee vs. Ranged).
type Delivery int

const (
	Melee Delivery = iota
	Ranged
	TargetedSpell
	AreaSpell
	Self
)

// Stat is a stat that can be raised or lowered in stages.
type Stat int

const (
	Attack Stat = iota
	Magic
	Defense
	Resistance
	Speed
	numStats
)

var statNames = [numStats]string{"Attack", "Magic", "Defense", "Resistance", "Speed"}

func (s Stat) String() string { return statNames[s] }

// Special marks abilities with rules of their own.
type Special int

const (
	NoSpecial      Special = iota
	Overwhelm              // usable only right after the user missed or was dodged; can't be dodged
	ManaShield             // half of the damage taken comes out of MP
	Blink                  // melee attacks can't reach the user for Duration rounds
	Smoke                  // ranged attacks and targeted spells can't target the user
	Enrage                 // passive: taking damage may enrage (+40% damage), always on critical hits
	SecondWind             // passive: taking a hit may start Regen (30%, always on critical hits)
	Shield                 // absorbs damage up to a share of max HP
	HolySeal               // physical hits deal extra Holy damage
	BearForm               // rest of the battle: Defense up, attacks are melee and hit Defense
	DemonArmor             // passive: heals 1/16 max HP every round
	CheatDeath             // passive: once per battle, an attack that would defeat the user leaves it at 10% HP
	WrathTotem             // passive: puts down a totem at the start of a battle, Magic +1 stage
	HealingTotem           // passive: puts down a totem at the start of a battle that heals 1/16 max HP every round
	PreyOnWeak             // passive: +10% crit chance for each level the user is above the target
	PoisonResist           // passive: poison does 75% less damage
	Thorns                 // passive: melee attackers that hit the user take 1/16 of its max HP
	IceBlock               // the user is immune to attacks for a few rounds and can only wait
	CancelIceBlock         // ends the user's Ice Block
)

// StatChange raises or lowers a stat of the user or the target.
type StatChange struct {
	Stat   Stat
	Stages int
	OnSelf bool
	Chance float64 // 0 means always
}

// Inflict applies a status effect to the target.
type Inflict struct {
	Status      StatusKind
	Chance      float64 // 0 means always
	Duration    int     // rounds; 0 uses the status default
	MaxDuration int     // if above Duration: a random duration from Duration to MaxDuration
}

// Ability is one combat move. Abilities are data; see design/classes.md for the pools.
type Ability struct {
	ID       string
	Name     string
	Element  Element
	Alt      []Element // other elements it can strike with: the most effective one applies
	Category Category
	Delivery Delivery
	Power    int
	Cost     int     // MP, rage or energy, whichever the user has
	Gain     int     // resource gained by using it (the warrior's rage)
	Priority int     // higher goes first, regardless of Speed (like Pokemon move priority)
	CritRate float64 // 0 uses the base rate
	Passive  bool    // always on, never selected in battle
	Uses     int     // uses per battle; 0 = unlimited
	Opener   bool    // only usable in the first round of a battle
	// Fallback attacks are the free basic move of a class: they don't take an ability slot
	// and are always offered in battle, next to Wait.
	Fallback bool
	Special  Special
	Duration int     // rounds, for self effects like Mana Shield
	Heal     float64 // self: restores this share of max HP
	Regen    int     // self: rounds of Regen (1/16 max HP per round)
	Stats    []StatChange
	Inflict  []Inflict
	Text     string // short description for menus
}

// Elements names the ability's element for menus, e.g. "Ice/Fire" for Frostfire Bolt.
func (a *Ability) Elements() string {
	s := a.Element.String()
	for _, e := range a.Alt {
		s += "/" + e.String()
	}
	return s
}

// WaitID is the "do nothing" move everyone has.
const WaitID = "wait"

// CancelIceBlockID is the move that takes Ice Block's place in the menu while it is up.
const CancelIceBlockID = "cancel_ice_block"

var abilities = map[string]*Ability{}

func register(list ...Ability) {
	for i := range list {
		a := list[i]
		abilities[a.ID] = &a
	}
}

// Get returns an ability by ID, or nil.
func Get(id string) *Ability { return abilities[id] }

func init() {
	register(
		Ability{ID: WaitID, Name: "Wait", Category: StatusCat, Delivery: Self, Text: "Do nothing this turn"},

		// Warrior (design/classes.md, numbers in 1.2).
		Ability{ID: "heavy_strike", Name: "Heavy Strike", Element: Physical, Category: PhysicalCat, Delivery: Melee, Power: 50, Cost: 15,
			Text: "A strong melee strike"},
		Ability{ID: "punch", Name: "Punch", Element: Physical, Category: PhysicalCat, Delivery: Melee, Power: 30, Fallback: true, Gain: 10,
			Text: "A free attack that builds rage"},
		Ability{ID: "battle_cry", Name: "Battle Cry", Category: StatusCat, Delivery: Self, Cost: 20,
			Stats: []StatChange{{Stat: Attack, Stages: 1, OnSelf: true}}, Text: "Attack up"},
		Ability{ID: "enrage", Name: "Enrage", Passive: true, Special: Enrage,
			Text: "Passive: taking damage may enrage you (+40% damage); critical hits always do"},
		Ability{ID: "charge", Name: "Charge", Element: Physical, Category: PhysicalCat, Delivery: Melee, Power: 50, Priority: 1,
			Uses: 1, Opener: true, Gain: 15, Inflict: []Inflict{{Status: Stun, Duration: 1}},
			Text: "First round only, once: stuns the target for this round"},
		Ability{ID: "second_wind", Name: "Second Wind", Passive: true, Special: SecondWind,
			Text: "Passive: hits may start Regen, 1/16 HP for 3 rounds (30%); critical hits always do"},
		Ability{ID: "overwhelm", Name: "Overwhelm", Element: Physical, Category: PhysicalCat, Delivery: Melee, Power: 100,
			Special: Overwhelm, Text: "Only after a miss or dodge; can't be dodged"},
		Ability{ID: "rend", Name: "Rend", Element: Physical, Category: PhysicalCat, Delivery: Melee, Power: 20, Cost: 20,
			Inflict: []Inflict{{Status: Bleed}}, Text: "Bleed: 1/16 HP per round"},

		// Mage.
		Ability{ID: "fire_bolt", Name: "Fire Bolt", Element: Fire, Category: MagicalCat, Delivery: TargetedSpell, Power: 40, Fallback: true,
			Text: "A free attack, always ready"},
		Ability{ID: "ice_bolt", Name: "Ice Bolt", Element: Ice, Category: MagicalCat, Delivery: TargetedSpell, Power: 55, Cost: 25,
			Stats: []StatChange{{Stat: Speed, Stages: -1}}, Text: "Target's Speed down"},
		Ability{ID: "tidal_splash", Name: "Tidal Splash", Element: Water, Category: MagicalCat, Delivery: TargetedSpell, Power: 50, Cost: 25,
			Text: "A wave of water"},
		Ability{ID: "mana_shield", Name: "Mana Shield", Category: StatusCat, Delivery: Self, Cost: 30, Special: ManaShield, Duration: 5, Priority: 2,
			Text: "Half of the damage taken drains MP instead"},
		Ability{ID: "frostfire_bolt", Name: "Frostfire Bolt", Element: Ice, Alt: []Element{Fire}, Category: MagicalCat, Delivery: TargetedSpell,
			Power: 55, Cost: 25, Stats: []StatChange{{Stat: Speed, Stages: -1}}, Text: "Ice or Fire, whichever hurts the target more; Speed down"},
		Ability{ID: "arcane_explosion", Name: "Arcane Explosion", Element: Arcane, Category: MagicalCat, Delivery: AreaSpell, Power: 60, Cost: 30,
			Text: "Area spell: raw arcane power, neutral to every type (gets through smoke)"},
		Ability{ID: "blink", Name: "Blink", Category: StatusCat, Delivery: Self, Cost: 20, Special: Blink, Priority: 2, Duration: 2,
			Text: "Melee attacks can't reach you this round and the next"},
		Ability{ID: "ignite", Name: "Ignite", Element: Fire, Category: StatusCat, Delivery: TargetedSpell, Cost: 20,
			Inflict: []Inflict{{Status: Ignite, Duration: igniteRounds}}, Text: "Ignite: 1/16 HP fire damage per round, 8 rounds"},
		Ability{ID: "ice_block", Name: "Ice Block", Category: StatusCat, Delivery: Self, Cost: 25, Special: IceBlock, Duration: 2, Priority: 1,
			Text: "2 rounds: immune to attacks, but you can only wait or cancel it"},
		Ability{ID: CancelIceBlockID, Name: "Cancel", Category: StatusCat, Delivery: Self, Special: CancelIceBlock,
			Text: "Break out of the ice block, then choose this round's move"},
		Ability{ID: "frost_ring", Name: "Frost Ring", Element: Ice, Category: MagicalCat, Delivery: AreaSpell, Power: 45, Cost: 30,
			Inflict: []Inflict{{Status: Freeze, Chance: 0.2, Duration: 1}}, Text: "20%: Freeze"},

		// Rogue. Quick Stab and Poisoned Blade also exist as free enemy moves (the bandit's).
		Ability{ID: "rogue_quick_stab", Name: "Quick Stab", Element: Physical, Category: PhysicalCat, Delivery: Melee, Power: 45,
			CritRate: 0.25, Priority: 1, Cost: 40, Text: "Goes first; high critical rate"},
		Ability{ID: "rogue_smoke_veil", Name: "Smoke Veil", Category: StatusCat, Delivery: Self, Special: Smoke, Duration: 2, Cost: 20,
			Text: "2 rounds: no ranged attacks or targeted spells on you"},
		Ability{ID: "jab", Name: "Jab", Element: Physical, Category: PhysicalCat, Delivery: Melee, Power: 30, Fallback: true,
			Text: "A free attack, always ready"},
		Ability{ID: "knife_throw", Name: "Knife Throw", Element: Physical, Category: PhysicalCat, Delivery: Ranged, Power: 45, Cost: 20,
			Text: "A ranged attack"},
		Ability{ID: "rogue_poisoned_blade", Name: "Poisoned Blade", Element: Nature, Category: PhysicalCat, Delivery: Melee, Power: 20, Cost: 30,
			Inflict: []Inflict{{Status: Poison}}, Text: "Poison: 1/16 HP per round"},
		Ability{ID: "wrath_totem", Name: "Magic Wrath Totem", Passive: true, Special: WrathTotem,
			Text: "Passive: puts down a totem at the start of every battle that raises Magic by 1 stage"},
		Ability{ID: "healing_stream_totem", Name: "Healing Stream Totem", Passive: true, Special: HealingTotem,
			Text: "Passive: puts down a totem at the start of every battle that heals 1/32 HP per round"},
		Ability{ID: "improved_sprint", Name: "Improved Sprint", Passive: true,
			Text: "Passive, on the map: sprinting uses up endurance at only 20% of the normal rate"},
		Ability{ID: "poison_resistance", Name: "Poison Resistance", Passive: true, Special: PoisonResist,
			Text: "Passive: poison does 75% less damage to you"},
		Ability{ID: "prey_on_the_weak", Name: "Prey on the Weak", Passive: true, Special: PreyOnWeak,
			Text: "Passive: +10% critical hit chance for each level you are above the enemy"},
		Ability{ID: "cheat_death", Name: "Cheat Death", Passive: true, Special: CheatDeath,
			Text: "Passive: once per battle, an attack that would kill you leaves you at 10% HP"},
		Ability{ID: "cheap_shot", Name: "Cheap Shot", Element: Physical, Category: PhysicalCat, Delivery: Melee, Power: 10, Cost: 25,
			Uses: 1, Inflict: []Inflict{{Status: Stun, Duration: 2}}, Text: "Once per battle: stuns for 2 rounds"},
		Ability{ID: "gut_slash", Name: "Gut Slash", Element: Physical, Category: PhysicalCat, Delivery: Melee, Power: 75, Cost: 30,
			Text: "A vicious, powerful slash"},

		// Hunter.
		Ability{ID: "steady_shot", Name: "Steady Shot", Element: Physical, Category: PhysicalCat, Delivery: Ranged, Power: 50, Cost: 20,
			Text: "A well-aimed arrow"},
		Ability{ID: "venom_arrow", Name: "Venom Arrow", Element: Nature, Category: PhysicalCat, Delivery: Ranged, Power: 25, Cost: 20,
			Inflict: []Inflict{{Status: Poison}}, Text: "Poison: 1/16 HP per round"},
		Ability{ID: "kick", Name: "Kick", Element: Physical, Category: PhysicalCat, Delivery: Melee, Power: 30, Fallback: true,
			Text: "A free attack, always ready"},
		Ability{ID: "mark_prey", Name: "Mark Prey", Category: StatusCat, Delivery: Ranged, Cost: 20,
			Stats: []StatChange{{Stat: Defense, Stages: -1}}, Text: "Target's Defense down"},
		Ability{ID: "concussive_shot", Name: "Concussive Shot", Element: Physical, Category: PhysicalCat, Delivery: Ranged, Power: 35, Cost: 20,
			Stats: []StatChange{{Stat: Speed, Stages: -1}}, Text: "Target's Speed down"},
		Ability{ID: "volley", Name: "Volley", Element: Nature, Category: PhysicalCat, Delivery: AreaSpell, Power: 45, Cost: 30,
			Text: "A rain of arrows; can't be blocked by smoke"},
		Ability{ID: "savage_strike", Name: "Savage Strike", Element: Physical, Category: PhysicalCat, Delivery: Melee, Power: 55, Cost: 20,
			Text: "A melee strike"},
		Ability{ID: "first_aid", Name: "First Aid", Category: StatusCat, Delivery: Self, Cost: 30, Heal: 0.3,
			Text: "Heal 30% of max HP"},

		// Warlock.
		Ability{ID: "drain_life", Name: "Drain Life", Element: Dark, Category: MagicalCat, Delivery: TargetedSpell, Power: 20, Cost: 30,
			Inflict: []Inflict{{Status: Drain}}, Text: "Drain: 1/8 HP per round, healing you (5 rounds)"},
		Ability{ID: "wither", Name: "Wither", Element: Dark, Category: MagicalCat, Delivery: TargetedSpell, Power: 20, Cost: 20,
			Inflict: []Inflict{{Status: Wither}}, Text: "Wither: 1/16 HP per round"},
		Ability{ID: "hex_bolt", Name: "Hex Bolt", Element: Dark, Category: MagicalCat, Delivery: TargetedSpell, Power: 40, Fallback: true,
			Text: "A free attack, always ready"},
		Ability{ID: "curse_of_weakness", Name: "Curse of Weakness", Category: StatusCat, Delivery: TargetedSpell, Cost: 20,
			Stats: []StatChange{{Stat: Attack, Stages: -1}}, Text: "Target's Attack down"},
		Ability{ID: "hellfire_rain", Name: "Hellfire Rain", Element: Fire, Category: MagicalCat, Delivery: AreaSpell, Power: 45, Cost: 30,
			Text: "Area spell: a rain of fire (gets through smoke)"},
		Ability{ID: "demon_armor", Name: "Demon Armor", Passive: true, Special: DemonArmor,
			Text: "Passive: regenerate 1/16 HP every round"},
		Ability{ID: "dread", Name: "Dread", Category: StatusCat, Delivery: TargetedSpell, Cost: 25,
			Inflict: []Inflict{{Status: Dread, Duration: 3}}, Text: "3 rounds: the target may lose its turn (50%)"},

		// Priest.
		Ability{ID: "smite", Name: "Smite", Element: Holy, Category: MagicalCat, Delivery: TargetedSpell, Power: 50, Cost: 20,
			Text: "A bolt of holy light"},
		Ability{ID: "mend", Name: "Mend", Category: StatusCat, Delivery: Self, Cost: 25, Heal: 0.25, Text: "Heal 25% of max HP"},
		Ability{ID: "light_spark", Name: "Light Spark", Element: Holy, Category: MagicalCat, Delivery: TargetedSpell, Power: 40, Fallback: true,
			Text: "A free attack, always ready"},
		Ability{ID: "holy_burst", Name: "Holy Burst", Element: Holy, Category: MagicalCat, Delivery: AreaSpell, Power: 45, Cost: 30,
			Text: "Area spell: a burst of light (gets through smoke)"},
		Ability{ID: "ward_of_light", Name: "Ward of Light", Category: StatusCat, Delivery: Self, Cost: 25, Special: Shield,
			Text: "A shield absorbs damage up to 25% of max HP"},
		Ability{ID: "renewal", Name: "Renewal", Category: StatusCat, Delivery: Self, Cost: 25, Regen: 8, Text: "Regen: 1/16 HP per round, 8 rounds"},
		Ability{ID: "mind_pain", Name: "Mind Pain", Element: Dark, Category: MagicalCat, Delivery: TargetedSpell, Power: 20, Cost: 20,
			Inflict: []Inflict{{Status: Pain}}, Text: "Pain: 1/16 HP per round"},

		// Paladin.
		Ability{ID: "righteous_strike", Name: "Righteous Strike", Element: Holy, Category: PhysicalCat, Delivery: Melee, Power: 50, Cost: 20,
			Text: "A holy melee strike"},
		Ability{ID: "holy_seal", Name: "Holy Seal", Category: StatusCat, Delivery: Self, Cost: 20, Special: HolySeal, Duration: 5,
			Text: "5 rounds: physical hits deal extra Holy damage"},
		Ability{ID: "hammer_blow", Name: "Hammer Blow", Element: Physical, Category: PhysicalCat, Delivery: Melee, Power: 30, Fallback: true,
			Text: "A free attack, always ready"},
		Ability{ID: "judgment_hammer", Name: "Judgment Hammer", Element: Holy, Category: PhysicalCat, Delivery: Melee, Power: 30, Cost: 25,
			Uses: 1, Priority: 1, Inflict: []Inflict{{Status: Stun, Duration: 1, MaxDuration: 3}}, Text: "Once per battle: stuns for 1-3 rounds"},
		Ability{ID: "devotion", Name: "Devotion", Category: StatusCat, Delivery: Self, Cost: 20,
			Stats: []StatChange{{Stat: Defense, Stages: 1, OnSelf: true}}, Text: "Defense up"},
		Ability{ID: "healing_light", Name: "Healing Light", Category: StatusCat, Delivery: Self, Cost: 30, Heal: 0.3, Text: "Heal 30% of max HP"},

		// Shaman.
		Ability{ID: "storm_bolt", Name: "Storm Bolt", Element: Air, Category: MagicalCat, Delivery: TargetedSpell, Power: 50, Cost: 20,
			Text: "A bolt of lightning"},
		Ability{ID: "flame_shock", Name: "Flame Shock", Element: Fire, Category: MagicalCat, Delivery: TargetedSpell, Power: 25, Cost: 20,
			Inflict: []Inflict{{Status: Burn}}, Text: "Burn: 1/16 HP per round"},
		Ability{ID: "zap", Name: "Zap", Element: Air, Category: MagicalCat, Delivery: TargetedSpell, Power: 40, Fallback: true,
			Text: "A free attack, always ready"},
		Ability{ID: "chain_lightning", Name: "Chain Lightning", Element: Air, Category: MagicalCat, Delivery: AreaSpell, Power: 45, Cost: 30,
			Text: "Area spell: forking lightning (gets through smoke)"},
		Ability{ID: "healing_wave", Name: "Healing Wave", Element: Water, Category: StatusCat, Delivery: Self, Cost: 30, Heal: 0.2,
			Text: "Heal 20% of max HP"},
		Ability{ID: "stone_shock", Name: "Stone Shock", Element: Nature, Category: MagicalCat, Delivery: TargetedSpell, Power: 40, Cost: 25,
			Inflict: []Inflict{{Status: Silence, Chance: 0.3, Duration: 2}}, Text: "30%: Silence for 2 rounds (no spells)"},
		Ability{ID: "frost_shock", Name: "Frost Shock", Element: Ice, Category: MagicalCat, Delivery: TargetedSpell, Power: 45, Cost: 25,
			Stats: []StatChange{{Stat: Speed, Stages: -1}}, Text: "Target's Speed down"},

		// Druid.
		Ability{ID: "thorn_lash", Name: "Thorn Lash", Element: Nature, Category: MagicalCat, Delivery: TargetedSpell, Power: 50, Cost: 20,
			Text: "A whip of thorny vines"},
		Ability{ID: "rejuvenate", Name: "Rejuvenate", Category: StatusCat, Delivery: Self, Cost: 25, Regen: 8,
			Text: "Regen: 1/16 HP per round, 8 rounds"},
		Ability{ID: "bramble", Name: "Bramble", Element: Nature, Category: MagicalCat, Delivery: TargetedSpell, Power: 40, Fallback: true,
			Text: "A free attack, always ready"},
		Ability{ID: "insect_swarm", Name: "Insect Swarm", Element: Nature, Category: MagicalCat, Delivery: AreaSpell, Power: 30, Cost: 30,
			Inflict: []Inflict{{Status: Swarm}}, Stats: []StatChange{{Stat: Attack, Stages: -1, Chance: 0.3}},
			Text: "Area spell: Swarm 1/16 HP per round; 30%: Attack down"},
		Ability{ID: "grasping_roots", Name: "Grasping Roots", Element: Nature, Category: MagicalCat, Delivery: TargetedSpell, Power: 30, Cost: 20,
			Stats: []StatChange{{Stat: Speed, Stages: -1}}, Text: "Target's Speed down"},
		Ability{ID: "moonbeam", Name: "Moonbeam", Element: Holy, Category: MagicalCat, Delivery: TargetedSpell, Power: 50, Cost: 25,
			Text: "A beam of moonlight"},
		Ability{ID: "thorns", Name: "Thorns", Passive: true, Special: Thorns,
			Text: "Passive: melee attackers that hit you take damage, 1/16 of your max HP"},
		Ability{ID: "bear_form", Name: "Bear Form", Category: StatusCat, Delivery: Self, Cost: 30, Uses: 1, Special: BearForm,
			Text: "Once: Defense up, heal 25%; attacks become melee and hit Defense"},

		// Enemies (design/world.md, Enemies).
		Ability{ID: "bite", Name: "Bite", Element: Physical, Category: PhysicalCat, Delivery: Melee, Power: 40},
		Ability{ID: "howl", Name: "Howl", Category: StatusCat, Delivery: Self, Stats: []StatChange{{Stat: Attack, Stages: 1, OnSelf: true}}},
		Ability{ID: "gore", Name: "Gore", Element: Physical, Category: PhysicalCat, Delivery: Melee, Power: 45},
		Ability{ID: "tusk_charge", Name: "Tusk Charge", Element: Nature, Category: PhysicalCat, Delivery: Melee, Power: 55, Priority: 1},
		Ability{ID: "club", Name: "Club", Element: Physical, Category: PhysicalCat, Delivery: Melee, Power: 40},
		Ability{ID: "sling_stone", Name: "Sling Stone", Element: Physical, Category: PhysicalCat, Delivery: Ranged, Power: 30,
			Stats: []StatChange{{Stat: Speed, Stages: -1, Chance: 0.3}}},
		Ability{ID: "quick_stab", Name: "Quick Stab", Element: Physical, Category: PhysicalCat, Delivery: Melee, Power: 45, CritRate: 0.25, Priority: 1},
		Ability{ID: "poisoned_blade", Name: "Poisoned Blade", Element: Nature, Category: PhysicalCat, Delivery: Melee, Power: 20,
			Inflict: []Inflict{{Status: Poison}}},
		Ability{ID: "smoke_screen", Name: "Smoke Screen", Category: StatusCat, Delivery: Self, Special: Smoke, Duration: 3},

		// Desert enemies.
		Ability{ID: "venom_bite", Name: "Venom Bite", Element: Nature, Category: PhysicalCat, Delivery: Melee, Power: 35,
			Inflict: []Inflict{{Status: Poison, Chance: 0.3}}},
		Ability{ID: "coil", Name: "Coil", Category: StatusCat, Delivery: Self, Stats: []StatChange{{Stat: Defense, Stages: 1, OnSelf: true}}},
		Ability{ID: "pincer", Name: "Pincer", Element: Physical, Category: PhysicalCat, Delivery: Melee, Power: 40},
		Ability{ID: "sting", Name: "Sting", Element: Nature, Category: PhysicalCat, Delivery: Melee, Power: 30,
			Inflict: []Inflict{{Status: Poison, Chance: 0.5}}},
		Ability{ID: "ember", Name: "Ember", Element: Fire, Category: MagicalCat, Delivery: TargetedSpell, Power: 40,
			Inflict: []Inflict{{Status: Burn, Chance: 0.2}}},
		Ability{ID: "flare", Name: "Flare", Element: Fire, Category: MagicalCat, Delivery: AreaSpell, Power: 35},
		Ability{ID: "scimitar_slash", Name: "Scimitar Slash", Element: Physical, Category: PhysicalCat, Delivery: Melee, Power: 50},
		Ability{ID: "fire_brand", Name: "Fire Brand", Element: Fire, Category: PhysicalCat, Delivery: Melee, Power: 35,
			Inflict: []Inflict{{Status: Burn, Chance: 0.3}}},
		Ability{ID: "sand_veil", Name: "Sand Veil", Category: StatusCat, Delivery: Self, Special: Smoke, Duration: 2},

		// Ancient desert enemies.
		Ability{ID: "wrap", Name: "Wrap", Element: Physical, Category: PhysicalCat, Delivery: Melee, Power: 40,
			Stats: []StatChange{{Stat: Speed, Stages: -1, Chance: 0.3}}},
		Ability{ID: "mummy_curse", Name: "Ancient Curse", Element: Dark, Category: MagicalCat, Delivery: TargetedSpell, Power: 35,
			Stats: []StatChange{{Stat: Attack, Stages: -1, Chance: 0.5}}},
		Ability{ID: "mandible", Name: "Mandible", Element: Physical, Category: PhysicalCat, Delivery: Melee, Power: 35},
		Ability{ID: "carapace", Name: "Carapace", Category: StatusCat, Delivery: Self, Stats: []StatChange{{Stat: Defense, Stages: 1, OnSelf: true}}},
		Ability{ID: "venom_spit", Name: "Venom Spit", Element: Nature, Category: PhysicalCat, Delivery: Ranged, Power: 40,
			Inflict: []Inflict{{Status: Poison, Chance: 0.3}}},
		Ability{ID: "hood_flare", Name: "Hood Flare", Category: StatusCat, Delivery: AreaSpell, Stats: []StatChange{{Stat: Attack, Stages: -1}}},
		Ability{ID: "sun_strike", Name: "Sun Strike", Element: Holy, Category: PhysicalCat, Delivery: Melee, Power: 50},
		Ability{ID: "judgment_ray", Name: "Judgment Ray", Element: Holy, Category: MagicalCat, Delivery: TargetedSpell, Power: 45},
		Ability{ID: "sacred_ward", Name: "Sacred Ward", Category: StatusCat, Delivery: Self,
			Stats: []StatChange{{Stat: Defense, Stages: 1, OnSelf: true}, {Stat: Resistance, Stages: 1, OnSelf: true}}},

		// North enemies.
		Ability{ID: "frost_bite", Name: "Frost Bite", Element: Ice, Category: PhysicalCat, Delivery: Melee, Power: 40,
			Stats: []StatChange{{Stat: Speed, Stages: -1, Chance: 0.2}}},
		Ability{ID: "ice_shard", Name: "Ice Shard", Element: Ice, Category: MagicalCat, Delivery: TargetedSpell, Power: 40,
			Inflict: []Inflict{{Status: Freeze, Chance: 0.15, Duration: 1}}},
		Ability{ID: "chill_wind", Name: "Chill Wind", Element: Ice, Category: MagicalCat, Delivery: AreaSpell, Power: 35},
		Ability{ID: "troll_smash", Name: "Troll Smash", Element: Physical, Category: PhysicalCat, Delivery: Melee, Power: 60},
		Ability{ID: "hurl_ice", Name: "Hurl Ice", Element: Ice, Category: PhysicalCat, Delivery: Ranged, Power: 40},
		Ability{ID: "frozen_hide", Name: "Frozen Hide", Category: StatusCat, Delivery: Self, Stats: []StatChange{{Stat: Defense, Stages: 1, OnSelf: true}}},
		Ability{ID: "cleave", Name: "Cleave", Element: Physical, Category: PhysicalCat, Delivery: Melee, Power: 55},
		Ability{ID: "berserk", Name: "Berserk", Category: StatusCat, Delivery: Self,
			Stats: []StatChange{{Stat: Attack, Stages: 2, OnSelf: true}, {Stat: Defense, Stages: -1, OnSelf: true}}},
		Ability{ID: "frost_axe", Name: "Frost Axe", Element: Ice, Category: PhysicalCat, Delivery: Melee, Power: 40,
			Inflict: []Inflict{{Status: Freeze, Chance: 0.15, Duration: 1}}},

		// Mediterranean enemies.
		Ability{ID: "pinch", Name: "Pinch", Element: Physical, Category: PhysicalCat, Delivery: Melee, Power: 45},
		Ability{ID: "bubble_spray", Name: "Bubble Spray", Element: Water, Category: MagicalCat, Delivery: TargetedSpell, Power: 35,
			Stats: []StatChange{{Stat: Speed, Stages: -1, Chance: 0.3}}},
		Ability{ID: "hard_shell", Name: "Hard Shell", Category: StatusCat, Delivery: Self, Stats: []StatChange{{Stat: Defense, Stages: 2, OnSelf: true}}},
		Ability{ID: "gust", Name: "Gust", Element: Air, Category: MagicalCat, Delivery: TargetedSpell, Power: 40},
		Ability{ID: "whirlwind", Name: "Whirlwind", Element: Air, Category: MagicalCat, Delivery: AreaSpell, Power: 35,
			Stats: []StatChange{{Stat: Speed, Stages: -1, Chance: 0.2}}},
		Ability{ID: "cutlass_slash", Name: "Cutlass Slash", Element: Physical, Category: PhysicalCat, Delivery: Melee, Power: 50},
		Ability{ID: "pistol_shot", Name: "Pistol Shot", Element: Physical, Category: PhysicalCat, Delivery: Ranged, Power: 60, Uses: 1,
			Priority: 1},
		Ability{ID: "smoke_flare", Name: "Smoke Flare", Category: StatusCat, Delivery: Self, Special: Smoke, Duration: 2},
		Ability{ID: "spear_thrust", Name: "Spear Thrust", Element: Physical, Category: PhysicalCat, Delivery: Melee, Power: 55},
		Ability{ID: "shield_wall", Name: "Shield Wall", Category: StatusCat, Delivery: Self,
			Stats: []StatChange{{Stat: Defense, Stages: 2, OnSelf: true}}},
		Ability{ID: "sun_spear", Name: "Sun Spear", Element: Holy, Category: PhysicalCat, Delivery: Melee, Power: 45},

		// Unholy enemies.
		Ability{ID: "bat_bites", Name: "Bat Bites", Element: Physical, Category: PhysicalCat, Delivery: Melee, Power: 35, CritRate: 0.2},
		Ability{ID: "screech", Name: "Screech", Category: StatusCat, Delivery: AreaSpell, Stats: []StatChange{{Stat: Attack, Stages: -1}}},
		Ability{ID: "savage_bite", Name: "Savage Bite", Element: Physical, Category: PhysicalCat, Delivery: Melee, Power: 45,
			Inflict: []Inflict{{Status: Bleed, Chance: 0.25}}},
		Ability{ID: "dark_howl", Name: "Dark Howl", Category: StatusCat, Delivery: Self, Stats: []StatChange{{Stat: Attack, Stages: 1, OnSelf: true}}},
		Ability{ID: "shadow_bolt", Name: "Shadow Bolt", Element: Dark, Category: MagicalCat, Delivery: TargetedSpell, Power: 50},
		Ability{ID: "dark_pact", Name: "Dark Pact", Category: StatusCat, Delivery: Self,
			Stats: []StatChange{{Stat: Attack, Stages: 1, OnSelf: true}, {Stat: Magic, Stages: 1, OnSelf: true}}},
		Ability{ID: "life_drain", Name: "Life Drain", Element: Dark, Category: MagicalCat, Delivery: TargetedSpell, Power: 30,
			Inflict: []Inflict{{Status: Leech}}}, // half the warlock's Drain
		Ability{ID: "vampire_claw", Name: "Vampire Claw", Element: Physical, Category: PhysicalCat, Delivery: Melee, Power: 55},
		Ability{ID: "mesmerize", Name: "Mesmerize", Category: StatusCat, Delivery: TargetedSpell, Uses: 1,
			Inflict: []Inflict{{Status: Dread, Duration: 2}}},
	)
}
