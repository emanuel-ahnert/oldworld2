package game

import (
	"testing"
	"time"

	"oldworld2/internal/battle"
	"oldworld2/internal/character"
	"oldworld2/internal/world"
)

// The demo bot's paths go one tile at a time over open ground, and prefer roads.
func TestDemoPathWalksOpenGround(t *testing.T) {
	w := world.New(1)
	sx, sy := w.FindSpawn()
	found := 0
	for _, off := range [][2]int{{25, 0}, {-20, 10}, {5, -30}, {30, 25}} {
		to := [2]int{sx + off[0], sy + off[1]}
		if w.Blocked(to[0], to[1]) {
			continue
		}
		path := findPath(w, [2]int{sx, sy}, to, false)
		if path == nil {
			continue
		}
		found++
		at := [2]int{sx, sy}
		for _, step := range path {
			if abs(step[0]-at[0])+abs(step[1]-at[1]) != 1 || w.Blocked(step[0], step[1]) {
				t.Fatalf("bad step %v → %v on the way to %v", at, step, to)
			}
			at = step
		}
		if at != to {
			t.Fatalf("the path ends at %v, not %v", at, to)
		}
	}
	if found == 0 {
		t.Fatal("no path found to any of the spots")
	}
}

// With next set, the path ends beside the goal (where an enemy stands).
func TestDemoPathEndsBesideEnemy(t *testing.T) {
	w := world.New(1)
	sx, sy := w.FindSpawn()
	to := [2]int{sx + 6, sy}
	path := findPath(w, [2]int{sx, sy}, to, true)
	if path == nil {
		t.Skip("no path here")
	}
	end := [2]int{sx, sy}
	if len(path) > 0 {
		end = path[len(path)-1]
	}
	if abs(end[0]-to[0])+abs(end[1]-to[1]) != 1 {
		t.Fatalf("the path ends at %v, not beside %v", end, to)
	}
}

// Diagonal paths run in long straight stretches, not a staircase of turns.
func TestDemoPathTurnsRarely(t *testing.T) {
	w := world.New(1)
	sx, sy := w.FindSpawn()
	tried := 0
	for _, off := range [][2]int{{24, 24}, {-24, 20}, {20, -26}, {-22, -22}} {
		to := [2]int{sx + off[0], sy + off[1]}
		if w.Blocked(to[0], to[1]) {
			continue
		}
		start := time.Now()
		path := findPath(w, [2]int{sx, sy}, to, false)
		if path == nil {
			continue
		}
		tried++
		if d := time.Since(start); d > 100*time.Millisecond {
			t.Errorf("finding a path took %v", d)
		}
		turns, at, last := 0, [2]int{sx, sy}, [2]int{}
		for i, step := range path {
			dir := [2]int{step[0] - at[0], step[1] - at[1]}
			if i > 0 && dir != last {
				turns++
			}
			at, last = step, dir
		}
		if turns > 8 {
			t.Errorf("the path to %v (%d steps) turns %d times", off, len(path), turns)
		}
	}
	if tried == 0 {
		t.Fatal("no path found to any of the spots")
	}
}

// The demo bot slots its stun opener and uses it on its first turn.
func TestDemoOpensWithKeyMoves(t *testing.T) {
	for _, tc := range []struct {
		class character.Class
		level int
		want  string
	}{{character.Warrior, 6, "charge"}, {character.Rogue, 6, "cheap_shot"}, {character.Paladin, 2, "judgment_hammer"}} {
		c := character.New(tc.class)
		for c.Level < tc.level {
			c.GainXP(character.XPToNext(c.Level))
		}
		demoSlotOpeners(c)
		b := battle.New(c.Combatant(), character.Enemy(0, tc.level), 1)
		if got := demoOpener(b); got != tc.want {
			t.Fatalf("%v opens with %q, want %q (slots %q)", tc.class, got, tc.want, c.Active)
		}
		b.Turn(tc.want, battle.WaitID)
		if got := demoOpener(b); got != "" {
			t.Fatalf("%v: after the first round the opener is %q, want none", tc.class, got)
		}
	}
}
