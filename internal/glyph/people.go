package glyph

import (
	"math"

	"github.com/Cid-Emmerich/SopeBox/internal/paint"
)

func initpeople() {
	add("people",
		Def{Emoji: "👋", Name: "wave", Blurb: "a hand waving hello", Aliases: []string{"waving hand", "hello", "hi", "bye", "hand"}, New: stateless(drawWaveHand)},
		Def{Emoji: "🏃", Name: "runner", Blurb: "someone running flat out", Aliases: []string{"running", "run", "jog", "sprint"}, New: func() Anim { return &runner{fig: defaultFigure(), dust: paint.NewSystem(21)} }},
		Def{Emoji: "💃", Name: "dancer", Blurb: "dancing in a red dress", Aliases: []string{"dancing", "dance", "salsa", "flamenco"}, New: func() Anim { return &dancer{notes: paint.NewSystem(22)} }},
		Def{Emoji: "🧘", Name: "meditate", Blurb: "breathing slowly in the lotus position", Aliases: []string{"meditation", "yoga", "zen", "lotus", "calm"}, New: stateless(drawMeditate)},
		Def{Emoji: "🤸", Name: "cartwheel", Blurb: "cartwheeling across the screen", Aliases: []string{"gymnast", "gymnastics", "flip", "tumble"}, New: stateless(drawCartwheel)},
		Def{Emoji: "🏋️", Name: "lifter", Blurb: "lifting a heavy barbell", Aliases: []string{"weightlifting", "weights", "gym", "barbell", "lift"}, New: func() Anim { return &lifter{sweat: paint.NewSystem(23)} }},
		Def{Emoji: "👏", Name: "clap", Blurb: "hands clapping", Aliases: []string{"clapping", "applause", "bravo"}, New: func() Anim { return &clap{sparks: paint.NewSystem(24)} }},
		Def{Emoji: "🙋", Name: "raise", Blurb: "raising a hand to ask a question", Aliases: []string{"raising hand", "question", "volunteer", "me"}, New: stateless(drawRaiseHand)},
		Def{Emoji: "🚶", Name: "walker", Blurb: "a relaxed stroll", Aliases: []string{"walking", "walk", "stroll"}, New: func() Anim { return &runner{fig: defaultFigure(), dust: paint.NewSystem(25), walk: true} }},
	)
}

// hand draws a palm with fingers, rotated by a around the wrist (wx,wy).
func hand(p *paint.Painter, wx, wy, size, a float64, skin paint.RGB, fingers [5]float64) {
	pt := func(x, y float64) (float64, float64) {
		return paint.Rot(wx, wy, wx+x*size, wy+y*size, a)
	}
	// palm
	x0, y0 := pt(-0.30, -0.15)
	x1, y1 := pt(0.30, -0.15)
	x2, y2 := pt(0.30, -0.65)
	x3, y3 := pt(-0.30, -0.65)
	p.Poly(skin, x0, y0, x1, y1, x2, y2, x3, y3)
	cx, cy := pt(0, -0.15)
	p.Circle(cx, cy, size*0.30, skin)
	// four fingers along the top edge, thumb to the side
	for i := 0; i < 4; i++ {
		bx := -0.24 + float64(i)*0.16
		lx, ly := pt(bx, -0.62)
		tx, ty := pt(bx+0.05*fingers[i], -0.62-0.36*fingers[i]-0.04)
		p.Line(lx, ly, tx, ty, size*0.13, skin)
	}
	lx, ly := pt(-0.28, -0.40)
	tx, ty := pt(-0.56, -0.62*fingers[4]-0.1)
	p.Line(lx, ly, tx, ty, size*0.14, skin)
}

func drawWaveHand(p *paint.Painter, t, dt float64) {
	a := 0.45 * math.Sin(t*5.5)
	wx, wy := 0.52, 0.80
	// sleeve
	p.RoundRect(wx-0.14, wy-0.06, 0.28, 0.22, 0.04, paint.Blue)
	f := [5]float64{0.95, 1, 1, 0.9, 1}
	hand(p, wx, wy, 0.55, a, paint.Skin, f)
	// motion arcs on the side the hand is swinging toward
	k := math.Abs(math.Sin(t * 5.5))
	side := 1.0
	if math.Cos(t*5.5) < 0 {
		side = -1
	}
	for i := 0; i < 3; i++ {
		r := 0.42 + float64(i)*0.07
		al := (0.55 - float64(i)*0.15) * k
		if side > 0 {
			p.Arc(wx, wy-0.35, r, -0.6, 0.2, 0.014, paint.Mix(paint.DarkGrey, paint.White, al))
		} else {
			p.Arc(wx, wy-0.35, r, math.Pi-0.2, math.Pi+0.6, 0.014, paint.Mix(paint.DarkGrey, paint.White, al))
		}
	}
}

// ---------------------------------------------------------------------------

type runner struct {
	fig  figure
	dust *paint.System
	walk bool
}

func (r *runner) Draw(p *paint.Painter, t, dt float64) {
	speed, amp, lean := 9.0, 0.95, 0.30
	if r.walk {
		speed, amp, lean = 4.0, 0.45, 0.05
	}
	ph := t * speed
	bob := 0.02 * math.Abs(math.Sin(ph))
	hipX, hipY := 0.46, 0.55-bob
	legs := [2]float64{amp * math.Sin(ph), amp * math.Sin(ph+math.Pi)}
	knees := [2]float64{-1.1 * math.Max(0, -math.Sin(ph)), -1.1 * math.Max(0, math.Sin(ph))}
	if r.walk {
		knees = [2]float64{-0.4 * math.Max(0, -math.Sin(ph)), -0.4 * math.Max(0, math.Sin(ph))}
	}
	arms := [2]float64{math.Pi - 0.3 + amp*0.9*math.Sin(ph+math.Pi), math.Pi - 0.3 + amp*0.9*math.Sin(ph)}
	elbows := [2]float64{-1.4, -1.4}
	if r.walk {
		arms = [2]float64{amp * 0.8 * math.Sin(ph+math.Pi), amp * 0.8 * math.Sin(ph)}
		elbows = [2]float64{-0.3, -0.3}
	}
	// dust kicked up when a foot lands
	r.dust.Every(dt, speed/math.Pi, func() {
		r.dust.Emit(paint.Particle{X: hipX - 0.05, Y: 0.86, VX: r.dust.Rand(-0.25, -0.05), VY: r.dust.Rand(-0.12, -0.02), Life: 0.7, Size: r.dust.Rand(0.01, 0.025)})
	})
	r.dust.Step(dt, 0, 1)
	for _, q := range r.dust.P {
		p.CircleA(q.X, q.Y, q.Size*(1+q.T()), paint.Grey, 0.6*(1-q.T()))
	}
	ground(p, 0.87, speed*0.08, t, paint.DarkGrey)
	r.fig.draw(p, hipX, hipY, lean, legs, knees, arms, elbows)
	if !r.walk {
		// speed lines
		for i := 0; i < 3; i++ {
			y := 0.35 + float64(i)*0.12
			off := math.Mod(t*1.4+float64(i)*0.3, 0.5)
			p.Line(0.20-off, y, 0.30-off, y, 0.01, paint.Grey)
		}
	}
}

// ---------------------------------------------------------------------------

type dancer struct{ notes *paint.System }

func (d *dancer) Draw(p *paint.Painter, t, dt float64) {
	sway := math.Sin(t * 3)
	lean := 0.22 * sway
	dress := paint.Hex(0xE53935)
	hipX, hipY := 0.5+0.03*sway, 0.55
	// dress: a swirling skirt
	sx := hipX + math.Sin(lean)*0.2
	sy := hipY - math.Cos(lean)*0.2
	flare := 0.18 + 0.06*math.Abs(sway)
	p.Poly(dress,
		sx-0.06, sy, sx+0.06, sy,
		hipX+flare+0.05*sway, 0.82,
		hipX+flare*0.5+0.08*sway, 0.86,
		hipX-flare*0.5+0.08*sway, 0.86,
		hipX-flare+0.05*sway, 0.82)
	// legs peeking out
	p.Line(hipX-0.05, 0.80, hipX-0.06-0.04*sway, 0.90, 0.045, paint.Skin)
	p.Line(hipX+0.05, 0.80, hipX+0.10+0.04*sway, 0.88, 0.045, paint.Skin)
	// arms: one up flourishing, one out
	p.Line(sx, sy, sx-0.14, sy-0.18-0.03*sway, 0.04, paint.Skin)
	p.Line(sx-0.14, sy-0.18-0.03*sway, sx-0.08, sy-0.36, 0.04, paint.Skin)
	p.Line(sx, sy, sx+0.18, sy-0.02+0.05*sway, 0.04, paint.Skin)
	p.Line(sx+0.18, sy-0.02+0.05*sway, sx+0.30, sy-0.14+0.05*sway, 0.04, paint.Skin)
	// head with hair and a rose
	hx, hy := sx+math.Sin(lean)*0.1, sy-math.Cos(lean)*0.1
	p.Circle(hx, hy, 0.075, paint.Skin)
	p.Arc(hx, hy, 0.06, math.Pi+0.3, 2*math.Pi-0.3, 0.05, paint.Hex(0x3E2723))
	p.Circle(hx-0.06, hy-0.05, 0.025, dress)
	// notes drifting up
	d.notes.Every(dt, 1.5, func() {
		d.notes.Emit(paint.Particle{X: d.notes.Rand(0.15, 0.85), Y: 0.75, VY: -0.15, VX: d.notes.Rand(-0.03, 0.03), Life: 2.5, Size: 0.03, Spin: d.notes.Rand(0, 6)})
	})
	d.notes.Step(dt, 0, 0)
	for _, q := range d.notes.P {
		a := 1 - q.T()
		x := q.X + 0.03*math.Sin(q.Age*3+q.Spin)
		c := paint.Mix(paint.Black, paint.Pink, a)
		p.Ellipse(x, q.Y, q.Size*0.6, q.Size*0.4, c)
		p.Line(x+q.Size*0.5, q.Y, x+q.Size*0.5, q.Y-q.Size*1.5, 0.008, c)
	}
}

// ---------------------------------------------------------------------------

func drawMeditate(p *paint.Painter, t, dt float64) {
	breath := 0.5 + 0.5*math.Sin(t*1.2)
	cx := 0.5
	// aura rings drifting outward
	for i := 0; i < 3; i++ {
		k := math.Mod(t*0.25+float64(i)/3, 1)
		r := 0.18 + k*0.35
		p.Ring(cx, 0.55, r, 0.012, paint.Mix(paint.Hex(0x1C1F2B), paint.Purple, (1-k)*0.8))
	}
	// floating light orbs orbiting
	for i := 0; i < 6; i++ {
		a := t*0.6 + float64(i)*math.Pi/3
		ox, oy := cx+0.40*math.Cos(a), 0.55+0.18*math.Sin(a)
		p.Glow(ox, oy, 0.05, paint.Gold, 0.8)
	}
	// crossed legs (a wide lap)
	p.Ellipse(cx, 0.78, 0.30, 0.09, paint.Hex(0x37474F))
	p.Ellipse(cx-0.2, 0.80, 0.07, 0.035, paint.Skin)
	p.Ellipse(cx+0.2, 0.80, 0.07, 0.035, paint.Skin)
	// torso rises and widens with each breath
	tw, th := 0.13+0.01*breath, 0.30+0.02*breath
	p.RoundRect(cx-tw, 0.74-th, tw*2, th, 0.08, paint.Hex(0xFF7043))
	// arms resting on knees
	p.Line(cx-0.10, 0.52, cx-0.24, 0.72, 0.045, paint.Hex(0xFF7043))
	p.Line(cx+0.10, 0.52, cx+0.24, 0.72, 0.045, paint.Hex(0xFF7043))
	p.Circle(cx-0.24, 0.73, 0.03, paint.Skin)
	p.Circle(cx+0.24, 0.73, 0.03, paint.Skin)
	// head, closed eyes, serene smile
	hy := 0.36 - 0.01*breath
	p.Circle(cx, hy, 0.085, paint.Skin)
	p.Arc(cx, hy, 0.075, math.Pi+0.25, 2*math.Pi-0.25, 0.045, paint.Brown)
	p.Arc(cx-0.035, hy+0.005, 0.02, 0.2, math.Pi-0.2, 0.008, paint.Brown)
	p.Arc(cx+0.035, hy+0.005, 0.02, 0.2, math.Pi-0.2, 0.008, paint.Brown)
	p.Arc(cx, hy+0.02, 0.03, 0.4, math.Pi-0.4, 0.008, paint.Brown)
}

// ---------------------------------------------------------------------------

func drawCartwheel(p *paint.Painter, t, dt float64) {
	ground(p, 0.86, 0.9, t, paint.DarkGrey)
	// travels left to right over 3.5s, then reappears
	k := cycle(t, 3.5)
	x := lerp(p.L-0.3, p.R+0.3, k)
	a := t * 4.5
	cy := 0.58
	f := defaultFigure()
	// star pose: limbs at 45 degrees, whole body rotates around the belly
	limb := func(ang, l float64, c paint.RGB, th float64) {
		x0, y0 := paint.Rot(x, cy, x+0.05*math.Cos(ang+a), cy+0.05*math.Sin(ang+a), 0)
		x1, y1 := x+l*math.Cos(ang+a), cy+l*math.Sin(ang+a)
		p.Line(x0, y0, x1, y1, th, c)
	}
	p.Circle(x, cy, 0.09, f.shirt)
	limb(math.Pi/4, 0.30, f.pants, 0.06)
	limb(3*math.Pi/4, 0.30, f.pants, 0.06)
	limb(-math.Pi/4, 0.28, f.skin, 0.045)
	limb(-3*math.Pi/4, 0.28, f.skin, 0.045)
	hx, hy := x+0.17*math.Cos(a-math.Pi/2), cy+0.17*math.Sin(a-math.Pi/2)
	p.Circle(hx, hy, f.head, f.skin)
	// motion trail
	for i := 1; i <= 4; i++ {
		tx := x - float64(i)*0.06
		p.Ring(tx, cy, 0.2, 0.01, paint.Mix(paint.DarkGrey, paint.Black, float64(i)/5))
	}
}

// ---------------------------------------------------------------------------

type lifter struct{ sweat *paint.System }

func (l *lifter) Draw(p *paint.Painter, t, dt float64) {
	k := cycle(t, 3)
	// bar height: rise with strain, hold, drop
	var h float64
	switch {
	case k < 0.35:
		h = smooth(k / 0.35)
	case k < 0.7:
		h = 1
	default:
		h = 1 - smooth((k-0.7)/0.3)
	}
	barY := lerp(0.55, 0.18, h)
	cx := 0.5
	f := defaultFigure()
	f.shirt = paint.Hex(0x8E24AA)
	// knees bend while pushing
	bend := 0.35 * (1 - h)
	legs := [2]float64{-0.35 + bend*0.3, 0.35 - bend*0.3}
	knees := [2]float64{bend * 1.5, -bend * 1.5}
	hipY := 0.60 + bend*0.06
	// arms reach for the bar
	sx, sy := cx, hipY-f.torso
	dx, dy := 0.14, barY-sy
	ang := math.Atan2(dx, -dy)
	arms := [2]float64{math.Pi - ang, math.Pi + ang}
	elbows := [2]float64{0, 0}
	// stretch the arm length by drawing our own lines after the figure
	f.limb = 0.15
	f.draw(p, cx, hipY, 0, legs, knees, arms, elbows)
	p.Line(sx, sy, cx-dx, barY, 0.05, f.shirt)
	p.Line(sx, sy, cx+dx, barY, 0.05, f.shirt)
	// barbell
	p.Line(cx-0.40, barY, cx+0.40, barY, 0.03, paint.Steel)
	for _, s := range []float64{-1, 1} {
		p.RoundRect(cx+s*0.34-0.04, barY-0.12, 0.08, 0.24, 0.02, paint.Hex(0x263238))
		p.RoundRect(cx+s*0.27-0.03, barY-0.09, 0.06, 0.18, 0.02, paint.Hex(0x37474F))
	}
	// sweat when straining
	if h > 0.05 && h < 0.95 {
		l.sweat.Every(dt, 6, func() {
			l.sweat.Emit(paint.Particle{X: cx + l.sweat.Rand(-0.08, 0.08), Y: sy - 0.10, VX: l.sweat.Rand(-0.2, 0.2), VY: -0.1, Life: 0.8, Size: 0.014})
		})
	}
	l.sweat.Step(dt, 0.8, 0)
	for _, q := range l.sweat.P {
		p.CircleA(q.X, q.Y, q.Size, paint.SkyBlue, 1-q.T())
	}
	ground(p, 0.9, 0, t, paint.DarkGrey)
}

// ---------------------------------------------------------------------------

type clap struct {
	sparks *paint.System
	was    bool
}

func (c *clap) Draw(p *paint.Painter, t, dt float64) {
	gap := math.Abs(math.Sin(t * 5))
	d := 0.05 + 0.20*gap
	f := [5]float64{1, 1, 1, 0.95, 0.8}
	cy := 0.62
	// left hand tilts right, right hand tilts left (mirrored via rotation)
	hand(p, 0.5-d, cy, 0.5, 0.55, paint.Skin, f)
	hand(p, 0.5+d, cy, 0.5, -0.55, paint.Skin, f)
	p.RoundRect(0.5-d-0.11, cy-0.03, 0.22, 0.18, 0.03, paint.Teal)
	p.RoundRect(0.5+d-0.11, cy-0.03, 0.22, 0.18, 0.03, paint.Teal)
	hit := gap < 0.12
	if hit && !c.was {
		for i := 0; i < 10; i++ {
			a := float64(i) * math.Pi / 5
			c.sparks.Emit(paint.Particle{X: 0.5, Y: cy - 0.28, VX: 0.6 * math.Cos(a), VY: 0.6 * math.Sin(a), Life: 0.45, Size: 0.02, C: paint.Gold})
		}
	}
	c.was = hit
	c.sparks.Step(dt, 0, 2)
	for _, q := range c.sparks.P {
		l := q.Size * (1 - q.T()) * 3
		p.Line(q.X, q.Y, q.X+q.VX*0.05, q.Y+q.VY*0.05, l, q.C)
	}
}

// ---------------------------------------------------------------------------

func drawRaiseHand(p *paint.Painter, t, dt float64) {
	f := defaultFigure()
	f.shirt = paint.Hex(0x43A047)
	k := cycle(t, 4)
	// arm rises over half a second, waves while up, then lowers
	var up float64
	switch {
	case k < 0.15:
		up = smooth(k / 0.15)
	case k < 0.8:
		up = 1
	default:
		up = 1 - smooth((k-0.8)/0.2)
	}
	wave := 0.25 * math.Sin(t*9) * up
	arms := [2]float64{lerp(0.3, math.Pi+0.2, up) + wave, -0.4}
	elbows := [2]float64{lerp(-0.4, -0.15, up) + wave*0.5, 0.9}
	legs := [2]float64{-0.12, 0.12}
	f.draw(p, 0.5, 0.60, 0, legs, [2]float64{0, 0}, arms, elbows)
	if up > 0.9 {
		// excited bounce lines
		for i := 0; i < 2; i++ {
			p.Line(0.68+float64(i)*0.04, 0.12+float64(i)*0.05, 0.72+float64(i)*0.04, 0.09+float64(i)*0.05, 0.01, paint.Gold)
		}
		p.TextCentred(0.5, 0.05+0.01*math.Sin(t*9), 0.08, "ME!", paint.Gold)
	}
	ground(p, 0.9, 0, t, paint.DarkGrey)
}
