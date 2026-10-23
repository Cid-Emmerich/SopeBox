package vis

import (
	"fmt"
	"math"

	"github.com/Cid-Emmerich/SopeBox/internal/paint"
	"github.com/Cid-Emmerich/SopeBox/internal/voices"
)

// drawIdleOrb shows a calm breathing ring while nobody is speaking yet.
func drawIdleOrb(c *Canvas, f *Frame) {
	d := NewDots(c.W, c.H)
	cx, cy := float64(d.W)/2, float64(d.H)/2
	r := math.Min(cx, cy) * (0.25 + 0.05*math.Sin(f.Time*1.5)) * f.Opts.OrbSize
	col := paint.Mix(f.Theme.Accent, paint.RGB{R: 20, G: 20, B: 28}, 0.5)
	d.Circle(cx, cy, r, col)
	// a faint spectrum so silence still moves when there's music
	spec := f.Analyzer.Spectrum(32, 0, true, f.Opts.Gain)
	for i := 0; i < 64; i++ {
		k := i
		if k >= 32 {
			k = 63 - i
		}
		v := clamp01(spec[k]) * 0.6
		a := 2*math.Pi*float64(i)/64 - math.Pi/2 + f.Time*f.Opts.Rotate
		d.Line(int(cx+math.Cos(a)*r), int(cy+math.Sin(a)*r), int(cx+math.Cos(a)*(r+v*r)), int(cy+math.Sin(a)*(r+v*r)), voiceColour(f, f.Theme.Accent, v, float64(k)/32))
	}
	d.Flush(c, 0, 0)
	msg := "listening for voices…"
	if !f.Playing {
		msg = "press space to play"
	}
	Text(c, (c.W-len(msg))/2, c.H/2+int(r/4)+2, msg, f.Theme.Muted)
}

// ---------------------------------------------------------------------------
// Constellation: orbs as stars, joined by who-answers-whom lines.

type Constellation struct {
	edges map[[2]int]float64
	last  int
}

func (*Constellation) Name() string     { return "constellation" }
func (*Constellation) Describe() string { return "orbs linked by the flow of conversation" }

func (s *Constellation) Draw(c *Canvas, f *Frame) {
	if s.edges == nil {
		s.edges = map[[2]int]float64{}
	}
	orbs := f.Tracker.Visible()
	if len(orbs) == 0 {
		drawIdleOrb(c, f)
		return
	}
	layout(f, orbs)
	idx := orbIndex(f)
	if cur := f.Tracker.Current(); cur != nil && cur.ID != s.last {
		if s.last != 0 {
			k := [2]int{min(s.last, cur.ID), max(s.last, cur.ID)}
			s.edges[k] = math.Min(1, s.edges[k]+0.35)
		}
		s.last = cur.ID
	}
	for k := range s.edges {
		s.edges[k] *= 1 - 0.04*f.DT*10
		if s.edges[k] < 0.02 {
			delete(s.edges, k)
		}
	}
	d := NewDots(c.W, c.H)
	pos := map[int][2]float64{}
	for _, o := range orbs {
		pos[o.ID] = [2]float64{o.X * float64(d.W), o.Y * float64(d.H)}
	}
	for k, w := range s.edges {
		a, ok1 := pos[k[0]]
		b, ok2 := pos[k[1]]
		if !ok1 || !ok2 {
			continue
		}
		col := paint.Mix(f.Theme.Select, paint.Mix(colourOf(f, idx[k[0]]), colourOf(f, idx[k[1]]), 0.5), w)
		// energy pulse travelling along the line
		d.Line(int(a[0]), int(a[1]), int(b[0]), int(b[1]), col)
		t := math.Mod(f.Time*0.8, 1)
		px := a[0] + (b[0]-a[0])*t
		py := a[1] + (b[1]-a[1])*t
		d.Plot(int(px), int(py), paint.White)
	}
	base := math.Min(float64(c.W)/4, float64(c.H)/2) * 0.32 * f.Opts.OrbSize
	for _, o := range orbs {
		col := colourOf(f, idx[o.ID])
		cx, cy := o.X*float64(d.W), o.Y*float64(d.H)
		r := base * (0.5 + 0.6*o.Level) * 4
		// star: rays whose length follows the spectrum
		n := len(o.Spectrum)
		for i := 0; i < n && n > 0; i++ {
			v := clamp01(o.Spectrum[i] * f.Opts.Gain)
			a := 2*math.Pi*float64(i)/float64(n) + f.Time*f.Opts.Rotate
			d.Line(int(cx), int(cy), int(cx+math.Cos(a)*r*(0.3+v)), int(cy+math.Sin(a)*r*(0.3+v)), voiceColour(f, col, v, float64(i)/float64(n)))
		}
		d.Circle(cx, cy, 2+3*o.Level, paint.Mix(col, paint.White, 0.5))
	}
	d.Flush(c, 0, 0)
	for _, o := range orbs {
		label(c, o, o.X*float64(c.W), o.Y*float64(c.H), base*(0.5+0.6*o.Level)+1, colourOf(f, idx[o.ID]), f)
	}
}

func fitName(s string, w int) string {
	r := []rune(s)
	if len(r) <= w {
		return s
	}
	return string(r[:w-1]) + "…"
}

// ---------------------------------------------------------------------------
// WordFlow: orbs plus the words they speak drifting away from them.

type WordFlow struct{}

func (*WordFlow) Name() string     { return "wordflow" }
func (*WordFlow) Describe() string { return "spoken words float away from the voice that said them" }

func (s *WordFlow) Draw(c *Canvas, f *Frame) {
	orbs := f.Tracker.Visible()
	idx := orbIndex(f)
	if len(orbs) == 0 {
		drawIdleOrb(c, f)
	} else {
		layout(f, orbs)
		d := NewDots(c.W, c.H)
		base := math.Min(float64(c.W)/4, float64(c.H)/2) * 0.45 * f.Opts.OrbSize
		for _, o := range orbs {
			drawOrb(c, d, f, o, o.X*float64(c.W), o.Y*float64(c.H), base*(0.7+0.4*o.Level), colourOf(f, idx[o.ID]), idx[o.ID])
		}
		d.Flush(c, 0, 0)
	}
	pos := map[int][2]float64{}
	for _, o := range orbs {
		pos[o.ID] = [2]float64{o.X, o.Y}
	}
	for _, w := range f.Words {
		p, ok := pos[w.Orb]
		if !ok {
			p = [2]float64{0.5, 0.5}
		}
		life := w.Age / 3.5
		if life > 1 {
			continue
		}
		ang := w.Seed * 2 * math.Pi
		dist := 0.08 + life*0.42
		x := p[0] + math.Cos(ang)*dist*1.3
		y := p[1] + math.Sin(ang)*dist - life*0.15
		col := colourOf(f, idx[w.Orb])
		col = paint.Mix(paint.Mix(col, paint.White, 0.4*w.Loud), f.Theme.Select, life*life)
		txt := w.Text
		if w.Loud > 0.75 {
			txt = "✦" + txt
		}
		Text(c, int(x*float64(c.W))-len([]rune(txt))/2, int(y*float64(c.H)), txt, col)
	}
	if f.Opts.Names {
		for _, o := range orbs {
			label(c, o, o.X*float64(c.W), o.Y*float64(c.H), math.Min(float64(c.W)/4, float64(c.H)/2)*0.45*f.Opts.OrbSize, colourOf(f, idx[o.ID]), f)
		}
	}
}

// ---------------------------------------------------------------------------
// TalkTime: a donut of talk share with the orbs sitting on their slice.

type TalkTime struct{}

func (*TalkTime) Name() string     { return "talktime" }
func (*TalkTime) Describe() string { return "who has talked how much, as a living donut" }

func (s *TalkTime) Draw(c *Canvas, f *Frame) {
	orbs := f.Tracker.Visible()
	if len(orbs) == 0 {
		drawIdleOrb(c, f)
		return
	}
	idx := orbIndex(f)
	total := 0.0
	for _, o := range orbs {
		total += o.Talk
	}
	if total <= 0 {
		total = 1
	}
	d := NewDots(c.W, c.H)
	cx, cy := float64(d.W)/2, float64(d.H)/2
	rmax := math.Min(cx, cy) * 0.9 * f.Opts.OrbSize
	a0 := -math.Pi/2 + f.Time*f.Opts.Rotate*0.3
	for _, o := range orbs {
		share := o.Talk / total
		a1 := a0 + share*2*math.Pi
		col := colourOf(f, idx[o.ID])
		inner := rmax * 0.55
		outer := rmax * (0.75 + 0.2*o.Level)
		steps := int((a1-a0)*outer) + 2
		for i := 0; i <= steps; i++ {
			a := a0 + (a1-a0)*float64(i)/float64(steps)
			v := 0.5 + 0.5*o.Level
			if len(o.Spectrum) > 0 {
				v = clamp01(o.Spectrum[i%len(o.Spectrum)]*f.Opts.Gain)*0.7 + 0.3
			}
			ro := inner + (outer-inner)*v
			d.Line(int(cx+math.Cos(a)*inner), int(cy+math.Sin(a)*inner), int(cx+math.Cos(a)*ro), int(cy+math.Sin(a)*ro), voiceColour(f, col, v, share))
		}
		// name at the middle of the slice
		am := (a0 + a1) / 2
		lx := cx + math.Cos(am)*rmax*1.15
		ly := cy + math.Sin(am)*rmax*1.05
		if f.Opts.Names {
			txt := fmt.Sprintf("%s %d%%", fitName(o.Name, 12), int(share*100+0.5))
			Text(c, int(lx/2)-len([]rune(txt))/2, int(ly/4), txt, paint.Mix(col, paint.White, 0.3*o.Level))
		}
		a0 = a1
	}
	d.Flush(c, 0, 0)
	if cur := f.Tracker.Current(); cur != nil {
		Text(c, int(cx/2)-len([]rune(cur.Name))/2, int(cy/4), cur.Name, colourOf(f, idx[cur.ID]))
	}
}

// ---------------------------------------------------------------------------
// Bars: the classic analyser, tinted by whoever is speaking.

type Bars struct{ b bands }

func (*Bars) Name() string     { return "bars" }
func (*Bars) Describe() string { return "classic spectrum bars in the speaker's colour" }

func (s *Bars) Draw(c *Canvas, f *Frame) {
	n := c.W / 2
	if n < 8 {
		n = 8
	}
	vals := s.b.update(f, n)
	if f.Opts.Mirror {
		vals = mirrorVals(vals)
	}
	col := activeColour(f)
	x0 := (c.W - n*2) / 2
	for i, v := range vals {
		h := v * float64(c.H)
		full := int(h)
		frac := h - float64(full)
		u := float64(i) / float64(n)
		for k := 0; k < full && k < c.H; k++ {
			vv := float64(k) / float64(c.H)
			c.Set(x0+i*2, c.H-1-k, '█', voiceColour(f, col, vv, u))
		}
		if full < c.H && frac > 0.1 {
			c.Set(x0+i*2, c.H-1-full, partialBlocks[int(frac*8.999)], voiceColour(f, col, float64(full)/float64(c.H), u))
		}
		if f.Opts.Peaks && s.b.peaks[i] > 0.02 {
			py := c.H - 1 - int(s.b.peaks[i]*float64(c.H-1)+0.5)
			c.Set(x0+i*2, py, '▔', paint.Mix(col, paint.White, 0.4))
		}
	}
	if f.Opts.Names {
		if o := f.Tracker.Current(); o != nil {
			Text(c, 1, 0, "● "+o.Name, col)
		}
	}
}

func mirrorVals(v []float64) []float64 {
	n := len(v)
	out := make([]float64, n)
	half := (n + 1) / 2
	for i := 0; i < n; i++ {
		d := i - n/2
		if d < 0 {
			d = -d - 1 + n%2
		}
		if d >= half {
			d = half - 1
		}
		out[i] = v[d]
	}
	return out
}

var _ = voices.Orb{}
