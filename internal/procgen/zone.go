package procgen

// Zones are cells of a square grid (see design/world.md, Zone Grid & Size). Zone (0, 0) is
// centered on the world origin. Neighboring zones are separated by a barrier band of dense
// forest with a few narrow crossings.

// ZoneSize is the edge length of a zone cell in tiles (starting guess, tune by playtesting).
const ZoneSize = 512

const (
	zoneOffset      = ZoneSize / 2 // zone (0, 0) spans -256..255
	borderHalfWidth = 5            // average barrier width on each side of a zone edge
	borderWobble    = 3            // the barrier edge varies by up to this many tiles
	borderScale     = 20.0         // tiles per wobble bump along the edge
	lakeFreeMargin  = borderHalfWidth + borderWobble + 6
	crossingHalf    = 1  // crossings are 3 tiles wide
	crossingMargin  = 48 // crossings stay away from zone corners
)

const (
	saltBorder uint64 = iota + 100
	saltCrossingCount
	saltCrossingPos
)

// ZoneOf returns the zone cell containing a world tile.
func ZoneOf(x, y int) (zx, zy int) {
	return floorDiv(x+zoneOffset, ZoneSize), floorDiv(y+zoneOffset, ZoneSize)
}

// zoneLocal returns the position of a coordinate inside its zone cell, 0..ZoneSize-1.
func zoneLocal(v int) int { return floorMod(v+zoneOffset, ZoneSize) }

// edgeDistance is how many tiles a coordinate is away from the nearest zone edge on its axis
// (0 = right next to the edge).
func edgeDistance(v int) int {
	l := zoneLocal(v)
	return min(l, ZoneSize-1-l)
}

// nearestEdge returns the coordinate of the zone edge closest to v: the first tile of the
// zone on the far side of the edge.
func nearestEdge(v int) int {
	z := floorDiv(v+zoneOffset, ZoneSize)
	if zoneLocal(v) >= ZoneSize/2 {
		z++
	}
	return z*ZoneSize - zoneOffset
}

// inBarrier reports whether a tile lies in the barrier band between two zones.
func inBarrier(seed int64, x, y int) bool {
	if edgeDistance(x) < borderWidthAt(seed, nearestEdge(x), y, 0) && !inCrossing(seed, x, y, true) {
		return true
	}
	return edgeDistance(y) < borderWidthAt(seed, nearestEdge(y), x, 1) && !inCrossing(seed, x, y, false)
}

// borderWidthAt returns the barrier half-width at position `along` of the edge at `edge`.
// axis 0 = vertical edge (x = edge), 1 = horizontal edge (y = edge).
func borderWidthAt(seed int64, edge, along, axis int) int {
	n := fbm(seed, saltBorder+uint64(axis), edge, along, borderScale, 2)
	return borderHalfWidth + int((n-0.5)*2*borderWobble)
}

// inCrossing reports whether a tile is in the corridor of a crossing over the nearest
// vertical (vertical = true) or horizontal zone edge.
func inCrossing(seed int64, x, y int, vertical bool) bool {
	across, along := x, y
	if !vertical {
		across, along = y, x
	}
	edge := nearestEdge(across)
	// The segment of the edge between two zone corners, identified by its zone cell index.
	seg := floorDiv(along+zoneOffset, ZoneSize)
	axis := uint64(0)
	if !vertical {
		axis = 1
	}
	for _, c := range crossings(seed, axis, edge, seg) {
		if along >= c-crossingHalf && along <= c+crossingHalf {
			return true
		}
	}
	return false
}

// crossings returns the positions (along the edge) of the 1–3 crossings of an edge segment.
func crossings(seed int64, axis uint64, edge, seg int) []int {
	r := rand01(seed, saltCrossingCount+axis*7, edge, seg)
	n := 1
	switch {
	case r >= 0.8:
		n = 3
	case r >= 0.5:
		n = 2
	}
	start := seg*ZoneSize - zoneOffset + crossingMargin
	span := (ZoneSize - 2*crossingMargin) / n
	out := make([]int, n)
	for i := range out {
		// Each crossing sits somewhere in the middle 60% of its share of the edge.
		t := 0.2 + 0.6*rand01(seed, saltCrossingPos+axis*7+uint64(i), edge, seg)
		out[i] = start + i*span + int(t*float64(span))
	}
	return out
}

// nearZoneEdge reports whether a tile is within margin tiles of a zone edge on either axis.
func nearZoneEdge(x, y, margin int) bool {
	return edgeDistance(x) < margin || edgeDistance(y) < margin
}

// nearCrossing reports whether a tile is inside or right next to a crossing corridor.
func nearCrossing(seed int64, x, y int) bool {
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			if inCrossing(seed, x+dx, y+dy, true) && edgeDistance(x+dx) < lakeFreeMargin ||
				inCrossing(seed, x+dx, y+dy, false) && edgeDistance(y+dy) < lakeFreeMargin {
				return true
			}
		}
	}
	return false
}

func floorDiv(a, b int) int {
	q := a / b
	if a%b != 0 && (a < 0) != (b < 0) {
		q--
	}
	return q
}

// ZonesNear lists the zones (other than the tile's own) whose cell is within d tiles of a
// tile, for preparing them before the player arrives.
func ZonesNear(x, y, d int) [][2]int {
	zx, zy := ZoneOf(x, y)
	var out [][2]int
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			if dx == 0 && dy == 0 {
				continue
			}
			// Distance from the tile to the neighbor cell's rectangle.
			x0, y0 := (zx+dx)*ZoneSize-zoneOffset, (zy+dy)*ZoneSize-zoneOffset
			ddx := max(x0-x, 0, x-(x0+ZoneSize-1))
			ddy := max(y0-y, 0, y-(y0+ZoneSize-1))
			if max(ddx, ddy) <= d {
				out = append(out, [2]int{zx + dx, zy + dy})
			}
		}
	}
	return out
}
