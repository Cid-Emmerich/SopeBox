// Package paint is G1yph's drawing engine: an RGB pixel buffer, a vector
// painter that draws in a resolution independent unit square, a tiny block
// font, particle systems, and the rasterisers that turn pixels into
// terminal cells in several ASCII / block styles.
package paint

import "math"

// RGB is a true colour.
type RGB struct{ R, G, B uint8 }

// Hex builds a colour from 0xRRGGBB.
func Hex(h uint32) RGB { return RGB{uint8(h >> 16), uint8(h >> 8), uint8(h)} }

// Scale multiplies brightness by k (clamped).
func (c RGB) Scale(k float64) RGB {
	return RGB{clamp8(float64(c.R) * k), clamp8(float64(c.G) * k), clamp8(float64(c.B) * k)}
}

// Mix blends a towards b by t (0..1).
func Mix(a, b RGB, t float64) RGB {
	t = Clamp01(t)
	return RGB{
		clamp8(float64(a.R) + (float64(b.R)-float64(a.R))*t),
		clamp8(float64(a.G) + (float64(b.G)-float64(a.G))*t),
		clamp8(float64(a.B) + (float64(b.B)-float64(a.B))*t),
	}
}

// Mix3 blends across three stops.
func Mix3(a, b, c RGB, t float64) RGB {
	if t < 0.5 {
		return Mix(a, b, t*2)
	}
	return Mix(b, c, (t-0.5)*2)
}

// Luminance returns perceived brightness 0..1.
func Luminance(c RGB) float64 {
	return (0.2126*float64(c.R) + 0.7152*float64(c.G) + 0.0722*float64(c.B)) / 255
}

// HSL converts hue (degrees), saturation and lightness (0..1) to RGB.
func HSL(h, s, l float64) RGB {
	h = math.Mod(h, 360)
	if h < 0 {
		h += 360
	}
	c := (1 - math.Abs(2*l-1)) * s
	x := c * (1 - math.Abs(math.Mod(h/60, 2)-1))
	m := l - c/2
	var r, g, b float64
	switch {
	case h < 60:
		r, g, b = c, x, 0
	case h < 120:
		r, g, b = x, c, 0
	case h < 180:
		r, g, b = 0, c, x
	case h < 240:
		r, g, b = 0, x, c
	case h < 300:
		r, g, b = x, 0, c
	default:
		r, g, b = c, 0, x
	}
	return RGB{clamp8((r + m) * 255), clamp8((g + m) * 255), clamp8((b + m) * 255)}
}

func clamp8(v float64) uint8 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v + 0.5)
}

// Clamp01 clamps to 0..1 (NaN becomes 0).
func Clamp01(x float64) float64 {
	if x < 0 || math.IsNaN(x) {
		return 0
	}
	if x > 1 {
		return 1
	}
	return x
}

// Handy named colours used across the glyph library.
var (
	White     = Hex(0xFFFFFF)
	Black     = Hex(0x000000)
	Grey      = Hex(0x9AA0A6)
	DarkGrey  = Hex(0x4A4E55)
	LightGrey = Hex(0xD8DCE0)
	Red       = Hex(0xE53935)
	Orange    = Hex(0xFF9800)
	Amber     = Hex(0xFFC107)
	Yellow    = Hex(0xFFEB3B)
	Green     = Hex(0x43A047)
	Lime      = Hex(0x8BC34A)
	Teal      = Hex(0x26A69A)
	Blue      = Hex(0x1E88E5)
	SkyBlue   = Hex(0x4FC3F7)
	Indigo    = Hex(0x3F51B5)
	Purple    = Hex(0x8E24AA)
	Pink      = Hex(0xEC407A)
	Brown     = Hex(0x6D4C41)
	Tan       = Hex(0xD7A86E)
	Skin      = Hex(0xF5C99B)
	Cream     = Hex(0xFFF3E0)
	Gold      = Hex(0xFFD54F)
	Navy      = Hex(0x1A237E)
	Steel     = Hex(0x78909C)
)
