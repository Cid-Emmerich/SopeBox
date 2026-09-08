package paint

import "math"

// Cell is one terminal cell of rendered output.
type Cell struct {
	Ch     rune
	Fg, Bg RGB
	HasBg  bool
}

// Canvas is a W x H grid of cells.
type Canvas struct {
	W, H  int
	Cells []Cell
}

// NewCanvas allocates an empty canvas.
func NewCanvas(w, h int) *Canvas {
	if w < 0 {
		w = 0
	}
	if h < 0 {
		h = 0
	}
	return &Canvas{W: w, H: h, Cells: make([]Cell, w*h)}
}

// Clear blanks the canvas.
func (c *Canvas) Clear() {
	for i := range c.Cells {
		c.Cells[i] = Cell{}
	}
}

// Set writes a glyph with a foreground colour.
func (c *Canvas) Set(x, y int, ch rune, fg RGB) {
	if x < 0 || y < 0 || x >= c.W || y >= c.H {
		return
	}
	c.Cells[y*c.W+x] = Cell{Ch: ch, Fg: fg}
}

// SetBg writes a glyph with both colours.
func (c *Canvas) SetBg(x, y int, ch rune, fg, bg RGB) {
	if x < 0 || y < 0 || x >= c.W || y >= c.H {
		return
	}
	c.Cells[y*c.W+x] = Cell{Ch: ch, Fg: fg, Bg: bg, HasBg: true}
}

// Get returns a cell (zero cell if out of range).
func (c *Canvas) Get(x, y int) Cell {
	if x < 0 || y < 0 || x >= c.W || y >= c.H {
		return Cell{}
	}
	return c.Cells[y*c.W+x]
}

// StyleNames lists render styles in cycling order.
var StyleNames = []string{"blocks", "braille", "ascii", "chunky"}

// Charsets available for the ascii style (dark -> light).
var Charsets = map[string]string{
	"standard": " .:-=+*#%@",
	"detailed": " .'`^\",:;Il!i><~+_-?][}{1)(|\\/tfjrxnuvczXYUJCLQ0OZmwqpdbkhao*#MW&8%B@$",
	"blocks":   " ░▒▓█",
	"minimal":  " .oO@",
	"dots":     " ·∙•●",
	"lines":    " -=≡",
	"binary":   " 01",
	"hearts":   " ·♡♥",
}

// CharsetNames lists ascii charsets in cycling order.
var CharsetNames = []string{"standard", "detailed", "blocks", "minimal", "dots", "lines", "binary", "hearts"}

// RenderOpts control how a bitmap is turned into cells.
type RenderOpts struct {
	Style   string
	Charset string
	Colour  string
	Palette Palette
	Time    float64
}

// BufferSize returns the pixel size a bitmap must have to render into
// w x h cells with the given style (pixels are square in every style).
func BufferSize(style string, w, h int) (int, int) {
	switch style {
	case "braille", "ascii":
		return w * 2, h * 4
	default:
		return w, h * 2
	}
}

// colourise applies the colour mode to a pixel at pixel coords.
func colourise(c RGB, x, y float64, b *Bitmap, o *RenderOpts) RGB {
	if o.Colour == "emoji" || o.Colour == "" {
		return c
	}
	v := 1 - y/float64(max(b.H, 1))
	u := x / float64(max(b.W, 1))
	g := Gradient(o.Colour, v, u, o.Time, o.Palette)
	lum := Luminance(c)
	return g.Scale(0.35 + 0.75*lum)
}

// Render rasterises b into cv.
func Render(b *Bitmap, cv *Canvas, o RenderOpts) {
	cv.Clear()
	switch o.Style {
	case "braille":
		renderBraille(b, cv, &o)
	case "ascii":
		renderASCII(b, cv, &o)
	case "chunky":
		renderChunky(b, cv, &o)
	default:
		renderBlocks(b, cv, &o)
	}
}

func renderBlocks(b *Bitmap, cv *Canvas, o *RenderOpts) {
	for y := 0; y < cv.H; y++ {
		for x := 0; x < cv.W; x++ {
			top := b.Get(x, 2*y)
			bot := b.Get(x, 2*y+1)
			switch {
			case top.A > 0 && bot.A > 0:
				cv.SetBg(x, y, '▀', colourise(top.C, float64(x), float64(2*y), b, o), colourise(bot.C, float64(x), float64(2*y+1), b, o))
			case top.A > 0:
				cv.Set(x, y, '▀', colourise(top.C, float64(x), float64(2*y), b, o))
			case bot.A > 0:
				cv.Set(x, y, '▄', colourise(bot.C, float64(x), float64(2*y+1), b, o))
			}
		}
	}
}

func renderChunky(b *Bitmap, cv *Canvas, o *RenderOpts) {
	for y := 0; y < cv.H; y++ {
		for x := 0; x < cv.W; x++ {
			top := b.Get(x, 2*y)
			bot := b.Get(x, 2*y+1)
			var c RGB
			switch {
			case top.A > 0 && bot.A > 0:
				c = Mix(top.C, bot.C, 0.5)
			case top.A > 0:
				c = top.C
			case bot.A > 0:
				c = bot.C
			default:
				continue
			}
			cv.Set(x, y, '█', colourise(c, float64(x), float64(2*y), b, o))
		}
	}
}

// braille dot bit layout:
// (0,0)=0x01 (1,0)=0x08
// (0,1)=0x02 (1,1)=0x10
// (0,2)=0x04 (1,2)=0x20
// (0,3)=0x40 (1,3)=0x80
var dotBits = [4][2]int{{0x01, 0x08}, {0x02, 0x10}, {0x04, 0x20}, {0x40, 0x80}}

// sample averages the 2x4 pixel block for cell x,y.
func sample(b *Bitmap, x, y int) (bits, n int, avg RGB) {
	var r, g, bl int
	for dy := 0; dy < 4; dy++ {
		for dx := 0; dx < 2; dx++ {
			p := b.Get(2*x+dx, 4*y+dy)
			if p.A == 0 {
				continue
			}
			bits |= dotBits[dy][dx]
			n++
			r += int(p.C.R)
			g += int(p.C.G)
			bl += int(p.C.B)
		}
	}
	if n > 0 {
		avg = RGB{uint8(r / n), uint8(g / n), uint8(bl / n)}
	}
	return
}

func renderBraille(b *Bitmap, cv *Canvas, o *RenderOpts) {
	for y := 0; y < cv.H; y++ {
		for x := 0; x < cv.W; x++ {
			bits, n, avg := sample(b, x, y)
			if n == 0 {
				continue
			}
			cv.Set(x, y, rune(0x2800+bits), colourise(avg, float64(2*x), float64(4*y), b, o))
		}
	}
}

func renderASCII(b *Bitmap, cv *Canvas, o *RenderOpts) {
	set := []rune(Charsets[o.Charset])
	if len(set) == 0 {
		set = []rune(Charsets["standard"])
	}
	for y := 0; y < cv.H; y++ {
		for x := 0; x < cv.W; x++ {
			_, n, avg := sample(b, x, y)
			if n == 0 {
				continue
			}
			cover := float64(n) / 8
			density := cover * (0.35 + 0.65*Luminance(avg))
			idx := int(math.Ceil(density * float64(len(set)-1)))
			if idx < 1 {
				idx = 1
			}
			if idx >= len(set) {
				idx = len(set) - 1
			}
			// lift dark colours a little so dim glyphs stay legible as text
			c := Mix(avg, White, 0.15)
			cv.Set(x, y, set[idx], colourise(c, float64(2*x), float64(4*y), b, o))
		}
	}
}
