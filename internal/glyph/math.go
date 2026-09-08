package glyph

import (
	"fmt"
	"math"

	"github.com/Cid-Emmerich/SopeBox/internal/paint"
)

func initmath() {
	add("math",
		Def{Emoji: "➕", Name: "plus", Blurb: "plus morphs into times, minus, divide, equals and back", Aliases: []string{"add", "operators", "arithmetic", "cross", "times", "minus", "divide", "equals", "percent", "✖", "➖", "➗", "🟰"}, New: stateless(drawOperators)},
		Def{Emoji: "♾️", Name: "infinity", Blurb: "a comet loops the lemniscate forever", Aliases: []string{"infinite", "forever", "loop", "lemniscate"}, New: stateless(drawInfinity)},
		Def{Emoji: "π", Name: "pi", Blurb: "pi on a pie with its digits streaming past", Aliases: []string{"pie", "3.14", "circle constant"}, New: stateless(drawPi)},
		Def{Emoji: "∑", Name: "sigma", Blurb: "numbers drop into a sum", Aliases: []string{"sum", "summation", "total"}, New: stateless(drawSigma)},
		Def{Emoji: "🔢", Name: "numbers", Blurb: "an odometer counting up", Aliases: []string{"counter", "odometer", "digits", "count", "1234"}, New: stateless(drawOdometer)},
		Def{Emoji: "💯", Name: "hundred", Blurb: "one hundred percent, with feeling", Aliases: []string{"100", "perfect", "score", "hundred points"}, New: func() Anim { return &hundred{sparks: paint.NewSystem(31)} }},
		Def{Emoji: "√", Name: "root", Blurb: "the square root pulls numbers out of the radical", Aliases: []string{"sqrt", "square root", "radical"}, New: stateless(drawRoot)},
		Def{Emoji: "🥧", Name: "pie chart", Blurb: "slices grow and shrink", Aliases: []string{"chart", "📊", "📈", "graph", "stats"}, New: stateless(drawPieChart)},
	)
}

// stroke is a rounded bar that can also be a dot (length 0).
type stroke struct{ x, y, a, l, th float64 }

type symbol [3]stroke

var operatorSymbols = []symbol{
	{{0.5, 0.5, 0, 0.58, 0.12}, {0.5, 0.5, math.Pi / 2, 0.58, 0.12}, {0.5, 0.5, 0, 0, 0}},               // +
	{{0.5, 0.5, math.Pi / 4, 0.58, 0.12}, {0.5, 0.5, 3 * math.Pi / 4, 0.58, 0.12}, {0.5, 0.5, 0, 0, 0}}, // ×
	{{0.5, 0.5, 0, 0.58, 0.12}, {0.5, 0.5, math.Pi / 2, 0, 0}, {0.5, 0.5, 0, 0, 0}},                     // −
	{{0.5, 0.5, 0, 0.58, 0.12}, {0.5, 0.30, math.Pi / 2, 0, 0.13}, {0.5, 0.70, 0, 0, 0.13}},             // ÷
	{{0.5, 0.40, 0, 0.58, 0.12}, {0.5, 0.60, 0, 0.58, 0.12}, {0.5, 0.70, 0, 0, 0}},                      // =
	{{0.5, 0.5, -math.Pi / 4, 0.62, 0.11}, {0.33, 0.33, 0, 0, 0.16}, {0.67, 0.67, 0, 0, 0.16}},          // %
	{{0.5, 0.5, 0, 0.58, 0.12}, {0.5, 0.5, math.Pi / 2, 0.58, 0.12}, {0.5, 0.5, 0, 0, 0}},               // + again
}

var operatorColours = []paint.RGB{paint.Hex(0xFF7043), paint.Hex(0xEF5350), paint.Hex(0x66BB6A), paint.Hex(0x42A5F5), paint.Hex(0xAB47BC), paint.Hex(0x26C6DA), paint.Hex(0xFF7043)}

func lerpAngle(a, b, t float64) float64 {
	d := math.Mod(b-a+3*math.Pi, 2*math.Pi) - math.Pi
	return a + d*t
}

func drawOperators(p *paint.Painter, t, dt float64) {
	const hold, move = 1.1, 0.7
	n := len(operatorSymbols) - 1
	k := math.Mod(t, float64(n)*(hold+move))
	i := int(k / (hold + move))
	f := k - float64(i)*(hold+move)
	m := 0.0
	if f > hold {
		m = smooth((f - hold) / move)
	}
	a, b := operatorSymbols[i], operatorSymbols[i+1]
	col := paint.Mix(operatorColours[i], operatorColours[i+1], m)
	// squircle background, breathing while morphing
	s := 0.86 + 0.06*math.Sin(m*math.Pi)
	p.RoundRect(0.5-s/2, 0.5-s/2, s, s, 0.14, col)
	for j := 0; j < 3; j++ {
		x := lerp(a[j].x, b[j].x, m)
		y := lerp(a[j].y, b[j].y, m)
		ang := lerpAngle(a[j].a, b[j].a, m)
		l := lerp(a[j].l, b[j].l, m)
		th := lerp(a[j].th, b[j].th, m)
		if th < 0.01 {
			continue
		}
		dx, dy := math.Cos(ang)*l/2, math.Sin(ang)*l/2
		p.Line(x-dx, y-dy, x+dx, y+dy, th, paint.White)
	}
}

func drawInfinity(p *paint.Painter, t, dt float64) {
	cx, cy, a := 0.5, 0.5, 0.42
	pt := func(th float64) (float64, float64) {
		s, c := math.Sincos(th)
		d := 1 + s*s
		return cx + a*c/d, cy + a*s*c/d*1.3
	}
	col := paint.Hex(0x7E57C2)
	th := 0.075 + 0.015*math.Sin(t*2)
	const n = 90
	for i := 0; i < n; i++ {
		x0, y0 := pt(float64(i) / n * 2 * math.Pi)
		x1, y1 := pt(float64(i+1) / n * 2 * math.Pi)
		u := float64(i) / n
		p.Line(x0, y0, x1, y1, th, paint.Mix(col, paint.Hex(0x26C6DA), 0.5+0.5*math.Sin(u*2*math.Pi+t)))
	}
	// comet with a trail
	head := t * 1.6
	for i := 14; i >= 0; i-- {
		x, y := pt(head - float64(i)*0.07)
		k := 1 - float64(i)/15
		p.CircleA(x, y, 0.02+0.05*k, paint.Mix(paint.Gold, paint.White, k), 0.25+0.75*k)
	}
	hx, hy := pt(head)
	p.Glow(hx, hy, 0.16, paint.Gold, 0.8)
}

func drawPi(p *paint.Painter, t, dt float64) {
	// the pie: crust ring and filling with a wandering missing slice
	slice := math.Mod(t*0.8, 2*math.Pi)
	p.Arc(0.5, 0.52, 0.30, slice+0.9, slice+2*math.Pi, 0.14, paint.Hex(0xD7A86E))
	p.Arc(0.5, 0.52, 0.30, slice+0.9, slice+2*math.Pi, 0.07, paint.Hex(0xB71C1C))
	p.Arc(0.5, 0.52, 0.36, slice+0.9, slice+2*math.Pi, 0.03, paint.Hex(0xA1733F))
	// π: a wavy top bar and two legs, bobbing
	bob := 0.015 * math.Sin(t*2.5)
	ink := paint.Hex(0x263238)
	p.Line(0.30, 0.35+bob, 0.72, 0.33+bob, 0.075, ink)
	p.Line(0.37, 0.36+bob, 0.35, 0.70+bob, 0.075, ink)
	p.Line(0.35, 0.70+bob, 0.30, 0.74+bob, 0.06, ink)
	p.Line(0.62, 0.36+bob, 0.64, 0.70+bob, 0.075, ink)
	p.Arc(0.685, 0.68+bob, 0.05, 0.3, math.Pi+0.3, 0.05, ink)
	// digits stream along the bottom
	const digits = "3.1415926535897932384626433832795028841971693993751"
	size := 0.09
	w := paint.TextWidth(digits, size) + 0.4
	off := math.Mod(t*0.25, w)
	x := p.R + 0.1 - off
	for i, r := range digits {
		gx := x + float64(i)*size*0.8
		if gx > p.R || gx < p.L-0.1 {
			continue
		}
		c := paint.Mix(paint.Hex(0x26A69A), paint.White, 0.5+0.5*math.Sin(t*3+float64(i)*0.4))
		p.Text(gx, 0.88, size, string(r), c)
	}
}

func drawSigma(p *paint.Painter, t, dt float64) {
	ink := paint.Hex(0x1E88E5)
	// Σ
	p.Line(0.68, 0.22, 0.30, 0.22, 0.06, ink)
	p.Line(0.30, 0.22, 0.52, 0.50, 0.06, ink)
	p.Line(0.52, 0.50, 0.30, 0.78, 0.06, ink)
	p.Line(0.30, 0.78, 0.68, 0.78, 0.06, ink)
	// numbers 1..4 fall in one at a time; the total grows on the right
	const per = 1.1
	k := math.Mod(t, per*5)
	step := int(k / per)
	f := (k - float64(step)*per) / per
	total := 0
	for i := 1; i <= step && i <= 4; i++ {
		total += i
	}
	if step < 4 {
		y := lerp(-0.1, 0.42, smooth(f*1.3))
		if f > 0.77 {
			y = 0.42 + 0.02*math.Sin((f-0.77)*30)
		}
		p.TextCentred(0.14, y, 0.16, fmt.Sprint(step+1), paint.Gold)
	}
	pk := 1.0
	if step > 0 && f < 0.3 {
		pk = pop(f)
	}
	size := 0.22 * pk
	p.Text(0.76, 0.5-size/2, size, fmt.Sprintf("=%d", total), paint.Mix(paint.Gold, paint.White, 0.3))
	if step == 4 {
		p.TextCentred(0.5, 0.88, 0.07, "1+2+3+4", paint.Grey)
	}
}

func drawOdometer(p *paint.Painter, t, dt float64) {
	value := t * 7.3
	const n = 4
	w, h := 0.16, 0.30
	x0 := 0.5 - (w*float64(n)+0.02*float64(n-1))/2
	p.RoundRect(x0-0.05, 0.5-h/2-0.06, w*float64(n)+0.02*float64(n-1)+0.10, h+0.12, 0.04, paint.Hex(0x37474F))
	for i := 0; i < n; i++ {
		pw := math.Pow(10, float64(n-1-i))
		v := value / pw
		digit := int(v) % 10
		frac := v - math.Floor(v)
		// a wheel only turns while the wheel to its right wraps
		roll := 0.0
		if i == n-1 {
			roll = frac
		} else {
			lower := math.Mod(value, pw)
			if lower > pw-1 {
				roll = lower - (pw - 1)
			}
		}
		roll = smooth(roll)
		x := x0 + float64(i)*(w+0.02)
		p.RoundRect(x, 0.5-h/2, w, h, 0.02, paint.Hex(0x111318))
		cl := p.Clip(x, 0.5-h/2, w, h)
		size := h * 0.7
		cl.TextCentred(x+w/2, 0.5-size/2+roll*h, size, fmt.Sprint(digit), paint.Hex(0xF5F5F5))
		cl.TextCentred(x+w/2, 0.5-size/2+roll*h-h, size, fmt.Sprint((digit+1)%10), paint.Hex(0xF5F5F5))
		// a shadow line across the middle
		p.RectA(x, 0.5-0.006, w, 0.012, paint.Black, 0.5)
	}
}

type hundred struct{ sparks *paint.System }

func (h *hundred) Draw(p *paint.Painter, t, dt float64) {
	k := cycle(t, 3)
	s := pop(k * 3)
	red := paint.Hex(0xE53935)
	size := 0.30 * s
	p.TextCentred(0.5, 0.42-size/2, size, "100", red)
	// double underline sweeping in
	l := smooth(k * 3)
	p.Line(0.5-0.34*l, 0.63, 0.5+0.34*l, 0.63, 0.035*s, red)
	p.Line(0.5-0.30*l, 0.71, 0.5+0.30*l, 0.71, 0.035*s, red)
	if k < 0.05 {
		h.sparks.Every(dt, 80, func() {
			a := h.sparks.Rand(0, 2*math.Pi)
			h.sparks.Emit(paint.Particle{X: 0.5, Y: 0.5, VX: 0.9 * math.Cos(a), VY: 0.9 * math.Sin(a), Life: h.sparks.Rand(0.5, 1), Size: h.sparks.Rand(0.01, 0.03)})
		})
	}
	h.sparks.Step(dt, 0.4, 1.5)
	for _, q := range h.sparks.P {
		p.Star(q.X, q.Y, q.Size*(1-q.T())*1.5, q.Size*(1-q.T())*0.6, 4, q.Age*5, paint.Gold)
	}
	p.TextCentred(0.5, 0.82, 0.07, "PERCENT", paint.Grey)
}

func drawRoot(p *paint.Painter, t, dt float64) {
	ink := paint.Hex(0x00897B)
	// radical sign
	p.Line(0.12, 0.55, 0.20, 0.50, 0.04, ink)
	p.Line(0.20, 0.50, 0.30, 0.80, 0.05, ink)
	p.Line(0.30, 0.80, 0.42, 0.22, 0.05, ink)
	p.Line(0.42, 0.22, 0.88, 0.22, 0.045, ink)
	// perfect squares walk in, their roots pop out on top
	squares := []int{4, 9, 16, 25, 36, 49, 64, 81}
	const per = 1.6
	k := math.Mod(t, per*float64(len(squares)))
	i := int(k / per)
	f := (k - float64(i)*per) / per
	sq := squares[i]
	in := smooth(f * 2)
	alpha := 1.0
	if f > 0.85 {
		alpha = 1 - (f-0.85)/0.15
	}
	x := lerp(1.1, 0.64, in)
	p.TextCentred(x, 0.40, 0.24, fmt.Sprint(sq), paint.Mix(paint.Black, paint.Hex(0xFFA726), alpha))
	if f > 0.5 {
		out := smooth((f - 0.5) * 2.5)
		y := lerp(0.40, 0.06, out)
		s := 0.10 + 0.14*out
		p.TextCentred(0.64, y-s/2+0.07, s, fmt.Sprintf("=%d", int(math.Sqrt(float64(sq)))), paint.Mix(paint.Black, paint.Gold, alpha))
	}
}

func drawPieChart(p *paint.Painter, t, dt float64) {
	cols := []paint.RGB{paint.Hex(0x42A5F5), paint.Hex(0xFFCA28), paint.Hex(0xEF5350), paint.Hex(0x66BB6A), paint.Hex(0xAB47BC)}
	vals := make([]float64, len(cols))
	sum := 0.0
	for i := range vals {
		vals[i] = 1 + 0.9*math.Sin(t*0.9+float64(i)*1.7)
		sum += vals[i]
	}
	a := -math.Pi / 2
	for i, v := range vals {
		span := v / sum * 2 * math.Pi
		// the biggest slice pulls out of the pie
		mid := a + span/2
		off := 0.0
		if v == maxf(vals) {
			off = 0.03
		}
		cx, cy := 0.42+off*math.Cos(mid), 0.5+off*math.Sin(mid)
		p.Arc(cx, cy, 0.17, a, a+span-0.02, 0.32, cols[i])
		// legend bar
		w := v / sum
		p.Rect(0.72, 0.28+float64(i)*0.10, 0.22*w+0.02, 0.06, cols[i])
		a += span
	}
	p.Circle(0.42, 0.5, 0.0, paint.Black)
}

func maxf(v []float64) float64 {
	m := v[0]
	for _, x := range v {
		if x > m {
			m = x
		}
	}
	return m
}
