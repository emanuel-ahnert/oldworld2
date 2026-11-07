package save

import (
	"errors"
	"path/filepath"
	"reflect"
	"testing"
)

func TestRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "save.db")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Load(); !errors.Is(err, ErrNoSave) {
		t.Fatalf("empty file: got %v, want ErrNoSave", err)
	}
	want := &State{
		Seed:   42,
		Home:   [2]int{0, -1},
		Player: Player{X: -6, Y: -5, Facing: 1, CheckpointX: -6, CheckpointY: -5},
		Character: Character{Class: 1, Level: 3, XP: 12, HP: 80.5, MP: 200, Money: 1234,
			Known: []string{"ice_bolt", "fire_bolt"}, Active: []string{"ice_bolt"}},
	}
	if err := s.New(want); err != nil {
		t.Fatal(err)
	}
	// Partial updates.
	want.Player.X, want.Player.Y = 10, 20
	if err := s.SavePlayer(want.Player); err != nil {
		t.Fatal(err)
	}
	want.Character.Level = 4
	if err := s.SaveCharacter(want.Character); err != nil {
		t.Fatal(err)
	}
	for _, id := range []uint64{7, 1 << 63, 7} {
		if err := s.AddDefeated(id); err != nil {
			t.Fatal(err)
		}
	}
	want.Defeated = []uint64{7, 1 << 63}
	s.Close()

	s, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	got, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Defeated) != 2 {
		t.Fatalf("defeated: %v", got.Defeated)
	}
	got.Defeated, want.Defeated = nil, nil
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("loaded %+v\nwant   %+v", got, want)
	}

	// Starting a new game clears everything.
	if err := s.New(&State{Seed: 1, Character: Character{Known: []string{}, Active: []string{}}}); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.Load(); got.Seed != 1 || len(got.Defeated) != 0 {
		t.Fatalf("new game kept old data: %+v", got)
	}
}
