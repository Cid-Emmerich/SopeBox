// Package mentions finds the people, places, events and things an episode
// talks about, and when. Claude reads the whole transcript once and returns
// a timeline; Wikipedia supplies a picture and a one-line description for
// each entry. The collage visualizer shows them as they are mentioned.
package mentions

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/Cid-Emmerich/SopeBox/internal/transcript"
)

// Kinds are the categories Claude sorts mentions into.
var Kinds = []string{"person", "place", "event", "group", "work", "thing", "era"}

// Mention is one moment an entity comes up in the conversation.
type Mention struct {
	At      float64  `json:"at"`                // seconds into the episode, at the spoken word
	Name    string   `json:"name"`              // canonical name ("Henry VIII")
	Kind    string   `json:"kind"`              // one of Kinds
	Wiki    string   `json:"wiki,omitempty"`    // English Wikipedia article title
	Spoken  string   `json:"spoken,omitempty"`  // the words as transcribed
	Aliases []string `json:"aliases,omitempty"` // other ways it is referred to
}

// Key identifies the entity so repeated mentions share one tile.
func (m Mention) Key() string {
	if m.Wiki != "" {
		return strings.ToLower(m.Wiki)
	}
	return strings.ToLower(m.Name)
}

// Timeline is every mention in an episode, in time order.
type Timeline struct {
	Model        string    `json:"model"`
	Source       string    `json:"source"` // the transcript it was made from
	Created      time.Time `json:"created"`
	InputTokens  int64     `json:"input_tokens,omitempty"`
	OutputTokens int64     `json:"output_tokens,omitempty"`
	Mentions     []Mention `json:"mentions"`
	Found        []Mention `json:"found,omitempty"` // Claude's own list, before recur
}

// Rebuild recomputes the mentions from Claude's list and the transcript,
// so improvements to matching apply to timelines already paid for.
// Timelines saved before Found existed use the first mention of each
// entity, which is what Claude listed.
func (t *Timeline) Rebuild(tr *transcript.Transcript) {
	if t == nil || tr == nil {
		return
	}
	if len(t.Found) == 0 {
		seen := map[string]bool{}
		for _, m := range t.Mentions {
			if !seen[m.Key()] {
				seen[m.Key()] = true
				t.Found = append(t.Found, m)
			}
		}
	}
	found := append([]Mention(nil), t.Found...)
	t.Mentions = tidy(recur(tr, found), returnGap)
}

// Recent returns up to n distinct entities mentioned at or before pos,
// most recently mentioned first. Each carries its latest mention time.
func (t *Timeline) Recent(pos float64, n int) []Mention {
	if t == nil || n <= 0 {
		return nil
	}
	i := sort.Search(len(t.Mentions), func(i int) bool { return t.Mentions[i].At > pos })
	seen := map[string]bool{}
	var out []Mention
	for i--; i >= 0 && len(out) < n; i-- {
		m := t.Mentions[i]
		if seen[m.Key()] {
			continue
		}
		seen[m.Key()] = true
		out = append(out, m)
	}
	return out
}

// Between returns the mentions with from < At <= to.
func (t *Timeline) Between(from, to float64) []Mention {
	if t == nil {
		return nil
	}
	i := sort.Search(len(t.Mentions), func(i int) bool { return t.Mentions[i].At > from })
	var out []Mention
	for ; i < len(t.Mentions) && t.Mentions[i].At <= to; i++ {
		out = append(out, t.Mentions[i])
	}
	return out
}

// Distinct counts the different entities in the timeline.
func (t *Timeline) Distinct() int {
	if t == nil {
		return 0
	}
	seen := map[string]bool{}
	for _, m := range t.Mentions {
		seen[m.Key()] = true
	}
	return len(seen)
}

// ---------------------------------------------------------------------------
// Cache

// CachePath is where an episode's timeline is stored. The key is the same
// one transcripts use (feed URL + "|" + episode GUID).
func CachePath(cacheDir, key string) string {
	sum := sha1.Sum([]byte(key))
	return filepath.Join(cacheDir, "mentions", hex.EncodeToString(sum[:8])+".json")
}

// LoadCached reads a cached timeline.
func LoadCached(path string) (*Timeline, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var t Timeline
	if err := json.Unmarshal(raw, &t); err != nil {
		return nil, err
	}
	return &t, nil
}

// SaveCached writes a timeline to the cache.
func SaveCached(path string, t *Timeline) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(t, "", " ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o644)
}

// ---------------------------------------------------------------------------
// Timing

// refine moves a mention from the start of its caption line to the moment
// its first spoken word is said, so the tile appears as the name is heard.
func refine(tr *transcript.Transcript, lineStart float64, spoken string) float64 {
	target := firstWord(spoken)
	if tr == nil || target == "" {
		return lineStart
	}
	i := sort.Search(len(tr.Segments), func(i int) bool { return tr.Segments[i].Start >= lineStart-0.5 })
	for ; i < len(tr.Segments) && tr.Segments[i].Start <= lineStart+30; i++ {
		for _, w := range tr.Segments[i].Words {
			if norm(w.Text) == target {
				return w.Start
			}
		}
	}
	return lineStart
}

// firstWord picks the first word of a phrase worth matching on, skipping
// articles, which match far too early.
func firstWord(s string) string {
	skip := map[string]bool{"the": true, "a": true, "an": true, "of": true}
	for _, f := range strings.Fields(s) {
		if w := norm(f); w != "" && !skip[w] {
			return w
		}
	}
	return ""
}

func norm(s string) string {
	return strings.ToLower(strings.TrimFunc(s, func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) }))
}

// returnGap is how long an entity must go unmentioned before its tile is
// brought back to the front when it comes up again.
const returnGap = 180

// recur adds a mention wherever an entity's name, spoken form or aliases
// are said again after Claude's first mention of it, so a tile comes back
// when the conversation returns to it. tidy thins them out afterwards.
func recur(tr *transcript.Transcript, ms []Mention) []Mention {
	if tr == nil {
		return ms
	}
	type phrase struct {
		words []string
		m     Mention
	}
	first := map[string]float64{}
	owner := map[string]string{} // phrase -> entity key ("" when shared)
	tmpl := map[string]Mention{}
	inNames := map[string]map[string]bool{} // word -> entities whose name has it
	for _, m := range ms {
		for _, w := range phraseWords(m.Name) {
			if inNames[w] == nil {
				inNames[w] = map[string]bool{}
			}
			inNames[w][m.Key()] = true
		}
	}
	for _, m := range ms {
		k := m.Key()
		if t, ok := first[k]; !ok || m.At < t {
			first[k] = m.At
			tmpl[k] = Mention{Name: m.Name, Kind: m.Kind, Wiki: m.Wiki}
		}
		for _, p := range append([]string{m.Name, m.Spoken}, m.Aliases...) {
			ws := strings.Join(phraseWords(p), " ")
			if ws == "" {
				continue
			}
			if !strings.Contains(ws, " ") && len(inNames[ws]) > 1 {
				// "Henry" alone could be Henry V, Henry VII or Henry Stern
				owner[ws] = ""
				continue
			}
			if o, ok := owner[ws]; ok && o != k {
				owner[ws] = "" // "Tewodros" for Tewodros I and II: trust neither
			} else if !ok {
				owner[ws] = k
			}
		}
	}
	byWord := map[string][]phrase{} // indexed by the phrase's first word
	for ws, k := range owner {
		if k == "" {
			continue
		}
		words := strings.Fields(ws)
		byWord[words[0]] = append(byWord[words[0]], phrase{words, tmpl[k]})
	}
	out := ms
	for _, s := range tr.Segments {
		words := make([]string, len(s.Words))
		for i, w := range s.Words {
			words[i] = norm(w.Text)
		}
		for i, w := range words {
		candidates:
			for _, p := range byWord[w] {
				if i+len(p.words) > len(words) {
					continue
				}
				for j, pw := range p.words {
					if words[i+j] != pw {
						continue candidates
					}
				}
				at := s.Words[i].Start
				if at <= first[p.m.Key()] {
					continue
				}
				m := p.m
				m.At = at
				m.Spoken = strings.Join(p.words, " ")
				out = append(out, m)
			}
		}
	}
	return out
}

// phraseWords normalises a name or alias for matching, dropping leading
// articles and anything too short to be distinctive on its own.
func phraseWords(s string) []string {
	var ws []string
	for _, f := range strings.Fields(s) {
		if w := norm(f); w != "" {
			ws = append(ws, w)
		}
	}
	for len(ws) > 0 && (ws[0] == "the" || ws[0] == "a" || ws[0] == "an") {
		ws = ws[1:]
	}
	if len(ws) == 1 && len([]rune(ws[0])) < 3 {
		return nil
	}
	return ws
}

// tidy sorts mentions by time and drops an entity that repeats within
// gap seconds of its previous mention.
func tidy(ms []Mention, gap float64) []Mention {
	sort.SliceStable(ms, func(i, j int) bool { return ms[i].At < ms[j].At })
	last := map[string]float64{}
	out := ms[:0]
	for _, m := range ms {
		if m.Name == "" {
			continue
		}
		if t, ok := last[m.Key()]; ok && m.At-t < gap {
			continue
		}
		last[m.Key()] = m.At
		out = append(out, m)
	}
	return out
}
