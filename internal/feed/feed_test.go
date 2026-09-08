package feed

import (
	"os"
	"testing"
)

func load(t *testing.T, name string) *Feed {
	raw, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	f, err := Parse(raw)
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	return f
}

func TestParseArt19(t *testing.T) {
	f := load(t, "art19.xml")
	if f.Title != "Dear Hank & John" {
		t.Errorf("title = %q", f.Title)
	}
	if f.Image == "" {
		t.Error("no image")
	}
	if len(f.Episodes) != 3 {
		t.Fatalf("episodes = %d", len(f.Episodes))
	}
	e := f.Episodes[0]
	if e.URL == "" || e.Duration < 2000 || e.Published.IsZero() {
		t.Errorf("episode not parsed: %+v", e)
	}
	if len(f.Categories) == 0 {
		t.Error("no categories")
	}
}

func TestParsePodcasting20(t *testing.T) {
	f := load(t, "noagenda.xml")
	if len(f.Episodes) == 0 {
		t.Fatal("no episodes")
	}
	e := f.Episodes[0]
	if len(e.Transcripts) == 0 {
		t.Error("no transcript tags")
	}
	if e.Chapters == "" {
		t.Error("no chapters")
	}
	if len(e.Persons) == 0 {
		t.Error("no persons")
	}
	f2 := load(t, "buzzsprout.xml")
	n := 0
	for _, e := range f2.Episodes {
		n += len(e.Transcripts)
	}
	if n == 0 {
		t.Error("buzzsprout: no transcripts parsed")
	}
}

func TestDuration(t *testing.T) {
	for in, want := range map[string]float64{"3600": 3600, "1:00:00": 3600, "48:16": 2896, "0:05": 5, "": 0, "abc": 0} {
		if got := ParseDuration(in); got != want {
			t.Errorf("ParseDuration(%q) = %v want %v", in, got, want)
		}
	}
}

func TestStripHTML(t *testing.T) {
	got := StripHTML("<p>Hello <b>world</b></p><p>Second&amp;line</p>")
	if got != "Hello world\nSecond&line" {
		t.Errorf("got %q", got)
	}
}
