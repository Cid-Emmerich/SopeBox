package transcript

import (
	"os"
	"testing"
)

func read(t *testing.T, name string) []byte {
	raw, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestJSONWordLevel(t *testing.T) {
	tr, err := ParseJSON(read(t, "p20.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(tr.Segments) == 0 {
		t.Fatal("no segments")
	}
	if !tr.HasSpeakers() || tr.Speakers[0] != "Speaker 2" {
		t.Errorf("speakers = %v", tr.Speakers)
	}
	s := tr.Segments[0]
	if len(s.Words) == 0 || s.Words[0].Text != "Overcome" {
		t.Errorf("first words = %+v", s.Words)
	}
	if got := tr.SpeakerAt(40); got != "Speaker 2" {
		t.Errorf("SpeakerAt(40) = %q", got)
	}
	if i, in := tr.At(40.3); i != 0 || !in {
		t.Errorf("At = %d %v", i, in)
	}
	if w := tr.WordAt(0, 40.3); w != 1 {
		t.Errorf("WordAt = %d", w)
	}
}

func TestSRT(t *testing.T) {
	tr, err := ParseSRT(string(read(t, "sample.srt")))
	if err != nil {
		t.Fatal(err)
	}
	if len(tr.Segments) < 3 {
		t.Fatalf("segments = %d", len(tr.Segments))
	}
	if tr.Segments[0].Speaker != "Speaker 2" {
		t.Errorf("speaker = %q", tr.Segments[0].Speaker)
	}
	if tr.Segments[0].Start < 39 || tr.Segments[0].Start > 40 {
		t.Errorf("start = %v", tr.Segments[0].Start)
	}
	if len(tr.Segments[0].Words) == 0 {
		t.Error("words not spread")
	}
}

func TestVTT(t *testing.T) {
	tr, err := ParseVTT(string(read(t, "sample.vtt")))
	if err != nil {
		t.Fatal(err)
	}
	if tr.Segments[0].Speaker != "Speaker 2" {
		t.Errorf("speaker = %q", tr.Segments[0].Speaker)
	}
	if tr.Segments[0].Text == "" || tr.Segments[0].Text[0] == '<' {
		t.Errorf("text = %q", tr.Segments[0].Text)
	}
}

func TestHTML(t *testing.T) {
	tr, err := ParseHTML(string(read(t, "sample.html")))
	if err != nil {
		t.Fatal(err)
	}
	if len(tr.Segments) == 0 {
		t.Fatal("no segments")
	}
	if tr.Segments[0].Start < 39 || tr.Segments[0].Start > 40 {
		t.Errorf("start = %v", tr.Segments[0].Start)
	}
}

func TestWhisperJSON(t *testing.T) {
	raw := []byte(`{"transcription":[{"offsets":{"from":0,"to":400},"text":" Hello"},{"offsets":{"from":400,"to":900},"text":" world."},{"offsets":{"from":2500,"to":2900},"text":" [SPEAKER_TURN] Hi"}]}`)
	tr, err := ParseJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(tr.Segments) != 2 {
		t.Fatalf("segments = %d: %+v", len(tr.Segments), tr.Segments)
	}
	if tr.Segments[1].Speaker != "Turn 1" {
		t.Errorf("turn speaker = %q", tr.Segments[1].Speaker)
	}
}

func TestBracketSpeakers(t *testing.T) {
	tr, err := ParseVTT("WEBVTT\n\n00:00:01.000 --> 00:00:02.000\n[SPEAKER_01]: Hello there.\n\n00:00:02.000 --> 00:00:03.000\n[SPEAKER_02]: Hi.\n")
	if err != nil {
		t.Fatal(err)
	}
	if len(tr.Speakers) != 2 || tr.Speakers[0] != "Speaker 1" || tr.Segments[0].Text != "Hello there." {
		t.Errorf("speakers=%v text=%q", tr.Speakers, tr.Segments[0].Text)
	}
}

func TestSniff(t *testing.T) {
	tr, err := ParseAny(read(t, "sample.vtt"), "application/x-subrip", "x.srt")
	if err != nil || tr.Source != "feed:vtt" {
		t.Errorf("sniff failed: %v %v", err, tr)
	}
}
