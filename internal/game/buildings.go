package game

import (
	"image/color"
	"path/filepath"

	rl "github.com/gen2brain/raylib-go/raylib"

	"oldworld2/internal/r3d"
	"oldworld2/internal/tiles"
)

// Buildings are built from simple shapes (boxes, gable roofs, domes, pyramids), toon-shaded
// like the models, since the free packs have no houses. Each stands on its footprint in the
// tile grid (the tiles it blocks), with the door in the middle of the front (south) wall,
// where the game expects it. Surfaces are tinted generated textures (plaster, wood, roof
// tiles, stone; assets/generated/buildings, made by tools/artgen/buildings.json).

const sqrt2 = 1.41421356

// surface is the look of a building part: a texture (a file in assets/generated/buildings,
// "" for none) at a scale in repeats per meter, tinted.
type surface struct {
	tex   string
	scale float32
	tint  color.RGBA
}

var (
	plaster  = surface{"tex_plaster", 0.45, color.RGBA{225, 212, 188, 255}}
	timber   = surface{"tex_wood", 0.8, color.RGBA{120, 85, 60, 255}}
	glass    = surface{"", 0, color.RGBA{90, 120, 150, 255}}
	stone    = surface{"tex_stone", 0.6, color.RGBA{175, 175, 182, 255}}
	sand     = surface{"tex_sandstone", 0.4, color.RGBA{228, 198, 145, 255}}
	sandLit  = surface{"tex_sandstone", 0.4, color.RGBA{238, 214, 166, 255}}
	adobe    = surface{"tex_adobe", 0.6, color.RGBA{228, 198, 145, 255}}
	adobeLit = surface{"tex_adobe", 0.6, color.RGBA{238, 214, 166, 255}}
	gold     = surface{"", 0, color.RGBA{230, 190, 90, 255}}
	logs     = surface{"tex_wood", 0.6, color.RGBA{105, 78, 58, 255}}
	logsDark = surface{"tex_wood", 0.6, color.RGBA{72, 54, 42, 255}}
	snowRoof = surface{"tex_plaster", 0.3, color.RGBA{236, 240, 248, 255}}
	white    = surface{"tex_plaster", 0.45, color.RGBA{246, 244, 238, 255}}
	darkWall = surface{"tex_stone", 0.6, color.RGBA{112, 108, 120, 255}}
	slate    = surface{"tex_rooftiles", 0.9, color.RGBA{70, 72, 88, 255}}
	glowing  = plain(color.RGBA{255, 186, 90, 255}) // lit windows
	seaBlue  = plain(color.RGBA{40, 96, 170, 255})
)

func roof(c color.RGBA) surface  { return surface{"tex_rooftiles", 0.7, c} }
func wall(c color.RGBA) surface  { return surface{"tex_plaster", 0.45, c} }
func plain(c color.RGBA) surface { return surface{"", 0, c} }

// shape is a part of a building: a mesh with a position (bottom center for boxes) relative
// to the footprint's north-west corner.
func (sc *scene) shape(mesh rl.Mesh, x, y, z float32, s surface) r3d.Object {
	o := r3d.Object{Model: rl.LoadModelFromMesh(mesh), Pos: rl.NewVector3(x, y, z), Scale: 1, Tint: s.tint, Outline: 0.025}
	if s.tex != "" {
		rl.SetMaterialTexture(o.Model.Materials, rl.MapAlbedo, sc.surfaceTex(s.tex))
		o.Detail = s.scale
	}
	return o
}

// surfaceTex loads a building texture once.
func (sc *scene) surfaceTex(name string) rl.Texture2D {
	t, ok := sc.surfaces[name]
	if !ok {
		t = r3d.Detail(filepath.Join(sc.g.opts.AssetsDir, "generated/buildings", name+".png"))
		sc.surfaces[name] = t
	}
	return t
}

func (sc *scene) box(w, h, d, x, y, z float32, s surface) r3d.Object {
	return sc.shape(rl.GenMeshCube(w, h, d), x, y+h/2, z, s)
}

// gable is a roof over walls of w×d whose top is at y, with the ridge running north-south
// (so the map camera, looking from the south, sees both slopes, lit differently). The roof
// is one 45° slab, overhanging the walls on all sides, and the gable ends are triangles
// that fit right under it.
func (sc *scene) gable(w, d, x, y, z float32, top, fill surface) []r3d.Object {
	return sc.pitchedGable(w, d, x, y, z, 1, top, fill)
}

// pitchedGable is a gable roof rising pitch times as steeply as a 45° one (2 for the steep
// roofs of the unholy lands).
func (sc *scene) pitchedGable(w, d, x, y, z, pitch float32, top, fill surface) []r3d.Object {
	const over, thick = 0.3, 0.16 * sqrt2 // eaves overhang; slab thickness, measured vertically
	a := w / 2
	h := a * pitch
	slab := r3d.Prism([][2]float32{
		{-(a + over), -over * pitch}, {0, h}, {a + over, -over * pitch}, // underside: through the wall tops
		{a + over, -over*pitch + thick}, {0, h + thick}, {-(a + over), -over*pitch + thick},
	}, d+0.7, [][]int{{0, 1, 4, 5}, {1, 2, 3, 4}})
	ends := r3d.Prism([][2]float32{{-a, 0}, {a, 0}, {0, h}}, d, nil)
	return []r3d.Object{sc.shape(ends, x, y, z, fill), sc.shape(slab, x, y, z, top)}
}

// pyramid is a four-sided pyramid with a square base of side a.
func (sc *scene) pyramid(a, h, x, y, z float32, s surface) r3d.Object {
	o := sc.shape(rl.GenMeshCone(a/sqrt2, h, 4), x, y, z, s)
	o.Yaw = 45
	return o
}

// window is a framed window on a wall facing south (+z) or, with side, east or west: dark
// glass in a timber frame with a lintel and a sill, and shutters of the given color.
func (sc *scene) window(x, y, z float32, side int, shutters surface) []r3d.Object {
	const ww, wh = 0.6, 0.7
	if side == 0 { // in the front (south) wall
		return []r3d.Object{
			sc.box(ww, wh, 0.08, x, y, z, glass),
			sc.box(0.06, wh, 0.1, x, y, z, timber), // the mullion
			sc.box(ww+0.2, 0.1, 0.14, x, y+wh, z, timber),
			sc.box(ww+0.2, 0.08, 0.18, x, y-0.08, z+0.04, timber),
			sc.box(0.26, wh+0.04, 0.06, x-ww/2-0.16, y-0.02, z+0.05, shutters),
			sc.box(0.26, wh+0.04, 0.06, x+ww/2+0.16, y-0.02, z+0.05, shutters),
		}
	}
	s := float32(side) // in a side wall: the same, turned to face east (+1) or west (-1)
	return []r3d.Object{
		sc.box(0.08, wh, ww, x, y, z, glass),
		sc.box(0.1, wh, 0.06, x, y, z, timber),
		sc.box(0.14, 0.1, ww+0.2, x, y+wh, z, timber),
		sc.box(0.18, 0.08, ww+0.2, x+s*0.04, y-0.08, z, timber),
		sc.box(0.06, wh+0.04, 0.26, x+s*0.05, y-0.02, z-ww/2-0.16, shutters),
		sc.box(0.06, wh+0.04, 0.26, x+s*0.05, y-0.02, z+ww/2+0.16, shutters),
	}
}

// frontWindows spaces windows along a front wall from left to right: in pairs out from the
// door, with one over the door too when over is set (an upper floor).
func frontWindows(doorX, left, right float32, over bool) []float32 {
	var xs []float32
	if over {
		xs = append(xs, doorX)
	}
	for k := float32(1); ; k++ {
		l, r := doorX-1.55*k, doorX+1.55*k
		if l < left+0.55 && r > right-0.55 {
			return xs
		}
		if l >= left+0.55 {
			xs = append(xs, l)
		}
		if r <= right-0.55 {
			xs = append(xs, r)
		}
	}
}

// house builds a cottage or inn on a w×d footprint: a stone footing, plastered walls in a
// timber frame (corner posts, a beam under the eaves, and one between the floors of an inn),
// framed windows with shutters on every wall, a door with a frame and a step, a ridge beam
// and a chimney; an inn has two floors of windows, a hanging sign and a lantern.
func (sc *scene) house(w, d float32, walls, top surface, wallH float32, inn bool, shutters surface) []r3d.Object {
	ww, wd := w-0.3, d-0.6
	cz := 0.15 + wd/2 // walls from z = 0.15 to the front at d - 0.45
	front := cz + wd/2
	back := cz - wd/2
	left, right := w/2-ww/2, w/2+ww/2
	doorX := float32(int(w)/2) + 0.5
	parts := []r3d.Object{
		sc.box(ww+0.16, 0.3, wd+0.16, w/2, 0, cz, stone), // footing
		sc.box(ww, wallH, wd, w/2, 0.3, cz, walls),
		sc.box(ww+0.1, 0.16, wd+0.1, w/2, 0.3+wallH-0.16, cz, timber), // beam under the eaves
	}
	for _, c := range [][2]float32{{left, back}, {right, back}, {left, front}, {right, front}} {
		parts = append(parts, sc.box(0.2, wallH, 0.2, c[0], 0.3, c[1], timber))
	}
	floors := []float32{1.3}
	if inn {
		floors = []float32{1.2, 2.75}
		parts = append(parts, sc.box(ww+0.08, 0.14, wd+0.08, w/2, 2.4, cz, timber)) // between the floors
	}
	// Door: a frame, a lintel and a stone step.
	parts = append(parts,
		sc.box(0.95, 1.8, 0.1, doorX, 0.3, front, timber),
		sc.box(0.12, 1.95, 0.16, doorX-0.53, 0.3, front, timber), sc.box(0.12, 1.95, 0.16, doorX+0.53, 0.3, front, timber),
		sc.box(1.25, 0.14, 0.18, doorX, 2.2, front, timber),
		sc.box(1.3, 0.16, 0.45, doorX, 0.14, front+0.22, stone),
	)
	// Windows: in pairs either side of the door (on the upper floor of an inn, over it too),
	// then on the sides.
	for _, y := range floors {
		for _, wx := range frontWindows(doorX, left, right, y > 2) {
			parts = append(parts, sc.window(wx, y+0.3, front+0.02, 0, shutters)...)
		}
		for wz := back + 0.8; wz < front-0.6; wz += 1.4 {
			parts = append(parts, sc.window(left-0.02, y+0.3, wz, -1, shutters)...)
			parts = append(parts, sc.window(right+0.02, y+0.3, wz, 1, shutters)...)
		}
	}
	parts = append(parts, sc.gable(ww, wd, w/2, 0.3+wallH, cz, top, walls)...)
	parts = append(parts, sc.box(0.16, 0.16, wd+0.7, w/2, 0.3+wallH+ww/2+0.05, cz, timber)) // ridge beam
	chimney := surface{"tex_stone", 1.2, color.RGBA{150, 120, 100, 255}}
	parts = append(parts, sc.box(0.5, 1.8, 0.5, w/2+ww/4, 0.3+wallH, cz-wd/5, chimney),
		sc.box(0.62, 0.12, 0.62, w/2+ww/4, 0.3+wallH+1.8, cz-wd/5, stone))
	if inn {
		// A sign hanging from a bracket beside the door, and a lantern on the other side.
		parts = append(parts,
			sc.box(0.08, 0.08, 0.8, doorX+1.1, 2.35, front+0.4, timber),
			sc.box(0.08, 0.5, 0.7, doorX+1.1, 1.8, front+0.7, timber), // the sign board, edge on to the road
			sc.box(0.1, 0.36, 0.56, doorX+1.1, 1.87, front+0.7, surface{"", 0, color.RGBA{200, 160, 70, 255}}),
			sc.box(0.06, 0.06, 0.3, doorX-0.95, 1.85, front+0.15, timber),
			sc.box(0.2, 0.28, 0.2, doorX-0.95, 1.55, front+0.28, surface{"", 0, color.RGBA{255, 210, 120, 255}}),
		)
	}
	return parts
}

// cabin builds a northern log house on a w×d footprint: a stone footing, timber walls banded
// with darker logs and corner posts, a steep roof under snow, a plank door and shuttered
// windows; with chimney, a stone chimney through the roof.
func (sc *scene) cabin(w, d, wallH float32, chimney bool) []r3d.Object {
	ww, wd := w-0.4, d-0.6
	cz := 0.15 + wd/2
	front := cz + wd/2
	doorX := float32(int(w)/2) + 0.5
	parts := []r3d.Object{
		sc.box(ww+0.2, 0.35, wd+0.2, w/2, 0, cz, stone),
		sc.box(ww, wallH, wd, w/2, 0.35, cz, logs),
	}
	for _, y := range []float32{0.9, 1.5} {
		if y < wallH {
			parts = append(parts, sc.box(ww+0.06, 0.12, wd+0.06, w/2, y, cz, logsDark))
		}
	}
	for _, c := range [][2]float32{{w/2 - ww/2, cz - wd/2}, {w/2 + ww/2, cz - wd/2}, {w/2 - ww/2, front}, {w/2 + ww/2, front}} {
		parts = append(parts, sc.box(0.28, wallH+0.2, 0.28, c[0], 0.3, c[1], logsDark))
	}
	parts = append(parts, sc.box(0.9, 1.7, 0.12, doorX, 0.35, front, logsDark), sc.box(1.2, 0.15, 0.4, doorX, 0.2, front+0.2, stone))
	for _, wx := range frontWindows(doorX, w/2-ww/2, w/2+ww/2, false) {
		parts = append(parts, sc.window(wx, 1.2, front+0.02, 0, logsDark)...)
	}
	for wz := cz - wd/2 + 0.8; wz < front-0.6; wz += 1.5 {
		parts = append(parts, sc.window(w/2-ww/2-0.02, 1.2, wz, -1, logsDark)...)
		parts = append(parts, sc.window(w/2+ww/2+0.02, 1.2, wz, 1, logsDark)...)
	}
	parts = append(parts, sc.gable(ww, wd, w/2, wallH+0.35, cz, snowRoof, logs)...)
	if chimney {
		parts = append(parts, sc.box(0.5, 2.0, 0.5, w/2+ww/4, wallH, cz-wd/5, stone), sc.box(0.6, 0.15, 0.6, w/2+ww/4, wallH+2.0, cz-wd/5, snowRoof))
	}
	return parts
}

// whiteHouse builds a whitewashed Mediterranean house on a w×d footprint: a flat roof with a
// low parapet, blue shutters and a blue door, a small blue dome on one corner and a flowering
// shrub by the door.
func (sc *scene) whiteHouse(w, d float32) []r3d.Object {
	const wallH = 2.9
	ww, wd := w-0.4, d-0.6
	cz := 0.15 + wd/2
	front := cz + wd/2
	left, right := w/2-ww/2, w/2+ww/2
	doorX := float32(int(w)/2) + 0.5
	parts := []r3d.Object{
		sc.box(ww, wallH, wd, w/2, 0, cz, white),
		sc.box(ww+0.2, 0.18, wd+0.2, w/2, wallH, cz, white),           // the roof slab
		sc.box(ww+0.2, 0.3, 0.14, w/2, wallH+0.18, front+0.03, white), // the parapet, front and sides
		sc.box(0.14, 0.3, wd+0.2, left-0.03, wallH+0.18, cz, white),
		sc.box(0.14, 0.3, wd+0.2, right+0.03, wallH+0.18, cz, white),
		sc.box(0.9, 1.85, 0.1, doorX, 0, front, seaBlue),
		sc.box(1.1, 0.12, 0.14, doorX, 1.85, front, white),
		sc.box(1.3, 0.12, 0.5, doorX, 0, front+0.25, stone), // the step
	}
	for _, wx := range frontWindows(doorX, left, right, false) {
		parts = append(parts, sc.window(wx, 1.35, front+0.02, 0, seaBlue)...)
	}
	for wz := cz - wd/2 + 0.8; wz < front-0.6; wz += 1.5 {
		parts = append(parts, sc.window(left-0.02, 1.35, wz, -1, seaBlue)...)
		parts = append(parts, sc.window(right+0.02, 1.35, wz, 1, seaBlue)...)
	}
	// A small blue dome on a drum at the back corner.
	parts = append(parts, sc.shape(rl.GenMeshCylinder(0.75, 0.5, 14), right-0.9, wallH+0.18, cz-wd/4, white),
		sc.shape(rl.GenMeshHemiSphere(0.8, 8, 14), right-0.9, wallH+0.68, cz-wd/4, seaBlue))
	// Bougainvillea: a shrub of pink flowers against the wall beside the door.
	pink, leaf := plain(color.RGBA{214, 72, 150, 255}), plain(color.RGBA{70, 120, 60, 255})
	bx := doorX + 1.0
	if bx > right-0.5 {
		bx = doorX - 1.0
	}
	parts = append(parts, sc.shape(rl.GenMeshSphere(0.42, 8, 10), bx, 0.4, front+0.25, leaf),
		sc.shape(rl.GenMeshSphere(0.3, 8, 10), bx-0.15, 0.75, front+0.3, pink),
		sc.shape(rl.GenMeshSphere(0.26, 8, 10), bx+0.2, 1.05, front+0.25, pink))
	return parts
}

// gothicHouse builds a house of the unholy lands on a w×d footprint: dark stone walls, a steep
// slate roof, tall narrow windows glowing warm in the gloom, a dark door under a pointed hood,
// a chimney and a round turret with a cone roof on one corner.
func (sc *scene) gothicHouse(w, d float32) []r3d.Object {
	const wallH = 3.0
	ww, wd := w-0.5, d-0.6
	cz := 0.15 + wd/2
	front := cz + wd/2
	left, right := w/2-ww/2, w/2+ww/2
	doorX := float32(int(w)/2) + 0.5
	parts := []r3d.Object{
		sc.box(ww+0.2, 0.3, wd+0.2, w/2, 0, cz, stone),
		sc.box(ww, wallH, wd, w/2, 0.3, cz, darkWall),
		sc.box(0.9, 1.9, 0.1, doorX, 0.3, front, plain(color.RGBA{46, 34, 32, 255})),
		sc.pyramid(1.1, 0.5, doorX, 2.2, front+0.06, stone), // the door's pointed hood
	}
	for _, wx := range frontWindows(doorX, left, right, false) {
		parts = append(parts, sc.box(0.4, 1.0, 0.08, wx, 1.2, front+0.02, glowing), sc.box(0.56, 0.12, 0.14, wx, 1.12, front+0.02, stone),
			sc.pyramid(0.48, 0.3, wx, 2.2, front+0.02, stone))
	}
	for wz := cz - wd/2 + 0.8; wz < front-0.6; wz += 1.5 {
		parts = append(parts, sc.box(0.08, 1.0, 0.4, left-0.02, 1.2, wz, glowing), sc.box(0.08, 1.0, 0.4, right+0.02, 1.2, wz, glowing))
	}
	parts = append(parts, sc.pitchedGable(ww, wd, w/2, 0.3+wallH, cz, 1.7, slate, darkWall)...)
	parts = append(parts, sc.box(0.5, 2.4, 0.5, w/2-ww/4, 0.3+wallH, cz-wd/5, stone))
	// The turret: a round tower on the front corner, a little taller than the walls.
	tx, tz := right-0.1, front-0.3
	parts = append(parts, sc.shape(rl.GenMeshCylinder(0.55, wallH+1.0, 12), tx, 0.3, tz, darkWall),
		sc.box(0.3, 0.6, 0.08, tx, 2.6, tz+0.55, glowing),
		sc.shape(rl.GenMeshCone(0.7, 1.6, 12), tx, 0.3+wallH+1.0, tz, slate))
	return parts
}

// crypt builds a stone mausoleum on a w×d footprint: a plinth, thick walls under a steep
// stone roof, a dark iron gate between two columns and a stone cross on the ridge.
func (sc *scene) crypt(w, d float32) []r3d.Object {
	const wallH = 2.4
	ww, wd := w-0.8, d-1.0
	cz := 0.3 + wd/2
	front := cz + wd/2
	doorX := float32(int(w)/2) + 0.5
	grave := surface{"tex_stone", 0.7, color.RGBA{138, 136, 146, 255}}
	parts := []r3d.Object{
		sc.box(ww+0.5, 0.4, wd+0.6, w/2, 0, cz, grave),
		sc.box(ww, wallH, wd, w/2, 0.4, cz, grave),
		sc.box(1.0, 1.7, 0.1, doorX, 0.4, front, plain(color.RGBA{30, 30, 36, 255})), // the iron gate
	}
	for i := float32(0); i < 4; i++ {
		parts = append(parts, sc.box(0.05, 1.7, 0.06, doorX-0.36+i*0.24, 0.4, front+0.06, plain(color.RGBA{70, 70, 80, 255})))
	}
	for _, x := range []float32{doorX - 0.85, doorX + 0.85} {
		parts = append(parts, sc.shape(rl.GenMeshCylinder(0.16, wallH, 10), x, 0.4, front+0.25, grave))
	}
	parts = append(parts, sc.box(ww+0.3, 0.25, 0.5, w/2, 0.4+wallH, front+0.1, grave))
	parts = append(parts, sc.pitchedGable(ww, wd, w/2, 0.4+wallH, cz, 1.4, grave, grave)...)
	ridge := 0.4 + wallH + ww/2*1.4
	parts = append(parts, sc.box(0.14, 0.8, 0.14, w/2, ridge, front-0.3, grave), sc.box(0.5, 0.14, 0.14, w/2, ridge+0.45, front-0.3, grave))
	return parts
}

// building returns the parts of a building type (nil if it isn't one). They are built once.
func (sc *scene) building(id tiles.ID) []r3d.Object {
	if parts, ok := sc.buildings[id]; ok {
		return parts
	}
	s := tiles.Get(id)
	w, d := float32(s.W), float32(s.H)
	var parts []r3d.Object
	switch id {
	case tiles.HouseOrange:
		parts = sc.house(w, d, plaster, roof(color.RGBA{205, 105, 60, 255}), 2.9, false, plain(color.RGBA{70, 110, 70, 255}))
	case tiles.HouseBeige:
		parts = sc.house(w, d, wall(color.RGBA{228, 210, 175, 255}), roof(color.RGBA{150, 100, 70, 255}), 2.9, false,
			plain(color.RGBA{60, 90, 140, 255}))
	case tiles.HouseOrange2:
		parts = sc.house(w, d, plaster, roof(color.RGBA{215, 135, 70, 255}), 2.7, false, plain(color.RGBA{140, 60, 50, 255}))
	case tiles.HouseRed:
		parts = sc.house(w, d, plaster, roof(color.RGBA{175, 60, 50, 255}), 2.9, false, plain(color.RGBA{60, 90, 140, 255}))
	case tiles.Inn:
		parts = sc.house(w, d, wall(color.RGBA{232, 216, 180, 255}), roof(color.RGBA{85, 105, 140, 255}), 4.4, true,
			plain(color.RGBA{60, 100, 70, 255}))
	case tiles.Shrine:
		parts = []r3d.Object{
			sc.box(1.6, 0.3, 1.6, w/2, 0, d/2, stone),
			sc.box(0.7, 1.3, 0.7, w/2, 0.3, d/2, stone),
			sc.pyramid(0.9, 0.5, w/2, 1.6, d/2, surface{"tex_stone", 1.2, color.RGBA{120, 130, 160, 255}}),
		}
	case tiles.DomeHut:
		bw, bd := w-0.6, d-0.8
		parts = []r3d.Object{
			sc.box(bw, 2.2, bd, w/2, 0, 0.3+bd/2, adobe),
			sc.shape(rl.GenMeshHemiSphere(min(bw, bd)/2*0.95, 8, 14), w/2, 2.2, 0.3+bd/2, adobeLit),
			sc.box(0.8, 1.6, 0.1, float32(int(w)/2)+0.5, 0, 0.3+bd, timber),
		}
	case tiles.DomeTower:
		parts = []r3d.Object{
			sc.shape(rl.GenMeshCylinder(1.15, 4.5, 16), w/2, 0, d/2, adobe),
			sc.shape(rl.GenMeshHemiSphere(1.25, 8, 16), w/2, 4.5, d/2, adobeLit),
			sc.box(0.8, 1.6, 0.1, float32(int(w)/2)+0.5, 0, d/2+1.1, timber),
		}
	case tiles.SandHouse:
		bw, bd := w-0.4, d-0.7
		front := 0.2 + bd
		parts = []r3d.Object{
			sc.box(bw, 2.6, bd, w/2, 0, 0.2+bd/2, adobe),
			sc.box(bw+0.2, 0.3, bd+0.2, w/2, 2.6, 0.2+bd/2, adobeLit),
			sc.box(0.8, 1.7, 0.1, float32(int(w)/2)+0.5, 0, front, timber),
			sc.box(1.0, 0.12, 0.16, float32(int(w)/2)+0.5, 1.75, front, timber), // the door's lintel
		}
		dark := plain(color.RGBA{60, 50, 45, 255})
		for _, wx := range frontWindows(float32(int(w)/2)+0.5, w/2-bw/2, w/2+bw/2, false) {
			parts = append(parts, sc.box(0.5, 0.45, 0.08, wx, 1.45, front, dark), sc.box(0.62, 0.08, 0.14, wx, 1.4, front, timber))
		}
		for _, sx := range []float32{w/2 - bw/2, w/2 + bw/2} { // small windows in the side walls
			parts = append(parts, sc.box(0.08, 0.4, 0.45, sx, 1.5, 0.2+bd/2, dark))
		}
		for bx := w/2 - bw/2 + 0.5; bx < w/2+bw/2-0.3; bx += 0.9 { // roof beams sticking out of the front
			parts = append(parts, sc.box(0.12, 0.12, 0.5, bx, 2.45, front+0.2, timber))
		}
	case tiles.Temple:
		parts = []r3d.Object{
			sc.box(w-0.2, 0.4, d-0.2, w/2, 0, d/2, sandLit),
			sc.box(w-0.2, 0.45, d-0.2, w/2, 3.4, d/2, sandLit),
		}
		parts = append(parts, sc.gable(w-0.2, d-0.2, w/2, 3.85, d/2, sand, wall(sandLit.tint))...) // the fill is turned 45°: no blocks
		for _, c := range [][2]float32{{0.6, 0.6}, {w - 0.6, 0.6}, {0.6, d - 0.6}, {w - 0.6, d - 0.6}} {
			parts = append(parts, sc.shape(rl.GenMeshCylinder(0.25, 3.0, 10), c[0], 0.4, c[1], sand))
		}
	case tiles.Pyramid, tiles.PyramidSmall:
		a := min(w, d) * 0.95
		parts = []r3d.Object{sc.pyramid(a, a*0.8, w/2, 0, d/2, sand)}
	case tiles.GothicHouse:
		parts = sc.gothicHouse(w, d)
	case tiles.Crypt:
		parts = sc.crypt(w, d)
	case tiles.WhiteHouse:
		parts = sc.whiteHouse(w, d)
	case tiles.Villa:
		parts = sc.house(w, d, white, roof(color.RGBA{196, 98, 60, 255}), 3.0, false, seaBlue)
		front := 0.15 + (d - 0.6)
		doorX := float32(int(w)/2) + 0.5
		for _, x := range []float32{doorX - 1.1, doorX + 1.1} { // a porch over the door, on two white columns
			parts = append(parts, sc.shape(rl.GenMeshCylinder(0.14, 2.4, 10), x, 0.3, front+0.9, white))
		}
		parts = append(parts, sc.box(2.8, 0.18, 1.1, doorX, 2.7, front+0.5, white), sc.box(2.9, 0.1, 1.2, doorX, 2.88, front+0.5,
			roof(color.RGBA{196, 98, 60, 255})))
	case tiles.Longhouse:
		parts = sc.cabin(w, d, 2.2, false)
	case tiles.NordHut:
		parts = sc.cabin(w, d, 1.9, true)
	case tiles.Obelisk:
		parts = []r3d.Object{
			sc.box(0.55, 3.4, 0.55, w/2, 0, d-0.5, sandLit),
			sc.pyramid(0.55, 0.5, w/2, 3.4, d-0.5, gold),
		}
	}
	sc.buildings[id] = parts
	return parts
}
