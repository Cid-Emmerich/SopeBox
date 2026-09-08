package glyph

import (
	"math"
	"math/rand"
	"time"

	"github.com/Cid-Emmerich/SopeBox/internal/paint"
)

func initnature() {
	add("nature",
		Def{Emoji: "🔥", Name: "fire", Blurb: "a real flame simulation", Aliases: []string{"flame", "hot", "lit", "burn", "campfire"}, New: func() Anim { return newFire() }},
		Def{Emoji: "🌧️", Name: "rain", Blurb: "a cloud pouring rain", Aliases: []string{"rain cloud", "raining", "cloud", "☁️", "shower"}, New: func() Anim { return &rain{drops: paint.NewSystem(51)} }},
		Def{Emoji: "⛈️", Name: "storm", Blurb: "thunder and lightning", Aliases: []string{"thunder", "thunderstorm", "lightning cloud"}, New: func() Anim { return &rain{drops: paint.NewSystem(52), storm: true} }},
		Def{Emoji: "☀️", Name: "sun", Blurb: "the sun with turning rays", Aliases: []string{"sunny", "sunshine", "day", "bright"}, New: stateless(func(p *paint.Painter, t, dt float64) { drawSun(p, t, false) })},
		Def{Emoji: "🌞", Name: "sun face", Blurb: "a smiling sun", Aliases: []string{"happy sun", "sun with face"}, New: stateless(func(p *paint.Painter, t, dt float64) { drawSun(p, t, true) })},
		Def{Emoji: "🌙", Name: "moon", Blurb: "a crescent moon under twinkling stars", Aliases: []string{"crescent", "night", "sleep", "🌛", "stars"}, New: stateless(drawMoon)},
		Def{Emoji: "🌊", Name: "ocean", Blurb: "rolling ocean waves", Aliases: []string{"waves", "sea", "water", "surf", "tide"}, New: stateless(drawOcean)},
		Def{Emoji: "🌈", Name: "rainbow", Blurb: "a rainbow paints itself across the sky", Aliases: []string{"colours", "colors", "pride", "arc"}, New: stateless(drawRainbow)},
		Def{Emoji: "⚡", Name: "bolt", Blurb: "high voltage", Aliases: []string{"lightning", "zap", "electric", "power", "energy"}, New: func() Anim { return &bolt{sparks: paint.NewSystem(53)} }},
		Def{Emoji: "🌸", Name: "flower", Blurb: "a blossom opening", Aliases: []string{"blossom", "cherry blossom", "sakura", "bloom", "🌼", "🌺"}, New: func() Anim { return &flower{petals: paint.NewSystem(54)} }},
		Def{Emoji: "❄️", Name: "snowflake", Blurb: "a snowflake turning in a flurry", Aliases: []string{"snow", "winter", "cold", "frost", "ice"}, New: stateless(drawSnowflake)},
		Def{Emoji: "🌪️", Name: "tornado", Blurb: "a twister flinging debris", Aliases: []string{"twister", "cyclone", "whirlwind", "storm"}, New: func() Anim { return &tornado{debris: paint.NewSystem(55)} }},
		Def{Emoji: "🌱", Name: "seedling", Blurb: "a sprout growing", Aliases: []string{"sprout", "plant", "grow", "seed", "garden"}, New: stateless(drawSeedling)},
		Def{Emoji: "🌋", Name: "volcano", Blurb: "an eruption in progress", Aliases: []string{"eruption", "lava", "mountain"}, New: func() Anim { return &volcano{lava: paint.NewSystem(56)} }},
	)
}

// ---------------------------------------------------------------------------

const fireW, fireH = 48, 60

type fire struct {
	heat [fireH][fireW]float64
	rng  *rand.Rand
	acc  float64
}

func newFire() *fire { return &fire{rng: rand.New(rand.NewSource(time.Now().UnixNano()))} }

func (f *fire) step(t float64) {
	// seed the bottom rows in a flame shaped band
	width := 0.55 + 0.1*math.Sin(t*3) + 0.06*math.Sin(t*7.3)
	for x := 0; x < fireW; x++ {
		u := (float64(x)/fireW - 0.5) * 2
		v := 0.0
		if math.Abs(u) < width {
			v = 1 - math.Pow(math.Abs(u)/width, 2)
			v *= 0.7 + 0.5*f.rng.Float64()
		}
		f.heat[fireH-1][x] = v
		f.heat[fireH-2][x] = v * (0.8 + 0.3*f.rng.Float64())
	}
	for y := 0; y < fireH-2; y++ {
		for x := 0; x < fireW; x++ {
			l, r := max(x-1, 0), min(x+1, fireW-1)
			// drift slightly with a wind that changes over time
			wx := x + int(math.Round(math.Sin(t*1.5)*1.0))
			wx = min(max(wx, 0), fireW-1)
			v := (f.heat[y+1][l] + f.heat[y+1][wx] + f.heat[y+1][r] + f.heat[y+2][x]) / 4.0
			v -= 0.012 + 0.02*f.rng.Float64()
			if v < 0 {
				v = 0
			}
			f.heat[y][x] = v
		}
	}
}

func (f *fire) Draw(p *paint.Painter, t, dt float64) {
	f.acc += dt
	for f.acc >= 1.0/30 {
		f.acc -= 1.0 / 30
		f.step(t)
	}
	if dt == 0 && f.acc == 0 {
		// paused before any frame: still show something
		f.step(t)
	}
	p.Shade(0.15, 0.08, 0.70, 0.80, func(u, v float64) (paint.RGB, float64) {
		x := int(u * fireW)
		y := int(v * fireH)
		if x >= fireW {
			x = fireW - 1
		}
		if y >= fireH {
			y = fireH - 1
		}
		h := f.heat[y][x]
		if h < 0.06 {
			return paint.Black, 0
		}
		var c paint.RGB
		switch {
		case h < 0.3:
			c = paint.Mix(paint.Hex(0x7A0A00), paint.Hex(0xE53935), (h-0.06)/0.24)
		case h < 0.6:
			c = paint.Mix(paint.Hex(0xE53935), paint.Hex(0xFFA000), (h-0.3)/0.3)
		case h < 0.85:
			c = paint.Mix(paint.Hex(0xFFA000), paint.Hex(0xFFEE58), (h-0.6)/0.25)
		default:
			c = paint.Mix(paint.Hex(0xFFEE58), paint.White, (h-0.85)/0.15)
		}
		return c, 1
	})
	// logs
	p.Line(0.28, 0.90, 0.72, 0.82, 0.05, paint.Hex(0x6D4C41))
	p.Line(0.28, 0.82, 0.72, 0.90, 0.05, paint.Hex(0x5D4037))
}

// ---------------------------------------------------------------------------

type rain struct {
	drops  *paint.System
	storm  bool
	flash  float64
	boltPt []float64
}

func cloud(p *paint.Painter, cx, cy, s float64, c paint.RGB) {
	p.Circle(cx-0.22*s, cy+0.02*s, 0.16*s, c)
	p.Circle(cx-0.05*s, cy-0.10*s, 0.22*s, c)
	p.Circle(cx+0.18*s, cy-0.02*s, 0.18*s, c)
	p.Circle(cx+0.30*s, cy+0.06*s, 0.12*s, c)
	p.RoundRect(cx-0.34*s, cy+0.02*s, 0.72*s, 0.14*s, 0.06*s, c)
}

func (r *rain) Draw(p *paint.Painter, t, dt float64) {
	cc := paint.Hex(0xCFD8DC)
	if r.storm {
		cc = paint.Hex(0x78909C)
	}
	bob := 0.01 * math.Sin(t*1.5)
	cy := 0.30 + bob
	// lightning: every 2.4s a bolt for 0.35s
	if r.storm {
		k := math.Mod(t, 2.4)
		if k < 0.35 {
			if r.flash <= 0 {
				// build a jagged bolt path
				r.boltPt = r.boltPt[:0]
				x, y := 0.5+r.drops.Rand(-0.1, 0.1), cy+0.15
				for y < 0.92 {
					r.boltPt = append(r.boltPt, x, y)
					x += r.drops.Rand(-0.09, 0.09)
					y += r.drops.Rand(0.06, 0.12)
				}
			}
			r.flash = 1 - k/0.35
		} else {
			r.flash = 0
		}
		if r.flash > 0 {
			p.RectA(p.L, p.T, p.R-p.L, p.Bt-p.T, paint.White, 0.10*r.flash)
			for i := 0; i+3 < len(r.boltPt); i += 2 {
				p.Line(r.boltPt[i], r.boltPt[i+1], r.boltPt[i+2], r.boltPt[i+3], 0.03*r.flash+0.01, paint.Mix(paint.Yellow, paint.White, r.flash))
			}
		}
	}
	// rain drops
	r.drops.Every(dt, 45, func() {
		r.drops.Emit(paint.Particle{X: r.drops.Rand(0.18, 0.82), Y: cy + 0.15, VY: r.drops.Rand(0.9, 1.3), VX: -0.1, Life: 1.2, Size: r.drops.Rand(0.02, 0.045)})
	})
	r.drops.Step(dt, 0.5, 0)
	blue := paint.Hex(0x4FC3F7)
	for i := range r.drops.P {
		q := &r.drops.P[i]
		if q.Y > 0.90 {
			// splash
			p.CircleA(q.X-0.02, 0.90, 0.008, blue, 0.7)
			p.CircleA(q.X+0.02, 0.89, 0.008, blue, 0.7)
			q.Age = q.Life
			continue
		}
		p.Line(q.X, q.Y, q.X+q.VX*0.03, q.Y+q.Size, 0.012, blue)
	}
	cloud(p, 0.5, cy, 1, cc)
	cloud(p, 0.47, cy-0.04, 0.8, paint.Mix(cc, paint.White, 0.25))
	// puddle
	p.Ellipse(0.5, 0.92, 0.36+0.01*math.Sin(t*3), 0.025, paint.Mix(blue, paint.Black, 0.5))
}

// ---------------------------------------------------------------------------

func drawSun(p *paint.Painter, t float64, face bool) {
	yellow := paint.Hex(0xFFC107)
	orange := paint.Hex(0xFF8F00)
	p.Glow(0.5, 0.5, 0.55, orange, 0.5+0.1*math.Sin(t*2))
	rot := t * 0.35
	for i := 0; i < 12; i++ {
		a := rot + float64(i)*math.Pi/6
		l := 0.42 + 0.05*math.Sin(t*3+float64(i))
		if i%2 == 1 {
			l -= 0.06
		}
		x0, y0 := 0.5+0.26*math.Cos(a), 0.5+0.26*math.Sin(a)
		x1, y1 := 0.5+l*math.Cos(a), 0.5+l*math.Sin(a)
		p.Line(x0, y0, x1, y1, 0.045, orange)
	}
	p.Circle(0.5, 0.5, 0.25, orange)
	p.Circle(0.5, 0.5, 0.22, yellow)
	if face {
		b := blink(t, 4, 0)
		p.Ellipse(0.43, 0.46, 0.028, 0.045*b+0.004, paint.Hex(0x4E342E))
		p.Ellipse(0.57, 0.46, 0.028, 0.045*b+0.004, paint.Hex(0x4E342E))
		p.Arc(0.5, 0.52, 0.10, 0.35, math.Pi-0.35, 0.025, paint.Hex(0x4E342E))
		p.Circle(0.38, 0.56, 0.03, paint.Mix(yellow, paint.Pink, 0.5))
		p.Circle(0.62, 0.56, 0.03, paint.Mix(yellow, paint.Pink, 0.5))
	}
}

func drawMoon(p *paint.Painter, t, dt float64) {
	stars(p, t, 40, 5)
	moon := paint.Hex(0xFFEB3B)
	p.Glow(0.5, 0.5, 0.55, paint.Hex(0xFFF59D), 0.3)
	tilt := 0.05 * math.Sin(t*0.8)
	p.Crescent(0.5, 0.5+tilt, 0.32, 0.14, -0.06, 0.28, moon)
	// craters
	p.CircleA(0.36, 0.42+tilt, 0.04, paint.Hex(0xFBC02D), 1)
	p.CircleA(0.42, 0.62+tilt, 0.03, paint.Hex(0xFBC02D), 1)
	p.CircleA(0.33, 0.55+tilt, 0.02, paint.Hex(0xFBC02D), 1)
	// a sleeping face: closed eye and a small smile
	p.Arc(0.40, 0.48+tilt, 0.03, 0.2, math.Pi-0.2, 0.01, paint.Hex(0x5D4037))
	p.Arc(0.37, 0.58+tilt, 0.03, 0.4, math.Pi-0.4, 0.01, paint.Hex(0x5D4037))
	// zzz floating off
	for i := 0; i < 3; i++ {
		k := math.Mod(t*0.5+float64(i)/3, 1)
		x := 0.62 + k*0.2
		y := 0.35 - k*0.25
		s := 0.05 + k*0.05
		p.Text(x, y, s, "Z", paint.Mix(paint.Black, paint.LightGrey, 1-k))
	}
}

func drawOcean(p *paint.Painter, t, dt float64) {
	layers := []struct {
		c     paint.RGB
		base  float64
		amp   float64
		freq  float64
		speed float64
		foam  bool
	}{
		{paint.Hex(0x0D47A1), 0.48, 0.05, 1.5, 1.2, false},
		{paint.Hex(0x1976D2), 0.58, 0.06, 2.0, -1.6, false},
		{paint.Hex(0x29B6F6), 0.68, 0.07, 1.2, 2.2, true},
	}
	w := p.R - p.L
	for _, l := range layers {
		l := l
		p.Shade(p.L, l.base-l.amp-0.02, w, 1-(l.base-l.amp-0.02), func(u, v float64) (paint.RGB, float64) {
			x := u * w * 2 * math.Pi * l.freq
			h := l.base + l.amp*math.Sin(x+t*l.speed) + l.amp*0.4*math.Sin(x*2.3-t*l.speed*1.7)
			y := (l.base - l.amp - 0.02) + v*(1-(l.base-l.amp-0.02))
			if y < h {
				return l.c, 0
			}
			if l.foam && y-h < 0.02 && math.Sin(x+t*l.speed) > 0.5 {
				return paint.White, 1
			}
			return l.c, 1
		})
	}
	// the big curling crest travelling across
	k := cycle(t, 4)
	cx := p.L + w*k
	p.Arc(cx, 0.50, 0.14, math.Pi*0.9, math.Pi*1.9, 0.05, paint.Hex(0x29B6F6))
	p.Arc(cx, 0.50, 0.14, math.Pi*1.05, math.Pi*1.75, 0.02, paint.White)
	p.Circle(cx+0.13, 0.55, 0.03, paint.White)
	p.Circle(cx+0.17, 0.59, 0.02, paint.White)
	// sun glitter
	for i := 0; i < 12; i++ {
		x := p.L + w*noise(i*3)
		y := 0.55 + 0.25*noise(i*3+1)
		b := 0.5 + 0.5*math.Sin(t*4+float64(i))
		if b > 0.8 {
			p.Dot(x, y, 0.01, paint.White)
		}
	}
}

func drawRainbow(p *paint.Painter, t, dt float64) {
	bands := []paint.RGB{paint.Hex(0xE53935), paint.Hex(0xFF9800), paint.Hex(0xFFEB3B), paint.Hex(0x43A047), paint.Hex(0x1E88E5), paint.Hex(0x3949AB), paint.Hex(0x8E24AA)}
	const period = 7.0
	k := math.Mod(t, period)
	shift := 0.0
	if k > 3 {
		shift = (k - 3) * 60
	}
	fade := 1.0
	if k > period-0.8 {
		fade = 1 - (k-(period-0.8))/0.8
	}
	for i, c := range bands {
		prog := smooth((k - float64(i)*0.15) / 2.0)
		if prog <= 0 {
			continue
		}
		r := 0.46 - float64(i)*0.045
		col := c
		if shift > 0 {
			col = paint.HSL(float64(i)*51+shift, 0.85, 0.55)
		}
		p.Arc(0.5, 0.82, r, math.Pi, math.Pi+prog*math.Pi, 0.045, col.Scale(fade))
	}
	// clouds at the feet
	cloud(p, 0.14, 0.80, 0.42, paint.Hex(0xECEFF1))
	cloud(p, 0.86, 0.80, 0.42, paint.Hex(0xECEFF1))
	// sparkles drifting along the top band
	for i := 0; i < 5; i++ {
		a := math.Pi + math.Mod(t*0.4+float64(i)*0.2, 1)*math.Pi
		x, y := 0.5+0.50*math.Cos(a), 0.82+0.50*math.Sin(a)
		p.Star(x, y, 0.025, 0.01, 4, t*3, paint.White.Scale(fade))
	}
}

type bolt struct{ sparks *paint.System }

func (b *bolt) Draw(p *paint.Painter, t, dt float64) {
	flash := 0.6 + 0.4*math.Abs(math.Sin(t*9))*(0.5+0.5*math.Sin(t*2.1))
	shake := 0.006 * math.Sin(t*40) * flash
	yellow := paint.Hex(0xFFD600)
	p.Glow(0.5, 0.5, 0.5, paint.Hex(0xFFF176), 0.6*flash)
	pts := []float64{0.62, 0.05, 0.30, 0.52, 0.48, 0.52, 0.38, 0.95, 0.72, 0.42, 0.54, 0.42}
	for i := 0; i < len(pts); i += 2 {
		pts[i] += shake
	}
	p.Poly(paint.Mix(yellow, paint.White, 0.3*flash), pts...)
	// inner highlight
	p.Line(0.55+shake, 0.15, 0.40+shake, 0.45, 0.02, paint.White)
	b.sparks.Every(dt, 20*flash, func() {
		b.sparks.Emit(paint.Particle{X: 0.38, Y: 0.95, VX: b.sparks.Rand(-0.4, 0.4), VY: b.sparks.Rand(-0.5, -0.1), Life: 0.5, Size: 0.012})
	})
	b.sparks.Step(dt, 1.5, 0)
	for _, q := range b.sparks.P {
		p.Dot(q.X, q.Y, q.Size, paint.Mix(yellow, paint.White, 1-q.T()))
	}
}

type flower struct{ petals *paint.System }

func (f *flower) Draw(p *paint.Painter, t, dt float64) {
	const period = 8.0
	k := math.Mod(t, period)
	bloom := smooth(k / 2.5)
	if k > period-1 {
		bloom *= 1 - smooth(k-(period-1))
	}
	sway := 0.04 * math.Sin(t*1.5)
	cx, cy := 0.5+sway, 0.42
	// stem and leaf
	p.Line(0.5, 0.95, cx, cy, 0.03, paint.Hex(0x388E3C))
	p.Ellipse(0.42+sway*0.5, 0.75, 0.09, 0.035, paint.Hex(0x66BB6A))
	pink := paint.Hex(0xF48FB1)
	for i := 0; i < 5; i++ {
		a := -math.Pi/2 + float64(i)*2*math.Pi/5 + sway + 0.05*math.Sin(t*3+float64(i))
		d := 0.16 * bloom
		r := 0.11 * bloom
		px, py := cx+d*math.Cos(a), cy+d*math.Sin(a)
		p.Circle(px, py, r, pink)
		p.Circle(px, py, r*0.6, paint.Mix(pink, paint.White, 0.3))
	}
	p.Circle(cx, cy, 0.07*bloom+0.02, paint.Hex(0xFFEB3B))
	// falling petals once bloomed
	if bloom >= 1 {
		f.petals.Every(dt, 0.8, func() {
			f.petals.Emit(paint.Particle{X: cx + f.petals.Rand(-0.2, 0.2), Y: cy, VY: 0.12, VX: f.petals.Rand(-0.05, 0.05), Life: 4, Size: 0.03, Spin: f.petals.Rand(0, 6)})
		})
	}
	f.petals.Step(dt, 0, 0)
	for _, q := range f.petals.P {
		x := q.X + 0.05*math.Sin(q.Age*2+q.Spin)
		p.Ellipse(x, q.Y, q.Size, q.Size*0.5, paint.Mix(pink, paint.Black, q.T()*0.6))
	}
}

func drawSnowflake(p *paint.Painter, t, dt float64) {
	// drifting flurry behind
	for i := 0; i < 30; i++ {
		k := math.Mod(t*0.1*(0.5+noise(i))+noise(i+100), 1)
		x := p.L + (p.R-p.L)*noise(i+50) + 0.03*math.Sin(t+float64(i))
		y := p.T + (p.Bt-p.T)*k
		p.Dot(x, y, 0.012, paint.Mix(paint.DarkGrey, paint.White, 0.5))
	}
	ice := paint.Hex(0xB3E5FC)
	rot := t * 0.5
	cx, cy := 0.5, 0.5
	for i := 0; i < 6; i++ {
		a := rot + float64(i)*math.Pi/3
		x1, y1 := cx+0.38*math.Cos(a), cy+0.38*math.Sin(a)
		p.Line(cx, cy, x1, y1, 0.035, ice)
		for _, f := range []float64{0.45, 0.7} {
			bx, by := cx+0.38*f*math.Cos(a), cy+0.38*f*math.Sin(a)
			for _, s := range []float64{-1, 1} {
				ba := a + s*math.Pi/3
				l := 0.12 * (1 - f*0.5)
				p.Line(bx, by, bx+l*math.Cos(ba), by+l*math.Sin(ba), 0.025, ice)
			}
		}
		p.Circle(x1, y1, 0.03, paint.White)
	}
	p.Circle(cx, cy, 0.05, paint.White)
	p.Glow(cx, cy, 0.2, paint.White, 0.3+0.2*math.Sin(t*3))
}

type tornado struct{ debris *paint.System }

func (tn *tornado) Draw(p *paint.Painter, t, dt float64) {
	// stacked ellipses, each swaying with a phase lag
	const n = 14
	for i := 0; i < n; i++ {
		v := float64(i) / (n - 1)
		y := 0.12 + v*0.74
		w := 0.42*(1-v) + 0.04
		x := 0.5 + math.Sin(t*5-v*4)*0.06*(v+0.2)
		shade := paint.Mix(paint.Hex(0x90A4AE), paint.Hex(0x546E7A), 0.5+0.5*math.Sin(t*10-float64(i)*1.2))
		p.Ellipse(x, y, w, 0.045, shade)
	}
	tn.debris.Every(dt, 6, func() {
		tn.debris.Emit(paint.Particle{X: 0.5, Y: tn.debris.Rand(0.2, 0.85), Life: 3, Spin: tn.debris.Rand(0, 6), Size: tn.debris.Rand(0.01, 0.025), Kind: tn.debris.Rng.Intn(3)})
	})
	tn.debris.Step(dt, 0, 0)
	cols := []paint.RGB{paint.Hex(0x8D6E63), paint.Hex(0x66BB6A), paint.Hex(0xFFCA28)}
	for _, q := range tn.debris.P {
		v := (q.Y - 0.12) / 0.74
		r := (0.42*(1-v) + 0.08) * (1 + q.Age*0.5)
		a := q.Spin + q.Age*6
		x := 0.5 + r*math.Cos(a)
		y := q.Y - q.Age*0.05 + 0.02*math.Sin(a)
		p.Dot(x, y, q.Size, cols[q.Kind])
	}
	// dust at the foot
	p.Ellipse(0.5, 0.88, 0.2+0.03*math.Sin(t*8), 0.03, paint.Hex(0x795548))
}

func drawSeedling(p *paint.Painter, t, dt float64) {
	k := cycle(t, 6)
	grow := smooth(k * 1.4)
	green := paint.Hex(0x66BB6A)
	dark := paint.Hex(0x2E7D32)
	// soil
	p.Ellipse(0.5, 0.85, 0.32, 0.06, paint.Hex(0x5D4037))
	p.Ellipse(0.5, 0.84, 0.28, 0.04, paint.Hex(0x795548))
	// stem grows upward with a gentle sway
	top := 0.84 - 0.45*grow
	sway := 0.03 * math.Sin(t*2) * grow
	p.Line(0.5, 0.84, 0.5+sway, top, 0.03, dark)
	// two leaves unfold
	if grow > 0.3 {
		l := smooth((grow - 0.3) / 0.5)
		for _, s := range []float64{-1, 1} {
			lx := 0.5 + sway + s*0.12*l
			ly := top + 0.06
			p.Ellipse(lx, ly-0.02*l, 0.12*l, 0.05*l, green)
			p.Line(0.5+sway, top+0.05, lx, ly-0.02*l, 0.01, dark)
		}
	}
	// a bud at the top
	if grow > 0.9 {
		b := (grow - 0.9) / 0.1
		p.Circle(0.5+sway, top-0.02, 0.04*b, paint.Hex(0x9CCC65))
	}
	// sparkles of growth
	for i := 0; i < 4; i++ {
		s := math.Mod(t*0.7+float64(i)*0.25, 1)
		p.Dot(0.5+sway+0.2*math.Cos(float64(i)*1.6+t), 0.84-0.5*s, 0.012, paint.Mix(paint.Black, paint.Gold, 1-s))
	}
}

type volcano struct{ lava *paint.System }

func (v *volcano) Draw(p *paint.Painter, t, dt float64) {
	rock := paint.Hex(0x5D4037)
	p.Poly(rock, 0.10, 0.90, 0.38, 0.32, 0.62, 0.32, 0.90, 0.90)
	p.Poly(paint.Hex(0x4E342E), 0.50, 0.32, 0.62, 0.32, 0.90, 0.90, 0.60, 0.90)
	burst := 0.6 + 0.4*math.Sin(t*0.7)
	v.lava.Every(dt, 30*burst, func() {
		v.lava.Emit(paint.Particle{X: v.lava.Rand(0.42, 0.58), Y: 0.32, VX: v.lava.Rand(-0.35, 0.35), VY: v.lava.Rand(-1.1, -0.6) * burst, Life: 1.6, Size: v.lava.Rand(0.012, 0.03), Kind: v.lava.Rng.Intn(2)})
	})
	v.lava.Step(dt, 1.2, 0)
	for _, q := range v.lava.P {
		c := paint.Mix(paint.Hex(0xFF6D00), paint.Hex(0xFFEB3B), 1-q.T())
		if q.Kind == 1 {
			c = paint.Mix(paint.Hex(0x616161), paint.Hex(0x3A3A3A), q.T())
		}
		p.Circle(q.X, q.Y, q.Size, c)
	}
	// glowing crater and lava rivers
	p.Glow(0.5, 0.32, 0.25, paint.Hex(0xFF6D00), 0.5+0.3*burst)
	p.Ellipse(0.5, 0.32, 0.13, 0.03, paint.Hex(0xFF3D00))
	for i, s := range []float64{-1, 1} {
		l := 0.5 + 0.5*math.Sin(t*0.9+float64(i)*2)
		p.Line(0.5+s*0.06, 0.34, 0.5+s*(0.06+0.22*l), 0.34+0.5*l, 0.03, paint.Hex(0xFF5722))
	}
	// smoke
	for i := 0; i < 5; i++ {
		k := math.Mod(t*0.3+float64(i)/5, 1)
		p.CircleA(0.5+0.08*math.Sin(k*6+float64(i)), 0.28-k*0.28, 0.03+k*0.06, paint.Hex(0x9E9E9E), (1-k)*0.5)
	}
}
