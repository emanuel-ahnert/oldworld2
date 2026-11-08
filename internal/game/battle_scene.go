package game

import (
	"fmt"
	"image/color"
	"math"
	"slices"

	"oldworld2/internal/battle"
	"oldworld2/internal/character"
	"oldworld2/internal/gfx"
	"oldworld2/internal/procgen"
	"oldworld2/internal/world"
)

type battlePhase int

const (
	phaseIntro battlePhase = iota
	phaseChoose
	phasePlay
	phaseResult
)

const (
	introFrames    = 30 // a pause before the first message
	autoAdvance    = 70 // scripted runs advance messages after this many frames
	hpAnimPerFrame = 1.5
	flashFrames    = 24
)

// battlePalette colors the battle background for a zone type.
type battlePalette struct{ sky, ground, horizon, platform color.RGBA }

var battlePalettes = map[procgen.ZoneType]battlePalette{
	procgen.Grassland:     {color.RGBA{214, 238, 246, 255}, color.RGBA{200, 226, 150, 255}, color.RGBA{150, 190, 110, 255}, color.RGBA{150, 186, 100, 255}},
	procgen.Desert:        {color.RGBA{250, 234, 206, 255}, color.RGBA{242, 204, 146, 255}, color.RGBA{222, 170, 104, 255}, color.RGBA{222, 168, 98, 255}},
	procgen.Mediterranean: {color.RGBA{190, 222, 246, 255}, color.RGBA{214, 206, 134, 255}, color.RGBA{170, 160, 96, 255}, color.RGBA{190, 178, 110, 255}},
	procgen.North:         {color.RGBA{214, 226, 240, 255}, color.RGBA{236, 240, 246, 255}, color.RGBA{170, 186, 204, 255}, color.RGBA{200, 210, 222, 255}},
}

func init() {
	battlePalettes[procgen.AncientDesert] = battlePalettes[procgen.Desert]
}

var (
	ink      = color.RGBA{60, 40, 30, 255}
	inkFaded = color.RGBA{170, 150, 135, 255}
	paper    = color.RGBA{250, 240, 220, 255}
)

// battleScene shows one battle and collects the player's choices.
type battleScene struct {
	g      *Game
	b      *battle.Battle
	npc    *world.NPC
	tier   string
	zone   procgen.ZoneType
	phase  battlePhase
	frame  int // frames in the current phase or message
	cursor int

	messages []battle.Event // queued events of the current turn
	serial   int            // counts started messages, so the 3D scene animates each once
	shown    struct{ playerHP, playerMP, enemyHP float64 }
	flash    map[*battle.Combatant]int

	plan     string   // the demo bot's chosen move (demo.go)
	result   []string // lines shown after the battle
	resultSn []string // the sound of each result line ("" for none)
	finished bool
}

func newBattleScene(g *Game, npc *world.NPC) *battleScene {
	if g.demo != nil {
		demoSlotOpeners(g.char)
	}
	player := g.char.Combatant()
	enemy := character.Enemy(npc.Spawn.Kind, npc.Spawn.Level)
	s := &battleScene{
		g: g, npc: npc, tier: npc.Info().Tier,
		zone:  procgen.ZoneTypeAt(g.world.Seed, g.player.TX, g.player.TY),
		b:     battle.New(player, enemy, npc.Spawn.ID^uint64(g.frame)),
		flash: map[*battle.Combatant]int{},
	}
	s.shown.playerHP, s.shown.playerMP, s.shown.enemyHP = float64(player.HP), float64(player.MP), float64(enemy.HP)
	appear := "A " + enemy.Name + " attacks!"
	if s.tier == "trash" {
		appear = "A wild " + enemy.Name + " appears!"
	}
	s.messages = []battle.Event{{Text: appear, PlayerHP: player.HP, PlayerMP: player.MP, EnemyHP: enemy.HP}}
	s.messages = append(s.messages, s.b.Start()...) // passives that act as the battle begins
	return s
}

// menuSlots is how many active abilities the menu has room for, in two columns; the row
// below always holds the free basic move and Wait.
const menuSlots = 4

// options are the menu cells in reading order: the active abilities (empty strings for
// unused slots), then the basic move (empty if the class has none) and Wait.
func (s *battleScene) options() []string {
	var cells, basic []string
	for _, id := range s.b.MenuOptions(s.b.Player) {
		switch a := battle.Get(id); {
		case id == battle.WaitID:
		case a.Fallback:
			basic = append(basic, id)
		case len(cells) < menuSlots:
			cells = append(cells, id)
		}
	}
	for len(cells) < menuSlots {
		cells = append(cells, "")
	}
	if len(basic) == 0 {
		basic = []string{""}
	}
	return append(cells, basic[0], battle.WaitID)
}

func (s *battleScene) update() {
	s.frame++
	for c, f := range s.flash {
		if f > 0 {
			s.flash[c] = f - 1
		}
	}
	s.animateBars()

	switch s.phase {
	case phaseIntro:
		if s.frame >= introFrames {
			s.phase, s.frame = phasePlay, 0
		}
	case phasePlay:
		if !s.advance() {
			return
		}
		s.messages = s.messages[1:]
		s.frame = 0
		if len(s.messages) > 0 {
			s.startMessage()
			return
		}
		if s.b.Over() {
			s.finish()
			return
		}
		s.phase = phaseChoose
	case phaseChoose:
		s.updateMenu()
	case phaseResult:
		if !s.advance() {
			return
		}
		s.result, s.resultSn = s.result[1:], s.resultSn[1:]
		s.frame = 0
		if len(s.result) == 0 {
			s.finished = true
		} else {
			s.g.sound(s.resultSn[0])
		}
	}
}

// advance reports whether the current message should be closed.
func (s *battleScene) advance() bool {
	if s.frame < 12 { // ignore presses that started the previous message
		return false
	}
	if s.g.automated() {
		return s.frame >= autoAdvance
	}
	return s.g.confirmPressed()
}

func (s *battleScene) startMessage() {
	s.serial++
	s.g.battleEventSound(s.messages[0])
	if hit := s.messages[0].Hit; hit != nil {
		s.flash[hit] = flashFrames
	}
}

func (s *battleScene) animateBars() {
	if len(s.messages) == 0 || s.phase != phasePlay {
		return
	}
	ev := s.messages[0]
	step := func(v *float64, target int, speed float64) {
		t := float64(target)
		if math.Abs(*v-t) <= speed {
			*v = t
		} else if *v < t {
			*v += speed
		} else {
			*v -= speed
		}
	}
	step(&s.shown.playerHP, ev.PlayerHP, hpAnimPerFrame)
	step(&s.shown.playerMP, ev.PlayerMP, hpAnimPerFrame*2)
	step(&s.shown.enemyHP, ev.EnemyHP, hpAnimPerFrame)
}

func (s *battleScene) updateMenu() {
	opts := s.options()
	if s.cursor >= len(opts) || opts[s.cursor] == "" {
		s.cursor = slices.IndexFunc(opts, func(id string) bool { return id != "" })
	}
	if s.g.demo != nil {
		s.demoChoice(opts)
		return
	}
	if s.g.opts.Script != nil {
		if s.frame < 60 { // long enough to see the menu in test screenshots
			return
		}
		// Scripted runs pick the first usable ability.
		for _, id := range opts {
			if ok, _ := s.b.Usable(s.b.Player, id); id != "" && (ok || id == battle.WaitID) {
				s.choose(id)
				return
			}
		}
	}
	// The grid has two columns; moves skip empty slots and stop at the edges.
	moveTo := func(steps ...int) {
		for _, i := range steps {
			if i >= 0 && i < len(opts) && opts[i] != "" {
				s.cursor = i
				return
			}
		}
	}
	col := s.cursor % 2
	switch {
	case s.g.menuPressed(world.Left), s.g.menuPressed(world.Right):
		moveTo(s.cursor - col + 1 - col)
	case s.g.menuPressed(world.Down):
		var below []int
		for i := s.cursor + 2; i < len(opts); i += 2 {
			below = append(below, i, i-col+1-col) // the same column, else the other one
		}
		moveTo(below...)
	case s.g.menuPressed(world.Up):
		var above []int
		for i := s.cursor - 2; i >= 0; i -= 2 {
			above = append(above, i, i-col+1-col)
		}
		moveTo(above...)
	case s.g.confirmPressed():
		id := opts[s.cursor]
		if ok, _ := s.b.Usable(s.b.Player, id); ok || id == battle.WaitID {
			s.choose(id)
		} else {
			s.g.sound(sndCancel)
		}
	}
}

// demoChoice is the demo bot's turn: it plans its move, shows it with the cursor for a moment
// so viewers can follow, then uses it.
func (s *battleScene) demoChoice(opts []string) {
	switch s.frame {
	case 20:
		s.plan = demoOpener(s.b)
		if s.plan == "" {
			s.plan = s.b.PlanPlayerMove(uint64(s.g.frame))
		}
		if i := slices.Index(opts, s.plan); i >= 0 {
			s.cursor = i
		}
	case 55:
		s.choose(s.plan)
	}
}

func (s *battleScene) choose(id string) {
	s.messages = s.b.Turn(id, s.b.ChooseEnemyAbility())
	s.phase, s.frame = phasePlay, 0
	if len(s.messages) > 0 {
		s.startMessage()
	}
}

// finish hands out rewards or applies the defeat, and queues the result lines.
func (s *battleScene) finish() {
	ch := s.g.char
	ch.HP, ch.MP = float64(s.b.Player.HP), float64(s.b.Player.MP)
	s.phase, s.frame = phaseResult, 0
	if !s.b.Won() {
		money, xp := ch.Defeated()
		s.addResult("You were defeated...", "")
		s.addResult(fmt.Sprintf("You lose %s and %d XP.", formatMoney(money), xp), "")
		s.addResult("You wake up at the checkpoint.", "")
		return
	}
	xp := character.EnemyXP(s.npc.Spawn.Level, ch.Level, s.tier)
	coins := s.g.rollLoot(s.npc)
	ch.Money += coins
	if xp > 0 {
		s.addResult(fmt.Sprintf("You gain %d XP.", xp), "")
	} else {
		s.addResult("You gain no XP: this enemy is far below your level.", "")
	}
	if coins > 0 {
		s.addResult(fmt.Sprintf("You find %s.", formatMoney(coins)), sndCoins)
	}
	for _, up := range ch.GainXP(xp) {
		s.addResult(fmt.Sprintf("You reach level %d!", up.Level), sndLevel)
		for _, id := range up.Learned {
			if a := battle.Get(id); a != nil {
				s.addResult(fmt.Sprintf("You learned %s!", a.Name), sndLearn)
				if ch.Slottable(a.ID) && !slices.Contains(ch.Active, a.ID) {
					s.addResult("Your ability slots are full. Press N to swap it in.", "")
				}
			}
		}
	}
}

// addResult queues a line for after the battle, with a sound played when it shows; the
// first line's sound plays right away.
func (s *battleScene) addResult(line, sound string) {
	s.result, s.resultSn = append(s.result, line), append(s.resultSn, sound)
	if len(s.result) == 1 {
		s.g.sound(sound)
	}
}

// ---- drawing ----

// Battle layout in canvas units (the small internal resolution). The fighters are drawn in
// 3D (scene.drawBattle); the info boxes and the message box on the canvas on top (draw).
type battleLayout struct {
	vw, vh, boxH, field int
}

func (s *battleScene) layout(vw, vh int) battleLayout {
	l := battleLayout{vw: vw, vh: vh, boxH: 4*LineHeight + 12}
	l.field = vh - l.boxH
	return l
}

// draw draws the info boxes and the message or menu box onto the canvas.
func (s *battleScene) draw(dst *gfx.Image) {
	l := s.layout(dst.Bounds().Dx(), dst.Bounds().Dy())
	if s.phase != phaseIntro {
		s.drawEnemyInfo(dst)
		s.drawPlayerInfo(dst, l.vw, l.field)
	}
	s.drawBox(dst, l.vw, l.vh, l.boxH)
}

func (s *battleScene) drawEnemyInfo(dst *gfx.Image) {
	e := s.b.Enemy
	// The name on its own row, so long names fit; the level and the tier below it.
	x, y, w := 6, 6, max(96, s.g.font.Width(e.Name)+10)
	panel(dst, x, y, w, 24+LineHeight)
	s.g.font.Draw(dst, e.Name, x+5, y+4, ink)
	lv := fmt.Sprintf("lvl %d", e.Level)
	if s.tier == "elite" {
		lv += " elite"
	}
	s.g.font.Draw(dst, lv, x+5, y+4+LineHeight, ink)
	bar(dst, x+5, y+15+LineHeight, w-10, s.shown.enemyHP/float64(e.Base.MaxHP), hpColor(s.shown.enemyHP/float64(e.Base.MaxHP)))
	s.drawEffects(dst, s.effects(e), x, y+26+LineHeight, w+40, false)
}

func (s *battleScene) drawPlayerInfo(dst *gfx.Image, vw, field int) {
	p := s.b.Player
	nums := fmt.Sprintf("%d/%d HP  %d MP", int(math.Ceil(s.shown.playerHP)), p.Base.MaxHP, int(s.shown.playerMP))
	w := max(104, s.g.font.Width(nums)+10)
	x, y := vw-w-6, field-42
	panel(dst, x, y, w, 38)
	s.g.font.Draw(dst, s.g.char.Class.Info().Name, x+5, y+4, ink)
	lv := fmt.Sprintf("Lv%d", p.Level)
	s.g.font.Draw(dst, lv, x+w-5-s.g.font.Width(lv), y+4, ink)
	hpFrac := s.shown.playerHP / float64(p.Base.MaxHP)
	bar(dst, x+5, y+15, w-10, hpFrac, hpColor(hpFrac))
	bar(dst, x+5, y+21, w-10, s.shown.playerMP/float64(max(1, p.Base.MaxMP)), mpColor)
	s.g.font.Draw(dst, nums, x+5, y+28, ink)
	s.drawEffects(dst, s.effects(p), x, y-2, w, true)
}

// effects returns a combatant's buffs, debuffs and statuses as of the message being shown:
// while a turn's messages play, the turn is already resolved, so its later effects mustn't
// show yet. The opening message comes before anything has happened.
func (s *battleScene) effects(c *battle.Combatant) []battle.Effect {
	if s.phase != phasePlay || len(s.messages) == 0 {
		return c.Effects()
	}
	if c == s.b.Player {
		return s.messages[0].PlayerEffects
	}
	return s.messages[0].EnemyEffects
}

// drawEffects shows buffs (green) and debuffs and statuses (red) as small labels in rows
// next to an info box. above stacks the rows upward from y instead of downward.
func (s *battleScene) drawEffects(dst *gfx.Image, effects []battle.Effect, x, y, maxW int, above bool) {
	const chipH = LineHeight + 2
	cx, row := x, 0
	for _, e := range effects {
		w := s.g.font.Width(e.Label) + 8
		if cx > x && cx+w > x+maxW {
			cx, row = x, row+1
		}
		cy := y + row*(chipH+2)
		if above {
			cy = y - chipH - row*(chipH+2)
		}
		col := color.RGBA{50, 130, 50, 255}
		if !e.Good {
			col = color.RGBA{180, 50, 40, 255}
		}
		gfx.FillRect(dst, float32(cx), float32(cy), float32(w), chipH, paper)
		gfx.StrokeRect(dst, float32(cx)+0.5, float32(cy)+0.5, float32(w)-1, chipH-1, 1, col)
		s.g.font.Draw(dst, e.Label, cx+4, cy+2, col)
		cx += w + 2
	}
}

func (s *battleScene) drawBox(dst *gfx.Image, vw, vh, boxH int) {
	x, y, w := 4, vh-boxH-4, vw-8
	panel(dst, x, y, w, boxH)
	tx, ty := x+8, y+8

	switch {
	case s.phase == phaseResult && len(s.result) > 0:
		s.g.font.Draw(dst, s.result[0], tx, ty, ink)
	case s.phase == phasePlay && len(s.messages) > 0:
		s.g.font.Draw(dst, s.messages[0].Text, tx, ty, ink)
	case s.phase == phaseChoose:
		s.drawMenu(dst, x, y, w, boxH)
	}
}

func (s *battleScene) drawMenu(dst *gfx.Image, x, y, w, h int) {
	opts := s.options()
	if s.cursor >= len(opts) || opts[s.cursor] == "" {
		s.cursor = slices.IndexFunc(opts, func(id string) bool { return id != "" })
	}
	colW := 0
	for _, id := range opts {
		if id != "" {
			colW = max(colW, s.g.font.Width(battle.Get(id).Name)+14)
		}
	}
	for i, id := range opts {
		cx, cy := x+14+(i%2)*colW, y+7+(i/2)*LineHeight
		if id == "" {
			s.g.font.Draw(dst, "-", cx, cy, inkFaded) // an empty ability slot
			continue
		}
		a := battle.Get(id)
		c := ink
		if ok, _ := s.b.Usable(s.b.Player, id); !ok && id != battle.WaitID {
			c = inkFaded
		}
		if i == s.cursor {
			s.g.font.Draw(dst, ">", cx-9, cy, ink)
		}
		s.g.font.Draw(dst, a.Name, cx, cy, c)
	}

	// Details of the selected move on the right, wrapped to the space left.
	dx := x + 14 + 2*colW + 4
	dw := x + w - 8 - dx
	gfx.FillRect(dst, float32(dx-6), float32(y+4), 1, float32(h-8), inkFaded)
	a := battle.Get(opts[s.cursor])
	var lines []string
	if a.ID != battle.WaitID {
		cost := "Free"
		if a.Cost > 0 {
			cost = fmt.Sprintf("%d MP", a.Cost)
		}
		if a.Power > 0 {
			lines = append(lines, fmt.Sprintf("%s  Power %d", a.Element, a.Power))
		}
		if a.Priority > 0 {
			cost += fmt.Sprintf("  Priority +%d", a.Priority)
		}
		lines = append(lines, cost)
	}
	desc := a.Text
	if ok, why := s.b.Usable(s.b.Player, a.ID); !ok && why != "" {
		desc = why
	}
	lines = append(lines, s.g.font.Wrap(desc, dw)...)
	maxLines := (h - 10) / LineHeight
	if len(lines) > maxLines {
		lines = lines[:maxLines]
	}
	for i, l := range lines {
		s.g.font.Draw(dst, l, dx, y+7+i*LineHeight, ink)
	}
}

// ---- small drawing helpers ----

func panel(dst *gfx.Image, x, y, w, h int) {
	gfx.FillRect(dst, float32(x), float32(y), float32(w), float32(h), paper)
	gfx.StrokeRect(dst, float32(x)+1, float32(y)+1, float32(w)-2, float32(h)-2, 2, ink)
}

func bar(dst *gfx.Image, x, y, w int, frac float64, c color.Color) {
	frac = math.Max(0, math.Min(1, frac))
	gfx.FillRect(dst, float32(x), float32(y), float32(w), 5, color.RGBA{70, 60, 55, 255})
	gfx.FillRect(dst, float32(x)+1, float32(y)+1, float32(float64(w-2)*frac), 3, c)
}

var (
	mpColor = color.RGBA{80, 130, 230, 255}
	xpColor = color.RGBA{150, 80, 210, 255}
)

// thinBar is a 3-pixel bar for secondary values like XP.
func hpColor(frac float64) color.Color {
	switch {
	case frac > 0.5:
		return color.RGBA{90, 200, 90, 255}
	case frac > 0.2:
		return color.RGBA{230, 190, 50, 255}
	default:
		return color.RGBA{220, 70, 50, 255}
	}
}

// formatMoney shows bronze as gold, silver and bronze amounts, each followed by its coin icon
// (design/items.md, Currency).
func formatMoney(bronze int) string {
	g, s, b := bronze/10000, bronze/100%100, bronze%100
	switch {
	case g > 0:
		return fmt.Sprintf("%d%c %d%c %d%c", g, coinGold, s, coinSilver, b, coinBronze)
	case s > 0:
		return fmt.Sprintf("%d%c %d%c", s, coinSilver, b, coinBronze)
	default:
		return fmt.Sprintf("%d%c", b, coinBronze)
	}
}
