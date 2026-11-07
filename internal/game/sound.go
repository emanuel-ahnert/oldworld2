package game

import (
	rl "github.com/gen2brain/raylib-go/raylib"

	"oldworld2/internal/battle"
	"oldworld2/internal/procgen"
)

// Sound and music (internal/audio, made by tools/audio/makeaudio.py). Music follows what is
// on screen: the title, the zone (with its own tune near a settlement), battles; the map tune
// resumes where it left off after a battle. Sounds are named by their file in sounds/.

const (
	sndMove   = "menu_move"
	sndAccept = "menu_accept"
	sndCancel = "menu_cancel"
	sndOpen   = "menu_open"
	sndAlert  = "alert"
	sndHeal   = "heal"
	sndLight  = "light"
	sndLevel  = "level_up"
	sndCoins  = "coins"
	sndLearn  = "learn"

	villageRadius = 40 // tiles from a settlement where the village tune plays
)

// zoneMusic is the map tune of each zone type, and settlementMusic the one near its hub.
var (
	zoneMusic = map[procgen.ZoneType]string{
		procgen.Grassland:     "grassland",
		procgen.Desert:        "desert",
		procgen.AncientDesert: "ancient",
		procgen.North:         "north",
		procgen.Mediterranean: "mediterranean",
		procgen.Unholy:        "unholy",
	}
	settlementMusic = map[procgen.ZoneType]string{
		procgen.Grassland:     "village",
		procgen.Desert:        "oasis",
		procgen.AncientDesert: "oasis",
		procgen.North:         "hall",
		procgen.Mediterranean: "harbor",
		procgen.Unholy:        "chapel",
	}
)

// cueSounds are the battle event sounds; abilities have their own (abilitySound).
var cueSounds = map[battle.Cue]string{
	battle.CueHit:      "hit",
	battle.CueCrit:     "crit",
	battle.CueMiss:     "miss",
	battle.CueNoEffect: "no_effect",
	battle.CueDefeat:   "defeat",
	battle.CueStatUp:   "stat_up",
	battle.CueStatDown: "stat_down",
	battle.CueStatus:   "status",
	battle.CueHeal:     "heal_soft",
}

// abilitySound is the sound of using an ability: spells by element, weapons and claws by
// how they're delivered.
func abilitySound(a *battle.Ability) string {
	switch {
	case a.Category == battle.StatusCat:
		return "curse"
	case a.Category == battle.MagicalCat:
		switch a.Element {
		case battle.Fire:
			return "fireball"
		case battle.Water:
			return "water"
		case battle.Holy:
			return "holy"
		case battle.Nature:
			return "nature"
		case battle.Dark:
			return "dark"
		}
		return "magic"
	case a.Delivery == battle.Ranged:
		return "launch"
	case a.Element == battle.Fire:
		return "fire_strike"
	}
	return "slash"
}

// sound plays a sound effect.
func (g *Game) sound(name string) { g.audio.Sound(name) }

// updateAudio picks the music for what is on screen and plays the camp fire and the
// enemies' alerts. It runs once per frame, after the game update. While the window is in the
// background (unfocused or minimized) everything is paused.
func (g *Game) updateAudio() {
	g.audio.SetPaused(!rl.IsWindowFocused() || rl.IsWindowMinimized())
	switch {
	case g.loading != nil:
	case g.title != nil:
		g.audio.Music("title", false)
	case g.battle != nil:
		s := g.battle
		switch {
		case s.phase == phaseResult && !s.b.Won():
			g.audio.Music("sad", true)
		case s.phase == phaseResult:
			g.audio.Music("", false)
		case s.tier == "elite":
			g.audio.Music("fight_elite", true)
		default:
			g.audio.Music("fight", true)
		}
	case g.player != nil:
		if g.frame%30 == 0 || g.mapMusic == "" { // the settlement check needn't run every frame
			g.mapMusic = g.zoneTune()
		}
		g.audio.Music(g.mapMusic, false)
		if g.camp != nil && g.frame%20 == 0 && (g.frame/20*7919)%5 < 2 {
			g.audio.SoundAt([]string{"fire_crackle1", "fire_crackle2"}[g.frame/20%2], 0.35)
		}
		for _, n := range g.world.NPCs() {
			if n.SpottedNow() {
				g.sound(sndAlert)
				break
			}
		}
	}
	g.audio.Update()
}

// zoneTune is the map music where the player stands.
func (g *Game) zoneTune() string {
	l := procgen.LayoutAt(g.world.Seed, g.player.TX, g.player.TY)
	tunes := zoneMusic
	if dx, dy := g.player.TX-l.HubX, g.player.TY-l.HubY; dx*dx+dy*dy < villageRadius*villageRadius {
		tunes = settlementMusic
	}
	if t, ok := tunes[l.Type]; ok {
		return t
	}
	return tunes[procgen.Grassland]
}

// battleEventSound plays the sound of a battle event as its message starts.
func (g *Game) battleEventSound(ev battle.Event) {
	if ev.Cue == battle.CueUse && ev.Ability != nil {
		g.sound(abilitySound(ev.Ability))
		return
	}
	if name, ok := cueSounds[ev.Cue]; ok {
		g.sound(name)
	}
}

// menuConfirm is confirmPressed for menus: it plays the accept sound.
func (g *Game) menuConfirm() bool {
	if g.confirmPressed() {
		g.sound(sndAccept)
		return true
	}
	return false
}
