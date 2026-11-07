package procgen

import (
	"runtime"
	"sync"
)

// Reachability: forests, bushes and water can enclose areas of any size. When a zone is
// first needed, its whole terrain is generated and the largest connected walkable area is
// marked as the zone's main area. Enemies and the player only spawn there, so nothing is
// trapped in a hole the player can't reach.

// TerrainCache generates chunk terrain (without spawns) on demand and knows each zone's main
// area. The world keeps one so terrain is generated only once.
type TerrainCache struct {
	seed   int64
	mu     sync.Mutex
	chunks map[[2]int]*Chunk
	zones  map[[2]int]*zoneJob
}

// zoneJob computes a zone's main area once; others wait for it instead of repeating it.
type zoneJob struct {
	done chan struct{}
	area *zoneArea
}

// Background zone preparation uses few workers so the game loop keeps its CPU time.
const backgroundWorkers = 2

type terrainCache = TerrainCache

// NewTerrainCache creates an empty terrain cache for a seed.
func NewTerrainCache(seed int64) *TerrainCache {
	return &TerrainCache{seed: seed, chunks: map[[2]int]*Chunk{}, zones: map[[2]int]*zoneJob{}}
}

// Drop forgets chunks farther than radius chunks from (cx, cy). Zone areas are small and
// kept.
func (t *TerrainCache) Drop(cx, cy, radius int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	for key := range t.chunks {
		if abs(key[0]-cx) > radius || abs(key[1]-cy) > radius {
			delete(t.chunks, key)
		}
	}
}

func (t *terrainCache) chunk(cx, cy int) *Chunk {
	key := [2]int{cx, cy}
	t.mu.Lock()
	c, ok := t.chunks[key]
	t.mu.Unlock()
	if !ok {
		c = generateTerrain(t.seed, cx, cy)
		t.mu.Lock()
		t.chunks[key] = c
		t.mu.Unlock()
	}
	return c
}

func (t *terrainCache) blocked(x, y int) bool {
	cx, cy := floorDiv(x, ChunkSize), floorDiv(y, ChunkSize)
	return t.chunk(cx, cy).Blocked[Index(x-cx*ChunkSize, y-cy*ChunkSize)]
}

// zoneArea marks the main walkable area of one zone cell.
type zoneArea struct {
	x0, y0 int // world tile of the zone's top-left corner
	main   []bool
}

// Main reports whether a tile belongs to its zone's main walkable area.
func (t *TerrainCache) Main(x, y int) bool {
	zx, zy := ZoneOf(x, y)
	z := t.zone(zx, zy)
	return z.main[(y-z.y0)*ZoneSize+(x-z.x0)]
}

func (t *TerrainCache) zone(zx, zy int) *zoneArea {
	job, start := t.job(zx, zy)
	if start {
		t.run(job, zx, zy, runtime.GOMAXPROCS(0))
	}
	<-job.done
	return job.area
}

// Prefetch prepares a zone in the background, so entering it later doesn't stall the game.
func (t *TerrainCache) Prefetch(zx, zy int) {
	if job, start := t.job(zx, zy); start {
		go t.run(job, zx, zy, backgroundWorkers)
	}
}

// job returns the zone's job and whether the caller has to start it.
func (t *TerrainCache) job(zx, zy int) (*zoneJob, bool) {
	key := [2]int{zx, zy}
	t.mu.Lock()
	defer t.mu.Unlock()
	if job, ok := t.zones[key]; ok {
		return job, false
	}
	job := &zoneJob{done: make(chan struct{})}
	t.zones[key] = job
	return job, true
}

func (t *TerrainCache) run(job *zoneJob, zx, zy, workers int) {
	job.area = t.computeZone(zx, zy, workers)
	close(job.done)
}

func (t *terrainCache) computeZone(zx, zy, workers int) *zoneArea {
	z := &zoneArea{x0: zx*ZoneSize - zoneOffset, y0: zy*ZoneSize - zoneOffset}
	const n = ZoneSize / ChunkSize
	cx0, cy0 := floorDiv(z.x0, ChunkSize), floorDiv(z.y0, ChunkSize)

	// Generate the zone's terrain in parallel; chunks are independent.
	blocked := make([]bool, ZoneSize*ZoneSize)
	jobs := make(chan [2]int)
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				c := t.chunk(cx0+j[0], cy0+j[1])
				for ly := 0; ly < ChunkSize; ly++ {
					row := (j[1]*ChunkSize + ly) * ZoneSize
					for lx := 0; lx < ChunkSize; lx++ {
						blocked[row+j[0]*ChunkSize+lx] = c.Blocked[Index(lx, ly)]
					}
				}
			}
		}()
	}
	for j := 0; j < n*n; j++ {
		jobs <- [2]int{j % n, j / n}
	}
	close(jobs)
	wg.Wait()

	// Label connected areas and keep the largest.
	label := make([]int32, ZoneSize*ZoneSize)
	best, bestSize := int32(0), 0
	var queue []int
	next := int32(0)
	for start := range blocked {
		if blocked[start] || label[start] != 0 {
			continue
		}
		next++
		label[start] = next
		queue = append(queue[:0], start)
		size := 0
		for len(queue) > 0 {
			i := queue[len(queue)-1]
			queue = queue[:len(queue)-1]
			size++
			x, y := i%ZoneSize, i/ZoneSize
			for _, nb := range [4][2]int{{x + 1, y}, {x - 1, y}, {x, y + 1}, {x, y - 1}} {
				if nb[0] < 0 || nb[1] < 0 || nb[0] >= ZoneSize || nb[1] >= ZoneSize {
					continue
				}
				j := nb[1]*ZoneSize + nb[0]
				if !blocked[j] && label[j] == 0 {
					label[j] = next
					queue = append(queue, j)
				}
			}
		}
		if size > bestSize {
			best, bestSize = next, size
		}
	}
	z.main = make([]bool, ZoneSize*ZoneSize)
	for i, l := range label {
		z.main[i] = l == best
	}
	return z
}
