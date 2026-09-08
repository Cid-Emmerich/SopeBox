package art

import (
	"image"
	"math"
	"sort"

	"github.com/Cid-Emmerich/SopeBox/internal/paint"
	"github.com/Cid-Emmerich/SopeBox/internal/theme"
)

// Palette is a set of colours pulled from artwork, ready to drive a theme.
type Palette struct {
	Accent    paint.RGB
	Secondary paint.RGB
	Dim       paint.RGB
	Dark      paint.RGB
	Light     paint.RGB
	Dominant  []paint.RGB
}

type bucket struct {
	r, g, b float64
	n       int
}

// ExtractPalette quantises the image into colour buckets, ranks them by
// frequency and vividness, and derives a small usable palette.
func ExtractPalette(img image.Image) Palette {
	small := scale(img, 64, 64)
	buckets := map[uint32]*bucket{}
	for y := 0; y < 64; y++ {
		for x := 0; x < 64; x++ {
			c := rgbAt(small, x, y)
			key := uint32(c.R>>4)<<8 | uint32(c.G>>4)<<4 | uint32(c.B>>4)
			bk, ok := buckets[key]
			if !ok {
				bk = &bucket{}
				buckets[key] = bk
			}
			bk.r += float64(c.R)
			bk.g += float64(c.G)
			bk.b += float64(c.B)
			bk.n++
		}
	}
	type entry struct {
		c        paint.RGB
		n        int
		sat, lum float64
	}
	var entries []entry
	for _, bk := range buckets {
		c := paint.RGB{R: uint8(bk.r / float64(bk.n)), G: uint8(bk.g / float64(bk.n)), B: uint8(bk.b / float64(bk.n))}
		_, s, l := theme.ToHSL(c)
		entries = append(entries, entry{c, bk.n, s, l})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].n > entries[j].n })
	if len(entries) > 12 {
		entries = entries[:12]
	}
	p := Palette{}
	for _, e := range entries {
		p.Dominant = append(p.Dominant, e.c)
	}
	if len(entries) == 0 {
		return p
	}
	score := func(e entry) float64 {
		midness := 1 - math.Abs(e.lum-0.5)*2
		return e.sat*(0.4+0.6*midness)*math.Sqrt(float64(e.n)) + 0.001*float64(e.n)
	}
	bestI := 0
	for i, e := range entries {
		if score(e) > score(entries[bestI]) {
			bestI = i
		}
	}
	p.Accent = ensureVisible(entries[bestI].c)
	h0, _, _ := theme.ToHSL(entries[bestI].c)
	secI := -1
	for i, e := range entries {
		if i == bestI {
			continue
		}
		h, _, _ := theme.ToHSL(e.c)
		if hueDist(h, h0) < 25 && e.sat > 0.15 {
			continue
		}
		if secI == -1 || score(e) > score(entries[secI]) {
			secI = i
		}
	}
	if secI == -1 {
		p.Secondary = shiftHue(p.Accent, 40)
	} else {
		p.Secondary = ensureVisible(entries[secI].c)
	}
	dark, light := entries[0], entries[0]
	for _, e := range entries {
		if e.lum < dark.lum {
			dark = e
		}
		if e.lum > light.lum {
			light = e
		}
	}
	p.Dark = dark.c
	p.Light = light.c
	p.Dim = paint.Mix(p.Accent, paint.RGB{R: 128, G: 128, B: 128}, 0.55)
	return p
}

func ensureVisible(c paint.RGB) paint.RGB {
	h, s, l := theme.ToHSL(c)
	if l < 0.35 {
		l = 0.45
	}
	if l > 0.85 {
		l = 0.75
	}
	if s < 0.25 {
		s = 0.35
	}
	return paint.HSL(h, s, l)
}

func shiftHue(c paint.RGB, deg float64) paint.RGB {
	h, s, l := theme.ToHSL(c)
	return paint.HSL(math.Mod(h+deg+360, 360), s, l)
}

func hueDist(a, b float64) float64 {
	d := math.Abs(a - b)
	if d > 180 {
		d = 360 - d
	}
	return d
}

// MatchTheme builds a "match" theme from artwork colours. base supplies the
// text/selection colours so readability never depends on the picture.
func MatchTheme(p Palette, base theme.Theme) theme.Theme {
	t := base
	t.Name = "match"
	if len(p.Dominant) == 0 {
		return t
	}
	t.Accent = p.Accent
	t.Secondary = p.Secondary
	t.Tertiary = paint.Mix(p.Accent, p.Light, 0.5)
	t.Muted = p.Dim
	t.Select = paint.Mix(p.Dark, paint.RGB{R: 40, G: 40, B: 48}, 0.6)
	if paint.Luminance(t.Muted) < 0.25 {
		t.Muted = paint.Mix(t.Muted, paint.RGB{R: 160, G: 160, B: 170}, 0.5)
	}
	// Voices: spread hues around the accent so every orb reads distinctly.
	h, s, l := theme.ToHSL(p.Accent)
	if s < 0.3 {
		s = 0.45
	}
	if l < 0.45 {
		l = 0.55
	}
	t.Voices = make([]paint.RGB, 8)
	for i := range t.Voices {
		t.Voices[i] = paint.HSL(math.Mod(h+float64(i)*47, 360), s, l)
	}
	t.Voices[0] = p.Accent
	t.Voices[1] = p.Secondary
	return t
}
