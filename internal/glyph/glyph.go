// Package glyph is the library of animated emoji. Every glyph draws itself
// into a paint.Painter using unit coordinates, so it looks the same at any
// terminal size.
package glyph

import (
	"math"
	"sort"
	"strings"

	"github.com/Cid-Emmerich/SopeBox/internal/paint"
)

// Anim draws one animated glyph. t is the animation clock in seconds and
// dt the time since the previous frame (0 while paused).
type Anim interface {
	Draw(p *paint.Painter, t, dt float64)
}

// Def describes a glyph and knows how to make a fresh animation for it.
type Def struct {
	Emoji   string
	Name    string
	Group   string
	Blurb   string
	Aliases []string
	New     func() Anim
}

// fn adapts a plain function into a stateless Anim.
type fn func(p *paint.Painter, t, dt float64)

func (f fn) Draw(p *paint.Painter, t, dt float64) { f(p, t, dt) }

func stateless(f func(p *paint.Painter, t, dt float64)) func() Anim {
	return func() Anim { return fn(f) }
}

// Registry lists every glyph in cycling order.
var Registry []Def

func add(group string, defs ...Def) {
	for i := range defs {
		defs[i].Group = group
		Registry = append(Registry, defs[i])
	}
}

// Groups lists the group names in order of first appearance.
func Groups() []string {
	var out []string
	seen := map[string]bool{}
	for _, d := range Registry {
		if !seen[d.Group] {
			seen[d.Group] = true
			out = append(out, d.Group)
		}
	}
	return out
}

// Find returns the index of the glyph matching q (an emoji, a name or an
// alias; partial names work) or -1.
func Find(q string) int {
	q = strings.TrimSpace(q)
	if q == "" {
		return -1
	}
	stripped := stripVariation(q)
	lq := strings.ToLower(q)
	for i, d := range Registry {
		if d.Emoji == q || stripVariation(d.Emoji) == stripped {
			return i
		}
	}
	for i, d := range Registry {
		if d.Name == lq {
			return i
		}
		for _, a := range d.Aliases {
			if a == lq {
				return i
			}
		}
	}
	if m := Search(lq); len(m) > 0 {
		return m[0]
	}
	return -1
}

// Search returns glyph indexes whose name, aliases or group contain q,
// best matches first.
func Search(q string) []int {
	q = strings.ToLower(strings.TrimSpace(q))
	type hit struct{ i, score int }
	var hits []hit
	for i, d := range Registry {
		if q == "" {
			hits = append(hits, hit{i, 0})
			continue
		}
		s := 0
		switch {
		case d.Emoji == q || stripVariation(d.Emoji) == stripVariation(q):
			s = 100
		case d.Name == q:
			s = 90
		case strings.HasPrefix(d.Name, q):
			s = 80
		case strings.Contains(d.Name, q):
			s = 60
		}
		for _, a := range d.Aliases {
			switch {
			case a == q && s < 85:
				s = 85
			case strings.HasPrefix(a, q) && s < 70:
				s = 70
			case strings.Contains(a, q) && s < 50:
				s = 50
			}
		}
		if strings.Contains(d.Group, q) && s < 30 {
			s = 30
		}
		if strings.Contains(strings.ToLower(d.Blurb), q) && s < 20 {
			s = 20
		}
		if s > 0 {
			hits = append(hits, hit{i, s})
		}
	}
	sort.SliceStable(hits, func(a, b int) bool { return hits[a].score > hits[b].score })
	out := make([]int, len(hits))
	for i, h := range hits {
		out[i] = h.i
	}
	return out
}

func stripVariation(s string) string {
	return strings.Map(func(r rune) rune {
		if r == 0xFE0F || r == 0xFE0E || r == 0x200D {
			return -1
		}
		return r
	}, s)
}

// ---------------------------------------------------------------------------
// Shared animation helpers

func lerp(a, b, t float64) float64 { return a + (b-a)*t }

// smooth is a smoothstep from 0 to 1.
func smooth(t float64) float64 {
	t = paint.Clamp01(t)
	return t * t * (3 - 2*t)
}

// cycle returns where t sits inside a repeating period, 0..1.
func cycle(t, period float64) float64 {
	return math.Mod(t, period) / period
}

// pop is a springy scale factor for x seconds after an event: 1 → ~1.35 → 1.
func pop(x float64) float64 {
	if x < 0 {
		return 1
	}
	return 1 + 0.5*math.Exp(-4.5*x)*math.Sin(9*x)
}

// bounceIn eases from 0 to 1 with an overshoot.
func bounceIn(t float64) float64 {
	if t <= 0 {
		return 0
	}
	if t >= 1 {
		return 1
	}
	return 1 - math.Pow(1-t, 2)*math.Cos(t*math.Pi*1.5)*1.0 - math.Pow(1-t, 2)*0
}

// blink returns eye openness 0..1: mostly open with a quick blink every
// period seconds (offset by seed so several eyes don't sync).
func blink(t, period, seed float64) float64 {
	x := math.Mod(t+seed*period, period)
	const d = 0.16
	if x > d {
		return 1
	}
	return math.Abs(math.Cos(x / d * math.Pi))
}

// wobble is a slow organic drift built from two sines.
func wobble(t, seed float64) float64 {
	return 0.6*math.Sin(t*1.3+seed) + 0.4*math.Sin(t*2.9+seed*1.7)
}

// noise is a cheap deterministic hash → 0..1.
func noise(i int) float64 {
	x := uint32(i)*2654435761 + 0x9E3779B9
	x ^= x >> 15
	x *= 0x2C1B3C6D
	x ^= x >> 12
	return float64(x%10000) / 10000
}

// stars twinkles n stars across the visible region behind the glyph.
func stars(p *paint.Painter, t float64, n int, seed int) {
	w, h := p.R-p.L, p.Bt-p.T
	for i := 0; i < n; i++ {
		x := p.L + w*noise(seed+i*7)
		y := p.T + h*noise(seed+i*7+3)
		ph := noise(seed+i*7+5) * 6.28
		b := 0.5 + 0.5*math.Sin(t*(1.5+noise(seed+i)*2)+ph)
		c := paint.Mix(paint.Hex(0x30364A), paint.White, b)
		p.Dot(x, y, 0.012, c)
		if b > 0.85 {
			p.Line(x-0.02, y, x+0.02, y, 0.005, c)
			p.Line(x, y-0.02, x, y+0.02, 0.005, c)
		}
	}
}

// ground draws a dashed scrolling floor line across the visible width.
func ground(p *paint.Painter, y, speed, t float64, c paint.RGB) {
	const seg = 0.12
	off := math.Mod(t*speed, seg*2)
	for x := p.L - seg*2 - off; x < p.R; x += seg * 2 {
		p.Line(x, y, x+seg, y, 0.012, c)
	}
}

// figure is a simple jointed stick person.
type figure struct {
	head, torso, limb        float64 // sizes
	skin, shirt, pants, hair paint.RGB
}

func defaultFigure() figure {
	return figure{head: 0.07, torso: 0.22, limb: 0.15, skin: paint.Skin, shirt: paint.Blue, pants: paint.Hex(0x37474F), hair: paint.Brown}
}

// draw renders the figure. hip is the hip joint; lean tilts the torso
// (radians, positive leans forward/right). Limb angles are radians from
// straight down for legs and from straight down for arms; bend adds a
// knee/elbow fold.
func (f figure) draw(p *paint.Painter, hipX, hipY, lean float64, legs, knees, arms, elbows [2]float64) {
	th := f.limb * 0.42
	// torso
	sx := hipX + math.Sin(lean)*f.torso
	sy := hipY - math.Cos(lean)*f.torso
	// legs first so torso overlaps them
	for i := 0; i < 2; i++ {
		a := legs[i]
		kx, ky := hipX+math.Sin(a)*f.limb, hipY+math.Cos(a)*f.limb
		b := a + knees[i]
		fx, fy := kx+math.Sin(b)*f.limb, ky+math.Cos(b)*f.limb
		p.Line(hipX, hipY, kx, ky, th, f.pants)
		p.Line(kx, ky, fx, fy, th, f.pants)
		p.Circle(fx, fy+th*0.2, th*0.75, paint.DarkGrey)
	}
	p.Line(hipX, hipY, sx, sy, th*1.6, f.shirt)
	// arms from the shoulder
	for i := 0; i < 2; i++ {
		a := arms[i]
		ex, ey := sx+math.Sin(a)*f.limb, sy+math.Cos(a)*f.limb
		b := a + elbows[i]
		hx, hy := ex+math.Sin(b)*f.limb, ey+math.Cos(b)*f.limb
		p.Line(sx, sy, ex, ey, th*0.9, f.shirt)
		p.Line(ex, ey, hx, hy, th*0.8, f.skin)
	}
	// head
	hx := sx + math.Sin(lean)*(f.head+th*0.4)
	hy := sy - math.Cos(lean)*(f.head+th*0.4)
	p.Circle(hx, hy, f.head, f.skin)
	p.Arc(hx, hy, f.head*0.8, math.Pi+0.2, 2*math.Pi-0.2, f.head*0.5, f.hair)
}

func init() {
	initdrinks()
	initpeople()
	initflags()
	initmath()
	initbuttons()
	initnature()
	initobjects()
	initanimals()
	initfaces()
}
