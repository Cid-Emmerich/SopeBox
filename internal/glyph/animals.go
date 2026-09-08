package glyph

import (
	"math"

	"github.com/Cid-Emmerich/SopeBox/internal/paint"
)

func initanimals() {
	add("animals",
		Def{Emoji: "🐱", Name: "cat", Blurb: "a cat watching something only it can see", Aliases: []string{"kitty", "kitten", "meow", "cat face", "🐈"}, New: stateless(drawCat)},
		Def{Emoji: "🐶", Name: "dog", Blurb: "a happy dog with floppy ears", Aliases: []string{"puppy", "doggo", "woof", "dog face", "🐕"}, New: stateless(drawDog)},
		Def{Emoji: "👻", Name: "ghost", Blurb: "a friendly ghost saying boo", Aliases: []string{"boo", "spooky", "halloween", "spirit"}, New: stateless(drawGhost)},
		Def{Emoji: "💀", Name: "skull", Blurb: "a chattering skull with glowing eyes", Aliases: []string{"skeleton", "dead", "death", "☠️", "bones"}, New: stateless(drawSkull)},
		Def{Emoji: "🐍", Name: "snake", Blurb: "a snake slithering by", Aliases: []string{"serpent", "python", "slither", "hiss"}, New: stateless(drawSnake)},
		Def{Emoji: "🦋", Name: "butterfly", Blurb: "a butterfly fluttering", Aliases: []string{"flutter", "moth", "wings", "insect"}, New: stateless(drawButterfly)},
		Def{Emoji: "🐟", Name: "fish", Blurb: "a fish swimming back and forth", Aliases: []string{"swim", "tropical", "🐠", "🐡", "aquarium"}, New: func() Anim { return &fish{bubbles: paint.NewSystem(71)} }},
		Def{Emoji: "🐝", Name: "bee", Blurb: "a bee buzzing a figure of eight", Aliases: []string{"buzz", "honey", "bumblebee", "wasp"}, New: stateless(drawBee)},
		Def{Emoji: "🐸", Name: "frog", Blurb: "a frog catching flies", Aliases: []string{"toad", "ribbit", "frog face", "kermit"}, New: func() Anim { return &frog{sys: paint.NewSystem(72)} }},
		Def{Emoji: "🐢", Name: "turtle", Blurb: "a turtle in no hurry", Aliases: []string{"tortoise", "slow", "shell"}, New: stateless(drawTurtle)},
		Def{Emoji: "🦀", Name: "crab", Blurb: "a crab scuttling sideways", Aliases: []string{"crustacean", "beach", "pinch", "rust"}, New: stateless(drawCrab)},
		Def{Emoji: "🐙", Name: "octopus", Blurb: "an octopus waving all eight arms", Aliases: []string{"squid", "tentacles", "🦑", "kraken"}, New: stateless(drawOctopus)},
		Def{Emoji: "🦉", Name: "owl", Blurb: "an owl blinking and swivelling its head", Aliases: []string{"hoot", "night bird", "wise"}, New: stateless(drawOwl)},
	)
}

func drawCat(p *paint.Painter, t, dt float64) {
	fur := paint.Hex(0xFFA726)
	dark := paint.Hex(0xEF6C00)
	cx, cy := 0.5, 0.52
	// ears twitch now and then
	tw := 0.0
	if math.Mod(t, 2.7) < 0.3 {
		tw = 0.12 * math.Sin(math.Mod(t, 2.7)/0.3*math.Pi*2)
	}
	for _, s := range []float64{-1, 1} {
		bx := cx + s*0.20
		tx, ty := paint.Rot(bx, cy-0.18, bx+s*0.08, cy-0.44, s*tw)
		p.Triangle(bx-0.12, cy-0.18, bx+0.12, cy-0.18, tx, ty, fur)
		p.Triangle(bx-0.06, cy-0.20, bx+0.06, cy-0.20, tx*0.6+bx*0.4, ty*0.6+(cy-0.2)*0.4, paint.Pink)
	}
	p.Circle(cx, cy, 0.32, fur)
	for i := 0; i < 3; i++ {
		x := cx - 0.06 + float64(i)*0.06
		p.Line(x, cy-0.32, x+0.02*float64(i-1), cy-0.22, 0.02, dark)
	}
	// eyes: pupils drift, blink
	b := blink(t, 3.5, 0.2)
	look := 0.02 * wobble(t*0.6, 1)
	for _, s := range []float64{-1, 1} {
		ex := cx + s*0.12
		p.Ellipse(ex, cy-0.03, 0.065, 0.07*b+0.004, paint.Hex(0x7CB342))
		p.Ellipse(ex+look, cy-0.03, 0.02+0.02*(1-b), 0.06*b+0.004, paint.Hex(0x212121))
		p.Circle(ex+look-0.01, cy-0.06, 0.012*b, paint.White)
	}
	// nose, mouth, whiskers
	p.Triangle(cx-0.025, cy+0.06, cx+0.025, cy+0.06, cx, cy+0.09, paint.Pink)
	p.Arc(cx-0.035, cy+0.09, 0.035, 0.1, math.Pi-0.1, 0.012, dark)
	p.Arc(cx+0.035, cy+0.09, 0.035, 0.1, math.Pi-0.1, 0.012, dark)
	wig := 0.01 * math.Sin(t*5)
	for i := 0; i < 3; i++ {
		y := cy + 0.05 + float64(i)*0.04
		p.Line(cx-0.10, y, cx-0.34, y-0.03+float64(i)*0.03+wig, 0.008, paint.LightGrey)
		p.Line(cx+0.10, y, cx+0.34, y-0.03+float64(i)*0.03-wig, 0.008, paint.LightGrey)
	}
	// something to watch: a fly circling
	fa := t * 2.5
	p.Dot(cx+0.42*math.Cos(fa), cy-0.35+0.1*math.Sin(fa*2), 0.012, paint.DarkGrey)
}

func drawDog(p *paint.Painter, t, dt float64) {
	fur := paint.Hex(0xD7A86E)
	ear := paint.Hex(0x8D6E63)
	bob := 0.015 * math.Sin(t*6)
	tilt := 0.1 * math.Sin(t*0.9)
	cx, cy := 0.5, 0.52+bob
	q := func(x, y float64) (float64, float64) { return paint.Rot(cx, cy, x, y, tilt) }
	// floppy ears swing with the bob
	for _, s := range []float64{-1, 1} {
		ex, ey := q(cx+s*0.30, cy-0.05-bob*3)
		p.Ellipse(ex, ey, 0.10, 0.20, ear)
	}
	hx, hy := q(cx, cy)
	p.Circle(hx, hy, 0.30, fur)
	p.Ellipse(hx, hy+0.13, 0.16, 0.12, paint.Mix(fur, paint.White, 0.4))
	b := blink(t, 4, 0.5)
	for _, s := range []float64{-1, 1} {
		ex, ey := q(cx+s*0.11, cy-0.06)
		p.Ellipse(ex, ey, 0.04, 0.05*b+0.004, paint.Hex(0x212121))
		p.Circle(ex-0.012, ey-0.015, 0.012*b, paint.White)
	}
	nx, ny := q(cx, cy+0.08)
	p.Ellipse(nx, ny, 0.05, 0.035, paint.Hex(0x212121))
	// tongue wags
	tw := 0.02 * math.Sin(t*8)
	tx, ty := q(cx+0.04+tw, cy+0.22)
	p.Ellipse(tx, ty, 0.045, 0.07, paint.Hex(0xF06292))
	p.Arc(hx, hy+0.10, 0.08, 0.3, math.Pi-0.3, 0.015, paint.Hex(0x5D4037))
	// happy motion marks
	if math.Sin(t*6) > 0.7 {
		p.Line(cx-0.40, cy-0.28, cx-0.36, cy-0.24, 0.012, paint.Gold)
		p.Line(cx+0.40, cy-0.28, cx+0.36, cy-0.24, 0.012, paint.Gold)
	}
}

func drawGhost(p *paint.Painter, t, dt float64) {
	bob := 0.03 * math.Sin(t*1.8)
	cx, cy := 0.5+0.02*math.Sin(t*0.7), 0.45+bob
	white := paint.Hex(0xF5F5F5)
	const period = 4.0
	k := math.Mod(t, period)
	s := 1.0
	if k < 0.6 {
		s = pop(k)
	}
	w, h := 0.26*s, 0.34*s
	p.Circle(cx, cy-0.02, w, white)
	p.Rect(cx-w, cy-0.02, 2*w, h, white)
	// wavy hem
	p.Shade(cx-w, cy-0.02+h, 2*w, 0.08*s, func(u, v float64) (paint.RGB, float64) {
		hem := 0.5 + 0.5*math.Sin(u*3*2*math.Pi+t*6)
		if v < hem {
			return white, 1
		}
		return white, 0
	})
	// eyes dart, mouth "o"
	look := 0.03 * wobble(t, 3)
	for _, sd := range []float64{-1, 1} {
		p.Ellipse(cx+sd*0.09*s+look, cy-0.03, 0.045*s, 0.06*s, paint.Hex(0x212121))
		p.Circle(cx+sd*0.09*s+look+0.012, cy-0.05, 0.012*s, white)
	}
	mo := 0.03 + 0.03*math.Max(0, math.Sin(t*2))
	if k < 0.8 {
		mo = 0.07
	}
	p.Ellipse(cx+look*0.5, cy+0.12*s, 0.035*s, mo*s, paint.Hex(0x212121))
	// arms
	p.Ellipse(cx-w-0.04, cy+0.06+0.02*math.Sin(t*3), 0.06, 0.035, white)
	p.Ellipse(cx+w+0.04, cy+0.06-0.02*math.Sin(t*3), 0.06, 0.035, white)
	if k < 0.9 {
		ss := pop(k)
		p.TextCentred(cx+0.30, 0.10, 0.10*ss, "BOO!", paint.Mix(paint.Black, paint.LightGrey, 1-smooth((k-0.5)/0.4)))
	}
}

func drawSkull(p *paint.Painter, t, dt float64) {
	bone := paint.Hex(0xEEEEEE)
	cx, cy := 0.5, 0.42
	// chatter in bursts
	chat := 0.0
	if math.Mod(t, 3) < 1.1 {
		chat = 0.04 * math.Abs(math.Sin(t*22))
	}
	p.Circle(cx, cy, 0.30, bone)
	p.RoundRect(cx-0.20, cy+0.12, 0.40, 0.18, 0.05, bone)
	// jaw
	jy := cy + 0.30 + chat
	p.RoundRect(cx-0.17, jy, 0.34, 0.12, 0.04, bone)
	for i := 0; i < 5; i++ {
		x := cx - 0.13 + float64(i)*0.065
		p.Rect(x-0.02, cy+0.26, 0.04, 0.05, bone)
		p.Rect(x-0.02, jy-0.01, 0.04, 0.04, paint.Mix(bone, paint.Grey, 0.3))
		p.Line(x+0.032, cy+0.26, x+0.032, jy+0.03, 0.006, paint.Hex(0x9E9E9E))
	}
	// sockets with glowing pupils
	glow := 0.5 + 0.5*math.Sin(t*3)
	for _, s := range []float64{-1, 1} {
		ex := cx + s*0.12
		p.Circle(ex, cy-0.02, 0.085, paint.Hex(0x111111))
		p.Glow(ex, cy-0.02, 0.07, paint.Red, glow)
		p.Circle(ex+0.01*wobble(t, 4), cy-0.02, 0.025, paint.Mix(paint.Red, paint.White, glow*0.5))
	}
	p.Triangle(cx-0.035, cy+0.16, cx+0.035, cy+0.16, cx, cy+0.08, paint.Hex(0x111111))
	// crack
	p.Line(cx+0.08, cy-0.30, cx+0.14, cy-0.20, 0.008, paint.Grey)
	p.Line(cx+0.14, cy-0.20, cx+0.11, cy-0.12, 0.008, paint.Grey)
}

func drawSnake(p *paint.Painter, t, dt float64) {
	green := paint.Hex(0x66BB6A)
	dark := paint.Hex(0x2E7D32)
	w := p.R - p.L
	// the body is a chain of discs along a travelling sine
	n := 24
	head := p.L - 0.2 + math.Mod(t*0.35, w+0.8)
	var hx, hy, hdx, hdy float64
	for i := n; i >= 0; i-- {
		f := float64(i) / float64(n)
		x := head - f*0.85
		y := 0.55 + 0.12*math.Sin(x*9-t*4)
		r := 0.045 * (1 - f*0.7)
		c := green
		if i%2 == 0 {
			c = dark
		}
		p.Circle(x, y, r, c)
		if i == 0 {
			hx, hy = x, y
			hdy = 0.12 * 9 * math.Cos(x*9-t*4)
			hdx = 1
		}
	}
	// head
	l := math.Hypot(hdx, hdy)
	hdx, hdy = hdx/l, hdy/l
	p.Ellipse(hx+hdx*0.04, hy+hdy*0.04, 0.07, 0.05, green)
	p.Circle(hx+0.01-hdy*0.03, hy-0.02, 0.012, paint.Hex(0x212121))
	// tongue flicks
	if math.Mod(t, 1.4) < 0.3 {
		tx, ty := hx+hdx*0.11, hy+hdy*0.11
		p.Line(tx, ty, tx+hdx*0.06, ty+hdy*0.06, 0.008, paint.Red)
		p.Line(tx+hdx*0.06, ty+hdy*0.06, tx+hdx*0.09-hdy*0.02, ty+hdy*0.09+hdx*0.02, 0.008, paint.Red)
		p.Line(tx+hdx*0.06, ty+hdy*0.06, tx+hdx*0.09+hdy*0.02, ty+hdy*0.09-hdx*0.02, 0.008, paint.Red)
	}
	ground(p, 0.82, 0, t, paint.Hex(0x795548))
	if math.Mod(t, 3) < 0.8 {
		p.Text(hx+0.05, hy-0.16, 0.05, "HISS", paint.Mix(paint.Black, paint.Lime, 1-math.Mod(t, 3)/0.8))
	}
}

func drawButterfly(p *paint.Painter, t, dt float64) {
	flap := math.Abs(math.Cos(t * 7))
	cx := 0.5 + 0.10*math.Sin(t*0.7)
	cy := 0.5 + 0.05*math.Sin(t*1.9)
	tilt := 0.2 * math.Sin(t*0.7)
	blue := paint.Hex(0x42A5F5)
	purple := paint.Hex(0x7E57C2)
	for _, s := range []float64{-1, 1} {
		wx := s * flap
		// upper and lower wings as rotated ellipses approximated by polygons
		wing := func(rx, ry, ox, oy float64, c paint.RGB) {
			var pts []float64
			for i := 0; i < 20; i++ {
				a := float64(i) / 20 * 2 * math.Pi
				x := cx + wx*(ox+rx*math.Cos(a))
				y := cy + oy + ry*math.Sin(a)
				x, y = paint.Rot(cx, cy, x, y, tilt)
				pts = append(pts, x, y)
			}
			p.Poly(c, pts...)
		}
		wing(0.20, 0.17, 0.22, -0.12, blue)
		wing(0.15, 0.13, 0.17, 0.14, purple)
		wing(0.08, 0.06, 0.24, -0.12, paint.Mix(blue, paint.White, 0.5))
		wing(0.05, 0.04, 0.18, 0.14, paint.Mix(purple, paint.White, 0.5))
	}
	bx0, by0 := paint.Rot(cx, cy, cx, cy-0.22, tilt)
	bx1, by1 := paint.Rot(cx, cy, cx, cy+0.26, tilt)
	p.Line(bx0, by0, bx1, by1, 0.05, paint.Hex(0x37474F))
	p.Circle(bx0, by0, 0.035, paint.Hex(0x37474F))
	for _, s := range []float64{-1, 1} {
		ax, ay := paint.Rot(cx, cy, cx+s*0.08, cy-0.34+0.01*math.Sin(t*5), tilt)
		p.Line(bx0, by0, ax, ay, 0.01, paint.Hex(0x37474F))
		p.Circle(ax, ay, 0.015, paint.Hex(0x37474F))
	}
}

type fish struct{ bubbles *paint.System }

func (f *fish) Draw(p *paint.Painter, t, dt float64) {
	w := p.R - p.L
	// water line
	for x := p.L; x < p.R; x += 0.02 {
		p.Dot(x, 0.14+0.01*math.Sin(x*20+t*3), 0.012, paint.Hex(0x4FC3F7))
	}
	k := math.Mod(t*0.25, 2)
	dir := 1.0
	f0 := k
	if k > 1 {
		dir = -1
		f0 = 2 - k
	}
	cx := p.L + 0.2 + (w-0.4)*f0
	cy := 0.52 + 0.04*math.Sin(t*2)
	orange := paint.Hex(0xFF7043)
	// tail wags
	tw := 0.05 * math.Sin(t*10)
	p.Triangle(cx-dir*0.18, cy, cx-dir*0.34, cy-0.12+tw, cx-dir*0.34, cy+0.12+tw, orange)
	p.Ellipse(cx, cy, 0.22, 0.13, orange)
	p.Triangle(cx-0.02, cy-0.10, cx+0.08, cy-0.10, cx+0.02, cy-0.20, paint.Hex(0xFF8A65))
	p.Ellipse(cx-dir*0.02, cy+0.02, 0.06, 0.04, paint.Hex(0xFFAB91))
	p.Circle(cx+dir*0.12, cy-0.03, 0.03, paint.White)
	p.Circle(cx+dir*0.13, cy-0.03, 0.015, paint.Hex(0x212121))
	f.bubbles.Every(dt, 3, func() {
		f.bubbles.Emit(paint.Particle{X: cx + dir*0.22, Y: cy, VY: -0.15, Life: 2.5, Size: f.bubbles.Rand(0.008, 0.02), Spin: f.bubbles.Rand(0, 6)})
	})
	f.bubbles.Step(dt, 0, 0)
	for _, q := range f.bubbles.P {
		if q.Y < 0.16 {
			continue
		}
		p.Ring(q.X+0.02*math.Sin(q.Age*3+q.Spin), q.Y, q.Size, q.Size*0.5, paint.Hex(0x81D4FA))
	}
	// seaweed
	for i := 0; i < 4; i++ {
		x := p.L + 0.1 + w*float64(i)/4
		var px, py = x, 0.95
		for j := 0; j < 6; j++ {
			nx := x + 0.03*math.Sin(t*2+float64(j)*0.8+float64(i))
			ny := 0.95 - float64(j+1)*0.05
			p.Line(px, py, nx, ny, 0.02, paint.Hex(0x2E7D32))
			px, py = nx, ny
		}
	}
}

func drawBee(p *paint.Painter, t, dt float64) {
	// figure of eight path
	a := t * 1.4
	cx := 0.5 + 0.34*math.Sin(a)
	cy := 0.5 + 0.16*math.Sin(2*a)
	dx, dy := math.Cos(a), 2*math.Cos(2*a)*0.16/0.34
	ang := math.Atan2(dy, dx)
	dir := 1.0
	if dx < 0 {
		dir = -1
		ang += math.Pi
	}
	// dotted trail
	for i := 1; i < 12; i++ {
		ta := a - float64(i)*0.08
		tx := 0.5 + 0.34*math.Sin(ta)
		ty := 0.5 + 0.16*math.Sin(2*ta)
		p.Dot(tx, ty, 0.01, paint.Mix(paint.Black, paint.Grey, 1-float64(i)/12))
	}
	q := func(x, y float64) (float64, float64) { return paint.Rot(cx, cy, cx+x*dir, cy+y, ang*0.3) }
	// wings
	fl := 0.6 + 0.4*math.Abs(math.Sin(t*40))
	for _, s := range []float64{-1, 1} {
		wx, wy := q(-0.02+s*0.05, -0.10)
		p.Ellipse(wx, wy, 0.06, 0.07*fl, paint.Mix(paint.SkyBlue, paint.White, 0.6))
	}
	// striped body
	bx, by := q(0, 0)
	p.Shade(bx-0.14, by-0.09, 0.28, 0.18, func(u, v float64) (paint.RGB, float64) {
		x, y := (u-0.5)*2, (v-0.5)*2
		if x*x+y*y > 1 {
			return paint.Black, 0
		}
		if int(u*5)%2 == 1 {
			return paint.Hex(0x212121), 1
		}
		return paint.Hex(0xFFC107), 1
	})
	hx, hy := q(0.15, 0)
	p.Circle(hx, hy, 0.05, paint.Hex(0x212121))
	p.Circle(hx+dir*0.015, hy-0.015, 0.012, paint.White)
	sx, sy := q(-0.14, 0)
	p.Triangle(sx, sy-0.02, sx, sy+0.02, sx-dir*0.05, sy, paint.Hex(0x212121))
	if math.Sin(t*1.4) > 0.9 {
		p.Text(cx+0.08, cy-0.2, 0.05, "BZZ", paint.Gold)
	}
}

type frog struct {
	sys   *paint.System
	flyX  float64
	flyY  float64
	catch float64
}

func (f *frog) Draw(p *paint.Painter, t, dt float64) {
	green := paint.Hex(0x66BB6A)
	dark := paint.Hex(0x388E3C)
	cx, cy := 0.5, 0.60
	// fly buzzes until caught every 4s
	const period = 4.0
	k := math.Mod(t, period)
	if k < 0.05 && f.catch >= 0 {
		f.catch = -1
	}
	fx := cx + 0.32*math.Cos(t*2.3) + 0.05*math.Sin(t*11)
	fy := 0.28 + 0.08*math.Sin(t*3.1)
	tongue := 0.0
	if k > 2.5 && k < 3.1 {
		e := (k - 2.5) / 0.6
		tongue = math.Sin(e * math.Pi)
		if e > 0.45 {
			f.catch = 1
		}
	}
	if f.catch < 0 {
		p.Ellipse(fx, fy, 0.02, 0.014, paint.Hex(0x212121))
		p.Ellipse(fx-0.015, fy-0.012, 0.012, 0.008, paint.Mix(paint.SkyBlue, paint.White, 0.5))
		p.Ellipse(fx+0.015, fy-0.012, 0.012, 0.008, paint.Mix(paint.SkyBlue, paint.White, 0.5))
	}
	p.Ellipse(cx, cy, 0.36, 0.26, green)
	// eyes on top
	b := blink(t, 3, 0.7)
	for _, s := range []float64{-1, 1} {
		ex := cx + s*0.18
		p.Circle(ex, cy-0.22, 0.10, green)
		p.Ellipse(ex, cy-0.22, 0.07, 0.07*b+0.004, paint.White)
		look := 0.02 * (fx - cx)
		p.Ellipse(ex+look, cy-0.22, 0.03, 0.045*b+0.003, paint.Hex(0x212121))
	}
	// cheeks and mouth
	p.Circle(cx-0.22, cy+0.02, 0.05, paint.Mix(green, paint.Pink, 0.4))
	p.Circle(cx+0.22, cy+0.02, 0.05, paint.Mix(green, paint.Pink, 0.4))
	open := 0.02 + 0.1*tongue
	p.Arc(cx, cy+0.02, 0.18, 0.2, math.Pi-0.2, 0.02, dark)
	if tongue > 0 {
		p.Ellipse(cx, cy+0.08, 0.14, open, paint.Hex(0xB71C1C))
		tx, ty := lerp(cx, fx, tongue), lerp(cy+0.08, fy, tongue)
		p.Line(cx, cy+0.08, tx, ty, 0.035, paint.Hex(0xEF5350))
		p.Circle(tx, ty, 0.03, paint.Hex(0xEF5350))
	}
	// belly and feet
	p.Ellipse(cx, cy+0.14, 0.20, 0.08, paint.Mix(green, paint.White, 0.4))
	p.Ellipse(cx-0.25, cy+0.24, 0.10, 0.04, dark)
	p.Ellipse(cx+0.25, cy+0.24, 0.10, 0.04, dark)
	if k > 3.1 && k < 3.9 {
		p.TextCentred(cx, 0.06, 0.06, "RIBBIT", paint.Mix(paint.Black, paint.Lime, 1-(k-3.1)/0.8))
	}
	// lily pad
	p.Ellipse(cx, 0.90, 0.42, 0.05, paint.Hex(0x2E7D32))
}

func drawTurtle(p *paint.Painter, t, dt float64) {
	w := p.R - p.L
	x := p.L - 0.3 + math.Mod(t*0.08, w+0.6)
	cy := 0.62
	shell := paint.Hex(0x2E7D32)
	skin := paint.Hex(0x9CCC65)
	step := math.Sin(t * 3)
	// legs shuffle
	p.Ellipse(x-0.18, cy+0.10+0.01*step, 0.06, 0.04, skin)
	p.Ellipse(x+0.18, cy+0.10-0.01*step, 0.06, 0.04, skin)
	p.Ellipse(x-0.12+0.02*step, cy+0.12, 0.06, 0.04, skin)
	p.Ellipse(x+0.12-0.02*step, cy+0.12, 0.06, 0.04, skin)
	// shell with hex-ish plates
	p.Ellipse(x, cy, 0.28, 0.16, shell)
	for i := 0; i < 5; i++ {
		a := float64(i) * 2 * math.Pi / 5
		p.Circle(x+0.13*math.Cos(a), cy-0.02+0.07*math.Sin(a), 0.045, paint.Hex(0x43A047))
	}
	p.Circle(x, cy-0.02, 0.05, paint.Hex(0x43A047))
	p.Ellipse(x, cy+0.10, 0.30, 0.04, paint.Hex(0x1B5E20))
	// head bobs out
	hx := x + 0.30 + 0.02*math.Sin(t*1.5)
	p.Circle(hx, cy+0.02, 0.07, skin)
	p.Circle(hx+0.03, cy-0.01, 0.014, paint.Hex(0x212121))
	p.Arc(hx+0.02, cy+0.04, 0.025, 0.2, math.Pi-0.2, 0.008, paint.Hex(0x33691E))
	// tail
	p.Triangle(x-0.27, cy+0.02, x-0.27, cy+0.08, x-0.36, cy+0.06+0.01*step, skin)
	ground(p, 0.80, 0.08, t, paint.Hex(0x795548))
	p.TextCentred(0.5, 0.90, 0.05, "NO RUSH", paint.Grey)
}

func drawCrab(p *paint.Painter, t, dt float64) {
	w := p.R - p.L
	// scuttles sideways, pausing between dashes
	k := cycle(t, 3)
	x := 0.5 + 0.5*(w-0.4)*math.Sin(smooth(math.Mod(k*2, 1))*math.Pi+math.Floor(k*2)*math.Pi)
	x = p.L + 0.2 + (w-0.4)*(0.5+0.5*math.Sin(t*1.2))
	cy := 0.58
	red := paint.Hex(0xEF5350)
	dark := paint.Hex(0xC62828)
	leg := math.Sin(t * 12)
	for i := 0; i < 3; i++ {
		for _, s := range []float64{-1, 1} {
			ph := leg * float64(1-2*(i%2))
			bx := x + s*0.18
			by := cy + 0.02 + float64(i)*0.05
			ex := bx + s*(0.12+0.02*ph)
			ey := by + 0.06 + 0.02*ph
			p.Line(bx, by, ex, ey, 0.025, dark)
			p.Line(ex, ey, ex+s*0.03, ey+0.06, 0.02, dark)
		}
	}
	p.Ellipse(x, cy, 0.24, 0.15, red)
	// claws snap
	snap := 0.3 + 0.3*math.Max(0, math.Sin(t*5))
	for _, s := range []float64{-1, 1} {
		ax := x + s*0.30
		p.Line(x+s*0.18, cy-0.05, ax, cy-0.14, 0.035, dark)
		p.Circle(ax, cy-0.16, 0.06, red)
		p.Triangle(ax, cy-0.16, ax+s*0.10, cy-0.16-0.06*snap, ax+s*0.10, cy-0.16+0.02, red)
		p.Triangle(ax, cy-0.16, ax+s*0.10, cy-0.16+0.06*snap, ax+s*0.10, cy-0.16-0.02, red)
	}
	// eye stalks
	for _, s := range []float64{-1, 1} {
		p.Line(x+s*0.07, cy-0.10, x+s*0.09, cy-0.22, 0.02, dark)
		p.Circle(x+s*0.09, cy-0.23, 0.03, paint.White)
		p.Circle(x+s*0.09, cy-0.23, 0.014, paint.Hex(0x212121))
	}
	p.Arc(x, cy+0.02, 0.06, 0.3, math.Pi-0.3, 0.012, dark)
	ground(p, 0.86, 0, t, paint.Hex(0xE0C097))
}

func drawOctopus(p *paint.Painter, t, dt float64) {
	purple := paint.Hex(0xAB47BC)
	dark := paint.Hex(0x7B1FA2)
	cx, cy := 0.5, 0.42+0.02*math.Sin(t*1.5)
	// eight arms waving with staggered phases
	for i := 0; i < 8; i++ {
		f := (float64(i) + 0.5) / 8
		bx := cx - 0.26 + f*0.52
		px, py := bx, cy+0.18
		for j := 1; j <= 7; j++ {
			nx := bx + 0.05*float64(j)*math.Sin(t*3+float64(i)*0.8+float64(j)*0.5)*(f-0.5)*2 + 0.03*math.Sin(t*4+float64(j)+float64(i))
			ny := cy + 0.18 + float64(j)*0.055
			p.Line(px, py, nx, ny, 0.045*(1-float64(j)*0.1), purple)
			if j%2 == 0 {
				p.Circle(nx, ny, 0.012, paint.Pink)
			}
			px, py = nx, ny
		}
	}
	p.Circle(cx, cy, 0.28, purple)
	p.Ellipse(cx, cy+0.14, 0.26, 0.10, purple)
	b := blink(t, 4, 0.1)
	for _, s := range []float64{-1, 1} {
		p.Ellipse(cx+s*0.11, cy+0.02, 0.06, 0.07*b+0.004, paint.White)
		p.Ellipse(cx+s*0.11+0.01*wobble(t, 2), cy+0.03, 0.03, 0.04*b+0.003, paint.Hex(0x212121))
	}
	p.Arc(cx, cy+0.10, 0.06, 0.3, math.Pi-0.3, 0.015, dark)
	p.Circle(cx-0.20, cy+0.08, 0.035, paint.Mix(purple, paint.Pink, 0.5))
	p.Circle(cx+0.20, cy+0.08, 0.035, paint.Mix(purple, paint.Pink, 0.5))
	// bubbles
	for i := 0; i < 5; i++ {
		k := math.Mod(t*0.2+float64(i)/5, 1)
		p.Ring(cx+0.35*math.Cos(float64(i)*1.3)+0.02*math.Sin(k*8), 0.9-k*0.9, 0.012+k*0.015, 0.006, paint.Mix(paint.Black, paint.SkyBlue, 1-k))
	}
}

func drawOwl(p *paint.Painter, t, dt float64) {
	brown := paint.Hex(0x8D6E63)
	light := paint.Hex(0xD7CCC8)
	cx, cy := 0.5, 0.50
	// head swivels: turns briefly every few seconds
	turn := 0.0
	k := math.Mod(t, 5)
	if k > 3 && k < 4 {
		turn = 0.06 * math.Sin((k-3)*math.Pi)
	}
	stars(p, t, 14, 33)
	// branch
	p.Line(p.L, 0.90, p.R, 0.86, 0.05, paint.Hex(0x5D4037))
	p.Ellipse(cx, cy+0.18, 0.24, 0.30, brown)
	p.Ellipse(cx, cy+0.24, 0.16, 0.20, light)
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			p.Arc(cx-0.08+float64(j)*0.08, cy+0.14+float64(i)*0.09, 0.03, 0.2, math.Pi-0.2, 0.01, brown)
		}
	}
	// feet grip the branch
	for _, s := range []float64{-1, 1} {
		p.Line(cx+s*0.08, cy+0.46, cx+s*0.10, cy+0.50, 0.02, paint.Hex(0xFFB300))
		p.Line(cx+s*0.08, cy+0.46, cx+s*0.05, cy+0.50, 0.02, paint.Hex(0xFFB300))
	}
	hx := cx + turn
	p.Circle(hx, cy-0.06, 0.26, brown)
	for _, s := range []float64{-1, 1} {
		p.Triangle(hx+s*0.12, cy-0.28, hx+s*0.26, cy-0.16, hx+s*0.30, cy-0.36, brown)
	}
	b1, b2 := blink(t, 4, 0.2), blink(t, 4, 0.2)
	if math.Mod(t, 7) < 0.4 {
		b2 = 0.05 // a wink
	}
	for i, s := range []float64{-1, 1} {
		b := b1
		if i == 1 {
			b = b2
		}
		ex := hx + s*0.11
		p.Circle(ex, cy-0.06, 0.10, light)
		p.Ellipse(ex, cy-0.06, 0.075, 0.075*b+0.004, paint.Hex(0xFFB300))
		p.Ellipse(ex+0.01*wobble(t, 5), cy-0.06, 0.04, 0.045*b+0.003, paint.Hex(0x212121))
		p.Circle(ex-0.015, cy-0.08, 0.012*b, paint.White)
	}
	p.Triangle(hx-0.03, cy+0.02, hx+0.03, cy+0.02, hx, cy+0.09, paint.Hex(0xFFB300))
	if k > 1 && k < 1.8 {
		p.Text(hx+0.30, cy-0.30, 0.06, "HOOT", paint.Mix(paint.Black, paint.LightGrey, 1-(k-1)/0.8))
	}
}
