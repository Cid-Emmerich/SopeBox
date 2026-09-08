package captions

import (
	"github.com/Cid-Emmerich/SopeBox/internal/glyph"
	"github.com/Cid-Emmerich/SopeBox/internal/paint"
)

// Emoji animates the glyph most recently triggered by a spoken word.
type Emoji struct {
	idx    int
	anim   glyph.Anim
	word   string
	t      float64
	shown  float64 // seconds the current glyph has been on screen
	queue  []queued
	bitmap *paint.Bitmap
	canvas *paint.Canvas
}

type queued struct {
	idx  int
	word string
}

const (
	minShow = 1.4 // seconds before the next queued glyph may replace this one
	maxShow = 4.0 // seconds a glyph stays with nothing new
)

// NewEmoji creates an idle animator.
func NewEmoji() *Emoji { return &Emoji{idx: -1} }

// Fire requests a glyph for a word. The current glyph keeps its minimum
// screen time; later requests queue (only the newest two are kept).
func (e *Emoji) Fire(idx int, word string) {
	if idx < 0 || idx >= len(glyph.Registry) {
		return
	}
	if e.idx == idx && e.shown < maxShow {
		e.shown = 0 // same word again: just extend
		return
	}
	if e.idx < 0 || e.shown >= minShow {
		e.set(idx, word)
		return
	}
	e.queue = append(e.queue, queued{idx, word})
	if len(e.queue) > 2 {
		e.queue = e.queue[len(e.queue)-2:]
	}
}

func (e *Emoji) set(idx int, word string) {
	e.idx = idx
	e.anim = glyph.Registry[idx].New()
	e.word = word
	e.t = 0
	e.shown = 0
}

// Clear removes the current glyph.
func (e *Emoji) Clear() {
	e.idx = -1
	e.anim = nil
	e.queue = nil
}

// Active reports whether a glyph is on screen.
func (e *Emoji) Active() bool { return e.idx >= 0 && e.anim != nil }

// Word is the word that fired the current glyph.
func (e *Emoji) Word() string { return e.word }

// Def returns the current glyph definition.
func (e *Emoji) Def() *glyph.Def {
	if !e.Active() {
		return nil
	}
	return &glyph.Registry[e.idx]
}

// Step advances the animation clock.
func (e *Emoji) Step(dt float64) {
	if !e.Active() {
		return
	}
	e.t += dt
	e.shown += dt
	if len(e.queue) > 0 && e.shown >= minShow {
		q := e.queue[0]
		e.queue = e.queue[1:]
		e.set(q.idx, q.word)
		return
	}
	if e.shown >= maxShow {
		e.Clear()
	}
}

// Render draws the current glyph into w x h cells.
func (e *Emoji) Render(w, h int, style, colour string, pal paint.Palette, paused bool) *paint.Canvas {
	if !e.Active() || w <= 0 || h <= 0 {
		return nil
	}
	bw, bh := paint.BufferSize(style, w, h)
	if e.bitmap == nil || e.bitmap.W != bw || e.bitmap.H != bh {
		e.bitmap = paint.NewBitmap(bw, bh)
	}
	if e.canvas == nil || e.canvas.W != w || e.canvas.H != h {
		e.canvas = paint.NewCanvas(w, h)
	}
	e.bitmap.Clear()
	p := paint.NewPainter(e.bitmap, 1, false)
	dt := 1.0 / 30
	if paused {
		dt = 0
	}
	// pop in
	zoom := 1.0
	if e.t < 0.5 {
		x := e.t / 0.5
		zoom = x * (1 + 0.35*(1-x))
		if zoom < 0.05 {
			zoom = 0.05
		}
	}
	if zoom != 1 {
		p = p.Sub(0.5-zoom/2, 0.5-zoom/2, zoom, zoom)
	}
	e.anim.Draw(p, e.t, dt)
	paint.Render(e.bitmap, e.canvas, paint.RenderOpts{Style: style, Charset: "standard", Colour: colour, Palette: pal, Time: e.t})
	return e.canvas
}
