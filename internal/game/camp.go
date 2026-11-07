package game

import (
	rl "github.com/gen2brain/raylib-go/raylib"

	"oldworld2/internal/world"
)

// Camp mode: press C to light a camp fire and rest. HP regenerates at 10% per second until
// the player moves again (see design/mechanics.md, HP & MP).

// camp is an active camp fire.
type camp struct {
	x, y int // the fire's tile
}

func (g *Game) campPressed() bool {
	if g.opts.Script != nil {
		return scriptCamp(g.opts.Script, g.frame)
	}
	return keyPressed(rl.KeyC) || padPressed(rl.GamepadButtonLeftTrigger1) // LB
}

// startCamp lights a fire next to the player: in front if possible, otherwise on any free
// neighboring tile. It does nothing if there's no room or the player is walking.
func (g *Game) startCamp() {
	p := g.player
	if p.Moving() {
		return
	}
	dirs := []world.Dir{p.Facing, world.Down, world.Left, world.Right, world.Up}
	for _, d := range dirs {
		dx, dy := d.Delta()
		x, y := p.TX+dx, p.TY+dy
		if !g.world.BlockedForPlayer(x, y) {
			p.Facing = d
			g.camp = &camp{x, y}
			g.sound(sndLight)
			return
		}
	}
}
