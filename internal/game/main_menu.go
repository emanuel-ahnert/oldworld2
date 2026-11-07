package game

import (
	"image/color"

	rl "github.com/gen2brain/raylib-go/raylib"

	"oldworld2/internal/gfx"
	"oldworld2/internal/world"
)

// The main menu (Escape, gamepad X) on the adventure map.

type menuItem int

const (
	itemResume menuItem = iota
	itemAbilities
	itemControls
	itemTitle
	itemQuit
)

var menuLabels = map[menuItem]string{
	itemResume:    "Resume",
	itemAbilities: "Abilities",
	itemControls:  "Controls",
	itemTitle:     "Save and return to title",
	itemQuit:      "Save and quit",
}

var controlsHelp = [][2]string{
	{"WASD / arrows", "Move, menus"},
	{"Shift (hold)", "Run"},
	{"Space", "Interact, confirm"},
	{"C", "Camp: rest at a fire"},
	{"N", "Abilities menu"},
	{"F3", "Comic / pixel map"},
	{"B / Escape", "Back"},
	{"Escape", "Main menu"},
	{"Gamepad", "D-pad, A, B, X = Esc, Y = run, Select = N, LB = C"},
}

type mainMenu struct {
	g        *Game
	cursor   int
	controls bool // showing the controls
}

func (g *Game) menuKeyPressed() bool {
	if g.opts.Script != nil {
		s, start := scriptStep(g.opts.Script, g.frame)
		return s.Esc && start
	}
	return keyPressed(rl.KeyEscape) || padPressed(rl.GamepadButtonRightFaceLeft) // X
}

// update handles one frame of input. It returns false when the menu closes.
func (m *mainMenu) update() bool {
	g := m.g
	if g.opts.Script != nil {
		return g.frame%240 != 0 // scripted runs just look at the menu
	}
	if m.controls {
		if g.backPressed() || g.menuConfirm() {
			m.controls = false
		}
		return true
	}
	if g.backPressed() {
		return false
	}
	switch {
	case g.menuPressed(world.Up):
		m.cursor = (m.cursor + len(menuLabels) - 1) % len(menuLabels)
	case g.menuPressed(world.Down):
		m.cursor = (m.cursor + 1) % len(menuLabels)
	case g.menuConfirm():
		switch menuItem(m.cursor) {
		case itemResume:
			return false
		case itemAbilities:
			g.abilities = &abilitiesMenu{g: g}
			return false
		case itemControls:
			m.controls = true
		case itemTitle:
			g.returnToTitle()
			return false
		case itemQuit:
			g.done = true // Update ends the game; the caller saves on exit
			return false
		}
	}
	return true
}

func (m *mainMenu) draw(dst *gfx.Image) {
	g := m.g
	vw, vh := dst.Bounds().Dx(), dst.Bounds().Dy()
	gfx.FillRect(dst, 0, 0, float32(vw), float32(vh), color.RGBA{0, 0, 0, 110})

	switch {
	case m.controls:
		w, h := min(vw-16, 300), len(controlsHelp)*LineHeight+30
		x, y := (vw-w)/2, (vh-h)/2
		panel(dst, x, y, w, h)
		g.font.Draw(dst, "Controls", x+8, y+7, ink)
		for i, c := range controlsHelp {
			g.font.Draw(dst, c[0], x+10, y+22+i*LineHeight, ink)
			g.font.Draw(dst, c[1], x+90, y+22+i*LineHeight, inkFaded)
		}
	default:
		w, h := 160, len(menuLabels)*LineHeight+26
		x, y := (vw-w)/2, (vh-h)/2
		panel(dst, x, y, w, h)
		g.font.Draw(dst, "Menu", x+8, y+7, ink)
		for i := 0; i < len(menuLabels); i++ {
			cy := y + 21 + i*LineHeight
			if i == m.cursor {
				g.font.Draw(dst, ">", x+10, cy, ink)
			}
			g.font.Draw(dst, menuLabels[menuItem(i)], x+20, cy, ink)
		}
	}
}
