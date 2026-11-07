package procgen

import (
	"math"

	"oldworld2/internal/creature"
	"oldworld2/internal/tiles"
)

// Spawn is a creature placed by world generation.
type Spawn struct {
	ID    uint64 // stable per world, used to remember defeated enemies
	Kind  creature.Kind
	X, Y  int // home tile
	Level int
}

const (
	spawnCell      = 8    // a chunk is split into cells; each cell can host one group
	spawnBaseOdds  = 0.16 // chance that a cell hosts a group, near the start
	spawnMaxOdds   = 0.32 // chance far away from the start
	spawnRampTiles = 300.0
	safeRadius     = 16 // no enemies this close to the start
	banditMinDist  = 60 // bandit camps stay away from the start
	banditRoadMax  = 24 // bandit camps are at most this far from a road
	roadQuiet      = 5  // no enemy groups this close to a road's center line
	guardianRange  = 30 // tomb guardians stay within this many tiles of a monument
	shrineSafe     = 12 // no enemy groups this close to a shrine
)

const (
	saltSpawnCell uint64 = iota + 200
	saltSpawnKind
	saltSpawnGroup
	saltSpawnID
)

// spawns places creature groups in a generated chunk. Placement follows habitats (see
// design/world.md, Enemy Placement) with seeded randomness on top.
func spawns(seed int64, c *Chunk, t *terrainCache) []Spawn {
	var out []Spawn
	x0, y0 := c.CX*ChunkSize, c.CY*ChunkSize
	taken := map[[2]int]bool{}
	for cy := 0; cy < ChunkSize; cy += spawnCell {
		for cx := 0; cx < ChunkSize; cx += spawnCell {
			wx, wy := x0+cx+spawnCell/2, y0+cy+spawnCell/2
			dist := math.Hypot(float64(wx), float64(wy))
			layout := LayoutAt(seed, wx, wy)
			roadDist := layout.roadDist(wx, wy)
			if dist < safeRadius || nearZoneEdge(wx, wy, lakeFreeMargin) ||
				layout.hubDist(wx, wy) < hubSafeRadius || roadDist < roadQuiet || layout.nearBuilding(wx, wy, shrineSafe) {
				continue
			}
			odds := spawnBaseOdds + (spawnMaxOdds-spawnBaseOdds)*min(1, dist/spawnRampTiles)
			if rand01(seed, saltSpawnCell, wx, wy) >= odds {
				continue
			}
			kind, ok := pickKind(seed, c, wx, wy, dist, roadDist)
			if !ok {
				continue
			}
			info := creature.Get(kind)
			level := enemyLevel(info.Tier, layout.hubDist(wx, wy), layout.Difficulty)
			n := info.GroupMin + int(rand01(seed, saltSpawnGroup, wx, wy)*float64(info.GroupMax-info.GroupMin+1))
			members := freeTilesNear(c, wx-x0, wy-y0, n, taken)
			for _, p := range members {
				// Enemies must not be trapped in holes enclosed by trees, bushes and water.
				if !t.Main(x0+p[0], y0+p[1]) {
					continue
				}
				out = append(out, Spawn{
					ID:   hash(seed, saltSpawnID, x0+p[0], y0+p[1]),
					Kind: kind,
					X:    x0 + p[0], Y: y0 + p[1],
					Level: level,
				})
			}
		}
	}
	return out
}

// enemyLevel picks a level inside the zone's range (difficulty d → levels 5d−4 to 5d, see
// design/world.md): trash at the low end, elites at the high end, rising with the distance
// from the settlement.
func enemyLevel(tier string, hubDist float64, difficulty int) int {
	base := 5 * (difficulty - 1)
	if tier == "elite" {
		return base + min(5, 3+int(hubDist/150))
	}
	return base + min(3, 1+int(hubDist/80))
}

// pickKind chooses a creature for a spawn cell from the habitat around it.
func pickKind(seed int64, c *Chunk, wx, wy int, dist, roadDist float64) (creature.Kind, bool) {
	forest := fbm(seed, saltForest, wx, wy, forestScale, 3)
	waterNearby := false
	lx, ly := wx-c.CX*ChunkSize, wy-c.CY*ChunkSize
	for dy := -spawnCell / 2; dy < spawnCell/2 && !waterNearby; dy++ {
		for dx := -spawnCell / 2; dx < spawnCell/2; dx++ {
			x, y := lx+dx, ly+dy
			if x >= 0 && y >= 0 && x < ChunkSize && y < ChunkSize && tiles.IsWater(c.Ground[Index(x, y)]) {
				waterNearby = true
				break
			}
		}
	}

	var opts []spawnOption
	switch ZoneTypeAt(seed, wx, wy) {
	case AncientDesert:
		bm := biomes[AncientDesert]
		switch {
		case forest > bm.forestStart-0.04: // boulder fields
			opts = append(opts, spawnOption{creature.Scarab, 2}, spawnOption{creature.Cobra, 1})
		default:
			opts = append(opts, spawnOption{creature.Cobra, 1.5}, spawnOption{creature.Scarab, 1.5}, spawnOption{creature.Mummy, 1.2})
		}
		// Tomb guardians watch over the monuments.
		if LayoutAt(seed, wx, wy).nearMonument(wx, wy, guardianRange) {
			opts = append(opts, spawnOption{creature.TombGuardian, 3}, spawnOption{creature.Mummy, 1.5})
		}
		return pickOption(seed, wx, wy, opts)
	case North:
		bm := biomes[North]
		switch {
		case forest > bm.forestStart-0.04 && forest < bm.forestStart+0.1: // forest edge
			opts = append(opts, spawnOption{creature.SnowWolf, 2}, spawnOption{creature.IceTroll, 0.8})
		case forest >= bm.forestStart+0.1: // deep in the pines
			opts = append(opts, spawnOption{creature.IceTroll, 1.5}, spawnOption{creature.SnowWolf, 1})
		default: // open snow
			opts = append(opts, spawnOption{creature.SnowWolf, 0.6}, spawnOption{creature.FrostWisp, 1.5})
		}
		if waterNearby { // wisps drift over the frozen lakes
			opts = append(opts, spawnOption{creature.FrostWisp, 2})
		}
		if dist > banditMinDist && roadDist < banditRoadMax && forest < bm.forestStart {
			opts = append(opts, spawnOption{creature.Berserker, 1.5})
		}
		return pickOption(seed, wx, wy, opts)
	case Mediterranean:
		bm := biomes[Mediterranean]
		switch {
		case forest > bm.forestStart-0.04: // olive groves and cypress stands
			opts = append(opts, spawnOption{creature.Boar, 1.5}, spawnOption{creature.WindSpirit, 0.5})
		default: // open, sunny hills
			opts = append(opts, spawnOption{creature.WindSpirit, 1.5}, spawnOption{creature.Boar, 0.8})
		}
		if waterNearby { // crabs on the shores
			opts = append(opts, spawnOption{creature.GiantCrab, 3})
		}
		if dist > banditMinDist && roadDist < banditRoadMax && forest < bm.forestStart {
			corsairs := 1.0
			if waterNearby {
				corsairs = 2.5
			}
			opts = append(opts, spawnOption{creature.Corsair, corsairs}, spawnOption{creature.Hoplite, 0.6 + min(1.2, dist/400)})
		}
		return pickOption(seed, wx, wy, opts)
	case Unholy:
		bm := biomes[Unholy]
		switch {
		case forest > bm.forestStart-0.04: // the dead woods
			opts = append(opts, spawnOption{creature.DireWolf, 2}, spawnOption{creature.BatSwarm, 1})
		default: // open moors and graveyards
			opts = append(opts, spawnOption{creature.BatSwarm, 1.5}, spawnOption{creature.DireWolf, 0.8})
		}
		if dist > banditMinDist && roadDist < banditRoadMax && forest < bm.forestStart {
			opts = append(opts, spawnOption{creature.Cultist, 1.4})
		}
		if dist > banditMinDist*2 && forest < bm.forestStart { // vampires keep to themselves, far from the settlement
			opts = append(opts, spawnOption{creature.Vampire, 0.06 + min(0.2, dist/1500)})
		}
		return pickOption(seed, wx, wy, opts)
	case Desert:
		bm := biomes[Desert]
		switch {
		case forest > bm.forestStart-0.04: // boulder fields
			opts = append(opts, spawnOption{creature.Scorpion, 3}, spawnOption{creature.SandSnake, 1})
		default: // open sand
			opts = append(opts, spawnOption{creature.SandSnake, 2}, spawnOption{creature.Scorpion, 0.5}, spawnOption{creature.EmberSpirit, 1.3})
		}
		if roadDist < banditRoadMax {
			opts = append(opts, spawnOption{creature.NomadRaider, 0.5})
		}
		return pickOption(seed, wx, wy, opts)
	}
	switch {
	case forest > forestStart-0.04 && forest < forestStart+0.1: // forest edge
		opts = append(opts, spawnOption{creature.Wolf, 3}, spawnOption{creature.Boar, 0.5})
	case forest < forestStart-0.06: // open meadow
		opts = append(opts, spawnOption{creature.Boar, 2}, spawnOption{creature.Wolf, 0.3})
	default:
		opts = append(opts, spawnOption{creature.Wolf, 1}, spawnOption{creature.Boar, 1})
	}
	if waterNearby { // goblins camp by ponds and streams
		opts = append(opts, spawnOption{creature.Goblin, 2.5})
	}
	// Bandit camps sit near roads, away from the start (design/world.md, Enemy Placement).
	if dist > banditMinDist && roadDist < banditRoadMax && forest < forestStart {
		opts = append(opts, spawnOption{creature.Bandit, 2})
	}

	return pickOption(seed, wx, wy, opts)
}

// spawnOption is a creature with a weight for the habitat roll.
type spawnOption struct {
	kind   creature.Kind
	weight float64
}

// pickOption rolls a weighted choice between creatures for a spawn cell.
func pickOption(seed int64, wx, wy int, opts []spawnOption) (creature.Kind, bool) {
	total := 0.0
	for _, o := range opts {
		total += o.weight
	}
	r := rand01(seed, saltSpawnKind, wx, wy) * total
	for _, o := range opts {
		if r < o.weight {
			return o.kind, true
		}
		r -= o.weight
	}
	return 0, false
}

// freeTilesNear finds up to n walkable, untaken tiles close to a local position, spiraling out.
func freeTilesNear(c *Chunk, lx, ly, n int, taken map[[2]int]bool) [][2]int {
	var out [][2]int
	for r := 0; r <= 3 && len(out) < n; r++ {
		for dy := -r; dy <= r && len(out) < n; dy++ {
			for dx := -r; dx <= r && len(out) < n; dx++ {
				if max(abs(dx), abs(dy)) != r {
					continue
				}
				x, y := lx+dx, ly+dy
				if x < 0 || y < 0 || x >= ChunkSize || y >= ChunkSize || c.Blocked[Index(x, y)] {
					continue
				}
				// Keep one tile between group members so they don't stand in a clump.
				if taken[[2]int{x, y}] || taken[[2]int{x - 1, y}] || taken[[2]int{x, y - 1}] {
					continue
				}
				taken[[2]int{x, y}] = true
				out = append(out, [2]int{x, y})
			}
		}
	}
	return out
}

func abs(a int) int {
	if a < 0 {
		return -a
	}
	return a
}
