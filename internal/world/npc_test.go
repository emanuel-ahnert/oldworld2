package world

import (
	"testing"

	"oldworld2/internal/creature"
	"oldworld2/internal/procgen"
)

// openArea returns a world without generated enemies and the center of a 17×17 area free of
// obstacles, so tests control exactly which enemies exist.
func openArea(t *testing.T) (*World, int, int) {
	t.Helper()
	w := New(1)
	for y := -200; y <= 200; y += 4 {
		for x := -200; x <= 200; x += 4 {
			if w.free(x, y, 8) {
				w.npcs = map[uint64]*NPC{}
				return w, x, y
			}
		}
	}
	t.Fatal("no open area found")
	return nil, 0, 0
}

func (w *World) free(cx, cy, r int) bool {
	for y := cy - r; y <= cy+r; y++ {
		for x := cx - r; x <= cx+r; x++ {
			if w.Blocked(x, y) {
				return false
			}
		}
	}
	return true
}

func place(w *World, k creature.Kind, x, y int) *NPC {
	n := newNPC(procgen.Spawn{ID: uint64(len(w.npcs) + 1), Kind: k, X: x, Y: y})
	w.npcs[n.Spawn.ID] = n
	return n
}

func runNPCs(w *World, p *Player, frames int, canEngage bool) *NPC {
	for i := 0; i < frames; i++ {
		if n := w.UpdateNPCs(p, canEngage); n != nil {
			return n
		}
	}
	return nil
}

func TestAggressiveEnemyCatchesPlayer(t *testing.T) {
	w, x, y := openArea(t)
	p := NewPlayer(x, y)
	wolf := place(w, creature.Wolf, x+6, y)
	if got := runNPCs(w, p, 300, true); got != wolf {
		t.Fatalf("wolf 6 tiles away should catch a standing player, got %v", got)
	}
}

func TestDefensiveEnemyOnlyFightsWhenAdjacent(t *testing.T) {
	w, x, y := openArea(t)
	p := NewPlayer(x, y)
	boar := place(w, creature.Boar, x+3, y)
	boar.timer = 1 << 30 // no wandering, so the distance stays the same
	if got := runNPCs(w, p, 300, true); got != nil {
		t.Fatal("a defensive boar 3 tiles away must not engage")
	}
	p.TX = x + 2
	if got := runNPCs(w, p, 10, true); got != boar {
		t.Fatal("a defensive boar right next to the player should engage")
	}
}

func TestNoEngagementDuringGrace(t *testing.T) {
	w, x, y := openArea(t)
	p := NewPlayer(x, y)
	place(w, creature.Wolf, x+1, y)
	if got := runNPCs(w, p, 60, false); got != nil {
		t.Fatal("no enemy may engage during the grace period")
	}
}

func TestEnemyGivesUpAndWalksHome(t *testing.T) {
	w, x, y := openArea(t)
	p := NewPlayer(x, y)
	wolf := place(w, creature.Wolf, x+7, y)
	runNPCs(w, p, 1, false) // spots the player
	if !wolf.Chasing() {
		t.Fatal("wolf should start chasing")
	}
	wolf.timer = 30 // the chase ends soon
	runNPCs(w, p, 60, false)
	if wolf.state != returning {
		t.Fatalf("wolf should be returning home, state %v", wolf.state)
	}
	p.TX, p.TY = x-30, y // walk far away so it doesn't spot the player again at home
	runNPCs(w, p, 600, false)
	near := abs(wolf.TX-wolf.Spawn.X) <= wanderRadius && abs(wolf.TY-wolf.Spawn.Y) <= wanderRadius
	if wolf.state != idle || !near {
		t.Fatalf("wolf should be idle near home, got state %v at (%d,%d)", wolf.state, wolf.TX, wolf.TY)
	}
}

// Enemies defeated in a saved game don't come back when the world is loaded again.
func TestDefeatedEnemiesStayDefeated(t *testing.T) {
	w := New(1)
	var victim *NPC
	for cx := 0; victim == nil && cx < 20; cx++ {
		w.Chunk(cx, 3)
		if npcs := w.NPCs(); len(npcs) > 0 {
			victim = npcs[0]
		}
	}
	if victim == nil {
		t.Skip("no enemy found")
	}
	loaded := New(1)
	loaded.SetDefeated([]uint64{victim.Spawn.ID})
	cx, cy := ChunkCoord(victim.Spawn.X, victim.Spawn.Y)
	loaded.Chunk(cx, cy)
	for _, n := range loaded.NPCs() {
		if n.Spawn.ID == victim.Spawn.ID {
			t.Fatal("a defeated enemy came back after loading")
		}
	}
}

func TestChaserKeepsGapDuringGrace(t *testing.T) {
	w, x, y := openArea(t)
	p := NewPlayer(x, y)
	wolf := place(w, creature.Wolf, x+6, y)
	runNPCs(w, p, 300, false)
	if d := abs(wolf.TX-p.TX) + abs(wolf.TY-p.TY); d != waitGap {
		t.Fatalf("a chasing wolf should wait %d tiles away during grace, got %d", waitGap, d)
	}
	close := place(w, creature.Wolf, x-1, y)
	runNPCs(w, p, 60, false)
	if d := abs(close.TX-p.TX) + abs(close.TY-p.TY); d < waitGap {
		t.Fatalf("a wolf right next to the player should back off during grace, got distance %d", d)
	}
	if got := runNPCs(w, p, 120, true); got == nil {
		t.Fatal("after grace a waiting wolf should close in and engage")
	}
}
