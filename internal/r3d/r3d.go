// Package r3d draws the game's 3D scenes with raylib: low-poly models with toon shading
// (light in three flat bands) and ink outlines, animated characters built from parts, and
// textured ground. Models come from the Quaternius packs in assets/quaternius (CC0).
package r3d

import (
	"encoding/json"
	"image/color"
	"math"
	"os"
	"path/filepath"
	"strings"
	"unsafe"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	cullFront = 0 // rlgl RL_CULL_FACE_FRONT
	cullBack  = 1
)

// LightDir is the direction the sun shines in (from the upper left, like the 2D art).
var LightDir = rl.Vector3Normalize(rl.NewVector3(-0.5, -0.8, -0.35))

const toonVS = `#version 330
in vec3 vertexPosition;
in vec2 vertexTexCoord;
in vec3 vertexNormal;
in vec4 vertexColor;
uniform mat4 mvp;
uniform mat4 matModel;
uniform mat4 matNormal;
out vec2 uv;
out vec3 normal;
out vec4 vcol;
out vec3 local;
out vec3 localNormal;
out float height;
void main() {
	height = (matModel * vec4(vertexPosition, 1.0)).y;
	uv = vertexTexCoord;
	local = vertexPosition;
	localNormal = vertexNormal;
	normal = normalize(vec3(matNormal * vec4(vertexNormal, 0.0)));
	vcol = vertexColor;
	gl_Position = mvp * vec4(vertexPosition, 1.0);
}`

// Toon shading: the light is quantized into three bands, so surfaces read as flat comic
// colors. "keep" limits drawing to a sphere in model space (used to show only the head of
// a character's base body under its outfit). With "detail" > 0 the texture is a surface
// texture (see Detail) mapped by position instead of UVs: projected along the axis the
// surface faces, at detail repeats per meter. Metal is marked by a vertex color alpha below
// 1 (tools/models/human.py): it gets a highlight band toward the light, deeper mid tones and
// darker rims, like inked steel.
const toonFS = `#version 330
in vec2 uv;
in vec3 normal;
in vec4 vcol;
in vec3 local;
in vec3 localNormal;
in float height;
uniform sampler2D texture0;
uniform vec4 colDiffuse;
uniform vec3 lightDir;
uniform vec4 keep;
uniform float fade;
uniform float detail;
uniform float recolor;
uniform vec3 viewDir; // toward the camera
out vec4 fragColor;
// fade > 0 makes the model see-through with a dither pattern (needs no sorting).
bool faded() {
	vec2 p = floor(mod(gl_FragCoord.xy, 4.0));
	float d = mod(p.x * 2.0 + p.y * 3.0, 4.0) / 4.0 + mod(p.y, 2.0) * 0.125;
	return fade > 0.0 && d < fade;
}
void main() {
	if (keep.w > 0.0 && distance(local, keep.xyz) > keep.w) discard;
	if (faded()) discard;
	vec4 c;
	if (detail > 0.0) {
		// Box mapping: rows run along z on roofs (y-facing) and sides (x-facing). Sharp
		// weights keep flat faces crisp; curved ones blend. Slopes (domes, pyramids) favor
		// the side projections, so their rows stay level.
		vec3 w = pow(abs(normalize(localNormal)), vec3(8.0)) * vec3(1.0, 0.03, 1.0);
		w /= w.x + w.y + w.z;
		vec3 p = local * detail;
		vec3 t = texture(texture0, vec2(p.z, -p.y)).rgb * w.x + texture(texture0, vec2(p.z, p.x)).rgb * w.y +
			texture(texture0, vec2(p.x, -p.y)).rgb * w.z;
		c = vec4(mix(vec3(1.0), t * 2.0, 0.65), 1.0) * colDiffuse * vec4(vcol.rgb, 1.0); // the texture is normalized to mid grey
		c.rgb *= mix(vec3(0.5, 0.48, 0.45), vec3(1.0), smoothstep(0.1, 1.6, height)); // grime near the ground
	} else {
		c = texture(texture0, uv);
		// Recolor: the texture's own colors give way to its brightness, so the tint sets the
		// color while the shading and detail stay (outfits tinted lighter than they are).
		float l = dot(c.rgb, vec3(0.3, 0.59, 0.11));
		c.rgb = mix(c.rgb, vec3(min(1.0, l * 1.9)), recolor);
		c *= colDiffuse * vec4(vcol.rgb, 1.0);
	}
	if (c.a < 0.5) discard;
	vec3 n = normalize(normal);
	if (!gl_FrontFacing) n = -n; // leaves are double-sided cards
	float d = max(dot(n, -lightDir), 0.0);
	float metal = clamp((1.0 - vcol.a) * 2.0, 0.0, 1.0);
	float light = d > 0.6 ? 1.0 : (d > 0.2 ? 0.82 : 0.64);
	vec3 lit = c.rgb * light;
	if (metal > 0.0) {
		vec3 v = normalize(viewDir);
		float spec = pow(max(dot(n, normalize(v - lightDir)), 0.0), 24.0);
		vec3 steel = c.rgb * (d > 0.6 ? 1.05 : (d > 0.2 ? 0.75 : 0.5)); // harder bands than cloth
		if (spec > 0.35) steel = mix(steel, vec3(1.0), 0.6);
		if (1.0 - max(dot(n, v), 0.0) > 0.78) steel *= 0.62; // dark toward the silhouette
		lit = mix(lit, steel, metal);
	}
	fragColor = vec4(lit, c.a);
}`

// Outline: the model drawn a second time, pushed out along its normals, in ink, with front
// faces culled ("inverted hull"), so only a rim around the silhouette shows.
const outlineVS = `#version 330
in vec3 vertexPosition;
in vec2 vertexTexCoord;
in vec3 vertexNormal;
uniform mat4 mvp;
uniform float width;
out vec2 uv;
out vec3 local;
void main() {
	uv = vertexTexCoord;
	local = vertexPosition;
	gl_Position = mvp * vec4(vertexPosition + normalize(vertexNormal) * width, 1.0);
}`

const outlineFS = `#version 330
in vec2 uv;
in vec3 local;
uniform sampler2D texture0;
uniform vec3 ink;
uniform vec4 keep;
uniform float fade;
out vec4 fragColor;
void main() {
	if (keep.w > 0.0 && distance(local, keep.xyz) > keep.w) discard;
	if (fade > 0.0) discard;
	if (texture(texture0, uv).a < 0.5) discard;
	fragColor = vec4(ink, 1.0);
}`

const hiddenFS = "#version 330\nvoid main() { discard; }"

// Renderer holds the shaders.
type Renderer struct {
	toon, outline, hidden rl.Shader
	outlineWidth          int32
	keepToon, keepOutline int32
	fadeToon, fadeOutline int32
	detailToon            int32
	recolorToon           int32
	viewToon              int32
	Ground                *Ground
}

// NewRenderer compiles the shaders. Needs an open window.
func NewRenderer() *Renderer {
	r := &Renderer{
		toon:    rl.LoadShaderFromMemory(toonVS, toonFS),
		outline: rl.LoadShaderFromMemory(outlineVS, outlineFS),
		hidden:  rl.LoadShaderFromMemory(outlineVS, hiddenFS),
	}
	rl.SetShaderValue(r.toon, rl.GetShaderLocation(r.toon, "lightDir"), []float32{LightDir.X, LightDir.Y, LightDir.Z}, rl.ShaderUniformVec3)
	rl.SetShaderValue(r.outline, rl.GetShaderLocation(r.outline, "ink"), []float32{0.1, 0.08, 0.07}, rl.ShaderUniformVec3)
	r.outlineWidth = rl.GetShaderLocation(r.outline, "width")
	r.keepToon = rl.GetShaderLocation(r.toon, "keep")
	r.keepOutline = rl.GetShaderLocation(r.outline, "keep")
	r.fadeToon = rl.GetShaderLocation(r.toon, "fade")
	r.fadeOutline = rl.GetShaderLocation(r.outline, "fade")
	r.detailToon = rl.GetShaderLocation(r.toon, "detail")
	r.recolorToon = rl.GetShaderLocation(r.toon, "recolor")
	r.viewToon = rl.GetShaderLocation(r.toon, "viewDir")
	r.SetView(rl.NewVector3(0, 0.6, 0.8))
	return r
}

// SetView sets where the camera looks from (a direction toward it), for metal highlights.
func (r *Renderer) SetView(toward rl.Vector3) {
	v := rl.Vector3Normalize(toward)
	rl.SetShaderValue(r.toon, r.viewToon, []float32{v.X, v.Y, v.Z}, rl.ShaderUniformVec3)
}

// Object is a model placed in the scene.
type Object struct {
	Model   rl.Model
	Pos     rl.Vector3
	Yaw     float32 // degrees around the up axis; 0 faces +Z (toward the map camera)
	Scale   float32
	Tint    color.RGBA
	Outline float32   // outline width in meters, 0 for none
	NoInk   []bool    // per material: no outline (leaves and flowers)
	Keep    []float32 // only draw within this sphere (model space): x, y, z, radius
	Fade    float32   // 0..1: how see-through (dithered); faded objects have no outline
	Detail  float32   // texture repeats per meter for a surface texture (see Detail), 0 for UVs
	Recolor float32   // 0..1: how much the tint replaces the texture's colors instead of multiplying them
}

// Draw draws an object with its outline, then toon-shaded.
func (r *Renderer) Draw(o Object) {
	keep := o.Keep
	if keep == nil {
		keep = []float32{0, 0, 0, 0}
	}
	if o.Scale == 0 {
		o.Scale = 1
	}
	if o.Tint.A == 0 {
		o.Tint = rl.White
	}
	rl.SetShaderValue(r.toon, r.keepToon, keep, rl.ShaderUniformVec4)
	rl.SetShaderValue(r.outline, r.keepOutline, keep, rl.ShaderUniformVec4)
	rl.SetShaderValue(r.toon, r.fadeToon, []float32{o.Fade}, rl.ShaderUniformFloat)
	rl.SetShaderValue(r.outline, r.fadeOutline, []float32{o.Fade}, rl.ShaderUniformFloat)
	rl.SetShaderValue(r.toon, r.detailToon, []float32{o.Detail}, rl.ShaderUniformFloat)
	rl.SetShaderValue(r.toon, r.recolorToon, []float32{o.Recolor}, rl.ShaderUniformFloat)
	mats := o.Model.GetMaterials()
	scale := rl.NewVector3(o.Scale, o.Scale, o.Scale)
	up := rl.NewVector3(0, 1, 0)
	if o.Outline > 0 {
		for i := range mats {
			mats[i].Shader = r.outline
			if i < len(o.NoInk) && o.NoInk[i] {
				mats[i].Shader = r.hidden
			}
		}
		rl.SetShaderValue(r.outline, r.outlineWidth, []float32{o.Outline / o.Scale}, rl.ShaderUniformFloat)
		rl.SetCullFace(cullFront)
		rl.DrawModelEx(o.Model, o.Pos, up, o.Yaw, scale, rl.White)
		rl.SetCullFace(cullBack)
	}
	for i := range mats {
		mats[i].Shader = r.toon
	}
	rl.DisableBackfaceCulling() // double-sided foliage
	rl.DrawModelEx(o.Model, o.Pos, up, o.Yaw, scale, o.Tint)
	rl.EnableBackfaceCulling()
}

// Detail loads a surface texture (stone, plaster, wood...) for models without UVs, such as
// generated shapes: its colors are normalized to mid grey on average, so the object's tint
// sets the color and the texture only adds grain. Set it as the model's texture and give the
// Object a Detail scale.
func Detail(path string) rl.Texture2D {
	img := rl.LoadImage(path)
	defer rl.UnloadImage(img)
	rl.ImageFormat(img, rl.UncompressedR8g8b8a8)
	px := rl.LoadImageColors(img)
	var sum [3]float64
	for _, c := range px {
		sum[0] += float64(c.R)
		sum[1] += float64(c.G)
		sum[2] += float64(c.B)
	}
	n := float64(len(px))
	scale := func(v uint8, ch int) uint8 { return uint8(min(255, float64(v)*128*n/max(sum[ch], 1))) }
	for i, c := range px {
		px[i] = color.RGBA{scale(c.R, 0), scale(c.G, 1), scale(c.B, 2), 255}
	}
	tex := rl.LoadTextureFromImage(img)
	rl.UpdateTexture(tex, px)
	rl.UnloadImageColors(px)
	rl.GenTextureMipmaps(&tex)
	rl.SetTextureFilter(tex, rl.FilterTrilinear)
	rl.SetTextureWrap(tex, rl.WrapRepeat)
	return tex
}

// Library loads models and animations once and keeps them.
type Library struct {
	Dir    string // assets/quaternius
	files  []string
	models map[string]rl.Model
	inks   map[string][]bool
	anims  map[string]*Anims
}

// NewLibrary returns a library for the packs in dir.
func NewLibrary(dir string) *Library {
	return &Library{Dir: dir, models: map[string]rl.Model{}, inks: map[string][]bool{}, anims: map[string]*Anims{}}
}

// Find returns the path of the first glTF/GLB file with this name in the packs ("" if none),
// or of a file outside them given as a relative path starting with "../". The name may
// include folders ("outfits/glTF/Name.gltf") where a file name exists in several
// packs. The packs are indexed once; pack folder names contain brackets, so filepath.Glob
// can't be used.
func (l *Library) Find(name string) string {
	// A path relative to the packs, e.g. "../generated/creatures/Scarab.gltf" for models
	// made by tools/models.
	if strings.HasPrefix(name, "../") {
		return filepath.Join(l.Dir, name)
	}
	if l.files == nil {
		filepath.WalkDir(l.Dir, func(p string, d os.DirEntry, err error) error {
			if err == nil && !d.IsDir() && (strings.HasSuffix(p, ".gltf") || strings.HasSuffix(p, ".glb")) &&
				!strings.Contains(p, "FBX") && !strings.Contains(p, "Origin at 0") {
				l.files = append(l.files, filepath.ToSlash(p)) // the names below are matched with "/"
			}
			return nil
		})
	}
	// Prefer the light copy without unused texture maps (tools/models/prepare.py).
	if lite, ok := strings.CutSuffix(name, ".gltf"); ok {
		for _, p := range l.files {
			if p == lite+".lite.gltf" || strings.HasSuffix(p, "/"+lite+".lite.gltf") {
				return p
			}
		}
	}
	for _, p := range l.files {
		if p == name || strings.HasSuffix(p, "/"+name) {
			return p
		}
	}
	return ""
}

// Model returns a shared model by file name. Shared models must not be animated per
// instance unless the animation is applied right before each draw.
func (l *Library) Model(name string) rl.Model {
	if m, ok := l.models[name]; ok {
		return m
	}
	path := l.Find(name)
	m := rl.LoadModel(path)
	l.models[name] = m
	l.inks[name] = foliage(path)
	return m
}

// Fresh loads a model that isn't shared (for animated characters with their own pose).
func (l *Library) Fresh(name string) rl.Model { return rl.LoadModel(l.Find(name)) }

// NoInk returns which materials of a model get no outline: alpha-masked leaves in the packs,
// and thin parts of tools/models/nature.py (materials named "Thin...").
func (l *Library) NoInk(name string) []bool {
	l.Model(name)
	return l.inks[name]
}

// Anims returns the animations in a file.
func (l *Library) Anims(name string) *Anims {
	if a, ok := l.anims[name]; ok {
		return a
	}
	a := &Anims{List: rl.LoadModelAnimations(l.Find(name))}
	l.anims[name] = a
	return a
}

// foliage reports for each material of a glTF model as raylib loads it (material 0 is
// raylib's default) whether it is foliage: leaves, flowers and grass get no outline.
func foliage(path string) []bool {
	var doc struct {
		Materials []struct {
			Name      string `json:"name"`
			AlphaMode string `json:"alphaMode"`
		} `json:"materials"`
	}
	data, err := os.ReadFile(path)
	if err != nil || json.Unmarshal(data, &doc) != nil {
		return nil
	}
	out := make([]bool, len(doc.Materials)+1)
	for i, m := range doc.Materials {
		out[i+1] = m.AlphaMode == "MASK" && !strings.HasPrefix(m.Name, "Bark") || strings.HasPrefix(m.Name, "Thin")
	}
	return out
}

// Height returns a model's height in its own units (for scaling packs to real sizes).
func Height(m rl.Model) float32 {
	b := rl.GetModelBoundingBox(m)
	return b.Max.Y - b.Min.Y
}

// Anims is a set of animations.
type Anims struct {
	List  []rl.ModelAnimation
	speed map[int]float32 // playback speed per animation, if not 1 (SetSpeed)
}

// SetSpeed sets how fast an animation plays: 0.5 is half speed.
func (a *Anims) SetSpeed(i int, speed float32) {
	if a.speed == nil {
		a.speed = map[int]float32{}
	}
	a.speed[i] = speed
}

// rate is an animation's frames per second: raylib samples glTF animations at 60, times its
// speed.
func (a *Anims) rate(i int) float32 {
	if s, ok := a.speed[i]; ok {
		return 60 * s
	}
	return 60
}

// Trim adds a part of an animation as a new one, from and to given as shares of its length,
// and returns its index (or -1 if there is no such animation). The new animation shares the
// original's frames.
func (a *Anims) Trim(name, as string, from, to float32) int {
	if i := a.Find(as); i >= 0 {
		return i
	}
	i := a.Find(name)
	if i < 0 {
		return -1
	}
	an := a.List[i]
	start := int32(from * float32(an.FrameCount))
	end := max(start+1, min(an.FrameCount, int32(to*float32(an.FrameCount))))
	an.FramePoses = (**rl.Transform)(unsafe.Add(unsafe.Pointer(an.FramePoses), uintptr(start)*unsafe.Sizeof(an.FramePoses)))
	an.FrameCount = end - start
	an.Name = [32]uint8{}
	copy(an.Name[:31], as)
	a.List = append(a.List, an)
	return len(a.List) - 1
}

// Find returns the index of the first animation with one of the names (-1 if none).
func (a *Anims) Find(names ...string) int {
	for _, n := range names {
		for i, an := range a.List {
			if an.GetName() == n {
				return i
			}
		}
	}
	return -1
}

// Moves are the animations a fighter needs, as indices into its Anims.
type Moves struct {
	Idle, Walk, Run, Attack, Hit, Death, Cast int
	Shoot                                     int // a ranged attack (bow)
	Ready                                     int // the stance in battle
	Rest                                      int // sitting (on a seat about 0.5 m high)
}

// MovesFor picks the animations by their usual names in the Quaternius packs.
func MovesFor(a *Anims) Moves {
	m := Moves{
		Idle:   a.Find("Idle_Loop", "Idle", "Flying_Idle"),
		Walk:   a.Find("Walk_Loop", "Walk", "Fast_Flying", "Flying_Idle"),
		Attack: a.Find("Sword_Attack", "Attack", "Attack_Headbutt", "Bite_Front", "Weapon", "Punch", "Headbutt", "Jump"),
		Hit:    a.Find("Hit_Chest", "HitReact", "HitRecieve", "Idle_HitReact1"),
		Death:  a.Find("Death01", "Death"),
		Cast:   a.Find("Spell_Simple_Shoot", "Punch_Jab"),
		Shoot:  a.Find("Pistol_Shoot", "Spell_Simple_Shoot", "Punch_Jab"),
		Run:    a.Find("Jog_Fwd_Loop", "Run", "Gallop", "Fast_Flying"),
		Ready:  a.Find("Sword_Idle", "Idle_Loop", "Idle", "Flying_Idle"),
		Rest:   a.Find("Sitting_Idle_Loop", "Idle_Loop", "Idle", "Flying_Idle"),
	}
	if m.Run < 0 {
		m.Run = m.Walk
	}
	for _, p := range []*int{&m.Walk, &m.Run, &m.Attack, &m.Hit, &m.Death, &m.Cast, &m.Shoot, &m.Ready, &m.Rest} {
		if *p < 0 {
			*p = m.Idle
		}
	}
	return m
}

// Player plays animations: one looping animation, optionally interrupted by a one-shot
// (attack, hit) that returns to the loop when done, or holds its last frame (death).
type Player struct {
	Anims  *Anims
	loop   int
	once   int
	hold   bool
	time   float32
	frame  int32
	offset float32 // start offset, so a group doesn't move in step
}

// NewPlayer returns a player starting with a loop.
func NewPlayer(a *Anims, loop int, offset float32) *Player {
	return &Player{Anims: a, loop: loop, once: -1, offset: offset}
}

// Loop sets the looping animation.
func (p *Player) Loop(i int) {
	if i != p.loop && p.once < 0 {
		p.time = 0
	}
	p.loop = i
}

// Once plays an animation once; with hold it stays on the last frame. Asking again for the
// animation that is already held (a death pose) does nothing.
func (p *Player) Once(i int, hold bool) {
	if p.hold && p.once == i {
		return
	}
	p.once, p.hold, p.time = i, hold, 0
}

// Reset drops a one-shot animation (e.g. a held death pose) and plays a loop.
func (p *Player) Reset(loop int) {
	p.loop, p.once, p.hold, p.time = loop, -1, false, 0
}

// Busy reports whether a one-shot animation is playing.
func (p *Player) Busy() bool { return p.once >= 0 && !p.hold }

// Update advances time; animations play at 60 frames per second times their speed.
func (p *Player) Update(dt float32) {
	if p.Anims == nil || len(p.Anims.List) == 0 {
		return
	}
	p.time += dt
	cur := p.loop
	if p.once >= 0 {
		cur = p.once
	}
	if cur < 0 {
		return
	}
	an := p.Anims.List[cur]
	rate := p.Anims.rate(cur)
	f := int32(p.time * rate)
	if p.once >= 0 {
		if f >= an.FrameCount-1 {
			if p.hold {
				f = an.FrameCount - 1
			} else {
				p.once, p.time = -1, 0
				p.Update(0)
				return
			}
		}
	} else {
		f = int32((p.time+p.offset)*rate) % an.FrameCount
	}
	p.frame = f
}

// Current returns the playing animation and frame.
func (p *Player) Current() (rl.ModelAnimation, int32, bool) {
	if p.Anims == nil || len(p.Anims.List) == 0 {
		return rl.ModelAnimation{}, 0, false
	}
	cur := p.loop
	if p.once >= 0 {
		cur = p.once
	}
	if cur < 0 {
		return rl.ModelAnimation{}, 0, false
	}
	return p.Anims.List[cur], p.frame, true
}

// Pose applies the current frame to a model (call right before drawing it).
func (p *Player) Pose(m rl.Model) {
	if an, f, ok := p.Current(); ok {
		rl.UpdateModelAnimation(m, an, f)
	}
}

// Creature is an animated model that may be shared between several creatures: each draw
// applies its own pose first.
type Creature struct {
	Model  rl.Model
	Anims  *Anims
	Moves  Moves
	Scale  float32
	Tint   color.RGBA
	NoInk  []bool
	Height float32 // in meters, after scaling
	Detail float32 // surface texture repeats per model unit (see Detail), 0 for none
}

// Draw poses and draws the creature.
func (r *Renderer) DrawCreature(c *Creature, p *Player, pos rl.Vector3, yaw float32) {
	p.Pose(c.Model)
	r.Draw(Object{Model: c.Model, Pos: pos, Yaw: yaw, Scale: c.Scale, Tint: c.Tint, Outline: 0.012, NoInk: c.NoInk, Detail: c.Detail})
}

// Character is a humanoid built from parts that share one skeleton: the base body (only its
// head is drawn), outfit pieces per equipment slot, hair, and items held at bones.
type Character struct {
	Parts []Object
	Held  []Held
	Play  *Player
	Moves Moves
	Stow  bool // don't draw the held items (e.g. while sitting at camp)
	Fight bool // in the battle stance: held items use their battle offset, if they have one
	bones map[string]int

	showBody bool // see Outfit.ShowBody
}

// Held is a static model (weapon, shield) drawn at a bone with an offset.
type Held struct {
	Object
	Bone   string
	Offset rl.Matrix
	// FightOffset replaces Offset in the battle stance, whose hand pose differs (HoldFight).
	FightOffset *rl.Matrix
}

// Outfit names the outfit part for each equipment slot (files in the outfit pack) and the
// look of the head.
type Outfit struct {
	Body                               string // base body file
	Hair                               []string
	HairTint                           color.RGBA
	Chest, Arms, Legs, Feet, Shoulders string
	Head                               string // e.g. a hood; hides the hair
	Tints                              map[string]color.RGBA
	Recolor                            float32 // how much tints replace the parts' colors (see Object.Recolor)
	// ShowBody draws the whole base body, not only its head: for bodies made by
	// tools/models/human.py, whose outfit pieces don't include the skin they leave bare.
	ShowBody bool
}

// NewCharacter loads a character's parts. anims holds the shared humanoid animations.
func (l *Library) NewCharacter(o Outfit, anims *Anims) *Character {
	c := &Character{bones: map[string]int{}, Moves: MovesFor(anims), showBody: o.ShowBody}
	c.Play = NewPlayer(anims, c.Moves.Idle, 0)
	add := func(file string, tint color.RGBA, recolor float32) {
		m := l.Fresh(file + ".gltf")
		if tint.A == 0 {
			tint, recolor = rl.White, 0
		}
		c.Parts = append(c.Parts, Object{Model: m, Scale: 1, Tint: tint, Outline: 0.012, Recolor: recolor})
	}
	add(o.Body, color.RGBA{}, 0)
	if o.Head == "" {
		for _, h := range o.Hair {
			add(h, o.HairTint, 0)
		}
	}
	for _, part := range []string{o.Head, o.Chest, o.Arms, o.Legs, o.Feet, o.Shoulders} {
		if part != "" {
			add(part, o.Tints[part], o.Recolor)
		}
	}
	for i, b := range c.Parts[0].Model.GetBones() {
		c.bones[cString(b.Name[:])] = i
	}
	return c
}

// Hold attaches a static model to a bone; rot is in degrees, move in meters (bone space).
func (l *Library) Hold(c *Character, file, bone string, rot, move rl.Vector3) {
	m := l.Fresh(file)
	c.Held = append(c.Held, Held{Object: Object{Model: m, Scale: 1, Outline: 0.012}, Bone: bone, Offset: holdOffset(rot, move)})
}

// HoldFight sets the offset of the last held item for the battle stance.
func (c *Character) HoldFight(rot, move rl.Vector3) {
	off := holdOffset(rot, move)
	c.Held[len(c.Held)-1].FightOffset = &off
}

func holdOffset(rot, move rl.Vector3) rl.Matrix {
	return rl.MatrixMultiply(rl.MatrixRotateXYZ(rl.Vector3Scale(rot, math.Pi/180)), rl.MatrixTranslate(move.X, move.Y, move.Z))
}

// DrawCharacter poses and draws a character at a position.
func (r *Renderer) DrawCharacter(c *Character, pos rl.Vector3, yaw float32) {
	an, frame, ok := c.Play.Current()
	if !ok {
		return
	}
	// Only the head of the base body is shown; the outfit covers the rest.
	head := an.GetFramePose(int(frame), c.bones["Head"]).Translation
	for i, p := range c.Parts {
		rl.UpdateModelAnimation(p.Model, an, frame)
		p.Pos, p.Yaw = pos, yaw
		if i == 0 && !c.showBody {
			p.Keep = []float32{head.X, head.Y + 0.07, head.Z, 0.17}
		}
		r.Draw(p)
	}
	for _, h := range c.Held {
		if c.Stow {
			break
		}
		t := an.GetFramePose(int(frame), c.bones[h.Bone])
		pose := rl.MatrixMultiply(rl.MatrixMultiply(rl.MatrixScale(t.Scale.X, t.Scale.Y, t.Scale.Z),
			rl.QuaternionToMatrix(t.Rotation)), rl.MatrixTranslate(t.Translation.X, t.Translation.Y, t.Translation.Z))
		off := h.Offset
		if c.Fight && h.FightOffset != nil {
			off = *h.FightOffset
		}
		h.Model.Transform = rl.MatrixMultiply(off, pose)
		h.Pos, h.Yaw = pos, yaw
		r.Draw(h.Object)
	}
}

func cString(b []int8) string {
	n := 0
	for n < len(b) && b[n] != 0 {
		n++
	}
	out := make([]byte, n)
	for i := range out {
		out[i] = byte(b[i])
	}
	return string(out)
}

// Shadow draws a soft round shadow on the ground.
func Shadow(pos rl.Vector3, radius float32) {
	ShadowAlpha(pos, radius, 60)
}

// ShadowAlpha draws a round shadow with the given opacity.
func ShadowAlpha(pos rl.Vector3, radius float32, alpha uint8) {
	rl.DrawCylinder(rl.NewVector3(pos.X, pos.Y+0.01, pos.Z), radius, radius, 0.005, 20, color.RGBA{0, 0, 0, alpha})
}
