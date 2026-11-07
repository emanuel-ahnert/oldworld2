package game

import (
	"slices"
	"testing"

	"oldworld2/internal/battle"
	"oldworld2/internal/character"
)

// The battle menu has the four ability slots on top (empty ones marked "") and the basic
// move and Wait in the bottom row, however many abilities are slotted.
func TestBattleMenuLayout(t *testing.T) {
	c := character.New(character.Mage)
	s := &battleScene{b: battle.New(c.Combatant(), character.Enemy(0, 1), 1)}
	want := []string{"ice_bolt", "", "", "", "fire_bolt", battle.WaitID}
	if got := s.options(); !slices.Equal(got, want) {
		t.Fatalf("menu %q, want %q", got, want)
	}
}
