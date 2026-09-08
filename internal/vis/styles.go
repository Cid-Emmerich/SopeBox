package vis

import (
	"fmt"
	"math"

	"github.com/Cid-Emmerich/SopeBox/internal/paint"
	"github.com/Cid-Emmerich/SopeBox/internal/voices"
)

// ---------------------------------------------------------------------------
// Orbs: one radial spectrum per voice, floating together.

type Orbs struct{}

func (*Orbs) Name() string     { return "orbs" }
func (*Orbs) Describe() string { return "a radial spectrum orb for every voice" }

func (s *Orbs) Draw(c *Canvas, f *Frame) {
	orbs := f.Tracker.Visible()
	if len(orbs) == 0 {
		drawIdleOrb(c, f)
		return
	}
	layout(f, orbs)
	d := NewDots(c.W, c.H)
	idx := orbIndex(f)
	n := len(orbs)
	base := math.Min(float64(c.W)/4, float64(c.H)/2) * 0.9
	if n > 1 {
		base *= 0.62
	}
	if n > 4 {
		base *= 0.85
	}
	base *= f.Opts.OrbSize
	for _, o := range orbs {
		r := base * (0.7 + 0.3*o.Level)
		if o.Active {
			r = base * (0.85 + 0.25*o.Level)
		}
		cx, cy := o.X*float64(c.W), o.Y*float64(c.H)
		col := colourOf(f, idx[o.ID])
		drawOrb(c, d, f, o, cx, cy, r, col, idx[o.ID])
	}
	d.Flush(c, 0, 0)
	for _, o := range orbs {
		cx, cy := o.X*float64(c.W), o.Y*float64(c.H)
		r := base * (0.7 + 0.3*o.Level)
		label(c, o, cx, cy, r, colourOf(f, idx[o.ID]), f)
	}
}

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

// ---------------------------------------------------------------------------
// Halo: nested rings, one per voice, around a shared centre.

type Halo struct{}

func (*Halo) Name() string     { return "halo" }
func (*Halo) Describe() string { return "nested rings, one per voice" }

func (s *Halo) Draw(c *Canvas, f *Frame) {
	orbs := f.Tracker.Visible()
	if len(orbs) == 0 {
		drawIdleOrb(c, f)
		return
	}
	idx := orbIndex(f)
	d := NewDots(c.W, c.H)
	cx, cy := float64(d.W)/2, float64(d.H)/2
	rmax := math.Min(cx, cy) * 0.95 * f.Opts.OrbSize
	n := len(orbs)
	for k, o := range orbs {
		col := colourOf(f, idx[o.ID])
		inner := rmax * (0.18 + 0.75*float64(k)/float64(n))
		outer := rmax * (0.18 + 0.75*float64(k+1)/float64(n))
		spec := o.Spectrum
		m := len(spec)
		if m == 0 {
			d.Circle(cx, cy, inner, paint.Mix(col, paint.RGB{}, 0.6))
			continue
		}
		rot := f.Time * f.Opts.Rotate * float64(1+k%2*-2)
		for i := 0; i < m*2; i++ {
			j := i
			if i >= m {
				j = 2*m - 1 - i
			}
			v := clamp01(spec[j] * f.Opts.Gain)
			a := 2*math.Pi*float64(i)/float64(m*2) + rot
			r1 := inner + v*(outer-inner)*(0.4+0.6*o.Level)
			d.Line(int(cx+math.Cos(a)*inner), int(cy+math.Sin(a)*inner), int(cx+math.Cos(a)*r1), int(cy+math.Sin(a)*r1), voiceColour(f, col, v, float64(j)/float64(m)))
		}
		if o.Active {
			d.Circle(cx, cy, inner-1, paint.Mix(col, paint.White, 0.4))
		}
	}
	d.Flush(c, 0, 0)
	if f.Opts.Names {
		y := c.H - 1
		x := 1
		for _, o := range orbs {
			col := colourOf(f, idx[o.ID])
			mark := "○ "
			if o.Active {
				mark = "● "
			}
			s := mark + o.Name + "  "
			Text(c, x, y, s, paint.Mix(col, paint.RGB{R: 30, G: 30, B: 40}, 0.4*(1-o.Level)))
			x += len([]rune(s))
		}
	}
}

// ---------------------------------------------------------------------------
// Ribbon: a scrolling who-spoke-when timeline under a row of orbs.

type Ribbon struct {
	hist []int // orb ID per column tick, newest last
	acc  float64
}

func (*Ribbon) Name() string     { return "ribbon" }
func (*Ribbon) Describe() string { return "orbs above a scrolling conversation timeline" }

func (s *Ribbon) Draw(c *Canvas, f *Frame) {
	orbs := f.Tracker.Visible()
	idx := orbIndex(f)
	s.acc += f.DT
	for s.acc >= 0.25 {
		s.acc -= 0.25
		id := 0
		if o := f.Tracker.Current(); o != nil {
			id = o.ID
		}
		s.hist = append(s.hist, id)
	}
	if len(s.hist) > 600 {
		s.hist = s.hist[len(s.hist)-600:]
	}
	ribH := max(3, c.H/4)
	top := c.H - ribH
	// orbs row
	if len(orbs) == 0 {
		sub := NewCanvas(c.W, top)
		drawIdleOrb(sub, f)
		copySub(c, sub, 0, 0)
	} else {
		d := NewDots(c.W, top)
		n := len(orbs)
		slot := float64(c.W) / float64(n)
		r := math.Min(slot/4, float64(top)/2) * 0.8 * f.Opts.OrbSize
		for k, o := range orbs {
			cx := slot * (float64(k) + 0.5)
			cy := float64(top)/2 - 1
			col := colourOf(f, idx[o.ID])
			drawOrb(c, d, f, o, cx, cy, r*(0.75+0.25*o.Level), col, idx[o.ID])
		}
		d.Flush(c, 0, 0)
		for k, o := range orbs {
			label(c, o, slot*(float64(k)+0.5), float64(top)/2-1, r, colourOf(f, idx[o.ID]), f)
		}
	}
	// timeline
	for x := 0; x < c.W; x++ {
		Text(c, x, top, "─", f.Theme.Select)
	}
	rows := ribH - 1
	if rows < 1 {
		return
	}
	lanes := map[int]int{}
	for _, o := range orbs {
		lanes[o.ID] = idx[o.ID] % rows
	}
	start := len(s.hist) - c.W
	if start < 0 {
		start = 0
	}
	for i := start; i < len(s.hist); i++ {
		id := s.hist[i]
		if id == 0 {
			continue
		}
		x := c.W - (len(s.hist) - i)
		lane, ok := lanes[id]
		if !ok {
			continue
		}
		age := float64(len(s.hist)-i) / float64(c.W)
		col := paint.Mix(colourOf(f, idx[id]), f.Theme.Select, age*0.6)
		c.Set(x, top+1+lane, '█', col)
	}
	if f.Opts.Names {
		for _, o := range orbs {
			Text(c, 0, top+1+lanes[o.ID], fitName(o.Name, 12), paint.Mix(colourOf(f, idx[o.ID]), paint.White, 0.2))
		}
	}
}

func fitName(s string, w int) string {
	r := []rune(s)
	if len(r) <= w {
		return s
	}
	return string(r[:w-1]) + "…"
}

func copySub(dst, src *Canvas, ox, oy int) {
	for y := 0; y < src.H; y++ {
		for x := 0; x < src.W; x++ {
			cell := src.Cells[y*src.W+x]
			if cell.Ch != 0 {
				if cell.HasBg {
					dst.SetBg(ox+x, oy+y, cell.Ch, cell.Fg, cell.Bg)
				} else {
					dst.Set(ox+x, oy+y, cell.Ch, cell.Fg)
				}
			}
		}
	}
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
// Pulse: one big breathing orb that takes the colour of whoever speaks.

type Pulse struct {
	col   paint.RGB
	level float64
	b     bands
}

func (*Pulse) Name() string { return "pulse" }
func (*Pulse) Describe() string {
	return "a single orb that breathes and changes colour with the speaker"
}

func (s *Pulse) Draw(c *Canvas, f *Frame) {
	target := activeColour(f)
	if s.col == (paint.RGB{}) {
		s.col = target
	}
	s.col = paint.Mix(s.col, target, 0.08)
	lv := f.Level
	sm := f.Opts.Smoothing
	if lv > s.level {
		s.level = s.level*sm + lv*(1-sm)
	} else {
		s.level = math.Max(lv, s.level-f.Opts.Falloff)
	}
	vals := s.b.update(f, 32)
	cx, cy := float64(c.W)/2, float64(c.H)/2
	rmax := math.Min(cx/2, cy) * 0.95 * f.Opts.OrbSize
	r := rmax * (0.3 + 0.55*s.level)
	shades := []rune{'░', '▒', '▓', '█'}
	for y := 0; y < c.H; y++ {
		for x := 0; x < c.W; x++ {
			dx := (float64(x) - cx + 0.5) / 2
			dy := float64(y) - cy + 0.5
			dist := math.Sqrt(dx*dx + dy*dy)
			if dist <= r {
				v := 1 - dist/r
				c.Set(x, y, shades[int(clamp01(v)*3.999)], voiceColour(f, s.col, v, s.level))
			} else {
				ang := math.Atan2(dy, dx) + math.Pi + f.Time*f.Opts.Rotate
				i := int(math.Mod(ang/(2*math.Pi), 1) * 32)
				if i >= 32 || i < 0 {
					i = 0
				}
				hr := r + vals[i]*(rmax*1.15-r) + 1
				if dist <= hr {
					v := 1 - (dist-r)/(hr-r+0.001)
					c.Set(x, y, []rune{'·', '∙', '•', '●'}[int(clamp01(v)*3.999)], voiceColour(f, s.col, v*0.6, float64(i)/32))
				}
			}
		}
	}
	if f.Opts.Names {
		if o := f.Tracker.Current(); o != nil {
			Text(c, int(cx)-len([]rune(o.Name))/2, int(cy), o.Name, paint.White)
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
