package glyph

import (
	"fmt"
	"math"
	"time"

	"github.com/Cid-Emmerich/SopeBox/internal/paint"
)

func initobjects() {
	add("objects",
		Def{Emoji: "🚀", Name: "rocket", Blurb: "a rocket riding its exhaust", Aliases: []string{"launch", "space", "ship", "blast off", "to the moon"}, New: func() Anim { return &rocket{exhaust: paint.NewSystem(61)} }},
		Def{Emoji: "🎉", Name: "party", Blurb: "a party popper firing confetti", Aliases: []string{"confetti", "celebrate", "celebration", "tada", "popper", "🎊"}, New: func() Anim { return &party{confetti: paint.NewSystem(62)} }},
		Def{Emoji: "⏰", Name: "clock", Blurb: "an alarm clock showing the real time, ringing now and then", Aliases: []string{"alarm", "time", "alarm clock", "🕐", "⏱"}, New: stateless(drawClock)},
		Def{Emoji: "⌛", Name: "hourglass", Blurb: "sand runs out, then it flips", Aliases: []string{"⏳", "timer", "sand", "wait", "loading"}, New: func() Anim { return &hourglass{grains: paint.NewSystem(63)} }},
		Def{Emoji: "⛄", Name: "snowman", Blurb: "a snowman waving in the snow", Aliases: []string{"☃️", "winter", "frosty", "snow man"}, New: stateless(drawSnowman)},
		Def{Emoji: "🎈", Name: "balloon", Blurb: "a balloon bobbing on its string", Aliases: []string{"birthday", "float", "helium"}, New: stateless(drawBalloon)},
		Def{Emoji: "💡", Name: "bulb", Blurb: "a light bulb having an idea", Aliases: []string{"idea", "light", "lightbulb", "light bulb", "lamp"}, New: stateless(drawBulb)},
		Def{Emoji: "💣", Name: "bomb", Blurb: "a fuse burns down and... boom", Aliases: []string{"explode", "explosion", "boom", "💥", "kaboom"}, New: func() Anim { return &bomb{sparks: paint.NewSystem(64)} }},
		Def{Emoji: "🎵", Name: "music", Blurb: "notes bouncing to the beat", Aliases: []string{"🎶", "note", "notes", "song", "melody", "tune"}, New: func() Anim { return &music{notes: paint.NewSystem(65)} }},
		Def{Emoji: "🎯", Name: "target", Blurb: "arrows thud into the bullseye", Aliases: []string{"bullseye", "dart", "darts", "archery", "aim", "goal"}, New: stateless(drawTarget)},
		Def{Emoji: "🎲", Name: "dice", Blurb: "a die tumbling to a random face", Aliases: []string{"die", "roll", "game", "random", "chance"}, New: func() Anim { return &dice{sys: paint.NewSystem(66)} }},
		Def{Emoji: "🎁", Name: "gift", Blurb: "a present that can't keep its lid on", Aliases: []string{"present", "box", "birthday", "surprise", "wrapped"}, New: func() Anim { return &gift{sparks: paint.NewSystem(67)} }},
		Def{Emoji: "🔋", Name: "battery", Blurb: "charging to one hundred percent", Aliases: []string{"charge", "charging", "power", "energy", "🪫"}, New: stateless(drawBattery)},
		Def{Emoji: "🎂", Name: "cake", Blurb: "a birthday cake with flickering candles", Aliases: []string{"birthday", "birthday cake", "candles", "🧁", "dessert"}, New: stateless(drawCake)},
		Def{Emoji: "🔄", Name: "refresh", Blurb: "arrows chasing each other", Aliases: []string{"reload", "sync", "loading", "spinner", "🔁", "🔃"}, New: stateless(drawRefresh)},
		Def{Emoji: "📡", Name: "signal", Blurb: "a dish broadcasting rings", Aliases: []string{"satellite", "antenna", "broadcast", "wifi", "📶", "radar"}, New: stateless(drawSignal)},
		Def{Emoji: "🕯️", Name: "candle", Blurb: "a candle flame in a draught", Aliases: []string{"flame", "light", "wax", "vigil"}, New: stateless(drawCandle)},
		Def{Emoji: "⚙️", Name: "gears", Blurb: "cogs meshing", Aliases: []string{"gear", "cog", "settings", "machine", "engine"}, New: stateless(drawGears)},
	)
}

// ---------------------------------------------------------------------------

type rocket struct{ exhaust *paint.System }

func (r *rocket) Draw(p *paint.Painter, t, dt float64) {
	// stars streaming downward suggest motion
	for i := 0; i < 30; i++ {
		sp := 0.3 + noise(i)*0.6
		y := p.T + math.Mod(noise(i+9)*3+t*sp, p.Bt-p.T)
		x := p.L + (p.R-p.L)*noise(i+31)
		p.Line(x, y, x, y+0.02*sp, 0.008, paint.Mix(paint.DarkGrey, paint.White, noise(i+3)))
	}
	boost := math.Max(0, math.Sin(t*0.9))
	shake := 0.006 * math.Sin(t*50)
	cx := 0.5 + shake
	cy := 0.45 - 0.05*boost + 0.01*math.Sin(t*3)
	// exhaust
	r.exhaust.Every(dt, 60+60*boost, func() {
		r.exhaust.Emit(paint.Particle{X: cx + r.exhaust.Rand(-0.04, 0.04), Y: cy + 0.30, VX: r.exhaust.Rand(-0.15, 0.15), VY: r.exhaust.Rand(0.5, 0.9) * (1 + boost), Life: r.exhaust.Rand(0.4, 0.8), Size: r.exhaust.Rand(0.02, 0.05)})
	})
	r.exhaust.Step(dt, 0, 1)
	for _, q := range r.exhaust.P {
		k := q.T()
		c := paint.Mix3(paint.White, paint.Hex(0xFF9800), paint.Hex(0x616161), k)
		p.CircleA(q.X, q.Y, q.Size*(0.6+k), c, 1-k*0.7)
	}
	// flame core
	fl := 0.10 + 0.05*boost + 0.02*math.Sin(t*30)
	p.Triangle(cx-0.06, cy+0.30, cx+0.06, cy+0.30, cx, cy+0.30+fl*1.6, paint.Hex(0xFF9800))
	p.Triangle(cx-0.03, cy+0.30, cx+0.03, cy+0.30, cx, cy+0.30+fl, paint.Hex(0xFFEB3B))
	// fins, body, nose, window
	red := paint.Hex(0xE53935)
	p.Triangle(cx-0.10, cy+0.30, cx-0.10, cy+0.05, cx-0.22, cy+0.32, red)
	p.Triangle(cx+0.10, cy+0.30, cx+0.10, cy+0.05, cx+0.22, cy+0.32, red)
	p.RoundRect(cx-0.11, cy-0.20, 0.22, 0.50, 0.06, paint.Hex(0xECEFF1))
	p.Rect(cx+0.04, cy-0.15, 0.06, 0.42, paint.Hex(0xCFD8DC))
	p.Triangle(cx-0.11, cy-0.18, cx+0.11, cy-0.18, cx, cy-0.42, red)
	p.Circle(cx, cy, 0.06, paint.Hex(0x37474F))
	p.Circle(cx, cy, 0.045, paint.Hex(0x4FC3F7))
	p.Circle(cx-0.015, cy-0.015, 0.012, paint.White)
}

// ---------------------------------------------------------------------------

type party struct {
	confetti *paint.System
	last     int
}

func (pt *party) Draw(p *paint.Painter, t, dt float64) {
	const period = 1.6
	n := int(t / period)
	x := t - float64(n)*period
	if n != pt.last {
		pt.last = n
		for i := 0; i < 40; i++ {
			a := -math.Pi/4 + pt.confetti.Rand(-0.5, 0.5)
			sp := pt.confetti.Rand(0.9, 1.8)
			pt.confetti.Emit(paint.Particle{X: 0.44, Y: 0.56, VX: sp * math.Cos(a), VY: sp * math.Sin(a), Life: pt.confetti.Rand(1.2, 2.0), Size: pt.confetti.Rand(0.015, 0.035), Kind: i % 5, Spin: pt.confetti.Rand(0, 6)})
		}
	}
	pt.confetti.Step(dt, 1.4, 1.2)
	cols := []paint.RGB{paint.Hex(0xFF5252), paint.Hex(0xFFD740), paint.Hex(0x40C4FF), paint.Hex(0x69F0AE), paint.Hex(0xE040FB)}
	for _, q := range pt.confetti.P {
		k := 1 - q.T()
		an := q.Spin + q.Age*8
		s := q.Size * (0.5 + k)
		if q.Kind%2 == 0 {
			p.Line(q.X-s*math.Cos(an), q.Y-s*math.Sin(an), q.X+s*math.Cos(an), q.Y+s*math.Sin(an), 0.015, cols[q.Kind])
		} else {
			p.Circle(q.X, q.Y, s*0.5, cols[q.Kind])
		}
	}
	// the cone, recoiling on each pop
	kick := 0.03 * math.Exp(-6*x)
	cx, cy := 0.44-kick, 0.56+kick
	ang := -math.Pi / 4
	pt2 := func(u, v float64) (float64, float64) { return paint.Rot(cx, cy, cx+u, cy+v, ang) }
	x0, y0 := pt2(0, 0)
	x1, y1 := pt2(-0.14, 0.42)
	x2, y2 := pt2(0.14, 0.42)
	p.Triangle(x0, y0, x1, y1, x2, y2, paint.Hex(0xFFA000))
	for i := 1; i < 4; i++ {
		f := float64(i) / 4
		ax, ay := pt2(-0.14*f, 0.42*f)
		bx, by := pt2(0.14*f, 0.42*f)
		p.Line(ax, ay, bx, by, 0.02, paint.Hex(0xE53935))
	}
	ex, ey := pt2(0, 0.42)
	p.Ellipse(ex, ey, 0.14, 0.05, paint.Hex(0xFFD54F))
	// burst rays right after the pop
	if x < 0.25 {
		k := 1 - x/0.25
		for i := 0; i < 6; i++ {
			a := -math.Pi/4 + (float64(i)-2.5)*0.25
			p.Line(cx+0.05*math.Cos(a), cy+0.05*math.Sin(a), cx+(0.10+0.2*(1-k))*math.Cos(a), cy+(0.10+0.2*(1-k))*math.Sin(a), 0.015*k, paint.Gold)
		}
	}
}

// ---------------------------------------------------------------------------

func drawClock(p *paint.Painter, t, dt float64) {
	now := time.Now()
	ring := math.Mod(t, 6) < 1.2
	shake := 0.0
	if ring {
		shake = 0.015 * math.Sin(t*60)
	}
	cx, cy := 0.5+shake, 0.52
	red := paint.Hex(0xE53935)
	// legs and bells
	p.Line(cx-0.18, cy+0.36, cx-0.28, cy+0.46, 0.035, paint.DarkGrey)
	p.Line(cx+0.18, cy+0.36, cx+0.28, cy+0.46, 0.035, paint.DarkGrey)
	bell := 0.0
	if ring {
		bell = 0.02 * math.Sin(t*70)
	}
	p.Arc(cx-0.24+bell, cy-0.30, 0.10, math.Pi*0.75, math.Pi*1.85, 0.06, red)
	p.Arc(cx+0.24-bell, cy-0.30, 0.10, math.Pi*1.15, math.Pi*2.25, 0.06, red)
	p.Line(cx-0.20, cy-0.28, cx-0.13, cy-0.22, 0.035, paint.DarkGrey)
	p.Line(cx+0.20, cy-0.28, cx+0.13, cy-0.22, 0.035, paint.DarkGrey)
	// face
	p.Circle(cx, cy, 0.36, red)
	p.Circle(cx, cy, 0.31, paint.Hex(0xFAFAFA))
	for i := 0; i < 12; i++ {
		a := float64(i) * math.Pi / 6
		l := 0.03
		if i%3 == 0 {
			l = 0.05
		}
		p.Line(cx+0.28*math.Cos(a), cy+0.28*math.Sin(a), cx+(0.28-l)*math.Cos(a), cy+(0.28-l)*math.Sin(a), 0.015, paint.DarkGrey)
	}
	h := float64(now.Hour()%12) + float64(now.Minute())/60
	m := float64(now.Minute()) + float64(now.Second())/60
	s := float64(now.Second()) + float64(now.Nanosecond())/1e9
	// ticking second hand: snaps each second with a tiny overshoot
	sf := s - math.Floor(s)
	sa := (math.Floor(s) + smooth(sf*6)) / 60 * 2 * math.Pi
	ha := h / 12 * 2 * math.Pi
	ma := m / 60 * 2 * math.Pi
	hand := func(a, l, th float64, c paint.RGB) {
		p.Line(cx, cy, cx+l*math.Sin(a), cy-l*math.Cos(a), th, c)
	}
	hand(ha, 0.16, 0.035, paint.Hex(0x263238))
	hand(ma, 0.24, 0.03, paint.Hex(0x263238))
	hand(sa, 0.26, 0.012, red)
	p.Circle(cx, cy, 0.025, red)
	if ring {
		for i := 0; i < 2; i++ {
			r := 0.42 + float64(i)*0.06
			c := paint.Mix(paint.Black, paint.Gold, 0.8-float64(i)*0.3)
			p.Arc(cx, cy, r, -math.Pi*0.9, -math.Pi*0.6, 0.015, c)
			p.Arc(cx, cy, r, -math.Pi*0.4, -math.Pi*0.1, 0.015, c)
		}
	}
	p.TextCentred(cx, 0.94, 0.05, now.Format("15:04:05"), paint.Grey)
}

// ---------------------------------------------------------------------------

type hourglass struct{ grains *paint.System }

func (h *hourglass) Draw(p *paint.Painter, t, dt float64) {
	const run, flip = 6.0, 0.8
	k := math.Mod(t, run+flip)
	f := math.Min(k/run, 1)
	rot := 0.0
	if k > run {
		rot = smooth((k-run)/flip) * math.Pi
	}
	cx, cy := 0.5, 0.5
	q := func(x, y float64) (float64, float64) { return paint.Rot(cx, cy, x, y, rot) }
	glass := paint.Hex(0x80DEEA)
	wood := paint.Hex(0x8D6E63)
	sand := paint.Hex(0xFFCA28)
	// glass outline: two triangles meeting at the waist
	tri := func(top bool, c paint.RGB, th float64) {
		y0, y1 := 0.16, 0.5
		if !top {
			y0, y1 = 0.84, 0.5
		}
		ax, ay := q(0.28, y0)
		bx, by := q(0.72, y0)
		wx, wy := q(0.5, y1)
		p.Line(ax, ay, wx, wy, th, c)
		p.Line(bx, by, wx, wy, th, c)
	}
	// sand fills: top drains from the top, bottom heaps from the bottom
	fillTri := func(top bool, level float64) {
		// level 0..1 of the triangle height that contains sand
		if level <= 0 {
			return
		}
		var pts []float64
		if top {
			// the sand surface sits at height (1-level) below the cap, with a dip in the middle
			y := lerp(0.5, 0.18, level)
			half := 0.21 * (0.5 - y) / 0.32
			pts = []float64{0.5 - half, y, 0.5, y + 0.03*level, 0.5 + half, y, 0.5, 0.5}
		} else {
			y := lerp(0.82, 0.52, level)
			half := 0.21 * (y - 0.5) / 0.32
			pts = []float64{0.5 - half, y, 0.5, y - 0.04*level, 0.5 + half, y, 0.72, 0.82, 0.28, 0.82}
		}
		var rp []float64
		for i := 0; i < len(pts); i += 2 {
			x, y := q(pts[i], pts[i+1])
			rp = append(rp, x, y)
		}
		p.Poly(sand, rp...)
	}
	fillTri(true, 1-f)
	fillTri(false, f)
	// falling stream
	if f < 1 && rot == 0 {
		p.Line(0.5, 0.5, 0.5, 0.80-0.28*f, 0.012, sand)
		h.grains.Every(dt, 12, func() {
			h.grains.Emit(paint.Particle{X: 0.5 + h.grains.Rand(-0.006, 0.006), Y: 0.5, VY: 0.6, Life: 0.5, Size: 0.008})
		})
	}
	h.grains.Step(dt, 0.8, 0)
	for _, g := range h.grains.P {
		if g.Y > 0.80-0.28*f {
			continue
		}
		p.Dot(g.X, g.Y, g.Size, sand)
	}
	tri(true, glass, 0.03)
	tri(false, glass, 0.03)
	for _, y := range []float64{0.13, 0.87} {
		ax, ay := q(0.24, y)
		bx, by := q(0.76, y)
		p.Line(ax, ay, bx, by, 0.05, wood)
	}
	for _, x := range []float64{0.25, 0.75} {
		ax, ay := q(x, 0.13)
		bx, by := q(x, 0.87)
		p.Line(ax, ay, bx, by, 0.025, wood)
	}
}

// ---------------------------------------------------------------------------

func drawSnowman(p *paint.Painter, t, dt float64) {
	for i := 0; i < 40; i++ {
		k := math.Mod(t*0.08*(0.5+noise(i))+noise(i+100), 1)
		x := p.L + (p.R-p.L)*noise(i+50) + 0.04*math.Sin(t*0.8+float64(i))
		y := p.T + (p.Bt-p.T)*k
		p.Dot(x, y, 0.012, paint.Mix(paint.DarkGrey, paint.White, 0.6))
	}
	white := paint.Hex(0xF5F5F5)
	p.Ellipse(0.5, 0.94, 0.36, 0.04, paint.Hex(0xCFD8DC))
	p.Circle(0.5, 0.74, 0.22, white)
	p.Circle(0.5, 0.46, 0.17, white)
	p.Circle(0.5, 0.24, 0.13, white)
	coal := paint.Hex(0x212121)
	for _, y := range []float64{0.68, 0.76, 0.84} {
		p.Circle(0.5, y, 0.02, coal)
	}
	p.Circle(0.5, 0.42, 0.017, coal)
	p.Circle(0.5, 0.50, 0.017, coal)
	// face
	b := blink(t, 5, 0.3)
	p.Ellipse(0.455, 0.21, 0.018, 0.022*b+0.003, coal)
	p.Ellipse(0.545, 0.21, 0.018, 0.022*b+0.003, coal)
	p.Triangle(0.5, 0.245, 0.5, 0.275, 0.62, 0.27, paint.Hex(0xFF7043))
	for i := 0; i < 5; i++ {
		a := 0.45 + float64(i)*0.55
		p.Circle(0.5+0.075*math.Cos(a), 0.26+0.06*math.Sin(a), 0.008, coal)
	}
	// hat
	p.Rect(0.36, 0.12, 0.28, 0.03, coal)
	p.Rect(0.41, 0.0, 0.18, 0.13, coal)
	p.Rect(0.41, 0.09, 0.18, 0.03, paint.Hex(0xE53935))
	// scarf with a flapping tail
	p.Ellipse(0.5, 0.35, 0.14, 0.04, paint.Hex(0xE53935))
	tail := 0.04 * math.Sin(t*3)
	p.Poly(paint.Hex(0xE53935), 0.56, 0.35, 0.62, 0.35, 0.66+tail, 0.50, 0.58+tail, 0.50)
	// stick arms: the right one waves
	p.Line(0.35, 0.46, 0.14, 0.36, 0.02, paint.Hex(0x6D4C41))
	wa := -0.5 + 0.35*math.Sin(t*4)
	ex, ey := 0.65+0.22*math.Cos(wa), 0.46+0.22*math.Sin(wa)
	p.Line(0.65, 0.46, ex, ey, 0.02, paint.Hex(0x6D4C41))
	p.Line(ex, ey, ex+0.04*math.Cos(wa-0.6), ey+0.04*math.Sin(wa-0.6), 0.015, paint.Hex(0x6D4C41))
	p.Line(ex, ey, ex+0.04*math.Cos(wa+0.6), ey+0.04*math.Sin(wa+0.6), 0.015, paint.Hex(0x6D4C41))
}

func drawBalloon(p *paint.Painter, t, dt float64) {
	cx := 0.5 + 0.06*math.Sin(t*0.9)
	cy := 0.36 + 0.03*math.Sin(t*1.7)
	tilt := 0.15 * math.Sin(t*0.9)
	red := paint.Hex(0xE53935)
	p.Ellipse(cx, cy, 0.20, 0.26, red)
	p.Ellipse(cx-0.07, cy-0.10, 0.045, 0.07, paint.Mix(red, paint.White, 0.5))
	kx, ky := paint.Rot(cx, cy, cx, cy+0.28, tilt)
	p.Triangle(kx-0.03, ky+0.02, kx+0.03, ky+0.02, kx, ky-0.03, red)
	// string: a chain of short segments swaying with a lag
	x, y := kx, ky+0.02
	for i := 0; i < 12; i++ {
		f := float64(i) / 12
		nx := x + 0.02*math.Sin(t*2-f*4)
		ny := y + 0.04
		p.Line(x, y, nx, ny, 0.008, paint.LightGrey)
		x, y = nx, ny
	}
	// a few drifting sparkles
	for i := 0; i < 4; i++ {
		k := math.Mod(t*0.25+float64(i)/4, 1)
		p.Dot(cx+0.4*math.Cos(float64(i)*1.6), 0.9-k*0.8, 0.012, paint.Mix(paint.Black, paint.LightGrey, math.Sin(k*math.Pi)))
	}
}

func drawBulb(p *paint.Painter, t, dt float64) {
	const period = 4.0
	x := math.Mod(t, period)
	// off for a second, flicker, then on
	on := 0.0
	switch {
	case x < 1:
		on = 0
	case x < 1.5:
		if math.Sin(x*60) > 0.2 {
			on = 1
		}
	case x < period-0.3:
		on = 1
	default:
		on = 0
	}
	glass := paint.Mix(paint.Hex(0x546E7A), paint.Hex(0xFFEE58), on)
	if on > 0 {
		p.Glow(0.5, 0.42, 0.55, paint.Hex(0xFFF176), 0.7)
		for i := 0; i < 8; i++ {
			a := float64(i)*math.Pi/4 + 0.3*math.Sin(t*2)
			l := 0.36 + 0.03*math.Sin(t*5+float64(i))
			p.Line(0.5+0.30*math.Cos(a), 0.42+0.30*math.Sin(a), 0.5+l*math.Cos(a), 0.42+l*math.Sin(a), 0.02, paint.Hex(0xFFD600))
		}
	}
	p.Circle(0.5, 0.40, 0.22, glass)
	p.Poly(glass, 0.36, 0.52, 0.64, 0.52, 0.57, 0.66, 0.43, 0.66)
	// filament
	fil := paint.Mix(paint.Hex(0x37474F), paint.Hex(0xFF6F00), on)
	p.Line(0.45, 0.62, 0.45, 0.48, 0.015, fil)
	p.Line(0.55, 0.62, 0.55, 0.48, 0.015, fil)
	p.Line(0.45, 0.48, 0.48, 0.42, 0.015, fil)
	p.Line(0.48, 0.42, 0.52, 0.48, 0.015, fil)
	p.Line(0.52, 0.48, 0.55, 0.42, 0.015, fil)
	p.Line(0.55, 0.42, 0.55, 0.48, 0.015, fil)
	// screw base
	for i := 0; i < 3; i++ {
		y := 0.68 + float64(i)*0.05
		p.RoundRect(0.42, y, 0.16, 0.035, 0.01, paint.Hex(0x9E9E9E))
	}
	p.RoundRect(0.45, 0.83, 0.10, 0.04, 0.02, paint.Hex(0x616161))
	if on > 0 && x > 1.5 {
		p.TextCentred(0.5, 0.06+0.01*math.Sin(t*4), 0.07, "IDEA!", paint.Gold)
	}
}

// ---------------------------------------------------------------------------

type bomb struct{ sparks *paint.System }

func (b *bomb) Draw(p *paint.Painter, t, dt float64) {
	const fuse, boom = 3.5, 1.3
	k := math.Mod(t, fuse+boom)
	if k < fuse {
		f := k / fuse
		shake := 0.01 * f * f * math.Sin(t*45)
		cx, cy := 0.5+shake, 0.58
		p.Circle(cx, cy, 0.28, paint.Hex(0x212121))
		p.Ellipse(cx-0.10, cy-0.10, 0.06, 0.04, paint.Hex(0x616161))
		p.RoundRect(cx-0.06, cy-0.36, 0.12, 0.10, 0.02, paint.Hex(0x616161))
		// fuse: an arc that shortens
		a0, a1 := math.Pi*1.0, math.Pi*1.85
		end := lerp(a1, a0, f)
		p.Arc(cx+0.12, cy-0.34, 0.14, a0, end, 0.025, paint.Hex(0xA1887F))
		tx, ty := cx+0.12+0.14*math.Cos(end), cy-0.34+0.14*math.Sin(end)
		b.sparks.Every(dt, 40, func() {
			an := b.sparks.Rand(0, 2*math.Pi)
			b.sparks.Emit(paint.Particle{X: tx, Y: ty, VX: 0.4 * math.Cos(an), VY: 0.4 * math.Sin(an), Life: 0.35, Size: 0.012})
		})
		p.Glow(tx, ty, 0.06, paint.Gold, 0.9)
		b.sparks.Step(dt, 1, 0)
		for _, q := range b.sparks.P {
			p.Dot(q.X, q.Y, q.Size, paint.Mix(paint.Yellow, paint.Red, q.T()))
		}
		if f > 0.75 {
			p.TextCentred(0.5, 0.06, 0.09, fmt.Sprint(int(math.Ceil((1-f)*fuse))), paint.Red)
		}
		return
	}
	// explosion
	e := (k - fuse) / boom
	for i, c := range []paint.RGB{paint.Hex(0xFF3D00), paint.Hex(0xFF9100), paint.Hex(0xFFEA00), paint.White} {
		r := (0.55 - float64(i)*0.1) * smooth(e*2.5)
		al := 1 - smooth((e-0.4)/0.6)
		p.CircleA(0.5, 0.55, r, c, al)
	}
	if e > 0.15 {
		s := pop(e - 0.15)
		p.TextCentred(0.5, 0.5-0.11*s, 0.22*s, "BOOM", paint.Mix(paint.Black, paint.White, 1-smooth((e-0.6)/0.4)))
	}
	for i := 0; i < 12; i++ {
		a := float64(i) * math.Pi / 6
		d := 0.3 + e*0.5
		p.Line(0.5+d*math.Cos(a), 0.55+d*math.Sin(a), 0.5+(d+0.1)*math.Cos(a), 0.55+(d+0.1)*math.Sin(a), 0.02*(1-e), paint.Hex(0xFF9100))
	}
}

// ---------------------------------------------------------------------------

type music struct{ notes *paint.System }

func note(p *paint.Painter, x, y, s, tilt float64, c paint.RGB) {
	p.Ellipse(x, y, s*0.55, s*0.38, c)
	sx, sy := paint.Rot(x, y, x+s*0.45, y-s*1.5, tilt)
	p.Line(x+s*0.45, y, sx, sy, s*0.16, c)
	fx, fy := paint.Rot(x, y, x+s*0.95, y-s*0.9, tilt)
	p.Line(sx, sy, fx, fy, s*0.18, c)
}

func (m *music) Draw(p *paint.Painter, t, dt float64) {
	beat := math.Exp(-6 * math.Mod(t, 0.5))
	purple := paint.Hex(0x7E57C2)
	// equaliser bars along the bottom
	for i := 0; i < 9; i++ {
		x := 0.14 + float64(i)*0.09
		h := 0.05 + 0.18*math.Abs(math.Sin(t*3+float64(i)*0.9))*(0.6+0.4*beat)
		p.RoundRect(x-0.03, 0.92-h, 0.06, h, 0.02, paint.Mix(purple, paint.Pink, float64(i)/8))
	}
	// two beamed notes bouncing
	bob := 0.05 * beat
	tilt := 0.12 * math.Sin(t*2)
	x1, y1 := 0.36, 0.55-bob
	x2, y2 := 0.60, 0.50-bob
	s := 0.13
	p.Ellipse(x1, y1, s*0.55, s*0.38, purple)
	p.Ellipse(x2, y2, s*0.55, s*0.38, purple)
	ax, ay := paint.Rot(0.5, 0.5, x1+s*0.45, y1-s*2.1, tilt)
	bx, by := paint.Rot(0.5, 0.5, x2+s*0.45, y2-s*2.1, tilt)
	p.Line(x1+s*0.45, y1, ax, ay, s*0.16, purple)
	p.Line(x2+s*0.45, y2, bx, by, s*0.16, purple)
	p.Line(ax, ay, bx, by, s*0.3, purple)
	// floating small notes
	m.notes.Every(dt, 2, func() {
		m.notes.Emit(paint.Particle{X: m.notes.Rand(0.1, 0.9), Y: 0.8, VY: -0.18, VX: m.notes.Rand(-0.04, 0.04), Life: 3, Size: m.notes.Rand(0.04, 0.07), Spin: m.notes.Rand(0, 6)})
	})
	m.notes.Step(dt, 0, 0)
	for _, q := range m.notes.P {
		x := q.X + 0.04*math.Sin(q.Age*2+q.Spin)
		note(p, x, q.Y, q.Size, 0.2*math.Sin(q.Age*3), paint.Mix(paint.Black, paint.Pink, 1-q.T()))
	}
}

// ---------------------------------------------------------------------------

func drawTarget(p *paint.Painter, t, dt float64) {
	cx, cy := 0.5, 0.5
	red, white := paint.Hex(0xE53935), paint.Hex(0xFAFAFA)
	for i := 0; i < 5; i++ {
		c := red
		if i%2 == 1 {
			c = white
		}
		p.Circle(cx, cy, 0.40-float64(i)*0.08, c)
	}
	const period = 2.5
	k := math.Mod(t, period)
	// the arrow flies in from the top-left over 0.5s then wobbles in place
	f := smooth(k / 0.5)
	sx, sy := -0.3, -0.2
	ex, ey := cx, cy
	ax, ay := lerp(sx, ex, f), lerp(sy, ey, f)
	wob := 0.0
	if k > 0.5 {
		wob = 0.15 * math.Exp(-3*(k-0.5)) * math.Sin((k-0.5)*30)
	}
	fade := 1.0
	if k > period-0.4 {
		fade = 1 - (k-(period-0.4))/0.4
	}
	dirx, diry := math.Cos(math.Pi/4+wob), math.Sin(math.Pi/4+wob)
	l := 0.42
	tx, ty := ax-dirx*l, ay-diry*l
	shaft := paint.Mix(paint.Black, paint.Hex(0xA1887F), fade)
	p.Line(tx, ty, ax, ay, 0.025, shaft)
	// fletching
	for _, s := range []float64{-1, 1} {
		fx, fy := tx+dirx*0.08, ty+diry*0.08
		p.Line(tx, ty, fx-diry*s*0.05, fy+dirx*s*0.05, 0.02, paint.Mix(paint.Black, paint.Hex(0x42A5F5), fade))
		p.Line(tx+dirx*0.06, ty+diry*0.06, fx+dirx*0.06-diry*s*0.05, fy+diry*0.06+dirx*s*0.05, 0.02, paint.Mix(paint.Black, paint.Hex(0x42A5F5), fade))
	}
	p.Circle(ax, ay, 0.02, paint.Mix(paint.Black, paint.DarkGrey, fade))
	// impact flash and shout
	if k > 0.5 && k < 1.0 {
		e := (k - 0.5) / 0.5
		p.Ring(cx, cy, 0.05+e*0.3, 0.02*(1-e), paint.Mix(paint.Black, paint.Gold, 1-e))
	}
	if k > 0.5 && k < 1.6 {
		p.TextCentred(0.5, 0.05, 0.07*pop(k-0.5), "BULLSEYE", paint.Gold)
	}
}

// ---------------------------------------------------------------------------

type dice struct {
	sys  *paint.System
	face int
	last int
}

func (d *dice) Draw(p *paint.Painter, t, dt float64) {
	const period = 2.4
	n := int(t / period)
	x := t - float64(n)*period
	if n != d.last {
		d.last = n
		d.face = d.sys.Rng.Intn(6) + 1
	}
	// tumble for the first 0.9s
	rot, lift, s := 0.0, 0.0, 1.0
	face := d.face
	if x < 0.9 {
		e := x / 0.9
		rot = smooth(e) * 4 * math.Pi
		lift = 0.25 * math.Sin(e*math.Pi)
		s = 1 - 0.15*math.Sin(e*math.Pi)
		face = int(x*14)%6 + 1
	} else if x < 1.2 {
		s = pop(x - 0.9)
	}
	cx, cy := 0.5, 0.52-lift
	h := 0.30 * s
	q := func(px, py float64) (float64, float64) { return paint.Rot(cx, cy, cx+px*h, cy+py*h, rot) }
	// rounded square: draw the polygon plus corner discs
	x0, y0 := q(-1, -1)
	x1, y1 := q(1, -1)
	x2, y2 := q(1, 1)
	x3, y3 := q(-1, 1)
	white := paint.Hex(0xFAFAFA)
	p.Poly(white, x0, y0, x1, y1, x2, y2, x3, y3)
	for _, c := range [][2]float64{{-1, -1}, {1, -1}, {1, 1}, {-1, 1}} {
		ax, ay := q(c[0]*0.95, c[1]*0.95)
		p.Circle(ax, ay, h*0.12, white)
	}
	pips := map[int][][2]float64{
		1: {{0, 0}},
		2: {{-0.5, -0.5}, {0.5, 0.5}},
		3: {{-0.5, -0.5}, {0, 0}, {0.5, 0.5}},
		4: {{-0.5, -0.5}, {0.5, -0.5}, {-0.5, 0.5}, {0.5, 0.5}},
		5: {{-0.5, -0.5}, {0.5, -0.5}, {0, 0}, {-0.5, 0.5}, {0.5, 0.5}},
		6: {{-0.5, -0.6}, {0.5, -0.6}, {-0.5, 0}, {0.5, 0}, {-0.5, 0.6}, {0.5, 0.6}},
	}
	for _, pp := range pips[face] {
		ax, ay := q(pp[0], pp[1])
		p.Circle(ax, ay, h*0.14, paint.Hex(0x212121))
	}
	p.Ellipse(0.5, 0.90, 0.28*s*(1-lift), 0.03, paint.Hex(0x2A2D33))
	if x >= 0.9 {
		p.TextCentred(0.5, 0.06, 0.08, fmt.Sprint(d.face), paint.Gold)
	}
}

// ---------------------------------------------------------------------------

type gift struct{ sparks *paint.System }

func (g *gift) Draw(p *paint.Painter, t, dt float64) {
	const period = 4.0
	k := math.Mod(t, period)
	box := paint.Hex(0xE53935)
	ribbon := paint.Hex(0xFFD54F)
	// shake, pop, settle
	shake := 0.0
	if k > 1 && k < 1.6 {
		shake = 0.02 * math.Sin(t*50)
	}
	lidY, lidRot := 0.0, 0.0
	if k > 1.6 && k < 3.2 {
		e := (k - 1.6) / 1.6
		lidY = -0.35 * math.Sin(e*math.Pi)
		lidRot = 0.6 * math.Sin(e*math.Pi)
	}
	cx := 0.5 + shake
	p.Rect(cx-0.26, 0.45, 0.52, 0.40, box)
	p.Rect(cx-0.05, 0.45, 0.10, 0.40, ribbon)
	// contents peek out while the lid is up
	if lidY < -0.05 {
		gl := -lidY / 0.35
		p.Glow(cx, 0.45, 0.35, paint.Gold, 0.8*gl)
		p.Star(cx, 0.40-0.1*gl, 0.08*gl, 0.035*gl, 5, t*2, paint.Gold)
		g.sparks.Every(dt, 30, func() {
			g.sparks.Emit(paint.Particle{X: cx + g.sparks.Rand(-0.2, 0.2), Y: 0.45, VY: g.sparks.Rand(-0.6, -0.2), VX: g.sparks.Rand(-0.2, 0.2), Life: 0.9, Size: g.sparks.Rand(0.01, 0.025)})
		})
	}
	g.sparks.Step(dt, 0.5, 0)
	for _, q := range g.sparks.P {
		p.Star(q.X, q.Y, q.Size*(1-q.T())*1.5, q.Size*(1-q.T())*0.5, 4, q.Age*4, paint.Mix(paint.Gold, paint.White, 0.4))
	}
	// lid with bow
	ly := 0.36 + lidY
	q := func(px, py float64) (float64, float64) { return paint.Rot(cx, ly+0.05, px, py, lidRot) }
	x0, y0 := q(cx-0.30, ly)
	x1, y1 := q(cx+0.30, ly)
	x2, y2 := q(cx+0.30, ly+0.10)
	x3, y3 := q(cx-0.30, ly+0.10)
	p.Poly(paint.Mix(box, paint.Black, 0.15), x0, y0, x1, y1, x2, y2, x3, y3)
	r0x, r0y := q(cx-0.05, ly)
	r1x, r1y := q(cx+0.05, ly+0.10)
	p.Line((r0x+r1x)/2, r0y, (r0x+r1x)/2, r1y, 0.10, ribbon)
	bx, by := q(cx, ly-0.02)
	p.Ellipse(bx-0.08, by-0.03, 0.08, 0.05, ribbon)
	p.Ellipse(bx+0.08, by-0.03, 0.08, 0.05, ribbon)
	p.Circle(bx, by-0.02, 0.035, paint.Mix(ribbon, paint.Black, 0.2))
}

func drawBattery(p *paint.Painter, t, dt float64) {
	const period = 6.0
	k := math.Mod(t, period)
	level := math.Min(k/(period-1), 1)
	var c paint.RGB
	switch {
	case level < 0.2:
		c = paint.Hex(0xE53935)
	case level < 0.5:
		c = paint.Hex(0xFFB300)
	default:
		c = paint.Hex(0x43A047)
	}
	shell := paint.Hex(0xB0BEC5)
	p.RoundRect(0.12, 0.30, 0.66, 0.40, 0.06, shell)
	p.RoundRect(0.78, 0.42, 0.08, 0.16, 0.02, shell)
	p.RoundRect(0.16, 0.34, 0.58, 0.32, 0.04, paint.Hex(0x263238))
	// four segments fill in turn
	for i := 0; i < 4; i++ {
		seg := paint.Clamp01(level*4 - float64(i))
		if seg <= 0 {
			continue
		}
		p.RoundRect(0.19+float64(i)*0.14, 0.38, 0.12*seg, 0.24, 0.02, c)
	}
	// charging bolt blinks while charging
	if level < 1 && math.Sin(t*6) > -0.3 {
		p.Poly(paint.White, 0.49, 0.36, 0.42, 0.52, 0.48, 0.52, 0.45, 0.64, 0.54, 0.47, 0.49, 0.47)
	}
	if level >= 1 {
		p.Glow(0.45, 0.5, 0.45, c, 0.4+0.2*math.Sin(t*8))
		p.TextCentred(0.45, 0.80, 0.08*pop(k-(period-1)), "FULL", c)
	} else {
		p.TextCentred(0.45, 0.80, 0.08, fmt.Sprintf("%d%%", int(level*100)), paint.LightGrey)
	}
}

func drawCake(p *paint.Painter, t, dt float64) {
	plate := paint.Hex(0xCFD8DC)
	p.Ellipse(0.5, 0.90, 0.40, 0.05, plate)
	sponge := paint.Hex(0xD7A86E)
	icing := paint.Hex(0xF48FB1)
	p.Rect(0.18, 0.58, 0.64, 0.30, sponge)
	p.Rect(0.18, 0.66, 0.64, 0.05, paint.Hex(0x8D6E63))
	p.Rect(0.18, 0.55, 0.64, 0.06, icing)
	for i := 0; i < 7; i++ {
		x := 0.22 + float64(i)*0.095
		p.Circle(x, 0.61+0.01*float64(i%2), 0.035, icing)
	}
	// candles with flickering flames
	for i := 0; i < 3; i++ {
		x := 0.34 + float64(i)*0.16
		p.Rect(x-0.02, 0.40, 0.04, 0.16, paint.Mix(paint.SkyBlue, paint.White, float64(i)*0.3))
		p.Line(x, 0.40, x, 0.37, 0.01, paint.DarkGrey)
		fl := 0.03 + 0.012*math.Sin(t*11+float64(i)*2) + 0.008*math.Sin(t*23+float64(i))
		fx := x + 0.008*math.Sin(t*7+float64(i)*3)
		p.Glow(fx, 0.35, 0.1, paint.Orange, 0.5)
		p.Ellipse(fx, 0.35-fl*0.5, 0.02, fl, paint.Hex(0xFF9800))
		p.Ellipse(fx, 0.35-fl*0.4, 0.01, fl*0.6, paint.Hex(0xFFEB3B))
		// wisps of smoke
		k := math.Mod(t*0.5+float64(i)*0.33, 1)
		p.CircleA(fx+0.03*math.Sin(k*8), 0.30-k*0.2, 0.01+k*0.02, paint.Grey, (1-k)*0.4)
	}
	p.TextCentred(0.5, 0.78, 0.06, "HBD", paint.Hex(0x8D6E63))
}

func drawRefresh(p *paint.Painter, t, dt float64) {
	a := t * 2.2
	blue := paint.Hex(0x42A5F5)
	for _, s := range []float64{0, math.Pi} {
		p.Arc(0.5, 0.5, 0.30, a+s+0.35, a+s+math.Pi-0.35, 0.07, blue)
		hx, hy := 0.5+0.30*math.Cos(a+s+0.35), 0.5+0.30*math.Sin(a+s+0.35)
		d := a + s + 0.35 - math.Pi/2
		p.Triangle(hx+0.09*math.Cos(d+math.Pi/2), hy+0.09*math.Sin(d+math.Pi/2), hx-0.09*math.Cos(d+math.Pi/2), hy-0.09*math.Sin(d+math.Pi/2), hx+0.11*math.Cos(d), hy+0.11*math.Sin(d), blue)
	}
	p.Glow(0.5, 0.5, 0.16, blue, 0.3+0.2*math.Sin(t*4))
}

func drawSignal(p *paint.Painter, t, dt float64) {
	// dish
	p.Line(0.5, 0.92, 0.5, 0.65, 0.04, paint.Steel)
	p.Line(0.36, 0.92, 0.64, 0.92, 0.04, paint.Steel)
	tilt := -math.Pi/4 + 0.1*math.Sin(t*0.7)
	cx, cy := 0.5, 0.62
	p.Arc(cx, cy, 0.18, tilt+math.Pi*0.55, tilt+math.Pi*1.45, 0.07, paint.Hex(0xB0BEC5))
	fx, fy := cx+0.15*math.Cos(tilt+math.Pi), cy+0.15*math.Sin(tilt+math.Pi)
	p.Line(cx, cy, fx, fy, 0.015, paint.Steel)
	p.Circle(fx, fy, 0.025, paint.Red)
	// rings broadcast outward
	dir := tilt + math.Pi
	for i := 0; i < 4; i++ {
		k := math.Mod(t*0.5+float64(i)/4, 1)
		r := 0.1 + k*0.55
		c := paint.Mix(paint.Black, paint.Hex(0x4FC3F7), 1-k)
		p.Arc(fx, fy, r, dir-0.5, dir+0.5, 0.02, c)
	}
	stars(p, t, 12, 77)
}

func drawCandle(p *paint.Painter, t, dt float64) {
	wax := paint.Hex(0xFFF8E1)
	p.Ellipse(0.5, 0.90, 0.20, 0.04, paint.Hex(0xB0BEC5))
	p.Rect(0.42, 0.42, 0.16, 0.48, wax)
	p.Ellipse(0.5, 0.42, 0.08, 0.025, paint.Mix(wax, paint.Grey, 0.2))
	// drips
	for i := 0; i < 3; i++ {
		x := 0.44 + float64(i)*0.06
		l := 0.05 + 0.04*noise(i+3) + 0.01*math.Sin(t+float64(i))
		p.Line(x, 0.43, x, 0.43+l, 0.02, wax)
	}
	p.Line(0.5, 0.42, 0.5, 0.36, 0.01, paint.DarkGrey)
	draught := 0.03*math.Sin(t*2.3) + 0.01*math.Sin(t*9.1)
	fl := 0.09 + 0.02*math.Sin(t*13) + 0.01*math.Sin(t*29)
	fx := 0.5 + draught
	p.Glow(fx, 0.30, 0.35, paint.Hex(0xFFB74D), 0.5+0.1*math.Sin(t*17))
	p.Ellipse(fx, 0.34-fl*0.5, 0.04, fl, paint.Hex(0xFF9800))
	p.Ellipse(fx+draught*0.3, 0.35-fl*0.4, 0.02, fl*0.6, paint.Hex(0xFFEB3B))
	p.Ellipse(fx+draught*0.5, 0.36-fl*0.2, 0.01, fl*0.25, paint.Hex(0x4FC3F7))
}

func drawGears(p *paint.Painter, t, dt float64) {
	gear := func(cx, cy, r float64, teeth int, a float64, c paint.RGB) {
		for i := 0; i < teeth; i++ {
			an := a + float64(i)*2*math.Pi/float64(teeth)
			p.Line(cx, cy, cx+(r+0.05)*math.Cos(an), cy+(r+0.05)*math.Sin(an), r*0.45, c)
		}
		p.Circle(cx, cy, r, c)
		p.Circle(cx, cy, r*0.35, paint.Hex(0x263238))
	}
	gear(0.36, 0.52, 0.18, 8, t*1.0, paint.Hex(0x90A4AE))
	gear(0.66, 0.34, 0.11, 6, -t*1.0*8/6+0.25, paint.Hex(0xFFB300))
	gear(0.68, 0.72, 0.12, 6, -t*1.0*8/6+0.1, paint.Hex(0x78909C))
}
