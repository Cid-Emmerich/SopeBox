// Package theme defines colour themes, including the "match" theme that is
// generated from the current podcast's artwork.
package theme

import (
	"math"
	"strings"

	"github.com/Cid-Emmerich/SopeBox/internal/paint"
)

// Theme is a small palette used everywhere in the UI. The terminal
// background is left untouched so SopeBox blends into whatever the terminal
// already uses.
type Theme struct {
	Name      string
	Text      paint.RGB // primary text
	Muted     paint.RGB // secondary text, borders
	Accent    paint.RGB // highlights, progress bar, active items
	Secondary paint.RGB // gradient partner for the accent
	Tertiary  paint.RGB // third gradient stop
	Select    paint.RGB // selection background in lists
	Warn      paint.RGB
	// Voices is the ordered set of orb colours. When more voices appear
	// than colours, hues are rotated from the last one.
	Voices []paint.RGB
}

func rgb(hex uint32) paint.RGB { return paint.Hex(hex) }

func voices(hexes ...uint32) []paint.RGB {
	out := make([]paint.RGB, len(hexes))
	for i, h := range hexes {
		out[i] = rgb(h)
	}
	return out
}

// Builtin themes, in cycling order. "match" is appended dynamically.
var Builtin = []Theme{
	{Name: "sopebox", Text: rgb(0xF2EEE6), Muted: rgb(0x7C7690), Accent: rgb(0xFF8A5B), Secondary: rgb(0xFFC46B), Tertiary: rgb(0x7FD1B9), Select: rgb(0x2E2A3F), Warn: rgb(0xFF5C7A),
		Voices: voices(0xFF8A5B, 0x7FD1B9, 0xC792EA, 0xFFD166, 0x62B6FF, 0xF78FB3, 0x9EE493, 0xFF6F91)},
	{Name: "wvfrm", Text: rgb(0xE6E6F0), Muted: rgb(0x6C6F85), Accent: rgb(0x7AA2F7), Secondary: rgb(0xBB9AF7), Tertiary: rgb(0x7DCFFF), Select: rgb(0x2A2E45), Warn: rgb(0xF7768E),
		Voices: voices(0x7AA2F7, 0xBB9AF7, 0x7DCFFF, 0x9ECE6A, 0xE0AF68, 0xF7768E, 0x2AC3DE, 0xFF9E64)},
	{Name: "nord", Text: rgb(0xECEFF4), Muted: rgb(0x4C566A), Accent: rgb(0x88C0D0), Secondary: rgb(0x81A1C1), Tertiary: rgb(0xB48EAD), Select: rgb(0x3B4252), Warn: rgb(0xBF616A),
		Voices: voices(0x88C0D0, 0xB48EAD, 0xA3BE8C, 0xEBCB8B, 0x81A1C1, 0xD08770, 0x5E81AC, 0xBF616A)},
	{Name: "dracula", Text: rgb(0xF8F8F2), Muted: rgb(0x6272A4), Accent: rgb(0xBD93F9), Secondary: rgb(0xFF79C6), Tertiary: rgb(0x8BE9FD), Select: rgb(0x44475A), Warn: rgb(0xFF5555),
		Voices: voices(0xBD93F9, 0xFF79C6, 0x8BE9FD, 0x50FA7B, 0xF1FA8C, 0xFFB86C, 0xFF5555, 0x6272A4)},
	{Name: "gruvbox", Text: rgb(0xEBDBB2), Muted: rgb(0x928374), Accent: rgb(0xFABD2F), Secondary: rgb(0xFE8019), Tertiary: rgb(0xB8BB26), Select: rgb(0x3C3836), Warn: rgb(0xFB4934),
		Voices: voices(0xFABD2F, 0xB8BB26, 0x83A598, 0xD3869B, 0xFE8019, 0x8EC07C, 0xFB4934, 0xEBDBB2)},
	{Name: "catppuccin", Text: rgb(0xCDD6F4), Muted: rgb(0x6C7086), Accent: rgb(0xCBA6F7), Secondary: rgb(0xF5C2E7), Tertiary: rgb(0x89DCEB), Select: rgb(0x313244), Warn: rgb(0xF38BA8),
		Voices: voices(0xCBA6F7, 0x89DCEB, 0xA6E3A1, 0xF9E2AF, 0xF5C2E7, 0xFAB387, 0x89B4FA, 0xF38BA8)},
	{Name: "solarized", Text: rgb(0xEEE8D5), Muted: rgb(0x586E75), Accent: rgb(0x2AA198), Secondary: rgb(0x268BD2), Tertiary: rgb(0xB58900), Select: rgb(0x073642), Warn: rgb(0xDC322F),
		Voices: voices(0x2AA198, 0xB58900, 0x268BD2, 0xD33682, 0x859900, 0xCB4B16, 0x6C71C4, 0xDC322F)},
	{Name: "synthwave", Text: rgb(0xF4EEFF), Muted: rgb(0x7A5C99), Accent: rgb(0xFF2ED2), Secondary: rgb(0x2DE2E6), Tertiary: rgb(0xF6F740), Select: rgb(0x3A1F5D), Warn: rgb(0xFF6E6E),
		Voices: voices(0xFF2ED2, 0x2DE2E6, 0xF6F740, 0xFF6E6E, 0x9D4EDD, 0x00FF9F, 0xFFA630, 0x72DDF7)},
	{Name: "sunset", Text: rgb(0xFFF1E6), Muted: rgb(0x8C6A5D), Accent: rgb(0xFF7B54), Secondary: rgb(0xFFB26B), Tertiary: rgb(0xFFD56F), Select: rgb(0x4A2B2B), Warn: rgb(0xFF4C4C),
		Voices: voices(0xFF7B54, 0xFFD56F, 0xC97B84, 0xFFB26B, 0x8E6C88, 0xF4A261, 0xE76F51, 0xFFE8D6)},
	{Name: "forest", Text: rgb(0xE3EBD9), Muted: rgb(0x6B7F62), Accent: rgb(0x8FD694), Secondary: rgb(0x4FA37A), Tertiary: rgb(0xC9E4A6), Select: rgb(0x2C3A2A), Warn: rgb(0xE07A5F),
		Voices: voices(0x8FD694, 0xC9E4A6, 0xE9C46A, 0x4FA37A, 0xE07A5F, 0xA7C957, 0xF2CC8F, 0x81B29A)},
	{Name: "ocean", Text: rgb(0xE0F2FE), Muted: rgb(0x557A95), Accent: rgb(0x38BDF8), Secondary: rgb(0x818CF8), Tertiary: rgb(0x34D399), Select: rgb(0x1E3A5F), Warn: rgb(0xFB7185),
		Voices: voices(0x38BDF8, 0x34D399, 0x818CF8, 0xFBBF24, 0xF472B6, 0x2DD4BF, 0xA78BFA, 0xFB7185)},
	{Name: "mono", Text: rgb(0xF0F0F0), Muted: rgb(0x707070), Accent: rgb(0xFFFFFF), Secondary: rgb(0xBDBDBD), Tertiary: rgb(0x8A8A8A), Select: rgb(0x333333), Warn: rgb(0xFFFFFF),
		Voices: voices(0xFFFFFF, 0xBDBDBD, 0x8A8A8A, 0xE0E0E0, 0xA0A0A0, 0xD0D0D0, 0x909090, 0xF5F5F5)},
	{Name: "amber", Text: rgb(0xFFE7B3), Muted: rgb(0x8A6A2E), Accent: rgb(0xFFB000), Secondary: rgb(0xFF8C00), Tertiary: rgb(0xFFD866), Select: rgb(0x3A2A0A), Warn: rgb(0xFF5A36),
		Voices: voices(0xFFB000, 0xFFD866, 0xFF8C00, 0xFFE7B3, 0xE07B00, 0xFFC94D, 0xCC7A00, 0xFF5A36)},
	{Name: "radio", Text: rgb(0xEDE4D3), Muted: rgb(0x7D6B58), Accent: rgb(0xE9C46A), Secondary: rgb(0xF4A261), Tertiary: rgb(0x2A9D8F), Select: rgb(0x3B2F2F), Warn: rgb(0xE76F51),
		Voices: voices(0xE9C46A, 0x2A9D8F, 0xF4A261, 0xE76F51, 0x8AB17D, 0xBABB74, 0xEFB366, 0x287271)},
}

// Names lists every theme name including "match".
func Names() []string {
	out := make([]string, 0, len(Builtin)+1)
	for _, t := range Builtin {
		out = append(out, t.Name)
	}
	return append(out, "match")
}

// Get returns a builtin theme by name (default theme when unknown).
func Get(name string) Theme {
	name = strings.ToLower(name)
	for _, t := range Builtin {
		if t.Name == name {
			return t
		}
	}
	return Builtin[0]
}

// Voice returns the colour for voice index i, rotating hues past the end
// of the palette so every orb stays distinguishable.
func (t Theme) Voice(i int) paint.RGB {
	if len(t.Voices) == 0 {
		return t.Accent
	}
	if i < len(t.Voices) {
		return t.Voices[i]
	}
	base := t.Voices[i%len(t.Voices)]
	h, s, l := ToHSL(base)
	shift := float64(i/len(t.Voices)) * 37
	return paint.HSL(math.Mod(h+shift, 360), s, l)
}

// Palette exposes the gradient stops for the paint package.
func (t Theme) Palette() paint.Palette {
	return paint.Palette{Accent: t.Accent, Secondary: t.Secondary, Tertiary: t.Tertiary}
}

// ToHSL converts a colour to hue (degrees), saturation and lightness.
func ToHSL(c paint.RGB) (h, s, l float64) {
	r, g, b := float64(c.R)/255, float64(c.G)/255, float64(c.B)/255
	mx := math.Max(r, math.Max(g, b))
	mn := math.Min(r, math.Min(g, b))
	l = (mx + mn) / 2
	if mx == mn {
		return 0, 0, l
	}
	d := mx - mn
	if l > 0.5 {
		s = d / (2 - mx - mn)
	} else {
		s = d / (mx + mn)
	}
	switch mx {
	case r:
		h = (g - b) / d
		if g < b {
			h += 6
		}
	case g:
		h = (b-r)/d + 2
	default:
		h = (r-g)/d + 4
	}
	return h * 60, s, l
}
