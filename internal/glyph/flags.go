package glyph

import (
	"math"

	"github.com/Cid-Emmerich/SopeBox/internal/paint"
)

func initflags() {
	add("flags",
		Def{Emoji: "🏁", Name: "chequered", Blurb: "the chequered flag waves the winner home", Aliases: []string{"checkered", "race", "finish", "flag"}, New: flag(patChequered, false)},
		Def{Emoji: "🚩", Name: "red flag", Blurb: "a red pennant flapping", Aliases: []string{"triangular flag", "pennant", "marker"}, New: flag(patRed, true)},
		Def{Emoji: "🏳️‍🌈", Name: "rainbow flag", Blurb: "pride stripes in the wind", Aliases: []string{"pride", "lgbt", "rainbow"}, New: flag(patRainbow, false)},
		Def{Emoji: "🏴‍☠️", Name: "pirate", Blurb: "the jolly roger", Aliases: []string{"jolly roger", "skull flag", "black flag"}, New: flag(patPirate, false)},
		Def{Emoji: "🇺🇸", Name: "usa", Blurb: "stars and stripes", Aliases: []string{"united states", "america", "us", "american flag"}, New: flag(patUSA, false)},
		Def{Emoji: "🇯🇵", Name: "japan", Blurb: "the rising sun", Aliases: []string{"japanese", "nippon", "jp"}, New: flag(patJapan, false)},
		Def{Emoji: "🇫🇷", Name: "france", Blurb: "le tricolore", Aliases: []string{"french", "fr"}, New: flag(vertical(paint.Hex(0x0055A4), paint.White, paint.Hex(0xEF4135)), false)},
		Def{Emoji: "🇮🇹", Name: "italy", Blurb: "il tricolore", Aliases: []string{"italian", "it"}, New: flag(vertical(paint.Hex(0x009246), paint.White, paint.Hex(0xCE2B37)), false)},
		Def{Emoji: "🇩🇪", Name: "germany", Blurb: "schwarz rot gold", Aliases: []string{"german", "de", "deutschland"}, New: flag(horizontal(paint.Black, paint.Hex(0xDD0000), paint.Hex(0xFFCE00)), false)},
		Def{Emoji: "🇺🇦", Name: "ukraine", Blurb: "blue sky over golden wheat", Aliases: []string{"ukrainian", "ua"}, New: flag(horizontal(paint.Hex(0x0057B7), paint.Hex(0xFFD700)), false)},
		Def{Emoji: "🇬🇧", Name: "uk", Blurb: "the union jack", Aliases: []string{"united kingdom", "britain", "british", "england", "gb"}, New: flag(patUK, false)},
		Def{Emoji: "🇧🇷", Name: "brazil", Blurb: "ordem e progresso", Aliases: []string{"brasil", "brazilian", "br"}, New: flag(patBrazil, false)},
		Def{Emoji: "🇨🇦", Name: "canada", Blurb: "the maple leaf", Aliases: []string{"canadian", "ca", "maple"}, New: flag(patCanada, false)},
	)
}

// pattern returns the colour of a flag at u,v (0..1 across the cloth) and
// whether that point is part of the cloth at all.
type pattern func(u, v, t float64) (paint.RGB, bool)

// flag builds a waving flag animation for a pattern. pennant flags are
// triangular.
func flag(pat pattern, pennant bool) func() Anim {
	return stateless(func(p *paint.Painter, t, dt float64) {
		const fx, fy, fw, fh = 0.19, 0.12, 0.72, 0.48
		// pole with a golden finial
		p.Line(0.16, 0.10, 0.16, 0.95, 0.028, paint.Hex(0x8D6E63))
		p.Circle(0.16, 0.085, 0.03, paint.Gold)
		p.Ellipse(0.16, 0.95, 0.10, 0.02, paint.DarkGrey)
		p.Shade(fx, fy-0.1, fw, fh+0.2, func(u, vv float64) (paint.RGB, float64) {
			// wave amplitude grows away from the pole
			ph := u*6.5 - t*4.5
			disp := 0.10 * u * math.Sin(ph)
			v := (vv*(fh+0.2) - 0.1 - disp) / fh
			if v < 0 || v > 1 {
				return paint.Black, 0
			}
			if pennant && v > 1-u*1.0 && v < u {
				// keep the pointed end: cloth only between the two edges
			}
			if pennant {
				// triangular: cloth exists where |v-0.5| < 0.5*(1-u)
				if math.Abs(v-0.5) > 0.5*(1-u) {
					return paint.Black, 0
				}
			}
			c, ok := pat(u, v, t)
			if !ok {
				return paint.Black, 0
			}
			// light follows the ripple
			light := 0.82 + 0.24*math.Cos(ph+0.6)
			return c.Scale(light), 1
		})
	})
}

func patChequered(u, v, _ float64) (paint.RGB, bool) {
	if (int(u*8)+int(v*5))%2 == 0 {
		return paint.White, true
	}
	return paint.Hex(0x111111), true
}

func patRed(u, v, _ float64) (paint.RGB, bool) { return paint.Hex(0xE53935), true }

var prideStripes = []paint.RGB{paint.Hex(0xE40303), paint.Hex(0xFF8C00), paint.Hex(0xFFED00), paint.Hex(0x008026), paint.Hex(0x24408E), paint.Hex(0x732982)}

func patRainbow(u, v, _ float64) (paint.RGB, bool) {
	i := int(v * 6)
	if i > 5 {
		i = 5
	}
	return prideStripes[i], true
}

func patPirate(u, v, t float64) (paint.RGB, bool) {
	// skull in the middle with crossed bones under it (aspect ≈ 3:2)
	x, y := (u-0.5)*1.5, v-0.42
	if x*x+y*y < 0.20*0.20 { // cranium
		// eye sockets and nose
		for _, ex := range []float64{-0.07, 0.07} {
			if (x-ex)*(x-ex)+(y+0.03)*(y+0.03) < 0.045*0.045 {
				return paint.Hex(0x111111), true
			}
		}
		if math.Abs(x) < 0.02 && y > 0.04 && y < 0.09 {
			return paint.Hex(0x111111), true
		}
		return paint.White, true
	}
	// jaw with teeth
	if math.Abs(x) < 0.12 && y > 0.16 && y < 0.26 {
		if int((x+0.12)/0.04)%2 == 0 || y < 0.20 {
			return paint.White, true
		}
		return paint.Hex(0x111111), true
	}
	// bones: two diagonals below
	by := y - 0.30
	for _, s := range []float64{1, -1} {
		d := math.Abs(by-s*x*0.6) / math.Sqrt(1+0.36)
		if d < 0.025 && math.Abs(x) < 0.30 {
			return paint.White, true
		}
	}
	return paint.Hex(0x111111), true
}

func patUSA(u, v, _ float64) (paint.RGB, bool) {
	red, blue := paint.Hex(0xB22234), paint.Hex(0x3C3B6E)
	if u < 0.4 && v < 7.0/13 {
		// star field: a grid of dots
		gx := math.Mod(u/0.4*6, 1)
		gy := math.Mod(v/(7.0/13)*5, 1)
		if (gx-0.5)*(gx-0.5)+(gy-0.5)*(gy-0.5) < 0.05 {
			return paint.White, true
		}
		return blue, true
	}
	if int(v*13)%2 == 0 {
		return red, true
	}
	return paint.White, true
}

func patJapan(u, v, _ float64) (paint.RGB, bool) {
	x, y := (u-0.5)*1.5, v-0.5
	if x*x+y*y < 0.3*0.3 {
		return paint.Hex(0xBC002D), true
	}
	return paint.White, true
}

func vertical(cs ...paint.RGB) pattern {
	return func(u, v, _ float64) (paint.RGB, bool) {
		i := int(u * float64(len(cs)))
		if i >= len(cs) {
			i = len(cs) - 1
		}
		return cs[i], true
	}
}

func horizontal(cs ...paint.RGB) pattern {
	return func(u, v, _ float64) (paint.RGB, bool) {
		i := int(v * float64(len(cs)))
		if i >= len(cs) {
			i = len(cs) - 1
		}
		return cs[i], true
	}
}

func patUK(u, v, _ float64) (paint.RGB, bool) {
	red, blue := paint.Hex(0xC8102E), paint.Hex(0x012169)
	x, y := (u-0.5)*2, (v-0.5)*2 // -1..1, aspect handled loosely
	// red cross
	if math.Abs(x) < 0.12 || math.Abs(y) < 0.18 {
		return red, true
	}
	// white cross border
	if math.Abs(x) < 0.22 || math.Abs(y) < 0.32 {
		return paint.White, true
	}
	// diagonals
	d1 := math.Abs(y-x*0.667) / 1.2
	d2 := math.Abs(y+x*0.667) / 1.2
	if d1 < 0.06 || d2 < 0.06 {
		return red, true
	}
	if d1 < 0.15 || d2 < 0.15 {
		return paint.White, true
	}
	return blue, true
}

func patBrazil(u, v, _ float64) (paint.RGB, bool) {
	green, yellow, blue := paint.Hex(0x009C3B), paint.Hex(0xFFDF00), paint.Hex(0x002776)
	x, y := (u-0.5)*1.5, (v-0.5)*1.0
	if x*x+y*y < 0.17*0.17 {
		if math.Abs(y+x*0.15+0.02) < 0.025 {
			return paint.White, true
		}
		return blue, true
	}
	if math.Abs(x)/0.62+math.Abs(y)/0.42 < 1 {
		return yellow, true
	}
	return green, true
}

func patCanada(u, v, _ float64) (paint.RGB, bool) {
	red := paint.Hex(0xFF0000)
	if u < 0.25 || u > 0.75 {
		return red, true
	}
	// a stylised maple leaf: a rotated square body with points
	x, y := (u-0.5)*3.0, (v-0.5)*2.0
	ax, ay := math.Abs(x), y
	leaf := (ax+math.Abs(ay+0.05) < 0.55 && ay < 0.35) || (ax < 0.06 && ay < 0.75 && ay > -0.75) ||
		(ax < 0.18 && ay < -0.3 && ay > -0.8 && ax < (-0.3-ay)*0.5) ||
		(ax > 0.25 && ax < 0.62 && math.Abs(ay+0.05) < 0.12+(0.62-ax)*0.6)
	if leaf {
		return red, true
	}
	return paint.White, true
}
