package game

import (
	"container/heap"
	"math"
	"math/rand/v2"
	"slices"

	"oldworld2/internal/procgen"
	"oldworld2/internal/save"
	"oldworld2/internal/world"
)

// Demo mode: the game plays itself with a random class in a random zone, started from the
// title screen. A bot stands in for the keyboard: on the map it walks by simple rules (rest
// when hurt, fight enemies it can beat, otherwise explore, preferring roads), and in battle
// the planner picks its moves by playing each one out (battle.PlanPlayerMove). Escape ends
// the demo. Nothing is saved.

const (
	demoRunFrames  = 10 * 60 * 60 // a run starts over with a new class and zone after 10 minutes
	demoSight      = 24           // tiles: how far away the bot goes for an enemy
	demoSearch     = 64           // tiles: how far from the player a path may lead
	demoSearchCap  = 12000        // tiles a path search looks at, at most
	demoStuck      = 90           // frames without getting anywhere before it tries elsewhere
	demoIgnore     = 20 * 60      // frames an enemy it couldn't reach is left alone
	demoAvoid      = 5 * 60 * 60  // frames an enemy that beat it is left alone
	demoRestBelow  = 0.5          // rest when HP or MP fall below this share...
	demoRestUntil  = 0.95         // ...until both are back above this
	demoRunStamina = 0.4          // run only with this share of stamina left
	demoAwait      = 4 * 60       // frames it waits for a chasing enemy before walking to it
)

type demoBot struct {
	g       *Game
	rng     *rand.Rand
	store   *save.Store // the save, put aside while the demo runs
	started int         // g.frame when this run started

	goal    [2]int
	hasGoal bool
	heading float64    // direction of travel while exploring, so it doesn't turn back and forth
	target  *world.NPC // the enemy it is walking to
	ignore  map[uint64]int
	path    [][2]int
	replan  int // frames until the path is planned again

	last   [2]int // where the player stood
	still  int    // frames the player has stood there
	camp   bool   // press the camp key this frame
	noRest int    // frames left without trying to rest (no room for a fire)
	hurt   bool   // HP or MP are low: it fights only enemies already after it
	await  int    // frames it has waited for a chasing enemy to come
}

// startDemo starts a demo run: a new, unsaved game with a random class, zone and world.
func (g *Game) startDemo() {
	d := g.demo
	if d == nil {
		d = &demoBot{g: g, store: g.opts.Store, rng: rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64()))}
		g.demo = d
	}
	g.opts.Store = nil // nothing is saved while the demo runs
	g.opts.Seed = d.rng.Int64N(1_000_000)
	g.opts.Class = classes[d.rng.IntN(len(classes))]
	g.opts.StartZone = procgen.Implemented[d.rng.IntN(len(procgen.Implemented))]
	g.opts.Level, g.opts.StartAt = 1, nil
	*d = demoBot{g: g, store: d.store, rng: d.rng, started: g.frame, ignore: map[uint64]int{}, heading: d.rng.Float64() * 2 * math.Pi}
	g.startLoading(func() error { return g.begin(true) })
}

// endDemo stops the demo and goes back to the title screen with the real save.
func (g *Game) endDemo() {
	g.opts.Store = g.demo.store
	g.demo = nil
	g.battle, g.camp, g.message = nil, nil, nil
	g.world, g.player, g.char = nil, nil, nil
	g.title = newTitle(g)
}

// defeated is called when the bot lost a battle and woke up at the checkpoint: it forgets
// where it was going and leaves the enemy that beat it alone for a while.
func (d *demoBot) defeated(n *world.NPC) {
	d.ignore[n.Spawn.ID] = d.g.frame + demoAvoid
	d.hasGoal, d.target, d.path = false, nil, nil
	d.heading = d.rng.Float64() * 2 * math.Pi
}

// campPressed is the bot pressing the camp key.
func (d *demoBot) campPressed() bool {
	c := d.camp
	d.camp = false
	return c
}

// input decides the bot's movement for this frame on the map.
func (d *demoBot) input() world.Input {
	g, p := d.g, d.g.player
	st := g.char.Stats()
	hp, mp := g.char.HP/float64(st.MaxHP), g.char.MP/float64(max(1, st.MaxMP))
	if d.noRest > 0 {
		d.noRest--
	}
	d.ticks()

	// Rest at a camp fire when hurt, unless an enemy is already on the way.
	d.hurt = hp < demoRestBelow || mp < demoRestBelow
	if g.camp != nil {
		if hp < demoRestUntil || mp < demoRestUntil {
			return world.Input{}
		}
	} else if d.hurt && d.noRest == 0 && !d.threatened() {
		if !p.Moving() {
			d.camp = true
			if d.still > 30 { // the fire didn't light: no room here
				d.noRest = 300
			}
			return world.Input{}
		}
	}

	d.chooseGoal()
	if !d.hasGoal {
		return world.Input{}
	}
	// An enemy chasing the player comes by itself: the bot waits for it instead of walking
	// at it. Right after a battle, chasers keep their distance and back off, so walking at
	// them only goes around in circles. Only if it doesn't come (stuck behind something) does
	// the bot go to it.
	if d.target != nil && d.target.Chasing() && d.await < demoAwait {
		if g.grace == 0 {
			d.await++
		}
		d.still = 0 // waiting isn't being stuck
		return world.Input{}
	}
	if d.target == nil || !d.target.Chasing() {
		d.await = 0
	}
	if d.replan <= 0 || len(d.path) == 0 {
		d.plan()
	}
	for len(d.path) > 0 && d.path[0] == [2]int{p.TX, p.TY} {
		d.path = d.path[1:]
	}
	if len(d.path) == 0 {
		if d.target == nil {
			d.hasGoal = false // arrived
		}
		return world.Input{}
	}
	next := d.path[0]
	dx, dy := next[0]-p.TX, next[1]-p.TY
	if abs(dx)+abs(dy) != 1 { // off the path
		d.replan = 0
		return world.Input{}
	}
	if g.world.BlockedForPlayer(next[0], next[1]) && d.still > 20 {
		d.replan = 0 // an enemy stands in the way
	}
	run := p.Stamina > world.StaminaMax*demoRunStamina && len(d.path) > 6
	return world.Input{Dir: dirOf(dx, dy), HasDir: true, Run: run}
}

// ticks counts frames: how long the player has stood still and until the next path plan.
// Standing still too long while trying to walk means it's stuck: it gives up on the goal.
func (d *demoBot) ticks() {
	p := d.g.player
	if pos := [2]int{p.TX, p.TY}; pos != d.last || p.Moving() || d.g.camp != nil {
		d.last, d.still = pos, 0
	} else {
		d.still++
	}
	d.replan--
	if d.still > demoStuck && d.hasGoal {
		if d.target != nil {
			d.ignore[d.target.Spawn.ID] = d.g.frame + demoIgnore
		}
		d.hasGoal, d.target, d.path = false, nil, nil
		d.heading = d.rng.Float64() * 2 * math.Pi
		d.still = 0
	}
}

// threatened reports whether an enemy is chasing the player.
func (d *demoBot) threatened() bool {
	p := d.g.player
	for _, n := range d.g.world.NPCs() {
		if n.Chasing() && abs(n.TX-p.TX)+abs(n.TY-p.TY) < 12 {
			return true
		}
	}
	return false
}

// beatable reports whether the bot should pick a fight with an enemy: not one far above the
// player's level, and elites only at its own level or below.
func (d *demoBot) beatable(n *world.NPC) bool {
	lv := d.g.char.Level
	if n.Info().Tier == "elite" {
		return n.Spawn.Level <= lv
	}
	return n.Spawn.Level <= lv+1
}

// chooseGoal picks where to go: the nearest enemy it can beat, otherwise a spot to explore.
func (d *demoBot) chooseGoal() {
	g, p := d.g, d.g.player
	var best *world.NPC
	bestDist := demoSight + 1
	for _, n := range g.world.NPCs() {
		dist := abs(n.TX-p.TX) + abs(n.TY-p.TY)
		if dist >= bestDist || g.frame < d.ignore[n.Spawn.ID] || (!n.Chasing() && (d.hurt || !d.beatable(n))) {
			continue
		}
		best, bestDist = n, dist
	}
	if best != nil {
		if best != d.target {
			d.target, d.replan = best, 0
		}
		d.goal, d.hasGoal = [2]int{best.TX, best.TY}, true
		return
	}
	if d.target != nil { // the enemy is gone (or out of sight)
		d.target, d.hasGoal = nil, false
	}
	if !d.hasGoal {
		d.explore()
	}
}

// explore picks a spot 20 to 40 tiles ahead, roughly in the current heading: open ground,
// in a zone not too hard for the player's level, a road if there is one.
func (d *demoBot) explore() {
	g, p := d.g, d.g.player
	maxDifficulty := 1 + (g.char.Level-1)/5 // enemy levels rise by 5 per difficulty step
	bestScore := math.Inf(-1)
	for range 24 {
		a := d.heading + (d.rng.Float64()-0.5)*math.Pi*0.8
		r := 20 + d.rng.Float64()*20
		x, y := p.TX+int(math.Round(r*math.Cos(a))), p.TY+int(math.Round(r*math.Sin(a)))
		if g.world.Blocked(x, y) {
			continue
		}
		zx, zy := procgen.ZoneOf(x, y)
		score := d.rng.Float64()
		if g.world.Road(x, y) {
			score += 1
		}
		if g.world.Difficulty(zx, zy) > maxDifficulty {
			score -= 3
		}
		if score > bestScore {
			bestScore, d.goal, d.hasGoal = score, [2]int{x, y}, true
		}
	}
	if !d.hasGoal {
		d.heading += math.Pi / 2 // nothing open ahead: turn
		return
	}
	d.heading = math.Atan2(float64(d.goal[1]-p.TY), float64(d.goal[0]-p.TX))
	d.replan = 0
}

// plan finds a path to the goal (next to it, when the goal is an enemy).
func (d *demoBot) plan() {
	p := d.g.player
	d.replan = 60 // enemies move: plan again now and then
	if d.target == nil {
		d.replan = 600
	}
	d.path = findPath(d.g.world, [2]int{p.TX, p.TY}, d.goal, d.target != nil)
	if d.path == nil { // no way there
		if d.target != nil {
			d.ignore[d.target.Spawn.ID] = d.g.frame + demoIgnore
		}
		d.hasGoal, d.target = false, nil
		d.heading = d.rng.Float64() * 2 * math.Pi
	}
}

// findPath finds the quickest path of tiles from one tile to another (A*), not counting the
// start; roads cost less, as they are walked faster. With next, any tile beside the goal
// will do. It returns nil if there's no path within demoSearch tiles of the start.
func findPath(w *world.World, from, to [2]int, next bool) [][2]int {
	arrived := func(t [2]int) bool {
		if next {
			return abs(t[0]-to[0])+abs(t[1]-to[1]) == 1
		}
		return t == to
	}
	if arrived(from) {
		return [][2]int{}
	}
	h := func(t [2]int) int { return (abs(t[0]-to[0]) + abs(t[1]-to[1])) * world.RoadWalkFrames }
	cost := map[[2]int]int{from: 0}
	prev := map[[2]int][2]int{}
	open := &tileHeap{{from, h(from)}}
	for open.Len() > 0 && len(cost) < demoSearchCap {
		cur := heap.Pop(open).(tileCost).t
		if arrived(cur) {
			var path [][2]int
			for t := cur; t != from; t = prev[t] {
				path = append(path, t)
			}
			slices.Reverse(path)
			return path
		}
		for _, dir := range [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			n := [2]int{cur[0] + dir[0], cur[1] + dir[1]}
			if abs(n[0]-from[0]) > demoSearch || abs(n[1]-from[1]) > demoSearch || w.Blocked(n[0], n[1]) {
				continue
			}
			step := world.WalkFrames
			if w.Road(n[0], n[1]) {
				step = world.RoadWalkFrames
			}
			c := cost[cur] + step
			if old, seen := cost[n]; seen && old <= c {
				continue
			}
			cost[n], prev[n] = c, cur
			heap.Push(open, tileCost{n, c + h(n)})
		}
	}
	return nil
}

type tileCost struct {
	t [2]int
	f int // cost so far plus the estimate to the goal
}

type tileHeap []tileCost

func (h tileHeap) Len() int           { return len(h) }
func (h tileHeap) Less(i, j int) bool { return h[i].f < h[j].f }
func (h tileHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h *tileHeap) Push(x any)        { *h = append(*h, x.(tileCost)) }
func (h *tileHeap) Pop() any {
	old := *h
	x := old[len(old)-1]
	*h = old[:len(old)-1]
	return x
}

func dirOf(dx, dy int) world.Dir {
	switch {
	case dx > 0:
		return world.Right
	case dx < 0:
		return world.Left
	case dy < 0:
		return world.Up
	}
	return world.Down
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
