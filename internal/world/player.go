package world

import "oldworld2/internal/tiles"

// Dir is a facing direction.
type Dir int

const (
	Down Dir = iota
	Up
	Left
	Right
)

// Delta returns the tile offset of one step in a direction.
func (d Dir) Delta() (dx, dy int) {
	switch d {
	case Up:
		return 0, -1
	case Left:
		return -1, 0
	case Right:
		return 1, 0
	}
	return 0, 1
}

// Movement tuning (see design/mechanics.md, Aggro & Movement). Times are in frames at 60 TPS.
const (
	WalkFrames = 12 // 5 tiles/s
	RunFrames  = 6
	// Steps onto a road are 20% faster.
	RoadWalkFrames = 10
	RoadRunFrames  = 5
	TurnFrames     = 6 // a tap shorter than this only turns the player in place

	StaminaMax         = 10.0 // seconds of running
	staminaRefillTime  = 20.0 // seconds to refill from empty while moving
	standingRefillRate = 2.0  // refill is twice as fast while standing still
)

// Input is the movement input for one frame.
type Input struct {
	Dir    Dir
	HasDir bool
	Run    bool
}

// Player moves tile by tile. A step always finishes on a tile.
type Player struct {
	TX, TY     int // current tile; the destination tile while a step is in progress
	Facing     Dir
	Stamina    float64
	StaminaUse float64 // how fast running uses stamina: 1 normally (0 also means 1), less with a perk

	moving     bool
	running    bool
	fromX      int
	fromY      int
	progress   int // frames into the current step
	stepFrames int
	turnWait   int  // frames left before a held direction starts walking after a turn
	walked     bool // a step finished last frame, so direction changes don't need a turn pause
	distance   int  // pixels walked in total, drives the walk animation
}

// NewPlayer places a player on a tile, facing down, with full stamina.
func NewPlayer(tx, ty int) *Player {
	return &Player{TX: tx, TY: ty, Facing: Down, Stamina: StaminaMax}
}

// Moving reports whether a step is in progress.
func (p *Player) Moving() bool { return p.moving }

// Running reports whether the current step is a running step.
func (p *Player) Running() bool { return p.moving && p.running }

// StepFrames returns how many frames the current step takes.
func (p *Player) StepFrames() int { return p.stepFrames }

// Update advances the player by one frame. road reports road tiles, which are walked faster
// (nil: no roads).
func (p *Player) Update(in Input, blocked, road func(tx, ty int) bool) {
	p.updateStamina(in)

	if p.moving {
		p.progress++
		p.distance += tiles.Size / p.stepFrames
		if p.progress < p.stepFrames {
			return
		}
		p.moving = false
		p.walked = true
		// Fall through so a held direction continues without a pause.
	}

	if !in.HasDir {
		p.turnWait = 0
		p.walked = false
		return
	}

	if in.Dir != p.Facing {
		p.Facing = in.Dir
		if !p.walked {
			// Counting this frame, walking starts once the direction is held for TurnFrames.
			p.turnWait = TurnFrames - 1
			return
		}
	}
	if p.turnWait > 0 {
		p.turnWait--
		if p.turnWait > 0 {
			return
		}
	}

	dx, dy := in.Dir.Delta()
	if blocked(p.TX+dx, p.TY+dy) {
		p.walked = false
		return
	}
	p.fromX, p.fromY = p.TX, p.TY
	p.TX, p.TY = p.TX+dx, p.TY+dy
	p.moving = true
	p.progress = 0
	p.running = in.Run && p.Stamina > 0
	onRoad := road != nil && road(p.TX, p.TY)
	switch {
	case p.running && onRoad:
		p.stepFrames = RoadRunFrames
	case p.running:
		p.stepFrames = RunFrames
	case onRoad:
		p.stepFrames = RoadWalkFrames
	default:
		p.stepFrames = WalkFrames
	}
	p.walked = false
}

func (p *Player) updateStamina(in Input) {
	const frame = 1.0 / 60
	switch {
	case p.Running():
		use := p.StaminaUse
		if use == 0 {
			use = 1
		}
		p.Stamina -= frame * use
	case p.moving || in.HasDir:
		p.Stamina += frame * StaminaMax / staminaRefillTime
	default:
		p.Stamina += frame * StaminaMax / staminaRefillTime * standingRefillRate
	}
	p.Stamina = min(max(p.Stamina, 0), StaminaMax)
}

// PixelPos returns the player's position in world pixels, interpolated during a step.
func (p *Player) PixelPos() (x, y float64) {
	if !p.moving {
		return float64(p.TX * tiles.Size), float64(p.TY * tiles.Size)
	}
	t := float64(p.progress) / float64(p.stepFrames)
	x = (float64(p.fromX) + float64(p.TX-p.fromX)*t) * tiles.Size
	y = (float64(p.fromY) + float64(p.TY-p.fromY)*t) * tiles.Size
	return x, y
}
