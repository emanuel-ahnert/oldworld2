// Package tiles defines the IDs of the world map's ground tiles and objects, and the objects'
// footprints. It is shared by world generation, the game and the map preview tool.
package tiles

// Size is the edge length of one tile in pixels: the unit of positions in the world package.
const Size = 16

// Footprint is the size of an object in tiles; it stands on the bottom row.
type Footprint struct {
	W, H int
}

// ID identifies a ground tile or an object.
type ID uint16

const (
	None ID = iota

	// Ground: grass.
	Grass
	GrassTuftA
	GrassTuftB
	GrassTuftC
	GrassTuftD
	GrassTuftE

	// Ground: water autotile pieces. Edge and corner names say where the land is.
	Water
	WaterN
	WaterS
	WaterE
	WaterW
	WaterNE
	WaterNW
	WaterSE
	WaterSW
	WaterInnerNE // land only at the north-east diagonal
	WaterInnerNW
	WaterInnerSE
	WaterInnerSW

	// Ground: oasis water with sand shores, same piece order as Water.
	Oasis
	OasisN
	OasisS
	OasisE
	OasisW
	OasisNE
	OasisNW
	OasisSE
	OasisSW
	OasisInnerNE
	OasisInnerNW
	OasisInnerSE
	OasisInnerSW

	// Ground: desert sand.
	Sand
	SandTuftA
	SandTuftB
	SandTuftC
	SandTuftD

	// Ground: light sand paths on sand, same piece order as Water.
	SandPath
	SandPathN
	SandPathS
	SandPathE
	SandPathW
	SandPathNE
	SandPathNW
	SandPathSE
	SandPathSW
	SandPathInnerNE
	SandPathInnerNW
	SandPathInnerSE
	SandPathInnerSW

	// Ground: dirt road autotile pieces, same naming as water (where the grass is).
	Dirt
	DirtN
	DirtS
	DirtE
	DirtW
	DirtNE
	DirtNW
	DirtSE
	DirtSW
	DirtInnerNE
	DirtInnerNW
	DirtInnerSE
	DirtInnerSW

	// Ground: snow (the North).
	Snow
	SnowTuftA
	SnowTuftB
	SnowTuftC

	// Ground: frozen lakes with snowy shores, same piece order as Water.
	Ice
	IceN
	IceS
	IceE
	IceW
	IceNE
	IceNW
	IceSE
	IceSW
	IceInnerNE
	IceInnerNW
	IceInnerSE
	IceInnerSW

	// Ground: sun-dried grass (the Mediterranean).
	DryGrass
	DryGrassTuftA
	DryGrassTuftB
	DryGrassTuftC

	// Ground: dead grass (the unholy lands).
	DeadGrass
	DeadGrassTuftA
	DeadGrassTuftB

	// Objects.
	TreeRound
	TreePine
	TreeRoots
	Bush
	PlantA
	PlantB
	PlantC
	FlowerSun
	FlowerOrange
	FlowerRed
	FlowerWhite
	Clover

	// Desert objects.
	Palm
	PalmB
	Boulder
	BoulderB
	RockStump
	SmallRock
	Skull
	Ribs
	Bone
	SkullB
	SproutOrange
	SproutOlive

	// North objects.
	SnowPine
	SnowPineB
	SnowRock
	SnowBush
	FrostFern
	DeadTwig
	IceShard

	// Mediterranean objects.
	OliveTree
	Cypress
	LimeRock
	Lavender
	Sage
	Poppies
	BrokenColumn
	Amphora

	// Unholy objects.
	DeadTree
	DarkPine
	DarkRock
	ThornBush
	Gravestone
	GravestoneB
	Mushrooms

	// Buildings (door in the middle of the bottom row).
	HouseOrange
	HouseBeige
	HouseOrange2
	HouseRed
	Inn
	Shrine // checkpoint along roads
	DomeHut
	DomeTower
	SandHouse
	Temple
	Pyramid
	PyramidSmall
	Obelisk
	Sphinx
	PharaohStatue
	Longhouse
	NordHut
	WhiteHouse
	Villa
	GothicHouse
	Crypt

	count
)

// footprints are the sizes of the objects bigger than one tile; everything else is 1×1.
var footprints = map[ID]Footprint{
	Palm:          {2, 2},
	PalmB:         {2, 2},
	Boulder:       {2, 2},
	BoulderB:      {2, 2},
	RockStump:     {2, 2},
	DomeHut:       {4, 4},
	DomeTower:     {3, 4},
	SandHouse:     {5, 4},
	Temple:        {4, 4},
	Pyramid:       {6, 5},
	PyramidSmall:  {4, 3},
	Obelisk:       {1, 3},
	Sphinx:        {2, 3},
	PharaohStatue: {2, 2},
	HouseOrange:   {5, 4},
	HouseBeige:    {5, 4},
	HouseOrange2:  {5, 4},
	HouseRed:      {5, 4},
	Inn:           {6, 4},
	Shrine:        {2, 2},
	TreeRound:     {2, 2},
	TreePine:      {2, 2},
	TreeRoots:     {2, 2},
	SnowPine:      {2, 2},
	SnowPineB:     {2, 2},
	SnowRock:      {2, 2},
	Longhouse:     {7, 4},
	NordHut:       {4, 4},
	WhiteHouse:    {5, 4},
	Villa:         {6, 4},
	OliveTree:     {2, 2},
	Cypress:       {2, 2},
	LimeRock:      {2, 2},
	DeadTree:      {2, 2},
	DarkPine:      {2, 2},
	DarkRock:      {2, 2},
	GothicHouse:   {5, 4},
	Crypt:         {4, 4},
}

// IsWater reports whether a ground tile is one of the water pieces (lakes, oases or frozen lakes).
func IsWater(id ID) bool {
	return (id >= Water && id <= OasisInnerSW) || (id >= Ice && id <= IceInnerSW)
}

// IsIce reports whether a ground tile is one of the frozen lake pieces.
func IsIce(id ID) bool { return id >= Ice && id <= IceInnerSW }

// IsSnow reports whether a ground tile is plain snow.
func IsSnow(id ID) bool { return id >= Snow && id <= SnowTuftC }

// IsRoad reports whether a ground tile is part of a road (dirt or sand path).
func IsRoad(id ID) bool {
	return (id >= Dirt && id <= DirtInnerSW) || (id >= SandPath && id <= SandPathInnerSW)
}

// IsSand reports whether a ground tile is plain desert sand.
func IsSand(id ID) bool { return id >= Sand && id <= SandTuftD }

// IsBuilding reports whether an object is a building.
func IsBuilding(id ID) bool { return id >= HouseOrange && id <= Crypt }

// IsDeadGrass reports whether a ground tile is the unholy lands' dead grass.
func IsDeadGrass(id ID) bool { return id >= DeadGrass && id <= DeadGrassTuftB }

// IsDryGrass reports whether a ground tile is the Mediterranean's dry grass.
func IsDryGrass(id ID) bool { return id >= DryGrass && id <= DryGrassTuftC }

// Shape maps a water piece (Water…WaterInnerSW) to the piece of the same shape in another
// autotile set that uses the same order, e.g. Shape(WaterN, Dirt) = DirtN.
func Shape(water, base ID) ID { return water - Water + base }

// Get returns the footprint of an ID.
func Get(id ID) Footprint {
	if f, ok := footprints[id]; ok {
		return f
	}
	return Footprint{1, 1}
}
