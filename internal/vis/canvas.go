// Package vis contains SopeBox's visualizers. Each style draws into a
// Canvas of coloured cells which the UI then blits to the terminal.
package vis

import (
	"math"

	"github.com/Cid-Emmerich/SopeBox/internal/paint"
	"github.com/Cid-Emmerich/SopeBox/internal/theme"
)

// Canvas is a W x H grid of cells.
type Canvas = paint.Canvas

// NewCanvas allocates an empty canvas.
func NewCanvas(w, h int) *Canvas { return paint.NewCanvas(w, h) }

// Text writes a string starting at x,y.
func Text(c *Canvas, x, y int, s string, fg paint.RGB) {
	for i, r := range []rune(s) {
		c.Set(x+i, y, r, fg)
	}
}

// ---------------------------------------------------------------------------
// Braille dot plotting: 2 x 4 dots per cell for fine-grained curves.

// Dots is a sub-cell resolution bitmap rendered with braille characters.
type Dots struct {
	W, H  int // in dots
	cw    int // width in cells
	ch    int
	bits  []uint8
	color []paint.RGB
	set   []bool
}

// NewDots creates a dot grid covering w x h cells.
func NewDots(w, h int) *Dots {
	d := &Dots{W: w * 2, H: h * 4, cw: w, ch: h}
	d.bits = make([]uint8, w*h)
	d.color = make([]paint.RGB, w*h)
	d.set = make([]bool, w*h)
	return d
}

var dotBits = [4][2]uint8{{0x01, 0x08}, {0x02, 0x10}, {0x04, 0x20}, {0x40, 0x80}}

// Plot lights a dot at dot-coordinates x,y.
func (d *Dots) Plot(x, y int, col paint.RGB) {
	if x < 0 || y < 0 || x >= d.W || y >= d.H {
		return
	}
	cx, cy := x/2, y/4
	i := cy*d.cw + cx
	d.bits[i] |= dotBits[y%4][x%2]
	if d.set[i] {
		d.color[i] = paint.Mix(d.color[i], col, 0.5)
	} else {
		d.color[i] = col
	}
	d.set[i] = true
}

// Line draws a dot line between two points.
func (d *Dots) Line(x0, y0, x1, y1 int, col paint.RGB) {
	dx := abs(x1 - x0)
	dy := -abs(y1 - y0)
	sx, sy := 1, 1
	if x0 > x1 {
		sx = -1
	}
	if y0 > y1 {
		sy = -1
	}
	err := dx + dy
	for {
		d.Plot(x0, y0, col)
		if x0 == x1 && y0 == y1 {
			return
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x0 += sx
		}
		if e2 <= dx {
			err += dx
			y0 += sy
		}
	}
}

// Circle draws a dotted circle outline.
func (d *Dots) Circle(cx, cy, r float64, col paint.RGB) {
	steps := int(2*math.Pi*r) + 8
	for i := 0; i < steps; i++ {
		a := 2 * math.Pi * float64(i) / float64(steps)
		d.Plot(int(cx+math.Cos(a)*r), int(cy+math.Sin(a)*r), col)
	}
}

// Flush writes the dots into the canvas at offset ox, oy. Existing cells
// are only overwritten where dots were set.
func (d *Dots) Flush(c *Canvas, ox, oy int) {
	for y := 0; y < d.ch; y++ {
		for x := 0; x < d.cw; x++ {
			i := y*d.cw + x
			if !d.set[i] {
				continue
			}
			c.Set(ox+x, oy+y, rune(0x2800+int(d.bits[i])), d.color[i])
		}
	}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// ---------------------------------------------------------------------------
// Gradients

// GradientNames lists gradient styles for cycling. "theme" shades every
// orb in its own voice colour; the others recolour by height/angle.
var GradientNames = []string{"theme", "horizontal", "rainbow", "spectrum", "fire", "ice", "neon", "heat", "mono", "pastel", "matrix"}

// Gradient returns the colour for a point: v is intensity (0 low, 1
// high), u is angular/horizontal fraction, t is time.
func Gradient(name string, v, u, t float64, th theme.Theme) paint.RGB {
	return paint.Gradient(name, v, u, t, th.Palette())
}

// voiceColour shades an orb's own colour by intensity v, or applies the
// chosen gradient blended with the voice colour so orbs stay distinct.
func voiceColour(f *Frame, base paint.RGB, v, u float64) paint.RGB {
	v = clamp01(v)
	switch f.Opts.Gradient {
	case "theme":
		dim := paint.Mix(base, paint.RGB{R: 20, G: 20, B: 28}, 0.55)
		bright := paint.Mix(base, paint.White, 0.35)
		return paint.Mix3(dim, base, bright, v)
	case "mono":
		return paint.Mix(paint.Mix(base, paint.RGB{}, 0.5), base, v)
	default:
		g := Gradient(f.Opts.Gradient, v, u, f.Time, f.Theme)
		return paint.Mix(base, g, 0.55)
	}
}

func clamp01(x float64) float64 {
	if x < 0 || math.IsNaN(x) {
		return 0
	}
	if x > 1 {
		return 1
	}
	return x
}
