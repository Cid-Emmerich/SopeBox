package vis

import (
	"fmt"
	"image"
	"strings"

	"github.com/Cid-Emmerich/SopeBox/internal/art"
	"github.com/Cid-Emmerich/SopeBox/internal/paint"
)

// CollageTile is one person, place or thing on the collage.
type CollageTile struct {
	Key     string
	Name    string
	Kind    string // person, place, event, …
	Desc    string // Wikipedia's one-line description
	Img     image.Image
	Ago     float64 // seconds since it was last mentioned
	Loading bool    // its picture is still being fetched
}

// Placement asks the terminal to paint a picture over a block of canvas
// cells (Kitty graphics). The canvas leaves those cells blank.
type Placement struct {
	Key        string
	Img        image.Image
	X, Y, W, H int
}

// Collage is what the UI hands the collage style each frame.
type Collage struct {
	Tiles  []CollageTile // most recently mentioned first
	Status string        // headline when there is nothing to show
	Detail string        // a second line under the status
	Footer string        // small print at the bottom (progress, counts)
	Kitty  bool          // pictures are painted by the terminal
	Place  []Placement   // filled in by Draw when Kitty is set
}

// CollageStyle shows pictures of who and what the conversation is about,
// the newest one large and a few earlier ones beside it.
type CollageStyle struct {
	cache map[string][][]paint.Cell
}

func (*CollageStyle) Name() string     { return "collage" }
func (*CollageStyle) Describe() string { return "pictures of the people and places being talked about" }

// collageMax is how many tiles fit on screen at once.
const collageMax = 5

type rect struct{ x, y, w, h int }

// collageLayout splits a w x h area into n tiles: the first (newest) large,
// the rest in a column or grid beside it, or in a row below it on narrow
// screens.
func collageLayout(n, w, h int) []rect {
	if n <= 0 || w < 8 || h < 4 {
		return nil
	}
	if n > collageMax {
		n = collageMax
	}
	if n == 1 {
		return []rect{{0, 0, w, h}}
	}
	rest := n - 1
	// cells are about twice as tall as wide; compare in square units
	if w < 60 || float64(w) < float64(h)*2*1.1 {
		heroH := h * 3 / 5
		out := []rect{{0, 0, w, heroH}}
		cols := min(rest, 3)
		return append(out, grid(rect{0, heroH, w, h - heroH}, cols, cols, 1)...)
	}
	heroW := w * 3 / 5
	side := rect{heroW, 0, w - heroW, h}
	out := []rect{{0, 0, heroW, h}}
	if rest <= 3 {
		return append(out, grid(side, rest, 1, rest)...)
	}
	return append(out, grid(side, rest, 2, 2)...)
}

// grid cuts r into cols x rows cells and returns the first n.
func grid(r rect, n, cols, rows int) []rect {
	var out []rect
	for j := 0; j < rows; j++ {
		for i := 0; i < cols && len(out) < n; i++ {
			x0 := r.x + r.w*i/cols
			x1 := r.x + r.w*(i+1)/cols
			y0 := r.y + r.h*j/rows
			y1 := r.y + r.h*(j+1)/rows
			out = append(out, rect{x0, y0, x1 - x0, y1 - y0})
		}
	}
	return out
}

// fitCells sizes an image to fit inside w x h cells, keeping its shape.
func fitCells(img image.Image, w, h int) (int, int) {
	if img == nil {
		return 0, 0
	}
	return art.Fit(img, w, h)
}

func (s *CollageStyle) Draw(c *Canvas, f *Frame) {
	col := f.Collage
	if col == nil {
		col = &Collage{Status: "nothing to show"}
	}
	col.Place = col.Place[:0]
	if s.cache == nil || len(s.cache) > 64 {
		s.cache = map[string][][]paint.Cell{}
	}
	footerH := 0
	if col.Footer != "" && c.H > 8 {
		footerH = 1
		Text(c, max(0, c.W-len([]rune(col.Footer))-1), c.H-1, col.Footer, f.Theme.Muted)
	}
	tiles := col.Tiles
	rects := collageLayout(len(tiles), c.W, c.H-footerH)
	if len(rects) == 0 {
		s.drawStatus(c, f, col)
		return
	}
	for i, r := range rects {
		s.drawTile(c, f, col, tiles[i], r, i == 0)
	}
}

func (s *CollageStyle) drawStatus(c *Canvas, f *Frame, col *Collage) {
	lines := []string{col.Status}
	if col.Detail != "" {
		lines = append(lines, "", col.Detail)
	}
	var wrapped []string
	for _, l := range lines {
		wrapped = append(wrapped, wrapText(l, max(10, c.W-6))...)
	}
	y := (c.H - len(wrapped)) / 2
	for i, l := range wrapped {
		colr := f.Theme.Muted
		if i == 0 {
			colr = f.Theme.Accent
		}
		Text(c, (c.W-len([]rune(l)))/2, y+i, l, colr)
	}
}

func (s *CollageStyle) drawTile(c *Canvas, f *Frame, col *Collage, t CollageTile, r rect, hero bool) {
	th := f.Theme
	// a newly mentioned tile's frame glows, then settles
	border := paint.Mix(th.Muted, th.Select, 0.5)
	if hero {
		border = paint.Mix(th.Accent, th.Muted, clamp01(t.Ago/12))
	}
	box(c, r, border)
	in := rect{r.x + 2, r.y + 1, r.w - 4, r.h - 2}
	if in.w < 4 || in.h < 2 {
		return
	}
	// caption lines under the picture
	var caps []capLine
	caps = append(caps, capLine{fitStr(t.Name, in.w), paint.Mix(th.Text, paint.White, 0.2)})
	if t.Desc != "" && in.h >= 5 {
		descLines := 1
		if hero {
			descLines = 2
		}
		for i, l := range wrapText(t.Desc, in.w) {
			if i >= descLines {
				break
			}
			caps = append(caps, capLine{l, th.Muted})
		}
	}
	if hero && in.h >= 8 {
		caps = append(caps, capLine{agoText(t.Ago), paint.Mix(th.Muted, th.Select, 0.4)})
	}
	if len(caps) >= in.h {
		caps = caps[:max(1, in.h-1)]
	}
	imgH := in.h - len(caps)
	cy := in.y + imgH
	for i, cl := range caps {
		Text(c, in.x+(in.w-len([]rune(cl.text)))/2, cy+i, cl.text, cl.col)
	}
	if imgH < 2 {
		return
	}
	area := rect{in.x, in.y, in.w, imgH}
	if t.Img == nil {
		s.drawCard(c, f, t, area, hero)
		return
	}
	iw, ih := fitCells(t.Img, area.w, area.h)
	ix := area.x + (area.w-iw)/2
	iy := area.y + (area.h-ih)/2
	if col.Kitty {
		col.Place = append(col.Place, Placement{Key: t.Key, Img: t.Img, X: ix, Y: iy, W: iw, H: ih})
		return
	}
	ck := fmt.Sprintf("%s|%d|%d", t.Key, iw, ih)
	cells, ok := s.cache[ck]
	if !ok {
		cells = art.Blocks(t.Img, iw, ih)
		s.cache[ck] = cells
	}
	for y, row := range cells {
		for x, cell := range row {
			if ix+x >= 0 && ix+x < c.W && iy+y >= 0 && iy+y < c.H {
				c.Cells[(iy+y)*c.W+ix+x] = cell
			}
		}
	}
}

// drawCard fills a tile that has no picture with its kind as a big glyph.
func (s *CollageStyle) drawCard(c *Canvas, f *Frame, t CollageTile, r rect, hero bool) {
	th := f.Theme
	glyph := map[string]string{"person": "☺", "place": "⌖", "event": "⚔", "group": "⚑", "work": "❧", "thing": "◆", "era": "⌛"}[t.Kind]
	if glyph == "" {
		glyph = "◆"
	}
	lines := []string{glyph, t.Kind}
	if t.Loading {
		lines = []string{"…", "finding a picture"}
	}
	y := r.y + (r.h-len(lines))/2
	for i, l := range lines {
		colr := th.Muted
		if i == 0 {
			colr = th.Secondary
		}
		Text(c, r.x+(r.w-len([]rune(l)))/2, y+i, l, colr)
	}
}

type capLine struct {
	text string
	col  paint.RGB
}

// box draws a rounded frame around r.
func box(c *Canvas, r rect, col paint.RGB) {
	x1, y1 := r.x+r.w-1, r.y+r.h-1
	for x := r.x + 1; x < x1; x++ {
		c.Set(x, r.y, '─', col)
		c.Set(x, y1, '─', col)
	}
	for y := r.y + 1; y < y1; y++ {
		c.Set(r.x, y, '│', col)
		c.Set(x1, y, '│', col)
	}
	c.Set(r.x, r.y, '╭', col)
	c.Set(x1, r.y, '╮', col)
	c.Set(r.x, y1, '╰', col)
	c.Set(x1, y1, '╯', col)
}

func agoText(s float64) string {
	switch {
	case s < 5:
		return "just mentioned"
	case s < 60:
		return fmt.Sprintf("mentioned %ds ago", int(s))
	default:
		return fmt.Sprintf("mentioned %dm ago", int(s/60))
	}
}

func fitStr(s string, w int) string {
	r := []rune(s)
	if len(r) <= w {
		return s
	}
	if w <= 1 {
		return string(r[:max(0, w)])
	}
	return string(r[:w-1]) + "…"
}

// wrapText breaks s into lines of at most w runes at spaces.
func wrapText(s string, w int) []string {
	var out []string
	line := ""
	for _, word := range strings.Fields(s) {
		switch {
		case line == "":
			line = fitStr(word, w)
		case len([]rune(line))+1+len([]rune(word)) <= w:
			line += " " + word
		default:
			out = append(out, line)
			line = fitStr(word, w)
		}
	}
	if line != "" {
		out = append(out, line)
	}
	return out
}
