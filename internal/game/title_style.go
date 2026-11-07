package game

import (
	"image"
	"image/color"
	"math"

	"oldworld2/internal/assets"
	"oldworld2/internal/gfx"
)

// The title screen's fantasy dressing: a gilded logo, framed parchment panels, a gem for the
// cursor, and a warm, vignetted backdrop with glowing motes drifting up.

var (
	gilt     = color.RGBA{230, 180, 70, 255}
	giltDark = color.RGBA{150, 100, 35, 255}
	giltLit  = color.RGBA{255, 235, 150, 255}
)

// fancyPanel is a parchment panel in a dark frame with a gilt inner line and gilt corner studs.
func fancyPanel(dst *gfx.Image, x, y, w, h int) {
	fx, fy, fw, fh := float32(x), float32(y), float32(w), float32(h)
	gfx.FillRect(dst, fx, fy, fw, fh, paper)
	gfx.StrokeRect(dst, fx+1, fy+1, fw-2, fh-2, 2, ink)
	gfx.StrokeRect(dst, fx+3.5, fy+3.5, fw-7, fh-7, 1, gilt)
	for _, c := range [][2]float32{{fx, fy}, {fx + fw - 5, fy}, {fx, fy + fh - 5}, {fx + fw - 5, fy + fh - 5}} {
		gfx.FillRect(dst, c[0], c[1], 5, 5, ink)
		gfx.FillRect(dst, c[0]+1, c[1]+1, 3, 3, gilt)
		gfx.FillRect(dst, c[0]+1, c[1]+1, 1, 1, giltLit)
	}
}

// gem draws a small gilt diamond with its top-left at (x, y): the menu cursor.
func gem(dst *gfx.Image, x, y int, c color.RGBA) {
	fx, fy := float32(x), float32(y)
	rows := []struct{ dx, w float32 }{{2, 1}, {1, 3}, {0, 5}, {1, 3}, {2, 1}}
	for i, r := range rows {
		gfx.FillRect(dst, fx+r.dx-1, fy+float32(i), r.w+2, 1, ink) // outline
	}
	gfx.FillRect(dst, fx+1, fy-1, 1, 1, ink)
	gfx.FillRect(dst, fx+1, fy+5, 1, 1, ink)
	for i, r := range rows {
		gfx.FillRect(dst, fx+r.dx, fy+float32(i), r.w, 1, c)
	}
	gfx.FillRect(dst, fx+1, fy+1, 1, 1, giltLit)
}

// flourish draws an ornamental rule centered at cx: gilt lines ending in small diamonds, with a
// larger diamond in the middle.
func flourish(dst *gfx.Image, cx, y, half int) {
	fx, fy := float32(cx), float32(y)
	gfx.FillRect(dst, fx-float32(half), fy+1, float32(2*half), 1, ink)
	gfx.FillRect(dst, fx-float32(half), fy, float32(2*half), 1, gilt)
	gem(dst, cx-2, y-2, gilt)
	for _, s := range []int{-1, 1} {
		gem(dst, cx+s*half-2+s*2, y-2, giltDark)
	}
}

// gildedText renders text from the font sheet in gilt, shaded from light at the top to deep at
// the bottom, with a one-pixel dark outline and a shadow below and right.
func gildedText(sheet image.Image, s string) image.Image {
	// Lay the glyphs out like bitmapFont does.
	type glyph struct{ sx, sy, w int }
	var gs []glyph
	width := 0
	for _, r := range s {
		if r < firstGlyph || r > lastGlyph {
			r = '?'
		}
		i := int(r - firstGlyph)
		gx, gy := (i%glyphsPerRow)*glyphSize, (i/glyphsPerRow)*glyphSize
		left, right := glyphSize, -1
		for x := 0; x < glyphSize; x++ {
			for y := 0; y < glyphSize; y++ {
				if _, _, _, a := sheet.At(gx+x, gy+y).RGBA(); a > 0x8000 {
					left, right = min(left, x), max(right, x)
				}
			}
		}
		w := spaceWidth
		if right >= 0 {
			w = right - left + 1
		}
		gs = append(gs, glyph{sx: gx + max(left, 0), sy: gy, w: w})
		width += w + 2
	}
	on := func(x, y int) bool { _, _, _, a := sheet.At(x, y).RGBA(); return a > 0x8000 }
	const pad = 2
	mask := make([]bool, (width+2*pad)*(glyphSize+2*pad))
	mw := width + 2*pad
	x := pad
	for _, g := range gs {
		for dx := 0; dx < g.w; dx++ {
			for dy := 0; dy < glyphSize; dy++ {
				if on(g.sx+dx, g.sy+dy) {
					mask[(dy+pad)*mw+x+dx] = true
					// Bold: strokes one pixel wider, except where that would close a
					// one-pixel gap (the middle of a w).
					if !(dx+2 < g.w && on(g.sx+dx+2, g.sy+dy)) {
						mask[(dy+pad)*mw+x+dx+1] = true
					}
				}
			}
		}
		x += g.w + 2
	}
	at := func(x, y int) bool { return x >= 0 && y >= 0 && x < mw && y < glyphSize+2*pad && mask[y*mw+x] }
	img := image.NewRGBA(image.Rect(0, 0, mw, glyphSize+2*pad))
	shadow, outline := color.RGBA{30, 18, 10, 200}, color.RGBA{50, 30, 18, 255}
	for y := 0; y < glyphSize+2*pad; y++ {
		for x := 0; x < mw; x++ {
			switch {
			case at(x, y):
				t := float64(y-pad) / float64(glyphSize-1)
				img.Set(x, y, lerpRGBA(giltLit, giltDark, t))
			case at(x-1, y) || at(x+1, y) || at(x, y-1) || at(x, y+1):
				img.Set(x, y, outline)
			case at(x-2, y-1) || at(x-1, y-2) || at(x-2, y-2):
				img.Set(x, y, shadow)
			}
		}
	}
	return img
}

func lerpRGBA(a, b color.RGBA, t float64) color.RGBA {
	t = max(0, min(1, t))
	l := func(x, y uint8) uint8 { return uint8(float64(x) + (float64(y)-float64(x))*t) }
	return color.RGBA{l(a.R, b.R), l(a.G, b.G), l(a.B, b.B), 255}
}

// vignette is a dark frame fading in toward the screen's edges.
func vignette(w, h int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			dx, dy := (float64(x)/float64(w)-0.5)*2, (float64(y)/float64(h)-0.5)*2
			d := math.Sqrt(dx*dx*0.8 + dy*dy)
			a := math.Pow(max(0, min(1, (d-0.55)/0.75)), 1.5) * 190
			img.Set(x, y, color.RGBA{0, 0, 0, uint8(a)}) // black at alpha a (premultiplied, so RGB stays 0)
		}
	}
	return img
}

// drawMotes draws glowing motes drifting up over the backdrop, swaying and flickering;
// positions are in canvas pixels, drawn scale times as large.
func drawMotes(dst *gfx.Image, frame, vw, vh int, scale float32) {
	t := float64(frame) / 60
	for i := 0; i < 55; i++ {
		h := uint32(i)*2654435761 ^ 0x9e3779b9
		speed := 5 + float64(h%9)
		x := float64(h>>8%uint32(vw)) + 4*math.Sin(t*0.7+float64(i))
		y := float64(vh) - math.Mod(t*speed+float64(h>>16%uint32(vh+20)), float64(vh+20)) + 10
		glow := 0.5 + 0.5*math.Sin(t*2.3+float64(i)*1.7)
		c := color.NRGBA{255, 225, 140, uint8(120 + 135*glow)}
		size := scale
		if h%4 == 0 {
			size = 2 * scale
		}
		gfx.FillRect(dst, float32(x)*scale, float32(y)*scale, size, size, c)
	}
}

// drawAtmosphere draws the backdrop's warm tint, vignette and motes straight onto the screen,
// over the 3D world and under the canvas: translucent colors drawn into the transparent canvas
// would have their alpha squared and all but vanish. scale is screen pixels per canvas pixel,
// vw and vh the canvas size.
func (t *titleScreen) drawAtmosphere(scale, vw, vh int) {
	screen := gfx.Screen()
	sw, sh := float32(vw*scale), float32(vh*scale)
	gfx.FillRect(screen, 0, 0, sw, sh, color.NRGBA{90, 50, 10, 50}) // golden hour
	if t.vignette == nil || t.vignette.Bounds().Dx() != vw || t.vignette.Bounds().Dy() != vh {
		if t.vignette != nil {
			t.vignette.Deallocate()
		}
		t.vignette = gfx.NewImageFromImage(vignette(vw, vh))
	}
	op := &gfx.DrawImageOptions{Filter: gfx.FilterLinear}
	op.GeoM.Scale(float64(scale), float64(scale))
	screen.DrawImage(t.vignette, op)
	drawMotes(screen, t.g.frame, vw, vh, float32(scale))
	gfx.Flush()
}

// drawDressing draws the gilded logo; the flourish and subtitle only on the main page, where
// there's room under it.
func (t *titleScreen) drawDressing(dst *gfx.Image, vw, vh int, subtitle bool) {
	g := t.g
	const zoom = 3
	if t.logo == nil {
		sheet, err := assets.LoadImage(g.opts.AssetsDir, assets.Font)
		if err != nil {
			return
		}
		t.logo = gfx.NewImageFromImage(gildedText(sheet, "Oldworld2"))
	}
	lw, lh := t.logo.Bounds().Dx()*zoom, t.logo.Bounds().Dy()*zoom
	op := &gfx.DrawImageOptions{}
	op.GeoM.Scale(zoom, zoom)
	op.GeoM.Translate(float64((vw-lw)/2), 16)
	dst.DrawImage(t.logo, op)
	if subtitle {
		y := 16 + lh + 4
		flourish(dst, vw/2, y, lw/2+10)
		const sub = "Tales of the Old World"
		sw := g.font.Width(sub)
		g.font.Draw(dst, sub, (vw-sw)/2+1, y+7, color.RGBA{30, 18, 10, 200})
		g.font.Draw(dst, sub, (vw-sw)/2, y+6, color.RGBA{250, 235, 200, 255})
	}
}
