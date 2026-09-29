package art

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/png"
	"os"
	"strings"

	"golang.org/x/image/draw"

	"github.com/Cid-Emmerich/SopeBox/internal/paint"
)

// Charsets available for ASCII rendering (dark -> light).
var Charsets = paint.Charsets

// CharsetNames lists charsets in a stable order for cycling.
var CharsetNames = paint.CharsetNames

// Fit returns the cell size (w, h) that fits an image into the box
// maxW x maxH while keeping the picture's aspect ratio, assuming terminal
// cells are twice as tall as they are wide.
func Fit(img image.Image, maxW, maxH int) (int, int) {
	if img == nil || maxW <= 0 || maxH <= 0 {
		return 0, 0
	}
	b := img.Bounds()
	iw, ih := float64(b.Dx()), float64(b.Dy())
	w := int(float64(maxH) * 2 * iw / ih)
	h := maxH
	if w > maxW {
		w = maxW
		h = int(float64(maxW) / 2 * ih / iw)
	}
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	return w, h
}

func scale(img image.Image, w, h int) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.CatmullRom.Scale(dst, dst.Bounds(), img, img.Bounds(), draw.Src, nil)
	return dst
}

func rgbAt(img *image.RGBA, x, y int) paint.RGB {
	c := img.RGBAAt(x, y)
	if c.A == 0 {
		return paint.RGB{}
	}
	return paint.RGB{R: c.R, G: c.G, B: c.B}
}

// Blocks renders the image as w x h cells using the upper-half-block glyph,
// so every cell shows two vertically stacked pixels in true colour.
func Blocks(img image.Image, w, h int) [][]paint.Cell {
	if w <= 0 || h <= 0 || img == nil {
		return nil
	}
	src := scale(img, w, h*2)
	out := make([][]paint.Cell, h)
	for y := 0; y < h; y++ {
		row := make([]paint.Cell, w)
		for x := 0; x < w; x++ {
			row[x] = paint.Cell{Ch: '▀', Fg: rgbAt(src, x, y*2), Bg: rgbAt(src, x, y*2+1), HasBg: true}
		}
		out[y] = row
	}
	return out
}

// ASCII renders the image as w x h characters chosen by brightness.
func ASCII(img image.Image, w, h int, charset string, colored bool) [][]paint.Cell {
	if w <= 0 || h <= 0 || img == nil {
		return nil
	}
	chars := []rune(Charsets[charset])
	if len(chars) == 0 {
		chars = []rune(Charsets["standard"])
	}
	src := scale(img, w, h)
	out := make([][]paint.Cell, h)
	for y := 0; y < h; y++ {
		row := make([]paint.Cell, w)
		for x := 0; x < w; x++ {
			c := rgbAt(src, x, y)
			lum := paint.Luminance(c)
			idx := int(lum * float64(len(chars)-1))
			cell := paint.Cell{Ch: chars[idx]}
			if colored {
				cell.Fg = c
				if lum < 0.25 {
					cell.Fg = c.Scale(1.6)
				}
			}
			row[x] = cell
		}
		out[y] = row
	}
	return out
}

// KittySupported guesses whether the terminal understands the Kitty
// graphics protocol (Kitty, Ghostty, WezTerm, Konsole).
func KittySupported() bool {
	if os.Getenv("SOPEBOX_KITTY") == "0" {
		return false
	}
	if os.Getenv("SOPEBOX_KITTY") == "1" || os.Getenv("KITTY_WINDOW_ID") != "" {
		return true
	}
	term := strings.ToLower(os.Getenv("TERM"))
	prog := strings.ToLower(os.Getenv("TERM_PROGRAM"))
	return strings.Contains(term, "kitty") || strings.Contains(term, "ghostty") ||
		prog == "ghostty" || prog == "wezterm" || strings.Contains(term, "wezterm") ||
		os.Getenv("KONSOLE_VERSION") != ""
}

// KittyImage encodes the image as a Kitty graphics protocol sequence that
// paints it at the current cursor position, sized to cols x rows cells.
func KittyImage(img image.Image, cols, rows int, id int) string {
	return kittySend(img, fmt.Sprintf("f=100,a=T,i=%d,q=2,c=%d,r=%d", id, cols, rows))
}

// KittyTransmit uploads an image under an id without showing it; place it
// afterwards (as often as needed) with KittyPlace.
func KittyTransmit(img image.Image, id int) string {
	return kittySend(img, fmt.Sprintf("f=100,a=t,i=%d,q=2", id))
}

// KittyPlace shows an uploaded image at the cursor, stretched over cols x
// rows cells, without moving the cursor. Placing it again moves it.
func KittyPlace(id, cols, rows int) string {
	return fmt.Sprintf("\x1b_Ga=p,i=%d,p=1,c=%d,r=%d,C=1,q=2\x1b\\", id, cols, rows)
}

// KittyUnplace hides an uploaded image but keeps it for placing again.
func KittyUnplace(id int) string {
	return fmt.Sprintf("\x1b_Ga=d,d=i,i=%d,q=2\x1b\\", id)
}

// kittySend encodes img as PNG and wraps it in (chunked) graphics commands.
func kittySend(img image.Image, keys string) string {
	var buf bytes.Buffer
	b := img.Bounds()
	if b.Dx() > 800 || b.Dy() > 800 {
		if b.Dx() >= b.Dy() {
			img = scale(img, 800, 800*b.Dy()/b.Dx())
		} else {
			img = scale(img, 800*b.Dx()/b.Dy(), 800)
		}
	}
	if err := png.Encode(&buf, img); err != nil {
		return ""
	}
	data := base64.StdEncoding.EncodeToString(buf.Bytes())
	var sb strings.Builder
	const chunk = 4096
	first := true
	for len(data) > 0 {
		n := chunk
		if n > len(data) {
			n = len(data)
		}
		more := 0
		if n < len(data) {
			more = 1
		}
		if first {
			fmt.Fprintf(&sb, "\x1b_G%s,m=%d;%s\x1b\\", keys, more, data[:n])
			first = false
		} else {
			fmt.Fprintf(&sb, "\x1b_Gm=%d;%s\x1b\\", more, data[:n])
		}
		data = data[n:]
	}
	return sb.String()
}

// KittyDelete removes the image with the given id (or all when id == 0).
func KittyDelete(id int) string {
	if id == 0 {
		return "\x1b_Ga=d,d=A,q=2;\x1b\\"
	}
	return fmt.Sprintf("\x1b_Ga=d,d=I,i=%d,q=2;\x1b\\", id)
}
