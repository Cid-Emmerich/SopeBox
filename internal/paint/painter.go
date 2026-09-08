package paint

import "math"

// Painter draws vector shapes into a Bitmap using a resolution independent
// coordinate system: the unit square (0,0)-(1,1) is mapped onto the largest
// centred square of the target region (scaled by zoom). y grows downward.
// Shapes may extend outside the unit square; L/R/T/B tell a glyph how far
// the visible region reaches in unit coordinates.
type Painter struct {
	B           *Bitmap
	S           float64 // pixels per unit
	OX, OY      float64 // pixel offset of unit origin
	Mirror      bool
	L, R, T, Bt float64 // visible extents in unit coords
	x0, y0      int     // clip rect in pixels
	x1, y1      int
}

// NewPainter maps the unit square into the whole bitmap.
func NewPainter(b *Bitmap, zoom float64, mirror bool) *Painter {
	return NewPainterRegion(b, 0, 0, b.W, b.H, zoom, mirror)
}

// NewPainterRegion maps the unit square into a pixel rectangle of b.
func NewPainterRegion(b *Bitmap, x0, y0, x1, y1 int, zoom float64, mirror bool) *Painter {
	w, h := float64(x1-x0), float64(y1-y0)
	if zoom <= 0 {
		zoom = 1
	}
	s := math.Min(w, h) * zoom
	if s < 1 {
		s = 1
	}
	p := &Painter{B: b, S: s, Mirror: mirror, x0: x0, y0: y0, x1: x1, y1: y1}
	p.OX = float64(x0) + (w-s)/2
	p.OY = float64(y0) + (h-s)/2
	p.L = (float64(x0) - p.OX) / s
	p.R = (float64(x1) - p.OX) / s
	p.T = (float64(y0) - p.OY) / s
	p.Bt = (float64(y1) - p.OY) / s
	return p
}

// Sub returns a painter whose unit square is the rectangle (x,y,w,h) of this
// painter's space. Handy for drawing a whole glyph smaller or offset.
func (p *Painter) Sub(x, y, w, h float64) *Painter {
	q := *p
	q.S = p.S * w
	q.OX = p.OX + x*p.S
	q.OY = p.OY + y*p.S
	if p.Mirror {
		q.OX = p.OX + (1-x-w)*p.S
	}
	_ = h
	q.L = (float64(p.x0) - q.OX) / q.S
	q.R = (float64(p.x1) - q.OX) / q.S
	q.T = (float64(p.y0) - q.OY) / q.S
	q.Bt = (float64(p.y1) - q.OY) / q.S
	return &q
}

// unit converts a pixel centre to unit coordinates (undoing mirror).
func (p *Painter) unit(px, py float64) (float64, float64) {
	x := (px - p.OX) / p.S
	if p.Mirror {
		x = 1 - x
	}
	return x, (py - p.OY) / p.S
}

// scan visits every pixel whose centre lies inside the unit bbox and calls
// f with the unit coordinates; f returns colour, alpha and whether to paint.
func (p *Painter) scan(ux0, uy0, ux1, uy1 float64, f func(x, y float64) (RGB, float64, bool)) {
	if ux0 > ux1 {
		ux0, ux1 = ux1, ux0
	}
	if uy0 > uy1 {
		uy0, uy1 = uy1, uy0
	}
	ax, bx := ux0, ux1
	if p.Mirror {
		ax, bx = 1-ux1, 1-ux0
	}
	px0 := int(math.Floor(ax*p.S + p.OX))
	px1 := int(math.Ceil(bx*p.S + p.OX))
	py0 := int(math.Floor(uy0*p.S + p.OY))
	py1 := int(math.Ceil(uy1*p.S + p.OY))
	if px0 < p.x0 {
		px0 = p.x0
	}
	if py0 < p.y0 {
		py0 = p.y0
	}
	if px1 > p.x1 {
		px1 = p.x1
	}
	if py1 > p.y1 {
		py1 = p.y1
	}
	for py := py0; py < py1; py++ {
		for px := px0; px < px1; px++ {
			x, y := p.unit(float64(px)+0.5, float64(py)+0.5)
			if c, a, ok := f(x, y); ok {
				p.B.Blend(px, py, c, a)
			}
		}
	}
}

// Px is the size of one pixel in unit coordinates.
func (p *Painter) Px() float64 { return 1 / p.S }

// Circle fills a disc.
func (p *Painter) Circle(cx, cy, r float64, c RGB) { p.CircleA(cx, cy, r, c, 1) }

// CircleA fills a disc with alpha.
func (p *Painter) CircleA(cx, cy, r float64, c RGB, a float64) {
	if r <= 0 {
		return
	}
	r = math.Max(r, 0.55/p.S)
	rr := r * r
	p.scan(cx-r, cy-r, cx+r, cy+r, func(x, y float64) (RGB, float64, bool) {
		dx, dy := x-cx, y-cy
		return c, a, dx*dx+dy*dy <= rr
	})
}

// Glow paints a soft radial highlight fading to nothing at r.
func (p *Painter) Glow(cx, cy, r float64, c RGB, strength float64) {
	if r <= 0 {
		return
	}
	p.scan(cx-r, cy-r, cx+r, cy+r, func(x, y float64) (RGB, float64, bool) {
		d := math.Hypot(x-cx, y-cy) / r
		if d >= 1 {
			return c, 0, false
		}
		a := (1 - d) * (1 - d) * strength
		return c, a, a > 0.02
	})
}

// Ellipse fills an axis aligned ellipse.
func (p *Painter) Ellipse(cx, cy, rx, ry float64, c RGB) {
	if rx <= 0 || ry <= 0 {
		return
	}
	rx = math.Max(rx, 0.55/p.S)
	ry = math.Max(ry, 0.55/p.S)
	p.scan(cx-rx, cy-ry, cx+rx, cy+ry, func(x, y float64) (RGB, float64, bool) {
		dx, dy := (x-cx)/rx, (y-cy)/ry
		return c, 1, dx*dx+dy*dy <= 1
	})
}

// Rect fills a rectangle.
func (p *Painter) Rect(x, y, w, h float64, c RGB) { p.RectA(x, y, w, h, c, 1) }

// RectA fills a rectangle with alpha.
func (p *Painter) RectA(x, y, w, h float64, c RGB, a float64) {
	p.scan(x, y, x+w, y+h, func(_, _ float64) (RGB, float64, bool) { return c, a, true })
}

// RoundRect fills a rectangle with rounded corners of radius r.
func (p *Painter) RoundRect(x, y, w, h, r float64, c RGB) {
	if r > w/2 {
		r = w / 2
	}
	if r > h/2 {
		r = h / 2
	}
	p.scan(x, y, x+w, y+h, func(px, py float64) (RGB, float64, bool) {
		dx := math.Max(math.Max(x+r-px, px-(x+w-r)), 0)
		dy := math.Max(math.Max(y+r-py, py-(y+h-r)), 0)
		return c, 1, dx*dx+dy*dy <= r*r
	})
}

// Line draws a segment with round caps and thickness th.
func (p *Painter) Line(x0, y0, x1, y1, th float64, c RGB) {
	hw := math.Max(th/2, 0.55/p.S)
	dx, dy := x1-x0, y1-y0
	l2 := dx*dx + dy*dy
	p.scan(math.Min(x0, x1)-hw, math.Min(y0, y1)-hw, math.Max(x0, x1)+hw, math.Max(y0, y1)+hw,
		func(x, y float64) (RGB, float64, bool) {
			t := 0.0
			if l2 > 0 {
				t = Clamp01(((x-x0)*dx + (y-y0)*dy) / l2)
			}
			ex, ey := x0+t*dx-x, y0+t*dy-y
			return c, 1, ex*ex+ey*ey <= hw*hw
		})
}

// Poly fills a polygon given as x0,y0,x1,y1,... (even-odd rule).
func (p *Painter) Poly(c RGB, pts ...float64) {
	n := len(pts) / 2
	if n < 3 {
		return
	}
	minx, miny, maxx, maxy := pts[0], pts[1], pts[0], pts[1]
	for i := 1; i < n; i++ {
		minx = math.Min(minx, pts[2*i])
		maxx = math.Max(maxx, pts[2*i])
		miny = math.Min(miny, pts[2*i+1])
		maxy = math.Max(maxy, pts[2*i+1])
	}
	p.scan(minx, miny, maxx, maxy, func(x, y float64) (RGB, float64, bool) {
		in := false
		j := n - 1
		for i := 0; i < n; i++ {
			xi, yi := pts[2*i], pts[2*i+1]
			xj, yj := pts[2*j], pts[2*j+1]
			if (yi > y) != (yj > y) && x < (xj-xi)*(y-yi)/(yj-yi)+xi {
				in = !in
			}
			j = i
		}
		return c, 1, in
	})
}

// Triangle fills a triangle.
func (p *Painter) Triangle(x0, y0, x1, y1, x2, y2 float64, c RGB) {
	p.Poly(c, x0, y0, x1, y1, x2, y2)
}

// Ring draws a circle outline of thickness th.
func (p *Painter) Ring(cx, cy, r, th float64, c RGB) { p.Arc(cx, cy, r, 0, 2*math.Pi, th, c) }

// Arc draws part of a ring from angle a0 to a1 (radians, clockwise from +x
// because y points down).
func (p *Painter) Arc(cx, cy, r, a0, a1, th float64, c RGB) {
	if r <= 0 {
		return
	}
	th = math.Max(th, 1.1/p.S)
	ro, ri := r+th/2, math.Max(r-th/2, 0)
	span := a1 - a0
	full := span >= 2*math.Pi-1e-9
	p.scan(cx-ro, cy-ro, cx+ro, cy+ro, func(x, y float64) (RGB, float64, bool) {
		dx, dy := x-cx, y-cy
		d := math.Hypot(dx, dy)
		if d > ro || d < ri {
			return c, 0, false
		}
		if full {
			return c, 1, true
		}
		a := math.Atan2(dy, dx) - a0
		for a < 0 {
			a += 2 * math.Pi
		}
		return c, 1, a <= span
	})
}

// Star fills an n pointed star.
func (p *Painter) Star(cx, cy, rOut, rIn float64, n int, rot float64, c RGB) {
	pts := make([]float64, 0, 4*n)
	for i := 0; i < 2*n; i++ {
		r := rOut
		if i%2 == 1 {
			r = rIn
		}
		a := rot + float64(i)*math.Pi/float64(n) - math.Pi/2
		pts = append(pts, cx+r*math.Cos(a), cy+r*math.Sin(a))
	}
	p.Poly(c, pts...)
}

// Heart fills a heart shape roughly size units wide and tall.
func (p *Painter) Heart(cx, cy, size float64, c RGB) {
	k := size * 0.42
	p.scan(cx-size*0.6, cy-size*0.6, cx+size*0.6, cy+size*0.6, func(px, py float64) (RGB, float64, bool) {
		x := (px - cx) / k
		y := -(py-cy)/k + 0.15
		v := x*x + y*y - 1
		return c, 1, v*v*v-x*x*y*y*y <= 0
	})
}

// Shade paints an arbitrary per-pixel function over a unit-space rectangle.
// f receives u,v in 0..1 across the rectangle and returns colour + alpha.
func (p *Painter) Shade(x, y, w, h float64, f func(u, v float64) (RGB, float64)) {
	p.scan(x, y, x+w, y+h, func(px, py float64) (RGB, float64, bool) {
		u, v := (px-x)/w, (py-y)/h
		if u < 0 || v < 0 || u >= 1 || v >= 1 {
			return RGB{}, 0, false
		}
		c, a := f(u, v)
		return c, a, a > 0.01
	})
}

// Dot paints a small square dot of side d.
func (p *Painter) Dot(x, y, d float64, c RGB) { p.Rect(x-d/2, y-d/2, d, d, c) }

// Rot rotates point (x,y) around (cx,cy) by angle a.
func Rot(cx, cy, x, y, a float64) (float64, float64) {
	s, c := math.Sincos(a)
	dx, dy := x-cx, y-cy
	return cx + dx*c - dy*s, cy + dx*s + dy*c
}

// Clip returns a painter restricted to the unit-space rectangle (x,y,w,h).
// Drawing outside the rectangle is discarded.
func (p *Painter) Clip(x, y, w, h float64) *Painter {
	q := *p
	ax, bx := x, x+w
	if p.Mirror {
		ax, bx = 1-(x+w), 1-x
	}
	px0 := int(math.Floor(ax*p.S + p.OX))
	px1 := int(math.Ceil(bx*p.S + p.OX))
	py0 := int(math.Floor(y*p.S + p.OY))
	py1 := int(math.Ceil((y+h)*p.S + p.OY))
	q.x0 = max(p.x0, px0)
	q.x1 = min(p.x1, px1)
	q.y0 = max(p.y0, py0)
	q.y1 = min(p.y1, py1)
	return &q
}

// HeartRing draws a heart outline of thickness th.
func (p *Painter) HeartRing(cx, cy, size, th float64, c RGB) {
	ko := size * 0.42
	ki := (size - th*2) * 0.42
	test := func(px, py, k float64) bool {
		x := (px - cx) / k
		y := -(py-cy)/k + 0.15
		v := x*x + y*y - 1
		return v*v*v-x*x*y*y*y <= 0
	}
	p.scan(cx-size*0.6, cy-size*0.6, cx+size*0.6, cy+size*0.6, func(px, py float64) (RGB, float64, bool) {
		return c, 1, test(px, py, ko) && !test(px, py, ki)
	})
}

// Crescent fills a circle minus a second circle offset by (dx,dy).
func (p *Painter) Crescent(cx, cy, r, dx, dy, r2 float64, c RGB) {
	p.scan(cx-r, cy-r, cx+r, cy+r, func(x, y float64) (RGB, float64, bool) {
		ax, ay := x-cx, y-cy
		bx, by := x-(cx+dx), y-(cy+dy)
		return c, 1, ax*ax+ay*ay <= r*r && bx*bx+by*by > r2*r2
	})
}
