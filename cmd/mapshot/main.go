// Command mapshot renders a part of the generated world to a PNG file without opening a window,
// as a colored map: every tile a square colored by its ground, objects over their footprints,
// enemies as dots and the start point as a cross. Useful to check world generation quickly.
// With -list it also prints where each enemy stands, e.g. to start a test run next to one.
package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"

	"oldworld2/internal/creature"
	"oldworld2/internal/procgen"
	"oldworld2/internal/tiles"
	"oldworld2/internal/world"
)

func main() {
	seed := flag.Int64("seed", 1, "world seed")
	x := flag.Int("x", -48, "left tile")
	y := flag.Int("y", -32, "top tile")
	w := flag.Int("w", 96, "width in tiles")
	h := flag.Int("h", 64, "height in tiles")
	zoom := flag.Int("zoom", 4, "pixels per tile (1 for a minimap of whole zones)")
	out := flag.String("out", "map.png", "output PNG")
	list := flag.Bool("list", false, "also print every enemy in the area: tile and kind")
	flag.Parse()

	if err := run(*seed, *x, *y, *w, *h, max(1, *zoom), *out, *list); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

var (
	grass     = color.RGBA{150, 190, 80, 255}
	water     = color.RGBA{80, 150, 210, 255}
	road      = color.RGBA{185, 135, 90, 255}
	sand      = color.RGBA{235, 200, 130, 255}
	sandPath  = color.RGBA{210, 170, 110, 255}
	snow      = color.RGBA{236, 240, 246, 255}
	dryGrass  = color.RGBA{190, 185, 100, 255}
	deadGrass = color.RGBA{96, 88, 98, 255}
	ice       = color.RGBA{160, 200, 225, 255}
	tree      = color.RGBA{40, 100, 50, 255}
	palm      = color.RGBA{70, 140, 60, 255}
	bush      = color.RGBA{80, 140, 60, 255}
	flower    = color.RGBA{240, 200, 80, 255}
	rock      = color.RGBA{140, 135, 125, 255}
	bone      = color.RGBA{240, 235, 215, 255}
	house     = color.RGBA{200, 90, 60, 255}
	monument  = color.RGBA{215, 175, 100, 255}
	enemy     = color.RGBA{200, 30, 40, 255}
	start     = color.RGBA{255, 255, 255, 255}
)

func groundColor(id tiles.ID) color.RGBA {
	switch {
	case tiles.IsIce(id):
		return ice
	case tiles.IsWater(id):
		return water
	case id >= tiles.SandPath && id <= tiles.SandPathInnerSW:
		return sandPath
	case tiles.IsRoad(id):
		return road
	case tiles.IsSand(id):
		return sand
	case tiles.IsSnow(id):
		return snow
	case tiles.IsDryGrass(id):
		return dryGrass
	case tiles.IsDeadGrass(id):
		return deadGrass
	}
	return grass
}

// objectColor is the color of an object, and whether it fills its footprint (trees, rocks,
// buildings) or is a small dot in its tile (flowers, plants, bones).
func objectColor(id tiles.ID) (c color.RGBA, fill bool) {
	switch id {
	case tiles.TreeRound, tiles.TreePine, tiles.TreeRoots, tiles.SnowPine, tiles.SnowPineB, tiles.OliveTree, tiles.Cypress, tiles.DeadTree, tiles.DarkPine:
		return tree, true
	case tiles.Palm, tiles.PalmB:
		return palm, true
	case tiles.Bush:
		return bush, true
	case tiles.Boulder, tiles.BoulderB, tiles.RockStump, tiles.SmallRock, tiles.SnowRock, tiles.LimeRock, tiles.DarkRock:
		return rock, true
	case tiles.Skull, tiles.SkullB, tiles.Ribs, tiles.Bone:
		return bone, false
	case tiles.Pyramid, tiles.PyramidSmall, tiles.Obelisk, tiles.Sphinx, tiles.PharaohStatue, tiles.Temple:
		return monument, true
	}
	if tiles.IsBuilding(id) {
		return house, true
	}
	if id == tiles.FlowerSun || id == tiles.FlowerOrange || id == tiles.FlowerRed || id == tiles.FlowerWhite {
		return flower, false
	}
	return bush, false
}

func run(seed int64, x0, y0, w, h, zoom int, out string, list bool) error {
	wld := world.New(seed)
	img := image.NewRGBA(image.Rect(0, 0, w*zoom, h*zoom))
	rect := func(tx, ty int, inset int, c color.RGBA) {
		if tx < x0 || ty < y0 || tx >= x0+w || ty >= y0+h {
			return
		}
		for py := inset; py < zoom-inset; py++ {
			for px := inset; px < zoom-inset; px++ {
				img.SetRGBA((tx-x0)*zoom+px, (ty-y0)*zoom+py, c)
			}
		}
	}
	for ty := y0; ty < y0+h; ty++ {
		for tx := x0; tx < x0+w; tx++ {
			cx, cy := world.ChunkCoord(tx, ty)
			rect(tx, ty, 0, groundColor(wld.Chunk(cx, cy).Ground[procgen.Index(tx-cx*procgen.ChunkSize, ty-cy*procgen.ChunkSize)]))
		}
	}
	cx0, cy0 := world.ChunkCoord(x0-6, y0-6)
	cx1, cy1 := world.ChunkCoord(x0+w, y0+h)
	dot := max(0, zoom/2-1)
	for cy := cy0; cy <= cy1; cy++ {
		for cx := cx0; cx <= cx1; cx++ {
			for _, o := range wld.Chunk(cx, cy).Objects {
				c, fill := objectColor(o.Sprite)
				f := tiles.Get(o.Sprite)
				if !fill {
					rect(o.X, o.Y, dot, c)
					continue
				}
				for ty := o.Y; ty < o.Y+f.H; ty++ {
					for tx := o.X; tx < o.X+f.W; tx++ {
						rect(tx, ty, 0, c)
					}
				}
			}
		}
	}
	for _, n := range wld.NPCs() {
		rect(n.TX, n.TY, dot/2, enemy)
		if list && n.TX >= x0 && n.TY >= y0 && n.TX < x0+w && n.TY < y0+h {
			fmt.Printf("%d,%d %s\n", n.TX, n.TY, creature.Get(n.Spawn.Kind).Name)
		}
	}
	sx, sy := wld.FindSpawn()
	for d := -2; d <= 2; d++ {
		rect(sx+d, sy, 0, start)
		rect(sx, sy+d, 0, start)
	}
	f, err := os.Create(out)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}
