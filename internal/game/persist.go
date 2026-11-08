package game

import (
	"errors"
	"fmt"
	"os"
	"reflect"

	"oldworld2/internal/character"
	"oldworld2/internal/procgen"
	"oldworld2/internal/save"
	"oldworld2/internal/world"
)

// Autosave (design/game-design.md, Save System): important events save right away, walking
// around saves position and HP/MP at most every 30 seconds, and quitting saves the rest.
// Each part is only written when it changed since its last save.

const periodicSaveFrames = 30 * 60

// persisted remembers what was last written, so unchanged parts aren't written again.
type persisted struct {
	player    save.Player
	character save.Character
}

// loadOrStart continues the saved game, or starts a new one (and saves it).
func (g *Game) loadOrStart() error {
	if g.opts.Store != nil && !g.opts.NewGame {
		st, err := g.opts.Store.Load()
		switch {
		case err == nil:
			g.restore(st)
			return nil
		case !errors.Is(err, save.ErrNoSave):
			return fmt.Errorf("load save: %w (start with -new to begin a new game)", err)
		}
	}

	g.world = world.New(g.opts.Seed)
	g.char = character.New(g.opts.Class)
	for g.char.Level < g.opts.Level {
		g.char.GainXP(character.XPToNext(g.char.Level))
	}
	g.char.Heal()
	if g.opts.StartZone != procgen.Grassland {
		if zx, zy, ok := procgen.NearestZone(g.opts.Seed, g.opts.StartZone); ok {
			g.world.Home = [2]int{zx, zy} // before any chunk is generated: it sets the enemies' levels
		}
	}
	sx, sy := g.world.FindSpawnIn(g.world.Home[0], g.world.Home[1])
	g.checkpoint = [2]int{sx, sy}
	if at := g.opts.StartAt; at != nil {
		sx, sy = at[0], at[1]
		g.world.Prepare(sx, sy)
	}
	g.player = world.NewPlayer(sx, sy)

	if g.opts.Store == nil {
		return nil
	}
	st := &save.State{Seed: g.opts.Seed, Home: g.world.Home, Player: g.playerState(), Character: g.characterState()}
	if err := g.opts.Store.New(st); err != nil {
		return fmt.Errorf("create save: %w", err)
	}
	g.saved = persisted{st.Player, st.Character}
	return nil
}

func (g *Game) restore(st *save.State) {
	g.world = world.New(st.Seed)
	g.world.Home = st.Home
	g.world.SetDefeated(st.Defeated)
	c := st.Character
	g.char = &character.Character{
		Class: character.Class(c.Class), Level: c.Level, XP: c.XP,
		HP: c.HP, MP: c.MP, Money: c.Money, Known: renamedAbilities(c.Known), Active: renamedAbilities(c.Active),
	}
	p := st.Player
	g.world.Prepare(p.X, p.Y)
	g.player = world.NewPlayer(p.X, p.Y)
	g.player.Facing = world.Dir(p.Facing)
	g.checkpoint = [2]int{p.CheckpointX, p.CheckpointY}
	g.saved = persisted{p, c}
}

// abilityRenames maps ability IDs of older saves to their current ones.
var abilityRenames = map[string]string{"kidney_shot": "cheap_shot"}

func renamedAbilities(ids []string) []string {
	for i, id := range ids {
		if to, ok := abilityRenames[id]; ok {
			ids[i] = to
		}
	}
	return ids
}

func (g *Game) playerState() save.Player {
	return save.Player{X: g.player.TX, Y: g.player.TY, Facing: int(g.player.Facing),
		CheckpointX: g.checkpoint[0], CheckpointY: g.checkpoint[1]}
}

func (g *Game) characterState() save.Character {
	c := g.char
	return save.Character{Class: int(c.Class), Level: c.Level, XP: c.XP, HP: c.HP, MP: c.MP, Money: c.Money,
		Known: append([]string(nil), c.Known...), Active: append([]string(nil), c.Active...)}
}

// autosave writes the player and character parts if they changed.
func (g *Game) autosave() {
	if g.opts.Store == nil {
		return
	}
	if p := g.playerState(); p != g.saved.player {
		if err := g.opts.Store.SavePlayer(p); err != nil {
			fmt.Fprintln(os.Stderr, "autosave:", err)
		} else {
			g.saved.player = p
		}
	}
	if c := g.characterState(); !reflect.DeepEqual(c, g.saved.character) {
		if err := g.opts.Store.SaveCharacter(c); err != nil {
			fmt.Fprintln(os.Stderr, "autosave:", err)
		} else {
			g.saved.character = c
		}
	}
}

// saveDefeated remembers a defeated enemy right away.
func (g *Game) saveDefeated(n *world.NPC) {
	if g.opts.Store == nil {
		return
	}
	if err := g.opts.Store.AddDefeated(n.Spawn.ID); err != nil {
		fmt.Fprintln(os.Stderr, "autosave:", err)
	}
}

// Close saves what changed. Call it when the game ends.
func (g *Game) Close() {
	if g.player != nil { // nothing to save on the title screen
		g.autosave()
	}
	g.audio.Close()
}
