package game

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"oldworld2/internal/r3d"
	"oldworld2/internal/tiles"
)

// Map objects have no 2D fallback any more: every one needs a scenery model, or a building
// shape (scene.building).
func TestEveryObjectHasAModel(t *testing.T) {
	for id := tiles.TreeRound; id <= tiles.Crypt; id++ {
		if _, ok := sceneryModels[id]; !ok && !tiles.IsBuilding(id) {
			t.Errorf("object %d has no scenery model", id)
		}
	}
}

// Every outfit piece and held prop a hero or enemy wears must exist: a wrong name would load
// nothing, which only shows in the game.
func TestOutfitFilesExist(t *testing.T) {
	lib := r3d.NewLibrary(filepath.Join("..", "..", "assets", "quaternius"))
	check := func(name string) {
		if p := lib.Find(name + ".gltf"); p == "" {
			t.Errorf("%q not found", name)
		} else if _, err := os.Stat(p); err != nil {
			t.Errorf("%q: %v", name, err)
		}
	}
	outfits := []r3d.Outfit{banditOutfit, nomadOutfit, berserkerOutfit, corsairOutfit, hopliteOutfit, cultistOutfit, vampireOutfit}
	for _, l := range heroLooks {
		outfits = append(outfits, l.outfit)
		for _, prop := range []string{l.weapon, l.shield} {
			if prop != "" {
				check("../generated/props/" + prop)
			}
		}
	}
	for _, o := range outfits {
		for _, part := range append([]string{o.Body, o.Head, o.Chest, o.Arms, o.Legs, o.Feet, o.Shoulders}, o.Hair...) {
			if part != "" {
				check(part)
			}
		}
	}
}

// A release holds only the pack models in tools/release/pack_models.txt: one the game loads but
// the list lacks would be missing only there.
func TestPackModelsInRelease(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "tools", "release", "pack_models.txt"))
	if err != nil {
		t.Fatal(err)
	}
	listed := strings.Split(string(data), "\n")
	dir := filepath.Join("..", "..", "assets", "quaternius")
	lib := r3d.NewLibrary(dir)
	check := func(name string) {
		rel, err := filepath.Rel(dir, lib.Find(name))
		if err != nil {
			t.Fatal(err)
		}
		if !slices.Contains(listed, filepath.ToSlash(rel)) {
			t.Errorf("%s (%s) is not in pack_models.txt", name, filepath.ToSlash(rel))
		}
	}
	check("UAL1_Standard.glb")
	outfits := []r3d.Outfit{banditOutfit, nomadOutfit, berserkerOutfit, corsairOutfit, hopliteOutfit, cultistOutfit, vampireOutfit}
	for _, l := range heroLooks {
		outfits = append(outfits, l.outfit)
	}
	for _, o := range outfits {
		for _, part := range append([]string{o.Body, o.Head, o.Chest, o.Arms, o.Legs, o.Feet, o.Shoulders}, o.Hair...) {
			if part != "" && !strings.HasPrefix(part, "../") {
				check(part + ".gltf")
			}
		}
	}
}
