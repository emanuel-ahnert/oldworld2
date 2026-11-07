package procgen

import (
	"math"
	"sort"
	"sync"

	"oldworld2/internal/tiles"
)

// Zone layout: every zone has a settlement (the hub) and dirt roads from each crossing to
// it (see design/world.md, Roads and Settlements). The layout is computed once per zone and
// queried per tile during chunk generation.

const (
	hubJitter      = 128 // settlements sit within ±64 tiles of the zone center (start zone: origin)
	plazaRadius    = 5   // dirt plaza in the middle of a settlement
	clearRadius    = 24  // no trees or bushes this close to a hub
	hubLakeFree    = 28  // no lakes this close to a hub
	hubSafeRadius  = 40  // no enemies this close to a hub
	roadHalfWidth  = 1.0 // roads are about 3 tiles wide
	roadClearance  = 2.0 // no trees or bushes this close to a road's center line
	roadLakeFree   = 4.0 // no lakes this close to a road's center line
	roadEntry      = 16  // roads leave a crossing straight for this many tiles
	roadEdgeMargin = 24  // winding roads stay this far from zone edges
	roadDepth      = 4   // midpoint displacement steps (2^depth segments per road)
	roadWobble     = 0.3 // how far roads wind, relative to the segment length
	shrineMinRoad  = 120 // roads at least this long get a shrine about halfway
	shrineMinHub   = 50  // shrines stay this far from the settlement
)

const (
	saltHub uint64 = iota + 300
	saltRoad
	saltHouse
	saltShrine
	saltMonument
)

type point struct{ x, y float64 }

type segment struct {
	a, b                   point
	minX, minY, maxX, maxY float64
}

// Building is a house or the inn, anchored at its top-left tile.
type Building struct {
	X, Y   int
	Sprite tiles.ID
}

// Door returns the door tile: the middle of the bottom row.
func (b Building) Door() (x, y int) {
	s := tiles.Get(b.Sprite)
	return b.X + s.W/2, b.Y + s.H - 1
}

func (b Building) covers(x, y int) bool {
	s := tiles.Get(b.Sprite)
	return x >= b.X && y >= b.Y && x < b.X+s.W && y < b.Y+s.H
}

// ZoneLayout is the road network, settlement and shrines of one zone.
type ZoneLayout struct {
	Name       string // generated, shown when entering the zone
	Type       ZoneType
	Difficulty int
	HubX, HubY int
	Buildings  []Building // houses, the inn and shrines
	segments   []segment
	roads      [][]point // each road from its crossing to the hub
}

var layouts sync.Map // layoutKey → *ZoneLayout

type layoutKey struct {
	seed   int64
	zx, zy int
}

// Layout returns the layout of a zone.
func Layout(seed int64, zx, zy int) *ZoneLayout {
	key := layoutKey{seed, zx, zy}
	if l, ok := layouts.Load(key); ok {
		return l.(*ZoneLayout)
	}
	l := buildLayout(seed, zx, zy)
	layouts.Store(key, l)
	return l
}

// LayoutAt returns the layout of the zone containing a tile.
func LayoutAt(seed int64, x, y int) *ZoneLayout {
	zx, zy := ZoneOf(x, y)
	return Layout(seed, zx, zy)
}

func buildLayout(seed int64, zx, zy int) *ZoneLayout {
	l := &ZoneLayout{Type: ZoneTypeOf(seed, zx, zy), Difficulty: Difficulty(zx, zy)}
	x0, y0 := zx*ZoneSize-zoneOffset, zy*ZoneSize-zoneOffset
	if zx == 0 && zy == 0 {
		l.HubX, l.HubY = 0, 0
	} else {
		l.HubX = x0 + ZoneSize/2 + int((rand01(seed, saltHub, zx, zy)-0.5)*hubJitter)
		l.HubY = y0 + ZoneSize/2 + int((rand01(seed, saltHub+1, zx, zy)-0.5)*hubJitter)
	}
	hub := point{float64(l.HubX), float64(l.HubY)}

	// One road from each crossing on the zone's four edges.
	left, right := x0, x0+ZoneSize // first tile of this zone, first tile of the next
	top, bottom := y0, y0+ZoneSize
	for _, c := range crossings(seed, 0, left, zy) {
		l.addRoad(seed, point{float64(left), float64(c)}, point{float64(left + roadEntry), float64(c)}, hub, x0, y0)
	}
	for _, c := range crossings(seed, 0, right, zy) {
		l.addRoad(seed, point{float64(right - 1), float64(c)}, point{float64(right - 1 - roadEntry), float64(c)}, hub, x0, y0)
	}
	for _, c := range crossings(seed, 1, top, zx) {
		l.addRoad(seed, point{float64(c), float64(top)}, point{float64(c), float64(top + roadEntry)}, hub, x0, y0)
	}
	for _, c := range crossings(seed, 1, bottom, zx) {
		l.addRoad(seed, point{float64(c), float64(bottom - 1)}, point{float64(c), float64(bottom - 1 - roadEntry)}, hub, x0, y0)
	}
	l.placeBuildings(seed)
	l.placeShrines(seed)
	if l.Type == AncientDesert {
		l.placeMonuments(seed)
	}
	l.Name = ZoneName(seed, zx, zy, l.Type)
	return l
}

// Monuments of the ancient desert.
const (
	pyramidGroups  = 2
	pyramidMinDist = 60  // pyramid groups stand this far from the settlement at least…
	pyramidMaxDist = 200 // …and at most this far
)

// placeMonuments puts obelisks and a sphinx by the settlement and a few pyramid groups out
// in the desert.
func (l *ZoneLayout) placeMonuments(seed int64) {
	// Two obelisks and a sphinx near the plaza.
	for i, kind := range []tiles.ID{tiles.Obelisk, tiles.Obelisk, tiles.Sphinx} {
		s := tiles.Get(kind)
		for _, p := range l.spotsAround(seed, i, plazaRadius+3, clearRadius-2, s.W, s.H) {
			if b := (Building{p[0], p[1], kind}); l.free(b, 1) {
				l.Buildings = append(l.Buildings, b)
				break
			}
		}
	}
	// Pyramid groups: a big pyramid with one or two small ones beside it.
	for g := 0; g < pyramidGroups; g++ {
		big := tiles.Get(tiles.Pyramid)
		for _, p := range l.spotsAround(seed, 10+g, pyramidMinDist, pyramidMaxDist, big.W+10, big.H+2) {
			group := []Building{
				{p[0] + 4, p[1], tiles.Pyramid},
				{p[0], p[1] + 2, tiles.PyramidSmall},
				{p[0] + 11, p[1] + 2, tiles.PyramidSmall},
				{p[0] + 6, p[1] + big.H + 2, tiles.PharaohStatue},
			}
			ok := true
			for _, b := range group {
				ok = ok && l.free(b, 2) && l.insideZone(b, roadEdgeMargin)
			}
			if ok {
				l.Buildings = append(l.Buildings, group...)
				break
			}
		}
	}
}

// spotsAround lists candidate top-left positions for a w×h object between rMin and rMax
// tiles from the hub, in a seeded order.
func (l *ZoneLayout) spotsAround(seed int64, salt, rMin, rMax, w, h int) [][2]int {
	var out [][2]int
	for i := 0; i < 64; i++ {
		a := rand01(seed, saltMonument+uint64(salt), l.HubX+i, l.HubY) * 2 * math.Pi
		r := float64(rMin) + rand01(seed, saltMonument+uint64(salt)+50, l.HubX, l.HubY+i)*float64(rMax-rMin)
		out = append(out, [2]int{l.HubX + int(math.Cos(a)*r) - w/2, l.HubY + int(math.Sin(a)*r) - h/2})
	}
	return out
}

// insideZone reports whether a building keeps margin tiles from the zone's edges.
func (l *ZoneLayout) insideZone(b Building, margin int) bool {
	s := tiles.Get(b.Sprite)
	zx, zy := ZoneOf(l.HubX, l.HubY)
	x0, y0 := zx*ZoneSize-zoneOffset, zy*ZoneSize-zoneOffset
	return b.X >= x0+margin && b.Y >= y0+margin && b.X+s.W <= x0+ZoneSize-margin && b.Y+s.H <= y0+ZoneSize-margin
}

// nearMonument reports whether a tile is within d tiles of a pyramid.
func (l *ZoneLayout) nearMonument(x, y, d int) bool {
	for _, b := range l.Buildings {
		if b.Sprite != tiles.Pyramid {
			continue
		}
		s := tiles.Get(b.Sprite)
		if x >= b.X-d && y >= b.Y-d && x < b.X+s.W+d && y < b.Y+s.H+d {
			return true
		}
	}
	return false
}

// placeShrines puts a checkpoint shrine beside every long road, about halfway between its
// crossing and the settlement.
func (l *ZoneLayout) placeShrines(seed int64) {
	for i, road := range l.roads {
		total := 0.0
		for j := 1; j < len(road); j++ {
			total += math.Hypot(road[j].x-road[j-1].x, road[j].y-road[j-1].y)
		}
		if total < shrineMinRoad {
			continue
		}
		// Walk to the middle of the road.
		at, dir := road[0], point{}
		left := total / 2
		for j := 1; j < len(road); j++ {
			a, b := road[j-1], road[j]
			seg := math.Hypot(b.x-a.x, b.y-a.y)
			if seg >= left && seg > 0 {
				at = point{a.x + (b.x-a.x)*left/seg, a.y + (b.y-a.y)*left/seg}
				dir = point{(b.x - a.x) / seg, (b.y - a.y) / seg}
				break
			}
			left -= seg
		}
		// Beside the road: try both sides, a few distances out.
		side := 1.0
		if rand01(seed, saltShrine, int(at.x), int(at.y)+i) < 0.5 {
			side = -1
		}
		for _, d := range []float64{4, 5, 6, -4, -5, -6} {
			off := d * side
			x := int(math.Round(at.x - dir.y*off - 0.5))
			y := int(math.Round(at.y + dir.x*off - 0.5))
			b := Building{x, y, tiles.Shrine}
			if l.hubDist(x, y) >= shrineMinHub && l.free(b, 1) {
				l.Buildings = append(l.Buildings, b)
				break
			}
		}
	}
}

// free reports whether a building fits with gap tiles of room: off the roads and the
// plaza, and not touching other buildings.
func (l *ZoneLayout) free(b Building, gap int) bool {
	s := tiles.Get(b.Sprite)
	for y := b.Y - gap; y < b.Y+s.H+gap; y++ {
		for x := b.X - gap; x < b.X+s.W+gap; x++ {
			if l.roadDist(x, y) <= roadClearance+0.5 || l.hubDist(x, y) <= plazaRadius+1 {
				return false
			}
			if _, ok := l.building(x, y); ok {
				return false
			}
		}
	}
	return true
}

// nearBuilding reports whether a tile is within d tiles of a building, so trees, bushes and
// lakes keep their distance.
func (l *ZoneLayout) nearBuilding(x, y, d int) bool {
	for _, b := range l.Buildings {
		s := tiles.Get(b.Sprite)
		if x >= b.X-d && y >= b.Y-d && x < b.X+s.W+d && y < b.Y+s.H+d {
			return true
		}
	}
	return false
}

// addRoad lays a road from a crossing: straight in from the edge, then winding to the hub.
func (l *ZoneLayout) addRoad(seed int64, edge, entry, hub point, x0, y0 int) {
	pts := []point{edge, entry}
	pts = append(pts, displace(seed, entry, hub, roadDepth, x0, y0)...)
	l.roads = append(l.roads, pts)
	for i := 1; i < len(pts); i++ {
		a, b := pts[i-1], pts[i]
		l.segments = append(l.segments, segment{a, b,
			math.Min(a.x, b.x), math.Min(a.y, b.y), math.Max(a.x, b.x), math.Max(a.y, b.y)})
	}
}

// displace returns the points after a, up to and including b, of a midpoint-displaced curve.
func displace(seed int64, a, b point, depth, x0, y0 int) []point {
	if depth == 0 {
		return []point{b}
	}
	dx, dy := b.x-a.x, b.y-a.y
	length := math.Hypot(dx, dy)
	r := rand01(seed, saltRoad, int(a.x)*31+int(b.x), int(a.y)*31+int(b.y)) - 0.5
	m := point{(a.x + b.x) / 2, (a.y + b.y) / 2}
	if length > 0 {
		m.x += -dy / length * r * length * roadWobble
		m.y += dx / length * r * length * roadWobble
	}
	// Keep winding roads out of the barrier bands.
	m.x = math.Max(float64(x0+roadEdgeMargin), math.Min(float64(x0+ZoneSize-roadEdgeMargin), m.x))
	m.y = math.Max(float64(y0+roadEdgeMargin), math.Min(float64(y0+ZoneSize-roadEdgeMargin), m.y))
	m.x, m.y = math.Round(m.x), math.Round(m.y)
	return append(displace(seed, a, m, depth-1, x0, y0), displace(seed, m, b, depth-1, x0, y0)...)
}

// placeBuildings puts the inn and a few houses around the plaza, avoiding the roads. It tries
// every position near the hub, closest first, so it works whatever way the roads run.
func (l *ZoneLayout) placeBuildings(seed int64) {
	hx, hy := l.HubX, l.HubY
	houses := biomes[l.Type].houses

	fits := l.free
	// candidates lists top-left positions for a w×h building, sorted by distance of the
	// building's center to a preferred spot.
	candidates := func(w, h int, prefX, prefY float64) [][2]int {
		var out [][2]int
		for dy := -clearRadius; dy <= clearRadius; dy++ {
			for dx := -clearRadius; dx <= clearRadius; dx++ {
				cx, cy := float64(dx)+float64(w)/2, float64(dy)+float64(h)/2
				if d := math.Hypot(cx, cy); d < plazaRadius+3 || d > clearRadius-3 {
					continue
				}
				out = append(out, [2]int{dx, dy})
			}
		}
		dist := func(p [2]int) float64 {
			return math.Hypot(float64(p[0])+float64(w)/2-prefX, float64(p[1])+float64(h)/2-prefY)
		}
		sort.Slice(out, func(i, j int) bool { return dist(out[i]) < dist(out[j]) })
		return out
	}

	// The inn north of the plaza if possible, its door facing the plaza.
	inn := tiles.Get(tiles.Inn)
	for _, p := range candidates(inn.W, inn.H, 0, -10) {
		if b := (Building{hx + p[0], hy + p[1], tiles.Inn}); fits(b, 1) {
			l.Buildings = append(l.Buildings, b)
			break
		}
	}
	// Up to four houses spread around the plaza: one preferred spot per direction.
	for i, dir := range [][2]float64{{-12, -6}, {12, -6}, {-12, 7}, {12, 7}} {
		kind := houses[(i+int(rand01(seed, saltHouse, hx, hy+i)*4))%len(houses)]
		hs := tiles.Get(kind)
		for _, p := range candidates(hs.W, hs.H, dir[0], dir[1]) {
			if b := (Building{hx + p[0], hy + p[1], kind}); fits(b, 2) {
				l.Buildings = append(l.Buildings, b)
				break
			}
		}
	}
}

// roadDist returns the distance from a tile to the nearest road center line.
func (l *ZoneLayout) roadDist(x, y int) float64 {
	p := point{float64(x), float64(y)}
	best := math.Inf(1)
	for _, s := range l.segments {
		if p.x < s.minX-best || p.x > s.maxX+best || p.y < s.minY-best || p.y > s.maxY+best {
			continue
		}
		best = math.Min(best, distToSegment(p, s.a, s.b))
	}
	return best
}

func (l *ZoneLayout) hubDist(x, y int) float64 {
	return math.Hypot(float64(x-l.HubX), float64(y-l.HubY))
}

// isRoad reports whether a tile is dirt: a road or the settlement plaza.
func (l *ZoneLayout) isRoad(x, y int) bool {
	return l.hubDist(x, y) <= plazaRadius || l.roadDist(x, y) <= roadHalfWidth
}

// building returns the building covering a tile, if any.
func (l *ZoneLayout) building(x, y int) (Building, bool) {
	for _, b := range l.Buildings {
		if b.covers(x, y) {
			return b, true
		}
	}
	return Building{}, false
}

// Inn returns the zone's inn, if it has one.
func (l *ZoneLayout) Inn() (Building, bool) {
	for _, b := range l.Buildings {
		if b.Sprite == tiles.Inn {
			return b, true
		}
	}
	return Building{}, false
}

// Checkpoint kinds.
const (
	NoCheckpoint = iota
	InnCheckpoint
	ShrineCheckpoint
)

// CheckpointAt reports whether a tile is a checkpoint the player can use by facing it: an inn
// door or a shrine.
func CheckpointAt(seed int64, x, y int) int {
	l := LayoutAt(seed, x, y)
	b, ok := l.building(x, y)
	switch {
	case !ok:
		return NoCheckpoint
	case b.Sprite == tiles.Shrine:
		return ShrineCheckpoint
	case b.Sprite == tiles.Inn:
		if dx, dy := b.Door(); dx == x && dy == y {
			return InnCheckpoint
		}
	}
	return NoCheckpoint
}

// StartPoint is where a new game begins: in front of the start zone's inn.
func StartPoint(seed int64) (x, y int, ok bool) {
	return InnPoint(seed, 0, 0)
}

// InnPoint returns the tile in front of the inn door of a zone's settlement.
func InnPoint(seed int64, zx, zy int) (x, y int, ok bool) {
	inn, found := Layout(seed, zx, zy).Inn()
	if !found {
		return 0, 0, false
	}
	dx, dy := inn.Door()
	return dx, dy + 1, true
}

func distToSegment(p, a, b point) float64 {
	dx, dy := b.x-a.x, b.y-a.y
	l2 := dx*dx + dy*dy
	if l2 == 0 {
		return math.Hypot(p.x-a.x, p.y-a.y)
	}
	t := math.Max(0, math.Min(1, ((p.x-a.x)*dx+(p.y-a.y)*dy)/l2))
	return math.Hypot(p.x-(a.x+t*dx), p.y-(a.y+t*dy))
}
