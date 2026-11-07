package game

import (
	"path/filepath"
	"testing"

	"oldworld2/internal/character"
	"oldworld2/internal/procgen"
	"oldworld2/internal/save"
	"oldworld2/internal/world"
)

// Starting a new game replaces the character, the world and the save.
func TestNewGameReplacesSave(t *testing.T) {
	store, err := save.Open(filepath.Join(t.TempDir(), "save.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	g := &Game{opts: Options{Seed: 1, Class: character.Warrior, Store: store, NewGame: true}}
	if err := g.loadOrStart(); err != nil {
		t.Fatal(err)
	}
	g.char.GainXP(1000)
	g.char.Money = 500
	g.autosave()

	g.opts.Seed, g.opts.Class = 77, character.Mage
	if err := g.loadOrStart(); err != nil {
		t.Fatal(err)
	}
	if g.char.Class != character.Mage || g.char.Level != 1 || g.char.Money != 0 {
		t.Fatalf("new game kept progress: %+v", g.char)
	}
	st, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if st.Character.Class != int(character.Mage) || st.Character.Level != 1 || st.Seed != 77 {
		t.Fatalf("save after new game: seed %d, %+v", st.Seed, st.Character)
	}
}

// The title screen starts a new game with the typed seed (or a random one if none was typed),
// the chosen class and the chosen start zone (Grassland by default).
func TestTitleStartUsesSeedClassAndZone(t *testing.T) {
	g := &Game{}
	ts := &titleScreen{g: g, class: 1, zone: 3, seed: "20"}
	ts.start()
	if g.opts.Seed != 20 || g.opts.Class != character.Mage || g.opts.StartZone != procgen.Implemented[3] || g.loading == nil {
		t.Fatalf("start: seed %d, class %v, zone %v, loading %v", g.opts.Seed, g.opts.Class, g.opts.StartZone, g.loading != nil)
	}
	g2 := &Game{}
	(&titleScreen{g: g2}).start()
	if g2.opts.Seed < 0 || g2.opts.Class != character.Warrior || g2.opts.StartZone != procgen.Grassland {
		t.Fatalf("random start: seed %d, class %v", g2.opts.Seed, g2.opts.Class)
	}
}

// A new game started in another zone type begins at a settlement of that type, which is also
// the first checkpoint; that zone counts as difficulty 1, so its enemies are levels 1-5.
func TestNewGameInStartZone(t *testing.T) {
	g := &Game{opts: Options{Seed: 1, Class: character.Warrior, StartZone: procgen.North, NewGame: true}}
	if err := g.loadOrStart(); err != nil {
		t.Fatal(err)
	}
	if got := procgen.ZoneTypeAt(1, g.player.TX, g.player.TY); got != procgen.North {
		t.Fatalf("started in %v at %d,%d", got, g.player.TX, g.player.TY)
	}
	if g.checkpoint != [2]int{g.player.TX, g.player.TY} {
		t.Fatalf("checkpoint %v, start %d,%d", g.checkpoint, g.player.TX, g.player.TY)
	}
	if zx, zy := procgen.ZoneOf(g.player.TX, g.player.TY); g.world.Difficulty(zx, zy) != 1 {
		t.Fatalf("start zone difficulty %d", g.world.Difficulty(zx, zy))
	}
	cx, cy := world.ChunkCoord(g.player.TX, g.player.TY)
	for dy := -3; dy <= 3; dy++ { // load the enemies around the start
		for dx := -3; dx <= 3; dx++ {
			g.world.Chunk(cx+dx, cy+dy)
		}
	}
	home := 0
	for _, n := range g.world.NPCs() {
		if zx, zy := procgen.ZoneOf(n.TX, n.TY); [2]int{zx, zy} == g.world.Home {
			home++
			if n.Spawn.Level < 1 || n.Spawn.Level > 5 {
				t.Errorf("%v at %d,%d: level %d", n.Spawn.Kind, n.TX, n.TY, n.Spawn.Level)
			}
		}
	}
	if home == 0 {
		t.Fatal("no enemies loaded in the start zone")
	}
}
