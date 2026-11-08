package battle

import (
	"math"
	"slices"
	"strings"
	"testing"
)

func fighter(name string, types []Element, s Stats, abilities ...string) *Combatant {
	return &Combatant{Name: name, Level: 1, Types: types, Base: s, HP: s.MaxHP, MP: s.MaxMP, Abilities: abilities}
}

func warrior() *Combatant {
	return fighter("Warrior", []Element{Physical},
		Stats{MaxHP: 120, MaxMP: 150, Attack: 14, Magic: 4, Defense: 13, Resistance: 8, Speed: 10},
		"heavy_strike", "battle_cry", "overwhelm", "enrage")
}

func wolf() *Combatant {
	return fighter("Wolf", []Element{Physical},
		Stats{MaxHP: 50, Attack: 8, Magic: 2, Defense: 10, Resistance: 8, Speed: 12}, "bite", "howl")
}

func TestEffectiveness(t *testing.T) {
	cases := []struct {
		attack Element
		def    []Element
		want   float64
	}{
		{Fire, []Element{Ice, Dark}, 4},
		{Fire, []Element{Water, Nature}, 0.25},
		{Nature, []Element{Air, Fire}, 0},
		{Physical, []Element{Holy, Dark}, 1},
		{Ice, []Element{Fire, Water, Ice}, 0.125},
	}
	for _, c := range cases {
		if got := Effectiveness(c.attack, c.def); got != c.want {
			t.Errorf("%v vs %v = %v, want %v", c.attack, c.def, got, c.want)
		}
	}
}

// The rough check in design/world.md: a level 1 Warrior deals about 26 with Heavy Strike to a
// wolf and takes about 14 per Bite.
func TestDamageMatchesDesignNumbers(t *testing.T) {
	w, f := warrior(), wolf()
	b := New(w, f, 1)
	if got := b.Damage(w, f, Get("heavy_strike"), false, 1); got < 24 || got > 28 {
		t.Errorf("Heavy Strike on a wolf = %d, want about 26", got)
	}
	if got := b.Damage(f, w, Get("bite"), false, 1); got < 12 || got > 16 {
		t.Errorf("Bite on a warrior = %d, want about 14", got)
	}
}

func TestOverwhelmOnlyAfterDodge(t *testing.T) {
	w := warrior()
	w.MP = 9999 // enough for 20 Heavy Strikes
	flyer := fighter("Flyer", []Element{Air}, Stats{MaxHP: 999, Attack: 1, Defense: 7, Resistance: 8, Speed: 1}, "bite")
	flyer.Flying = true
	b := New(w, flyer, 3)
	if ok, _ := b.Usable(w, "overwhelm"); ok {
		t.Fatal("Overwhelm must not be usable before a miss")
	}
	for i := 0; i < 20; i++ {
		b.Turn("heavy_strike", WaitID)
		if ok, _ := b.Usable(w, "overwhelm"); ok {
			hp := flyer.HP
			b.Turn("overwhelm", WaitID)
			if flyer.HP >= hp {
				t.Fatal("Overwhelm should hit and can't be dodged")
			}
			return
		}
	}
	t.Fatal("a flying target should dodge a melee attack at some point in 20 tries")
}

func TestDamageOverTimeIsSixteenthOfMaxHP(t *testing.T) {
	w := warrior()
	w.Abilities = append(w.Abilities, "rend")
	f := wolf()
	f.Base.MaxHP, f.HP = 160, 160
	b := New(w, f, 7)
	b.Turn("rend", WaitID)
	afterHit := f.HP
	b.Turn(WaitID, WaitID)
	if got := afterHit - f.HP; got != 10 {
		t.Fatalf("bleed tick = %d, want 160/16 = 10", got)
	}
}

func TestBattleEnds(t *testing.T) {
	w, f := warrior(), wolf()
	b := New(w, f, 11)
	for i := 0; i < 50 && !b.Over(); i++ {
		b.Turn("heavy_strike", b.ChooseEnemyAbility())
	}
	if !b.Over() || !b.Won() {
		t.Fatalf("warrior should beat a level 1 wolf: warrior %d HP, wolf %d HP", w.HP, f.HP)
	}
}

func TestStageMultiplier(t *testing.T) {
	for n, want := range map[int]float64{-2: 0.5, -1: 2.0 / 3, 0: 1, 1: 1.5, 2: 2, 3: 2.5} {
		if got := stageMultiplier(n); math.Abs(got-want) > 1e-9 {
			t.Errorf("stage %d = %v, want %v", n, got, want)
		}
	}
}

func mage(mp int) *Combatant {
	m := fighter("Mage", []Element{Ice, Water, Fire},
		Stats{MaxHP: 85, MaxMP: 225, Attack: 4, Magic: 16, Defense: 7, Resistance: 12, Speed: 10},
		"ice_bolt", "mana_shield", "fire_bolt")
	m.MP = mp
	return m
}

// The free basic move is always offered and usable, after the regular abilities and before
// Wait.
func TestBasicMoveAlwaysOffered(t *testing.T) {
	b := New(mage(225), wolf(), 1)
	for _, mp := range []int{225, 10} {
		b.Player.MP = mp
		opts := b.MenuOptions(b.Player)
		if want := []string{"ice_bolt", "mana_shield", "fire_bolt", WaitID}; !slices.Equal(opts, want) {
			t.Fatalf("with %d MP the menu is %v, want %v", mp, opts, want)
		}
		if ok, _ := b.Usable(b.Player, "fire_bolt"); !ok {
			t.Fatalf("with %d MP Fire Bolt must be usable", mp)
		}
	}
}

// Mana Shield (+2) goes before Tusk Charge (+1), which goes before a faster normal move.
func TestPriorityBeatsSpeed(t *testing.T) {
	m := mage(225)
	m.Base.Speed = 1
	boar := fighter("Boar", []Element{Physical, Nature},
		Stats{MaxHP: 60, Attack: 9, Defense: 12, Resistance: 6, Speed: 50}, "gore", "tusk_charge")
	b := New(m, boar, 5)
	events := b.Turn("mana_shield", "tusk_charge")
	if !strings.Contains(events[0].Text, "Mana Shield") {
		t.Fatalf("Mana Shield should go first, got %q", events[0].Text)
	}
	b2 := New(mage(225), boar, 6)
	b2.Player.Base.Speed = 99
	events = b2.Turn("ice_bolt", "tusk_charge")
	if !strings.Contains(events[0].Text, "Tusk Charge") {
		t.Fatalf("Tusk Charge (+1) should beat a faster normal move, got %q", events[0].Text)
	}
}

// Charge goes first (+1), stuns the target for this round only, and works once, in round 1.
func TestCharge(t *testing.T) {
	w := warrior()
	w.Abilities = append(w.Abilities, "charge")
	f := wolf() // faster than the warrior, but Charge has priority
	b := New(w, f, 3)
	if ok, _ := b.Usable(w, "charge"); !ok {
		t.Fatal("Charge should be usable in the first round")
	}
	events := b.Turn("charge", "bite")
	text := ""
	for _, e := range events {
		text += e.Text + "\n"
	}
	if !strings.Contains(text, "Wolf is stunned and can't move") || strings.Contains(text, "Wolf uses Bite") {
		t.Fatalf("Charge should stun the wolf for this round:\n%s", text)
	}
	if ok, _ := b.Usable(w, "charge"); ok {
		t.Fatal("Charge must not be usable after the first round")
	}
	events = b.Turn("heavy_strike", "bite")
	stunned := false
	for _, e := range events {
		stunned = stunned || strings.Contains(e.Text, "stunned")
	}
	if stunned {
		t.Fatal("the stun must only last the round of the Charge")
	}
}

// Enrage always triggers on a critical hit taken; so does Second Wind, which never triggers
// on the Warrior's own attacks.
func TestCritTriggers(t *testing.T) {
	for seed := uint64(1); seed <= 200; seed++ {
		w := warrior()
		w.Abilities = append(w.Abilities, "second_wind")
		w.Base.MaxHP, w.HP = 9999, 9999
		b := New(w, wolf(), seed)
		events := b.Turn(WaitID, "bite")
		crit, healed := false, false
		for _, e := range events {
			crit = crit || strings.Contains(e.Text, "critical")
			healed = healed || strings.Contains(e.Text, "Second Wind")
		}
		if crit && !healed {
			t.Fatalf("seed %d: Second Wind must trigger on a critical hit taken", seed)
		}
	}
	for seed := uint64(1); seed <= 200; seed++ {
		w := warrior()
		w.Abilities = append(w.Abilities, "second_wind")
		w.HP = 50
		f := wolf()
		f.Base.MaxHP, f.HP = 9999, 9999
		b := New(w, f, seed)
		for _, e := range b.Turn("heavy_strike", WaitID) {
			if strings.Contains(e.Text, "Second Wind") {
				t.Fatalf("seed %d: Second Wind must not trigger on the Warrior's own hit", seed)
			}
		}
	}
	for seed := uint64(1); seed <= 200; seed++ {
		w := warrior()
		w.Base.MaxHP, w.HP = 9999, 9999
		b := New(w, wolf(), seed)
		events := b.Turn(WaitID, "bite")
		crit, enraged := false, false
		for _, e := range events {
			crit = crit || strings.Contains(e.Text, "critical")
			enraged = enraged || strings.Contains(e.Text, "enraged")
		}
		if crit && !enraged {
			t.Fatalf("seed %d: Enrage must trigger when taking a critical hit", seed)
		}
	}
}

// Second Wind's Regen heals 1/16 max HP at the end of each of its rounds, then stops.
func TestSecondWindRegen(t *testing.T) {
	w := warrior()
	w.Base.MaxHP, w.HP = 160, 10
	b := New(w, wolf(), 1)
	b.Player.regen = secondWindRegen
	for i := 0; i < secondWindRegen+2; i++ {
		b.endOfRound()
	}
	if want := 10 + secondWindRegen*10; b.Player.HP != want {
		t.Fatalf("HP %d after Regen, want %d", b.Player.HP, want)
	}
}

func caster(abilities ...string) *Combatant {
	return fighter("Caster", []Element{Holy},
		Stats{MaxHP: 160, MaxMP: 300, Attack: 8, Magic: 14, Defense: 10, Resistance: 12, Speed: 30}, abilities...)
}

// A heal restores its share of max HP, never above the maximum.
func TestHeal(t *testing.T) {
	c := caster("mend")
	c.HP = 10
	b := New(c, wolf(), 1)
	b.Turn("mend", WaitID)
	if c.HP != 50 {
		t.Fatalf("HP %d after Mend, want 50", c.HP)
	}
	c.HP = 150
	b.Turn("mend", WaitID)
	if c.HP != 160 {
		t.Fatalf("HP %d, want capped at 160", c.HP)
	}
}

// A Shield takes damage before HP, up to 25% of max HP.
func TestShieldAbsorbs(t *testing.T) {
	c := caster("ward_of_light")
	b := New(c, wolf(), 1)
	b.Turn("ward_of_light", WaitID)
	if c.shield != 40 {
		t.Fatalf("shield %d, want 40", c.shield)
	}
	if got := b.absorb(c, 30); got != 0 || c.shield != 10 {
		t.Fatalf("absorbed to %d, shield %d; want 0 and 10", got, c.shield)
	}
	if got := b.absorb(c, 30); got != 20 || c.shield != 0 {
		t.Fatalf("absorbed to %d, shield %d; want 20 and 0", got, c.shield)
	}
}

// Drain deals 1/8 of the target's max HP per round and heals the caster as much, for 5 rounds.
func TestDrainHealsCaster(t *testing.T) {
	c := caster()
	c.HP = 50
	w := wolf()
	w.Base.MaxHP, w.HP = 800, 800
	b := New(c, w, 1)
	b.inflict(w, Inflict{Status: Drain})
	for i := 0; i < 7; i++ {
		b.endOfRound()
	}
	if w.HP != 800-5*100 || c.HP != 160 {
		t.Fatalf("target HP %d, caster HP %d; want 300 and 160", w.HP, c.HP)
	}
}

// Silence stops spells but not physical abilities.
func TestSilence(t *testing.T) {
	c := caster("smite", "volley", "mend")
	b := New(c, wolf(), 1)
	c.status[Silence] = 2
	if ok, _ := b.Usable(c, "smite"); ok {
		t.Fatal("Smite must not be usable while silenced")
	}
	for _, id := range []string{"volley", "mend"} {
		if ok, _ := b.Usable(c, id); !ok {
			t.Fatalf("%s must stay usable while silenced", id)
		}
	}
}

// In Bear Form, spells are melee: smoke doesn't block them and they hit Defense.
func TestBearForm(t *testing.T) {
	c := caster("thorn_lash", "bear_form")
	w := wolf()
	b := New(c, w, 1)
	w.smoke = 2
	if ok, _ := b.Usable(c, "thorn_lash"); ok {
		t.Fatal("a targeted spell must not get through smoke")
	}
	b.Turn("bear_form", WaitID)
	if !c.bear || c.stages[Defense] != bearDefense {
		t.Fatal("Bear Form must be on with Defense up")
	}
	w.smoke = 2
	if ok, _ := b.Usable(c, "thorn_lash"); !ok {
		t.Fatal("in Bear Form, attacks are melee and get through smoke")
	}
	w.Base.Resistance = 1000 // only Defense counts now
	if d := b.Damage(c, w, Get("thorn_lash"), false, 1); d < 5 {
		t.Fatalf("damage %d: Bear Form must hit Defense, not Resistance", d)
	}
}

// Judgment Hammer stuns for 1 to 3 rounds, and every length happens.
func TestRandomStunLength(t *testing.T) {
	seen := map[int]bool{}
	for seed := uint64(1); seed <= 100; seed++ {
		b := New(caster(), wolf(), seed)
		d := b.durationFor(Get("judgment_hammer").Inflict[0])
		if d < 1 || d > 3 {
			t.Fatalf("stun of %d rounds", d)
		}
		seen[d] = true
	}
	if len(seen) != 3 {
		t.Fatalf("lengths seen: %v", seen)
	}
}

// Holy Seal adds Holy damage to physical hits.
func TestHolySeal(t *testing.T) {
	c := caster("holy_seal", "hammer_blow")
	w := wolf()
	w.Base.MaxHP, w.HP = 9999, 9999
	b := New(c, w, 1)
	b.Turn("holy_seal", WaitID)
	burned := false
	for _, e := range b.Turn("hammer_blow", WaitID) {
		burned = burned || strings.Contains(e.Text, "seal burns")
	}
	if !burned {
		t.Fatal("a physical hit under Holy Seal must deal extra Holy damage")
	}
}

// Dread sometimes costs the target its turn.
func TestDread(t *testing.T) {
	lost := 0
	for seed := uint64(1); seed <= 50; seed++ {
		c := caster()
		w := wolf()
		b := New(c, w, seed)
		w.status[Dread] = 3
		for _, e := range b.Turn(WaitID, "bite") {
			if strings.Contains(e.Text, "dread") {
				lost++
			}
		}
	}
	if lost < 10 || lost > 40 {
		t.Fatalf("Dread cost %d of 50 turns, want about half", lost)
	}
}

// Demon Armor heals 1/16 max HP every round, for the whole battle.
func TestDemonArmor(t *testing.T) {
	c := caster("demon_armor")
	c.HP = 10
	b := New(c, wolf(), 1)
	for i := 0; i < 12; i++ {
		b.endOfRound()
	}
	if c.HP != 10+12*10 {
		t.Fatalf("HP %d after 12 rounds, want 130", c.HP)
	}
}

// The bandit's Smoke Screen keeps ranged attacks and targeted spells away for 3 rounds
// (the round it's used and the two after).
func TestSmokeScreenLasts(t *testing.T) {
	c := caster("smite")
	bandit := fighter("Bandit", []Element{Physical}, Stats{MaxHP: 9999, Attack: 1, Defense: 10, Resistance: 9999, Speed: 1},
		"smoke_screen")
	b := New(c, bandit, 1)
	b.Turn(WaitID, "smoke_screen")
	for round := 2; round <= 3; round++ {
		if ok, _ := b.Usable(c, "smite"); ok {
			t.Fatalf("round %d: Smite must not get through the smoke", round)
		}
		b.Turn(WaitID, WaitID)
	}
	if ok, _ := b.Usable(c, "smite"); !ok {
		t.Fatal("the smoke must be gone after 3 rounds")
	}
}

// Damage over time says how much it deals.
func TestDotMessageShowsDamage(t *testing.T) {
	w := wolf()
	w.Base.MaxHP, w.HP = 160, 160
	b := New(caster(), w, 1)
	b.inflict(w, Inflict{Status: Wither})
	b.events = nil
	b.endOfRound()
	if len(b.events) == 0 || b.events[0].Text != "Wolf is hurt by withering for 10 damage." {
		t.Fatalf("events %v", b.events)
	}
}

func TestCheatDeathOncePerBattle(t *testing.T) {
	rogue := fighter("Rogue", []Element{Physical},
		Stats{MaxHP: 90, MaxMP: 150, Attack: 14, Magic: 5, Defense: 9, Resistance: 8, Speed: 11}, "cheat_death")
	rogue.You = true
	w := wolf()
	w.Base.Attack = 40 // every bite kills a rogue on 1 HP
	b := New(rogue, w, 1)
	rogue.HP = 1
	events := b.Turn(WaitID, "bite")
	if rogue.HP != 9 {
		t.Fatalf("after the first killing blow HP = %d, want 9 (10%%)", rogue.HP)
	}
	if !slices.ContainsFunc(events, func(e Event) bool { return strings.HasPrefix(e.Text, "Cheat Death!") }) {
		t.Fatal("no Cheat Death message")
	}
	for i := 0; i < 5 && rogue.Alive(); i++ {
		b.Turn(WaitID, "bite")
	}
	if rogue.Alive() {
		t.Fatal("Cheat Death worked more than once in a battle")
	}
	// A new battle, a new chance.
	rogue.HP = 1
	New(rogue, w, 2).Turn(WaitID, "bite")
	if rogue.HP != 9 {
		t.Fatalf("in the next battle HP = %d, want 9", rogue.HP)
	}
}

func TestWrathTotemRaisesMagicAtStart(t *testing.T) {
	shaman := fighter("Shaman", []Element{Physical},
		Stats{MaxHP: 105, MaxMP: 150, Attack: 11, Magic: 11, Defense: 11, Resistance: 10, Speed: 10}, "storm_bolt", "wrath_totem")
	b := New(shaman, wolf(), 1)
	events := b.Start()
	if shaman.stages[Magic] != 1 || !slices.Contains(shaman.Totems(), WrathTotem) {
		t.Fatalf("Magic stage %d, totems %v; want 1 and the Wrath Totem", shaman.stages[Magic], shaman.Totems())
	}
	if len(events) == 0 || events[0].Cue != CueTotem || events[0].User != shaman {
		t.Fatalf("the first start event should put down the totem: %+v", events)
	}
	if New(warrior(), wolf(), 1).Start() != nil {
		t.Fatal("a battle without the passive should start without events")
	}
}

func TestTotemsShareOneMessageAndHealingStreamHeals(t *testing.T) {
	shaman := fighter("Shaman", []Element{Physical},
		Stats{MaxHP: 160, MaxMP: 150, Attack: 11, Magic: 11, Defense: 11, Resistance: 10, Speed: 10},
		"storm_bolt", "wrath_totem", "healing_stream_totem")
	shaman.You = true
	b := New(shaman, wolf(), 1)
	var totemMessages []string
	for _, e := range b.Start() {
		if e.Cue == CueTotem {
			totemMessages = append(totemMessages, e.Text)
		}
	}
	if len(totemMessages) != 1 || totemMessages[0] != "You put down a Wrath Totem and a Healing Stream Totem!" {
		t.Fatalf("totem messages %q, want one naming both", totemMessages)
	}
	shaman.HP = 50
	b.Turn(WaitID, WaitID)
	if shaman.HP != 60 { // 1/16 of 160
		t.Fatalf("HP after a round %d, want 60", shaman.HP)
	}
}

// A turn is resolved before its messages are shown, so each message carries the effects as
// they were when it happened: Flame Shock's Burn shows only from its own message on.
func TestEventEffectsFollowTheMessages(t *testing.T) {
	hasBurn := func(es []Effect) bool {
		return slices.ContainsFunc(es, func(e Effect) bool { return e.Label == Burn.String() })
	}
	for seed := uint64(1); seed < 50; seed++ {
		shaman := fighter("Shaman", []Element{Physical},
			Stats{MaxHP: 105, MaxMP: 150, Attack: 11, Magic: 11, Defense: 11, Resistance: 10, Speed: 30}, "flame_shock")
		w := wolf()
		w.Base.MaxHP, w.HP = 500, 500
		b := New(shaman, w, seed)
		events := b.Turn("flame_shock", WaitID)
		if !hasBurn(w.Effects()) {
			continue // the burn didn't take this time
		}
		use := slices.IndexFunc(events, func(e Event) bool { return e.Cue == CueUse && e.User == shaman })
		burn := slices.IndexFunc(events, func(e Event) bool { return e.Cue == CueStatus })
		if use < 0 || burn <= use {
			t.Fatalf("expected the use, then the burn message: %+v", events)
		}
		if hasBurn(events[use].EnemyEffects) {
			t.Fatal("Burn shows with the message that Flame Shock is used")
		}
		if !hasBurn(events[burn].EnemyEffects) {
			t.Fatal("Burn doesn't show with its own message")
		}
		return
	}
	t.Fatal("Flame Shock never burned")
}

func TestPreyOnTheWeak(t *testing.T) {
	rogue := fighter("Rogue", []Element{Physical}, Stats{MaxHP: 90, Attack: 14, Defense: 9, Speed: 11}, "prey_on_the_weak")
	stab := Get("bite") // no crit chance of its own
	for _, c := range []struct {
		rogue, enemy int
		want         float64
	}{{8, 8, baseCrit}, {8, 11, baseCrit}, {8, 7, baseCrit + 0.1}, {8, 3, baseCrit + 0.5}, {20, 1, 1}} {
		rogue.Level = c.rogue
		w := wolf()
		w.Level = c.enemy
		if got := critChance(rogue, w, stab); math.Abs(got-c.want) > 1e-9 {
			t.Errorf("level %d against %d: crit chance %v, want %v", c.rogue, c.enemy, got, c.want)
		}
	}
	if got := critChance(warrior(), wolf(), stab); got != baseCrit {
		t.Errorf("without the passive: %v, want %v", got, baseCrit)
	}
}

func TestPoisonResistance(t *testing.T) {
	poisoned := func(abilities ...string) int {
		r := fighter("Rogue", []Element{Physical}, Stats{MaxHP: 160, Attack: 14, Defense: 9, Speed: 11}, abilities...)
		b := New(r, wolf(), 1)
		r.status[Poison] = 3
		b.Turn(WaitID, WaitID)
		return 160 - r.HP
	}
	if got := poisoned(); got != 10 {
		t.Fatalf("poison without the passive: %d damage, want 10 (1/16)", got)
	}
	if got := poisoned("poison_resistance"); got != 3 { // 10 × 25%, rounded
		t.Fatalf("poison with Poison Resistance: %d damage, want 3", got)
	}
}

func TestThornsHurtMeleeAttackers(t *testing.T) {
	attacked := func(enemyMove string, abilities ...string) int {
		d := fighter("Druid", []Element{Nature}, Stats{MaxHP: 160, Attack: 9, Defense: 10, Speed: 10}, abilities...)
		w := fighter("Wolf", []Element{Physical}, Stats{MaxHP: 50, Attack: 8, Defense: 10, Speed: 12}, "bite", "sling_stone")
		b := New(d, w, 1)
		b.Turn(WaitID, enemyMove)
		return 50 - w.HP
	}
	if got := attacked("bite"); got != 0 {
		t.Fatalf("without Thorns the wolf took %d damage", got)
	}
	if got := attacked("bite", "thorns"); got != 10 { // 160 / 16
		t.Fatalf("a melee bite against Thorns: the wolf took %d, want 10", got)
	}
	if got := attacked("sling_stone", "thorns"); got != 0 {
		t.Fatalf("a ranged attack against Thorns: the attacker took %d, want 0", got)
	}
}
