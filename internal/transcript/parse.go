package transcript

import (
	"encoding/json"
	"errors"
	"regexp"
	"strconv"
	"strings"

	"github.com/Cid-Emmerich/SopeBox/internal/feed"
)

// ---------------------------------------------------------------------------
// JSON: Podcasting 2.0 and whisper.cpp shapes

type p20Segment struct {
	Speaker   string  `json:"speaker"`
	StartTime float64 `json:"startTime"`
	EndTime   float64 `json:"endTime"`
	Body      string  `json:"body"`
}

type whisperOut struct {
	Transcription []struct {
		Offsets struct {
			From int64 `json:"from"`
			To   int64 `json:"to"`
		} `json:"offsets"`
		Text string `json:"text"`
	} `json:"transcription"`
}

// ParseJSON handles the Podcasting 2.0 JSON transcript (word or phrase
// level) and whisper.cpp's -oj output.
func ParseJSON(raw []byte) (*Transcript, error) {
	var p20 struct {
		Segments []p20Segment `json:"segments"`
	}
	if err := json.Unmarshal(raw, &p20); err == nil && len(p20.Segments) > 0 {
		return fromP20(p20.Segments, "feed:json"), nil
	}
	var wh whisperOut
	if err := json.Unmarshal(raw, &wh); err == nil && len(wh.Transcription) > 0 {
		var words []Word
		for _, s := range wh.Transcription {
			txt := strings.TrimSpace(s.Text)
			if txt == "" {
				continue
			}
			words = append(words, Word{Start: float64(s.Offsets.From) / 1000, End: float64(s.Offsets.To) / 1000, Text: txt})
		}
		return FromWords(words, "whisper"), nil
	}
	var arr []p20Segment
	if err := json.Unmarshal(raw, &arr); err == nil && len(arr) > 0 {
		return fromP20(arr, "feed:json"), nil
	}
	return nil, errors.New("unrecognised json transcript")
}

func fromP20(segs []p20Segment, src string) *Transcript {
	// Word-level files have one token per segment; group them into caption
	// lines per speaker. Phrase-level files are used as is.
	t := &Transcript{Source: src}
	var cur *Segment
	flush := func() {
		if cur != nil && strings.TrimSpace(cur.Text) != "" {
			t.Segments = append(t.Segments, *cur)
		}
		cur = nil
	}
	for _, s := range segs {
		body := strings.TrimSpace(s.Body)
		if body == "" {
			continue
		}
		isWord := !strings.Contains(body, " ")
		if cur == nil || cur.Speaker != s.Speaker || (isWord && (len(cur.Words) >= 14 || s.StartTime-cur.End > 1.2)) || !isWord {
			flush()
			cur = &Segment{Start: s.StartTime, End: s.EndTime, Speaker: s.Speaker}
		}
		if isWord {
			cur.Words = append(cur.Words, Word{Start: s.StartTime, End: s.EndTime, Text: body})
			cur.Text = strings.TrimSpace(cur.Text + " " + body)
		} else {
			cur.Text = body
		}
		if s.EndTime > cur.End {
			cur.End = s.EndTime
		}
		if !isWord {
			flush()
		}
	}
	flush()
	t.finish()
	return t
}

// FromWords groups timed words into caption segments split on pauses,
// sentence ends and "[SPEAKER_TURN]" markers from tinydiarize.
func FromWords(words []Word, src string) *Transcript {
	t := &Transcript{Source: src}
	var cur *Segment
	turn := 0
	flush := func() {
		if cur != nil && len(cur.Words) > 0 {
			t.Segments = append(t.Segments, *cur)
		}
		cur = nil
	}
	for _, w := range words {
		if strings.Contains(w.Text, "[SPEAKER_TURN]") {
			w.Text = strings.TrimSpace(strings.ReplaceAll(w.Text, "[SPEAKER_TURN]", ""))
			flush()
			turn++
			if w.Text == "" {
				continue
			}
		}
		if cur != nil {
			gap := w.Start - cur.End
			last := cur.Words[len(cur.Words)-1].Text
			ends := strings.HasSuffix(last, ".") || strings.HasSuffix(last, "?") || strings.HasSuffix(last, "!")
			if gap > 0.9 || len(cur.Words) >= 16 || (ends && len(cur.Words) >= 6) {
				flush()
			}
		}
		if cur == nil {
			cur = &Segment{Start: w.Start, End: w.End}
			if turn > 0 {
				cur.Speaker = "Turn " + strconv.Itoa(turn)
			}
		}
		cur.Words = append(cur.Words, w)
		cur.Text = strings.TrimSpace(cur.Text + " " + w.Text)
		if w.End > cur.End {
			cur.End = w.End
		}
	}
	flush()
	t.finish()
	return t
}

// ---------------------------------------------------------------------------
// SRT / VTT

var (
	timeRe    = regexp.MustCompile(`(\d+):(\d+):(\d+)(?:[.,](\d+))?|(\d+):(\d+)(?:[.,](\d+))?`)
	cueRe     = regexp.MustCompile(`(?m)^\s*(\d+:\d+(?::\d+)?[.,]\d+)\s*-->\s*(\d+:\d+(?::\d+)?[.,]\d+)`)
	speakerRe = regexp.MustCompile(`^\[?([A-Z][\w .'\-]{0,40}?)\]?:\s+(.*)$`)
	genericRe = regexp.MustCompile(`(?i)^speaker[_ ]?0*(\d+)$`)
	vttVoice  = regexp.MustCompile(`<v\s+([^>]+)>`)
	tagRe     = regexp.MustCompile(`<[^>]+>`)
)

func parseTime(s string) float64 {
	m := timeRe.FindStringSubmatch(s)
	if m == nil {
		return 0
	}
	if m[1] != "" {
		h, _ := strconv.Atoi(m[1])
		mi, _ := strconv.Atoi(m[2])
		se, _ := strconv.Atoi(m[3])
		ms := 0.0
		if m[4] != "" {
			ms, _ = strconv.ParseFloat("0."+m[4], 64)
		}
		return float64(h*3600+mi*60+se) + ms
	}
	mi, _ := strconv.Atoi(m[5])
	se, _ := strconv.Atoi(m[6])
	ms := 0.0
	if m[7] != "" {
		ms, _ = strconv.ParseFloat("0."+m[7], 64)
	}
	return float64(mi*60+se) + ms
}

// ParseSRT parses SubRip captions. "Name: text" prefixes become speakers.
func ParseSRT(s string) (*Transcript, error) {
	return parseCues(s, "feed:srt")
}

// ParseVTT parses WebVTT captions, including <v Speaker> voice tags.
func ParseVTT(s string) (*Transcript, error) {
	return parseCues(s, "feed:vtt")
}

func parseCues(s, src string) (*Transcript, error) {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	locs := cueRe.FindAllStringSubmatchIndex(s, -1)
	if len(locs) == 0 {
		return nil, errors.New("no cues found")
	}
	t := &Transcript{Source: src}
	lastSpeaker := ""
	for i, loc := range locs {
		start := parseTime(s[loc[2]:loc[3]])
		end := parseTime(s[loc[4]:loc[5]])
		bodyEnd := len(s)
		if i+1 < len(locs) {
			bodyEnd = locs[i+1][0]
		}
		body := s[loc[1]:bodyEnd]
		// drop the trailing cue number of the next block
		lines := strings.Split(strings.TrimSpace(body), "\n")
		if n := len(lines); n > 0 {
			if _, err := strconv.Atoi(strings.TrimSpace(lines[n-1])); err == nil {
				lines = lines[:n-1]
			}
		}
		text := strings.TrimSpace(strings.Join(lines, " "))
		speaker := ""
		if m := vttVoice.FindStringSubmatch(text); m != nil {
			speaker = strings.TrimSpace(m[1])
		}
		text = tagRe.ReplaceAllString(text, "")
		if m := speakerRe.FindStringSubmatch(text); m != nil && speaker == "" {
			speaker = strings.TrimSpace(m[1])
			text = m[2]
		}
		speaker = prettySpeaker(speaker)
		if speaker == "" {
			speaker = lastSpeaker
		} else {
			lastSpeaker = speaker
		}
		text = strings.TrimSpace(text)
		if text == "" {
			continue
		}
		seg := Segment{Start: start, End: end, Speaker: speaker, Text: text, Words: spread(text, start, end)}
		// Cues are short display chunks; fold them into sentence-sized
		// captions so lines read naturally and the spoken word still lights.
		if n := len(t.Segments); n > 0 {
			prev := &t.Segments[n-1]
			ends := strings.HasSuffix(prev.Text, ".") || strings.HasSuffix(prev.Text, "?") || strings.HasSuffix(prev.Text, "!")
			if prev.Speaker == seg.Speaker && start-prev.End < 0.6 && !ends && len(prev.Words)+len(seg.Words) <= 28 {
				prev.Text += " " + seg.Text
				prev.Words = append(prev.Words, seg.Words...)
				if end > prev.End {
					prev.End = end
				}
				continue
			}
		}
		t.Segments = append(t.Segments, seg)
	}
	t.finish()
	return t, nil
}

// ---------------------------------------------------------------------------
// HTML (buzzsprout style: <cite>Name:</cite> <time>0:00</time> <p>text</p>)

var (
	citeRe  = regexp.MustCompile(`(?is)<cite[^>]*>(.*?)</cite>`)
	htimeRe = regexp.MustCompile(`(?is)<time[^>]*>(.*?)</time>`)
	pRe     = regexp.MustCompile(`(?is)<p[^>]*>(.*?)</p>`)
)

// ParseHTML parses transcript HTML by locating <time> stamps and the text
// that follows each one.
func ParseHTML(s string) (*Transcript, error) {
	t := &Transcript{Source: "feed:html"}
	times := htimeRe.FindAllStringSubmatchIndex(s, -1)
	if len(times) == 0 {
		return nil, errors.New("no timestamps in html transcript")
	}
	for i, loc := range times {
		start := parseTime(feed.StripHTML(s[loc[2]:loc[3]]))
		chunkEnd := len(s)
		if i+1 < len(times) {
			chunkEnd = times[i+1][0]
		}
		chunk := s[loc[1]:chunkEnd]
		before := s[max(0, loc[0]-300):loc[0]]
		speaker := ""
		if m := citeRe.FindAllStringSubmatch(before, -1); len(m) > 0 {
			speaker = strings.TrimSuffix(strings.TrimSpace(feed.StripHTML(m[len(m)-1][1])), ":")
		}
		var text string
		if ps := pRe.FindAllStringSubmatch(chunk, -1); len(ps) > 0 {
			var parts []string
			for _, p := range ps {
				parts = append(parts, feed.StripHTML(p[1]))
			}
			text = strings.Join(parts, " ")
		} else {
			text = feed.StripHTML(chunk)
		}
		text = strings.Join(strings.Fields(text), " ")
		if text == "" {
			continue
		}
		t.Segments = append(t.Segments, Segment{Start: start, Speaker: speaker, Text: text})
	}
	for i := range t.Segments {
		if i+1 < len(t.Segments) {
			t.Segments[i].End = t.Segments[i+1].Start
		} else {
			t.Segments[i].End = t.Segments[i].Start + 0.35*float64(len(strings.Fields(t.Segments[i].Text)))
		}
	}
	t.finish()
	return t, nil
}

// prettySpeaker turns diarizer labels like "SPEAKER_01" into "Speaker 1".
func prettySpeaker(s string) string {
	s = strings.Trim(strings.TrimSpace(s), "[]")
	if m := genericRe.FindStringSubmatch(s); m != nil {
		return "Speaker " + m[1]
	}
	return s
}
