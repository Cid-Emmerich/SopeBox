package captions

import (
	"testing"

	"github.com/Cid-Emmerich/SopeBox/internal/glyph"
)

func TestTrigger(t *testing.T) {
	for word, name := range map[string]string{"Coffee,": "coffee", "laughing": "joy", "FIRE!": "fire", "rockets": "rocket", "really?": "raise", "[laughter]": "joy", "(music)": "music"} {
		i := Trigger(word)
		if i < 0 {
			t.Errorf("%q: no trigger", word)
			continue
		}
		if glyph.Registry[i].Name != name {
			t.Errorf("%q -> %q want %q", word, glyph.Registry[i].Name, name)
		}
	}
	for _, w := range []string{"the", "and", "like", "yes", "time", "a"} {
		if Trigger(w) >= 0 {
			t.Errorf("%q should not trigger", w)
		}
	}
	if TriggerCount() < 300 {
		t.Errorf("only %d triggers", TriggerCount())
	}
}

func TestWrap(t *testing.T) {
	lines := Wrap("Adam:", []string{"one", "two", "three", "four"}, 12)
	if len(lines) < 2 {
		t.Fatalf("lines = %v", lines)
	}
	if lines[0][0].Word != -1 {
		t.Error("prefix not marked")
	}
}

func TestEmoji(t *testing.T) {
	e := NewEmoji()
	e.Fire(glyph.Find("fire"), "fire")
	if !e.Active() {
		t.Fatal("not active")
	}
	cv := e.Render(20, 10, "blocks", "emoji", glyphPal(), false)
	if cv == nil {
		t.Fatal("no canvas")
	}
	lit := 0
	for _, c := range cv.Cells {
		if c.Ch != 0 {
			lit++
		}
	}
	if lit == 0 {
		t.Error("nothing rendered")
	}
	e.Fire(glyph.Find("cat"), "cat")
	for i := 0; i < 60; i++ {
		e.Step(0.05)
	}
	if e.Def().Name != "cat" {
		t.Errorf("queue did not advance: %s", e.Def().Name)
	}
	for i := 0; i < 200; i++ {
		e.Step(0.05)
	}
	if e.Active() {
		t.Error("should have expired")
	}
}
