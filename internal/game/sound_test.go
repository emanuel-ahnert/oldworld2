package game

import (
	"os"
	"path/filepath"
	"testing"

	"oldworld2/internal/battle"
)

// TestSoundFilesExist checks that every sound and tune the game names was made by
// tools/audio/makeaudio.py: a wrong name would just play nothing.
func TestSoundFilesExist(t *testing.T) {
	dir := filepath.Join("..", "..", "assets", "generated", "audio")
	sounds := []string{sndMove, sndAccept, sndCancel, sndOpen, sndAlert, sndHeal, sndLight, sndLevel, sndCoins, sndLearn,
		"fire_crackle1", "fire_crackle2"}
	for _, s := range cueSounds {
		sounds = append(sounds, s)
	}
	for _, e := range []battle.Element{battle.Physical, battle.Fire, battle.Water, battle.Ice, battle.Holy, battle.Nature, battle.Dark} {
		for _, c := range []battle.Category{battle.PhysicalCat, battle.MagicalCat, battle.StatusCat} {
			for _, d := range []battle.Delivery{battle.Melee, battle.Ranged} {
				sounds = append(sounds, abilitySound(&battle.Ability{Element: e, Category: c, Delivery: d}))
			}
		}
	}
	for _, s := range sounds {
		if _, err := os.Stat(filepath.Join(dir, "sounds", s+".wav")); err != nil {
			t.Errorf("sound %q: %v", s, err)
		}
	}
	tunes := []string{"title", "sad", "fight", "fight_elite"}
	for _, m := range zoneMusic {
		tunes = append(tunes, m)
	}
	for _, m := range settlementMusic {
		tunes = append(tunes, m)
	}
	for _, m := range tunes {
		if _, err := os.Stat(filepath.Join(dir, "music", m+".ogg")); err != nil {
			t.Errorf("tune %q: %v", m, err)
		}
	}
}
