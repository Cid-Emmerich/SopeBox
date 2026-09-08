package glyph

import (
	"math"

	"github.com/Cid-Emmerich/SopeBox/internal/paint"
)

func initdrinks() {
	add("drinks",
		Def{Emoji: "☕", Name: "coffee", Blurb: "a hot cup of coffee, steaming", Aliases: []string{"hot beverage", "cup", "espresso", "latte"}, New: func() Anim { return newCup(cupCoffee) }},
		Def{Emoji: "🍵", Name: "tea", Blurb: "green tea in a bowl, steaming", Aliases: []string{"teacup", "matcha", "green tea"}, New: func() Anim { return newCup(cupTea) }},
		Def{Emoji: "🍺", Name: "beer", Blurb: "a frosty mug with rising bubbles", Aliases: []string{"beer mug", "pint", "ale"}, New: func() Anim { return &beer{sys: paint.NewSystem(7)} }},
		Def{Emoji: "🍜", Name: "ramen", Blurb: "a steaming bowl of noodles", Aliases: []string{"noodles", "soup", "bowl"}, New: func() Anim { return &ramen{steam: newSteam(11)} }},
	)
}

// ---------------------------------------------------------------------------
// steam is a reusable wisp emitter.

type steam struct{ sys *paint.System }

func newSteam(seed int64) *steam { return &steam{sys: paint.NewSystem(seed)} }

// step emits from the segment x0..x1 at y and rises. rate wisps per second.
func (s *steam) step(dt, x0, x1, y, rate float64) {
	s.sys.Every(dt, rate, func() {
		s.sys.Emit(paint.Particle{
			X: s.sys.Rand(x0, x1), Y: y,
			VX: s.sys.Rand(-0.02, 0.02), VY: s.sys.Rand(-0.22, -0.14),
			Life: s.sys.Rand(1.3, 2.0), Size: s.sys.Rand(0.02, 0.05),
			Spin: s.sys.Rand(0, 6.28), C: paint.LightGrey,
		})
	})
	s.sys.Step(dt, 0, 0.3)
	for i := range s.sys.P {
		q := &s.sys.P[i]
		q.X += math.Sin(q.Age*3.5+q.Spin) * 0.09 * dt
	}
}

func (s *steam) draw(p *paint.Painter) {
	for _, q := range s.sys.P {
		k := q.T()
		a := (1 - k) * 0.75
		p.CircleA(q.X, q.Y, q.Size*(0.6+k*1.2), q.C, a)
	}
}

// ---------------------------------------------------------------------------

type cupKind int

const (
	cupCoffee cupKind = iota
	cupTea
)

type cup struct {
	kind  cupKind
	steam *steam
}

func newCup(k cupKind) *cup { return &cup{kind: k, steam: newSteam(int64(k) + 3)} }

func (c *cup) Draw(p *paint.Painter, t, dt float64) {
	if c.kind == cupCoffee {
		c.drawCoffee(p, t, dt)
	} else {
		c.drawTea(p, t, dt)
	}
}

func (c *cup) drawCoffee(p *paint.Painter, t, dt float64) {
	body := paint.Hex(0xFAFAFA)
	shade := paint.Hex(0xD9DDE3)
	liquid := paint.Hex(0x4E2A14)
	// saucer
	p.Ellipse(0.5, 0.83, 0.36, 0.06, shade)
	p.Ellipse(0.5, 0.815, 0.30, 0.04, body)
	// cup body (tapered) with a shaded side
	p.Poly(body, 0.27, 0.42, 0.73, 0.42, 0.66, 0.80, 0.34, 0.80)
	p.Poly(shade, 0.60, 0.42, 0.73, 0.42, 0.66, 0.80, 0.55, 0.80)
	// handle
	p.Arc(0.73, 0.58, 0.10, -math.Pi/2, math.Pi/2, 0.05, body)
	// rim + liquid with a gentle swirl
	p.Ellipse(0.5, 0.42, 0.23, 0.055, shade)
	p.Ellipse(0.5, 0.42, 0.20, 0.04, liquid)
	sw := math.Sin(t * 1.2)
	p.Ellipse(0.5+0.03*sw, 0.415, 0.09, 0.015, paint.Mix(liquid, paint.White, 0.18))
	// steam
	c.steam.step(dt, 0.38, 0.62, 0.36, 22)
	c.steam.draw(p)
}

func (c *cup) drawTea(p *paint.Painter, t, dt float64) {
	bowl := paint.Hex(0xE8F0DC)
	bowlShade := paint.Hex(0xB9C7A5)
	tea := paint.Hex(0x7CB342)
	p.Ellipse(0.5, 0.80, 0.30, 0.05, bowlShade)
	// bowl body: a flattened dome
	p.Shade(0.20, 0.46, 0.60, 0.34, func(u, v float64) (paint.RGB, float64) {
		x := (u - 0.5) * 2
		if x*x+(v*v)*0.9 > 1 {
			return bowl, 0
		}
		if u > 0.72 {
			return bowlShade, 1
		}
		return bowl, 1
	})
	p.Ellipse(0.5, 0.47, 0.30, 0.07, bowlShade)
	p.Ellipse(0.5, 0.47, 0.27, 0.055, tea)
	// a floating leaf drifting in circles
	a := t * 0.7
	lx, ly := 0.5+0.12*math.Cos(a), 0.47+0.025*math.Sin(a)
	p.Ellipse(lx, ly, 0.03, 0.012, paint.Hex(0x33691E))
	c.steam.step(dt, 0.30, 0.70, 0.42, 26)
	c.steam.draw(p)
}

// ---------------------------------------------------------------------------

type beer struct{ sys *paint.System }

func (b *beer) Draw(p *paint.Painter, t, dt float64) {
	amber := paint.Hex(0xF5A623)
	amberDark := paint.Hex(0xD98B12)
	glass := paint.Hex(0xE3F2FD)
	// mug body
	p.RoundRect(0.28, 0.30, 0.42, 0.55, 0.04, glass)
	p.Rect(0.31, 0.36, 0.36, 0.46, amber)
	for i := 0; i < 3; i++ {
		x := 0.36 + float64(i)*0.10
		p.Rect(x, 0.36, 0.02, 0.46, amberDark)
	}
	// handle
	p.Arc(0.72, 0.56, 0.11, -math.Pi/2, math.Pi/2, 0.06, glass)
	// bubbles rise inside the beer
	b.sys.Every(dt, 14, func() {
		b.sys.Emit(paint.Particle{X: b.sys.Rand(0.33, 0.65), Y: 0.82, VY: b.sys.Rand(-0.14, -0.08), Life: 4, Size: b.sys.Rand(0.008, 0.02), Spin: b.sys.Rand(0, 6)})
	})
	b.sys.Step(dt, 0, 0)
	clip := p.Clip(0.31, 0.37, 0.36, 0.45)
	for i := range b.sys.P {
		q := &b.sys.P[i]
		q.X += math.Sin(q.Age*4+q.Spin) * 0.03 * dt
		if q.Y < 0.38 {
			q.Age = q.Life
			continue
		}
		clip.Ring(q.X, q.Y, q.Size, q.Size*0.6, paint.Mix(amber, paint.White, 0.6))
	}
	// foam: a row of bumps, bobbing slightly
	foam := paint.Hex(0xFFFDF5)
	for i := 0; i < 5; i++ {
		x := 0.33 + float64(i)*0.085
		y := 0.32 + 0.01*math.Sin(t*2+float64(i))
		p.Circle(x, y, 0.055, foam)
	}
	p.Rect(0.31, 0.32, 0.36, 0.06, foam)
	// drips of foam sliding down the glass
	for i := 0; i < 2; i++ {
		d := math.Mod(t*0.08+float64(i)*0.5, 1)
		x := 0.30 + float64(i)*0.38
		p.Ellipse(x, 0.38+d*0.25, 0.012, 0.03, foam)
	}
	_ = t
}

// ---------------------------------------------------------------------------

type ramen struct{ steam *steam }

func (r *ramen) Draw(p *paint.Painter, t, dt float64) {
	bowl := paint.Hex(0xEF5350)
	bowlDark := paint.Hex(0xB71C1C)
	broth := paint.Hex(0xF6C453)
	noodle := paint.Hex(0xFFE082)
	// bowl
	p.Shade(0.15, 0.50, 0.70, 0.34, func(u, v float64) (paint.RGB, float64) {
		x := (u - 0.5) * 2
		if x*x*0.7+v*v > 1 {
			return bowl, 0
		}
		if v > 0.55 {
			return bowlDark, 1
		}
		return bowl, 1
	})
	p.Rect(0.32, 0.83, 0.36, 0.04, bowlDark)
	p.Ellipse(0.5, 0.50, 0.35, 0.08, bowlDark)
	p.Ellipse(0.5, 0.50, 0.32, 0.065, broth)
	// noodles: wavy lines that ripple
	for i := 0; i < 5; i++ {
		y := 0.47 + float64(i)*0.013
		var prevx, prevy float64
		for k := 0; k <= 16; k++ {
			u := float64(k) / 16
			x := 0.22 + u*0.56
			yy := y + 0.012*math.Sin(u*12+t*2+float64(i))
			if k > 0 {
				p.Line(prevx, prevy, x, yy, 0.012, noodle)
			}
			prevx, prevy = x, yy
		}
	}
	// egg half and a fish cake
	p.Ellipse(0.62, 0.49, 0.06, 0.03, paint.White)
	p.Ellipse(0.62, 0.49, 0.03, 0.017, paint.Amber)
	p.Circle(0.37, 0.485, 0.035, paint.Hex(0xFFF3E0))
	p.Circle(0.37, 0.485, 0.02, paint.Pink)
	// chopsticks resting on the rim, nudging with the steam
	tilt := 0.03 * math.Sin(t*0.9)
	p.Line(0.20, 0.30+tilt, 0.78, 0.44, 0.018, paint.Tan)
	p.Line(0.20, 0.34+tilt, 0.80, 0.47, 0.018, paint.Tan)
	r.steam.step(dt, 0.25, 0.75, 0.44, 32)
	r.steam.draw(p)
}
