package game

import (
	"fmt"
	"image"
	"image/color"
	"math/rand/v2"
	"strconv"

	rl "github.com/gen2brain/raylib-go/raylib"

	"oldworld2/internal/battle"
	"oldworld2/internal/character"
	"oldworld2/internal/gfx"
	"oldworld2/internal/procgen"
	"oldworld2/internal/world"
)

// The title screen: continue the saved game, or set up a new one (class and world seed).

type titlePage int

const (
	pageMain titlePage = iota
	pageSetup
	pageConfirm // a new game would replace the save
)

// Rows of the new-game setup page.
const (
	rowClass = iota
	rowZone
	rowSeed
	rowStart
	rowBack
	numRows
)

const maxSeedDigits = 18

var classes = func() []character.Class {
	var all []character.Class
	for c := character.Class(0); c < character.NumClasses; c++ {
		all = append(all, c)
	}
	return all
}()

type titleScreen struct {
	g        *Game
	page     titlePage
	cursor   int // main page item, or setup row
	hasSave  bool
	class    int             // index into classes
	zone     int             // index into procgen.Implemented: where the new game starts
	seed     string          // digits typed by the player; empty means a random world
	preview  image.Rectangle // where the 3D class preview goes (canvas pixels), empty if none
	logo     *gfx.Image      // the gilded title (title_style.go)
	vignette *gfx.Image
	note     string // a short message, e.g. after a failed load
}

func newTitle(g *Game) *titleScreen {
	t := &titleScreen{g: g}
	if g.opts.Store != nil {
		_, err := g.opts.Store.Load()
		t.hasSave = err == nil
	}
	return t
}

func (t *titleScreen) mainItems() []string {
	if t.hasSave {
		return []string{"Continue", "New game", "Quit"}
	}
	return []string{"New game", "Quit"}
}

func (t *titleScreen) update() {
	g := t.g
	switch t.page {
	case pageMain:
		items := t.mainItems()
		switch {
		case g.menuPressed(world.Up):
			t.cursor = (t.cursor + len(items) - 1) % len(items)
		case g.menuPressed(world.Down):
			t.cursor = (t.cursor + 1) % len(items)
		case g.menuConfirm():
			switch items[t.cursor] {
			case "Continue":
				g.startLoading(func() error { return g.begin(false) })
			case "New game":
				t.page, t.cursor = pageSetup, rowClass
			case "Quit":
				g.done = true
			}
		}
	case pageSetup:
		t.typeSeed()
		switch {
		case g.backPressed():
			t.page, t.cursor = pageMain, 0
		case g.menuPressed(world.Up):
			t.cursor = (t.cursor + numRows - 1) % numRows
		case g.menuPressed(world.Down):
			t.cursor = (t.cursor + 1) % numRows
		case t.cursor == rowClass && g.menuPressed(world.Left):
			t.class = (t.class + len(classes) - 1) % len(classes)
		case t.cursor == rowClass && g.menuPressed(world.Right):
			t.class = (t.class + 1) % len(classes)
		case t.cursor == rowZone && g.menuPressed(world.Left):
			t.zone = (t.zone + len(procgen.Implemented) - 1) % len(procgen.Implemented)
		case t.cursor == rowZone && g.menuPressed(world.Right):
			t.zone = (t.zone + 1) % len(procgen.Implemented)
		case g.menuConfirm():
			switch t.cursor {
			case rowClass:
				t.class = (t.class + 1) % len(classes)
			case rowZone:
				t.zone = (t.zone + 1) % len(procgen.Implemented)
			case rowStart:
				if t.hasSave {
					t.page = pageConfirm
				} else {
					t.start()
				}
			case rowBack:
				t.page, t.cursor = pageMain, 0
			}
		}
	case pageConfirm:
		switch {
		case g.backPressed():
			t.page = pageSetup
		case g.menuConfirm():
			t.start()
		}
	}
}

// typeSeed adds typed digits to the seed and handles Backspace.
func (t *titleScreen) typeSeed() {
	for _, r := range typedChars() {
		if r >= '0' && r <= '9' && len(t.seed) < maxSeedDigits {
			t.seed += string(r)
			t.cursor = rowSeed
		}
	}
	if t.cursor == rowSeed && len(t.seed) > 0 && repeatPressed(rl.KeyBackspace) {
		t.seed = t.seed[:len(t.seed)-1]
	}
}

// repeatPressed reports a key press, repeating while the key is held.
func repeatPressed(k int32) bool {
	d := heldKeys[k]
	return d == 1 || (d > 20 && d%4 == 0)
}

// start begins a new game with the chosen class, start zone and seed (random if none was
// typed).
func (t *titleScreen) start() {
	g := t.g
	seed := rand.Int64N(1_000_000)
	if t.seed != "" {
		if s, err := strconv.ParseInt(t.seed, 10, 64); err == nil {
			seed = s
		}
	}
	g.opts.Seed, g.opts.Class, g.opts.Level, g.opts.StartAt = seed, classes[t.class], 1, nil
	g.opts.StartZone = procgen.Implemented[t.zone]
	g.startLoading(func() error { return g.begin(true) })
}

func (t *titleScreen) draw(dst *gfx.Image) {
	g := t.g
	vw, vh := dst.Bounds().Dx(), dst.Bounds().Dy()
	t.preview = image.Rectangle{}

	// Background: the 3D world drawn beneath (scene.drawBackdrop), warmed and framed, with
	// the logo over it (title_style.go).
	t.drawDressing(dst, vw, vh, t.page == pageMain)

	switch t.page {
	case pageMain:
		items := t.mainItems()
		w, h := 110, len(items)*LineHeight+14
		x, y := (vw-w)/2, vh/2
		fancyPanel(dst, x, y, w, h)
		for i, it := range items {
			cy := y + 8 + i*LineHeight
			if i == t.cursor {
				gem(dst, x+9, cy+1, gilt)
			}
			g.font.Draw(dst, it, x+20, cy, ink)
		}
	case pageSetup:
		t.drawSetup(dst, vw, vh)
	case pageConfirm:
		t.drawSetup(dst, vw, vh)
		t.preview = image.Rectangle{} // the hero would stand over the question
		lines := []string{"This replaces your saved game.", "Space: start   B: back"}
		w := g.font.Width(lines[0]) + 20
		h := len(lines)*LineHeight + 14
		x, y := (vw-w)/2, (vh-h)/2
		fancyPanel(dst, x, y, w, h)
		for i, l := range lines {
			g.font.Draw(dst, l, x+10, y+8+i*LineHeight, ink)
		}
	}
	if t.note != "" {
		w := g.font.Width(t.note) + 16
		fancyPanel(dst, (vw-w)/2, vh-LineHeight-16, w, LineHeight+12)
		g.font.Draw(dst, t.note, (vw-w)/2+8, vh-LineHeight-10, color.RGBA{170, 50, 40, 255})
	}
}

func (t *titleScreen) drawSetup(dst *gfx.Image, vw, vh int) {
	g := t.g
	// The settings on the left, the class's hero standing in a frame on the right (drawn in 3D
	// over the canvas, see Game.Draw).
	const previewW = 68
	w, h := min(vw-previewW-12, 244), 100+LineHeight
	x, y := max(4, (vw-w-previewW-4)/2), vh/2-28
	fancyPanel(dst, x, y, w, h)
	t.preview = image.Rect(x+w+4, y-14, x+w+4+previewW, y+h)
	fancyPanel(dst, t.preview.Min.X, t.preview.Min.Y, previewW, t.preview.Dy())
	class := classes[t.class]
	info := class.Info()

	row := func(i int, label, value string) {
		cy := y + 8 + i*LineHeight
		if t.cursor == i {
			gem(dst, x+9, cy+1, gilt)
		}
		g.font.Draw(dst, label, x+18, cy, ink)
		g.font.Draw(dst, value, x+60, cy, ink)
	}
	row(rowClass, "Class", "< "+info.Name+" >")
	row(rowZone, "Zone", "< "+procgen.Implemented[t.zone].String()+" >")
	seed := t.seed
	switch {
	case seed == "" && t.cursor == rowSeed:
		seed = "_  (digits, empty = random)"
	case seed == "":
		seed = "random"
	case t.cursor == rowSeed && (g.frame/30)%2 == 0:
		seed += "_"
	}
	row(rowSeed, "Seed", seed)
	row(rowStart, "Start", "")
	row(rowBack, "Back", "")

	// The class's affinities and starting abilities.
	var types string
	for i, e := range info.Affinity {
		if i > 0 {
			types += ", "
		}
		types += e.String()
	}
	var start string
	for _, id := range info.Start {
		if a := battle.Get(id); a != nil && !a.Fallback {
			if start != "" {
				start += ", "
			}
			start += a.Name
		}
	}
	dy := y + 8 + numRows*LineHeight + 4
	gfx.FillRect(dst, float32(x+8), float32(dy-3), float32(w-16), 1, inkFaded)
	g.font.Draw(dst, info.Description, x+10, dy, ink)
	g.font.Draw(dst, fmt.Sprintf("Affinity: %s", types), x+10, dy+LineHeight, inkFaded)
	g.font.Draw(dst, fmt.Sprintf("Starts with: %s", start), x+10, dy+2*LineHeight, inkFaded)
}

// startLoading shows a loading screen for one frame, then runs f (which may take a moment
// while the zone is prepared).
func (g *Game) startLoading(f func() error) {
	g.loading = f
}

func drawLoading(g *Game, dst *gfx.Image) {
	vw, vh := dst.Bounds().Dx(), dst.Bounds().Dy()
	dst.Fill(color.RGBA{30, 40, 30, 255})
	msg := "Loading..."
	g.font.Draw(dst, msg, (vw-g.font.Width(msg))/2, vh/2-4, color.RGBA{255, 240, 200, 255})
}
