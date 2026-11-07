package r3d

import (
	"image/color"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// Ground draws the terrain around the camera: a flat square textured from an atlas of
// seamless ground textures. Which texture shows where comes from two masks with one pixel per
// tile (red: water, green: dirt road, blue: sand, alpha: sand path; second mask red: snow,
// green: ice, blue: snowy road, alpha: dry grass; third mask red: dead grass), sampled smoothly and roughened with noise, so edges are natural instead of
// following the tile grid. Water, ice and roads get an ink line along their edge.

// Ground texture slots in the atlas.
const (
	TexGrass = iota
	TexDirt
	TexWater
	TexSand
	TexPath
	TexSnow
	TexIce
	numTex
)

const groundVS = `#version 330
in vec3 vertexPosition;
uniform mat4 mvp;
uniform mat4 matModel;
out vec2 world;
void main() {
	world = (matModel * vec4(vertexPosition, 1.0)).xz;
	gl_Position = mvp * vec4(vertexPosition, 1.0);
}`

const groundFS = `#version 330
in vec2 world;
uniform sampler2D texture0; // atlas, 3×3 cells
uniform sampler2D texture1; // mask
uniform sampler2D texture2; // second mask
uniform sampler2D texture3; // third mask
uniform vec2 maskOrigin;
uniform float maskSize;
uniform float tileMeters;
out vec4 fragColor;

float hash(vec2 p) { return fract(sin(dot(p, vec2(127.1, 311.7))) * 43758.5453); }
float noise(vec2 p) {
	vec2 i = floor(p), f = fract(p), u = f * f * (3.0 - 2.0 * f);
	return mix(mix(hash(i), hash(i + vec2(1, 0)), u.x), mix(hash(i + vec2(0, 1)), hash(i + vec2(1, 1)), u.x), u.y);
}

vec3 cell(int i, vec2 w) {
	vec2 c = vec2(float(i % 3), float(i / 3));
	vec2 u = w / tileMeters;
	vec2 a = (c + fract(u) * 0.996 + 0.002) / 3.0;
	return textureGrad(texture0, a, dFdx(u) / 3.0, dFdy(u) / 3.0).rgb;
}

// Two scales of the same texture, so the repetition doesn't show.
vec3 tex(int i) { return mix(cell(i, world), cell(i, world * 0.7 + vec2(3.7, 1.3)), 0.5); }

// Sun-dried grass: the grass texture, faded and warmed toward straw.
vec3 dryGrass() {
	vec3 g = tex(0);
	float l = dot(g, vec3(0.3, 0.59, 0.11));
	return mix(vec3(l), g, 0.3) * vec3(1.55, 1.36, 0.7);
}

// The unholy lands' dead grass: the grass texture drained and withered to a dull brown-grey.
vec3 deadGrass() {
	vec3 g = tex(0);
	float l = dot(g, vec3(0.3, 0.59, 0.11));
	return mix(vec3(l), g, 0.35) * vec3(0.98, 0.84, 0.82);
}

// A road through snow: trampled, greyer snow with the dirt showing through in patches of slush.
vec3 snowRoad() {
	mat2 turn = mat2(0.8, -0.6, 0.6, 0.8); // rotated octaves, so the noise's grid doesn't show
	float n = noise(world * 0.8) * 0.5 + noise(turn * world * 2.1 + 5.3) * 0.32 + noise(turn * turn * world * 5.7 + 1.7) * 0.18;
	float slush = smoothstep(0.5, 0.75, n);
	return mix(tex(5) * vec3(0.86, 0.88, 0.92), tex(1) * vec3(0.82, 0.8, 0.78), slush * 0.75);
}

vec3 blend(vec3 base, float m, vec3 t, vec3 ink, float line) {
	float n = noise(world * 2.3) * 0.6 + noise(world * 6.0) * 0.4 - 0.5;
	float v = m + n * 0.45;
	float d = (v - 0.5) / max(fwidth(v), 0.0001); // signed distance to the edge in pixels
	float inside = clamp(d + 0.5, 0.0, 1.0);
	t *= 1.0 - clamp(1.0 - d / 12.0, 0.0, 1.0) * 0.2 * line; // darker band along the edge
	t = mix(t, ink, clamp(line * 2.5 - d, 0.0, 1.0) * line);
	return mix(base, t, inside);
}

void main() {
	vec4 m = texture(texture1, (world - maskOrigin) / maskSize);
	vec4 m2 = texture(texture2, (world - maskOrigin) / maskSize);
	vec4 m3 = texture(texture3, (world - maskOrigin) / maskSize);
	vec3 c = tex(0) * 1.12;
	c = blend(c, m2.a, dryGrass(), vec3(0), 0.0);
	c = blend(c, m3.r, deadGrass(), vec3(0), 0.0);
	c = blend(c, m2.r, tex(5) * 1.12, vec3(0), 0.0);
	c = blend(c, m.b, tex(3), vec3(0), 0.0);
	c = blend(c, m.a, tex(4), vec3(0), 0.0);
	c = blend(c, m.g, tex(1) * mix(vec3(1.0), vec3(0.6, 0.54, 0.58), m3.r), vec3(0.18, 0.12, 0.08), 1.0); // roads darker on dead ground
	c = blend(c, m2.b, snowRoad(), vec3(0.22, 0.2, 0.2), 0.6);
	c = blend(c, m.r, tex(2), vec3(0.12, 0.16, 0.24), 1.0);
	c = blend(c, m2.g, tex(6) * vec3(0.78, 0.92, 1.1), vec3(0.2, 0.28, 0.38), 1.0); // bluer than the snow
	fragColor = vec4(c, 1.0);
}`

// Ground is the terrain renderer.
type Ground struct {
	shader     rl.Shader
	model      rl.Model
	mask       rl.Texture2D
	mask2      rl.Texture2D
	mask3      rl.Texture2D
	size       int
	pixels     []color.RGBA
	pixels2    []color.RGBA
	pixels3    []color.RGBA
	originLoc  int32
	originX    float32
	originZ    float32
	TileMeters float32
}

// NewGround builds the atlas from the ground textures (paths per slot; empty or missing
// files get a plain color) and a mask covering size×size tiles.
func NewGround(textures [numTex]string, size int) *Ground {
	const cellPx = 512
	atlas := rl.GenImageColor(3*cellPx, 3*cellPx, rl.White)
	fallback := [numTex]color.RGBA{{90, 140, 60, 255}, {120, 90, 60, 255}, {60, 110, 160, 255}, {210, 180, 120, 255}, {225, 205, 160, 255},
		{235, 240, 248, 255}, {170, 205, 225, 255}}
	for i, path := range textures {
		dst := rl.NewRectangle(float32(i%3*cellPx), float32(i/3*cellPx), cellPx, cellPx)
		img := rl.LoadImage(path)
		if path == "" || img.Data == nil || img.Width == 0 {
			rl.ImageDrawRectangleRec(atlas, dst, fallback[i])
			continue
		}
		rl.ImageResize(img, cellPx, cellPx)
		rl.ImageDraw(atlas, img, rl.NewRectangle(0, 0, cellPx, cellPx), dst, rl.White)
		rl.UnloadImage(img)
	}
	tex := rl.LoadTextureFromImage(atlas)
	rl.UnloadImage(atlas)
	rl.GenTextureMipmaps(&tex)
	rl.SetTextureFilter(tex, rl.FilterTrilinear)

	g := &Ground{size: size, pixels: make([]color.RGBA, size*size), pixels2: make([]color.RGBA, size*size),
		pixels3: make([]color.RGBA, size*size), TileMeters: 8}
	maskImg := rl.GenImageColor(size, size, rl.Blank)
	g.mask = rl.LoadTextureFromImage(maskImg)
	g.mask2 = rl.LoadTextureFromImage(maskImg)
	g.mask3 = rl.LoadTextureFromImage(maskImg)
	rl.UnloadImage(maskImg)
	for _, m := range []rl.Texture2D{g.mask, g.mask2, g.mask3} {
		rl.SetTextureFilter(m, rl.FilterBilinear)
		rl.SetTextureWrap(m, rl.WrapClamp)
	}

	g.shader = rl.LoadShaderFromMemory(groundVS, groundFS)
	g.originLoc = rl.GetShaderLocation(g.shader, "maskOrigin")
	rl.SetShaderValue(g.shader, rl.GetShaderLocation(g.shader, "maskSize"), []float32{float32(size)}, rl.ShaderUniformFloat)
	g.model = rl.LoadModelFromMesh(rl.GenMeshPlane(float32(size), float32(size), 1, 1))
	mat := &g.model.GetMaterials()[0]
	mat.Shader = g.shader
	mat.GetMap(rl.MapDiffuse).Texture = tex
	mat.GetMap(rl.MapSpecular).Texture = g.mask
	mat.GetMap(rl.MapNormal).Texture = g.mask2
	// raylib binds texture0-2 by name; the third mask goes in the roughness map's slot.
	g.shader.UpdateLocation(rl.ShaderLocMapRoughness, rl.GetShaderLocation(g.shader, "texture3"))
	mat.GetMap(rl.MapRoughness).Texture = g.mask3
	return g
}

// Ground kinds per tile, for SetMask: what lies on the base ground.
type Kind uint8

const (
	Plain Kind = iota // just the base ground
	Dirt              // a road (on snow: trampled snow and slush)
	Water
	Sand
	Path // a sand path (on sand)
	Ice
)

// Base is the ground a zone is covered in, under roads, water and sand.
type Base uint8

const (
	GrassBase Base = iota
	SnowBase
	DryBase  // the Mediterranean's sun-dried grass
	DeadBase // the unholy lands' dead grass
)

// SetMask fills the mask for the tiles from (x0, z0) on; kind tells what a tile is and what
// ground it lies on.
func (g *Ground) SetMask(x0, z0 int, kind func(x, z int) (Kind, Base)) {
	for z := 0; z < g.size; z++ {
		for x := 0; x < g.size; x++ {
			var c, c2, c3 color.RGBA
			k, base := kind(x0+x, z0+z)
			switch base {
			case SnowBase:
				c2.R = 255
			case DryBase:
				c2.A = 255
			case DeadBase:
				c3.R = 255
			}
			switch k {
			case Water:
				c.R = 255
			case Dirt:
				if base == SnowBase {
					c2.B = 255 // a snowy road
				} else {
					c.G = 255
				}
			case Sand:
				c.B = 255
			case Path:
				c.B, c.A = 255, 255
			case Ice:
				c2.G = 255
			}
			g.pixels[z*g.size+x] = c
			g.pixels2[z*g.size+x] = c2
			g.pixels3[z*g.size+x] = c3
		}
	}
	rl.UpdateTexture(g.mask, g.pixels)
	rl.UpdateTexture(g.mask2, g.pixels2)
	rl.UpdateTexture(g.mask3, g.pixels3)
	// Texel centers sit on tile centers: tile x covers world x..x+1.
	g.originX, g.originZ = float32(x0), float32(z0)
	rl.SetShaderValue(g.shader, g.originLoc, []float32{g.originX, g.originZ}, rl.ShaderUniformVec2)
	rl.SetShaderValue(g.shader, rl.GetShaderLocation(g.shader, "tileMeters"), []float32{g.TileMeters}, rl.ShaderUniformFloat)
}

// Draw draws the ground square under the masked area.
func (g *Ground) Draw() {
	half := float32(g.size) / 2
	rl.DrawModel(g.model, rl.NewVector3(g.originX+half, 0, g.originZ+half), 1, rl.White)
}
