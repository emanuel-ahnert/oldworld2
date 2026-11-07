// Package game runs the game: the adventure map and battles in 3D (raylib, see r3d) with
// the pixel UI on top (see gfx).
package game

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"

	"oldworld2/internal/assets"
	"oldworld2/internal/audio"
	"oldworld2/internal/character"
	"oldworld2/internal/gfx"
	"oldworld2/internal/procgen"
	"oldworld2/internal/save"
	"oldworld2/internal/tiles"
	"oldworld2/internal/world"
)

// Base internal resolution. The image is scaled up by whole numbers; the canvas grows to
// fill the rest of the window, so the view shows a bit more instead of black bars.
const (
	ScreenW = 320
	ScreenH = 180
)

const (
	unloadRadius     = 3   // chunks kept around the player
	zoneBannerFrames = 210 // the zone name shows for 3.5 seconds when entering a zone
)

// Options configure a game run.
type Options struct {
	Seed      int64
	Class     character.Class
	Level     int              // start at this level (for testing); 0 or 1 = a new character
	StartAt   *[2]int          // start a new game at this tile instead of the start point (for testing)
	StartZone procgen.ZoneType // a new game starts at the nearest settlement of this zone type
	AssetsDir string
	Script    []ScriptStep // scripted input, replaces the keyboard when set
	ShotPath  string       // save a screenshot here and quit
	ShotFrame int          // frame to take the screenshot at
	Store     *save.Store  // the autosave; nil disables saving (test runs)
	NewGame   bool         // start a new game even if a save exists
	Title     bool         // show the title screen first instead of starting right away
	Sound     bool         // play music and sounds (off in test runs)
}

// ErrDone is returned by Update when the game should quit.
var ErrDone = errors.New("quit")

// Game is the game.
type Game struct {
	opts    Options
	world   *world.World
	player  *world.Player
	spotted *gfx.Image
	font    *bitmapFont
	canvas  *gfx.Image
	scene   *scene // the 3D renderer for the map and battles
	frame   int
	done    bool

	char       *character.Character
	checkpoint [2]int   // respawn tile after a defeat
	camp       *camp    // set while resting at a camp fire
	message    []string // lines shown in a box on the map until Space is pressed
	abilities  *abilitiesMenu
	menu       *mainMenu
	saved      persisted
	zone       [2]int // the zone the player was last in
	zoneBanner int    // frames left to show the zone name
	zoneKnown  bool
	title      *titleScreen          // set while the title screen is shown
	comic      map[string]*gfx.Image // comic art by path; nil entries mean "not available"
	loading    func() error          // runs after one frame of the loading screen
	battle     *battleScene
	grace      int // frames left in which no enemy can engage
	audio      *audio.Player
	mapMusic   string // the map tune where the player is (sound.go)
}

// New loads assets and places the player at the spawn point.
func New(opts Options) (*Game, error) {
	g := &Game{
		opts:  opts,
		audio: audio.New(opts.AssetsDir, opts.Sound),
	}

	g.spotted = gfx.NewImageFromImage(spottedBubble())

	fontSheet, err := assets.LoadImage(opts.AssetsDir, assets.Font)
	if err != nil {
		return nil, err
	}
	g.font = newBitmapFont(fontSheet)

	if opts.Title {
		g.title = newTitle(g)
		return g, nil
	}
	if err := g.begin(opts.NewGame); err != nil {
		return nil, err
	}
	return g, nil
}

// begin starts the game: a new one (with the class and seed in the options) or the saved one.
func (g *Game) begin(newGame bool) error {
	g.opts.NewGame = newGame
	g.battle, g.camp, g.message, g.abilities, g.menu, g.grace = nil, nil, nil, nil, nil, 0
	g.zoneKnown = false
	if err := g.loadOrStart(); err != nil {
		return err
	}
	g.title = nil
	return nil
}

// returnToTitle saves and goes back to the title screen.
func (g *Game) returnToTitle() {
	g.autosave()
	g.world, g.player, g.char = nil, nil, nil
	g.title = newTitle(g)
}

// Update advances the game by one tick.
func (g *Game) Update() error {
	if g.done {
		return ErrDone
	}
	err := g.update()
	g.updateAudio()
	return err
}

func (g *Game) update() error {
	g.frame++
	updateHeld()
	if g.loading != nil {
		f := g.loading
		g.loading = nil
		if err := f(); err != nil {
			if g.title == nil {
				g.title = newTitle(g)
			}
			g.title.note = "Couldn't start: " + err.Error()
		}
		return nil
	}
	if g.title != nil {
		g.title.update()
		return nil
	}
	if g.battle != nil {
		g.battle.update()
		if g.battle.finished {
			g.endBattle()
		}
		return nil
	}

	if g.message != nil {
		if g.confirmPressed() || (g.opts.Script != nil && g.frame%90 == 0) {
			g.message = nil
		}
		return nil
	}

	if g.menu != nil {
		if !g.menu.update() {
			g.menu = nil
		}
		return nil
	}
	if g.abilities != nil {
		if !g.abilities.update() {
			g.abilities = nil
			g.autosave()
		}
		return nil
	}

	in := g.readInput()
	if !in.HasDir && !g.player.Moving() && g.menuKeyPressed() {
		g.menu = &mainMenu{g: g}
		g.sound(sndOpen)
		return nil
	}
	if !in.HasDir && !g.player.Moving() && g.abilitiesPressed() {
		g.abilities = &abilitiesMenu{g: g}
		g.sound(sndOpen)
		return nil
	}
	if !in.HasDir && !g.player.Moving() && g.interactPressed() {
		g.interact()
	}
	if g.camp != nil && in.HasDir {
		g.camp = nil // moving breaks camp
	}
	if g.camp == nil && !in.HasDir && g.campPressed() {
		g.startCamp()
	}
	g.player.StaminaUse = g.char.StaminaUse()
	g.player.Update(in, g.world.BlockedForPlayer)
	if g.camp != nil {
		g.char.RegenerateAtCamp()
	} else {
		g.char.Regenerate()
	}
	if g.grace > 0 {
		g.grace--
	}
	if zx, zy := procgen.ZoneOf(g.player.TX, g.player.TY); !g.zoneKnown || [2]int{zx, zy} != g.zone {
		g.zone, g.zoneKnown, g.zoneBanner = [2]int{zx, zy}, true, zoneBannerFrames
	}
	if g.zoneBanner > 0 {
		g.zoneBanner--
	}
	if n := g.world.UpdateNPCs(g.player, g.grace == 0); n != nil {
		g.camp = nil
		g.battle = newBattleScene(g, n)
	}
	if g.frame%periodicSaveFrames == 0 {
		g.autosave()
	}
	if g.frame%60 == 0 {
		cx, cy := world.ChunkCoord(g.player.TX, g.player.TY)
		g.world.Unload(cx, cy, unloadRadius)
		g.world.Prefetch(g.player.TX, g.player.TY)
	}
	return nil
}

// interactPressed reports a press of the interact key on the map (Space / gamepad A).
func (g *Game) interactPressed() bool {
	if g.opts.Script != nil {
		return scriptUse(g.opts.Script, g.frame)
	}
	return g.confirmPressed()
}

// interact uses whatever is on the tile the player faces. For now: inn doors and shrines
// heal and set the checkpoint (design/mechanics.md, Checkpoints).
func (g *Game) interact() {
	dx, dy := g.player.Facing.Delta()
	x, y := g.player.TX+dx, g.player.TY+dy
	switch g.world.CheckpointAt(x, y) {
	case procgen.InnCheckpoint:
		g.char.Heal()
		g.checkpoint = [2]int{g.player.TX, g.player.TY}
		g.message = []string{"You rest at the inn. HP and MP are restored.", "This inn is now your checkpoint."}
		g.sound(sndHeal)
		g.autosave()
	case procgen.ShrineCheckpoint:
		g.char.Heal()
		g.checkpoint = [2]int{g.player.TX, g.player.TY}
		g.message = []string{"You pray at the shrine. HP and MP are restored.", "This shrine is now your checkpoint."}
		g.sound(sndHeal)
		g.autosave()
	}
}

func (g *Game) endBattle() {
	s := g.battle
	g.battle = nil
	g.grace = world.GraceFrames(s.npc)
	defer g.autosave() // XP, money, HP and MP changed
	if s.b.Won() {
		g.world.Defeat(s.npc)
		g.saveDefeated(s.npc)
		return
	}
	// Defeat: back to the checkpoint (HP, MP and money were handled by the battle scene).
	g.world.SendHome(s.npc)
	g.player = world.NewPlayer(g.checkpoint[0], g.checkpoint[1])
}

// rollLoot returns the money an enemy drops, in bronze. Trash drops little and not always;
// elites are the main source (design/items.md, Loot Sources).
func (g *Game) rollLoot(n *world.NPC) int {
	r := float64((uint64(g.frame)*0x9E3779B97F4A7C15^n.Spawn.ID)%1000) / 1000
	lvl := n.Spawn.Level
	if n.Info().Tier == "elite" {
		return lvl * (10 + int(r*16))
	}
	if r < 0.5 {
		return 0
	}
	return lvl * (1 + int((r-0.5)*8))
}

// menuPressed reports a fresh press of a direction (keys or d-pad), for menu navigation.
func (g *Game) menuPressed(d world.Dir) bool {
	if g.opts.Script != nil { // a scripted direction step counts as one press
		s, start := scriptStep(g.opts.Script, g.frame)
		return start && s.Input.HasDir && s.Input.Dir == d
	}
	keys := map[world.Dir][]int32{
		world.Up:    {rl.KeyW, rl.KeyUp},
		world.Down:  {rl.KeyS, rl.KeyDown},
		world.Left:  {rl.KeyA, rl.KeyLeft},
		world.Right: {rl.KeyD, rl.KeyRight},
	}
	pads := map[world.Dir]int32{
		world.Up: rl.GamepadButtonLeftFaceUp, world.Down: rl.GamepadButtonLeftFaceDown,
		world.Left: rl.GamepadButtonLeftFaceLeft, world.Right: rl.GamepadButtonLeftFaceRight,
	}
	if keyPressed(keys[d]...) || padPressed(pads[d]) {
		g.sound(sndMove)
		return true
	}
	return false
}

func (g *Game) confirmPressed() bool {
	if g.opts.Script != nil {
		return scriptUse(g.opts.Script, g.frame)
	}
	return keyPressed(rl.KeySpace) || padPressed(rl.GamepadButtonRightFaceDown) // A
}

func (g *Game) readInput() world.Input {
	if g.opts.Script != nil {
		return scriptInput(g.opts.Script, g.frame)
	}
	var in world.Input
	keys := []struct {
		dir  world.Dir
		keys []int32
		pad  int32
	}{
		{world.Up, []int32{rl.KeyW, rl.KeyUp}, rl.GamepadButtonLeftFaceUp},
		{world.Down, []int32{rl.KeyS, rl.KeyDown}, rl.GamepadButtonLeftFaceDown},
		{world.Left, []int32{rl.KeyA, rl.KeyLeft}, rl.GamepadButtonLeftFaceLeft},
		{world.Right, []int32{rl.KeyD, rl.KeyRight}, rl.GamepadButtonLeftFaceRight},
	}
	// The most recently pressed direction wins, so switching direction feels responsive.
	newest := -1
	for _, k := range keys {
		held, since := false, math.MaxInt
		for _, key := range k.keys {
			if d := heldKeys[key]; d > 0 {
				held, since = true, min(since, d)
			}
		}
		if d := heldPads[k.pad]; d > 0 {
			held, since = true, min(since, d)
		}
		if held && (newest < 0 || since < newest) {
			in.Dir, in.HasDir, newest = k.dir, true, since
		}
	}
	// Y on an Xbox-style pad is the top face button.
	in.Run = rl.IsKeyDown(rl.KeyLeftShift) || rl.IsKeyDown(rl.KeyRightShift) || padDown(rl.GamepadButtonRightFaceUp)
	return in
}

// Draw renders a frame: the 3D scene at full resolution, then the pixel UI, drawn on a
// small canvas and scaled up by whole numbers, on top.
func (g *Game) Draw() {
	sw, sh := rl.GetRenderWidth(), rl.GetRenderHeight()
	scale := max(1, min(sw/ScreenW, sh/ScreenH))
	cw, ch := (sw+scale-1)/scale, (sh+scale-1)/scale
	if g.canvas == nil || g.canvas.Bounds().Dx() != cw || g.canvas.Bounds().Dy() != ch {
		if g.canvas != nil {
			g.canvas.Deallocate()
		}
		g.canvas = gfx.NewImage(cw, ch)
	}
	g.canvas.Clear()
	switch {
	case g.loading != nil:
		drawLoading(g, g.canvas)
	case g.title != nil:
		g.title.draw(g.canvas)
	case g.battle != nil:
		g.battle.draw(g.canvas)
	default:
		g.drawMapOverlay(g.canvas, float64(scale))
		g.drawHUD(g.canvas)
		if g.abilities != nil {
			g.abilities.draw(g.canvas)
		}
		if g.menu != nil {
			g.menu.draw(g.canvas)
		}
	}
	gfx.Flush()

	rl.ClearBackground(color.RGBA{30, 40, 30, 255})
	switch {
	case g.loading != nil:
	case g.title != nil:
		g.scene3d().drawBackdrop()
		g.title.drawAtmosphere(scale, cw, ch)
	case g.battle != nil:
		g.scene3d().drawBattle(g.battle)
	default:
		g.scene3d().drawMap(g)
	}
	op := &gfx.DrawImageOptions{}
	op.GeoM.Scale(float64(scale), float64(scale))
	gfx.Screen().DrawImage(g.canvas, op)
	gfx.Flush()
	if g.title != nil && g.loading == nil && !g.title.preview.Empty() {
		r := g.title.preview.Inset(3)
		g.scene3d().drawPreview(classes[g.title.class], int32(r.Min.X*scale), int32(r.Min.Y*scale), int32(r.Dx()*scale), int32(r.Dy()*scale))
	}
}

// ShotDue reports whether the screenshot of a test run should be taken now; the caller
// saves the frame that was just drawn and the game quits.
func (g *Game) ShotDue() bool {
	if g.opts.ShotPath != "" && g.frame >= g.opts.ShotFrame && !g.done {
		g.done = true
		return true
	}
	return false
}

// drawHUD shows level, HP, MP and money on the adventure map, and stamina while it isn't full.
func (g *Game) drawHUD(dst *gfx.Image) {
	c, st := g.char, g.char.Stats()
	const x, y, w, h = 4, 4, 92, 47
	panel(dst, x, y, w, h)
	g.font.Draw(dst, fmt.Sprintf("Lv%d %s", c.Level, c.Class.Info().Name), x+5, y+4, ink)
	hpFrac := c.HP / float64(st.MaxHP)
	bar(dst, x+5, y+14, w-10, hpFrac, hpColor(hpFrac))
	bar(dst, x+5, y+21, w-10, c.MP/float64(max(1, st.MaxMP)), mpColor)
	bar(dst, x+5, y+28, w-10, float64(c.XP)/float64(character.XPToNext(c.Level)), xpColor)
	money := formatMoney(c.Money)
	g.font.Draw(dst, money, x+w-6-g.font.Width(money), y+35, ink)

	if g.camp != nil {
		msg := "Resting at the camp fire."
		vw, vh := dst.Bounds().Dx(), dst.Bounds().Dy()
		mw := g.font.Width(msg) + 12
		panel(dst, (vw-mw)/2, vh-LineHeight-12, mw, LineHeight+8)
		g.font.Draw(dst, msg, (vw-mw)/2+6, vh-LineHeight-8, ink)
	}
	if g.zoneBanner > 0 {
		l := procgen.Layout(g.world.Seed, g.zone[0], g.zone[1])
		sub := fmt.Sprintf("%s  -  Difficulty %d", l.Type, g.world.Difficulty(g.zone[0], g.zone[1]))
		vw := dst.Bounds().Dx()
		w := max(g.font.Width(l.Name), g.font.Width(sub)) + 20
		x := (vw - w) / 2
		panel(dst, x, 6, w, 2*LineHeight+10)
		g.font.Draw(dst, l.Name, (vw-g.font.Width(l.Name))/2, 11, ink)
		g.font.Draw(dst, sub, (vw-g.font.Width(sub))/2, 11+LineHeight, inkFaded)
	}
	if g.message != nil {
		vw, vh := dst.Bounds().Dx(), dst.Bounds().Dy()
		h := len(g.message)*LineHeight + 12
		panel(dst, 4, vh-h-4, vw-8, h)
		for i, l := range g.message {
			g.font.Draw(dst, l, 12, vh-h+2+i*LineHeight, ink)
		}
	}

	p := g.player
	if p.Stamina >= world.StaminaMax && !p.Running() {
		return
	}
	frac := float32(p.Stamina / world.StaminaMax)
	sc := color.RGBA{120, 210, 90, 255}
	if frac < 0.3 {
		sc = color.RGBA{230, 120, 60, 255}
	}
	gfx.FillRect(dst, x, y+h+2, w, 4, color.RGBA{20, 20, 30, 200})
	gfx.FillRect(dst, x+1, y+h+3, (w-2)*frac, 2, sc)
}

func bottom(o procgen.Object) int { return o.Y + tiles.Get(o.Sprite).H - 1 }

// spottedBubble draws the "!" shown over an enemy that just noticed the player: a round
// speech bubble with a tail, 15×16 pixels, the tail's tip at the bottom middle.
func spottedBubble() image.Image {
	img := image.NewRGBA(image.Rect(0, 0, 15, 16))
	paper, rim, mark := color.RGBA{250, 240, 220, 255}, color.RGBA{60, 40, 30, 255}, color.RGBA{200, 40, 30, 255}
	for y := 0; y < 16; y++ {
		for x := 0; x < 15; x++ {
			switch d := math.Hypot(float64(x)-7, float64(y)-6); {
			case d <= 5.5:
				img.Set(x, y, paper)
			case d <= 6.6:
				img.Set(x, y, rim)
			}
		}
	}
	// The tail, pointing down.
	for _, p := range [][2]int{{6, 12}, {7, 12}, {8, 12}, {7, 13}} {
		img.Set(p[0], p[1], paper)
	}
	for _, p := range [][2]int{{5, 12}, {9, 12}, {6, 13}, {8, 13}, {7, 14}} {
		img.Set(p[0], p[1], rim)
	}
	// The "!": a bar narrowing downward, then a dot.
	for y := 2; y <= 7; y++ {
		img.Set(7, y, mark)
		if y <= 4 {
			img.Set(6, y, mark)
			img.Set(8, y, mark)
		}
	}
	img.Set(7, 9, mark)
	return img
}
