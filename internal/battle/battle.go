package battle

import (
	"fmt"
	"math"
	"math/rand/v2"
	"sort"
	"strings"
)

// Tuning (design/mechanics.md, Damage Formula and Status Effects).
const (
	damageScale     = 3.0
	baseCrit        = 1.0 / 16
	critMultiplier  = 1.5
	affinityBonus   = 1.5
	enrageChance    = 0.5
	secondWindOdds  = 0.3
	enrageBonus     = 1.4
	enrageRounds    = 4
	secondWindRegen = 3 // rounds of Regen (1/16 max HP each) that Second Wind starts
	flyingDodge     = 0.5
	dotRounds       = 10 // 1/16 effects last about the rest of the battle
	drainRounds     = 5  // Drain is twice as strong and heals, so it is shorter
	manaShieldShare = 0.5
	shieldShare     = 0.25 // a Shield absorbs this share of max HP
	dreadOdds       = 0.5  // chance that Dread makes the target lose its turn
	sealPower       = 20   // Holy Seal: power of the extra Holy damage on physical hits
	bearDefense     = 2    // Bear Form: Defense stages gained
	bearHeal        = 0.25 // and the share of max HP restored
)

// StatusKind is a status effect.
type StatusKind int

const (
	Burn StatusKind = iota
	Poison
	Bleed
	Freeze
	Stun
	Drain   // 1/8 max HP per round, healing the other side (Drain Life)
	Wither  // Dark damage over time, 1/16
	Pain    // Mind Pain: like Wither
	Dread   // may lose the turn
	Silence // can't use spells
	Swarm   // Insect Swarm: 1/16
	numStatus
)

var statusNames = [numStatus]string{"Burn", "Poison", "Bleed", "Freeze", "Stun", "Drain", "Wither", "Pain", "Dread", "Silence", "Swarm"}

func (s StatusKind) String() string { return statusNames[s] }

func (s StatusKind) dot() bool {
	return s == Burn || s == Poison || s == Bleed || s == Drain || s == Wither || s == Pain || s == Swarm
}
func (s StatusKind) disables() bool    { return s == Freeze || s == Stun }
func (s StatusKind) breaksOnHit() bool { return s == Freeze }

// Stats are the combat stats of a combatant.
type Stats struct {
	MaxHP, MaxMP               int
	Attack, Magic              int
	Defense, Resistance, Speed int
}

// Combatant is one side of a battle. HP and MP are the live values; they are written back
// to the character after the battle (HP and MP are global, see design/mechanics.md).
type Combatant struct {
	Name      string
	Level     int
	Types     []Element
	Affinity  []Element // elements with the Affinity bonus; nil means Types
	Base      Stats
	HP, MP    int
	Abilities []string // active abilities, including passives
	Flying    bool
	You       bool // the player: messages say "You use" instead of "Wolf uses"

	stages      [numStats]int
	status      map[StatusKind]int // rounds left
	enraged     int
	regen       int // rounds of Regen left
	shield      int // damage a Shield still absorbs
	sealed      int // rounds of Holy Seal left
	bear        bool
	manaShield  int
	smoke       int
	evadeNext   bool
	overwhelmOK bool           // the user missed or was dodged last turn
	used        map[string]int // uses of limited abilities this battle
}

func (c *Combatant) init() {
	c.status = map[StatusKind]int{}
	c.stages = [numStats]int{}
	c.enraged, c.regen, c.shield, c.sealed, c.manaShield, c.smoke = 0, 0, 0, 0, 0, 0
	c.bear = false
	c.evadeNext, c.overwhelmOK = false, false
	c.used = map[string]int{}
}

// Subject, Object, Possessive, Verb and Be phrase battle messages for the player ("You
// use", "your") and for enemies ("Wolf uses", "Wolf's").
func (c *Combatant) Subject() string {
	if c.You {
		return "You"
	}
	return c.Name
}

func (c *Combatant) Object() string {
	if c.You {
		return "you"
	}
	return c.Name
}

func (c *Combatant) Possessive() string {
	if c.You {
		return "Your"
	}
	return c.Name + "'s"
}

func (c *Combatant) Verb(v string) string {
	switch {
	case c.You:
		return c.Subject() + " " + v
	case strings.HasSuffix(v, "s"), strings.HasSuffix(v, "sh"), strings.HasSuffix(v, "ch"), strings.HasSuffix(v, "x"):
		return c.Name + " " + v + "es"
	default:
		return c.Name + " " + v + "s"
	}
}

func (c *Combatant) Be() string {
	if c.You {
		return "You are"
	}
	return c.Name + " is"
}

// Alive reports whether the combatant can still fight.
func (c *Combatant) Alive() bool { return c.HP > 0 }

func (c *Combatant) has(special Special) bool {
	for _, id := range c.Abilities {
		if a := Get(id); a != nil && a.Special == special {
			return true
		}
	}
	return false
}

// stat returns a stat with its stage multiplier applied.
func (c *Combatant) stat(s Stat) float64 {
	var v int
	switch s {
	case Attack:
		v = c.Base.Attack
	case Magic:
		v = c.Base.Magic
	case Defense:
		v = c.Base.Defense
	case Resistance:
		v = c.Base.Resistance
	case Speed:
		v = c.Base.Speed
	}
	return float64(v) * stageMultiplier(c.stages[s])
}

func stageMultiplier(n int) float64 {
	if n >= 0 {
		return float64(2+n) / 2
	}
	return 2 / float64(2-n)
}

// Effect is an active buff, debuff or status, for display.
type Effect struct {
	Label string
	Good  bool // a buff (true) or a debuff or harmful status (false)
}

var statShort = [numStats]string{"Atk", "Mag", "Def", "Res", "Spd"}

// Effects lists everything currently affecting the combatant: stat stages, Enrage, Mana
// Shield, Smoke, Blink and status effects.
func (c *Combatant) Effects() []Effect {
	var out []Effect
	for s := Stat(0); s < numStats; s++ {
		if n := c.stages[s]; n != 0 {
			out = append(out, Effect{fmt.Sprintf("%s%+d", statShort[s], n), n > 0})
		}
	}
	if c.enraged > 0 {
		out = append(out, Effect{"Enraged", true})
	}
	if c.regen > 0 {
		out = append(out, Effect{"Regen", true})
	}
	if c.shield > 0 {
		out = append(out, Effect{"Shield", true})
	}
	if c.sealed > 0 {
		out = append(out, Effect{"Holy Seal", true})
	}
	if c.bear {
		out = append(out, Effect{"Bear Form", true})
	}
	if c.has(DemonArmor) {
		out = append(out, Effect{"Demon Armor", true})
	}
	if c.manaShield > 0 {
		out = append(out, Effect{"Mana Shield", true})
	}
	if c.smoke > 0 {
		out = append(out, Effect{"Smoke", true})
	}
	if c.evadeNext {
		out = append(out, Effect{"Blink", true})
	}
	for k := StatusKind(0); k < numStatus; k++ {
		if c.status[k] > 0 {
			out = append(out, Effect{k.String(), false})
		}
	}
	return out
}

// Event is one step of a turn for the battle screen to show. HP/MP values are the state
// after the event, so bars can animate to them.
type Event struct {
	Text     string
	PlayerHP int
	PlayerMP int
	EnemyHP  int
	EnemyMP  int
	Hit      *Combatant // flashes when it takes damage
	Cue      Cue        // what happened, for sound
	Ability  *Ability   // the ability used, for CueUse
	User     *Combatant // who used it, for CueUse
}

// Cue says what an event shows, so the battle screen can play a matching sound.
type Cue int

const (
	CueNone Cue = iota
	CueUse      // an ability is used (Event.Ability)
	CueHit
	CueCrit
	CueMiss
	CueNoEffect
	CueDefeat
	CueStatUp
	CueStatDown
	CueStatus // a status effect is inflicted
	CueHeal
)

// Battle is a 1v1 fight.
type Battle struct {
	Player, Enemy *Combatant
	Round         int
	started       bool // the first round is over
	rng           *rand.Rand
	events        []Event
}

// New starts a battle. The seed only affects random rolls (crits, damage spread, chances).
func New(player, enemy *Combatant, seed uint64) *Battle {
	player.init()
	enemy.init()
	return &Battle{Player: player, Enemy: enemy, rng: rand.New(rand.NewPCG(seed, seed^0x9E3779B97F4A7C15))}
}

// Over reports whether one side is defeated.
func (b *Battle) Over() bool { return !b.Player.Alive() || !b.Enemy.Alive() }

// Won reports whether the player won.
func (b *Battle) Won() bool { return b.Player.Alive() && !b.Enemy.Alive() }

// Usable reports whether a combatant can choose an ability right now, and why not.
func (b *Battle) Usable(user *Combatant, id string) (bool, string) {
	a := Get(id)
	if a == nil || a.Passive {
		return false, ""
	}
	target := b.foe(user)
	switch {
	case user.MP < a.Cost:
		return false, "Not enough MP"
	case a.Special == Overwhelm && !user.overwhelmOK:
		return false, "Only after a miss or dodge"
	case a.Opener && b.started:
		return false, "Only in the first round"
	case a.Uses > 0 && user.used[a.ID] >= a.Uses:
		return false, "Already used this battle"
	case user.status[Silence] > 0 && isSpell(user, a):
		return false, "Silenced"
	case target.smoke > 0 && (delivery(user, a) == Ranged || delivery(user, a) == TargetedSpell):
		return false, "Can't target through smoke"
	}
	return true, ""
}

// delivery is how an ability reaches its target: in Bear Form, every attack is melee.
func delivery(user *Combatant, a *Ability) Delivery {
	if user.bear && a.Category != StatusCat && a.Delivery != Self {
		return Melee
	}
	return a.Delivery
}

// isSpell reports whether Silence stops an ability: targeted and area spells that aren't
// physical (Volley is arrows), and nothing in Bear Form, where attacks are claws.
func isSpell(user *Combatant, a *Ability) bool {
	d := delivery(user, a)
	return (d == TargetedSpell || d == AreaSpell) && a.Category != PhysicalCat
}

// MenuOptions lists the moves to offer: the regular abilities, then the free basic moves
// (fallback attacks, always available) and Wait, which everyone always has.
func (b *Battle) MenuOptions(user *Combatant) []string {
	var regular, basic []string
	for _, id := range user.Abilities {
		a := Get(id)
		switch {
		case a == nil || a.Passive:
		case a.Fallback:
			basic = append(basic, id)
		default:
			regular = append(regular, id)
		}
	}
	return append(append(regular, basic...), WaitID)
}

func (b *Battle) foe(c *Combatant) *Combatant {
	if c == b.Player {
		return b.Enemy
	}
	return b.Player
}

// Turn resolves one round: both sides act in Speed order (priority first), then status
// effects tick. It returns the events of the round.
func (b *Battle) Turn(playerAbility, enemyAbility string) []Event {
	b.events = nil
	b.Round++
	type action struct {
		user *Combatant
		a    *Ability
	}
	acts := []action{{b.Player, Get(playerAbility)}, {b.Enemy, Get(enemyAbility)}}
	b.rng.Shuffle(len(acts), func(i, j int) { acts[i], acts[j] = acts[j], acts[i] }) // random speed ties
	sort.SliceStable(acts, func(i, j int) bool {
		if acts[i].a.Priority != acts[j].a.Priority {
			return acts[i].a.Priority > acts[j].a.Priority
		}
		return acts[i].user.stat(Speed) > acts[j].user.stat(Speed)
	})

	for _, act := range acts {
		if b.Over() {
			break
		}
		b.act(act.user, act.a)
	}
	if !b.Over() {
		b.endOfRound()
	}
	b.started = true
	return b.events
}

func (b *Battle) emit(format string, args ...any) {
	b.emitEvent(Event{}, format, args...)
}

func (b *Battle) emitCue(cue Cue, format string, args ...any) {
	b.emitEvent(Event{Cue: cue}, format, args...)
}

func (b *Battle) emitHit(hit *Combatant, cue Cue, format string, args ...any) {
	b.emitEvent(Event{Hit: hit, Cue: cue}, format, args...)
}

// emitEvent adds an event with the text and the current HP and MP filled in.
func (b *Battle) emitEvent(ev Event, format string, args ...any) {
	ev.Text = fmt.Sprintf(format, args...)
	ev.PlayerHP, ev.PlayerMP = b.Player.HP, b.Player.MP
	ev.EnemyHP, ev.EnemyMP = b.Enemy.HP, b.Enemy.MP
	b.events = append(b.events, ev)
}

func (b *Battle) act(user *Combatant, a *Ability) {
	target := b.foe(user)

	for k, left := range user.status {
		if left > 0 && k.disables() {
			b.emit("%s %s and can't move!", user.Be(), statusVerb(k))
			return
		}
	}
	if user.status[Dread] > 0 && a.ID != WaitID && b.rng.Float64() < dreadOdds {
		b.emit("%s frozen with dread!", user.Be())
		return
	}
	if a.ID == WaitID {
		b.emit("%s.", user.Verb("wait"))
		return
	}
	if ok, _ := b.Usable(user, a.ID); !ok {
		b.emit("%s can't use %s!", user.Subject(), a.Name)
		return
	}
	user.MP -= a.Cost
	user.used[a.ID]++
	b.emitEvent(Event{Cue: CueUse, Ability: a, User: user}, "%s %s!", user.Verb("use"), a.Name)
	user.overwhelmOK = false

	if a.Delivery == Self {
		b.applySelf(user, a)
		return
	}

	// Misses only come from visible mechanics (design/mechanics.md, No random misses).
	if a.Special != Overwhelm {
		if target.evadeNext {
			target.evadeNext = false
			b.missed(user, "%s the attack!", target.Verb("evade"))
			return
		}
		if target.Flying && delivery(user, a) == Melee && b.rng.Float64() < flyingDodge {
			b.missed(user, "%s in the air!", target.Verb("dodge"))
			return
		}
	}
	if a.Category != StatusCat {
		b.hit(user, target, a)
		if !target.Alive() {
			return
		}
		if user.sealed > 0 && (a.Category == PhysicalCat || user.bear) {
			b.sealBurn(user, target)
			if !target.Alive() {
				return
			}
		}
	}
	for _, sc := range a.Stats {
		if sc.Chance > 0 && b.rng.Float64() >= sc.Chance {
			continue
		}
		who := target
		if sc.OnSelf {
			who = user
		}
		b.changeStat(who, sc.Stat, sc.Stages)
	}
	for _, in := range a.Inflict {
		if in.Chance > 0 && b.rng.Float64() >= in.Chance {
			continue
		}
		b.inflict(target, in)
	}
}

func (b *Battle) missed(user *Combatant, format string, args ...any) {
	b.emitCue(CueMiss, format, args...)
	if user.has(Overwhelm) {
		user.overwhelmOK = true
	}
}

func (b *Battle) applySelf(user *Combatant, a *Ability) {
	switch a.Special {
	case ManaShield:
		user.manaShield = a.Duration
		b.emit("A shield of mana surrounds %s.", user.Object())
	case Blink:
		user.evadeNext = true
		b.emit("%s out of reach.", user.Verb("blink"))
	case Smoke:
		user.smoke = a.Duration
		b.emit("%s in smoke!", user.Verb("vanish"))
	case Shield:
		user.shield = max(1, int(float64(user.Base.MaxHP)*shieldShare))
		b.emitCue(CueStatUp, "A ward of light surrounds %s.", user.Object())
	case HolySeal:
		user.sealed = a.Duration
		b.emitCue(CueStatUp, "%s attacks are blessed with holy light.", user.Possessive())
	case BearForm:
		user.bear = true
		b.emitCue(CueStatUp, "%s into a bear!", user.Verb("turn"))
		b.heal(user, bearHeal)
		b.changeStat(user, Defense, bearDefense)
	}
	if a.Heal > 0 {
		b.heal(user, a.Heal)
	}
	if a.Regen > 0 {
		user.regen = a.Regen
		b.emitCue(CueStatUp, "%s to regenerate.", user.Verb("start"))
	}
	for _, sc := range a.Stats {
		b.changeStat(user, sc.Stat, sc.Stages)
	}
}

// heal restores a share of max HP.
func (b *Battle) heal(c *Combatant, share float64) {
	n := min(max(1, int(float64(c.Base.MaxHP)*share)), c.Base.MaxHP-c.HP)
	c.HP += n
	b.emitCue(CueHeal, "%s %d HP.", c.Verb("recover"), n)
}

// sealBurn is Holy Seal's extra Holy damage after a physical hit.
func (b *Battle) sealBurn(user, target *Combatant) {
	seal := &Ability{Name: "Holy Seal", Element: Holy, Category: PhysicalCat, Power: sealPower}
	if Effectiveness(Holy, target.Types) == 0 {
		return
	}
	dmg := b.absorb(target, b.Damage(user, target, seal, false, 0.85+0.15*b.rng.Float64()))
	target.HP = max(0, target.HP-dmg)
	b.emitHit(target, CueHit, "The seal burns %s for %d.", target.Object(), dmg)
	if !target.Alive() {
		b.emitCue(CueDefeat, "%s defeated!", target.Be())
	}
}

// absorb takes damage out of a Shield and Mana Shield first; it returns what reaches HP.
func (b *Battle) absorb(target *Combatant, dmg int) int {
	if target.shield > 0 {
		n := min(dmg, target.shield)
		target.shield -= n
		dmg -= n
		if target.shield == 0 {
			b.emit("%s ward breaks.", target.Possessive())
		}
	}
	if target.manaShield > 0 {
		toMP := min(int(float64(dmg)*manaShieldShare), target.MP)
		target.MP -= toMP
		dmg -= toMP
	}
	return dmg
}

// Damage computes the damage of an ability before Mana Shield; exported for tests.
func (b *Battle) Damage(user, target *Combatant, a *Ability, crit bool, roll float64) int {
	atk, def := user.stat(Attack), target.stat(Defense)
	switch {
	case user.bear:
		atk = user.stat(Magic) // a bear's spells are claws: the Druid's power against Defense
	case a.Category == MagicalCat:
		atk, def = user.stat(Magic), target.stat(Resistance)
	}
	base := ((2*float64(user.Level)/5+2)*float64(a.Power)*atk/math.Max(def, 1))/50 + 2
	m := damageScale * Effectiveness(a.Element, target.Types)
	if m == 0 {
		return 0
	}
	affinity := user.Affinity
	if affinity == nil {
		affinity = user.Types
	}
	for _, t := range affinity {
		if t == a.Element {
			m *= affinityBonus
			break
		}
	}
	if crit {
		m *= critMultiplier
	}
	if user.enraged > 0 {
		m *= enrageBonus
	}
	if user.status[Burn] > 0 && (a.Category == PhysicalCat || user.bear) {
		m *= 0.5
	}
	return max(1, int(base*m*roll))
}

func (b *Battle) hit(user, target *Combatant, a *Ability) {
	eff := Effectiveness(a.Element, target.Types)
	if eff == 0 {
		b.emitCue(CueNoEffect, "It had no effect on %s.", target.Object())
		return
	}
	critRate := baseCrit
	if a.CritRate > 0 {
		critRate = a.CritRate
	}
	crit := b.rng.Float64() < critRate
	dmg := b.absorb(target, b.Damage(user, target, a, crit, 0.85+0.15*b.rng.Float64()))
	target.HP = max(0, target.HP-dmg)
	cue := CueHit
	if crit {
		cue = CueCrit
	}
	switch {
	case crit && eff > 1:
		b.emitHit(target, cue, "A critical hit! It's super effective!")
	case crit:
		b.emitHit(target, cue, "A critical hit!")
	case eff > 1:
		b.emitHit(target, cue, "It's super effective!")
	case eff < 1:
		b.emitHit(target, cue, "It's not very effective...")
	default:
		b.emitHit(target, cue, "%s %d damage.", target.Verb("take"), dmg)
	}
	if !target.Alive() {
		b.emitCue(CueDefeat, "%s defeated!", target.Be())
		return
	}
	for k := range target.status {
		if k.breaksOnHit() && target.status[k] > 0 {
			target.status[k] = 0
			b.emit("%s out!", target.Verb("thaw"))
		}
	}
	if target.has(Enrage) && target.enraged == 0 && (crit || b.rng.Float64() < enrageChance) {
		target.enraged = enrageRounds
		b.emitCue(CueStatUp, "%s enraged!", target.Verb("become"))
	}
	if target.has(SecondWind) && (crit || b.rng.Float64() < secondWindOdds) {
		target.regen = secondWindRegen // a new trigger starts over, it doesn't stack
		b.emitCue(CueStatUp, "Second Wind! %s to regenerate.", target.Verb("start"))
	}
}

func (b *Battle) changeStat(c *Combatant, s Stat, stages int) {
	old := c.stages[s]
	c.stages[s] = min(6, max(-6, old+stages))
	switch {
	case c.stages[s] == old && stages > 0:
		b.emit("%s %s won't go higher!", c.Possessive(), s)
	case c.stages[s] == old:
		b.emit("%s %s won't go lower!", c.Possessive(), s)
	case stages > 0:
		b.emitCue(CueStatUp, "%s %s rises!", c.Possessive(), s)
	default:
		b.emitCue(CueStatDown, "%s %s falls!", c.Possessive(), s)
	}
}

func (b *Battle) inflict(c *Combatant, in Inflict) {
	if c.status[in.Status] > 0 {
		c.status[in.Status] = b.durationFor(in) // reapplying resets the duration
		return
	}
	c.status[in.Status] = b.durationFor(in)
	cue := CueStatus
	if in.Status == Stun {
		cue = CueNone // Charge already makes its sound; a second one right after is too much
	}
	b.emitCue(cue, "%s %s!", c.Be(), statusVerb(in.Status))
}

func (b *Battle) durationFor(in Inflict) int {
	switch {
	case in.MaxDuration > in.Duration:
		return in.Duration + b.rng.IntN(in.MaxDuration-in.Duration+1)
	case in.Duration > 0:
		return in.Duration
	case in.Status == Drain:
		return drainRounds
	case in.Status.dot():
		return dotRounds
	}
	return 1
}

func statusVerb(k StatusKind) string {
	switch k {
	case Burn:
		return "burned"
	case Poison:
		return "poisoned"
	case Bleed:
		return "bleeding"
	case Freeze:
		return "frozen"
	case Drain:
		return "being drained"
	case Wither:
		return "withering"
	case Pain:
		return "in pain"
	case Dread:
		return "filled with dread"
	case Silence:
		return "silenced"
	case Swarm:
		return "swarmed by insects"
	default:
		return "stunned"
	}
}

func (b *Battle) endOfRound() {
	for _, c := range []*Combatant{b.Player, b.Enemy} {
		for k := StatusKind(0); k < numStatus; k++ {
			left := c.status[k]
			if left <= 0 {
				continue
			}
			if k.dot() && c.Alive() {
				share := 16
				if k == Drain {
					share = 8
				}
				dmg := min(max(1, c.Base.MaxHP/share), c.HP)
				c.HP -= dmg
				b.emitHit(c, CueHit, "%s hurt by %s for %d damage.", c.Be(), statusNoun(k), dmg)
				if foe := b.foe(c); k == Drain && foe.Alive() && foe.HP < foe.Base.MaxHP {
					n := min(dmg, foe.Base.MaxHP-foe.HP)
					foe.HP += n
					b.emitCue(CueHeal, "%s %d HP.", foe.Verb("drain"), n)
				}
				if !c.Alive() {
					b.emitCue(CueDefeat, "%s defeated!", c.Be())
				}
			}
			c.status[k] = left - 1
		}
		if c.enraged > 0 {
			c.enraged--
		}
		if c.has(DemonArmor) && c.Alive() {
			if heal := min(max(1, c.Base.MaxHP/16), c.Base.MaxHP-c.HP); heal > 0 {
				c.HP += heal
				b.emitCue(CueHeal, "Demon Armor restores %d HP.", heal)
			}
		}
		if c.regen > 0 && c.Alive() {
			c.regen--
			if heal := min(max(1, c.Base.MaxHP/16), c.Base.MaxHP-c.HP); heal > 0 {
				c.HP += heal
				b.emitCue(CueHeal, "%s %d HP.", c.Verb("regenerate"), heal)
			}
		}
		if c.manaShield > 0 {
			if c.manaShield--; c.manaShield == 0 {
				b.emit("%s mana shield fades.", c.Possessive())
			}
		}
		if c.smoke > 0 {
			c.smoke--
		}
		if c.sealed > 0 {
			c.sealed--
		}
	}
}

func statusNoun(k StatusKind) string {
	switch k {
	case Burn:
		return "the burn"
	case Poison:
		return "poison"
	case Drain:
		return "the drain"
	case Wither:
		return "withering"
	case Pain:
		return "pain"
	case Swarm:
		return "the swarm"
	default:
		return "bleeding"
	}
}

// ChooseEnemyAbility is the enemy AI: a random usable ability, slightly weighted
// (design/mechanics.md: random, weighted). Self buffs are avoided once stacked.
func (b *Battle) ChooseEnemyAbility() string {
	e := b.Enemy
	var options []string
	var weights []float64
	for _, id := range e.Abilities {
		a := Get(id)
		if ok, _ := b.Usable(e, id); !ok {
			continue
		}
		w := 1.0
		if a.Delivery == Self {
			w = 0.4
			for _, sc := range a.Stats {
				if sc.OnSelf && e.stages[sc.Stat] >= 2 {
					w = 0.05
				}
			}
			if a.Special == Smoke && e.smoke > 0 {
				w = 0
			}
		}
		options = append(options, id)
		weights = append(weights, w)
	}
	total := 0.0
	for _, w := range weights {
		total += w
	}
	if total == 0 {
		return WaitID
	}
	r := b.rng.Float64() * total
	for i, w := range weights {
		if r < w {
			return options[i]
		}
		r -= w
	}
	return options[len(options)-1]
}
