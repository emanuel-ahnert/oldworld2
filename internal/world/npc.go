package world

import (
	"oldworld2/internal/creature"
	"oldworld2/internal/procgen"
	"oldworld2/internal/tiles"
)

type npcState int

const (
	idle npcState = iota
	chasing
	returning
)

const (
	wanderRadius  = 3  // idle enemies stay this close to home
	activeRadius  = 40 // enemies farther from the player than this don't move
	waitGap       = 2  // chasers that can't engage yet keep this distance (in steps)
	graceMinFrame = 60 // after a battle, no enemy can engage for 1–2 seconds
	graceMaxFrame = 120
)

// NPC is an enemy on the adventure map.
type NPC struct {
	Spawn  procgen.Spawn
	TX, TY int // current tile; the destination tile while a step is in progress
	Facing Dir
	Side   Dir // last horizontal facing, for side-view sprites

	state      npcState
	timer      int // frames: until the next wander step while idle, until giving up while chasing
	spotted    int // frames since the chase started
	moving     bool
	fromX      int
	fromY      int
	progress   int
	stepFrames int
	distance   int
	rng        uint64
}

func newNPC(s procgen.Spawn) *NPC {
	n := &NPC{Spawn: s, TX: s.X, TY: s.Y, Facing: Down, Side: Left, rng: s.ID | 1}
	n.timer = 30 + n.rand(120)
	return n
}

// Info returns the creature description.
func (n *NPC) Info() creature.Info { return creature.Get(n.Spawn.Kind) }

// Chasing reports whether the NPC is chasing the player.
// Moving reports whether the enemy is taking a step.
func (n *NPC) Moving() bool { return n.moving }

func (n *NPC) Chasing() bool { return n.state == chasing }

// JustSpotted reports whether the NPC noticed the player within the last moment.
func (n *NPC) JustSpotted() bool { return n.state == chasing && n.spotted < 45 }

// SpottedNow reports whether the NPC spotted the player in this frame's update.
func (n *NPC) SpottedNow() bool { return n.state == chasing && n.spotted == 0 }

// rand returns a pseudo-random number in [0, max) (xorshift; doesn't need to be stable).
func (n *NPC) rand(max int) int {
	n.rng ^= n.rng << 13
	n.rng ^= n.rng >> 7
	n.rng ^= n.rng << 17
	return int(n.rng % uint64(max))
}

// PixelPos returns the NPC's position in world pixels, interpolated during a step.
func (n *NPC) PixelPos() (x, y float64) {
	if !n.moving {
		return float64(n.TX * tiles.Size), float64(n.TY * tiles.Size)
	}
	t := float64(n.progress) / float64(n.stepFrames)
	return (float64(n.fromX) + float64(n.TX-n.fromX)*t) * tiles.Size,
		(float64(n.fromY) + float64(n.TY-n.fromY)*t) * tiles.Size
}

func (n *NPC) sees(p *Player) bool {
	dx, dy := float64(p.TX-n.TX), float64(p.TY-n.TY)
	r := n.Info().Aggro
	return dx*dx+dy*dy <= r*r
}

func adjacent(ax, ay, bx, by int) bool {
	return abs(ax-bx)+abs(ay-by) == 1
}

// UpdateNPCs moves the enemies near the player for one frame. It returns the NPC that
// caught the player, or nil. canEngage is false during the grace period after a battle.
func (w *World) UpdateNPCs(p *Player, canEngage bool) *NPC {
	var caught *NPC
	for _, n := range w.npcs {
		if abs(n.TX-p.TX) > activeRadius || abs(n.TY-p.TY) > activeRadius {
			continue
		}
		w.updateNPC(n, p, canEngage)
		if caught == nil && canEngage && !n.moving && n.state != returning && adjacent(n.TX, n.TY, p.TX, p.TY) {
			caught = n
		}
	}
	return caught
}

func (w *World) updateNPC(n *NPC, p *Player, canEngage bool) {
	// Timers run every frame, also during steps.
	if n.timer > 0 {
		n.timer--
	}
	if n.state == chasing {
		n.spotted++
	}
	if n.moving {
		n.progress++
		n.distance += max(1, tiles.Size/n.stepFrames)
		if n.progress < n.stepFrames {
			return
		}
		n.moving = false
	}

	switch n.state {
	case idle:
		if n.sees(p) {
			info := n.Info()
			n.state = chasing
			n.spotted = 0
			n.timer = (info.LeashMin + n.rand(info.LeashMax-info.LeashMin+1)) * 60
			return
		}
		if n.timer > 0 {
			return
		}
		n.timer = 60 + n.rand(150)
		d := Dir(n.rand(4))
		dx, dy := d.Delta()
		if abs(n.TX+dx-n.Spawn.X) <= wanderRadius && abs(n.TY+dy-n.Spawn.Y) <= wanderRadius {
			w.stepNPC(n, d, p)
		} else {
			n.face(d)
		}
	case chasing:
		if n.timer <= 0 {
			n.state = returning
			return
		}
		// While it can't engage yet (grace after a battle), it waits a tile further out
		// instead of standing nose to the player, and backs off if it's already closer.
		gap := 1
		if !canEngage {
			gap = waitGap
		}
		switch dist := abs(n.TX-p.TX) + abs(n.TY-p.TY); {
		case dist > gap:
			w.stepToward(n, p.TX, p.TY, p)
		case dist < gap && w.stepNPC(n, dirToward(p.TX, p.TY, n.TX, n.TY), p):
		default:
			n.face(dirToward(n.TX, n.TY, p.TX, p.TY))
		}
	case returning:
		if n.TX == n.Spawn.X && n.TY == n.Spawn.Y {
			n.state = idle
			n.timer = 60 + n.rand(150)
			return
		}
		if !w.stepToward(n, n.Spawn.X, n.Spawn.Y, p) {
			// Stuck on the way home: give up and stay here as the new home.
			n.Spawn.X, n.Spawn.Y = n.TX, n.TY
		}
	}
}

// stepToward takes a greedy step toward a target, trying the longer axis first.
func (w *World) stepToward(n *NPC, tx, ty int, p *Player) bool {
	dx, dy := tx-n.TX, ty-n.TY
	primary, secondary := axisDir(dx, 0), axisDir(0, dy)
	if abs(dy) > abs(dx) {
		primary, secondary = secondary, primary
	}
	for _, d := range []Dir{primary, secondary} {
		if d >= 0 && w.stepNPC(n, d, p) {
			return true
		}
	}
	return false
}

func (w *World) stepNPC(n *NPC, d Dir, p *Player) bool {
	n.face(d)
	dx, dy := d.Delta()
	x, y := n.TX+dx, n.TY+dy
	if w.Blocked(x, y) || w.npcAt(x, y) != nil || (x == p.TX && y == p.TY) {
		return false
	}
	n.fromX, n.fromY = n.TX, n.TY
	n.TX, n.TY = x, y
	n.moving = true
	n.progress = 0
	n.stepFrames = n.Info().StepFrames
	return true
}

func (n *NPC) face(d Dir) {
	n.Facing = d
	if d == Left || d == Right {
		n.Side = d
	}
}

// axisDir returns the direction of a single-axis offset, or -1 for no offset.
func axisDir(dx, dy int) Dir {
	switch {
	case dx > 0:
		return Right
	case dx < 0:
		return Left
	case dy > 0:
		return Down
	case dy < 0:
		return Up
	}
	return -1
}

func dirToward(ax, ay, bx, by int) Dir {
	if abs(bx-ax) >= abs(by-ay) {
		return axisDir(bx-ax, 0)
	}
	return axisDir(0, by-ay)
}

// npcAt returns the NPC standing on (or walking into) a tile.
func (w *World) npcAt(x, y int) *NPC {
	for _, n := range w.npcs {
		if n.TX == x && n.TY == y {
			return n
		}
	}
	return nil
}

// NPCs returns the enemies currently loaded.
func (w *World) NPCs() []*NPC {
	out := make([]*NPC, 0, len(w.npcs))
	for _, n := range w.npcs {
		out = append(out, n)
	}
	return out
}

// Defeat removes an enemy and remembers it, so it doesn't come back when its chunk reloads.
// (Enemy respawn is an open design question.)
func (w *World) Defeat(n *NPC) {
	w.defeated[n.Spawn.ID] = true
	delete(w.npcs, n.Spawn.ID)
}

// SetDefeated restores the defeated enemies from a save, before any chunk is generated.
func (w *World) SetDefeated(ids []uint64) {
	for _, id := range ids {
		w.defeated[id] = true
	}
}

// SendHome makes an enemy stop chasing and walk back to its home tile.
func (w *World) SendHome(n *NPC) {
	n.state = returning
}

// GraceFrames returns a random grace period length after a battle.
func GraceFrames(n *NPC) int {
	return graceMinFrame + n.rand(graceMaxFrame-graceMinFrame+1)
}
