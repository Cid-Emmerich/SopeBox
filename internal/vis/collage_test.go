package vis

import (
	"image"
	"testing"

	"github.com/Cid-Emmerich/SopeBox/internal/theme"
)

func TestCollageLayoutFitsWithoutOverlap(t *testing.T) {
	for _, sz := range [][2]int{{90, 33}, {150, 40}, {50, 30}, {40, 40}, {200, 20}} {
		for n := 1; n <= 7; n++ {
			rs := collageLayout(n, sz[0], sz[1])
			if want := min(n, collageMax); len(rs) > want || len(rs) < min(n, 4) {
				t.Errorf("%v n=%d: %d tiles", sz, n, len(rs))
			}
			for i, r := range rs {
				if r.x < 0 || r.y < 0 || r.x+r.w > sz[0] || r.y+r.h > sz[1] || r.w <= 0 || r.h <= 0 {
					t.Errorf("%v n=%d: tile %d out of bounds %+v", sz, n, i, r)
				}
				for j := i + 1; j < len(rs); j++ {
					o := rs[j]
					if r.x < o.x+o.w && o.x < r.x+r.w && r.y < o.y+o.h && o.y < r.y+r.h {
						t.Errorf("%v n=%d: tiles %d and %d overlap", sz, n, i, j)
					}
				}
			}
			if len(rs) > 1 && rs[0].w*rs[0].h <= rs[1].w*rs[1].h {
				t.Errorf("%v n=%d: the newest tile should be the largest", sz, n)
			}
		}
	}
}

func TestCollageKittyLeavesPictureCellsBlank(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 30, 40))
	col := &Collage{Kitty: true, Tiles: []CollageTile{
		{Key: "a", Name: "Tewodros II", Kind: "person", Desc: "Emperor of Ethiopia", Img: img},
		{Key: "b", Name: "Magdala", Kind: "place"},
	}}
	c := NewCanvas(90, 33)
	s := &CollageStyle{}
	s.Draw(c, &Frame{Theme: theme.Get("sopebox"), Opts: &Options{}, Collage: col})
	if len(col.Place) != 1 {
		t.Fatalf("placements: %+v", col.Place)
	}
	p := col.Place[0]
	if p.X < 0 || p.Y < 0 || p.X+p.W > c.W || p.Y+p.H > c.H {
		t.Fatalf("placement off the canvas: %+v", p)
	}
	for y := p.Y; y < p.Y+p.H; y++ {
		for x := p.X; x < p.X+p.W; x++ {
			if c.Get(x, y).Ch != 0 {
				t.Fatalf("cell %d,%d under the picture is not blank", x, y)
			}
		}
	}
	// drawing again must not accumulate placements
	s.Draw(c, &Frame{Theme: theme.Get("sopebox"), Opts: &Options{}, Collage: col})
	if len(col.Place) != 1 {
		t.Fatalf("placements grew to %d", len(col.Place))
	}
}
