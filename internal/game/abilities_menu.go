package game

import (
	"fmt"
	"image/color"
	"slices"

	rl "github.com/gen2brain/raylib-go/raylib"

	"oldworld2/internal/battle"
	"oldworld2/internal/character"
	"oldworld2/internal/gfx"
	"oldworld2/internal/world"
)

// The abilities menu (key N, gamepad Select): choose which 4 abilities are active in battle
// (see design/classes.md, Ability Loadout). Only available on the adventure map.

type abilitiesMenu struct {
	g       *Game
	column  int    // 0 = active slots, 1 = known abilities
	row     int    // cursor row in the current column
	pending string // ability waiting for a slot to replace, when the slots are full
}

func (g *Game) abilitiesPressed() bool {
	if g.opts.Script != nil {
		s, start := scriptStep(g.opts.Script, g.frame)
		return s.Menu && start
	}
	return keyPressed(rl.KeyN) || padPressed(rl.GamepadButtonMiddleLeft) // Select
}

func (g *Game) backPressed() bool {
	if keyPressed(rl.KeyB, rl.KeyEscape) || padPressed(rl.GamepadButtonRightFaceRight, rl.GamepadButtonRightFaceLeft) { // B or X
		g.sound(sndCancel)
		return true
	}
	return false
}

// known lists the abilities shown on the right: everything known that isn't in a slot.
// Passives and the basic move are listed too, but can't be selected for a slot.
func (m *abilitiesMenu) known() []string {
	c := m.g.char
	var out []string
	for _, id := range c.Known {
		if !slices.Contains(c.Active, id) {
			out = append(out, id)
		}
	}
	return out
}

// columnLen is the number of selectable rows: the 4 slots, or the known list.
func (m *abilitiesMenu) columnLen() int {
	if m.column == 0 {
		return character.MaxActive
	}
	return len(m.known())
}

// selected returns the ability under the cursor, or "" for an empty slot.
func (m *abilitiesMenu) selected() string {
	if m.column == 0 {
		if m.row < len(m.g.char.Active) {
			return m.g.char.Active[m.row]
		}
		return ""
	}
	if k := m.known(); m.row < len(k) {
		return k[m.row]
	}
	return ""
}

// update handles one frame of input. It returns false when the menu closes.
func (m *abilitiesMenu) update() bool {
	g := m.g
	if g.opts.Script != nil {
		return !(g.frame%240 == 0) // scripted runs just look at the menu
	}
	if g.abilitiesPressed() || g.backPressed() {
		if m.pending != "" && g.backPressed() {
			m.pending = ""
			return true
		}
		return false
	}
	switch {
	case g.menuPressed(world.Up):
		m.row = max(0, m.row-1)
	case g.menuPressed(world.Down):
		m.row = min(max(0, m.columnLen()-1), m.row+1)
	case g.menuPressed(world.Left), g.menuPressed(world.Right):
		if m.pending == "" {
			m.column = 1 - m.column
			m.row = min(m.row, max(0, m.columnLen()-1))
		}
	case g.menuConfirm():
		m.confirm()
	}
	return true
}

func (m *abilitiesMenu) confirm() {
	c := m.g.char
	id := m.selected()
	switch {
	case m.pending != "" && m.column == 0:
		if m.row < len(c.Active) {
			c.Replace(m.row, m.pending)
		} else {
			c.Equip(m.pending)
		}
		m.pending = ""
	case m.column == 0 && id != "":
		c.Unequip(m.row)
	case m.column == 1 && c.Slottable(id):
		if !c.Equip(id) {
			// Slots full: choose the slot to replace.
			m.pending = id
			m.column, m.row = 0, 0
		}
		m.row = min(m.row, max(0, m.columnLen()-1))
	}
}

func (m *abilitiesMenu) draw(dst *gfx.Image) {
	g, c := m.g, m.g.char
	vw, vh := dst.Bounds().Dx(), dst.Bounds().Dy()
	gfx.FillRect(dst, 0, 0, float32(vw), float32(vh), color.RGBA{0, 0, 0, 110})

	w, h := min(vw-16, 300), min(vh-16, 164)
	x, y := (vw-w)/2, (vh-h)/2
	panel(dst, x, y, w, h)
	g.font.Draw(dst, "Abilities", x+8, y+7, ink)
	hint := "Space: change  N: close"
	if m.pending != "" {
		hint = "Choose a slot to replace  B: cancel"
	}
	g.font.Draw(dst, hint, x+w-8-g.font.Width(hint), y+7, inkFaded)

	colW := (w - 24) / 2
	lx, rx, top := x+12, x+16+colW, y+24
	g.font.Draw(dst, fmt.Sprintf("Active (%d/%d)", len(c.Active), character.MaxActive), lx, top, ink)
	g.font.Draw(dst, "Known", rx, top, ink)
	gfx.FillRect(dst, float32(rx-6), float32(top), 1, float32(character.MaxActive+3)*LineHeight, inkFaded)

	cursor := func(col, row, cx, cy int) {
		if m.column == col && m.row == row {
			g.font.Draw(dst, ">", cx-8, cy, ink)
		}
	}
	for i := 0; i < character.MaxActive; i++ {
		cy := top + (i+1)*LineHeight + 2
		cursor(0, i, lx+8, cy)
		label, col := "-", inkFaded
		if i < len(c.Active) {
			label, col = battle.Get(c.Active[i]).Name, ink
		}
		g.font.Draw(dst, label, lx+8, cy, col)
	}
	for i, id := range m.known() {
		cy := top + (i+1)*LineHeight + 2
		cursor(1, i, rx+8, cy)
		a := battle.Get(id)
		label, col := a.Name, ink
		switch {
		case a.Passive:
			label, col = a.Name+" (passive)", inkFaded
		case a.Fallback:
			label, col = a.Name+" (always ready)", inkFaded
		case id == m.pending:
			label = a.Name + " *"
		}
		g.font.Draw(dst, label, rx+8, cy, col)
	}

	// Details of the selected ability.
	dy := y + h - 3*LineHeight - 10
	gfx.FillRect(dst, float32(x+8), float32(dy-4), float32(w-16), 1, inkFaded)
	id := m.selected()
	if m.pending != "" {
		id = m.pending
	}
	if a := battle.Get(id); a != nil {
		var head string
		if a.Power > 0 {
			head = fmt.Sprintf("%s  Power %d  ", a.Elements(), a.Power)
		}
		switch {
		case a.Passive:
			head += "Passive: always on"
		default:
			head += costText(a, c.Class.Info().Resource)
		}
		if a.Priority > 0 {
			head += fmt.Sprintf("  Priority +%d", a.Priority)
		}
		g.font.Draw(dst, a.Name, x+12, dy, ink)
		g.font.Draw(dst, head, x+12+g.font.Width(a.Name)+10, dy, inkFaded)
		for i, l := range g.font.Wrap(a.Text, w-24) {
			if i < 2 {
				g.font.Draw(dst, l, x+12, dy+(i+1)*LineHeight, ink)
			}
		}
	}
}
