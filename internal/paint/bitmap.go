package paint

// Pix is one pixel: a colour and coverage. A is 0 for untouched pixels.
// Translucent paint on an empty pixel darkens the colour (we assume a dark
// terminal) rather than storing a separate background.
type Pix struct {
	C RGB
	A uint8
}

// Bitmap is a square-pixel image the painter draws into.
type Bitmap struct {
	W, H int
	P    []Pix
}

// NewBitmap allocates a blank bitmap.
func NewBitmap(w, h int) *Bitmap {
	if w < 0 {
		w = 0
	}
	if h < 0 {
		h = 0
	}
	return &Bitmap{W: w, H: h, P: make([]Pix, w*h)}
}

// Clear blanks every pixel.
func (b *Bitmap) Clear() {
	for i := range b.P {
		b.P[i] = Pix{}
	}
}

// Get returns a pixel (empty when out of range).
func (b *Bitmap) Get(x, y int) Pix {
	if x < 0 || y < 0 || x >= b.W || y >= b.H {
		return Pix{}
	}
	return b.P[y*b.W+x]
}

// Set paints an opaque pixel.
func (b *Bitmap) Set(x, y int, c RGB) {
	if x < 0 || y < 0 || x >= b.W || y >= b.H {
		return
	}
	b.P[y*b.W+x] = Pix{C: c, A: 255}
}

// Blend paints a pixel with alpha a (0..1).
func (b *Bitmap) Blend(x, y int, c RGB, a float64) {
	if x < 0 || y < 0 || x >= b.W || y >= b.H {
		return
	}
	if a >= 1 {
		b.P[y*b.W+x] = Pix{C: c, A: 255}
		return
	}
	if a <= 0 {
		return
	}
	p := &b.P[y*b.W+x]
	if p.A == 0 {
		p.C = c.Scale(a)
		p.A = clamp8(a * 255)
		return
	}
	p.C = Mix(p.C, c, a)
	if na := clamp8(a * 255); na > p.A {
		p.A = na
	}
}
