// Package world holds the generated chunks around the player and answers tile queries.
package world

import (
	"oldworld2/internal/procgen"
	"oldworld2/internal/tiles"
)

// World caches generated chunks and the enemies living in them. Chunks are generated on
// first access.
type World struct {
	Seed     int64
	Home     [2]int // the zone the game started in: difficulty 1, rising outward (set before chunks are generated)
	terrain  *procgen.TerrainCache
	chunks   map[[2]int]*procgen.Chunk
	npcs     map[uint64]*NPC
	defeated map[uint64]bool
}

// New creates an empty world for a seed.
func New(seed int64) *World {
	return &World{
		Seed:     seed,
		terrain:  procgen.NewTerrainCache(seed),
		chunks:   make(map[[2]int]*procgen.Chunk),
		npcs:     make(map[uint64]*NPC),
		defeated: make(map[uint64]bool),
	}
}

// Difficulty is a zone's difficulty: 1 in the zone the game started in, rising outward.
func (w *World) Difficulty(zx, zy int) int {
	return procgen.DifficultyFrom(w.Home[0], w.Home[1], zx, zy)
}

// ChunkCoord returns the chunk that contains a world tile.
func ChunkCoord(tx, ty int) (cx, cy int) {
	return floorDiv(tx, procgen.ChunkSize), floorDiv(ty, procgen.ChunkSize)
}

func floorDiv(a, b int) int {
	q := a / b
	if a%b != 0 && (a < 0) != (b < 0) {
		q--
	}
	return q
}

// Chunk returns the chunk at chunk coordinates, generating it if needed.
func (w *World) Chunk(cx, cy int) *procgen.Chunk {
	key := [2]int{cx, cy}
	c, ok := w.chunks[key]
	if !ok {
		c = w.terrain.Generate(cx, cy)
		w.chunks[key] = c
		for _, s := range c.Spawns {
			if _, alive := w.npcs[s.ID]; !alive && !w.defeated[s.ID] {
				// Spawns are leveled for a game started in the start zone; a game started
				// elsewhere shifts them by 5 levels per difficulty step (design/world.md).
				zx, zy := procgen.ZoneOf(s.X, s.Y)
				s.Level = max(1, s.Level+5*(w.Difficulty(zx, zy)-procgen.Difficulty(zx, zy)))
				w.npcs[s.ID] = newNPC(s)
			}
		}
	}
	return c
}

// Blocked reports whether a world tile can't be walked on.
func (w *World) Blocked(tx, ty int) bool {
	cx, cy := ChunkCoord(tx, ty)
	c := w.Chunk(cx, cy)
	return c.Blocked[procgen.Index(tx-cx*procgen.ChunkSize, ty-cy*procgen.ChunkSize)]
}

// Road reports whether a world tile is part of a road.
func (w *World) Road(tx, ty int) bool {
	cx, cy := ChunkCoord(tx, ty)
	c := w.Chunk(cx, cy)
	return tiles.IsRoad(c.Ground[procgen.Index(tx-cx*procgen.ChunkSize, ty-cy*procgen.ChunkSize)])
}

// terrainKeepRadius is how many chunks of terrain are kept around the player.
const terrainKeepRadius = 24

// prefetchDistance: neighboring zones are prepared in the background once the player is this
// many tiles from them.
const prefetchDistance = 160

// Prepare readies the zone around a tile now (during loading), so the first frames don't
// stall on it.
func (w *World) Prepare(tx, ty int) { w.terrain.Main(tx, ty) }

// Prefetch starts preparing the zones the player is approaching.
func (w *World) Prefetch(tx, ty int) {
	for _, z := range procgen.ZonesNear(tx, ty, prefetchDistance) {
		w.terrain.Prefetch(z[0], z[1])
	}
}

// BlockedForPlayer also treats enemies as obstacles.
func (w *World) BlockedForPlayer(tx, ty int) bool {
	return w.Blocked(tx, ty) || w.npcAt(tx, ty) != nil
}

// Unload drops chunks farther than radius chunks from the given chunk, and the enemies
// standing in them, to bound memory. Enemies come back at home when the chunk reloads.
func (w *World) Unload(cx, cy, radius int) {
	far := func(x, y int) bool { return abs(x-cx) > radius || abs(y-cy) > radius }
	for key := range w.chunks {
		if far(key[0], key[1]) {
			delete(w.chunks, key)
		}
	}
	// Terrain is kept much longer than chunks: zones prepared in the background generate it
	// ahead of the player, and it is cheap to keep (about 6 KB per chunk).
	w.terrain.Drop(cx, cy, terrainKeepRadius)
	for id, n := range w.npcs {
		hx, hy := ChunkCoord(n.Spawn.X, n.Spawn.Y)
		if nx, ny := ChunkCoord(n.TX, n.TY); far(hx, hy) || far(nx, ny) {
			delete(w.npcs, id)
		}
	}
}

// CheckpointAt reports whether a tile is a checkpoint (inn door or shrine) and which kind.
func (w *World) CheckpointAt(tx, ty int) int { return procgen.CheckpointAt(w.Seed, tx, ty) }

func abs(a int) int {
	if a < 0 {
		return -a
	}
	return a
}

// FindSpawn returns the start point: in front of the start zone's inn, or else the walkable
// tile closest to the origin that has room to move around.
func (w *World) FindSpawn() (tx, ty int) {
	// Checking the main area also prepares the start zone now, during loading, instead of
	// on the first frame.
	if x, y, ok := procgen.StartPoint(w.Seed); ok && !w.Blocked(x, y) && w.terrain.Main(x, y) {
		return x, y
	}
	for r := 0; ; r++ {
		for y := -r; y <= r; y++ {
			for x := -r; x <= r; x++ {
				if max(abs(x), abs(y)) != r {
					continue
				}
				if w.open(x, y) && w.terrain.Main(x, y) {
					return x, y
				}
			}
		}
	}
}

// FindSpawnIn returns a start point in a zone: in front of its inn, or else the walkable
// tile closest to its settlement that has room to move around.
func (w *World) FindSpawnIn(zx, zy int) (tx, ty int) {
	if zx == 0 && zy == 0 {
		return w.FindSpawn()
	}
	if x, y, ok := procgen.InnPoint(w.Seed, zx, zy); ok && !w.Blocked(x, y) && w.terrain.Main(x, y) {
		return x, y
	}
	l := procgen.Layout(w.Seed, zx, zy)
	for r := 0; ; r++ {
		for y := -r; y <= r; y++ {
			for x := -r; x <= r; x++ {
				if max(abs(x), abs(y)) == r && w.open(l.HubX+x, l.HubY+y) && w.terrain.Main(l.HubX+x, l.HubY+y) {
					return l.HubX + x, l.HubY + y
				}
			}
		}
	}
}

// open reports whether a tile and its 8 neighbors are walkable.
func (w *World) open(tx, ty int) bool {
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			if w.Blocked(tx+dx, ty+dy) {
				return false
			}
		}
	}
	return true
}
