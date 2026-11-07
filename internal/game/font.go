package game

import (
	"image"
	"image/color"
	"strings"

	"oldworld2/internal/gfx"
)

// bitmapFont draws text with the 8×8 font sheet (tools/font/makefont.py): 15 glyphs per row in ASCII
// order starting at the space character. Glyphs are spaced by their actual width.
type bitmapFont struct {
	glyphs map[rune]*gfx.Image
	widths map[rune]int
}

const (
	glyphSize    = 8
	glyphsPerRow = 15
	firstGlyph   = ' '
	lastGlyph    = '~'
	spaceWidth   = 4
	LineHeight   = 10
)

func newBitmapFont(sheet image.Image) *bitmapFont {
	f := &bitmapFont{glyphs: map[rune]*gfx.Image{}, widths: map[rune]int{}}
	b := sheet.Bounds()
	// Recolor to white so text can be tinted to any color when drawn.
	white := image.NewRGBA(b)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if _, _, _, a := sheet.At(x, y).RGBA(); a > 0x8000 {
				white.Set(x, y, color.White)
			}
		}
	}
	img := gfx.NewImageFromImage(white)
	for r := firstGlyph; r <= lastGlyph; r++ {
		i := int(r - firstGlyph)
		gx, gy := (i%glyphsPerRow)*glyphSize, (i/glyphsPerRow)*glyphSize
		left, right := glyphSize, -1
		for x := 0; x < glyphSize; x++ {
			for y := 0; y < glyphSize; y++ {
				if white.RGBAAt(gx+x, gy+y).A > 0 {
					left, right = min(left, x), max(right, x)
				}
			}
		}
		if right < 0 { // empty glyph (space)
			f.widths[r] = spaceWidth
			continue
		}
		f.glyphs[r] = img.SubImage(image.Rect(gx+left, gy, gx+right+1, gy+glyphSize))
		f.widths[r] = right - left + 1
	}
	return f
}

// Draw renders a line of text with its top-left corner at (x, y).
func (f *bitmapFont) Draw(dst *gfx.Image, s string, x, y int, c color.Color) {
	for _, r := range s {
		w, ok := f.widths[r]
		if !ok {
			r, w = '?', f.widths['?']
		}
		if g := f.glyphs[r]; g != nil {
			op := &gfx.DrawImageOptions{}
			op.GeoM.Translate(float64(x), float64(y))
			op.ColorScale.ScaleWithColor(c)
			dst.DrawImage(g, op)
		}
		x += w + 1
	}
}

// Width returns the width of a line of text in pixels.
func (f *bitmapFont) Width(s string) int {
	w := 0
	for _, r := range s {
		gw, ok := f.widths[r]
		if !ok {
			gw = f.widths['?']
		}
		w += gw + 1
	}
	return max(0, w-1)
}

// Wrap splits text into lines no wider than width pixels, breaking at spaces.
func (f *bitmapFont) Wrap(s string, width int) []string {
	var lines []string
	line := ""
	for _, word := range strings.Fields(s) {
		next := word
		if line != "" {
			next = line + " " + word
		}
		if line != "" && f.Width(next) > width {
			lines = append(lines, line)
			next = word
		}
		line = next
	}
	if line != "" {
		lines = append(lines, line)
	}
	return lines
}
