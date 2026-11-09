package game

import (
	"container/heap"
	"math"
	"math/rand/v2"
	"slices"

	"oldworld2/internal/battle"
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
	demoSight        = 24          // tiles: how far away the bot goes for an enemy
	demoSearch       = 64          // tiles: how far from the player a path may lead
	demoSearchCap    = 40000       // tiles (in each direction walked) a path search looks at, at most
	demoTurnCost     = 8           // frames a turn counts extra in a path search
	demoStuck        = 90          // frames without getting anywhere before it tries elsewhere
	demoIgnore       = 20 * 60     // frames an enemy it couldn't reach is left alone
	demoAvoid        = 5 * 60 * 60 // frames an enemy that beat it is left alone
	demoRestHP       = 0.7         // rest when HP fall below this share...
	demoRestMP       = 0.5         // ...or MP below this one...
	demoRestUntil    = 0.95        // ...until both are back above this
	demoRunStamina   = 0.4         // run only with this share of stamina left
	demoAwait        = 4 * 60      // frames it waits for a chasing enemy before walking to it
	demoGroup        = 6           // tiles: enemies this close together count as one group
	demoPastCrossing = 6           // tiles into the next zone a trip there ends
)

type demoBot struct {
	g     *Game
	rng   *rand.Rand
	store *save.Store // the save, put aside while the demo runs

	goal    [2]int
	hasGoal bool
	heading float64    // direction of travel while exploring, so it doesn't turn back and forth
	target  *world.NPC // the enemy it is walking to
	ignore  map[uint64]int
	path    [][2]int
	replan  int // frames until the path is planned again

	last    [2]int         // where the player stood
	still   int            // frames the player has stood there
	camp    bool           // press the camp key this frame
	noRest  int            // frames left without trying to rest (no room for a fire)
	hurt    bool           // HP or MP are low: it fights only enemies already after it
	await   int            // frames it has waited for a chasing enemy to come
	tries   int            // camp key presses without a fire lighting
	flee    bool           // running from chasers while hurt
	ready   bool           // HP and MP are high enough for the enemy group it would go for next
	dead    map[[2]int]int // spots it found no way to, and until which frame it leaves them alone
	running bool           // it runs (see input)
}

// startDemo starts a demo run: a new, unsaved game with a random class, zone and world.
func (g *Game) startDemo() {
	d := g.demo
	restart := d != nil
	if d == nil {
		d = &demoBot{g: g, store: g.opts.Store, rng: rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64()))}
		g.demo = d
	}
	g.opts.Store = nil // nothing is saved while the demo runs
	g.opts.Seed = d.rng.Int64N(1_000_000)
	class, zone := g.opts.Class, g.opts.StartZone
	for {
		g.opts.Class = classes[d.rng.IntN(len(classes))]
		g.opts.StartZone = procgen.Implemented[d.rng.IntN(len(procgen.Implemented))]
		if !restart || g.opts.Class != class || g.opts.StartZone != zone { // a restart shows something new
			break
		}
	}
	g.opts.Level, g.opts.StartAt = 1, nil
	*d = demoBot{g: g, store: d.store, rng: d.rng, ignore: map[uint64]int{}, dead: map[[2]int]int{}, heading: d.rng.Float64() * 2 * math.Pi}
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
	hp, mp := g.char.HP/float64(st.MaxHP), g.char.ManaShare()
	if d.noRest > 0 {
		d.noRest--
	}
	d.ticks()

	// Rest at a camp fire when hurt, unless an enemy is already on the way.
	d.hurt = hp < demoRestHP || mp < demoRestMP
	if g.camp != nil {
		d.tries = 0
		if hp < demoRestUntil || mp < demoRestUntil {
			return world.Input{}
		}
	} else if (d.hurt || !d.ready) && d.noRest == 0 && d.chaser() == nil {
		// Stop (a step under way finishes first), then light the fire.
		if !p.Moving() {
			d.camp = true
			if d.tries++; d.tries > 5 { // the fire doesn't light: no room here
				d.noRest, d.tries = 300, 0
			}
		}
		return world.Input{}
	}

	// Hurt and chased, or chased by an enemy it can't beat: run instead of waiting for it. A
	// chase gives up after a while (5 to 15 seconds), and then it can rest; a fast enemy may
	// still catch it.
	if c := d.threat(); c != nil && g.camp == nil {
		if !d.flee || !d.hasGoal {
			d.target, d.hasGoal = nil, false
			d.heading = math.Atan2(float64(p.TY-c.TY), float64(p.TX-c.TX))
			d.explore()
			d.flee = d.hasGoal
		}
	} else if d.flee {
		d.flee, d.hasGoal, d.path = false, false, nil
	}
	if !d.flee {
		d.chooseGoal()
	}
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
	// Running starts on a long way with stamina to spare and then keeps on until stamina runs
	// low or the goal is near, so the pace doesn't flip between walking and running.
	switch {
	case d.flee:
		d.running = p.Stamina > 0
	case d.running:
		d.running = p.Stamina > world.StaminaMax*0.1 && len(d.path) > 2
	default:
		d.running = p.Stamina > world.StaminaMax*demoRunStamina && len(d.path) > 10
	}
	run := d.running
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
	if d.still > demoStuck && (d.hasGoal || d.still > 3*demoStuck) {
		if d.target != nil {
			d.ignore[d.target.Spawn.ID] = d.g.frame + demoIgnore
		}
		d.hasGoal, d.target, d.path = false, nil, nil
		d.heading = d.rng.Float64() * 2 * math.Pi
		d.still = 0
	}
}

// threat returns the chaser to run from: the nearest one when hurt, otherwise one it can't
// beat.
func (d *demoBot) threat() *world.NPC {
	if d.hurt {
		return d.chaser()
	}
	p := d.g.player
	for _, n := range d.g.world.NPCs() {
		if n.Chasing() && !d.beatable(n) && abs(n.TX-p.TX)+abs(n.TY-p.TY) < 12 {
			return n
		}
	}
	return nil
}

// chaser returns the nearest enemy chasing the player, if one is close.
func (d *demoBot) chaser() *world.NPC {
	p := d.g.player
	var near *world.NPC
	best := 12
	for _, n := range d.g.world.NPCs() {
		if dist := abs(n.TX-p.TX) + abs(n.TY-p.TY); n.Chasing() && dist < best {
			near, best = n, dist
		}
	}
	return near
}

// beatable reports whether an enemy is one the bot can take on: not far above the player's
// level, and elites only at its own level or below.
func (d *demoBot) beatable(n *world.NPC) bool {
	lv := d.g.char.Level
	if n.Info().Tier == "elite" {
		return n.Spawn.Level <= lv
	}
	return n.Spawn.Level <= lv+1
}

// worthwhile reports whether an enemy is worth going for: one it can beat that gives a fair
// share of XP (not 5 or more levels below the player), so it moves on to harder zones instead
// of farming weak enemies.
func (d *demoBot) worthwhile(n *world.NPC) bool {
	return d.beatable(n) && n.Spawn.Level > d.g.char.Level-5
}

// chooseGoal picks where to go: the nearest enemy it can beat, otherwise a spot to explore.
// Enemies in a group come one after another, so before going for a group it wants more HP
// and MP the bigger the group is, and rests first if it hasn't (ready).
func (d *demoBot) chooseGoal() {
	g, p := d.g, d.g.player
	var best *world.NPC
	bestDist := demoSight + 1
	d.ready = true
	for _, n := range g.world.NPCs() {
		dist := abs(n.TX-p.TX) + abs(n.TY-p.TY)
		if dist >= bestDist || g.frame < d.ignore[n.Spawn.ID] || (!n.Chasing() && (d.hurt || !d.worthwhile(n))) {
			continue
		}
		if !n.Chasing() && !d.readyFor(n) {
			d.ready = false
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

// reachable reports whether a spot isn't near one it recently found no way to.
func (d *demoBot) reachable(x, y int) bool {
	for spot, until := range d.dead {
		if d.g.frame >= until {
			delete(d.dead, spot)
		} else if abs(spot[0]-x)+abs(spot[1]-y) <= 8 {
			return false
		}
	}
	return true
}

// readyFor reports whether HP and MP are high enough to take on an enemy and the others
// standing with it: 70% HP and 50% MP for one alone, 15% more of each per extra enemy, up to
// the level resting stops at.
func (d *demoBot) readyFor(n *world.NPC) bool {
	group := 0
	for _, o := range d.g.world.NPCs() {
		if abs(o.TX-n.TX)+abs(o.TY-n.TY) <= demoGroup {
			group++
		}
	}
	st := d.g.char.Stats()
	extra := 0.15 * float64(group-1)
	return d.g.char.HP/float64(st.MaxHP) >= min(demoRestUntil, demoRestHP+extra) &&
		d.g.char.ManaShare() >= min(demoRestUntil, demoRestMP+extra)
}

// explore picks a spot 20 to 40 tiles ahead, roughly in the current heading: open ground,
// in a zone not too hard for the player's level, a road if there is one. Once the player's
// level suits a harder zone next door, it travels there instead.
func (d *demoBot) explore() {
	g, p := d.g, d.g.player
	maxDifficulty := 1 + (g.char.Level-1)/5 // enemy levels rise by 5 per difficulty step
	if d.travel(maxDifficulty) {
		return
	}
	bestScore := math.Inf(-1)
	for range 24 {
		a := d.heading + (d.rng.Float64()-0.5)*math.Pi*0.8
		r := 20 + d.rng.Float64()*20
		x, y := p.TX+int(math.Round(r*math.Cos(a))), p.TY+int(math.Round(r*math.Sin(a)))
		if g.world.Blocked(x, y) || !d.reachable(x, y) {
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

// travel heads for the next zone once the player's level allows a harder one than the zone
// it is in: to the nearest crossing into a neighboring zone of a higher difficulty it can
// take on, along the roads where it can (roads run from each crossing to the zone's
// settlement). Far from the crossing it walks there in legs of 25 to 45 tiles, each leg
// ending where it gets closest, a road counting as a few tiles closer. It reports false if
// there's no zone to go to.
func (d *demoBot) travel(maxDifficulty int) bool {
	g, p := d.g, d.g.player
	zx, zy := procgen.ZoneOf(p.TX, p.TY)
	here := g.world.Difficulty(zx, zy)
	var dest [2]int
	bestDist := math.MaxInt
	for _, dir := range [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
		if diff := g.world.Difficulty(zx+dir[0], zy+dir[1]); diff <= here || diff > maxDifficulty {
			continue
		}
		for _, c := range procgen.Crossings(g.world.Seed, zx, zy, dir[0], dir[1]) {
			past := [2]int{c[0] + dir[0]*demoPastCrossing, c[1] + dir[1]*demoPastCrossing}
			if dist := abs(past[0]-p.TX) + abs(past[1]-p.TY); dist < bestDist {
				dest, bestDist = past, dist
			}
		}
	}
	if bestDist == math.MaxInt {
		return false
	}
	if bestDist <= demoSearch*3/4 && d.reachable(dest[0], dest[1]) {
		d.goal, d.hasGoal = dest, true
	} else {
		toward := math.Atan2(float64(dest[1]-p.TY), float64(dest[0]-p.TX))
		best := math.Inf(-1)
		for range 48 {
			a := toward + (d.rng.Float64()-0.5)*math.Pi
			r := 25 + d.rng.Float64()*20
			x, y := p.TX+int(math.Round(r*math.Cos(a))), p.TY+int(math.Round(r*math.Sin(a)))
			if g.world.Blocked(x, y) || !d.reachable(x, y) {
				continue
			}
			score := float64(bestDist - abs(dest[0]-x) - abs(dest[1]-y))
			if g.world.Road(x, y) {
				score += 8
			}
			if score > best {
				best, d.goal, d.hasGoal = score, [2]int{x, y}, true
			}
		}
		if !d.hasGoal {
			return false
		}
	}
	d.heading = math.Atan2(float64(d.goal[1]-p.TY), float64(d.goal[0]-p.TX))
	d.replan = 0
	return true
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
		} else {
			d.dead[d.goal] = d.g.frame + demoIgnore
		}
		d.hasGoal, d.target = false, nil
		d.heading = d.rng.Float64() * 2 * math.Pi
	}
}

// findPath finds the quickest path of tiles from one tile to another (A*), not counting the
// start; roads cost less, as they are walked faster, and every turn costs a little extra, so
// the path runs in long straight stretches instead of zigzagging (on a grid, a staircase is
// as short as an L, but it looks twitchy to watch). With next, any tile beside the goal
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
	dirs := [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}
	h := func(t [2]int) int { return (abs(t[0]-to[0]) + abs(t[1]-to[1])) * world.RoadWalkFrames }
	start := pathState{from, -1}
	cost := map[pathState]int{start: 0}
	prev := map[pathState]pathState{}
	open := &tileHeap{{start, h(from)}}
	for open.Len() > 0 && len(cost) < demoSearchCap {
		cur := heap.Pop(open).(tileCost).s
		if arrived(cur.t) {
			var path [][2]int
			for s := cur; s != start; s = prev[s] {
				path = append(path, s.t)
			}
			slices.Reverse(path)
			return path
		}
		for d, dir := range dirs {
			n := pathState{[2]int{cur.t[0] + dir[0], cur.t[1] + dir[1]}, d}
			if abs(n.t[0]-from[0]) > demoSearch || abs(n.t[1]-from[1]) > demoSearch || w.Blocked(n.t[0], n.t[1]) {
				continue
			}
			step := world.WalkFrames
			if w.Road(n.t[0], n.t[1]) {
				step = world.RoadWalkFrames
			}
			if cur.dir >= 0 && cur.dir != d {
				step += demoTurnCost
			}
			c := cost[cur] + step
			if old, seen := cost[n]; seen && old <= c {
				continue
			}
			cost[n], prev[n] = c, cur
			heap.Push(open, tileCost{n, c + h(n.t)})
		}
	}
	return nil
}

// pathState is a tile reached walking in a direction (an index into findPath's dirs, -1 at
// the start), so turns can cost extra.
type pathState struct {
	t   [2]int
	dir int
}

type tileCost struct {
	s pathState
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

// demoOpeners are the key moves the demo bot opens a battle with when it knows them: the
// warrior's Charge, the rogue's Cheap Shot, the paladin's Judgment Hammer (all stuns, so the
// rest of the fight starts with a free round) and the mage's Blink (the first blow misses).
var demoOpeners = []string{"charge", "cheap_shot", "judgment_hammer", "blink"}

// demoOpener is the bot's first move of a battle, or "" to leave the move to the planner.
func demoOpener(b *battle.Battle) string {
	if b.Round > 0 {
		return ""
	}
	for _, id := range demoOpeners {
		if !slices.Contains(b.Player.Abilities, id) {
			continue
		}
		if ok, _ := b.Usable(b.Player, id); ok {
			return id
		}
	}
	return ""
}
