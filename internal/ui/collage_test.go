package ui

import (
	"bytes"
	"image"
	"image/color"
	"strings"
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"

	"github.com/Cid-Emmerich/SopeBox/internal/mentions"
	"github.com/Cid-Emmerich/SopeBox/internal/transcript"
	"github.com/Cid-Emmerich/SopeBox/internal/vis"
)

func portrait() image.Image {
	img := image.NewRGBA(image.Rect(0, 0, 60, 80))
	for y := 0; y < 80; y++ {
		for x := 0; x < 60; x++ {
			img.Set(x, y, color.RGBA{uint8(x * 4), 90, uint8(y * 3), 255})
		}
	}
	return img
}

// collageApp is a test app playing an episode with a timeline and
// Wikipedia answers already in memory (no network).
func collageApp(t *testing.T) (*App, tcell.SimulationScreen) {
	a, scr := newTestApp(t)
	it := a.lib.Items(a.lib.Podcasts[0])[0]
	a.cur = &it
	a.tr = &transcript.Transcript{Source: "test", Segments: []transcript.Segment{{Start: 0, End: 5, Text: "Henry the Eighth at Hampton Court"}}}
	a.trStatus = "ok"
	a.tl = &mentions.Timeline{Mentions: []mentions.Mention{
		{At: -20, Name: "Hampton Court Palace", Kind: "place", Wiki: "Hampton Court Palace"},
		{At: -3, Name: "Henry VIII", Kind: "person", Wiki: "Henry VIII"},
		{At: -1, Name: "the Field of the Cloth of Gold", Kind: "event"},
		{At: 400, Name: "Anne Boleyn", Kind: "person", Wiki: "Anne Boleyn"},
	}}
	a.tlStatus = "ok"
	a.wiki["henry viii"] = &wikiItem{entry: &mentions.Entry{Title: "Henry VIII", Description: "King of England from 1509 to 1547"}, img: portrait()}
	a.wiki["hampton court palace"] = &wikiItem{entry: &mentions.Entry{Title: "Hampton Court Palace", Description: "Royal palace in London"}, img: portrait()}
	a.wiki["anne boleyn"] = &wikiItem{loading: true} // pretend it's on its way
	a.setVis("collage")
	a.view = ViewNow
	scr.SetSize(150, 42)
	return a, scr
}

func TestCollageDrawsTilesAsBlocks(t *testing.T) {
	a, scr := collageApp(t)
	a.draw()
	out := dump(scr)
	for _, want := range []string{"Henry VIII", "King of England", "Hampton Court Palace", "Field of the Cloth", "▀"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "Anne Boleyn") {
		t.Error("a mention 400s in the future must not be shown yet")
	}
	// the newest mention is the big tile: its name sits left of the others
	lines := strings.Split(out, "\n")
	col := func(s string) int {
		for _, l := range lines {
			if i := strings.Index(l, s); i >= 0 {
				return len([]rune(l[:i]))
			}
		}
		return -1
	}
	if c1, c2 := col("Field of the Cloth"), col("Henry VIII"); c1 < 0 || c2 < 0 || c1 > c2 {
		t.Errorf("newest mention should be the large left tile (cols %d, %d)", c1, c2)
	}
}

func TestCollageKittyUploadsOnceAndHides(t *testing.T) {
	a, _ := collageApp(t)
	var buf bytes.Buffer
	a.canKitty = true
	a.kittyOut = &buf
	a.draw()
	first := buf.String()
	if strings.Count(first, "a=t,") != 2 || strings.Count(first, "a=p,") != 2 {
		t.Fatalf("expected two uploads and two placements, got %q", abbrev(first))
	}
	if !strings.HasPrefix(first, "\x1b7") || !strings.HasSuffix(first, "\x1b8") {
		t.Error("cursor must be saved and restored around graphics")
	}
	buf.Reset()
	a.draw()
	if buf.Len() != 0 {
		t.Fatalf("an unchanged collage must not resend anything, got %q", abbrev(buf.String()))
	}
	a.help = true
	a.draw()
	if strings.Count(buf.String(), "d=i,") != 2 {
		t.Fatalf("help overlay should hide both pictures, got %q", abbrev(buf.String()))
	}
	buf.Reset()
	a.help = false
	a.draw()
	if strings.Contains(buf.String(), "a=t,") || strings.Count(buf.String(), "a=p,") != 2 {
		t.Fatalf("closing help should re-place without re-uploading, got %q", abbrev(buf.String()))
	}
	buf.Reset()
	it := a.lib.Items(a.lib.Podcasts[0])[1]
	a.load(it, true)
	if strings.Count(buf.String(), "d=I,") != 2 {
		t.Fatalf("changing episode should free the pictures, got %q", abbrev(buf.String()))
	}
}

func abbrev(s string) string {
	s = strings.ReplaceAll(s, "\x1b", "⎋")
	if len(s) > 300 {
		return s[:300] + "…"
	}
	return s
}

func TestCollageStatusMessages(t *testing.T) {
	a, scr := collageApp(t)
	a.tl, a.tlStatus = nil, ""
	a.tr, a.trStatus = nil, "none"
	a.draw()
	if out := dump(scr); !strings.Contains(out, "needs a transcript") {
		t.Errorf("no transcript message:\n%s", out)
	}
	a.tr, a.trStatus = &transcript.Transcript{Source: "t", Segments: []transcript.Segment{{Start: 0, End: 1, Text: "hi"}}}, "ok"
	a.tlCheck = time.Time{}
	a.draw()
	if out := dump(scr); !strings.Contains(out, "needs a Claude API key") {
		t.Errorf("no key message:\n%s", out)
	}
	a.tlStatus, a.tlErr = "failed", "HTTP 529 overloaded"
	a.draw()
	if out := dump(scr); !strings.Contains(out, "HTTP 529 overloaded") || !strings.Contains(out, "press P") {
		t.Errorf("failure message:\n%s", out)
	}
}

func TestCollageLoadsCachedTimeline(t *testing.T) {
	a, _ := collageApp(t)
	it := *a.cur
	tl := &mentions.Timeline{Model: "claude-opus-5-5", Mentions: []mentions.Mention{{At: 1, Name: "Wolsey", Kind: "person"}}}
	if err := mentions.SaveCached(mentions.CachePath(a.cfg.CacheDir, cacheKeyOf(it)), tl); err != nil {
		t.Fatal(err)
	}
	a.tl, a.tlStatus = nil, ""
	events := make(chan tcell.Event, 8)
	go func() {
		for {
			ev := a.scr.PollEvent()
			if ev == nil {
				return
			}
			events <- ev
		}
	}()
	a.ensureMentions()
	if a.tlStatus != "loading" {
		t.Fatalf("status %q", a.tlStatus)
	}
	select {
	case ev := <-events:
		a.handle(ev)
	case <-time.After(3 * time.Second):
		t.Fatal("no event")
	}
	if a.tl == nil || a.tl.Mentions[0].Name != "Wolsey" || a.tlStatus != "ok" {
		t.Fatalf("timeline not loaded: %+v %q", a.tl, a.tlStatus)
	}
}

func TestPrepEventsFeedThePlayingEpisode(t *testing.T) {
	a, _ := collageApp(t)
	k := a.cur.Key()
	a.tr, a.trStatus, a.tl, a.tlStatus = nil, "none", nil, ""
	a.prepBusy, a.prepKey, a.prepTitle = true, k, "Episode A"
	a.trBusy[k], a.tlBusy[k] = true, true
	tr := &transcript.Transcript{Source: "whisper:base.en", Segments: []transcript.Segment{{Start: 0, End: 1, Text: "hello"}}}
	a.onPrep(prepEvent{key: k, tr: tr, phase: "Claude is reading it"})
	if a.tr != tr || a.trBusy[k] {
		t.Fatal("transcript not handed to the playing episode")
	}
	a.onPrep(prepEvent{key: k, tl: &mentions.Timeline{Mentions: []mentions.Mention{{At: 0, Name: "Cromwell"}}}})
	if a.tl == nil || a.tlBusy[k] {
		t.Fatal("timeline not handed to the playing episode")
	}
	a.onPrep(prepEvent{key: k, done: true})
	if a.prepBusy || !a.prepDone[k] {
		t.Fatal("prep should be finished")
	}
}

func TestCollagePrepToggle(t *testing.T) {
	a, _ := newTestApp(t)
	p := a.lib.Podcasts[0]
	p.AutoDownload = -1
	a.toggleCollagePrep(p)
	if !p.Collage || p.AutoDownload != 1 {
		t.Fatalf("collage=%v auto=%d", p.Collage, p.AutoDownload)
	}
	if !strings.Contains(a.toast, "ANTHROPIC_API_KEY") {
		t.Errorf("should warn about the missing key: %q", a.toast)
	}
	a.toggleCollagePrep(p)
	if p.Collage {
		t.Fatal("second toggle should turn it off")
	}
	// P in the now-playing view with nothing playing must not panic
	a.view = ViewNow
	a.handleKey(tcell.NewEventKey(tcell.KeyRune, 'P', 0))
	if vis.Registry[a.visIdx].Name() == "" {
		t.Fatal("unreachable")
	}
}
