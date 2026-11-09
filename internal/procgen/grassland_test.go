package procgen

import (
	"testing"

	"oldworld2/internal/creature"
	"oldworld2/internal/tiles"
)

func TestGenerateChunkIsDeterministic(t *testing.T) {
	a := generateTerrain(42, 3, -2)
	b := generateTerrain(42, 3, -2)
	if a.Ground != b.Ground || a.Blocked != b.Blocked || len(a.Objects) != len(b.Objects) {
		t.Fatal("same seed and coordinates produced different chunks")
	}
	c := generateTerrain(43, 3, -2)
	if a.Ground == c.Ground && a.Blocked == c.Blocked {
		t.Fatal("different seeds produced identical chunks")
	}
}

// Water masks are computed per chunk on a padded area; neighbors must agree on shared edges.
func TestWaterAgreesAcrossChunkBorders(t *testing.T) {
	for seed := int64(1); seed <= 20; seed++ {
		for cx := -2; cx <= 2; cx++ {
			left := waterMask(seed, cx*ChunkSize, 0)
			right := waterMask(seed, (cx+1)*ChunkSize, 0)
			x := (cx + 1) * ChunkSize
			for y := -1; y <= ChunkSize; y++ {
				for dx := -2; dx <= 1; dx++ {
					if left.at(x+dx, y) != right.at(x+dx, y) {
						t.Fatalf("seed %d: water at (%d,%d) differs between chunks", seed, x+dx, y)
					}
				}
			}
		}
	}
}

func TestEveryWaterTileHasAValidPiece(t *testing.T) {
	invalid := 0
	for seed := int64(1); seed <= 20; seed++ {
		for cy := -2; cy <= 2; cy++ {
			for cx := -2; cx <= 2; cx++ {
				m := waterMask(seed, cx*ChunkSize, cy*ChunkSize)
				for y := cy * ChunkSize; y < (cy+1)*ChunkSize; y++ {
					for x := cx * ChunkSize; x < (cx+1)*ChunkSize; x++ {
						if m.at(x, y) {
							if _, ok := m.neighbors(x, y).piece(); !ok {
								invalid++
							}
						}
					}
				}
			}
		}
	}
	if invalid > 0 {
		t.Fatalf("%d water tiles need a fallback piece", invalid)
	}
}

// A tree sticking out of its chunk must block the covered tile in the neighbor chunk too.
func TestTreesBlockAcrossChunkBorders(t *testing.T) {
	for seed := int64(1); seed <= 10; seed++ {
		for cx := -3; cx <= 3; cx++ {
			left, right := generateTerrain(seed, cx, 0), generateTerrain(seed, cx+1, 0)
			for _, o := range left.Objects {
				s := tiles.Get(o.Sprite)
				if o.X+s.W <= (cx+1)*ChunkSize {
					continue
				}
				for dy := 0; dy < s.H; dy++ {
					if !right.Blocked[Index(0, o.Y+dy)] {
						t.Fatalf("seed %d: tree at (%d,%d) doesn't block the neighbor chunk", seed, o.X, o.Y)
					}
				}
			}
		}
	}
}

// No enemy may spawn in a hole: every spawn in the start zone must be reachable on foot from
// the player's start. Checked with a separate flood fill over the whole zone.
func TestSpawnsAreReachable(t *testing.T) {
	for seed := int64(1); seed <= 3; seed++ {
		cache := NewTerrainCache(seed)
		sx, sy := 0, 0
		for !cache.Main(sx, sy) {
			sx++
		}
		reach := floodZone(cache, sx, sy)
		for cy := -8; cy < 8; cy++ {
			for cx := -8; cx < 8; cx++ {
				for _, s := range cache.Generate(cx, cy).Spawns {
					if !reach[[2]int{s.X, s.Y}] {
						t.Fatalf("seed %d: %v at (%d,%d) can't be reached from the start", seed, s.Kind, s.X, s.Y)
					}
				}
			}
		}
	}
}

func floodZone(t *TerrainCache, x, y int) map[[2]int]bool {
	seen := map[[2]int]bool{{x, y}: true}
	queue := [][2]int{{x, y}}
	for len(queue) > 0 {
		p := queue[0]
		queue = queue[1:]
		for _, d := range [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			n := [2]int{p[0] + d[0], p[1] + d[1]}
			if zx, zy := ZoneOf(n[0], n[1]); zx != 0 || zy != 0 || seen[n] || t.blocked(n[0], n[1]) {
				continue
			}
			seen[n] = true
			queue = append(queue, n)
		}
	}
	return seen
}

// The start zone has an inn, the game starts in front of it, and every road leads somewhere:
// the hub and all crossings are in the zone's main area.
func TestSettlementAndRoadsAreReachable(t *testing.T) {
	for seed := int64(1); seed <= 3; seed++ {
		cache := NewTerrainCache(seed)
		x, y, ok := StartPoint(seed)
		if !ok {
			t.Fatalf("seed %d: the start zone has no inn", seed)
		}
		if !cache.Main(x, y) {
			t.Fatalf("seed %d: the start point (%d,%d) isn't in the main area", seed, x, y)
		}
		l := Layout(seed, 0, 0)
		if !cache.Main(l.HubX, l.HubY) {
			t.Fatalf("seed %d: the settlement isn't in the main area", seed)
		}
		// Each road starts at a crossing on the zone border.
		for i, s := range l.segments {
			if i > 0 && l.segments[i-1].b == s.a {
				continue // not the start of a road
			}
			ex, ey := int(s.a.x), int(s.a.y)
			if !cache.Main(ex, ey) {
				t.Fatalf("seed %d: the road at crossing (%d,%d) is cut off", seed, ex, ey)
			}
		}
	}
}

func TestBuildingsDontBlockRoads(t *testing.T) {
	for seed := int64(1); seed <= 5; seed++ {
		l := Layout(seed, 0, 0)
		for _, b := range l.Buildings {
			s := tiles.Get(b.Sprite)
			for y := b.Y; y < b.Y+s.H; y++ {
				for x := b.X; x < b.X+s.W; x++ {
					if l.isRoad(x, y) {
						t.Fatalf("seed %d: %v at (%d,%d) stands on a road", seed, b.Sprite, b.X, b.Y)
					}
				}
			}
		}
	}
}

// Shrines exist along the roads of the start zone, and the player can walk up to each.
func TestShrinesAreReachable(t *testing.T) {
	for seed := int64(1); seed <= 3; seed++ {
		cache := NewTerrainCache(seed)
		shrines := 0
		for _, b := range Layout(seed, 0, 0).Buildings {
			if b.Sprite != tiles.Shrine {
				continue
			}
			shrines++
			reachable := false
			for _, d := range [][2]int{{0, 2}, {1, 2}, {-1, 0}, {2, 0}, {0, -1}, {1, -1}} {
				if cache.Main(b.X+d[0], b.Y+d[1]) {
					reachable = true
				}
			}
			if !reachable {
				t.Fatalf("seed %d: shrine at (%d,%d) can't be reached", seed, b.X, b.Y)
			}
		}
		if shrines == 0 {
			t.Fatalf("seed %d: the start zone has no shrines", seed)
		}
	}
}

// Elites keep 100 tiles away from settlements, where games start, so a new character meets
// trash first. The rule is for the group's spot; its members stand up to 4 tiles around it.
func TestNoElitesNearSettlements(t *testing.T) {
	for seed := int64(1); seed <= 3; seed++ {
		cache := NewTerrainCache(seed)
		trash := 0
		for cy := -24; cy < 24; cy++ {
			for cx := -24; cx < 24; cx++ {
				for _, s := range cache.Generate(cx, cy).Spawns {
					l := LayoutAt(seed, s.X, s.Y)
					if d := l.hubDist(s.X, s.Y); d < eliteHubSafe-6 {
						if creature.Get(s.Kind).Tier == "elite" {
							t.Fatalf("seed %d: elite %v %.0f tiles from a settlement", seed, s.Kind, d)
						}
						trash++
					}
				}
			}
		}
		if trash == 0 {
			t.Fatalf("seed %d: no enemies near the settlements at all", seed)
		}
	}
}
