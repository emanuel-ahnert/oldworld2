package procgen

import (
	"strings"
	"testing"

	"oldworld2/internal/creature"
	"oldworld2/internal/tiles"
)

func TestZoneTypes(t *testing.T) {
	for seed := int64(1); seed <= 20; seed++ {
		if ZoneTypeOf(seed, 0, 0) != Grassland {
			t.Fatalf("seed %d: the start zone must be grassland", seed)
		}
		for zx := -3; zx <= 3; zx++ {
			for zy := -3; zy <= 3; zy++ {
				if ZoneTypeOf(seed, zx, zy) != ZoneTypeOf(seed, zx, zy) {
					t.Fatal("zone types must be deterministic")
				}
				if zy < 0 && ZoneTypeOf(seed, zx, zy).IsDesert() && abs(zy) >= abs(zx) {
					t.Fatalf("seed %d: no desert north of the start (%d,%d)", seed, zx, zy)
				}
			}
		}
	}
	if Difficulty(0, 0) != 1 || Difficulty(0, 1) != 2 || Difficulty(-2, 1) != 3 {
		t.Fatal("difficulty should be 1 + zone ring")
	}
}

// Seed 1 has a desert directly south of the start (zone 0,1).
func TestDesertZone(t *testing.T) {
	const seed = 1
	if ZoneTypeOf(seed, 0, 1) != Desert {
		t.Skip("seed 1 changed; pick a seed with a desert south of the start")
	}
	cache := NewTerrainCache(seed)
	grass, sand := 0, 0
	for cy := 9; cy < 23; cy += 3 {
		for cx := -7; cx < 7; cx += 3 {
			c := cache.Generate(cx, cy)
			for _, g := range c.Ground {
				switch {
				case g >= tiles.Grass && g <= tiles.GrassTuftE:
					grass++
				case tiles.IsSand(g):
					sand++
				}
			}
			for _, s := range c.Spawns {
				switch s.Kind {
				case creature.Wolf, creature.Boar, creature.Goblin, creature.Bandit:
					t.Fatalf("grassland enemy %v in the desert at (%d,%d)", s.Kind, s.X, s.Y)
				}
				if s.Level < 6 || s.Level > 10 {
					t.Fatalf("difficulty 2 enemies should be level 6–10, got %d", s.Level)
				}
			}
		}
	}
	if grass > 0 || sand == 0 {
		t.Fatalf("desert chunks: %d grass tiles, %d sand tiles", grass, sand)
	}
	l := Layout(seed, 0, 1)
	if _, ok := l.Inn(); !ok {
		t.Fatal("the desert settlement needs an inn")
	}
	for _, b := range l.Buildings {
		if b.Sprite == tiles.HouseOrange || b.Sprite == tiles.HouseOrange2 {
			t.Fatal("desert settlements use desert houses")
		}
	}
}

// Names never repeat within uniqueRadius zones, and repeats farther away are very rare.
func TestZoneNames(t *testing.T) {
	const r = 15 // a 31×31 block of zones
	for _, seed := range []int64{1, 7, 20} {
		names := map[string][2]int{}
		repeats := 0
		for zx := -r; zx <= r; zx++ {
			for zy := -r; zy <= r; zy++ {
				n := ZoneName(seed, zx, zy, ZoneTypeOf(seed, zx, zy))
				if n == "" {
					t.Fatalf("zone (%d,%d) has no name", zx, zy)
				}
				if p, dup := names[n]; dup {
					repeats++
					if max(abs(p[0]-zx), abs(p[1]-zy)) <= uniqueRadius {
						t.Fatalf("seed %d: %q repeats at (%d,%d) and (%d,%d)", seed, n, p[0], p[1], zx, zy)
					}
				}
				names[n] = [2]int{zx, zy}
			}
		}
		total := (2*r + 1) * (2*r + 1)
		if repeats*200 > total { // fewer than 0.5% repeats
			t.Fatalf("seed %d: %d of %d zone names repeat", seed, repeats, total)
		}
		t.Logf("seed %d: %d repeats in %d zones", seed, repeats, total)
	}
}

// Seed 20 has an ancient desert directly south of the start (zone 0,1).
func TestAncientDesert(t *testing.T) {
	const seed = 20
	if ZoneTypeOf(seed, 0, 1) != AncientDesert {
		t.Skip("seed 20 changed; pick a seed with an ancient desert south of the start")
	}
	cache := NewTerrainCache(seed)
	l := Layout(seed, 0, 1)
	pyramids, sphinxes := 0, 0
	for _, b := range l.Buildings {
		switch b.Sprite {
		case tiles.Pyramid:
			pyramids++
		case tiles.Sphinx:
			sphinxes++
		}
		s := tiles.Get(b.Sprite)
		reachable := false
		for x := b.X - 1; x <= b.X+s.W && !reachable; x++ {
			reachable = cache.Main(x, b.Y+s.H) || cache.Main(x, b.Y-1)
		}
		if !reachable {
			t.Fatalf("%v at (%d,%d) can't be reached", b.Sprite, b.X, b.Y)
		}
	}
	if pyramids == 0 || sphinxes == 0 {
		t.Fatalf("ancient desert monuments: %d pyramids, %d sphinxes", pyramids, sphinxes)
	}
	allowed := map[creature.Kind]bool{creature.Mummy: true, creature.Scarab: true, creature.Cobra: true, creature.TombGuardian: true}
	for cy := 9; cy < 23; cy += 2 {
		for cx := -7; cx < 7; cx += 2 {
			for _, s := range cache.Generate(cx, cy).Spawns {
				if !allowed[s.Kind] {
					t.Fatalf("%v doesn't belong in the ancient desert", s.Kind)
				}
			}
		}
	}
}

// NearestZone finds a zone of each implemented type, of that type, and none closer to the
// start; ParseZoneType reads the names back.
func TestNearestZone(t *testing.T) {
	for _, want := range Implemented {
		zx, zy, ok := NearestZone(3, want)
		if !ok {
			t.Fatalf("no %v zone", want)
		}
		if got := ZoneTypeOf(3, zx, zy); got != want {
			t.Errorf("nearest %v is a %v zone", want, got)
		}
		r := max(abs(zx), abs(zy))
		for y := -r + 1; y < r; y++ {
			for x := -r + 1; x < r; x++ {
				if ZoneTypeOf(3, x, y) == want {
					t.Errorf("%v zone %d,%d is nearer than %d,%d", want, x, y, zx, zy)
				}
			}
		}
		if p, ok := ParseZoneType(strings.ReplaceAll(strings.ToLower(want.String()), " ", "_")); !ok || p != want {
			t.Errorf("ParseZoneType(%v) = %v, %v", want, p, ok)
		}
	}
}
