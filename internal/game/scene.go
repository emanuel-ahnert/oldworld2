package game

import (
	"fmt"
	"image/color"
	"math"
	"os"
	"path/filepath"
	"strings"

	rl "github.com/gen2brain/raylib-go/raylib"

	"oldworld2/internal/battle"
	"oldworld2/internal/character"
	"oldworld2/internal/creature"
	"oldworld2/internal/gfx"
	"oldworld2/internal/procgen"
	"oldworld2/internal/r3d"
	"oldworld2/internal/tiles"
	"oldworld2/internal/world"
)

// The 3D scene: the adventure map and battles, drawn with r3d. The game logic still works on
// the tile grid; one tile is one meter, tile (x, y) covers world x..x+1 and z..z+1 (+z is
// south, toward the camera).

const (
	frameDT   = float32(1.0 / 60)
	walkPace  = 1.6 // walk animations play faster, to match the faster movement
	runPace   = 1.7 // the run animation's stride is shorter for its speed
	maskTiles = 72  // the ground mask covers this many tiles around the player
	viewX     = 17  // tiles drawn left and right of the player
	viewUp    = 16  // tiles drawn north of the player (further from the camera)
	viewDown  = 10  // and south
)

// look is how a creature kind appears: a model scaled to a height in meters, tinted, or a
// humanoid built from outfit parts. Kinds without a matching model in the packs use the
// closest one, recolored.
type look struct {
	model  string
	height float32
	tint   color.RGBA
	hover  float32 // flying creatures float this high
	outfit *r3d.Outfit
	weapon string
	shield string  // for humans: a model on the left forearm
	fur    string  // surface texture in assets/generated/creatures, for models without one
	furPM  float32 // its repeats per meter
}

var looks = map[creature.Kind]look{
	creature.Wolf:         {model: "../generated/creatures/Wolf.gltf", height: 1.12, tint: color.RGBA{255, 255, 255, 255}},   // tools/models/beasts.py
	creature.Boar:         {model: "../generated/creatures/Boar.gltf", height: 0.92, tint: color.RGBA{255, 255, 255, 255}},   // tools/models/beasts.py
	creature.Goblin:       {model: "../generated/creatures/Goblin.gltf", height: 1.12, tint: color.RGBA{255, 255, 255, 255}}, // tools/models/goblin.py
	creature.Bandit:       {outfit: &banditOutfit, weapon: "../generated/props/Axe.gltf"},
	creature.SandSnake:    {model: "../generated/creatures/SandSnake.gltf", height: 0.75, tint: color.RGBA{255, 255, 255, 255}},               // tools/models/snake.py
	creature.Scorpion:     {model: "../generated/creatures/Scorpion.gltf", height: 1.0, tint: color.RGBA{255, 255, 255, 255}},                 // tools/models/scorpion.py
	creature.EmberSpirit:  {model: "../generated/creatures/EmberSpirit.gltf", height: 1.2, tint: color.RGBA{255, 255, 255, 255}, hover: 0.35}, // tools/models/ember.py
	creature.NomadRaider:  {outfit: &nomadOutfit, weapon: "../generated/props/Scimitar.gltf"},
	creature.Mummy:        {model: "../generated/creatures/Mummy.gltf", height: 1.85, tint: color.RGBA{255, 255, 255, 255}},                 // tools/models/mummy.py
	creature.Scarab:       {model: "../generated/creatures/Scarab.gltf", height: 0.75, tint: color.RGBA{255, 255, 255, 255}},                // tools/models/scarab.py
	creature.Cobra:        {model: "../generated/creatures/Cobra.gltf", height: 1.3, tint: color.RGBA{255, 255, 255, 255}},                  // tools/models/snake.py
	creature.TombGuardian: {model: "../generated/creatures/TombGuardian.gltf", height: 2.3, tint: color.RGBA{255, 255, 255, 255}},           // tools/models/guardian.py
	creature.SnowWolf:     {model: "../generated/creatures/SnowWolf.gltf", height: 1.12, tint: color.RGBA{255, 255, 255, 255}},              // tools/models/beasts.py
	creature.FrostWisp:    {model: "../generated/creatures/FrostWisp.gltf", height: 1.2, tint: color.RGBA{255, 255, 255, 255}, hover: 0.35}, // tools/models/ember.py
	creature.IceTroll:     {model: "../generated/creatures/IceTroll.gltf", height: 2.3, tint: color.RGBA{255, 255, 255, 255}},               // tools/models/troll.py
	creature.Berserker:    {outfit: &berserkerOutfit, weapon: "../generated/props/Axe_Dane.gltf"},
	creature.GiantCrab:    {model: "../generated/creatures/GiantCrab.gltf", height: 0.6, tint: color.RGBA{255, 255, 255, 255}},              // tools/models/crab.py
	creature.WindSpirit:   {model: "../generated/creatures/WindSpirit.gltf", height: 1.2, tint: color.RGBA{255, 255, 255, 255}, hover: 0.4}, // tools/models/ember.py
	creature.Corsair:      {outfit: &corsairOutfit, weapon: "../generated/props/Scimitar.gltf"},
	creature.Hoplite:      {outfit: &hopliteOutfit, weapon: "../generated/props/Spear.gltf", shield: "../generated/props/Shield_Hoplite.gltf"},
	creature.BatSwarm:     {model: "../generated/creatures/BatSwarm.gltf", height: 1.0, tint: color.RGBA{255, 255, 255, 255}, hover: 0.3}, // tools/models/bats.py
	creature.DireWolf:     {model: "../generated/creatures/DireWolf.gltf", height: 1.3, tint: color.RGBA{255, 255, 255, 255}},             // tools/models/beasts.py
	creature.Cultist:      {outfit: &cultistOutfit, weapon: "../generated/props/Dagger.gltf"},
	creature.Vampire:      {outfit: &vampireOutfit},
}

var (
	femaleHead = []string{"Hair_Long", "Eyebrows_Female"}
)

// humans is where tools/models/human.py writes its bodies, hair and outfit pieces.
const humans = "../generated/humans/"

// Outfits: the parts per equipment slot (see r3d.Outfit).
var (
	banditOutfit = r3d.Outfit{Body: humans + "Male_Body", Head: humans + "Bandit_Hood", Chest: humans + "Bandit_Tunic",
		Arms: humans + "Bandit_Arms", Legs: humans + "Bandit_Legs", Feet: humans + "Bandit_Boots", Shoulders: humans + "Bandit_Coat",
		ShowBody: true} // art/concept/bandit_s51.png
	nomadOutfit = r3d.Outfit{Body: humans + "Male_Body", Head: humans + "Nomad_Hood", Chest: humans + "Nomad_Tunic",
		Arms: humans + "Nomad_Arms", Legs: humans + "Nomad_Legs", Feet: humans + "Nomad_Boots", Shoulders: humans + "Nomad_Cape",
		ShowBody: true} // art/concept/nomad_s51.png
	berserkerOutfit = r3d.Outfit{Body: humans + "Male_Body", Head: humans + "Berserker_Head", Chest: humans + "Berserker_Harness",
		Arms: humans + "Berserker_Arms", Legs: humans + "Berserker_Legs", Feet: humans + "Boots_Fur", Shoulders: humans + "Berserker_Pelt",
		ShowBody: true} // art/concept/berserker_s51.png
	corsairOutfit = r3d.Outfit{Body: humans + "Male_Body", Head: humans + "Corsair_Hat", Chest: humans + "Corsair_Shirt",
		Arms: humans + "Corsair_Arms", Legs: humans + "Corsair_Legs", Feet: humans + "Corsair_Boots", Shoulders: humans + "Corsair_Coat",
		ShowBody: true} // art/concept/corsair_s51.png
	hopliteOutfit = r3d.Outfit{Body: humans + "Male_Body", Head: humans + "Hoplite_Helm", Chest: humans + "Hoplite_Cuirass",
		Arms: humans + "Hoplite_Arms", Legs: humans + "Hoplite_Legs", Feet: humans + "Hoplite_Feet", Shoulders: humans + "Hoplite_Mantle",
		ShowBody: true} // art/concept/hoplite_s51.png
	cultistOutfit = r3d.Outfit{Body: humans + "Male_Body_Robed", Head: humans + "Cultist_Hood", Chest: humans + "Cultist_Robe",
		Arms: humans + "Cultist_Arms", Feet: humans + "Shoes", Shoulders: humans + "Cultist_Cloak", ShowBody: true} // art/concept/cultist_s51.png
	vampireOutfit = r3d.Outfit{Body: humans + "Male_Body_Pale", Hair: []string{humans + "Hair_Long_Black"}, Chest: humans + "Vampire_Coat",
		Arms: humans + "Vampire_Arms", Legs: humans + "Vampire_Legs", Feet: humans + "Vampire_Boots", Shoulders: humans + "Vampire_Cape",
		ShowBody: true} // art/concept/vampire_s51.png
)

// staffCast is how staff wielders cast: a thrust of the staff hand toward the enemy, the
// middle part of a punch. The pack's spell cast opens both hands, which would drop the staff.
const staffCast = "Punch_Cross"

var castFrom, castTo float32 = 0.29, 0.62

// The battle camera's vertical field of view and how far it looks down, in degrees. With
// these, the horizon of the ground lies at 0.5 * (1 - tan(pitch) / tan(fovy/2)) of the screen
// height from the top: a quarter.
const (
	battleFovy        = 36
	battleCameraPitch = 9.2
)

// castSpeed slows casting down, so a spell reads as a deliberate gesture, not a jab.
const castSpeed = 0.5

// heroLook is how a class looks: an outfit and what it holds.
type heroLook struct {
	outfit r3d.Outfit
	weapon string // in the right hand, held like the sword
	shield string // on the left forearm: a model in assets/generated/props, "" for none
	bow    bool   // a bow in the left hand
	ready  string // battle stance, if not the sword stance
	cast   string // casting move, if not the open-handed spell cast
}

// Most classes wear pieces from tools/models/human.py; the hunter and the druid still wear the
// pack's female peasant and ranger outfits, told apart by tints and hair. Weapons: tools/models/weapons.py.
var heroLooks = map[character.Class]heroLook{
	character.Warrior: {outfit: r3d.Outfit{Body: humans + "Male_Body", Head: humans + "Knight_Helm", Chest: humans + "Knight_Cuirass",
		Arms: humans + "Knight_Arms", Legs: humans + "Knight_Legs", Feet: humans + "Knight_Sabatons", Shoulders: humans + "Knight_Mantle",
		ShowBody: true}, weapon: "Sword_Steel", shield: "Shield_Heater"}, // the knight: art/concept/knight_s51_*.png
	character.Mage: {outfit: r3d.Outfit{Body: humans + "Male_Body_Robed", Head: humans + "Mage_Hood", Chest: humans + "Mage_Robe",
		Arms: humans + "Mage_Sleeves", Feet: humans + "Shoes", Shoulders: humans + "Mage_Cloak", ShowBody: true}, // art/concept/mage_s51.png
		weapon: "Staff_Mage", cast: staffCast},
	character.Rogue: {outfit: r3d.Outfit{Body: humans + "Male_Body", Head: humans + "Rogue_Hood", Chest: humans + "Rogue_Jerkin",
		Arms: humans + "Rogue_Arms", Legs: humans + "Rogue_Legs", Feet: humans + "Rogue_Boots", Shoulders: humans + "Rogue_Cape", ShowBody: true}, // art/concept/rogue_s51.png
		weapon: "Dagger"},
	character.Hunter: {outfit: r3d.Outfit{Body: "Superhero_Female_FullBody", Hair: femaleHead, HairTint: color.RGBA{150, 90, 50, 255},
		Chest: "Female_Ranger_Body", Arms: "Female_Ranger_Arms", Legs: "Female_Ranger_Legs", Feet: "Female_Ranger_Feet",
		Recolor: 0.8, Tints: map[string]color.RGBA{"Female_Ranger_Body": {130, 160, 100, 255}, "Female_Ranger_Legs": {140, 120, 90, 255}}},
		bow: true, ready: "Pistol_Idle_Loop"},
	character.Warlock: {outfit: r3d.Outfit{Body: humans + "Male_Body_Robed", Head: humans + "Warlock_Hood", Chest: humans + "Warlock_Robe",
		Arms: humans + "Warlock_Sleeves", Feet: humans + "Shoes", Shoulders: humans + "Warlock_Cloak", ShowBody: true}, // art/concept/warlock_s51.png
		weapon: "Staff_Warlock", cast: staffCast},
	character.Priest: {outfit: r3d.Outfit{Body: humans + "Male_Body_Robed", Hair: []string{humans + "Hair_Short"}, Chest: humans + "Priest_Robe",
		Arms: humans + "Priest_Sleeves", Feet: humans + "Shoes", Shoulders: humans + "Priest_Stole", ShowBody: true},
		weapon: "Staff_Priest", cast: staffCast},
	character.Paladin: {outfit: r3d.Outfit{Body: humans + "Male_Body", Head: humans + "Paladin_Helm",
		Chest: humans + "Paladin_Cuirass", Arms: humans + "Paladin_Arms", Legs: humans + "Paladin_Legs", Feet: humans + "Paladin_Sabatons",
		Shoulders: humans + "Paladin_Mantle", ShowBody: true},
		weapon: "Hammer", shield: "Shield_Paladin"},
	character.Shaman: {outfit: r3d.Outfit{Body: humans + "Male_Body", Hair: []string{humans + "Hair_Long", humans + "Beard_Long", humans + "Hair_Feathers"},
		Chest: humans + "Shaman_Tunic", Arms: humans + "Shaman_Arms", Legs: humans + "Shaman_Legs", Feet: humans + "Boots_Fur",
		Shoulders: humans + "Shaman_Mantle", ShowBody: true}, // art/concept/shaman_s51.png
		weapon: "Staff_Shaman", cast: staffCast},
	character.Druid: {outfit: r3d.Outfit{Body: "Superhero_Female_FullBody", Hair: femaleHead, HairTint: color.RGBA{110, 70, 40, 255},
		Chest: "Female_Peasant_Body", Arms: "Female_Peasant_Arms", Legs: "Female_Peasant_Legs", Feet: "Female_Peasant_Feet",
		Recolor: 0.8, Tints: map[string]color.RGBA{"Female_Peasant_Body": {110, 150, 80, 255}, "Female_Peasant_Legs": {120, 95, 65, 255}}},
		weapon: "Staff_Druid", cast: staffCast},
}

// Where held items sit at the hand bones: rotation in degrees, offset in meters. Overridable
// with OLDWORLD_SWORD / OLDWORLD_SHIELD / OLDWORLD_BOW / OLDWORLD_STAFF / OLDWORLD_STAFF_FIGHT ("rx,ry,rz,x,y,z") for tuning.
var (
	swordAt      = [2]rl.Vector3{{X: -75}, {Y: 0.05}}
	shieldAt     = [2]rl.Vector3{{Y: 90}, {X: -0.11, Y: 0.14}}    // outside the forearm, far enough out that the fist stays behind it
	bowAt        = [2]rl.Vector3{{X: -80, Z: -90}, {Y: 0.05}}     // the string toward the archer
	staffAt      = [2]rl.Vector3{{X: -135}, {Y: 0.05}}            // at the side, the top tilted forward
	staffFightAt = [2]rl.Vector3{{X: -165}, {Y: 0.034, Z: 0.058}} // in the sword stance's fist, the top a little forward
)

func init() {
	for env, v := range map[string]*[2]rl.Vector3{"OLDWORLD_SWORD": &swordAt, "OLDWORLD_SHIELD": &shieldAt, "OLDWORLD_BOW": &bowAt, "OLDWORLD_STAFF": &staffAt, "OLDWORLD_STAFF_FIGHT": &staffFightAt} {
		if s := os.Getenv(env); s != "" {
			fmt.Sscanf(s, "%f,%f,%f,%f,%f,%f", &v[0].X, &v[0].Y, &v[0].Z, &v[1].X, &v[1].Y, &v[1].Z)
		}
	}
}

// fighter is a creature kind ready to draw: a shared model, or a shared humanoid.
type fighter struct {
	model *r3d.Creature
	human *r3d.Character
	hover float32
	moves r3d.Moves
	anims *r3d.Anims
	size  float32 // height in meters
}

type npcView struct {
	play *r3d.Player
	yaw  float32
	seen int // frame last drawn
}

type scene struct {
	g          *Game
	r          *r3d.Renderer
	lib        *r3d.Library
	ground     *r3d.Ground
	human      *r3d.Anims
	hero       *r3d.Character
	heroOf     character.Class
	heroYaw    float32
	heroMove   int  // the hero's animation for the ability being shown in battle
	heroActing bool // the hero used it and hasn't struck yet
	fighters   map[creature.Kind]*fighter
	npcs       map[uint64]*npcView
	buildings  map[tiles.ID][]r3d.Object
	surfaces   map[string]rl.Texture2D
	maskAt     [2]int
	maskSet    bool
	maskOf     *world.World // the world the ground mask was made for
	view       *world.World // the world being drawn: the game's, or the title's backdrop
	backdrop   *world.World
	backdropAt [2]int // its start, which the title camera drifts around
	cam        rl.Camera3D
	pebble     rl.Model
	seat       rl.Model // the log the hero sits on at camp
	firewood   rl.Model // one log of the camp fire, leaning in
	flame      rl.Model // a unit cone, drawn unlit
	embers     rl.Model

	previews   map[character.Class]*r3d.Character // the new-game screen's class previews
	previewTex rl.RenderTexture2D

	battleOf  *battleScene
	enemyPlay *r3d.Player
	seen      int     // battle message serial already animated
	totem     float32 // seconds since the hero put down their totems, -1 before
	bg        map[procgen.ZoneType]*rl.Texture2D
}

// scene3d returns the 3D scene, set up on first use.
func (g *Game) scene3d() *scene {
	if g.scene != nil {
		return g.scene
	}
	sc := &scene{
		g:         g,
		r:         r3d.NewRenderer(),
		lib:       r3d.NewLibrary(filepath.Join(g.opts.AssetsDir, "quaternius")),
		fighters:  map[creature.Kind]*fighter{},
		npcs:      map[uint64]*npcView{},
		buildings: map[tiles.ID][]r3d.Object{},
		surfaces:  map[string]rl.Texture2D{},
		bg:        map[procgen.ZoneType]*rl.Texture2D{},
	}
	tex := func(name string) string { return filepath.Join(g.opts.AssetsDir, "generated/map", name+".png") }
	sc.ground = r3d.NewGround([7]string{tex("tex_grass"), tex("tex_dirt"), tex("tex_water"), tex("tex_sand"), tex("tex_sandpath"),
		tex("tex_snow"), tex("tex_ice")}, maskTiles)
	sc.human = sc.lib.Anims("UAL1_Standard.glb")
	sc.pebble = sc.lib.Model(natureDir + "Pebble.gltf")
	// A log lying across, just behind the hero, its top at sitting height.
	sc.seat = rl.LoadModelFromMesh(rl.GenMeshCylinder(0.25, 0.75, 12))
	sc.seat.Transform = rl.MatrixMultiply(rl.MatrixMultiply(rl.MatrixTranslate(0, -0.375, 0), rl.MatrixRotateZ(math.Pi/2)),
		rl.MatrixTranslate(0, 0.25, -0.12))
	rl.SetMaterialTexture(sc.seat.Materials, rl.MapAlbedo, sc.surfaceTex(timber.tex))
	// Camp fire logs stand 0.3 out and lean in steeply, so their tops meet low over the middle
	// and the flames rise above them.
	sc.firewood = rl.LoadModelFromMesh(rl.GenMeshCylinder(0.045, 0.4, 7))
	sc.firewood.Transform = rl.MatrixMultiply(rl.MatrixRotateX(55*math.Pi/180), rl.MatrixTranslate(0, 0, 0.3))
	rl.SetMaterialTexture(sc.firewood.Materials, rl.MapAlbedo, sc.surfaceTex(timber.tex))
	sc.flame = rl.LoadModelFromMesh(rl.GenMeshCone(1, 1, 6))
	sc.embers = rl.LoadModelFromMesh(rl.GenMeshCylinder(0.15, 0.03, 10))
	sc.cam = rl.Camera3D{Up: rl.NewVector3(0, 1, 0), Fovy: 34, Projection: rl.CameraPerspective}
	g.scene = sc
	return sc
}

// heroModel returns the player's character, rebuilt when the class changes.
func (sc *scene) heroModel() *r3d.Character {
	class := sc.g.char.Class
	if sc.hero != nil && sc.heroOf == class {
		return sc.hero
	}
	sc.hero, sc.heroOf = sc.buildHero(class), class
	return sc.hero
}

// buildHero makes a class's character with what it holds.
func (sc *scene) buildHero(class character.Class) *r3d.Character {
	l := heroLooks[class]
	hero := sc.lib.NewCharacter(l.outfit, sc.human)
	if i := sc.human.Find(l.ready); l.ready != "" && i >= 0 {
		hero.Moves.Ready = i
	}
	if l.cast != "" {
		// The thrust of the punch only: its guard at the start and end turns the fist over.
		if i := sc.human.Trim(l.cast, "Staff_Cast", castFrom, castTo); i >= 0 {
			hero.Moves.Cast = i
		}
	}
	sc.human.SetSpeed(hero.Moves.Cast, castSpeed)
	if l.weapon != "" {
		staff := strings.HasPrefix(l.weapon, "Staff")
		at := swordAt
		if staff {
			at = staffAt // upright, not pointing forward
		}
		sc.lib.Hold(hero, "../generated/props/"+l.weapon+".gltf", "hand_r", at[0], at[1])
		if staff {
			hero.HoldFight(staffFightAt[0], staffFightAt[1])
		}
	}
	if l.shield != "" {
		sc.lib.Hold(hero, "../generated/props/"+l.shield+".gltf", "lowerarm_l", shieldAt[0], shieldAt[1])
	}
	if l.bow {
		sc.lib.Hold(hero, "../generated/props/Bow.gltf", "hand_l", bowAt[0], bowAt[1])
	}
	return hero
}

// drawPreview draws a class's hero standing idle into a rectangle of the screen (pixels), on
// top of what's there: the new-game screen's class preview.
func (sc *scene) drawPreview(class character.Class, x, y, w, h int32) {
	if sc.previews == nil {
		sc.previews = map[character.Class]*r3d.Character{}
	}
	hero := sc.previews[class]
	if hero == nil {
		hero = sc.buildHero(class)
		sc.previews[class] = hero
	}
	if sc.previewTex.ID == 0 || sc.previewTex.Texture.Width != w || sc.previewTex.Texture.Height != h {
		if sc.previewTex.ID != 0 {
			rl.UnloadRenderTexture(sc.previewTex)
		}
		sc.previewTex = rl.LoadRenderTexture(w, h)
	}
	hero.Play.Loop(hero.Moves.Idle)
	hero.Play.Update(frameDT)
	rl.BeginTextureMode(sc.previewTex)
	rl.ClearBackground(color.RGBA{})
	cam := rl.Camera3D{Position: rl.NewVector3(0.9, 1.5, 4.6), Target: rl.NewVector3(0, 0.92, 0), Up: rl.NewVector3(0, 1, 0),
		Fovy: 30, Projection: rl.CameraPerspective}
	sc.r.SetView(rl.Vector3Subtract(cam.Position, cam.Target))
	rl.BeginMode3D(cam)
	r3d.Shadow(rl.Vector3{}, 0.45)
	sc.r.DrawCharacter(hero, rl.Vector3{}, 20)
	rl.EndMode3D()
	rl.EndTextureMode()
	// Render textures are upside down.
	rl.DrawTextureRec(sc.previewTex.Texture, rl.NewRectangle(0, 0, float32(w), -float32(h)), rl.NewVector2(float32(x), float32(y)), rl.White)
}

// heroMove is the hero's animation for an ability: a swing for melee, a shot for ranged
// attacks, a cast for spells and self effects.
func heroMove(m r3d.Moves, a *battle.Ability) int {
	switch {
	case a.Delivery == battle.Melee:
		return m.Attack
	case a.Delivery == battle.Ranged || a.Category == battle.PhysicalCat:
		return m.Shoot
	}
	return m.Cast
}

// fighter returns the look of a creature kind, loaded on first use.
func (sc *scene) fighter(k creature.Kind) *fighter {
	if f, ok := sc.fighters[k]; ok {
		return f
	}
	l := looks[k]
	f := &fighter{hover: l.hover}
	if l.outfit != nil {
		f.human = sc.lib.NewCharacter(*l.outfit, sc.human)
		if l.weapon != "" {
			sc.lib.Hold(f.human, l.weapon, "hand_r", swordAt[0], swordAt[1])
		}
		if l.shield != "" {
			sc.lib.Hold(f.human, l.shield, "lowerarm_l", shieldAt[0], shieldAt[1])
		}
		f.moves, f.anims, f.size = f.human.Moves, sc.human, 1.85
	} else {
		m := sc.lib.Model(l.model)
		anims := sc.lib.Anims(l.model)
		scale := float32(1)
		if h := r3d.Height(m); h > 0 {
			scale = l.height / h
		}
		f.model = &r3d.Creature{Model: m, Anims: anims, Moves: r3d.MovesFor(anims), Scale: scale, Tint: l.tint, NoInk: sc.lib.NoInk(l.model)}
		if l.fur != "" {
			tex := r3d.Detail(filepath.Join(sc.g.opts.AssetsDir, "generated/creatures", l.fur+".png"))
			for i := range m.GetMaterials() {
				rl.SetMaterialTexture(&m.GetMaterials()[i], rl.MapAlbedo, tex)
			}
			f.model.Detail = l.furPM * scale
		}
		f.moves, f.anims, f.size = f.model.Moves, anims, l.height
	}
	sc.fighters[k] = f
	return f
}

// drawFighter draws a creature with its own animation state.
func (sc *scene) drawFighter(f *fighter, p *r3d.Player, pos rl.Vector3, yaw float32) {
	pos.Y += f.hover
	if f.human != nil {
		f.human.Play = p
		sc.r.DrawCharacter(f.human, pos, yaw)
		return
	}
	sc.r.DrawCreature(f.model, p, pos, yaw)
}

// facingYaw is the model rotation for a facing; models face +z (south) at 0°.
func facingYaw(d world.Dir) float32 {
	switch d {
	case world.Up:
		return 180
	case world.Left:
		return -90
	case world.Right:
		return 90
	}
	return 0
}

// turn moves an angle toward a target along the shorter way.
func turn(from, to, step float32) float32 {
	d := float32(math.Mod(float64(to-from)+540, 360) - 180)
	if d > step {
		d = step
	} else if d < -step {
		d = -step
	}
	return from + d
}

// worldPos converts a position in map pixels (tile × 16) to the 3D position of the tile's
// center on the ground.
func worldPos(px, py float64) rl.Vector3 {
	return rl.NewVector3(float32(px/tiles.Size)+0.5, 0, float32(py/tiles.Size)+0.5)
}

// mapCamera places the camera above and south of the player, looking down at about 55°.
func (sc *scene) mapCamera(p rl.Vector3) {
	sc.cam.Target = rl.NewVector3(p.X, 0.8, p.Z-0.8)
	sc.cam.Position = rl.NewVector3(p.X, 12.5, p.Z+8.2)
	sc.cam.Fovy = 34
	sc.r.SetView(rl.Vector3Subtract(sc.cam.Position, sc.cam.Target))
}

// groundKind classifies a ground tile for the ground mask.
func groundKind(id tiles.ID) r3d.Kind {
	switch {
	case tiles.IsIce(id):
		return r3d.Ice
	case tiles.IsWater(id):
		return r3d.Water
	case id >= tiles.Dirt && id <= tiles.DirtInnerSW:
		return r3d.Dirt
	case id >= tiles.SandPath && id <= tiles.SandPathInnerSW:
		return r3d.Path
	case tiles.IsSand(id):
		return r3d.Sand
	}
	return r3d.Plain
}

// groundBase is the ground a zone type is covered in, under its roads and lakes.
func groundBase(z procgen.ZoneType) r3d.Base {
	switch z {
	case procgen.North:
		return r3d.SnowBase
	case procgen.Mediterranean:
		return r3d.DryBase
	case procgen.Unholy:
		return r3d.DeadBase
	}
	return r3d.GrassBase
}

func (sc *scene) groundAt(tx, ty int) tiles.ID {
	cx, cy := world.ChunkCoord(tx, ty)
	return sc.view.Chunk(cx, cy).Ground[procgen.Index(tx-cx*procgen.ChunkSize, ty-cy*procgen.ChunkSize)]
}

// drawMap draws the adventure map around the player.
func (sc *scene) drawMap(g *Game) {
	if sc.battleOf != nil { // back from a battle
		sc.battleOf = nil
		sc.heroModel().Play.Reset(sc.hero.Moves.Idle)
	}
	sc.view = g.world
	sc.mapCamera(worldPos(g.player.PixelPos()))
	ptx, pty := g.player.TX, g.player.TY
	rl.BeginMode3D(sc.cam)
	sc.drawScenery(ptx, pty)
	x0, x1, y0, y1 := ptx-viewX, ptx+viewX, pty-viewUp, pty+viewDown

	// The player.
	hero := sc.heroModel()
	p := g.player
	dt := frameDT
	switch {
	case g.camp != nil:
		hero.Play.Loop(hero.Moves.Rest)
	case p.Running():
		hero.Play.Loop(hero.Moves.Run)
		dt *= runPace * world.RunFrames / float32(p.StepFrames()) // faster on roads
	case p.Moving():
		hero.Play.Loop(hero.Moves.Walk)
		dt *= walkPace * world.WalkFrames / float32(p.StepFrames())
	default:
		hero.Play.Loop(hero.Moves.Idle)
	}
	hero.Play.Update(dt)
	sc.heroYaw = turn(sc.heroYaw, facingYaw(p.Facing), 18)
	hp := worldPos(p.PixelPos())
	r3d.Shadow(hp, 0.4)
	hero.Stow, hero.Fight = g.camp != nil, false
	if g.camp != nil {
		sc.r.Draw(r3d.Object{Model: sc.seat, Pos: hp, Yaw: sc.heroYaw, Tint: timber.tint, Outline: 0.012, Detail: timber.scale})
	}
	sc.r.DrawCharacter(hero, hp, sc.heroYaw)

	// Enemies.
	for _, n := range g.world.NPCs() {
		if n.TX < x0-1 || n.TX > x1+1 || n.TY < y0-1 || n.TY > y1+1 {
			continue
		}
		f := sc.fighter(n.Spawn.Kind)
		v := sc.npcs[n.Spawn.ID]
		if v == nil {
			v = &npcView{play: r3d.NewPlayer(f.anims, f.moves.Idle, float32(n.Spawn.ID%97)/30), yaw: facingYaw(n.Facing)}
			sc.npcs[n.Spawn.ID] = v
		}
		dt := frameDT
		if n.Moving() {
			v.play.Loop(f.moves.Walk)
			dt *= walkPace
		} else {
			v.play.Loop(f.moves.Idle)
		}
		v.play.Update(dt)
		v.yaw = turn(v.yaw, facingYaw(n.Facing), 15)
		v.seen = g.frame
		np := worldPos(n.PixelPos())
		r3d.Shadow(np, 0.35+f.size*0.1)
		sc.drawFighter(f, v.play, np, v.yaw)
	}
	if g.frame%600 == 0 {
		for id, v := range sc.npcs {
			if g.frame-v.seen > 600 {
				delete(sc.npcs, id)
			}
		}
	}

	if g.camp != nil {
		sc.drawCamp(g)() // its flames go last, over the opaque scene
	}
	rl.EndMode3D()
}

// drawScenery draws the ground and the objects of sc.view around a tile, inside BeginMode3D.
func (sc *scene) drawScenery(ptx, pty int) {
	if at := [2]int{ptx - maskTiles/2, pty - maskTiles/2 - 6}; !sc.maskSet || at != sc.maskAt || sc.maskOf != sc.view {
		seed := sc.view.Seed
		sc.ground.SetMask(at[0], at[1], func(x, z int) (r3d.Kind, r3d.Base) {
			return groundKind(sc.groundAt(x, z)), groundBase(procgen.ZoneTypeAt(seed, x, z))
		})
		sc.maskAt, sc.maskSet, sc.maskOf = at, true, sc.view
	}
	sc.ground.Draw()
	x0, x1, y0, y1 := ptx-viewX, ptx+viewX, pty-viewUp, pty+viewDown
	cx0, cy0 := world.ChunkCoord(x0-6, y0-6)
	cx1, cy1 := world.ChunkCoord(x1+1, y1+1)
	for cy := cy0; cy <= cy1; cy++ {
		for cx := cx0; cx <= cx1; cx++ {
			for _, o := range sc.view.Chunk(cx, cy).Objects {
				s := tiles.Get(o.Sprite)
				if o.X+s.W < x0 || o.X > x1 || o.Y+s.H < y0 || o.Y > y1 {
					continue
				}
				sc.drawObject(o)
			}
		}
	}
	sc.drawTufts(x0, x1, y0, y1)
}

// drawBackdrop draws the title screen's background: the meadows around the start of a fixed
// world, the camera drifting slowly east and back.
func (sc *scene) drawBackdrop() {
	if sc.backdrop == nil {
		sc.backdrop = world.New(backdropSeed)
		sc.backdropAt = scenicSpot(sc.backdrop)
	}
	sc.view = sc.backdrop
	t := float64(sc.g.frame) / 60
	x := float32(sc.backdropAt[0]) + 0.5 + 8*float32(math.Sin(t/40))
	z := float32(sc.backdropAt[1]) + 0.5
	sc.mapCamera(rl.NewVector3(x, 0, z))
	rl.BeginMode3D(sc.cam)
	sc.drawScenery(int(x), int(z))
	rl.EndMode3D()
}

// backdropSeed is the world shown behind the title screen.
const backdropSeed = 1

// scenicSpot looks around a world's start for the prettiest view for the title screen: many
// trees at a forest's edge, some water, no roads and houses (the start is always a village).
func scenicSpot(w *world.World) [2]int {
	sx, sy := w.FindSpawn()
	best, bestScore := [2]int{sx, sy}, -1
	for dy := -60; dy <= 60; dy += 6 {
		for dx := -60; dx <= 60; dx += 6 {
			x, y := sx+dx, sy+dy
			trees, water, built := 0, 0, 0
			for ty := y - viewUp; ty <= y+viewDown; ty += 2 {
				for tx := x - viewX; tx <= x+viewX; tx += 2 {
					cx, cy := world.ChunkCoord(tx, ty)
					id := w.Chunk(cx, cy).Ground[procgen.Index(tx-cx*procgen.ChunkSize, ty-cy*procgen.ChunkSize)]
					switch {
					case tiles.IsWater(id):
						water++
					case id >= tiles.Dirt && id <= tiles.DirtInnerSW:
						built++
					}
				}
			}
			cx0, cy0 := world.ChunkCoord(x-viewX, y-viewUp)
			cx1, cy1 := world.ChunkCoord(x+viewX, y+viewDown)
			for cy := cy0; cy <= cy1; cy++ {
				for cx := cx0; cx <= cx1; cx++ {
					for _, o := range w.Chunk(cx, cy).Objects {
						if o.X < x-viewX || o.X > x+viewX || o.Y < y-viewUp || o.Y > y+viewDown {
							continue
						}
						switch {
						case isTreeID(o.Sprite):
							trees++
						case tiles.IsBuilding(o.Sprite):
							built += 20
						}
					}
				}
			}
			// A forest edge by water: enough trees to frame it, not a solid canopy.
			score := min(trees, 45)*3 - max(0, trees-65)*3 + min(water, 50)*2 - built*4
			if water > 110 { // mostly lake: not much to look at
				score -= 200
			}
			if score > bestScore {
				best, bestScore = [2]int{x, y}, score
			}
		}
	}
	return best
}

// scenery is a model for a map object: one of several files (picked by position) at a scale.
// The models come from tools/models/nature.py, made at their size on the map.
type scenery struct {
	files []string
	scale float32
	fixed bool // monuments: always facing south at their size, not turned and scaled at random
}

const natureDir = "../generated/nature/"

var sceneryModels = map[tiles.ID]scenery{
	tiles.TreeRound:     {files: []string{"Tree_1", "Tree_2", "Tree_3", "Tree_4"}, scale: 1},
	tiles.TreeRoots:     {files: []string{"Tree_Old_1", "Tree_Old_2"}, scale: 1},
	tiles.TreePine:      {files: []string{"Pine_1", "Pine_2", "Pine_3", "Pine_4"}, scale: 1},
	tiles.Bush:          {files: []string{"Bush", "Bush_Flowers"}, scale: 1}, // blocks its tile, so it fills it
	tiles.PlantA:        {files: []string{"Plant", "Fern"}, scale: 1},
	tiles.PlantB:        {files: []string{"Plant_Low", "Fern"}, scale: 1},
	tiles.PlantC:        {files: []string{"Plant_Big"}, scale: 1},
	tiles.FlowerSun:     {files: []string{"Flowers_Yellow"}, scale: 1},
	tiles.FlowerOrange:  {files: []string{"Flowers_Orange"}, scale: 1},
	tiles.FlowerRed:     {files: []string{"Flowers_Red"}, scale: 1},
	tiles.FlowerWhite:   {files: []string{"Flowers_White"}, scale: 1},
	tiles.Clover:        {files: []string{"Clover"}, scale: 1},
	tiles.Palm:          {files: []string{"Palm_1", "Palm_2", "Palm_3"}, scale: 1},
	tiles.PalmB:         {files: []string{"Palm_Pair_1", "Palm_Pair_2"}, scale: 1},
	tiles.Boulder:       {files: []string{"Rock_1", "Rock_2"}, scale: 1},
	tiles.BoulderB:      {files: []string{"Rock_3"}, scale: 1},
	tiles.RockStump:     {files: []string{"Rock_2"}, scale: 0.82},
	tiles.SmallRock:     {files: []string{"Rock_1", "Rock_2", "Rock_3"}, scale: 0.55}, // blocks its tile, so it fills it
	tiles.Skull:         {files: []string{"Skull_Horned"}, scale: 1},
	tiles.SkullB:        {files: []string{"Skull"}, scale: 1},
	tiles.Ribs:          {files: []string{"Ribs"}, scale: 1},
	tiles.Bone:          {files: []string{"Bone"}, scale: 1},
	tiles.SproutOrange:  {files: []string{"Shrub_Dry"}, scale: 1},
	tiles.SproutOlive:   {files: []string{"Agave"}, scale: 1},
	tiles.Sphinx:        {files: []string{"Sphinx"}, scale: 1.25, fixed: true},
	tiles.PharaohStatue: {files: []string{"Pharaoh_Statue"}, scale: 1, fixed: true},
	tiles.SnowPine:      {files: []string{"Snow_Pine_1", "Snow_Pine_2", "Snow_Pine_3"}, scale: 1},
	tiles.SnowPineB:     {files: []string{"Snow_Pine_Small_1", "Snow_Pine_Small_2"}, scale: 1},
	tiles.SnowRock:      {files: []string{"Snow_Rock_1", "Snow_Rock_2"}, scale: 1},
	tiles.SnowBush:      {files: []string{"Snow_Bush"}, scale: 1}, // blocks its tile, so it fills it
	tiles.FrostFern:     {files: []string{"Frost_Fern"}, scale: 1},
	tiles.DeadTwig:      {files: []string{"Dead_Twigs"}, scale: 1},
	tiles.IceShard:      {files: []string{"Ice_Shards"}, scale: 1},
	tiles.OliveTree:     {files: []string{"Olive_Tree_1", "Olive_Tree_2", "Olive_Tree_3"}, scale: 1},
	tiles.Cypress:       {files: []string{"Cypress_1", "Cypress_2"}, scale: 1},
	tiles.LimeRock:      {files: []string{"Lime_Rock_1", "Lime_Rock_2"}, scale: 1},
	tiles.Lavender:      {files: []string{"Lavender"}, scale: 1}, // a small obstacle (the biome's bush) and decor
	tiles.Sage:          {files: []string{"Sage"}, scale: 1},
	tiles.Poppies:       {files: []string{"Poppies"}, scale: 1},
	tiles.BrokenColumn:  {files: []string{"Broken_Column"}, scale: 1},
	tiles.Amphora:       {files: []string{"Amphora"}, scale: 1},
	tiles.DeadTree:      {files: []string{"Dead_Tree_1", "Dead_Tree_2", "Dead_Tree_3"}, scale: 1},
	tiles.DarkPine:      {files: []string{"Dark_Pine_1", "Dark_Pine_2"}, scale: 1},
	tiles.DarkRock:      {files: []string{"Dark_Rock_1", "Dark_Rock_2"}, scale: 1},
	tiles.ThornBush:     {files: []string{"Thorn_Bush"}, scale: 1},
	tiles.Gravestone:    {files: []string{"Gravestone"}, scale: 1},
	tiles.GravestoneB:   {files: []string{"Gravestone_Cross"}, scale: 1},
	tiles.Mushrooms:     {files: []string{"Mushrooms"}, scale: 1},
}

// spot returns a stable pseudo-random number for a tile.
func spot(x, y int) uint32 {
	h := uint32(x)*73856093 ^ uint32(y)*19349663
	h ^= h >> 13
	return h * 0x5bd1e995
}

// drawObject draws a map object: a scenery model or a building.
func (sc *scene) drawObject(o procgen.Object) {
	s := tiles.Get(o.Sprite)
	if m, ok := sceneryModels[o.Sprite]; ok {
		h := spot(o.X, o.Y)
		file := natureDir + m.files[h%uint32(len(m.files))] + ".gltf"
		// Stand on the middle of the bottom row of the footprint.
		pos := rl.NewVector3(float32(o.X)+float32(s.W)/2, 0, float32(o.Y+s.H)-0.5)
		scale, yaw := m.scale*(0.85+float32(h>>8%30)/100), float32(h>>4%360)
		if m.fixed {
			scale, yaw = m.scale, 0
		}
		if isTreeID(o.Sprite) {
			r3d.ShadowAlpha(pos, 0.9*scale, 40)
		}
		sc.r.Draw(r3d.Object{Model: sc.lib.Model(file), Pos: pos, Yaw: yaw, Scale: scale,
			Outline: 0.02, NoInk: sc.lib.NoInk(file), Fade: sc.fade(o)})
		return
	}
	if parts := sc.building(o.Sprite); parts != nil {
		fade := sc.fade(o)
		for _, p := range parts {
			p.Pos = rl.Vector3Add(p.Pos, rl.NewVector3(float32(o.X), 0, float32(o.Y)))
			p.Fade = fade
			sc.r.Draw(p)
		}
		return
	}
	// Every object has a model (TestEveryObjectHasAModel); a new one without shows nothing.
}

// fade returns how see-through an object is: tall objects in front of the player (south of
// them, so closer to the camera) that would hide them.
func (sc *scene) fade(o procgen.Object) float32 {
	if !fades(o.Sprite) {
		return 0
	}
	s := tiles.Get(o.Sprite)
	if sc.g.player == nil || sc.view != sc.g.world { // the title's backdrop: nothing to see through to
		return 0
	}
	px, py := sc.g.player.PixelPos()
	x, y := float32(px/tiles.Size)+0.5, float32(py/tiles.Size)+0.5
	reach := float32(2.2) // how far tall objects stick up toward the camera, in tiles
	if isTreeID(o.Sprite) {
		reach = 4
	}
	left, right := float32(o.X)-1, float32(o.X+s.W)+1
	front := float32(o.Y + s.H)
	if x < left || x > right || y > front || y < front-reach-float32(s.H) {
		return 0
	}
	return 0.5
}

func isTreeID(id tiles.ID) bool {
	switch id {
	case tiles.TreeRound, tiles.TreeRoots, tiles.TreePine, tiles.Palm, tiles.PalmB, tiles.SnowPine, tiles.SnowPineB, tiles.OliveTree,
		tiles.Cypress, tiles.DeadTree, tiles.DarkPine:
		return true
	}
	return false
}

// fades reports whether an object is tall enough to hide the player behind it, so it turns
// see-through when they walk behind it: trees, buildings and monuments, not stones, bushes
// or plants.
func fades(id tiles.ID) bool {
	return isTreeID(id) || tiles.IsBuilding(id)
}

// drawTufts puts grass tufts on the grass tiles that have them in the tile map.
func (sc *scene) drawTufts(x0, x1, y0, y1 int) {
	grass := sc.lib.Model(natureDir + "Grass.gltf")
	ink := sc.lib.NoInk(natureDir + "Grass.gltf")
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			id := sc.groundAt(x, y)
			h := spot(x, y)
			if id < tiles.GrassTuftA || id > tiles.GrassTuftE || h%3 != 0 {
				continue
			}
			pos := rl.NewVector3(float32(x)+0.2+float32(h%60)/100, 0, float32(y)+0.2+float32(h>>8%60)/100)
			sc.r.Draw(r3d.Object{Model: grass, Pos: pos, Yaw: float32(h % 360), Scale: 1, NoInk: ink})
		}
	}
}

// drawCamp draws the fire pit: a ring of stones and leaning logs. The flames are drawn
// afterwards: low-poly cones without shading (a fire is its own light), a solid yellow core
// with see-through orange tongues around it, each flickering on its own rhythm.
func (sc *scene) drawCamp(g *Game) func() {
	c := g.camp
	center := rl.NewVector3(float32(c.x)+0.5, 0, float32(c.y)+0.5)
	for i := 0; i < 7; i++ {
		a := float64(i) * 2 * math.Pi / 7
		pos := rl.NewVector3(center.X+0.35*float32(math.Cos(a)), 0, center.Z+0.35*float32(math.Sin(a)))
		sc.r.Draw(r3d.Object{Model: sc.pebble, Pos: pos, Yaw: float32(i * 50), Scale: 0.3, Outline: 0.01})
	}
	for i := 0; i < 4; i++ {
		sc.r.Draw(r3d.Object{Model: sc.firewood, Pos: center, Yaw: float32(i*90 + 20), Tint: color.RGBA{95, 65, 45, 255}, Outline: 0.008, Detail: 2})
	}
	return func() {
		t := float64(g.frame) / 60
		wave := func(speed, phase float64) float32 {
			return float32(0.5 + 0.3*math.Sin(t*speed+phase) + 0.2*math.Sin(t*speed*2.3+phase*1.7))
		}
		rl.DrawModel(sc.embers, center, 1, color.RGBA{200, 70, 25, 255})
		tongue := func(x, z, r, h float32, col color.RGBA, i float64) {
			sway := (wave(3, i*2.1) - 0.5) * 0.06
			rl.DrawModelEx(sc.flame, rl.NewVector3(center.X+x+sway, 0.02, center.Z+z), rl.NewVector3(0, 1, 0), float32(i*40),
				rl.NewVector3(r, h*(0.75+0.5*wave(7, i*1.3)), r), col)
		}
		tongue(0, 0, 0.08, 0.45, color.RGBA{255, 225, 110, 255}, 0)
		for i := 0; i < 5; i++ {
			a := float64(i)*2*math.Pi/5 + 0.4
			tongue(0.07*float32(math.Cos(a)), 0.07*float32(math.Sin(a)), 0.075, 0.5+0.12*float32(i%2), color.RGBA{250, 120, 35, 200}, float64(i+1))
		}
	}
}

// drawMapOverlay draws markers over the 3D map onto the canvas: the "!" over enemies that
// just spotted the player. scale is screen pixels per canvas pixel.
func (g *Game) drawMapOverlay(dst *gfx.Image, scale float64) {
	sc := g.scene3d()
	sc.mapCamera(worldPos(g.player.PixelPos()))
	for _, n := range g.world.NPCs() {
		if !n.JustSpotted() {
			continue
		}
		f := sc.fighter(n.Spawn.Kind)
		pos := worldPos(n.PixelPos())
		pos.Y = f.size + f.hover + 0.3
		sp := rl.GetWorldToScreen(pos, sc.cam)
		op := &gfx.DrawImageOptions{}
		op.GeoM.Translate(math.Round(float64(sp.X)/scale)-7, math.Round(float64(sp.Y)/scale)-14)
		dst.DrawImage(g.spotted, op)
	}
}

// ---- battle ----

// The shaman's totems (tools/models/totem.py) and where they stand in battle: one on each side
// of the hero.
const (
	totemRise  = 0.5 // seconds a totem takes to spring up out of the ground
	totemScale = 0.7
)

var (
	totemModels = map[battle.Special]string{
		battle.WrathTotem:   "../generated/props/Totem_Wrath.gltf",
		battle.HealingTotem: "../generated/props/Totem_Healing.gltf",
	}
	totemSpots = map[battle.Special]rl.Vector3{
		battle.WrathTotem:   {X: -0.9, Z: -0.8},
		battle.HealingTotem: {X: 0.7, Z: 0.5},
	}
)

// drawTotem draws one of the hero's totems. It springs up: grows from the ground to full size,
// overshooting a little, while turning to face the enemy.
func (sc *scene) drawTotem(model string, pos rl.Vector3, yaw float32) {
	t := min(1, sc.totem/totemRise)
	// Ease out with a little overshoot ("back" easing).
	const c1, c3 = 1.70158, 2.70158
	u := t - 1
	grow := 1 + c3*u*u*u + c1*u*u
	if t > 0.3 {
		r3d.Shadow(pos, 0.25)
	}
	sc.r.Draw(r3d.Object{Model: sc.lib.Model(model), Pos: pos, Yaw: yaw - 120*(1-t),
		Scale: max(0.01, grow*totemScale), Tint: rl.White, Outline: 0.02})
}

// drawBattle draws the fighters of a battle in front of the zone's painted background.
func (sc *scene) drawBattle(s *battleScene) {
	hero := sc.heroModel()
	f := sc.fighter(s.npc.Spawn.Kind)
	if sc.battleOf != s {
		sc.battleOf, sc.seen = s, 0
		sc.enemyPlay = r3d.NewPlayer(f.anims, f.moves.Idle, 0)
		hero.Play.Reset(hero.Moves.Ready)
		sc.heroMove, sc.heroActing = hero.Moves.Attack, false
		hero.Stow = false
		hero.Fight = true
		sc.totem = -1
	}
	// Start the animations for a new message: the attacker strikes, the target flinches,
	// the loser falls.
	if s.serial != sc.seen && len(s.messages) > 0 && s.phase == phasePlay {
		sc.seen = s.serial
		ev := s.messages[0]
		if ev.Cue == battle.CueTotem && ev.User == s.b.Player {
			sc.totem = 0
			hero.Play.Once(hero.Moves.Cast, false)
		}
		if ev.Cue == battle.CueUse && ev.User == s.b.Player && ev.Ability != nil {
			sc.heroMove, sc.heroActing = heroMove(hero.Moves, ev.Ability), true
			if ev.Ability.Delivery == battle.Self || ev.Ability.Category == battle.StatusCat {
				hero.Play.Once(sc.heroMove, false) // buffs, heals and curses show as they're cast
			}
		}
		switch ev.Hit {
		case s.b.Enemy:
			if sc.heroActing { // not for damage over time or a seal's extra damage
				hero.Play.Once(sc.heroMove, false)
				sc.heroActing = false
			}
			sc.enemyPlay.Once(f.moves.Hit, false)
		case s.b.Player:
			sc.enemyPlay.Once(f.moves.Attack, false)
			hero.Play.Once(hero.Moves.Hit, false)
		}
		if ev.EnemyHP <= 0 {
			sc.enemyPlay.Once(f.moves.Death, true)
		}
		if ev.PlayerHP <= 0 {
			hero.Play.Once(hero.Moves.Death, true)
		}
	}
	hero.Play.Loop(hero.Moves.Ready)
	hero.Play.Update(frameDT)
	sc.enemyPlay.Update(frameDT)
	if sc.totem >= 0 {
		sc.totem += frameDT
	}

	// Background.
	sw, sh := float32(rl.GetRenderWidth()), float32(rl.GetRenderHeight())
	if bg := sc.background(s.zone); bg != nil {
		bw, bh := float32(bg.Width), float32(bg.Height)
		k := max(sw/bw, sh/bh)
		rl.DrawTexturePro(*bg, rl.NewRectangle(0, 0, bw, bh), rl.NewRectangle((sw-bw*k)/2, (sh-bh*k)/2, bw*k, bh*k), rl.Vector2{}, 0, rl.White)
	} else {
		pal := battlePalettes[s.zone]
		rl.DrawRectangle(0, 0, int32(sw), int32(sh), pal.ground)
		rl.DrawRectangle(0, 0, int32(sw), int32(sh*0.25), pal.sky) // the horizon where the camera's is
		rl.DrawRectangle(0, int32(sh*0.25), int32(sw), int32(sh*0.008), pal.horizon)
	}

	// The camera stands behind and right of the hero, at about eye height, looking ahead: the
	// hero on the left, the enemy further back on the right. It looks down only a little
	// (battleCameraPitch), so the ground's horizon is a quarter of the way down the screen,
	// where the painted backgrounds put theirs (tools/artgen: horizon at row 180 of 720), and
	// the enemy's feet are on the open ground halfway down.
	heroPos := rl.NewVector3(0, 0, 0)
	enemyPos := rl.NewVector3(3.0, 0, -4.2)
	const eye, ahead = 1.59, 9.6
	drop := float32(ahead * math.Tan(battleCameraPitch*math.Pi/180))
	cam := rl.Camera3D{Position: rl.NewVector3(1.6, eye, 5.6), Target: rl.NewVector3(1.6, eye-drop, 5.6-ahead), Up: rl.NewVector3(0, 1, 0),
		Fovy: battleFovy, Projection: rl.CameraPerspective}
	toward := func(from, to rl.Vector3) float32 {
		return float32(math.Atan2(float64(to.X-from.X), float64(to.Z-from.Z)) * 180 / math.Pi)
	}
	visible := func(c *battle.Combatant) bool { fl := s.flash[c]; return fl == 0 || (fl/4)%2 == 0 }

	sc.r.SetView(rl.Vector3Subtract(cam.Position, cam.Target))
	rl.BeginMode3D(cam)
	if visible(s.b.Player) {
		r3d.Shadow(heroPos, 0.5)
		sc.r.DrawCharacter(hero, heroPos, toward(heroPos, enemyPos)+5)
	}
	if sc.totem >= 0 {
		for _, t := range s.b.Player.Totems() {
			pos := totemSpots[t]
			sc.drawTotem(totemModels[t], pos, toward(pos, enemyPos))
		}
	}
	if visible(s.b.Enemy) && (s.b.Enemy.Alive() || s.phase == phasePlay) {
		r3d.Shadow(enemyPos, 0.3+f.size*0.2)
		sc.drawFighter(f, sc.enemyPlay, enemyPos, toward(enemyPos, heroPos))
	}
	rl.EndMode3D()
}

// background returns the painted battle background of a zone type (nil if there is none).
func (sc *scene) background(z procgen.ZoneType) *rl.Texture2D {
	if t, ok := sc.bg[z]; ok {
		return t
	}
	var t *rl.Texture2D
	if img := sc.g.battleArt(battleBackgroundID(z)); img != nil {
		tex := img.Texture()
		rl.SetTextureFilter(tex, rl.FilterBilinear)
		t = &tex
	}
	sc.bg[z] = t
	return t
}
