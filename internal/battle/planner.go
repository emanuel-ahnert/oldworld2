package battle

import "math/rand/v2"

// The planner picks the player's move for the demo bot (and the balance tool) by trying each
// usable move on copies of the battle and playing them out with simple random play: no trained
// model, the battle rules themselves are the model.

const (
	planPlayouts  = 48 // playouts per candidate move
	planMaxRounds = 30 // a playout still running after this many rounds counts as unfinished
)

// clone returns an independent copy of the battle that rolls its dice from seed and
// doesn't write messages.
func (b *Battle) clone(seed uint64) *Battle {
	c := &Battle{Round: b.Round, started: b.started, quiet: true,
		rng: rand.New(rand.NewPCG(seed, seed^0x9E3779B97F4A7C15))}
	c.Player, c.Enemy = b.Player.clone(), b.Enemy.clone()
	return c
}

func (c *Combatant) clone() *Combatant {
	d := *c
	d.status = make(map[StatusKind]int, len(c.status))
	for k, v := range c.status {
		d.status[k] = v
	}
	d.used = make(map[string]int, len(c.used))
	for k, v := range c.used {
		d.used[k] = v
	}
	return &d
}

// usableMoves lists the moves a combatant can choose now; Wait only when nothing else is.
func (b *Battle) usableMoves(user *Combatant) []string {
	if user.iceBlock > 0 {
		return []string{WaitID} // the bot sits its Ice Block out
	}
	var out []string
	for _, id := range b.MenuOptions(user) {
		if ok, _ := b.Usable(user, id); ok && id != WaitID {
			out = append(out, id)
		}
	}
	if len(out) == 0 {
		out = []string{WaitID}
	}
	return out
}

// PlanPlayerMove returns the player's best move by playouts. Every candidate is played out
// with the same seeds, so the comparison isn't skewed by lucky dice.
func (b *Battle) PlanPlayerMove(seed uint64) string {
	moves := b.usableMoves(b.Player)
	if len(moves) == 1 {
		return moves[0]
	}
	best, bestScore := moves[0], -1.0
	for _, m := range moves {
		score := 0.0
		for i := range planPlayouts {
			score += b.playout(m, seed+uint64(i)*0x9E3779B97F4A7C15)
		}
		if score > bestScore {
			best, bestScore = m, score
		}
	}
	return best
}

// playout plays the battle to the end after the player opens with first, both sides then
// choosing at random among their usable moves (the enemy like in a real battle). It scores
// the outcome from the player's view: a win is worth 1 plus up to 0.4 for HP and 0.1 for MP
// left (both carry over to the next fight), a loss or an unfinished battle up to 0.5 for the
// damage dealt.
func (b *Battle) playout(first string, seed uint64) float64 {
	c := b.clone(seed)
	c.Turn(first, c.ChooseEnemyAbility())
	for !c.Over() && c.Round < b.Round+planMaxRounds {
		moves := c.usableMoves(c.Player)
		c.Turn(moves[c.rng.IntN(len(moves))], c.ChooseEnemyAbility())
	}
	p, e := c.Player, c.Enemy
	if c.Won() {
		mp := 0.0 // only mana carries over; rage and energy reset every battle
		if p.Resource == Mana {
			mp = float64(p.MP) / float64(max(1, p.Base.MaxMP))
		}
		return 1 + 0.4*float64(p.HP)/float64(p.Base.MaxHP) + 0.1*mp
	}
	return 0.5 * (1 - float64(max(0, e.HP))/float64(e.Base.MaxHP))
}

// Outcome is how a simulated battle ended.
type Outcome struct {
	Won    bool
	Rounds int
	HPLeft float64 // the player's HP left, as a share of the maximum
}

// Simulate plays a whole battle with the planner choosing the player's moves (the balance
// tool, cmd/balance).
func Simulate(player, enemy *Combatant, seed uint64) Outcome {
	b := New(player, enemy, seed)
	b.quiet = true
	b.Start()
	for !b.Over() && b.Round < 100 {
		b.Turn(b.PlanPlayerMove(seed+uint64(b.Round)), b.ChooseEnemyAbility())
	}
	return Outcome{Won: b.Won(), Rounds: b.Round, HPLeft: float64(max(0, player.HP)) / float64(player.Base.MaxHP)}
}
