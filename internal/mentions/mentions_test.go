package mentions

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Cid-Emmerich/SopeBox/internal/transcript"
)

func sampleTranscript() *transcript.Transcript {
	return &transcript.Transcript{Source: "whisper:base.en", Segments: []transcript.Segment{
		{Start: 10, End: 14, Text: "So today we're talking about the Tudors.", Words: []transcript.Word{
			{Start: 10, Text: "So"}, {Start: 10.3, Text: "today"}, {Start: 11, Text: "we're"}, {Start: 11.4, Text: "talking"},
			{Start: 12, Text: "about"}, {Start: 12.5, Text: "the"}, {Start: 12.8, Text: "Tudors."}}},
		{Start: 14, End: 20, Speaker: "Tom", Text: "And of course Henry the Eighth, and Anne Bolin.", Words: []transcript.Word{
			{Start: 14, Text: "And"}, {Start: 14.2, Text: "of"}, {Start: 14.4, Text: "course"}, {Start: 15.1, Text: "Henry"},
			{Start: 15.5, Text: "the"}, {Start: 15.7, Text: "Eighth,"}, {Start: 16.9, Text: "and"}, {Start: 17.2, Text: "Anne"}, {Start: 17.6, Text: "Bolin."}}},
	}}
}

func TestFormatTranscript(t *testing.T) {
	got := FormatTranscript(sampleTranscript())
	want := "[10] So today we're talking about the Tudors.\n[14] Tom: And of course Henry the Eighth, and Anne Bolin.\n"
	if got != want {
		t.Fatalf("got %q\nwant %q", got, want)
	}
}

func TestParseRefinesTimes(t *testing.T) {
	raw := `{"mentions":[
		{"t":14,"spoken":"Anne Bolin","name":"Anne Boleyn","kind":"person","wiki":"Anne Boleyn"},
		{"t":10,"spoken":"the Tudors","name":"House of Tudor","kind":"group","wiki":"House of Tudor"},
		{"t":14,"spoken":"Henry the Eighth","name":"Henry VIII","kind":"person","wiki":"Henry VIII"},
		{"t":14,"spoken":"Henry","name":"Henry VIII","kind":"person","wiki":"Henry VIII"}]}`
	tl, err := parse(raw, sampleTranscript())
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, m := range tl.Mentions {
		got = append(got, m.Name)
	}
	if strings.Join(got, ",") != "House of Tudor,Henry VIII,Anne Boleyn" {
		t.Fatalf("order/dedupe wrong: %v", got)
	}
	wantAt := []float64{12.8, 15.1, 17.2} // "the" is skipped; misspelt "Bolin" still matches "Anne"
	for i, m := range tl.Mentions {
		if m.At != wantAt[i] {
			t.Errorf("%s at %.1f, want %.1f", m.Name, m.At, wantAt[i])
		}
	}
}

func TestParseRejectsGarbage(t *testing.T) {
	if _, err := parse("not json", sampleTranscript()); err == nil {
		t.Fatal("expected an error")
	}
}

func TestRefineFallsBackToLineStart(t *testing.T) {
	if at := refine(sampleTranscript(), 14, "Cromwell"); at != 14 {
		t.Fatalf("got %.1f, want the line start", at)
	}
}

func TestTidyDropsQuickRepeats(t *testing.T) {
	ms := tidy([]Mention{
		{At: 100, Name: "Henry VIII", Wiki: "Henry VIII"},
		{At: 130, Name: "Henry VIII", Wiki: "henry VIII"},
		{At: 400, Name: "Henry VIII", Wiki: "Henry VIII"},
		{At: 50, Name: ""},
	}, 60)
	if len(ms) != 2 || ms[0].At != 100 || ms[1].At != 400 {
		t.Fatalf("got %+v", ms)
	}
}

func TestRecentAndBetween(t *testing.T) {
	tl := &Timeline{Mentions: []Mention{
		{At: 10, Name: "Tudors", Wiki: "House of Tudor"},
		{At: 15, Name: "Henry VIII", Wiki: "Henry VIII"},
		{At: 17, Name: "Anne Boleyn", Wiki: "Anne Boleyn"},
		{At: 300, Name: "Henry VIII", Wiki: "Henry VIII"},
		{At: 320, Name: "Hampton Court", Wiki: "Hampton Court Palace"},
	}}
	if r := tl.Recent(5, 5); len(r) != 0 {
		t.Fatalf("nothing said yet, got %v", r)
	}
	r := tl.Recent(310, 5)
	if len(r) != 3 || r[0].Name != "Henry VIII" || r[0].At != 300 || r[1].Name != "Anne Boleyn" {
		t.Fatalf("recent = %+v", r)
	}
	if r := tl.Recent(1000, 2); len(r) != 2 || r[0].Name != "Hampton Court" {
		t.Fatalf("limit: %+v", r)
	}
	if b := tl.Between(15, 300); len(b) != 2 || b[0].Name != "Anne Boleyn" {
		t.Fatalf("between = %+v", b)
	}
	if tl.Distinct() != 4 {
		t.Fatalf("distinct = %d", tl.Distinct())
	}
}

func TestPickImage(t *testing.T) {
	th := "https://thumb.wikimedia.org/x/330px-Henry.jpg?utm=1"
	if got := pickImage(th, 330, 2040); got != "https://thumb.wikimedia.org/x/500px-Henry.jpg?utm=1" {
		t.Fatalf("upsized: %s", got)
	}
	if got := pickImage(th, 330, 400); got != th {
		t.Fatalf("small original should keep the thumbnail: %s", got)
	}
	if pickImage("", 0, 0) != "" {
		t.Fatal("no thumbnail")
	}
}

func TestLookupCachesAndHandlesMissing(t *testing.T) {
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		switch r.URL.Path {
		case "/Henry_VIII":
			w.Write([]byte(`{"type":"standard","title":"Henry VIII","description":"King of England from 1509 to 1547",
				"thumbnail":{"source":"https://thumb.example/330px-H.jpg","width":330},"originalimage":{"source":"https://x/H.jpg","width":2040}}`))
		case "/Mercury":
			w.Write([]byte(`{"type":"disambiguation","title":"Mercury","thumbnail":{"source":"https://thumb.example/330px-M.jpg","width":330}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	old := summaryBase
	summaryBase = srv.URL + "/"
	defer func() { summaryBase = old }()
	dir := t.TempDir()

	e, err := Lookup(dir, "Henry VIII")
	if err != nil || e.Description != "King of England from 1509 to 1547" || e.ImageURL != "https://thumb.example/500px-H.jpg" {
		t.Fatalf("lookup: %+v %v", e, err)
	}
	if _, err := Lookup(dir, "Henry VIII"); err != nil || hits != 1 {
		t.Fatalf("second lookup should come from the cache (hits=%d, err=%v)", hits, err)
	}
	if e, err := Lookup(dir, "Mercury"); err != nil || e.ImageURL != "" {
		t.Fatalf("disambiguation pages have no picture: %+v %v", e, err)
	}
	if e, err := Lookup(dir, "Nobody In Particular"); err != nil || !e.Missing {
		t.Fatalf("missing article: %+v %v", e, err)
	}
}

// longTranscript has the emperor named at 60s, 100s, 400s and 700s, and a
// shared alias ("Tewodros") that belongs to two emperors.
func longTranscript() *transcript.Transcript {
	seg := func(at float64, words ...string) transcript.Segment {
		s := transcript.Segment{Start: at, End: at + float64(len(words))}
		for i, w := range words {
			s.Words = append(s.Words, transcript.Word{Start: at + float64(i), Text: w})
		}
		s.Text = strings.Join(words, " ")
		return s
	}
	return &transcript.Transcript{Segments: []transcript.Segment{
		seg(60, "Teodros", "takes", "the", "throne."),
		seg(100, "And", "Teodros", "is", "furious."),
		seg(200, "Tewodros", "the", "First", "had", "tried", "this."),
		seg(400, "Meanwhile", "the", "emperor", "writes", "to", "Victoria."),
		seg(700, "Theodore,", "as", "the", "British", "called", "him."),
		seg(800, "Tewodros", "again."),
		seg(900, "Henry", "wrote", "back."),
	}}
}

func TestRecurBringsEntitiesBack(t *testing.T) {
	ms := []Mention{
		{At: 60, Name: "Tewodros II", Kind: "person", Wiki: "Tewodros II", Spoken: "Teodros", Aliases: []string{"Theodore", "the emperor", "Tewodros", "II"}},
		{At: 200, Name: "Tewodros I", Kind: "person", Wiki: "Tewodros I", Spoken: "Tewodros the First", Aliases: []string{"Tewodros"}},
	}
	got := tidy(recur(longTranscript(), ms), returnGap)
	var times []float64
	for _, m := range got {
		if m.Wiki == "Tewodros II" {
			times = append(times, m.At)
		}
	}
	// 100s is within the gap; 400s ("the emperor") and 700s ("Theodore,")
	// bring him back; 800s ("Tewodros") is shared with Tewodros I, so ignored
	if len(times) != 3 || times[0] != 60 || times[1] != 402 || times[2] != 700 {
		t.Fatalf("Tewodros II at %v", times)
	}
	for _, m := range got {
		if m.Wiki == "Tewodros I" && m.At != 200 {
			t.Fatalf("shared alias must not bring back Tewodros I: %+v", m)
		}
	}
	// "Henry" is in two entities' names, so it is not a trigger for either
	ms = append(ms, Mention{At: 10, Name: "Henry V", Wiki: "Henry V of England", Aliases: []string{"Henry"}},
		Mention{At: 20, Name: "Henry Blanc", Aliases: []string{"Blonk"}})
	for _, m := range recur(longTranscript(), ms) {
		if m.At == 900 {
			t.Fatalf("ambiguous first name matched: %+v", m)
		}
	}
}

func TestRebuildFromOldTimeline(t *testing.T) {
	// a timeline saved before Found existed: originals plus old recurrences
	tl := &Timeline{Mentions: []Mention{
		{At: 60, Name: "Tewodros II", Wiki: "Tewodros II", Spoken: "Teodros", Aliases: []string{"Theodore"}},
		{At: 100, Name: "Tewodros II", Wiki: "Tewodros II", Spoken: "teodros"},
	}}
	tl.Rebuild(longTranscript())
	if len(tl.Found) != 1 || tl.Found[0].At != 60 {
		t.Fatalf("found = %+v", tl.Found)
	}
	if n := len(tl.Mentions); n != 2 || tl.Mentions[1].At != 700 {
		t.Fatalf("mentions = %+v", tl.Mentions)
	}
	tl.Rebuild(longTranscript()) // idempotent
	if len(tl.Mentions) != 2 {
		t.Fatalf("second rebuild changed it: %+v", tl.Mentions)
	}
}
