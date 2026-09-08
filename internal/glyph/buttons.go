package glyph

import (
	"math"

	"github.com/Cid-Emmerich/SopeBox/internal/paint"
)

func initbuttons() {
	add("buttons",
		Def{Emoji: "❤️", Name: "heart", Blurb: "a like button that bursts when tapped", Aliases: []string{"love", "like", "red heart", "♥", "<3"}, New: func() Anim { return &heart{burst: paint.NewSystem(41)} }},
		Def{Emoji: "👍", Name: "thumbs up", Blurb: "the thumbs up that pops when you like", Aliases: []string{"thumbsup", "like", "approve", "yes", "+1"}, New: func() Anim { return &thumbs{burst: paint.NewSystem(42)} }},
		Def{Emoji: "⭐", Name: "star", Blurb: "a rating star filling up gold", Aliases: []string{"favourite", "favorite", "rating", "gold star", "🌟"}, New: func() Anim { return &star{sparks: paint.NewSystem(43)} }},
		Def{Emoji: "🔔", Name: "bell", Blurb: "the notification bell rings and gets a badge", Aliases: []string{"notification", "subscribe", "ring", "alert"}, New: stateless(drawBell)},
		Def{Emoji: "▶️", Name: "play", Blurb: "play button with a progress ring, toggling to pause", Aliases: []string{"video", "pause", "⏯", "media", "youtube"}, New: stateless(drawPlay)},
		Def{Emoji: "✅", Name: "check", Blurb: "a tick draws itself and bounces", Aliases: []string{"tick", "done", "ok", "success", "✔", "complete"}, New: func() Anim { return &check{confetti: paint.NewSystem(44)} }},
		Def{Emoji: "🔗", Name: "share", Blurb: "the share arrow flies out with ripples", Aliases: []string{"link", "send", "forward", "share button"}, New: stateless(drawShare)},
		Def{Emoji: "💬", Name: "comment", Blurb: "speech bubbles typing dots", Aliases: []string{"chat", "message", "speech", "typing", "reply"}, New: stateless(drawComment)},
	)
}

// ---------------------------------------------------------------------------

type heart struct {
	burst *paint.System
	liked bool
	last  int
}

func (h *heart) Draw(p *paint.Painter, t, dt float64) {
	const period = 3.0
	n := int(t / period)
	if n != h.last {
		h.last = n
		h.liked = !h.liked
		if h.liked {
			for i := 0; i < 18; i++ {
				a := float64(i) * 2 * math.Pi / 18
				sp := h.burst.Rand(0.5, 0.9)
				h.burst.Emit(paint.Particle{X: 0.5, Y: 0.5, VX: sp * math.Cos(a), VY: sp * math.Sin(a), Life: 0.9, Size: h.burst.Rand(0.02, 0.045), Kind: i % 2})
			}
		}
	}
	x := t - float64(n)*period
	red := paint.Hex(0xFF1744)
	if h.liked {
		// ring ripple then a springy pop
		if x < 0.6 {
			r := 0.25 + x*0.6
			p.Ring(0.5, 0.5, r, 0.03*(1-x/0.6), paint.Mix(paint.Black, red, 1-x/0.6))
		}
		beat := math.Exp(-8*math.Mod(t, 1)) * 0.5
		beat += math.Exp(-8*math.Mod(t+0.75, 1)) * 0.3
		s := pop(x) * (1 + 0.05*beat)
		p.Heart(0.5, 0.5, 0.62*s, red)
		p.Heart(0.5, 0.5, 0.62*s*0.9, paint.Mix(red, paint.White, 0.06))
		p.Ellipse(0.40, 0.40, 0.05*s, 0.03*s, paint.Mix(red, paint.White, 0.45))
	} else {
		s := 1.0
		if x < 0.4 {
			s = 1 - 0.15*math.Sin(x/0.4*math.Pi)
		}
		p.HeartRing(0.5, 0.5, 0.62*s, 0.035, paint.Grey)
	}
	h.burst.Step(dt, 0.5, 2.5)
	for _, q := range h.burst.P {
		k := 1 - q.T()
		c := paint.Mix(red, paint.Pink, float64(q.Kind))
		if q.Kind == 0 {
			p.Heart(q.X, q.Y, q.Size*2*k, c)
		} else {
			p.Circle(q.X, q.Y, q.Size*k*0.7, c)
		}
	}
}

// ---------------------------------------------------------------------------

type thumbs struct {
	burst *paint.System
	last  int
}

func (th *thumbs) Draw(p *paint.Painter, t, dt float64) {
	const period = 3.0
	n := int(t / period)
	x := t - float64(n)*period
	if n != th.last {
		th.last = n
		for i := 0; i < 12; i++ {
			a := -math.Pi/2 + (float64(i)/11-0.5)*math.Pi*1.2
			th.burst.Emit(paint.Particle{X: 0.5, Y: 0.45, VX: 0.8 * math.Cos(a), VY: 0.8 * math.Sin(a), Life: 0.8, Size: 0.02, Kind: i % 3})
		}
	}
	// blue = liked for the first two thirds, grey afterwards
	liked := x < period*0.66
	col := paint.Hex(0x3EA6FF)
	if !liked {
		col = paint.Grey
	}
	s := 1.0
	rot := 0.0
	lift := 0.0
	if liked {
		s = pop(x)
		rot = -0.35 * math.Exp(-4*x) * math.Cos(8*x)
		lift = 0.05 * math.Exp(-3*x) * math.Sin(6*x)
	}
	cx, cy := 0.5, 0.55-lift
	pt := func(px, py float64) (float64, float64) {
		return paint.Rot(cx, cy, cx+px*s, cy+py*s, rot)
	}
	// fist block with four finger creases
	x0, y0 := pt(-0.10, -0.05)
	x1, y1 := pt(0.24, -0.05)
	x2, y2 := pt(0.24, 0.32)
	x3, y3 := pt(-0.10, 0.32)
	p.Poly(col, x0, y0, x1, y1, x2, y2, x3, y3)
	for i := 0; i < 3; i++ {
		ax, ay := pt(-0.02, 0.04+float64(i)*0.09)
		bx, by := pt(0.20, 0.04+float64(i)*0.09)
		p.Line(ax, ay, bx, by, 0.012*s, paint.Mix(col, paint.Black, 0.35))
	}
	// thumb
	tx0, ty0 := pt(-0.06, -0.05)
	tx1, ty1 := pt(-0.02, -0.32)
	p.Line(tx0, ty0, tx1, ty1, 0.13*s, col)
	// cuff
	cx0, cy0 := pt(-0.30, -0.02)
	cx1, cy1 := pt(-0.14, -0.02)
	cx2, cy2 := pt(-0.14, 0.32)
	cx3, cy3 := pt(-0.30, 0.32)
	p.Poly(paint.Mix(col, paint.Black, 0.3), cx0, cy0, cx1, cy1, cx2, cy2, cx3, cy3)
	if liked && x < 0.5 {
		p.Ring(cx, cy-0.05, 0.3+x*0.8, 0.025*(1-x*2), paint.Mix(paint.Black, col, 1-x*2))
	}
	th.burst.Step(dt, 0.6, 2)
	for _, q := range th.burst.P {
		k := 1 - q.T()
		c := []paint.RGB{col, paint.Gold, paint.White}[q.Kind]
		if q.Kind == 1 {
			p.Line(q.X-0.015*k, q.Y, q.X+0.015*k, q.Y, 0.008, c)
			p.Line(q.X, q.Y-0.015*k, q.X, q.Y+0.015*k, 0.008, c)
		} else {
			p.Circle(q.X, q.Y, q.Size*k, c)
		}
	}
}

// ---------------------------------------------------------------------------

type star struct{ sparks *paint.System }

func (s *star) Draw(p *paint.Painter, t, dt float64) {
	k := cycle(t, 4)
	rot := 0.12 * math.Sin(t*1.5)
	grey := paint.Hex(0x5A5F66)
	gold := paint.Hex(0xFFC107)
	// fill sweeps up from the bottom during the first second, then pops
	fill := smooth(k * 4)
	sc := 1.0
	if k > 0.25 && k < 0.6 {
		sc = pop(k - 0.25)
	}
	r := 0.42 * sc
	p.Star(0.5, 0.5, r, r*0.45, 5, rot, grey)
	top := 0.5 + r - fill*2*r
	p.Clip(0, top, 1, 1).Star(0.5, 0.5, r, r*0.45, 5, rot, gold)
	p.Star(0.5, 0.5, r*0.6, r*0.27, 5, rot, paint.Mix(gold, paint.White, 0.25*fill))
	if fill >= 1 && k < 0.4 {
		s.sparks.Every(dt, 30, func() {
			a := s.sparks.Rand(0, 2*math.Pi)
			d := s.sparks.Rand(0.3, 0.5)
			s.sparks.Emit(paint.Particle{X: 0.5 + d*math.Cos(a), Y: 0.5 + d*math.Sin(a), VX: 0.2 * math.Cos(a), VY: 0.2 * math.Sin(a), Life: 0.7, Size: s.sparks.Rand(0.015, 0.035)})
		})
	}
	s.sparks.Step(dt, 0, 1)
	for _, q := range s.sparks.P {
		p.Star(q.X, q.Y, q.Size*(1-q.T())*1.6, q.Size*(1-q.T())*0.5, 4, q.Age*3, paint.Mix(gold, paint.White, 0.5))
	}
	// little orbiting stars
	for i := 0; i < 3; i++ {
		a := t*1.2 + float64(i)*2.1
		p.Star(0.5+0.47*math.Cos(a), 0.5+0.3*math.Sin(a), 0.03, 0.013, 5, a, paint.Mix(gold, paint.White, 0.3))
	}
}

// ---------------------------------------------------------------------------

func drawBell(p *paint.Painter, t, dt float64) {
	const period = 4.0
	x := math.Mod(t, period)
	// ring for the first 1.2s with a decaying swing
	env := 0.0
	if x < 1.4 {
		env = math.Exp(-2.2 * x)
	}
	a := 0.45 * env * math.Sin(x*22)
	cx, cy := 0.5, 0.20 // pivot at the top of the bell
	gold := paint.Hex(0xFFB300)
	dark := paint.Hex(0xE08A00)
	pt := func(px, py float64) (float64, float64) { return paint.Rot(cx, cy, cx+px, cy+py, a) }
	// bell body as a polygon (dome + flared skirt)
	var pts []float64
	for i := 0; i <= 16; i++ {
		th := math.Pi + float64(i)*math.Pi/16
		bx, by := pt(0.22*math.Cos(th), 0.22*math.Sin(th)+0.20)
		pts = append(pts, bx, by)
	}
	sx0, sy0 := pt(0.30, 0.54)
	sx1, sy1 := pt(-0.30, 0.54)
	pts = append(pts, sx0, sy0, sx1, sy1)
	p.Poly(gold, pts...)
	rx0, ry0 := pt(-0.32, 0.54)
	rx1, ry1 := pt(0.32, 0.54)
	p.Line(rx0, ry0, rx1, ry1, 0.05, dark)
	hx, hy := pt(0, -0.03)
	p.Circle(hx, hy, 0.035, dark)
	// clapper swings the other way
	clx, cly := paint.Rot(cx, cy, cx, cy+0.62, -a*1.4)
	p.Circle(clx, cly, 0.045, dark)
	// sound waves
	if env > 0.05 {
		for i := 0; i < 3; i++ {
			r := 0.36 + float64(i)*0.08 + x*0.1
			al := env * (1 - float64(i)*0.25)
			c := paint.Mix(paint.Black, gold, al)
			p.Arc(cx, 0.45, r, -0.5, 0.5, 0.015, c)
			p.Arc(cx, 0.45, r, math.Pi-0.5, math.Pi+0.5, 0.015, c)
		}
	}
	// badge appears after the ring
	if x > 0.8 {
		s := pop(x - 0.8)
		p.Circle(0.74, 0.22, 0.09*s, paint.Hex(0xFF1744))
		p.TextCentred(0.74, 0.22-0.05*s, 0.10*s, "1", paint.White)
	}
	// ground shadow
	p.Ellipse(0.5, 0.92, 0.28, 0.03, paint.Hex(0x2A2D33))
}

// ---------------------------------------------------------------------------

func drawPlay(p *paint.Painter, t, dt float64) {
	const period = 4.0
	x := math.Mod(t, period)
	playing := int(t/period)%2 == 1
	red := paint.Hex(0xFF0000)
	s := 1.0
	if x < 0.5 {
		s = pop(x)
	}
	w, h := 0.62*s, 0.44*s
	p.RoundRect(0.5-w/2, 0.5-h/2, w, h, 0.10*s, red)
	m := smooth(x * 3) // morph triangle ↔ bars during the first third of a second
	if !playing {
		m = 1 - m
	}
	// triangle shrinks as bars grow
	tri := 1 - m
	if tri > 0.01 {
		k := 0.17 * s * tri
		p.Triangle(0.5-k*0.8, 0.5-k, 0.5-k*0.8, 0.5+k, 0.5+k*1.1, 0.5, paint.White)
	}
	if m > 0.01 {
		k := 0.17 * s * m
		p.RoundRect(0.5-k*0.8, 0.5-k, k*0.55, 2*k, 0.02, paint.White)
		p.RoundRect(0.5+k*0.25, 0.5-k, k*0.55, 2*k, 0.02, paint.White)
	}
	// progress ring around the button
	prog := x / period
	p.Ring(0.5, 0.5, 0.42, 0.02, paint.Hex(0x3A3D42))
	p.Arc(0.5, 0.5, 0.42, -math.Pi/2, -math.Pi/2+prog*2*math.Pi, 0.03, paint.Mix(red, paint.White, 0.2))
	// timestamp
	secs := int(x)
	p.TextCentred(0.5, 0.86, 0.06, "0:0"+string(rune('0'+secs))+" / 0:04", paint.Grey)
}

// ---------------------------------------------------------------------------

type check struct {
	confetti *paint.System
	last     int
}

func (c *check) Draw(p *paint.Painter, t, dt float64) {
	const period = 3.2
	n := int(t / period)
	x := t - float64(n)*period
	green := paint.Hex(0x2E7D32)
	s := 1.0
	if x > 0.75 && x < 1.5 {
		s = pop(x - 0.75)
	}
	if x > period-0.4 {
		s *= 1 - smooth((x-(period-0.4))/0.4)
	}
	if s < 0.01 {
		return
	}
	w := 0.72 * s
	p.RoundRect(0.5-w/2, 0.5-w/2, w, w, 0.12*s, green)
	// tick drawn as two strokes over the first 0.75s
	a := smooth(x / 0.4)
	b := smooth((x - 0.35) / 0.4)
	p0 := [2]float64{0.30, 0.52}
	p1 := [2]float64{0.44, 0.66}
	p2 := [2]float64{0.72, 0.34}
	tr := func(q [2]float64) (float64, float64) { return 0.5 + (q[0]-0.5)*s, 0.5 + (q[1]-0.5)*s }
	x0, y0 := tr(p0)
	x1, y1 := tr(p1)
	x2, y2 := tr(p2)
	if a > 0 {
		p.Line(x0, y0, lerp(x0, x1, a), lerp(y0, y1, a), 0.08*s, paint.White)
	}
	if b > 0 {
		p.Line(x1, y1, lerp(x1, x2, b), lerp(y1, y2, b), 0.08*s, paint.White)
	}
	if n != c.last && x > 0.75 {
		c.last = n
		for i := 0; i < 26; i++ {
			an := c.confetti.Rand(0, 2*math.Pi)
			sp := c.confetti.Rand(0.5, 1.1)
			c.confetti.Emit(paint.Particle{X: 0.5, Y: 0.5, VX: sp * math.Cos(an), VY: sp*math.Sin(an) - 0.3, Life: 1.2, Size: 0.025, Kind: i % 4, Spin: c.confetti.Rand(0, 6)})
		}
	}
	c.confetti.Step(dt, 1.2, 1)
	cols := []paint.RGB{paint.Gold, paint.Pink, paint.SkyBlue, paint.Lime}
	for _, q := range c.confetti.P {
		k := 1 - q.T()
		an := q.Spin + q.Age*6
		p.Line(q.X-q.Size*math.Cos(an)*k, q.Y-q.Size*math.Sin(an)*k, q.X+q.Size*math.Cos(an)*k, q.Y+q.Size*math.Sin(an)*k, 0.014, cols[q.Kind])
	}
}

// ---------------------------------------------------------------------------

func drawShare(p *paint.Painter, t, dt float64) {
	k := cycle(t, 2.5)
	col := paint.Hex(0x3EA6FF)
	// tray
	p.RoundRect(0.25, 0.45, 0.50, 0.40, 0.05, paint.Hex(0x3A3D42))
	p.RoundRect(0.29, 0.49, 0.42, 0.32, 0.04, paint.Hex(0x1E2024))
	// arrow flies up out of the tray and loops back
	fly := smooth(k * 1.6)
	y := lerp(0.62, 0.20, fly)
	if k > 0.8 {
		y = lerp(0.20, 0.62, smooth((k-0.8)/0.2))
	}
	p.Line(0.5, y+0.18, 0.5, y, 0.06, col)
	p.Triangle(0.5-0.12, y+0.08, 0.5+0.12, y+0.08, 0.5, y-0.08, col)
	// ripples at the top when it lands
	if k > 0.55 && k < 0.85 {
		r := (k - 0.55) / 0.3
		p.Ring(0.5, 0.20, 0.12+r*0.3, 0.02*(1-r), paint.Mix(paint.Black, col, 1-r))
	}
	// curved share arrow on the right
	p.Arc(0.82, 0.30, 0.10, math.Pi/2, math.Pi*1.5, 0.03, paint.Mix(col, paint.White, 0.4*math.Abs(math.Sin(t*3))))
	p.Triangle(0.82, 0.16, 0.90, 0.20, 0.82, 0.24, paint.Mix(col, paint.White, 0.3))
}

func drawComment(p *paint.Painter, t, dt float64) {
	blue := paint.Hex(0x1E88E5)
	white := paint.Hex(0xECEFF1)
	// two bubbles bob in turn
	b1 := 0.01 * math.Sin(t*2)
	b2 := 0.01 * math.Sin(t*2+math.Pi)
	p.RoundRect(0.12, 0.22+b1, 0.50, 0.28, 0.09, blue)
	p.Triangle(0.20, 0.49+b1, 0.30, 0.49+b1, 0.17, 0.58+b1, blue)
	p.RoundRect(0.38, 0.54+b2, 0.50, 0.28, 0.09, white)
	p.Triangle(0.70, 0.81+b2, 0.80, 0.81+b2, 0.83, 0.90+b2, white)
	// typing dots wave in whichever bubble is "talking"
	talker := int(t/2) % 2
	for i := 0; i < 3; i++ {
		d := math.Max(0, math.Sin(t*6-float64(i)*0.9))
		if talker == 0 {
			p.Circle(0.27+float64(i)*0.09, 0.36+b1-0.025*d, 0.03, paint.Mix(paint.White, blue, 0.2))
		} else {
			p.Circle(0.53+float64(i)*0.09, 0.68+b2-0.025*d, 0.03, paint.Hex(0x546E7A))
		}
	}
	// the other bubble shows a short line of "text"
	if talker == 1 {
		p.Line(0.22, 0.32+b1, 0.52, 0.32+b1, 0.03, paint.Mix(blue, paint.White, 0.6))
		p.Line(0.22, 0.40+b1, 0.42, 0.40+b1, 0.03, paint.Mix(blue, paint.White, 0.6))
	} else {
		p.Line(0.48, 0.64+b2, 0.78, 0.64+b2, 0.03, paint.Hex(0x90A4AE))
		p.Line(0.48, 0.72+b2, 0.66, 0.72+b2, 0.03, paint.Hex(0x90A4AE))
	}
}
