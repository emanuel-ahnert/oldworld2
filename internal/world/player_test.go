package world

import "testing"

func open(int, int) bool { return false }

func hold(p *Player, in Input, frames int, blocked func(int, int) bool) {
	for i := 0; i < frames; i++ {
		p.Update(in, blocked, nil)
	}
}

func TestWalkingStepTakesWalkFrames(t *testing.T) {
	p := NewPlayer(0, 0) // facing down, so walking down starts without a turn
	p.Update(Input{Dir: Down, HasDir: true}, open, nil)
	if !p.Moving() || p.TY != 1 {
		t.Fatalf("expected a step to start, got moving=%v tile=(%d,%d)", p.Moving(), p.TX, p.TY)
	}
	hold(p, Input{}, WalkFrames-1, open)
	if !p.Moving() {
		t.Fatal("step ended too early")
	}
	p.Update(Input{}, open, nil)
	if p.Moving() {
		t.Fatal("step should have ended")
	}
}

func TestRunningIsTwiceAsFast(t *testing.T) {
	p := NewPlayer(0, 0)
	hold(p, Input{Dir: Down, HasDir: true, Run: true}, 4*RunFrames, open)
	if p.TY != 4 {
		t.Fatalf("expected 4 tiles after %d running frames, got %d", 4*RunFrames, p.TY)
	}
}

func TestTapTurnsWithoutMoving(t *testing.T) {
	p := NewPlayer(0, 0)
	hold(p, Input{Dir: Left, HasDir: true}, TurnFrames-1, open)
	p.Update(Input{}, open, nil)
	if p.Facing != Left || p.TX != 0 || p.Moving() {
		t.Fatalf("tap should only turn: facing=%v tile=(%d,%d) moving=%v", p.Facing, p.TX, p.TY, p.Moving())
	}
}

func TestHoldingNewDirectionWalksAfterTurn(t *testing.T) {
	p := NewPlayer(0, 0)
	hold(p, Input{Dir: Left, HasDir: true}, TurnFrames, open)
	if !p.Moving() || p.TX != -1 {
		t.Fatalf("expected a step left after the turn pause, got tile=(%d,%d)", p.TX, p.TY)
	}
}

func TestBlockedTileStopsMovement(t *testing.T) {
	p := NewPlayer(0, 0)
	wall := func(x, y int) bool { return y == 1 }
	hold(p, Input{Dir: Down, HasDir: true}, 30, wall)
	if p.TY != 0 || p.Moving() {
		t.Fatalf("walked into a blocked tile: tile=(%d,%d)", p.TX, p.TY)
	}
}

func TestStaminaRunsOutThenWalks(t *testing.T) {
	p := NewPlayer(0, 0)
	run := Input{Dir: Down, HasDir: true, Run: true}
	frames := 0
	for p.Stamina > 0 {
		p.Update(run, open, nil)
		frames++
	}
	if want := int(StaminaMax * 60); frames < want-2 || frames > want+2 {
		t.Fatalf("stamina lasted %d frames, want about %d", frames, want)
	}
	hold(p, run, RunFrames, open) // finish the running step in progress
	if p.Running() {
		t.Fatal("should walk without stamina")
	}
}

func TestStaminaRefillsFasterStandingStill(t *testing.T) {
	moving, standing := NewPlayer(0, 0), NewPlayer(0, 0)
	moving.Stamina, standing.Stamina = 0, 0
	hold(moving, Input{Dir: Down, HasDir: true}, 120, open)
	hold(standing, Input{}, 120, open)
	if standing.Stamina <= moving.Stamina*1.9 {
		t.Fatalf("standing refill %.2f should be about twice moving refill %.2f", standing.Stamina, moving.Stamina)
	}
}

// Running with a lower stamina use (Improved Sprint) drains stamina that much slower.
func TestStaminaUse(t *testing.T) {
	normal, perk := NewPlayer(0, 0), NewPlayer(0, 0)
	perk.StaminaUse = 0.2
	run := Input{Dir: Down, HasDir: true, Run: true}
	hold(normal, run, 120, open)
	hold(perk, run, 120, open)
	used, usedPerk := StaminaMax-normal.Stamina, StaminaMax-perk.Stamina
	if used < 1.9 || usedPerk < 0.2*used-0.05 || usedPerk > 0.2*used+0.05 {
		t.Fatalf("2 s of running used %.2f s of stamina, %.2f with the perk", used, usedPerk)
	}
}

func TestRoadStepsAreFaster(t *testing.T) {
	road := func(_, ty int) bool { return ty >= 1 } // the road starts one tile down
	for _, c := range []struct {
		run         bool
		off, onRoad int
	}{{false, WalkFrames, RoadWalkFrames}, {true, RunFrames, RoadRunFrames}} {
		p := NewPlayer(0, 0)
		in := Input{Dir: Down, HasDir: true, Run: c.run}
		p.Update(in, open, road) // onto the road
		if p.StepFrames() != c.onRoad {
			t.Errorf("run=%v: step onto a road takes %d frames, want %d", c.run, p.StepFrames(), c.onRoad)
		}
		p = NewPlayer(0, 1)
		for i := 0; i < TurnFrames; i++ { // turn up, then step off the road
			p.Update(Input{Dir: Up, HasDir: true, Run: c.run}, open, road)
		}
		if p.StepFrames() != c.off {
			t.Errorf("run=%v: step off a road takes %d frames, want %d", c.run, p.StepFrames(), c.off)
		}
	}
}
