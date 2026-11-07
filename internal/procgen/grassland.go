package procgen

import "oldworld2/internal/tiles"

// ChunkSize is the edge length of a chunk in tiles.
const ChunkSize = 32

// Salts keep the noise layers independent of each other.
const (
	saltLake uint64 = iota + 1
	saltForest
	saltTree
	saltTreeKind
	saltGrass
	saltDecor
	saltDecorKind
)

const (
	lakeScale     = 32.0
	lakeThreshold = 0.70
	forestScale   = 34.0
	forestStart   = 0.52 // forest noise above this starts to grow trees
	// Forests stay porous so they don't wall off clearings; zone barriers are solid instead.
	maxForestDensity = 0.75
	loneTreeOdds     = 0.015
	bushOdds         = 0.012
	decorOdds        = 0.06

	// The water mask is cleaned up on a padded area so chunk borders agree:
	// each cleanup pass looks one tile further, and autotiling and tree placement look further.
	waterPasses = 16
	waterPad    = waterPasses + 6 // more for trees that stick out of the chunk and palms near water
	palmReach   = 4               // palms grow within this many tiles of an oasis
)

// Object is something placed on top of the ground, anchored at its top-left tile.
type Object struct {
	X, Y   int // world tile coordinates
	Sprite tiles.ID
}

// Chunk is a generated ChunkSize×ChunkSize area of the world.
type Chunk struct {
	CX, CY  int
	Ground  [ChunkSize * ChunkSize]tiles.ID
	Blocked [ChunkSize * ChunkSize]bool
	Objects []Object
	Spawns  []Spawn
}

// Index returns the array index of a local tile coordinate.
func Index(lx, ly int) int { return ly*ChunkSize + lx }

// GenerateChunk generates the chunk at chunk coordinates (cx, cy), including its enemy
// spawns. Its look comes from the zone type (see biome.go).
func GenerateChunk(seed int64, cx, cy int) *Chunk {
	return NewTerrainCache(seed).Generate(cx, cy)
}

// Generate returns a chunk with its spawns, reusing cached terrain.
func (t *TerrainCache) Generate(cx, cy int) *Chunk {
	c := t.chunk(cx, cy)
	if c.Spawns == nil {
		c.Spawns = spawns(t.seed, c, t)
		if c.Spawns == nil {
			c.Spawns = []Spawn{} // mark as done
		}
	}
	return c
}

// generateTerrain generates the ground, objects and collision of a chunk, without spawns.
func generateTerrain(seed int64, cx, cy int) *Chunk {
	c := &Chunk{CX: cx, CY: cy}
	x0, y0 := cx*ChunkSize, cy*ChunkSize
	layout := LayoutAt(seed, x0, y0) // zones are chunk-aligned, so one layout and biome per chunk
	bm := biomeAt(seed, x0, y0)
	water := waterMask(seed, x0, y0)
	road := roadMask(seed, x0, y0)

	for ly := 0; ly < ChunkSize; ly++ {
		for lx := 0; lx < ChunkSize; lx++ {
			i := Index(lx, ly)
			wx, wy := x0+lx, y0+ly
			switch {
			case road.at(wx, wy):
				c.Ground[i] = tiles.Shape(road.piece(wx, wy), bm.road)
			case water.at(wx, wy):
				c.Ground[i] = tiles.Shape(water.piece(wx, wy), bm.water)
				c.Blocked[i] = true
			default:
				c.Ground[i] = groundVariant(seed, bm, wx, wy)
			}
		}
	}

	// Buildings of the settlement; they can reach into neighboring chunks.
	for _, b := range layout.Buildings {
		s := tiles.Get(b.Sprite)
		if b.X >= x0 && b.X < x0+ChunkSize && b.Y >= y0 && b.Y < y0+ChunkSize {
			c.Objects = append(c.Objects, Object{b.X, b.Y, b.Sprite})
		}
		for y := max(b.Y, y0); y < min(b.Y+s.H, y0+ChunkSize); y++ {
			for x := max(b.X, x0); x < min(b.X+s.W, x0+ChunkSize); x++ {
				c.Blocked[Index(x-x0, y-y0)] = true
			}
		}
	}

	occupied := func(lx, ly int) bool { return c.Blocked[Index(lx, ly)] }
	keepClear := func(x, y int) bool { // settlements, roads and shrines stay free of trees and bushes
		l := LayoutAt(seed, x, y) // trees can reach into the next zone
		return l.hubDist(x, y) <= clearRadius || l.roadDist(x, y) <= roadClearance || l.nearBuilding(x, y, 2)
	}

	// Trees are 2×2 and sit on a staggered grid: rows are 2 tiles apart and every other row
	// is shifted by one tile, so forests don't look like orchards. A shifted tree can stick
	// out of the chunk on the left or right; its owner chunk (the anchor's) draws it, and
	// the neighbor blocks the tile it covers.
	for ly := 0; ly < ChunkSize; ly += 2 {
		shift := floorMod((y0+ly)/2, 2)
		for lx := shift - 2; lx < ChunkSize; lx += 2 {
			wx, wy := x0+lx, y0+ly
			if lx+1 < 0 || !wantTree(seed, wx, wy, water) || treeBlocked(c, lx, ly, keepClear, x0, y0) {
				continue
			}
			if lx >= 0 {
				c.Objects = append(c.Objects, Object{wx, wy, treeKind(seed, wx, wy, water)})
			}
			for dy := 0; dy < 2; dy++ {
				for dx := 0; dx < 2; dx++ {
					if lx+dx >= 0 && lx+dx < ChunkSize {
						c.Blocked[Index(lx+dx, ly+dy)] = true
					}
				}
			}
		}
	}

	for ly := 0; ly < ChunkSize; ly++ {
		for lx := 0; lx < ChunkSize; lx++ {
			wx, wy := x0+lx, y0+ly
			if occupied(lx, ly) || tiles.IsRoad(c.Ground[Index(lx, ly)]) {
				continue
			}
			r := rand01(seed, saltDecor, wx, wy)
			switch {
			case r < bushOdds && !nearCrossing(seed, wx, wy) && !keepClear(wx, wy):
				c.Objects = append(c.Objects, Object{wx, wy, bm.bush})
				c.Blocked[Index(lx, ly)] = true
			case r < bushOdds+bm.decorOdds:
				c.Objects = append(c.Objects, Object{wx, wy, pick(bm.decor, rand01(seed, saltDecorKind, wx, wy))})
			}
		}
	}
	return c
}

func floorMod(a, b int) int { return ((a % b) + b) % b }

func groundVariant(seed int64, bm *biome, x, y int) tiles.ID {
	r := rand01(seed, saltGrass, x, y)
	if r >= bm.variantOdds {
		return bm.ground
	}
	return pick(bm.variants, r/bm.variantOdds)
}

// pick chooses an element from a list with a value in [0, 1).
func pick(list []tiles.ID, r float64) tiles.ID {
	return list[min(int(r*float64(len(list))), len(list)-1)]
}

// treeBlocked reports whether a 2×2 tree at local (lx, ly) would cover a building or a tile
// that has to stay clear. Tiles outside the chunk are checked with keepClear only.
func treeBlocked(c *Chunk, lx, ly int, keepClear func(x, y int) bool, x0, y0 int) bool {
	for dy := 0; dy < 2; dy++ {
		for dx := 0; dx < 2; dx++ {
			x, y := lx+dx, ly+dy
			if keepClear(x0+x, y0+y) {
				return true
			}
			if x >= 0 && x < ChunkSize && c.Blocked[Index(x, y)] {
				return true
			}
		}
	}
	return false
}

// roadMask marks dirt (roads and plazas) on the area around a chunk, for autotiling.
func roadMask(seed int64, x0, y0 int) *mask {
	m := &mask{x0: x0 - 1, y0: y0 - 1, size: ChunkSize + 2}
	m.cells = make([]bool, m.size*m.size)
	for ly := 0; ly < m.size; ly++ {
		for lx := 0; lx < m.size; lx++ {
			x, y := m.x0+lx, m.y0+ly
			m.cells[ly*m.size+lx] = LayoutAt(seed, x, y).isRoad(x, y)
		}
	}
	return m
}

// wantTree decides whether a 2×2 tree (or boulder, or palm) grows with its top-left tile at
// (x, y). Zone barriers are solid except for the crossings. The tree's own zone decides, so
// both chunks agree on trees that reach over a zone border.
func wantTree(seed int64, x, y int, water *mask) bool {
	barrier, crossing := false, false
	for dy := 0; dy < 2; dy++ {
		for dx := 0; dx < 2; dx++ {
			barrier = barrier || inBarrier(seed, x+dx, y+dy)
			crossing = crossing || nearCrossing(seed, x+dx, y+dy)
		}
	}
	if crossing {
		return false
	}
	if barrier {
		return true
	}
	if water.near(x, y, 2, 2) {
		return false
	}
	bm := biomeAt(seed, x, y)
	if palmSpot(bm, x, y, water) && rand01(seed, saltTree+1, x, y) < 0.45 {
		return true
	}
	return treeAt(seed, bm, x, y)
}

// palmSpot reports whether palms can grow here: close to, but not right next to, water.
func palmSpot(bm *biome, x, y int, water *mask) bool {
	return len(bm.nearWater) > 0 && water.within(x, y, 2, 2, palmReach)
}

func treeAt(seed int64, bm *biome, x, y int) bool {
	forest := fbm(seed, saltForest, x, y, forestScale, 3)
	density := min((forest-bm.forestStart)*6, bm.maxDensity)
	density = max(density, loneTreeOdds)
	return rand01(seed, saltTree, x, y) < density
}

func treeKind(seed int64, x, y int, water *mask) tiles.ID {
	bm := biomeAt(seed, x, y)
	r := rand01(seed, saltTreeKind, x, y)
	for dy := 0; dy < 2; dy++ {
		for dx := 0; dx < 2; dx++ {
			if inBarrier(seed, x+dx, y+dy) {
				return pick(bm.barrier, r)
			}
		}
	}
	if palmSpot(bm, x, y, water) {
		return pick(bm.nearWater, r)
	}
	return pick(bm.trees, r)
}

// mask is a water mask over a padded area around a chunk.
type mask struct {
	x0, y0, size int // world coordinates of the padded area's top-left tile
	cells        []bool
}

func (m *mask) at(x, y int) bool {
	lx, ly := x-m.x0, y-m.y0
	if lx < 0 || ly < 0 || lx >= m.size || ly >= m.size {
		return false
	}
	return m.cells[ly*m.size+lx]
}

// near reports whether there is water within one tile around a w×h rectangle at (x, y).
func (m *mask) near(x, y, w, h int) bool {
	for yy := y - 1; yy <= y+h; yy++ {
		for xx := x - 1; xx <= x+w; xx++ {
			if m.at(xx, yy) {
				return true
			}
		}
	}
	return false
}

// within reports whether there is water within r tiles around a w×h rectangle at (x, y).
func (m *mask) within(x, y, w, h, r int) bool {
	for yy := y - r; yy < y+h+r; yy++ {
		for xx := x - r; xx < x+w+r; xx++ {
			if m.at(xx, yy) {
				return true
			}
		}
	}
	return false
}

// neighbors returns which of the 8 neighbors of a tile are land.
func (m *mask) neighbors(x, y int) landAround {
	return landAround{
		n: !m.at(x, y-1), s: !m.at(x, y+1), e: !m.at(x+1, y), w: !m.at(x-1, y),
		ne: !m.at(x+1, y-1), nw: !m.at(x-1, y-1), se: !m.at(x+1, y+1), sw: !m.at(x-1, y+1),
	}
}

func (m *mask) piece(x, y int) tiles.ID {
	if id, ok := m.neighbors(x, y).piece(); ok {
		return id
	}
	return m.neighbors(x, y).fallback()
}

func waterMask(seed int64, x0, y0 int) *mask {
	m := &mask{x0: x0 - waterPad, y0: y0 - waterPad, size: ChunkSize + 2*waterPad}
	m.cells = make([]bool, m.size*m.size)
	for ly := 0; ly < m.size; ly++ {
		for lx := 0; lx < m.size; lx++ {
			x, y := m.x0+lx, m.y0+ly
			l := LayoutAt(seed, x, y)
			m.cells[ly*m.size+lx] = !nearZoneEdge(x, y, lakeFreeMargin) &&
				l.hubDist(x, y) > hubLakeFree && l.roadDist(x, y) > roadLakeFree && !l.nearBuilding(x, y, 4) &&
				fbm(seed, saltLake, x, y, lakeScale, 4) > biomeAt(seed, x, y).lakeThreshold
		}
	}
	// Remove water tiles that no autotile piece can draw (thin strips, checkerboard corners).
	for pass := 0; pass < waterPasses; pass++ {
		next := make([]bool, len(m.cells))
		for ly := 0; ly < m.size; ly++ {
			for lx := 0; lx < m.size; lx++ {
				i := ly*m.size + lx
				if m.cells[i] {
					_, ok := m.neighbors(m.x0+lx, m.y0+ly).piece()
					next[i] = ok
				}
			}
		}
		m.cells = next
	}
	return m
}

type landAround struct{ n, s, e, w, ne, nw, se, sw bool }

// piece picks the autotile piece for a water tile. ok is false when no piece fits.
func (l landAround) piece() (tiles.ID, bool) {
	cardinals := 0
	for _, b := range []bool{l.n, l.s, l.e, l.w} {
		if b {
			cardinals++
		}
	}
	switch cardinals {
	case 0:
		diagonals := 0
		for _, b := range []bool{l.ne, l.nw, l.se, l.sw} {
			if b {
				diagonals++
			}
		}
		switch {
		case diagonals == 0:
			return tiles.Water, true
		case diagonals > 1:
			return tiles.None, false
		case l.ne:
			return tiles.WaterInnerNE, true
		case l.nw:
			return tiles.WaterInnerNW, true
		case l.se:
			return tiles.WaterInnerSE, true
		default:
			return tiles.WaterInnerSW, true
		}
	case 1:
		switch {
		case l.n && !l.se && !l.sw:
			return tiles.WaterN, true
		case l.s && !l.ne && !l.nw:
			return tiles.WaterS, true
		case l.e && !l.nw && !l.sw:
			return tiles.WaterE, true
		case l.w && !l.ne && !l.se:
			return tiles.WaterW, true
		}
	case 2:
		switch {
		case l.n && l.e && !l.sw:
			return tiles.WaterNE, true
		case l.n && l.w && !l.se:
			return tiles.WaterNW, true
		case l.s && l.e && !l.nw:
			return tiles.WaterSE, true
		case l.s && l.w && !l.ne:
			return tiles.WaterSW, true
		}
	}
	return tiles.None, false
}

// fallback picks the closest piece for configurations the cleanup didn't remove.
func (l landAround) fallback() tiles.ID {
	switch {
	case l.n && l.e:
		return tiles.WaterNE
	case l.n && l.w:
		return tiles.WaterNW
	case l.s && l.e:
		return tiles.WaterSE
	case l.s && l.w:
		return tiles.WaterSW
	case l.n:
		return tiles.WaterN
	case l.s:
		return tiles.WaterS
	case l.e:
		return tiles.WaterE
	case l.w:
		return tiles.WaterW
	}
	return tiles.Water
}
