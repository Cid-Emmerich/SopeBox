package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"

	"github.com/Cid-Emmerich/SopeBox/internal/audio"
	"github.com/Cid-Emmerich/SopeBox/internal/config"
	"github.com/Cid-Emmerich/SopeBox/internal/download"
	"github.com/Cid-Emmerich/SopeBox/internal/feed"
	"github.com/Cid-Emmerich/SopeBox/internal/store"
	"github.com/Cid-Emmerich/SopeBox/internal/transcript"
	"github.com/Cid-Emmerich/SopeBox/internal/vis"
)

// newTestApp builds an app on a simulation screen with a small library
// and no audio device.
func newTestApp(t *testing.T) (*App, tcell.SimulationScreen) {
	t.Helper()
	cfg := config.Default()
	cfg.CacheDir = t.TempDir()
	cfg.LibraryPath = cfg.CacheDir + "/library.json"
	cfg.ConfigPath = cfg.CacheDir + "/rc"
	cfg.DownloadDir = cfg.CacheDir + "/dl"
	cfg.RefreshStart = false
	lib, _ := store.Open(cfg.LibraryPath)
	p := lib.AddPlaceholder("https://example.com/feed.xml", "Test Cast")
	p.Author = "Someone"
	p.Description = "A podcast about testing things."
	for i := 0; i < 5; i++ {
		p.Episodes = append(p.Episodes, feed.Episode{GUID: string(rune('a' + i)), Title: "Episode " + string(rune('A'+i)),
			URL: "https://example.com/ep.mp3", Duration: 1800 + float64(i*60), Published: time.Now().Add(-time.Duration(i) * 24 * time.Hour),
			Description: "Show notes for the episode.", Persons: []feed.Person{{Name: "Alice"}, {Name: "Bob"}}})
	}
	pl := audio.New() // not started: no device needed
	a := New(&cfg, lib, pl)
	scr := tcell.NewSimulationScreen("")
	if err := scr.Init(); err != nil {
		t.Fatal(err)
	}
	scr.SetSize(120, 36)
	a.scr = scr
	a.dl = download.New(lib, cfg.DownloadDir, 1, nil)
	return a, scr
}

func dump(scr tcell.SimulationScreen) string {
	cells, w, h := scr.GetContents()
	var sb strings.Builder
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := cells[y*w+x]
			if len(c.Runes) > 0 {
				sb.WriteRune(c.Runes[0])
			} else {
				sb.WriteByte(' ')
			}
		}
		sb.WriteByte('\n')
	}
	return sb.String()
}

func TestViewsRender(t *testing.T) {
	a, scr := newTestApp(t)
	for v := ViewNow; v < numViews; v++ {
		a.view = v
		a.draw()
		out := dump(scr)
		if !strings.Contains(out, "SopeBox") {
			t.Errorf("view %d: header missing", v)
		}
	}
	a.view = ViewPodcasts
	a.draw()
	out := dump(scr)
	if !strings.Contains(out, "Test Cast") || !strings.Contains(out, "Episode A") {
		t.Errorf("podcast view missing content:\n%s", out)
	}
	a.help = true
	a.draw()
	if !strings.Contains(dump(scr), "voice sensitivity") {
		t.Error("help missing")
	}
	a.help = false
}

func TestPlayAndCaptions(t *testing.T) {
	a, scr := newTestApp(t)
	it := a.lib.Items(a.lib.Podcasts[0])[0]
	a.cur = &it
	a.tracker.SetPersons([]string{"Alice", "Bob"})
	a.tr = &transcript.Transcript{Source: "test", Segments: []transcript.Segment{
		{Start: 0, End: 4, Speaker: "Alice", Text: "Hello there, this is a coffee podcast", Words: []transcript.Word{{Start: 0, End: 1, Text: "Hello"}, {Start: 1, End: 2, Text: "there,"}, {Start: 2, End: 2.5, Text: "this"}, {Start: 2.5, End: 3, Text: "is"}, {Start: 3, End: 3.4, Text: "a"}, {Start: 3.4, End: 3.8, Text: "coffee"}, {Start: 3.8, End: 4, Text: "podcast"}}},
		{Start: 4, End: 8, Speaker: "Bob", Text: "and I love fire", Words: []transcript.Word{{Start: 4, End: 5, Text: "and"}, {Start: 5, End: 6, Text: "I"}, {Start: 6, End: 7, Text: "love"}, {Start: 7, End: 8, Text: "fire"}}},
	}, Speakers: []string{"Alice", "Bob"}}
	a.tracker.SetTranscript(a.tr)
	a.trStatus = "ok"
	// simulate the word at 3.5s being spoken
	a.onWord(0, 5, 0.2)
	if !a.emoji.Active() || a.emoji.Def().Name != "coffee" {
		t.Fatalf("emoji not fired: %v", a.emoji.Def())
	}
	for _, name := range vis.Names() {
		a.visIdx = vis.Index(name)
		a.view = ViewNow
		a.draw()
		out := dump(scr)
		if !strings.Contains(out, "captions") {
			t.Errorf("%s: captions pane missing", name)
		}
	}
	a.cfg.CaptionSide = "bottom"
	a.draw()
	if !strings.Contains(dump(scr), "coffee") {
		t.Errorf("caption text missing:\n%s", dump(scr))
	}
	// keys should not panic
	for _, r := range "cCuUjJkLgGixypwWeE,.[];'R{}()<>MBKX" {
		a.handleKey(tcell.NewEventKey(tcell.KeyRune, r, 0))
		a.draw()
	}
	a.prompt.active = false
}

func TestKeysAcrossViews(t *testing.T) {
	a, scr := newTestApp(t)
	for v := ViewNow; v < numViews; v++ {
		a.switchView(v)
		for _, r := range "jkhlJKeExzomarCA/" {
			a.handleKey(tcell.NewEventKey(tcell.KeyRune, r, 0))
			a.prompt.active = false
			a.pv.typing = false
			a.sv.typing = false
			a.draw()
		}
		for _, k := range []tcell.Key{tcell.KeyUp, tcell.KeyDown, tcell.KeyLeft, tcell.KeyRight, tcell.KeyPgDn, tcell.KeyPgUp, tcell.KeyHome, tcell.KeyEnd, tcell.KeyEscape} {
			a.handleKey(tcell.NewEventKey(k, 0, 0))
			a.draw()
		}
	}
	_ = dump(scr)
}
