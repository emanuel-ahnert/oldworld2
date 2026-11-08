package game

import (
	"testing"

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
