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
	icons  map[rune]*gfx.Image // drawn in their own colors, not tinted
}

// Coin icons, written into text like characters (formatMoney).
const (
	coinBronze = '\uE000'
	coinSilver = '\uE001'
	coinGold   = '\uE002'
)

const (
	glyphSize    = 8
	glyphsPerRow = 15
	firstGlyph   = ' '
	lastGlyph    = '~'
	spaceWidth   = 4
	LineHeight   = 10
)

func newBitmapFont(sheet image.Image) *bitmapFont {
	f := &bitmapFont{glyphs: map[rune]*gfx.Image{}, widths: map[rune]int{}, icons: map[rune]*gfx.Image{}}
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
	for r, c := range map[rune][3]color.RGBA{
		coinBronze: {{205, 127, 70, 255}, {120, 64, 30, 255}, {245, 190, 140, 255}},
		coinSilver: {{200, 204, 212, 255}, {110, 114, 126, 255}, {250, 250, 255, 255}},
		coinGold:   {{240, 196, 60, 255}, {150, 100, 20, 255}, {255, 240, 160, 255}},
	} {
		f.icons[r] = gfx.NewImageFromImage(coinIcon(c[0], c[1], c[2]))
		f.widths[r] = coinSize
	}
	return f
}

const coinSize = 7

// coinIcon draws a round coin on the text's 8-pixel line: a darker rim, the face, a bright
// glint at the top left and a shaded lower right, so it reads as metal even this small.
func coinIcon(face, rim, glint color.RGBA) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, coinSize, glyphSize))
	shade := color.RGBA{uint8(int(face.R) * 3 / 4), uint8(int(face.G) * 3 / 4), uint8(int(face.B) * 3 / 4), 255}
	const c = (coinSize - 1) / 2.0
	for y := 0; y < coinSize; y++ {
		for x := 0; x < coinSize; x++ {
			dx, dy := float64(x)-c, float64(y)-c
			switch d := dx*dx + dy*dy; {
			case d > 3.6*3.6-2:
			case d > 2.3*2.3:
				img.Set(x, y+1, rim)
			case dx+dy <= -2:
				img.Set(x, y+1, glint)
			case dx+dy >= 2:
				img.Set(x, y+1, shade)
			default:
				img.Set(x, y+1, face)
			}
		}
	}
	return img
}

// Draw renders a line of text with its top-left corner at (x, y).
func (f *bitmapFont) Draw(dst *gfx.Image, s string, x, y int, c color.Color) {
	for _, r := range s {
		w, ok := f.widths[r]
		if !ok {
			r, w = '?', f.widths['?']
		}
		if icon := f.icons[r]; icon != nil {
			op := &gfx.DrawImageOptions{}
			op.GeoM.Translate(float64(x), float64(y))
			dst.DrawImage(icon, op)
		} else if g := f.glyphs[r]; g != nil {
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
