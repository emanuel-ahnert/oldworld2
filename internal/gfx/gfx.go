// Package gfx is a small 2D drawing layer on raylib for the game's pixel UI (menus, HUD,
// battle boxes, the title screen). It offers images that can be drawn onto each other with
// scaling, mirroring and color tinting, and filled rectangles, which is all the UI needs.
//
// Images are raylib textures; drawable images are render textures. Drawing switches the
// render target as needed, so callers must call Flush before drawing anything else with
// raylib directly (e.g. 3D scenes) and at the end of the frame.
package gfx

import (
	"image"
	"image/color"
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// Filter selects how images are sampled when scaled.
type Filter int

const (
	FilterNearest Filter = iota
	FilterLinear
)

// Image is a texture, part of a texture, a drawable render texture, or the screen.
type Image struct {
	tex    rl.Texture2D
	target *rl.RenderTexture2D // set for drawable images
	rect   image.Rectangle     // the part of tex this image covers, in tex pixels
	screen bool
	filter Filter
}

var current *Image // the image raylib currently draws into (nil: the screen)

// Screen returns the window as an image; its size follows the window.
func Screen() *Image {
	return &Image{screen: true, rect: image.Rect(0, 0, rl.GetRenderWidth(), rl.GetRenderHeight())}
}

// NewImage returns a drawable, transparent image.
func NewImage(w, h int) *Image {
	rt := rl.LoadRenderTexture(int32(w), int32(h))
	rl.SetTextureFilter(rt.Texture, rl.FilterPoint)
	img := &Image{tex: rt.Texture, target: &rt, rect: image.Rect(0, 0, w, h)}
	img.Clear()
	return img
}

// NewImageFromImage uploads an image as a texture.
func NewImageFromImage(src image.Image) *Image {
	tex := rl.LoadTextureFromImage(rl.NewImageFromImage(src))
	rl.SetTextureFilter(tex, rl.FilterPoint)
	b := src.Bounds()
	return &Image{tex: tex, rect: image.Rect(0, 0, b.Dx(), b.Dy())}
}

// Bounds returns the image's rectangle; for sub-images it is in the parent's coordinates.
func (i *Image) Bounds() image.Rectangle { return i.rect }

// SubImage returns the part of the image inside r (in the image's coordinates).
func (i *Image) SubImage(r image.Rectangle) *Image {
	sub := *i
	sub.rect = r.Intersect(i.rect)
	sub.screen = false
	return &sub
}

// Texture returns the raylib texture behind the image (for 3D billboards).
func (i *Image) Texture() rl.Texture2D { return i.tex }

// Deallocate frees a drawable image.
func (i *Image) Deallocate() {
	if i.target != nil {
		if current == i {
			Flush()
		}
		rl.UnloadRenderTexture(*i.target)
		i.target = nil
	}
}

// use makes dst the render target.
func use(dst *Image) {
	if dst.screen {
		dst = nil
	}
	if dst == current {
		return
	}
	if current != nil {
		rl.EndTextureMode()
	}
	if dst != nil {
		rl.BeginTextureMode(*dst.target)
	}
	current = dst
}

// Flush ends drawing into an image, so raylib draws to the screen again.
func Flush() { use(&Image{screen: true}) }

// Fill sets every pixel of a drawable image (or the screen) to c.
func (i *Image) Fill(c color.Color) {
	use(i)
	rl.ClearBackground(rgba(c))
}

// Clear makes a drawable image transparent.
func (i *Image) Clear() { i.Fill(color.RGBA{}) }

// GeoM is a 2D transform made of scaling and translation (no rotation).
type GeoM struct {
	sx, sy, tx, ty float64
	set            bool
}

func (g *GeoM) init() {
	if !g.set {
		g.sx, g.sy, g.set = 1, 1, true
	}
}

// Scale scales everything drawn so far by (x, y); negative values mirror.
func (g *GeoM) Scale(x, y float64) {
	g.init()
	g.sx, g.sy, g.tx, g.ty = g.sx*x, g.sy*y, g.tx*x, g.ty*y
}

// Translate moves by (x, y).
func (g *GeoM) Translate(x, y float64) {
	g.init()
	g.tx, g.ty = g.tx+x, g.ty+y
}

// ColorScale multiplies the drawn colors.
type ColorScale struct {
	r, g, b, a float32
	set        bool
}

func (c *ColorScale) init() {
	if !c.set {
		c.r, c.g, c.b, c.a, c.set = 1, 1, 1, 1, true
	}
}

// ScaleWithColor multiplies by a color.
func (c *ColorScale) ScaleWithColor(col color.Color) {
	c.init()
	n := color.NRGBAModel.Convert(col).(color.NRGBA)
	c.r *= float32(n.R) / 255
	c.g *= float32(n.G) / 255
	c.b *= float32(n.B) / 255
	c.a *= float32(n.A) / 255
}

// ScaleAlpha multiplies the opacity.
func (c *ColorScale) ScaleAlpha(a float32) {
	c.init()
	c.a *= a
}

// DrawImageOptions describe how DrawImage draws.
type DrawImageOptions struct {
	GeoM       GeoM
	ColorScale ColorScale
	Filter     Filter
}

// DrawImage draws src onto i.
func (i *Image) DrawImage(src *Image, op *DrawImageOptions) {
	if op == nil {
		op = &DrawImageOptions{}
	}
	op.GeoM.init()
	op.ColorScale.init()
	use(i)
	if src.filter != op.Filter || op.Filter == FilterLinear {
		f := rl.FilterPoint
		if op.Filter == FilterLinear {
			f = rl.FilterBilinear
		}
		rl.SetTextureFilter(src.tex, f)
		src.filter = op.Filter
	}
	w, h := float64(src.rect.Dx()), float64(src.rect.Dy())
	g := op.GeoM
	x0, x1 := g.tx, g.tx+g.sx*w
	y0, y1 := g.ty, g.ty+g.sy*h
	sr := rl.NewRectangle(float32(src.rect.Min.X), float32(src.rect.Min.Y), float32(w), float32(h))
	if src.target != nil {
		// Render textures are stored upside down.
		sr.Y = float32(src.tex.Height) - sr.Y - sr.Height
		sr.Height = -sr.Height
	}
	if g.sx < 0 {
		sr.Width = -sr.Width
	}
	if g.sy < 0 {
		sr.Height = -sr.Height
	}
	dr := rl.NewRectangle(float32(math.Min(x0, x1)), float32(math.Min(y0, y1)), float32(math.Abs(x1-x0)), float32(math.Abs(y1-y0)))
	c := op.ColorScale
	tint := rl.NewColor(unit(c.r), unit(c.g), unit(c.b), unit(c.a))
	rl.DrawTexturePro(src.tex, sr, dr, rl.Vector2{}, 0, tint)
}

// FillRect fills a rectangle.
func FillRect(dst *Image, x, y, w, h float32, c color.Color) {
	use(dst)
	rl.DrawRectangleRec(rl.NewRectangle(x, y, w, h), rgba(c))
}

// StrokeRect draws a rectangle outline of the given width, centered on the rectangle's edge.
func StrokeRect(dst *Image, x, y, w, h, width float32, c color.Color) {
	use(dst)
	rl.DrawRectangleLinesEx(rl.NewRectangle(x-width/2, y-width/2, w+width, h+width), width, rgba(c))
}

func rgba(c color.Color) color.RGBA {
	n := color.NRGBAModel.Convert(c).(color.NRGBA)
	return color.RGBA{n.R, n.G, n.B, n.A}
}

func unit(v float32) uint8 { return uint8(math.Round(float64(max(0, min(1, v)) * 255))) }
