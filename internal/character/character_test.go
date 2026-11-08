package character

import (
	"slices"
	"strings"
	"testing"

	"oldworld2/internal/battle"
	"oldworld2/internal/creature"
)

// The XP table in design/mechanics.md (XP Curve).
func TestXPToNextMatchesDesign(t *testing.T) {
	for level, want := range map[int]int{1: 20, 2: 80, 3: 150, 4: 240, 5: 350, 10: 900, 20: 2200, 100: 15000} {
		if got := XPToNext(level); got != want {
			t.Errorf("XPToNext(%d) = %d, want %d", level, got, want)
		}
	}
}

func TestTwoWolvesReachLevelTwo(t *testing.T) {
	c := New(Warrior)
	xp := EnemyXP(1, 1, creature.Get(creature.Wolf).Tier)
	c.GainXP(xp)
	if c.Level != 1 {
		t.Fatalf("one wolf shouldn't level up, got level %d", c.Level)
	}
	ups := c.GainXP(xp)
	if c.Level != 2 || len(ups) != 1 {
		t.Fatalf("two wolves should reach level 2, got level %d", c.Level)
	}
}

func TestLevelUpLearnsAbilities(t *testing.T) {
	c := New(Mage)
	c.GainXP(XPToNext(1) + XPToNext(2))
	if c.Level != 3 || !slices.Contains(c.Known, "mana_shield") || !slices.Contains(c.Active, "mana_shield") {
		t.Fatalf("a level 3 mage should know and use Mana Shield: level %d, known %v", c.Level, c.Known)
	}
}

func TestPassivesDontTakeActiveSlots(t *testing.T) {
	c := New(Warrior)
	c.GainXP(XPToNext(1) + XPToNext(2)) // level 3: Enrage
	if slices.Contains(c.Active, "enrage") {
		t.Fatal("Enrage is passive and must not take an active slot")
	}
	if !slices.Contains(c.Combatant().Abilities, "enrage") {
		t.Fatal("the battle combatant must still have Enrage")
	}
}

func TestDefeatCostsMoneyAndLevelXP(t *testing.T) {
	c := New(Warrior)
	c.GainXP(XPToNext(1) + 15) // level 2 and 15 XP toward level 3
	c.Money, c.HP = 1234, 1
	if money, xp := c.Defeated(); money != 123 || c.Money != 1111 || xp != 15 || c.XP != 0 || c.Level != 2 {
		t.Fatalf("lost %d money and %d XP, left %d money, %d XP, level %d; want 123, 15, 1111, 0, 2",
			money, xp, c.Money, c.XP, c.Level)
	}
	if c.HP != float64(c.Stats().MaxHP) {
		t.Fatal("respawn should fully heal")
	}
}

func TestStatGrowth(t *testing.T) {
	c := New(Warrior)
	c.Level = 10
	s := c.Stats()
	// design/classes.md: HP × (1 + 0.09 × 9), others × (1 + 0.1 × 9); MP doesn't grow.
	if s.MaxHP != 217 || s.Attack != 27 || s.MaxMP != 150 {
		t.Fatalf("level 10 warrior: HP %d, Attack %d, MP %d", s.MaxHP, s.Attack, s.MaxMP)
	}
}

func TestCampRegeneration(t *testing.T) {
	c := New(Warrior)
	maxHP, maxMP := float64(c.Stats().MaxHP), float64(c.Stats().MaxMP)
	c.HP, c.MP = maxHP/2, maxMP/2
	for i := 0; i < 60; i++ {
		c.RegenerateAtCamp()
	}
	if got := c.HP - maxHP/2; got < maxHP*0.099 || got > maxHP*0.101 {
		t.Fatalf("one second at camp restored %.1f HP, want 10%% of %.0f", got, maxHP)
	}
	if got := c.MP - maxMP/2; got < maxMP*0.199 || got > maxMP*0.201 {
		t.Fatalf("one second at camp restored %.1f MP, want 20%% of %.0f", got, maxMP)
	}
}

// Wait, passives and fallback attacks never take one of the 4 ability slots.
func TestSlotsOnlyHoldRegularAbilities(t *testing.T) {
	c := New(Mage)
	if !slices.Equal(c.Active, []string{"ice_bolt"}) {
		t.Fatalf("a new mage's slots are %v, want only Ice Bolt (Fire Bolt is the fallback)", c.Active)
	}
	if !slices.Contains(c.Combatant().Abilities, "fire_bolt") {
		t.Fatal("the fallback attack must still be available in battle")
	}
	for c.Level < 10 {
		c.GainXP(XPToNext(c.Level))
	}
	if len(c.Active) != MaxActive || slices.Contains(c.Active, "fire_bolt") || slices.Contains(c.Active, "wait") {
		t.Fatalf("a level 10 mage's slots are %v", c.Active)
	}
}

func TestEquipReplaceUnequip(t *testing.T) {
	c := New(Mage)
	for c.Level < 8 {
		c.GainXP(XPToNext(c.Level))
	}
	// Level 8: Ice Bolt, Frost Ring, Mana Shield, Blink in the slots; Tidal Splash doesn't fit.
	if c.Equip("tidal_splash") {
		t.Fatal("equipping into full slots must fail")
	}
	if !c.Replace(1, "tidal_splash") || c.Active[1] != "tidal_splash" {
		t.Fatalf("replace failed: %v", c.Active)
	}
	c.Unequip(0)
	if len(c.Active) != 3 || !c.Equip("ice_bolt") {
		t.Fatalf("unequip/equip failed: %v", c.Active)
	}
	if c.Equip("fire_bolt") || c.Replace(0, "fire_bolt") {
		t.Fatal("the fallback attack must never take a slot")
	}
}

// Every class has known abilities, a free fallback attack among its starting ones, and its
// level-up abilities exist.
func TestAllClassesComplete(t *testing.T) {
	for c := Class(0); c < NumClasses; c++ {
		info := c.Info()
		if info.Name == "" {
			t.Fatalf("class %d has no info", c)
		}
		if got, ok := ParseClass(strings.ToLower(info.Name)); !ok || got != c {
			t.Errorf("ParseClass(%q) = %v, %v", info.Name, got, ok)
		}
		fallback := false
		for _, id := range info.Start {
			a := battle.Get(id)
			if a == nil {
				t.Fatalf("%s: unknown starting ability %q", info.Name, id)
			}
			fallback = fallback || (a.Fallback && a.Cost == 0)
		}
		if !fallback {
			t.Errorf("%s has no fallback attack", info.Name)
		}
		for level, id := range info.Learn {
			if battle.Get(id) == nil {
				t.Errorf("%s: unknown ability %q at level %d", info.Name, id, level)
			}
		}
		for level, id := range info.Perks {
			if a := battle.Get(id); a == nil || !a.Passive {
				t.Errorf("%s: perk %q at level %d is not a known passive", info.Name, id, level)
			}
		}
		ch := New(c)
		if len(ch.Active) == 0 {
			t.Errorf("%s starts without an active ability", info.Name)
		}
	}
}

// Casters learn an area spell by level 2, so smoke (bandits) can't leave them only Wait.
func TestCastersGetAreaSpellEarly(t *testing.T) {
	for _, c := range []Class{Mage, Warlock, Priest, Shaman, Druid} {
		a := battle.Get(c.Info().Learn[2])
		if a == nil || a.Delivery != battle.AreaSpell {
			t.Errorf("%s doesn't learn an area spell at level 2", c.Info().Name)
		}
	}
}

// Heroes take neutral damage from every element but keep their class's Affinity bonus.
func TestHeroesArePhysical(t *testing.T) {
	for c := Class(0); c < NumClasses; c++ {
		hero := New(c).Combatant()
		for e := battle.Physical; e <= battle.Dark; e++ {
			if m := battle.Effectiveness(e, hero.Types); m != 1 {
				t.Errorf("%s takes %v× from %s, want 1×", c.Info().Name, m, e)
			}
		}
	}
	enemy, bolt := Enemy(creature.Wolf, 5), battle.Get("fire_bolt")
	mage, warrior := New(Mage).Combatant(), New(Warrior).Combatant()
	warrior.Base = mage.Base
	b := battle.New(mage, enemy, 1)
	if b.Damage(mage, enemy, bolt, false, 1) <= b.Damage(warrior, enemy, bolt, false, 1) {
		t.Error("a Mage's Fire Bolt should get the Affinity bonus")
	}
}

// Rogues learn Improved Sprint at level 6 along with Kidney Shot; it makes sprinting use up
// stamina at 20% of the normal rate.
func TestRogueImprovedSprint(t *testing.T) {
	c := New(Rogue)
	if c.StaminaUse() != 1 {
		t.Fatalf("level 1 stamina use %v", c.StaminaUse())
	}
	var learned []string
	for c.Level < 6 {
		for _, up := range c.GainXP(XPToNext(c.Level)) {
			if up.Level == 6 {
				learned = up.Learned
			}
		}
	}
	if !slices.Contains(learned, "kidney_shot") || !slices.Contains(learned, "improved_sprint") {
		t.Fatalf("learned at level 6: %v", learned)
	}
	if c.StaminaUse() != 0.2 || slices.Contains(c.Active, "improved_sprint") {
		t.Fatalf("stamina use %v, active %v", c.StaminaUse(), c.Active)
	}
}

func TestXPFallsOffForWeakEnemies(t *testing.T) {
	for _, c := range []struct{ below, want int }{{0, 100}, {4, 60}, {5, 50}, {6, 40}, {9, 10}, {10, 0}, {14, 0}} {
		// A level 10 enemy against a player that many levels above it; full XP is 100.
		if got := EnemyXP(10, 10+c.below, "trash"); got != c.want {
			t.Errorf("%d levels below: %d XP, want %d", c.below, got, c.want)
		}
	}
}
