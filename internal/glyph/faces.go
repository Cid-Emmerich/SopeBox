package glyph

import (
	"math"

	"github.com/Cid-Emmerich/SopeBox/internal/paint"
)

func initfaces() {
	add("faces",
		Def{Emoji: "😀", Name: "grin", Blurb: "a big happy grin", Aliases: []string{"smile", "happy", "smiley", "grinning", "😃", "😄", "🙂"}, New: stateless(drawGrin)},
		Def{Emoji: "😉", Name: "wink", Blurb: "a cheeky wink", Aliases: []string{"winking", "flirt", "cheeky"}, New: stateless(drawWink)},
		Def{Emoji: "🙃", Name: "upside down", Blurb: "a face that keeps flipping over", Aliases: []string{"upsidedown", "flip", "silly", "ironic"}, New: stateless(drawUpsideDown)},
		Def{Emoji: "😎", Name: "cool", Blurb: "too cool, sunglasses included", Aliases: []string{"sunglasses", "shades", "deal with it"}, New: stateless(drawCool)},
		Def{Emoji: "😴", Name: "sleeping", Blurb: "fast asleep and snoring", Aliases: []string{"sleep", "zzz", "tired", "snore", "nap"}, New: stateless(drawSleeping)},
		Def{Emoji: "🤯", Name: "mind blown", Blurb: "the top of the head goes off like a volcano", Aliases: []string{"mindblown", "exploding head", "wow", "whoa"}, New: func() Anim { return &mindBlown{bits: paint.NewSystem(81)} }},
		Def{Emoji: "😍", Name: "heart eyes", Blurb: "in love, with beating heart eyes", Aliases: []string{"hearteyes", "love", "adore", "crush"}, New: stateless(drawHeartEyes)},
		Def{Emoji: "🤔", Name: "thinking", Blurb: "hmm...", Aliases: []string{"think", "hmm", "ponder", "wonder"}, New: stateless(drawThinking)},
		Def{Emoji: "😂", Name: "joy", Blurb: "laughing so hard it hurts", Aliases: []string{"laugh", "laughing", "lol", "tears", "haha", "🤣"}, New: stateless(drawJoy)},
		Def{Emoji: "🥳", Name: "party face", Blurb: "hat, horn and confetti", Aliases: []string{"partying", "celebrate", "birthday face", "woo"}, New: func() Anim { return &partyFace{confetti: paint.NewSystem(82)} }},
		Def{Emoji: "😱", Name: "scream", Blurb: "screaming in fear", Aliases: []string{"scared", "fear", "shock", "aah", "horror"}, New: stateless(drawScream)},
		Def{Emoji: "🤖", Name: "robot", Blurb: "a robot with blinking lights", Aliases: []string{"bot", "android", "machine", "beep"}, New: stateless(drawRobot)},
		Def{Emoji: "😡", Name: "angry", Blurb: "red hot fury", Aliases: []string{"mad", "rage", "furious", "😠", "grr"}, New: stateless(drawAngry)},
	)
}

// faceBase draws the yellow disc with shading.
func faceBase(p *paint.Painter, cx, cy, r float64) {
	p.Circle(cx, cy, r, paint.Hex(0xF9A825))
	p.Circle(cx-r*0.05, cy-r*0.05, r*0.94, paint.Hex(0xFFD54F))
}

// eyes draws two oval eyes with blink and gaze.
func eyes(p *paint.Painter, cx, cy, r, open, lookX, lookY float64) {
	for _, s := range []float64{-1, 1} {
		ex := cx + s*r*0.36 + lookX*r
		ey := cy - r*0.15 + lookY*r
		p.Ellipse(ex, ey, r*0.09, r*0.14*open+0.004, paint.Hex(0x4E342E))
		p.Circle(ex-r*0.03, ey-r*0.05, r*0.03*open, paint.White)
	}
}

func drawGrin(p *paint.Painter, t, dt float64) {
	cx, cy, r := 0.5, 0.5, 0.40
	bounce := 0.01 * math.Sin(t*4)
	cy += bounce
	faceBase(p, cx, cy, r)
	b := blink(t, 4, 0)
	eyes(p, cx, cy, r, b, 0.03*wobble(t*0.5, 0), 0)
	// the smile widens with a heartbeat
	wide := 0.55 + 0.08*math.Max(0, math.Sin(t*2))
	p.Arc(cx, cy+r*0.10, r*wide, 0.25, math.Pi-0.25, r*0.06, paint.Hex(0x4E342E))
	p.Shade(cx-r*0.55, cy+r*0.22, r*1.1, r*0.35, func(u, v float64) (paint.RGB, float64) {
		x := (u - 0.5) * 2
		if v < 0.35 && x*x < 1 {
			return paint.White, 1
		}
		return paint.White, 0
	})
	// eyebrows raise
	for _, s := range []float64{-1, 1} {
		y := cy - r*0.40 - 0.01*math.Max(0, math.Sin(t*2))
		p.Arc(cx+s*r*0.36, y+r*0.05, r*0.14, math.Pi+0.4, 2*math.Pi-0.4, r*0.04, paint.Hex(0x4E342E))
	}
	p.Circle(cx-r*0.62, cy+r*0.12, r*0.10, paint.Mix(paint.Hex(0xFFD54F), paint.Pink, 0.5))
	p.Circle(cx+r*0.62, cy+r*0.12, r*0.10, paint.Mix(paint.Hex(0xFFD54F), paint.Pink, 0.5))
}

func drawWink(p *paint.Painter, t, dt float64) {
	cx, cy, r := 0.5, 0.5, 0.40
	tilt := 0.06 * math.Sin(t*1.5)
	faceBase(p, cx, cy, r)
	// left eye open (blinks), right eye winks every 2.5s
	k := math.Mod(t, 2.5)
	wink := 0.0
	if k < 0.7 {
		wink = math.Sin(k / 0.7 * math.Pi)
	}
	openL := blink(t, 5, 0.3)
	ex, ey := cx-r*0.36, cy-r*0.15+tilt
	p.Ellipse(ex, ey, r*0.09, r*0.14*openL+0.004, paint.Hex(0x4E342E))
	p.Circle(ex-r*0.03, ey-r*0.05, r*0.03*openL, paint.White)
	ex2 := cx + r*0.36
	if wink > 0.5 {
		p.Arc(ex2, ey+r*0.02, r*0.10, math.Pi+0.3, 2*math.Pi-0.3, r*0.04, paint.Hex(0x4E342E))
	} else {
		p.Ellipse(ex2, ey, r*0.09, r*0.14*(1-wink*2)+0.004, paint.Hex(0x4E342E))
	}
	// lopsided smile
	p.Arc(cx+r*0.05, cy+r*0.12, r*0.42, 0.35, math.Pi-0.5, r*0.055, paint.Hex(0x4E342E))
	if wink > 0.3 {
		p.Star(cx+r*0.75, cy-r*0.5, 0.04*wink, 0.015*wink, 4, t*3, paint.White)
	}
}

func drawUpsideDown(p *paint.Painter, t, dt float64) {
	cx, cy, r := 0.5, 0.5, 0.40
	k := math.Mod(t, 4)
	rot := math.Pi
	if k < 0.8 {
		rot = math.Pi * (1 - smooth(k/0.8))
	} else if k > 2.5 && k < 3.3 {
		rot = math.Pi * smooth((k-2.5)/0.8)
	} else if k <= 2.5 {
		rot = 0
	}
	// rotate by drawing with rotated coordinates
	q := func(x, y float64) (float64, float64) { return paint.Rot(cx, cy, x, y, rot) }
	faceBase(p, cx, cy, r)
	b := blink(t, 4, 0.5)
	for _, s := range []float64{-1, 1} {
		ex, ey := q(cx+s*r*0.36, cy-r*0.15)
		p.Ellipse(ex, ey, r*0.09, r*0.14*b+0.004, paint.Hex(0x4E342E))
	}
	p.Arc(cx, cy+r*0.10*math.Cos(rot), r*0.45, 0.35+rot, math.Pi-0.35+rot, r*0.05, paint.Hex(0x4E342E))
	p.TextCentred(0.5, 0.94, 0.05, "WHY NOT", paint.Grey)
}

func drawCool(p *paint.Painter, t, dt float64) {
	cx, cy, r := 0.5, 0.5, 0.40
	faceBase(p, cx, cy, r)
	// shades slide down to reveal a wink, then back up
	k := math.Mod(t, 5)
	slide := 0.0
	if k > 3 && k < 4.2 {
		slide = math.Sin((k - 3) / 1.2 * math.Pi)
	}
	eyes(p, cx, cy, r, 1, 0, 0)
	if slide > 0.5 {
		p.Arc(cx+r*0.36, cy-r*0.13, r*0.10, math.Pi+0.3, 2*math.Pi-0.3, r*0.04, paint.Hex(0x4E342E))
		p.Circle(cx+r*0.36, cy-r*0.13, r*0.12, paint.Hex(0xFFD54F))
		p.Arc(cx+r*0.36, cy-r*0.13, r*0.10, math.Pi+0.3, 2*math.Pi-0.3, r*0.04, paint.Hex(0x4E342E))
	}
	gy := cy - r*0.15 + slide*r*0.28
	dark := paint.Hex(0x212121)
	for _, s := range []float64{-1, 1} {
		p.RoundRect(cx+s*r*0.36-r*0.20, gy-r*0.13, r*0.40, r*0.26, r*0.08, dark)
		// glint sweeping across
		g := math.Mod(t*0.6+float64(s)*0.1, 1)
		p.Line(cx+s*r*0.36-r*0.15+g*r*0.3, gy-r*0.10, cx+s*r*0.36-r*0.20+g*r*0.3, gy+r*0.10, 0.012, paint.Mix(dark, paint.White, 0.5))
	}
	p.Line(cx-r*0.16, gy, cx+r*0.16, gy, r*0.04, dark)
	p.Line(cx-r*0.56, gy-r*0.04, cx-r*0.7, gy-r*0.12, r*0.04, dark)
	p.Line(cx+r*0.56, gy-r*0.04, cx+r*0.7, gy-r*0.12, r*0.04, dark)
	p.Arc(cx, cy+r*0.15, r*0.40, 0.4, math.Pi-0.4, r*0.05, paint.Hex(0x4E342E))
}

func drawSleeping(p *paint.Painter, t, dt float64) {
	cx, cy, r := 0.5, 0.52, 0.40
	breath := 1 + 0.015*math.Sin(t*1.5)
	faceBase(p, cx, cy, r*breath)
	for _, s := range []float64{-1, 1} {
		p.Arc(cx+s*r*0.36, cy-r*0.18, r*0.12, 0.3, math.Pi-0.3, r*0.04, paint.Hex(0x4E342E))
	}
	// mouth opens with a snore
	snore := math.Max(0, math.Sin(t*1.5))
	p.Ellipse(cx, cy+r*0.30, r*0.10, r*(0.04+0.08*snore), paint.Hex(0x4E342E))
	// a snot bubble grows and pops
	bk := math.Mod(t, 3)
	if bk < 2.6 {
		br := r * 0.05 * (1 + bk)
		p.CircleA(cx+r*0.25, cy+r*0.30, br, paint.Mix(paint.SkyBlue, paint.White, 0.5), 0.8)
	}
	for i := 0; i < 3; i++ {
		k := math.Mod(t*0.4+float64(i)/3, 1)
		x := cx + r*0.6 + k*0.2
		y := cy - r*0.3 - k*0.35
		p.Text(x, y, 0.05+k*0.06, "Z", paint.Mix(paint.Black, paint.LightGrey, math.Sin(k*math.Pi)))
	}
}

type mindBlown struct{ bits *paint.System }

func (m *mindBlown) Draw(p *paint.Painter, t, dt float64) {
	cx, cy, r := 0.5, 0.55, 0.36
	const period = 4.0
	k := math.Mod(t, period)
	blow := 0.0
	if k > 1 && k < 2.5 {
		blow = smooth((k - 1) / 0.4)
	} else if k >= 2.5 {
		blow = 1 - smooth((k-2.5)/1.0)
	}
	shake := 0.0
	if k > 0.5 && k < 1.2 {
		shake = 0.01 * math.Sin(t*60)
	}
	faceBase(p, cx+shake, cy, r)
	// wide eyes, small o mouth
	for _, s := range []float64{-1, 1} {
		ex := cx + shake + s*r*0.36
		p.Circle(ex, cy-r*0.12, r*0.13, paint.White)
		p.Circle(ex, cy-r*0.12, r*0.06+r*0.03*blow, paint.Hex(0x4E342E))
	}
	p.Ellipse(cx+shake, cy+r*0.32, r*0.10, r*0.12+r*0.06*blow, paint.Hex(0x4E342E))
	// the top of the head lifts off
	if blow > 0 {
		lift := 0.25 * blow
		p.Circle(cx+shake, cy-lift-r*0.05, r*0.9, paint.Hex(0xFFD54F))
		p.Rect(cx-r, cy-r*0.4-lift*0.5, 2*r, r*0.5+lift, paint.Black)
		// cover the seam: redraw a bit of face below the blast
		p.Ellipse(cx+shake, cy-r*0.35, r*0.95, r*0.15, paint.Hex(0xFFD54F))
		p.Circle(cx+shake, cy-r*0.05-lift, r*0.9, paint.Hex(0xF9A825))
		// mushroom cloud
		p.Circle(cx, cy-r*0.9-lift, r*0.35*blow, paint.Hex(0xFF7043))
		p.Circle(cx-r*0.25, cy-r*0.8-lift, r*0.28*blow, paint.Hex(0xFF8A65))
		p.Circle(cx+r*0.25, cy-r*0.8-lift, r*0.28*blow, paint.Hex(0xFF8A65))
		p.Rect(cx-r*0.12, cy-r*0.9-lift, r*0.24, lift+r*0.4, paint.Hex(0xFF7043))
		m.bits.Every(dt, 25, func() {
			m.bits.Emit(paint.Particle{X: cx + m.bits.Rand(-0.1, 0.1), Y: cy - r*0.5, VX: m.bits.Rand(-0.6, 0.6), VY: m.bits.Rand(-0.9, -0.3), Life: 1, Size: m.bits.Rand(0.01, 0.025)})
		})
	}
	m.bits.Step(dt, 1.2, 0)
	for _, q := range m.bits.P {
		p.Dot(q.X, q.Y, q.Size, paint.Mix(paint.Hex(0xFFD54F), paint.Red, q.T()))
	}
}

func drawHeartEyes(p *paint.Painter, t, dt float64) {
	cx, cy, r := 0.5, 0.5, 0.40
	faceBase(p, cx, cy, r)
	beat := 1 + 0.12*math.Exp(-8*math.Mod(t, 0.9))
	red := paint.Hex(0xE53935)
	for _, s := range []float64{-1, 1} {
		p.Heart(cx+s*r*0.36, cy-r*0.15, r*0.42*beat, red)
	}
	p.Arc(cx, cy+r*0.10, r*0.48, 0.3, math.Pi-0.3, r*0.055, paint.Hex(0x4E342E))
	p.Shade(cx-r*0.48, cy+r*0.22, r*0.96, r*0.35, func(u, v float64) (paint.RGB, float64) {
		x := (u - 0.5) * 2
		if v < 0.3 && x*x < 1 {
			return paint.White, 1
		}
		return paint.White, 0
	})
	for i := 0; i < 5; i++ {
		k := math.Mod(t*0.3+float64(i)/5, 1)
		x := cx + r*1.1*math.Cos(float64(i)*1.26) + 0.03*math.Sin(k*8)
		y := 0.95 - k*0.9
		p.Heart(x, y, 0.05+0.03*math.Sin(k*math.Pi), paint.Mix(paint.Black, paint.Pink, math.Sin(k*math.Pi)))
	}
}

func drawThinking(p *paint.Painter, t, dt float64) {
	cx, cy, r := 0.5, 0.55, 0.36
	faceBase(p, cx, cy, r)
	look := 0.06 * wobble(t*0.5, 2)
	eyes(p, cx, cy, r, blink(t, 4, 0.8), look, -0.05)
	// one raised eyebrow, one flat
	p.Arc(cx-r*0.36, cy-r*0.30-0.01*math.Sin(t), r*0.14, math.Pi+0.4, 2*math.Pi-0.4, r*0.04, paint.Hex(0x4E342E))
	p.Line(cx+r*0.20, cy-r*0.38, cx+r*0.52, cy-r*0.34, r*0.04, paint.Hex(0x4E342E))
	// wry mouth
	p.Line(cx-r*0.25, cy+r*0.30, cx+r*0.15, cy+r*0.22, r*0.05, paint.Hex(0x4E342E))
	// hand on chin, tapping
	tap := 0.01 * math.Max(0, math.Sin(t*6))
	p.RoundRect(cx+r*0.05, cy+r*0.45-tap, r*0.55, r*0.30, r*0.10, paint.Hex(0xFFD54F))
	for i := 0; i < 3; i++ {
		p.RoundRect(cx+r*0.10+float64(i)*r*0.16, cy+r*0.30-tap, r*0.12, r*0.25, r*0.05, paint.Hex(0xFFD54F))
	}
	// thought bubbles
	for i := 0; i < 3; i++ {
		k := float64(i) / 3
		p.CircleA(cx+r*0.9+k*0.15, cy-r*0.6-k*0.2, 0.02+k*0.03, paint.White, 0.5+0.5*math.Sin(t*2-float64(i)))
	}
	dots := int(math.Mod(t*2, 4))
	s := ""
	for i := 0; i < dots; i++ {
		s += "."
	}
	p.Text(cx+r*1.15, cy-r*1.1, 0.06, s+"?", paint.White)
}

func drawJoy(p *paint.Painter, t, dt float64) {
	cx, cy, r := 0.5, 0.5, 0.40
	shake := 0.01 * math.Sin(t*25)
	tilt := 0.15 * math.Sin(t*3)
	faceBase(p, cx+shake, cy, r)
	// squeezed eyes
	for _, s := range []float64{-1, 1} {
		p.Arc(cx+shake+s*r*0.36, cy-r*0.10+tilt*0.05, r*0.14, math.Pi+0.3, 2*math.Pi-0.3, r*0.045, paint.Hex(0x4E342E))
	}
	// wide open laughing mouth
	open := 0.6 + 0.4*math.Abs(math.Sin(t*6))
	p.Arc(cx+shake, cy+r*0.05, r*0.45, 0.15, math.Pi-0.15, r*0.06, paint.Hex(0x4E342E))
	p.Shade(cx+shake-r*0.45, cy+r*0.05, r*0.9, r*0.45*open, func(u, v float64) (paint.RGB, float64) {
		x := (u - 0.5) * 2
		if x*x+v*v > 1 {
			return paint.Black, 0
		}
		if v < 0.25 {
			return paint.White, 1
		}
		return paint.Hex(0x4E342E), 1
	})
	p.Ellipse(cx+shake, cy+r*0.35*open, r*0.25, r*0.10*open, paint.Hex(0xE57373))
	// tears streaming
	for _, s := range []float64{-1, 1} {
		for i := 0; i < 3; i++ {
			k := math.Mod(t*1.2+float64(i)/3, 1)
			x := cx + shake + s*(r*0.62+k*0.1)
			y := cy - r*0.05 + k*0.25
			p.Ellipse(x, y, 0.015, 0.03, paint.Mix(paint.Black, paint.SkyBlue, 1-k))
		}
		p.Ellipse(cx+shake+s*r*0.62, cy-r*0.02, r*0.08, r*0.12, paint.SkyBlue)
	}
	p.TextCentred(0.5, 0.05, 0.06+0.01*math.Sin(t*10), "HAHAHA", paint.Gold)
}

type partyFace struct{ confetti *paint.System }

func (pf *partyFace) Draw(p *paint.Painter, t, dt float64) {
	cx, cy, r := 0.5, 0.55, 0.36
	tilt := 0.1 * math.Sin(t*2)
	faceBase(p, cx, cy, r)
	eyes(p, cx, cy, r, blink(t, 3, 0.4), 0, 0)
	p.Arc(cx, cy+r*0.10, r*0.45, 0.3, math.Pi-0.3, r*0.055, paint.Hex(0x4E342E))
	// hat
	q := func(x, y float64) (float64, float64) { return paint.Rot(cx, cy-r*0.9, x, y, tilt) }
	x0, y0 := q(cx-r*0.35, cy-r*0.85)
	x1, y1 := q(cx+r*0.35, cy-r*0.85)
	x2, y2 := q(cx, cy-r*1.6)
	p.Triangle(x0, y0, x1, y1, x2, y2, paint.Hex(0x7E57C2))
	for i := 1; i < 4; i++ {
		f := float64(i) / 4
		ax, ay := q(cx-r*0.35*(1-f), cy-r*0.85-r*0.75*f)
		bx, by := q(cx+r*0.35*(1-f), cy-r*0.85-r*0.75*f)
		p.Line(ax, ay, bx, by, 0.015, paint.Gold)
	}
	p.Circle(x2, y2, 0.03, paint.Gold)
	// horn blows and unrolls
	blow := math.Max(0, math.Sin(t*3))
	p.Line(cx+r*0.30, cy+r*0.30, cx+r*0.70, cy+r*0.42, r*0.12, paint.Hex(0xEF5350))
	p.Arc(cx+r*0.70, cy+r*0.42+0.05*blow, 0.05*blow+0.01, 0, math.Pi, 0.02, paint.Hex(0xFFEE58))
	pf.confetti.Every(dt, 12, func() {
		pf.confetti.Emit(paint.Particle{X: pf.confetti.Rand(p.L, p.R), Y: p.T - 0.05, VY: pf.confetti.Rand(0.15, 0.3), VX: pf.confetti.Rand(-0.05, 0.05), Life: 5, Size: 0.02, Kind: pf.confetti.Rng.Intn(4), Spin: pf.confetti.Rand(0, 6)})
	})
	pf.confetti.Step(dt, 0, 0)
	cols := []paint.RGB{paint.Gold, paint.Pink, paint.SkyBlue, paint.Lime}
	for _, c := range pf.confetti.P {
		an := c.Spin + c.Age*4
		p.Line(c.X-c.Size*math.Cos(an), c.Y-c.Size*math.Sin(an)*0.5, c.X+c.Size*math.Cos(an), c.Y+c.Size*math.Sin(an)*0.5, 0.012, cols[c.Kind])
	}
}

func drawScream(p *paint.Painter, t, dt float64) {
	cx, cy, r := 0.5, 0.52, 0.38
	shake := 0.012 * math.Sin(t*35)
	faceBase(p, cx+shake, cy, r)
	// hands on cheeks
	for _, s := range []float64{-1, 1} {
		p.RoundRect(cx+shake+s*r*0.85-r*0.15, cy-r*0.1, r*0.30, r*0.55, r*0.08, paint.Hex(0xFFD54F))
	}
	// pale blue forehead
	p.Shade(cx+shake-r, cy-r, 2*r, r*0.6, func(u, v float64) (paint.RGB, float64) {
		x, y := (u-0.5)*2, v*0.6-1
		if x*x+y*y > 1 {
			return paint.Black, 0
		}
		return paint.Hex(0x90CAF9), 0.6 * (1 - v)
	})
	for _, s := range []float64{-1, 1} {
		ex := cx + shake + s*r*0.36
		p.Circle(ex, cy-r*0.12, r*0.14, paint.White)
		p.Circle(ex+0.01*wobble(t*3, 1), cy-r*0.12, r*0.05, paint.Hex(0x4E342E))
		p.Arc(ex, cy-r*0.30, r*0.14, math.Pi+0.3, 2*math.Pi-0.3, r*0.04, paint.Hex(0x4E342E))
	}
	open := 0.28 + 0.06*math.Sin(t*12)
	p.Ellipse(cx+shake, cy+r*0.38, r*0.18, r*open, paint.Hex(0x4E342E))
	// sweat drops flying
	for i := 0; i < 4; i++ {
		k := math.Mod(t*1.5+float64(i)/4, 1)
		s := float64(1 - 2*(i%2))
		p.Ellipse(cx+s*(r*0.7+k*0.25), cy-r*0.6-k*0.1+k*k*0.4, 0.012, 0.025, paint.Mix(paint.Black, paint.SkyBlue, 1-k))
	}
	p.TextCentred(0.5, 0.04+0.01*math.Sin(t*30), 0.07, "AAAH!", paint.Red)
}

func drawRobot(p *paint.Painter, t, dt float64) {
	metal := paint.Hex(0x90A4AE)
	dark := paint.Hex(0x546E7A)
	cx, cy := 0.5, 0.52
	// antenna with blinking light
	p.Line(cx, cy-0.30, cx, cy-0.44, 0.02, dark)
	on := math.Sin(t*6) > 0
	p.Circle(cx, cy-0.46, 0.035, paint.Mix(paint.DarkGrey, paint.Red, boolf(on)))
	if on {
		p.Glow(cx, cy-0.46, 0.09, paint.Red, 0.7)
	}
	p.RoundRect(cx-0.30, cy-0.30, 0.60, 0.56, 0.06, metal)
	p.RoundRect(cx-0.36, cy-0.05, 0.06, 0.16, 0.02, dark)
	p.RoundRect(cx+0.30, cy-0.05, 0.06, 0.16, 0.02, dark)
	// eyes: scanning bars
	for _, s := range []float64{-1, 1} {
		ex := cx + s*0.14
		p.RoundRect(ex-0.09, cy-0.18, 0.18, 0.12, 0.02, paint.Hex(0x263238))
		scan := 0.5 + 0.5*math.Sin(t*3+s)
		p.RoundRect(ex-0.08+scan*0.10, cy-0.16, 0.06, 0.08, 0.01, paint.Hex(0x00E5FF))
	}
	// mouth: an equaliser of lights
	for i := 0; i < 6; i++ {
		x := cx - 0.20 + float64(i)*0.08
		h := 0.02 + 0.06*math.Abs(math.Sin(t*5+float64(i)*1.1))
		p.RoundRect(x-0.02, cy+0.16-h/2, 0.04, h, 0.01, paint.Hex(0x69F0AE))
	}
	// side panel lights
	for i := 0; i < 3; i++ {
		lit := math.Sin(t*2+float64(i)*2) > 0.3
		p.Circle(cx+0.24, cy-0.20+float64(i)*0.06, 0.012, paint.Mix(paint.DarkGrey, paint.Gold, boolf(lit)))
	}
	if math.Mod(t, 3) < 1 {
		p.TextCentred(cx, 0.90, 0.05, "BEEP BOOP", paint.Hex(0x00E5FF))
	}
}

func boolf(b bool) float64 {
	if b {
		return 1
	}
	return 0
}

func drawAngry(p *paint.Painter, t, dt float64) {
	cx, cy, r := 0.5, 0.52, 0.38
	shake := 0.008 * math.Sin(t*40)
	heat := 0.5 + 0.5*math.Sin(t*2)
	p.Circle(cx+shake, cy, r, paint.Hex(0xD84315))
	p.Circle(cx+shake-r*0.05, cy-r*0.05, r*0.94, paint.Mix(paint.Hex(0xFF7043), paint.Hex(0xE53935), heat))
	// furrowed brows
	for _, s := range []float64{-1, 1} {
		p.Line(cx+shake+s*r*0.15, cy-r*0.25, cx+shake+s*r*0.5, cy-r*0.42, r*0.06, paint.Hex(0x4E342E))
		ex := cx + shake + s*r*0.34
		p.Ellipse(ex, cy-r*0.08, r*0.10, r*0.10, paint.Hex(0x4E342E))
	}
	p.Arc(cx+shake, cy+r*0.45, r*0.30, math.Pi+0.4, 2*math.Pi-0.4, r*0.055, paint.Hex(0x4E342E))
	// steam from the ears and a throbbing vein
	for i := 0; i < 4; i++ {
		k := math.Mod(t*0.8+float64(i)/4, 1)
		s := float64(1 - 2*(i%2))
		p.CircleA(cx+s*(r*0.9+k*0.1)+0.02*math.Sin(k*10), cy-r*0.2-k*0.35, 0.02+k*0.03, paint.LightGrey, (1-k)*0.6)
	}
	vx := cx + shake + r*0.55
	vy := cy - r*0.62
	vs := 1 + 0.2*heat
	p.Line(vx, vy, vx+0.03*vs, vy-0.03*vs, 0.01, paint.Hex(0x4E342E))
	p.Line(vx+0.03*vs, vy-0.03*vs, vx+0.02*vs, vy-0.07*vs, 0.01, paint.Hex(0x4E342E))
	p.Line(vx-0.02*vs, vy-0.03*vs, vx+0.03*vs, vy-0.03*vs, 0.01, paint.Hex(0x4E342E))
	p.TextCentred(0.5, 0.94, 0.05, "GRRR", paint.Red)
}
