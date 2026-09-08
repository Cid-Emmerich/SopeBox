// Package transcript loads captions for an episode: from the feed's
// Podcasting 2.0 transcript files (JSON, VTT, SRT, HTML) or generated
// locally with whisper.cpp. Everything ends up as timed words with
// optional speaker names.
package transcript

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Cid-Emmerich/SopeBox/internal/art"
	"github.com/Cid-Emmerich/SopeBox/internal/feed"
)

// Word is one timed token.
type Word struct {
	Start float64 `json:"s"`
	End   float64 `json:"e"`
	Text  string  `json:"t"`
}

// Segment is a caption line: a run of words by one speaker.
type Segment struct {
	Start   float64 `json:"start"`
	End     float64 `json:"end"`
	Speaker string  `json:"speaker,omitempty"`
	Text    string  `json:"text"`
	Words   []Word  `json:"words,omitempty"`
}

// Transcript is a whole episode's captions.
type Transcript struct {
	Source   string    `json:"source"` // "feed:json", "feed:srt", "whisper:base.en", ...
	Segments []Segment `json:"segments"`
	Speakers []string  `json:"speakers,omitempty"`
}

// HasSpeakers reports whether any segment carries a speaker label.
func (t *Transcript) HasSpeakers() bool { return t != nil && len(t.Speakers) > 0 }

// Duration is the end time of the last segment.
func (t *Transcript) Duration() float64 {
	if t == nil || len(t.Segments) == 0 {
		return 0
	}
	return t.Segments[len(t.Segments)-1].End
}

// At returns the segment index containing time pos, or the last segment
// that started before pos (and whether pos is inside it).
func (t *Transcript) At(pos float64) (int, bool) {
	if t == nil || len(t.Segments) == 0 {
		return -1, false
	}
	i := sort.Search(len(t.Segments), func(i int) bool { return t.Segments[i].Start > pos }) - 1
	if i < 0 {
		return -1, false
	}
	return i, pos <= t.Segments[i].End+0.4
}

// SpeakerAt returns the speaker talking at pos ("" if none / unknown).
func (t *Transcript) SpeakerAt(pos float64) string {
	i, in := t.At(pos)
	if i < 0 || !in {
		return ""
	}
	return t.Segments[i].Speaker
}

// WordAt returns the index of the word being spoken at pos inside segment
// i (-1 if between words).
func (t *Transcript) WordAt(i int, pos float64) int {
	if i < 0 || i >= len(t.Segments) {
		return -1
	}
	ws := t.Segments[i].Words
	for k, w := range ws {
		if pos >= w.Start && pos < w.End+0.05 {
			return k
		}
	}
	// after the last started word
	best := -1
	for k, w := range ws {
		if w.Start <= pos {
			best = k
		}
	}
	return best
}

// finish fills in words for segments that have none and collects speakers.
func (t *Transcript) finish() {
	seen := map[string]bool{}
	for i := range t.Segments {
		s := &t.Segments[i]
		s.Text = strings.Join(strings.Fields(s.Text), " ")
		if len(s.Words) == 0 && s.Text != "" {
			s.Words = spread(s.Text, s.Start, s.End)
		}
		if s.Speaker != "" && !seen[s.Speaker] {
			seen[s.Speaker] = true
			t.Speakers = append(t.Speakers, s.Speaker)
		}
	}
	sort.SliceStable(t.Segments, func(i, j int) bool { return t.Segments[i].Start < t.Segments[j].Start })
}

// spread assigns word times proportionally to character length.
func spread(text string, start, end float64) []Word {
	fields := strings.Fields(text)
	if len(fields) == 0 {
		return nil
	}
	if end <= start {
		end = start + 0.3*float64(len(fields))
	}
	total := 0
	for _, f := range fields {
		total += len([]rune(f)) + 1
	}
	out := make([]Word, 0, len(fields))
	t := start
	for _, f := range fields {
		d := (end - start) * float64(len([]rune(f))+1) / float64(total)
		out = append(out, Word{Start: t, End: t + d, Text: f})
		t += d
	}
	return out
}

// ---------------------------------------------------------------------------
// Fetching from the feed

// preference order of transcript MIME types
func rank(typ, url string) int {
	typ = strings.ToLower(typ)
	u := strings.ToLower(url)
	switch {
	case strings.Contains(typ, "json") || strings.HasSuffix(u, ".json"):
		return 0
	case strings.Contains(typ, "vtt") || strings.HasSuffix(u, ".vtt"):
		return 1
	case strings.Contains(typ, "srt") || strings.Contains(typ, "subrip") || strings.HasSuffix(u, ".srt"):
		return 2
	case strings.Contains(typ, "html"):
		return 3
	}
	return 4
}

// CachePath is where a parsed transcript is stored for a key.
func CachePath(cacheDir, key string) string {
	sum := sha1.Sum([]byte(key))
	return filepath.Join(cacheDir, "transcripts", hex.EncodeToString(sum[:8])+".json")
}

// LoadCached reads a cached transcript.
func LoadCached(path string) (*Transcript, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var t Transcript
	if err := json.Unmarshal(raw, &t); err != nil {
		return nil, err
	}
	return &t, nil
}

// SaveCached writes a transcript to the cache.
func SaveCached(path string, t *Transcript) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := json.Marshal(t)
	if err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o644)
}

// FromFeed downloads the best transcript the feed offers for an episode.
func FromFeed(ep *feed.Episode) (*Transcript, error) {
	if ep == nil || len(ep.Transcripts) == 0 {
		return nil, errors.New("the feed offers no transcript for this episode")
	}
	ts := append([]feed.Transcript(nil), ep.Transcripts...)
	sort.SliceStable(ts, func(i, j int) bool { return rank(ts[i].Type, ts[i].URL) < rank(ts[j].Type, ts[j].URL) })
	var lastErr error
	for _, tr := range ts {
		raw, err := art.Get(tr.URL)
		if err != nil {
			lastErr = err
			continue
		}
		t, err := ParseAny(raw, tr.Type, tr.URL)
		if err != nil || len(t.Segments) == 0 {
			if err == nil {
				err = errors.New("empty transcript")
			}
			lastErr = err
			continue
		}
		return t, nil
	}
	return nil, lastErr
}

// ParseAny parses a transcript by MIME type / extension, sniffing when
// the type is wrong.
func ParseAny(raw []byte, typ, url string) (*Transcript, error) {
	s := strings.TrimSpace(string(raw))
	switch {
	case strings.HasPrefix(s, "{") || strings.HasPrefix(s, "["):
		return ParseJSON(raw)
	case strings.HasPrefix(s, "WEBVTT"):
		return ParseVTT(s)
	case strings.HasPrefix(strings.ToLower(s), "<!doctype") || strings.HasPrefix(s, "<html") || strings.HasPrefix(s, "<div") || strings.HasPrefix(s, "<p"):
		return ParseHTML(s)
	}
	switch rank(typ, url) {
	case 0:
		return ParseJSON(raw)
	case 1:
		return ParseVTT(s)
	case 3:
		return ParseHTML(s)
	}
	return ParseSRT(s)
}
