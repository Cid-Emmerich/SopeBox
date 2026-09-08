package paint

import "math"

// ColourNames lists the colour modes in cycling order. "emoji" keeps the
// glyph's own colours; the rest recolour the picture with a gradient (the
// same family of gradients wvfrm uses for its visualizers).
var ColourNames = []string{"emoji", "theme", "horizontal", "rainbow", "spectrum", "fire", "ice", "neon", "heat", "mono", "pastel", "matrix"}

// Palette is the small set of theme colours the gradients draw from.
type Palette struct{ Accent, Secondary, Tertiary RGB }

// Gradient returns the colour for a point: v is height fraction (0 bottom,
// 1 top), u is horizontal fraction (0 left, 1 right), t is time in seconds.
func Gradient(name string, v, u, t float64, pal Palette) RGB {
	v = Clamp01(v)
	u = Clamp01(u)
	switch name {
	case "horizontal":
		return Mix3(pal.Accent, pal.Secondary, pal.Tertiary, u)
	case "rainbow":
		return HSL(math.Mod(u*300+t*20, 360), 0.85, 0.58)
	case "spectrum":
		return HSL(math.Mod(240-v*240+t*10, 360), 0.9, 0.55)
	case "fire":
		return Mix3(RGB{180, 20, 0}, RGB{255, 160, 0}, RGB{255, 250, 200}, v)
	case "ice":
		return Mix3(RGB{20, 60, 200}, RGB{60, 200, 255}, RGB{230, 250, 255}, v)
	case "neon":
		return Mix3(RGB{255, 0, 200}, RGB{140, 60, 255}, RGB{0, 240, 255}, v)
	case "heat":
		return Mix3(RGB{40, 200, 80}, RGB{240, 220, 40}, RGB{255, 50, 50}, v)
	case "mono":
		return pal.Accent
	case "pastel":
		return HSL(math.Mod(u*360+t*15, 360), 0.6, 0.75)
	case "matrix":
		return Mix3(RGB{0, 90, 20}, RGB{0, 220, 60}, RGB{200, 255, 210}, v)
	default: // theme
		return Mix3(pal.Accent, pal.Secondary, pal.Tertiary, v)
	}
}
