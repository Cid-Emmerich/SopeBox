package vis

import (
	"math"

	"github.com/Cid-Emmerich/SopeBox/internal/config"
	"github.com/Cid-Emmerich/SopeBox/internal/dsp"
	"github.com/Cid-Emmerich/SopeBox/internal/paint"
	"github.com/Cid-Emmerich/SopeBox/internal/theme"
	"github.com/Cid-Emmerich/SopeBox/internal/voices"
)

// Options are the user-tunable visualizer settings shared by all styles.
type Options struct {
	Gradient  string
	Fill      string
	Peaks     bool
	Mirror    bool
	Smoothing float64
	Gain      float64
	Falloff   float64
	Rotate    float64
	OrbSize   float64
	Physics   bool
	Trails    bool
	Names     bool
}

// FillNames lists orb fill styles for cycling.
var FillNames = []string{"braille", "dots", "rings", "petals", "ascii", "block"}

// OptionsFromConfig copies the vis_* settings.
func OptionsFromConfig(c config.Config) Options {
	return Options{Gradient: c.VisGradient, Fill: c.VisFill, Peaks: c.VisPeaks, Mirror: c.VisMirror,
		Smoothing: c.VisSmoothing, Gain: c.VisGain, Falloff: c.VisFalloff, Rotate: c.VisRotate,
		OrbSize: c.VisOrbSize, Physics: c.VisPhysics, Trails: c.VisTrails, Names: c.Names}
}

// ApplyTo writes the options back into a config.
func (o Options) ApplyTo(c *config.Config) {
	c.VisGradient, c.VisFill, c.VisPeaks, c.VisMirror = o.Gradient, o.Fill, o.Peaks, o.Mirror
	c.VisSmoothing, c.VisGain, c.VisFalloff, c.VisRotate = o.Smoothing, o.Gain, o.Falloff, o.Rotate
	c.VisOrbSize, c.VisPhysics, c.VisTrails, c.Names = o.OrbSize, o.Physics, o.Trails, o.Names
}

// FlyWord is a recently spoken word for styles that animate text.
type FlyWord struct {
	Text string
	Orb  int     // orb ID
	Age  float64 // seconds since spoken
	Loud float64 // 0..1
	Seed float64
}

// Frame is everything a visualizer needs for one draw call.
type Frame struct {
	Analyzer *dsp.Analyzer
	Tracker  *voices.Tracker
	Opts     *Options
	Theme    theme.Theme
	Time     float64
	DT       float64
	Playing  bool
	Words    []FlyWord
	Level    float64 // overall smoothed level 0..1
	Chapter  string
	Collage  *Collage // pictures for the collage style
}

// Visualizer draws one style.
type Visualizer interface {
	Name() string
	Describe() string
	Draw(c *Canvas, f *Frame)
}

// Registry of all visualizers in cycling order.
var Registry = []Visualizer{
	&Orbs{},
	&Constellation{},
	&Halo{},
	&Ribbon{},
	&WordFlow{},
	&Pulse{},
	&TalkTime{},
	&Bars{},
	&CollageStyle{},
}

// Names lists visualizer names.
func Names() []string {
	out := make([]string, len(Registry))
	for i, v := range Registry {
		out[i] = v.Name()
	}
	return out
}

// Index returns the registry index of a name (0 if unknown).
func Index(name string) int {
	for i, v := range Registry {
		if v.Name() == name {
			return i
		}
	}
	return 0
}

// ---------------------------------------------------------------------------
// Shared helpers

// colourOf returns the theme colour for an orb by its display index.
func colourOf(f *Frame, idx int) paint.RGB { return f.Theme.Voice(idx) }

// orbIndex maps orb IDs to stable display indexes (order of appearance).
func orbIndex(f *Frame) map[int]int {
	m := map[int]int{}
	for i, o := range f.Tracker.Sorted() {
		m[o.ID] = i
	}
	return m
}

// layout moves orbs with a little physics so they float together without
// overlapping. Positions are in unit space (0..1 both axes); the active
// speaker drifts toward the centre and everyone else keeps a polite
// distance. Without physics orbs sit on a fixed ring.
func layout(f *Frame, orbs []*voices.Orb) {
	n := len(orbs)
	if n == 0 {
		return
	}
	dt := math.Min(f.DT, 0.05)
	rot := f.Time * f.Opts.Rotate * 0.35
	for i, o := range orbs {
		var tx, ty float64
		if n == 1 {
			tx, ty = 0.5, 0.5
		} else {
			a := 2*math.Pi*float64(i)/float64(n) - math.Pi/2 + rot
			r := 0.30
			if n > 4 {
				r = 0.36
			}
			if o.Active {
				r *= 0.72
			}
			tx, ty = 0.5+math.Cos(a)*r*1.05, 0.5+math.Sin(a)*r
		}
		if !o.Placed {
			o.X, o.Y = tx, ty
			o.VX, o.VY = 0, 0
			o.Placed = true
		}
		if !f.Opts.Physics {
			o.X += (tx - o.X) * 0.15
			o.Y += (ty - o.Y) * 0.15
			continue
		}
		// spring to target
		ax := (tx - o.X) * 6
		ay := (ty - o.Y) * 6
		// gentle organic drift
		ax += 0.15 * math.Sin(f.Time*0.7+float64(o.ID)*1.7)
		ay += 0.15 * math.Cos(f.Time*0.9+float64(o.ID)*2.3)
		// repulsion
		for _, p := range orbs {
			if p == o {
				continue
			}
			dx, dy := o.X-p.X, (o.Y-p.Y)*1.4
			d2 := dx*dx + dy*dy + 0.002
			minD := 0.24 * f.Opts.OrbSize
			if d2 < minD*minD {
				k := (minD*minD - d2) * 40
				ax += dx * k
				ay += dy * k
			}
		}
		o.VX = (o.VX + ax*dt) * 0.90
		o.VY = (o.VY + ay*dt) * 0.90
		o.X += o.VX * dt
		o.Y += o.VY * dt
		o.X = math.Max(0.08, math.Min(0.92, o.X))
		o.Y = math.Max(0.12, math.Min(0.88, o.Y))
	}
}

// radial draws one orb's radial spectrum centred at cx,cy (dot space) with
// outer radius rmax (dots). base is the orb's colour.
func radial(d *Dots, f *Frame, o *voices.Orb, cx, cy, rmax float64, base paint.RGB, idx int) {
	spec := o.Spectrum
	n := len(spec)
	if n == 0 {
		n = 32
		spec = make([]float64, n)
	}
	if rmax < 3 {
		rmax = 3
	}
	r0 := rmax * (0.26 + 0.12*o.Level)
	rot := f.Time*f.Opts.Rotate + float64(o.ID)*0.9
	if idx%2 == 1 {
		rot = -rot
	}
	spawn := o.Spawn
	if spawn < 1 {
		// pop in with an overshoot
		spawn = 1 + 0.5*math.Exp(-4.5*spawn)*math.Sin(9*spawn) - math.Exp(-6*spawn)
	}
	rmax *= spawn
	r0 *= spawn
	fill := f.Opts.Fill
	for i := 0; i < n*2; i++ {
		idx := i
		if i >= n {
			idx = 2*n - 1 - i
		}
		v := clamp01(spec[idx] * f.Opts.Gain)
		ang := 2*math.Pi*float64(i)/float64(n*2) - math.Pi/2 + rot
		r1 := r0 + v*(rmax-r0)
		col := voiceColour(f, base, v, float64(idx)/float64(n))
		x0 := cx + math.Cos(ang)*r0
		y0 := cy + math.Sin(ang)*r0
		x1 := cx + math.Cos(ang)*r1
		y1 := cy + math.Sin(ang)*r1
		switch fill {
		case "dots":
			steps := int(r1-r0) / 2
			for k := 0; k <= steps; k++ {
				t := float64(k) / float64(max(steps, 1))
				d.Plot(int(x0+(x1-x0)*t), int(y0+(y1-y0)*t), col)
			}
		case "rings":
			d.Plot(int(x1), int(y1), col)
			d.Plot(int(x0), int(y0), paint.Mix(col, paint.RGB{}, 0.5))
		default:
			d.Line(int(x0), int(y0), int(x1), int(y1), col)
		}
		if f.Opts.Peaks && idx < len(o.Peaks) && o.Peaks[idx] > 0.04 {
			pr := r0 + clamp01(o.Peaks[idx]*f.Opts.Gain)*(rmax-r0) + 1.5
			d.Plot(int(cx+math.Cos(ang)*pr), int(cy+math.Sin(ang)*pr), paint.Mix(base, paint.White, 0.6))
		}
	}
	if fill == "rings" {
		d.Circle(cx, cy, r0, paint.Mix(base, paint.RGB{}, 0.3))
	}
	// core glow: brighter when speaking
	core := paint.Mix(base, paint.White, 0.25+0.5*o.Level)
	cr := r0 * (0.35 + 0.35*o.Level)
	d.Circle(cx, cy, cr, core)
	if o.Level > 0.5 {
		d.Circle(cx, cy, cr*0.5, paint.White)
	}
}

// petals draws an orb as shaded wedges using cell glyphs (a chunkier look).
func petals(c *Canvas, f *Frame, o *voices.Orb, cx, cy, rmax float64, base paint.RGB, ascii bool) {
	spec := o.Spectrum
	n := len(spec)
	if n == 0 {
		return
	}
	rot := f.Time*f.Opts.Rotate + float64(o.ID)*0.9
	x0, x1 := int(cx-rmax*2)-1, int(cx+rmax*2)+1
	y0, y1 := int(cy-rmax)-1, int(cy+rmax)+1
	shades := []rune{'░', '▒', '▓', '█'}
	if ascii {
		shades = []rune{'.', ':', '*', '#'}
	}
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			dx := (float64(x) - cx + 0.5) / 2
			dy := float64(y) - cy + 0.5
			dist := math.Sqrt(dx*dx + dy*dy)
			if dist > rmax {
				continue
			}
			ang := math.Atan2(dy, dx) - rot + math.Pi/2
			for ang < 0 {
				ang += 2 * math.Pi
			}
			i := int(ang/(2*math.Pi)*float64(n*2)) % (n * 2)
			if i >= n {
				i = 2*n - 1 - i
			}
			v := clamp01(spec[i] * f.Opts.Gain)
			r0 := rmax * (0.22 + 0.1*o.Level)
			r1 := r0 + v*(rmax-r0)
			if dist > r1 {
				continue
			}
			t := 1 - (dist-r0)/(r1-r0+0.001)
			if dist < r0 {
				t = 1
			}
			ch := shades[int(clamp01(t)*3.999)]
			c.Set(x, y, ch, voiceColour(f, base, v*0.5+0.5*t, float64(i)/float64(n)))
		}
	}
}

// label writes a short name under an orb.
func label(c *Canvas, o *voices.Orb, cx, cy, rmax float64, col paint.RGB, f *Frame) {
	if !f.Opts.Names {
		return
	}
	name := o.Name
	if len([]rune(name)) > 14 {
		name = string([]rune(name)[:13]) + "…"
	}
	x := int(cx) - len([]rune(name))/2
	y := int(cy+rmax) + 1
	if y >= c.H {
		y = c.H - 1
	}
	if o.Active {
		Text(c, x, y, name, paint.Mix(col, paint.White, 0.3))
	} else {
		Text(c, x, y, name, paint.Mix(col, paint.RGB{R: 30, G: 30, B: 40}, 0.45))
	}
}

// drawOrb renders one orb in the right fill style at cell centre (cx,cy)
// with radius rmax in cell rows.
func drawOrb(c *Canvas, d *Dots, f *Frame, o *voices.Orb, cx, cy, rmax float64, base paint.RGB, idx int) {
	switch f.Opts.Fill {
	case "petals", "block":
		petals(c, f, o, cx, cy, rmax, base, false)
	case "ascii":
		petals(c, f, o, cx, cy, rmax, base, true)
	default:
		radial(d, f, o, cx*2, cy*4, rmax*4, base, idx)
	}
}

// classic band smoothing for the mono styles
type bands struct {
	vals  []float64
	peaks []float64
}

func (b *bands) update(f *Frame, n int) []float64 {
	if len(b.vals) != n {
		b.vals = make([]float64, n)
		b.peaks = make([]float64, n)
	}
	raw := f.Analyzer.Spectrum(n, 0, true, f.Opts.Gain)
	s := f.Opts.Smoothing
	for i := range b.vals {
		v := 0.0
		if i < len(raw) {
			v = raw[i]
		}
		if v > b.vals[i] {
			b.vals[i] = b.vals[i]*s + v*(1-s)
		} else {
			b.vals[i] = math.Max(v, b.vals[i]-f.Opts.Falloff)
		}
		if b.vals[i] >= b.peaks[i] {
			b.peaks[i] = b.vals[i]
		} else {
			b.peaks[i] = math.Max(0, b.peaks[i]-f.Opts.Falloff*0.6)
		}
	}
	return b.vals
}

var partialBlocks = []rune{' ', '▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

// activeColour is the colour of whoever is speaking (accent when nobody).
func activeColour(f *Frame) paint.RGB {
	if o := f.Tracker.Current(); o != nil {
		return colourOf(f, orbIndex(f)[o.ID])
	}
	return f.Theme.Accent
}
