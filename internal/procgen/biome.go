package procgen

import (
	"strings"

	"oldworld2/internal/tiles"
)

// Zone types and their look (see design/world.md, Zones and Zone Placement).

// ZoneType is the kind of a zone.
type ZoneType int

const (
	Grassland ZoneType = iota
	Desert
	North
	Mediterranean
	Unholy
	AncientDesert // the Egyptian-themed desert; Desert is the Middle Eastern one
)

var zoneNames = map[ZoneType]string{
	Grassland: "Grassland", Desert: "Desert", North: "North",
	Mediterranean: "Mediterranean", Unholy: "Unholy", AncientDesert: "Ancient Desert",
}

// Implemented lists the zone types that exist in the world so far (the others fall back to
// grassland, see ZoneTypeOf), in the order a menu shows them.
var Implemented = []ZoneType{Grassland, Desert, AncientDesert, North, Mediterranean, Unholy}

// ParseZoneType reads a zone type's name, ignoring case, spaces and underscores
// ("ancient_desert", "Ancient Desert").
func ParseZoneType(name string) (ZoneType, bool) {
	norm := func(s string) string {
		return strings.ToLower(strings.NewReplacer(" ", "", "_", "", "-", "").Replace(s))
	}
	for t, n := range zoneNames {
		if norm(n) == norm(name) {
			return t, true
		}
	}
	return 0, false
}

// NearestZone finds the zone of a type closest to the start zone (by zone ring, then
// clockwise from the north), within a few rings; ok is false if there is none.
func NearestZone(seed int64, t ZoneType) (zx, zy int, ok bool) {
	for r := 0; r <= nearestZoneRings; r++ {
		for _, z := range ring(r) {
			if ZoneTypeOf(seed, z[0], z[1]) == t {
				return z[0], z[1], true
			}
		}
	}
	return 0, 0, false
}

const nearestZoneRings = 12

// ring lists the zones at ring r around the start zone, from the north clockwise.
func ring(r int) [][2]int {
	if r == 0 {
		return [][2]int{{0, 0}}
	}
	var out [][2]int
	for x := -r; x < r; x++ { // top edge, west to east
		out = append(out, [2]int{x, -r})
	}
	for y := -r; y < r; y++ { // east edge, north to south
		out = append(out, [2]int{r, y})
	}
	for x := r; x > -r; x-- { // bottom edge, east to west
		out = append(out, [2]int{x, r})
	}
	for y := r; y > -r; y-- { // west edge, south to north
		out = append(out, [2]int{-r, y})
	}
	return out
}

// IsDesert reports whether a zone type is one of the deserts.
func (z ZoneType) IsDesert() bool { return z == Desert || z == AncientDesert }

func (z ZoneType) String() string { return zoneNames[z] }

const saltZoneType uint64 = 400

// zoneStayOdds is the chance that a zone near the start stays grassland, per ring.
var zoneStayOdds = []float64{1, 0.3, 0.15}

// ZoneTypeOf decides a zone's type from its direction from the start zone: north leads to
// the North, east to Unholy, west to the Mediterranean and south to the Mediterranean or the
// Desert; near the start a zone can stay grassland.
func ZoneTypeOf(seed int64, zx, zy int) ZoneType {
	t := plannedZoneType(seed, zx, zy)
	switch t {
	case Grassland, AncientDesert, North, Mediterranean, Unholy:
		return t
	case Desert:
		// Two desert themes; noise over the zone grid keeps neighboring deserts alike.
		if valueNoise(seed, saltZoneType+2, float64(zx)/2, float64(zy)/2) > 0.5 {
			return AncientDesert
		}
		return Desert
	}
	return Grassland // not implemented yet
}

func plannedZoneType(seed int64, zx, zy int) ZoneType {
	ring := max(abs(zx), abs(zy))
	stay := 0.05
	if ring < len(zoneStayOdds) {
		stay = zoneStayOdds[ring]
	}
	if rand01(seed, saltZoneType, zx, zy) < stay {
		return Grassland
	}
	// Smooth noise over the zone grid keeps deserts and coasts in connected regions.
	n := valueNoise(seed, saltZoneType+1, float64(zx)/2, float64(zy)/2)
	switch {
	case abs(zy) >= abs(zx) && zy > 0:
		if n+0.08*float64(zy-1) > 0.5 { // the farther south, the more desert
			return Desert
		}
		return Mediterranean
	case abs(zy) >= abs(zx):
		return North
	case zx > 0:
		if zx >= 3 && n > 0.75 { // desert can appear east of the unholy lands
			return Desert
		}
		return Unholy
	default:
		return Mediterranean
	}
}

// Difficulty is 1 in the start zone and rises by one per zone ring around it.
func Difficulty(zx, zy int) int { return DifficultyFrom(0, 0, zx, zy) }

// DifficultyFrom is the difficulty of a zone when the game started in zone (hx, hy) instead:
// 1 there, rising by one per ring around it.
func DifficultyFrom(hx, hy, zx, zy int) int { return 1 + max(abs(zx-hx), abs(zy-hy)) }

// biome holds the generation data of a zone type.
type biome struct {
	ground      tiles.ID
	variants    []tiles.ID // ground variants, used on variantOdds of the tiles
	variantOdds float64
	water, road tiles.ID // first piece of the lake and road autotile sets

	lakeThreshold float64
	forestStart   float64 // vegetation noise above this grows trees (or boulders)
	maxDensity    float64
	trees         []tiles.ID // 2×2 vegetation in forests
	barrier       []tiles.ID // 2×2 objects of zone barriers
	nearWater     []tiles.ID // 2×2 vegetation that grows around lakes (palms); none if empty
	bush          tiles.ID   // small 1×1 obstacle
	decor         []tiles.ID
	decorOdds     float64
	houses        []tiles.ID
}

var biomes = map[ZoneType]*biome{
	Grassland: {
		ground:      tiles.Grass,
		variants:    []tiles.ID{tiles.GrassTuftA, tiles.GrassTuftB, tiles.GrassTuftC, tiles.GrassTuftD, tiles.GrassTuftE},
		variantOdds: 0.12, water: tiles.Water, road: tiles.Dirt,
		lakeThreshold: lakeThreshold, forestStart: forestStart, maxDensity: maxForestDensity,
		trees:   []tiles.ID{tiles.TreeRound, tiles.TreeRound, tiles.TreeRound, tiles.TreeRound, tiles.TreeRound, tiles.TreeRound, tiles.TreePine, tiles.TreePine, tiles.TreePine, tiles.TreeRoots},
		barrier: []tiles.ID{tiles.TreeRound, tiles.TreeRound, tiles.TreePine},
		bush:    tiles.Bush,
		decor: []tiles.ID{
			tiles.PlantA, tiles.PlantB, tiles.PlantC, tiles.PlantB, tiles.PlantC,
			tiles.FlowerSun, tiles.FlowerOrange, tiles.FlowerRed, tiles.FlowerWhite, tiles.Clover,
		},
		decorOdds: decorOdds,
		houses:    []tiles.ID{tiles.HouseOrange, tiles.HouseOrange2},
	},
	Desert: {
		ground:      tiles.Sand,
		variants:    []tiles.ID{tiles.SandTuftA, tiles.SandTuftB, tiles.SandTuftC, tiles.SandTuftD},
		variantOdds: 0.08, water: tiles.Oasis, road: tiles.SandPath,
		lakeThreshold: 0.75,                  // only small, rare oases
		forestStart:   0.56, maxDensity: 0.5, // scattered boulder fields
		trees:     []tiles.ID{tiles.Boulder, tiles.Boulder, tiles.BoulderB, tiles.RockStump},
		barrier:   []tiles.ID{tiles.Boulder, tiles.BoulderB},
		nearWater: []tiles.ID{tiles.Palm, tiles.PalmB},
		bush:      tiles.SmallRock,
		decor:     []tiles.ID{tiles.Skull, tiles.Ribs, tiles.Bone, tiles.SkullB, tiles.SproutOrange, tiles.SproutOlive, tiles.SproutOlive},
		decorOdds: 0.025,
		houses:    []tiles.ID{tiles.DomeHut, tiles.SandHouse, tiles.DomeTower},
	},
	North: {
		ground:      tiles.Snow,
		variants:    []tiles.ID{tiles.SnowTuftA, tiles.SnowTuftB, tiles.SnowTuftC},
		variantOdds: 0.1, water: tiles.Ice, road: tiles.Dirt,
		lakeThreshold: lakeThreshold, forestStart: forestStart - 0.03, maxDensity: maxForestDensity, // dense pine forests
		trees:     []tiles.ID{tiles.SnowPine, tiles.SnowPine, tiles.SnowPine, tiles.SnowPineB, tiles.SnowPineB, tiles.SnowRock},
		barrier:   []tiles.ID{tiles.SnowPine, tiles.SnowPineB},
		bush:      tiles.SnowBush,
		decor:     []tiles.ID{tiles.FrostFern, tiles.FrostFern, tiles.DeadTwig, tiles.IceShard},
		decorOdds: decorOdds * 0.6,
		houses:    []tiles.ID{tiles.Longhouse, tiles.NordHut, tiles.NordHut},
	},
	Mediterranean: {
		ground:      tiles.DryGrass,
		variants:    []tiles.ID{tiles.DryGrassTuftA, tiles.DryGrassTuftB, tiles.DryGrassTuftC},
		variantOdds: 0.12, water: tiles.Water, road: tiles.Dirt,
		lakeThreshold: lakeThreshold, forestStart: forestStart + 0.02, maxDensity: 0.45, // open olive groves
		trees:   []tiles.ID{tiles.OliveTree, tiles.OliveTree, tiles.OliveTree, tiles.Cypress, tiles.Cypress, tiles.LimeRock},
		barrier: []tiles.ID{tiles.Cypress, tiles.Cypress, tiles.LimeRock},
		bush:    tiles.Lavender,
		decor: []tiles.ID{tiles.Sage, tiles.Sage, tiles.Poppies, tiles.Poppies, tiles.FlowerWhite, tiles.Lavender,
			tiles.BrokenColumn, tiles.Amphora},
		decorOdds: decorOdds * 0.8,
		houses:    []tiles.ID{tiles.WhiteHouse, tiles.WhiteHouse, tiles.Villa},
	},
	Unholy: {
		ground:      tiles.DeadGrass,
		variants:    []tiles.ID{tiles.DeadGrassTuftA, tiles.DeadGrassTuftB},
		variantOdds: 0.1, water: tiles.Water, road: tiles.Dirt,
		lakeThreshold: lakeThreshold + 0.02, forestStart: forestStart - 0.02, maxDensity: maxForestDensity, // dark, dense woods
		trees:   []tiles.ID{tiles.DeadTree, tiles.DeadTree, tiles.DeadTree, tiles.DarkPine, tiles.DarkPine, tiles.DarkRock},
		barrier: []tiles.ID{tiles.DarkPine, tiles.DeadTree},
		bush:    tiles.ThornBush,
		decor: []tiles.ID{tiles.Gravestone, tiles.GravestoneB, tiles.Mushrooms, tiles.Mushrooms, tiles.Skull, tiles.Bone,
			tiles.ThornBush},
		decorOdds: decorOdds * 0.7,
		houses:    []tiles.ID{tiles.GothicHouse, tiles.GothicHouse, tiles.Crypt},
	},
	AncientDesert: {
		ground:      tiles.Sand,
		variants:    []tiles.ID{tiles.SandTuftA, tiles.SandTuftB, tiles.SandTuftC, tiles.SandTuftD},
		variantOdds: 0.06, water: tiles.Oasis, road: tiles.SandPath,
		lakeThreshold: 0.76,
		forestStart:   0.58, maxDensity: 0.4, // fewer boulders: open sand around the monuments
		trees:     []tiles.ID{tiles.Boulder, tiles.BoulderB, tiles.BoulderB},
		barrier:   []tiles.ID{tiles.Boulder, tiles.BoulderB},
		nearWater: []tiles.ID{tiles.Palm, tiles.PalmB},
		bush:      tiles.SmallRock,
		decor:     []tiles.ID{tiles.Skull, tiles.Bone, tiles.SkullB, tiles.SproutOlive},
		decorOdds: 0.02,
		houses:    []tiles.ID{tiles.SandHouse, tiles.Temple},
	},
}

// biomeAt returns the generation data of the zone containing a tile.
func biomeAt(seed int64, x, y int) *biome {
	zx, zy := ZoneOf(x, y)
	return biomes[ZoneTypeOf(seed, zx, zy)]
}

// ZoneTypeAt returns the type of the zone containing a tile.
func ZoneTypeAt(seed int64, x, y int) ZoneType {
	zx, zy := ZoneOf(x, y)
	return ZoneTypeOf(seed, zx, zy)
}
