package game

import (
	"slices"
	"testing"

	"oldworld2/internal/character"
)

func levelledMage(level int) *character.Character {
	c := character.New(character.Mage)
	for c.Level < level {
		c.GainXP(character.XPToNext(c.Level))
	}
	return c
}

// With full slots, picking a known ability asks for a slot and then replaces it.
func TestMenuSwapWhenSlotsFull(t *testing.T) {
	c := levelledMage(8)
	m := &abilitiesMenu{g: &Game{char: c}}
	m.column, m.row = 1, slices.Index(m.known(), "tidal_splash")
	m.confirm()
	if m.pending != "tidal_splash" || m.column != 0 {
		t.Fatalf("expected to choose a slot for Tidal Splash, pending %q column %d", m.pending, m.column)
	}
	m.row = 3 // replace Blink
	m.confirm()
	if c.Active[3] != "tidal_splash" || m.pending != "" {
		t.Fatalf("slots after the swap: %v", c.Active)
	}
	if !slices.Contains(m.known(), "blink") {
		t.Fatal("the replaced ability should be back in the known list")
	}
}

// Passives and fallback attacks are listed but can't be put into a slot.
func TestMenuIgnoresFallback(t *testing.T) {
	c := levelledMage(1)
	m := &abilitiesMenu{g: &Game{char: c}}
	m.column, m.row = 1, slices.Index(m.known(), "fire_bolt")
	m.confirm()
	if slices.Contains(c.Active, "fire_bolt") || m.pending != "" {
		t.Fatalf("Fire Bolt must not be slotted: %v", c.Active)
	}
}
