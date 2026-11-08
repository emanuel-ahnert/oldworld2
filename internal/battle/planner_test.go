package battle

import (
	"testing"
	"time"
)

// A close fight: a warrior against a wolf with twice the usual HP and attack.
func closeFight() (*Combatant, *Combatant) {
	w := wolf()
	w.Base.MaxHP, w.Base.Attack = 2*w.Base.MaxHP, 2*w.Base.Attack
	w.HP = w.Base.MaxHP
	return warrior(), w
}

func TestPlanningLeavesTheBattleAlone(t *testing.T) {
	p, e := closeFight()
	b := New(p, e, 1)
	b.Turn("heavy_strike", "bite")
	php, ehp, pmp, round := p.HP, e.HP, p.MP, b.Round
	b.PlanPlayerMove(7)
	if p.HP != php || e.HP != ehp || p.MP != pmp || b.Round != round || len(p.used) != 1 {
		t.Fatalf("planning changed the battle: HP %d→%d, enemy HP %d→%d, MP %d→%d, round %d→%d, uses %v",
			php, p.HP, ehp, e.HP, pmp, p.MP, round, b.Round, p.used)
	}
}

func TestPlannerBeatsRandomPlay(t *testing.T) {
	const n = 200
	planned, random := 0, 0
	for i := range uint64(n) {
		p, e := closeFight()
		if Simulate(p, e, i).Won {
			planned++
		}
		p, e = closeFight()
		b := New(p, e, i)
		for !b.Over() && b.Round < 100 {
			moves := b.usableMoves(b.Player)
			b.Turn(moves[b.rng.IntN(len(moves))], b.ChooseEnemyAbility())
		}
		if b.Won() {
			random++
		}
	}
	t.Logf("wins out of %d: planner %d, random play %d", n, planned, random)
	if planned <= random {
		t.Fatalf("the planner won %d of %d battles, random play %d", planned, n, random)
	}
}

// The demo bot plans while the game runs, so a move has to be quick to find.
func TestPlanningIsFast(t *testing.T) {
	p, e := closeFight()
	b := New(p, e, 1)
	start := time.Now()
	b.PlanPlayerMove(1)
	if d := time.Since(start); d > 50*time.Millisecond {
		t.Fatalf("planning a move took %v", d)
	}
}
