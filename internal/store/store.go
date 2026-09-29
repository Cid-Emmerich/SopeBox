// Package store keeps the subscription library on disk: podcasts, their
// episodes, listening positions, downloads, the play queue and the voice
// profiles learned for each show.
package store

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Cid-Emmerich/SopeBox/internal/feed"
)

// VoiceProfile is a remembered speaker for a podcast so the same host gets
// the same orb (and name) in every episode.
type VoiceProfile struct {
	Name     string    `json:"name"`
	Centroid []float64 `json:"centroid"`
	F0       float64   `json:"f0"`
	Talk     float64   `json:"talk"` // seconds heard in total
}

// Podcast is a subscription plus everything learned about it.
type Podcast struct {
	feed.Feed
	Added        time.Time      `json:"added"`
	LastRefresh  time.Time      `json:"last_refresh"`
	AutoDownload int            `json:"auto_download"` // -1 = use the global setting
	IconPath     string         `json:"icon_path,omitempty"`
	IconSource   string         `json:"icon_source,omitempty"`
	Voices       []VoiceProfile `json:"voices,omitempty"`
	Collage      bool           `json:"collage,omitempty"` // prepare collages for new episodes
	Error        string         `json:"-"`
}

// State is what SopeBox remembers about one episode.
type State struct {
	Position   float64   `json:"position,omitempty"`
	Played     bool      `json:"played,omitempty"`
	Path       string    `json:"path,omitempty"` // downloaded file
	Downloaded time.Time `json:"downloaded,omitempty"`
	Transcript string    `json:"transcript,omitempty"` // cached transcript json
	Auto       bool      `json:"auto,omitempty"`       // downloaded by the auto policy
}

// Key identifies an episode across the library.
type Key struct {
	Feed string `json:"feed"`
	GUID string `json:"guid"`
}

// Library is the whole on-disk state.
type Library struct {
	mu       sync.RWMutex
	path     string
	Podcasts []*Podcast        `json:"podcasts"`
	States   map[string]*State `json:"states"`
	Queue    []Key             `json:"queue"`
	Last     *Key              `json:"last,omitempty"` // episode playing when SopeBox quit
	dirty    bool
}

func stateKey(k Key) string { return k.Feed + "\x00" + k.GUID }

// Open loads the library from path (an empty library when missing).
func Open(path string) (*Library, error) {
	l := &Library{path: path, States: map[string]*State{}}
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return l, nil
		}
		return l, err
	}
	if err := json.Unmarshal(raw, l); err != nil {
		return l, err
	}
	if l.States == nil {
		l.States = map[string]*State{}
	}
	for _, p := range l.Podcasts {
		if p.AutoDownload == 0 && p.Added.IsZero() {
			p.AutoDownload = -1
		}
	}
	return l, nil
}

// Save writes the library atomically.
func (l *Library) Save() error {
	l.mu.RLock()
	raw, err := json.MarshalIndent(l, "", " ")
	l.mu.RUnlock()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(l.path), 0o755); err != nil {
		return err
	}
	tmp := l.path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, l.path)
}

// ---------------------------------------------------------------------------
// Podcasts

// Find returns the podcast with the given feed URL.
func (l *Library) Find(url string) *Podcast {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.find(url)
}

func (l *Library) find(url string) *Podcast {
	for _, p := range l.Podcasts {
		if p.URL == url {
			return p
		}
	}
	return nil
}

// Subscribe fetches a feed and adds it (or refreshes it if present).
func (l *Library) Subscribe(url string) (*Podcast, error) {
	url = strings.TrimSpace(url)
	if url == "" {
		return nil, errors.New("empty feed url")
	}
	f, err := feed.Fetch(url)
	if err != nil {
		return nil, err
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	p := l.find(url)
	if p == nil {
		p = &Podcast{Added: time.Now(), AutoDownload: -1}
		l.Podcasts = append(l.Podcasts, p)
	}
	l.merge(p, f)
	return p, nil
}

// AddPlaceholder adds a subscription without fetching it (used by imports);
// Refresh fills it in later.
func (l *Library) AddPlaceholder(url, title string) *Podcast {
	l.mu.Lock()
	defer l.mu.Unlock()
	if p := l.find(url); p != nil {
		return p
	}
	p := &Podcast{Added: time.Now(), AutoDownload: -1}
	p.URL, p.Title = url, title
	l.Podcasts = append(l.Podcasts, p)
	return p
}

// Refresh re-fetches one podcast. New episodes are returned.
func (l *Library) Refresh(p *Podcast) ([]feed.Episode, error) {
	f, err := feed.Fetch(p.URL)
	l.mu.Lock()
	defer l.mu.Unlock()
	if err != nil {
		p.Error = err.Error()
		return nil, err
	}
	p.Error = ""
	before := map[string]bool{}
	for _, e := range p.Episodes {
		before[e.GUID] = true
	}
	l.merge(p, f)
	var fresh []feed.Episode
	for _, e := range p.Episodes {
		if !before[e.GUID] {
			fresh = append(fresh, e)
		}
	}
	return fresh, nil
}

// merge copies feed data into p, keeping episodes the feed no longer lists.
func (l *Library) merge(p *Podcast, f *feed.Feed) {
	old := p.Episodes
	byGUID := map[string]int{}
	for i, e := range f.Episodes {
		byGUID[e.GUID] = i
	}
	p.Feed = *f
	for _, e := range old {
		if _, ok := byGUID[e.GUID]; !ok {
			p.Episodes = append(p.Episodes, e)
		}
	}
	// Show notes can be enormous; keep enough to read in the details pane
	// and bound the library file's size (thousands of episodes per feed).
	for i := range p.Episodes {
		if r := []rune(p.Episodes[i].Description); len(r) > 1200 {
			p.Episodes[i].Description = string(r[:1200]) + "…"
		}
	}
	sort.SliceStable(p.Episodes, func(i, j int) bool {
		return p.Episodes[i].Published.After(p.Episodes[j].Published)
	})
	p.LastRefresh = time.Now()
}

// Unsubscribe removes a podcast and its episode state.
func (l *Library) Unsubscribe(url string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for i, p := range l.Podcasts {
		if p.URL == url {
			l.Podcasts = append(l.Podcasts[:i], l.Podcasts[i+1:]...)
			break
		}
	}
	for k := range l.States {
		if strings.HasPrefix(k, url+"\x00") {
			delete(l.States, k)
		}
	}
	q := l.Queue[:0]
	for _, k := range l.Queue {
		if k.Feed != url {
			q = append(q, k)
		}
	}
	l.Queue = q
}

// Sorted returns podcasts alphabetically.
func (l *Library) Sorted() []*Podcast {
	l.mu.RLock()
	defer l.mu.RUnlock()
	out := append([]*Podcast(nil), l.Podcasts...)
	sort.SliceStable(out, func(i, j int) bool {
		return strings.ToLower(out[i].Title) < strings.ToLower(out[j].Title)
	})
	return out
}

// ---------------------------------------------------------------------------
// Episodes

// Item is an episode together with its podcast and state.
type Item struct {
	Podcast *Podcast
	Episode *feed.Episode
	State   *State
}

// Key returns the item's library key.
func (it Item) Key() Key { return Key{Feed: it.Podcast.URL, GUID: it.Episode.GUID} }

// Downloaded reports whether the file is on disk.
func (it Item) Downloaded() bool {
	if it.State == nil || it.State.Path == "" {
		return false
	}
	_, err := os.Stat(it.State.Path)
	return err == nil
}

// Source is the local path when downloaded, else the stream URL.
func (it Item) Source() string {
	if it.Downloaded() {
		return it.State.Path
	}
	return it.Episode.URL
}

// Get resolves a key to an item (nil when unknown).
func (l *Library) Get(k Key) *Item {
	l.mu.RLock()
	defer l.mu.RUnlock()
	p := l.find(k.Feed)
	if p == nil {
		return nil
	}
	for i := range p.Episodes {
		if p.Episodes[i].GUID == k.GUID {
			return &Item{Podcast: p, Episode: &p.Episodes[i], State: l.States[stateKey(k)]}
		}
	}
	return nil
}

// State returns (creating if needed) the state record for a key.
func (l *Library) State(k Key) *State {
	l.mu.Lock()
	defer l.mu.Unlock()
	s := l.States[stateKey(k)]
	if s == nil {
		s = &State{}
		l.States[stateKey(k)] = s
	}
	return s
}

// PeekState returns the state record or nil without creating one.
func (l *Library) PeekState(k Key) *State {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.States[stateKey(k)]
}

// Items lists a podcast's episodes newest first.
func (l *Library) Items(p *Podcast) []Item {
	l.mu.RLock()
	defer l.mu.RUnlock()
	out := make([]Item, 0, len(p.Episodes))
	for i := range p.Episodes {
		e := &p.Episodes[i]
		out = append(out, Item{Podcast: p, Episode: e, State: l.States[stateKey(Key{p.URL, e.GUID})]})
	}
	return out
}

// AllItems lists every episode across podcasts, newest first.
func (l *Library) AllItems() []Item {
	l.mu.RLock()
	defer l.mu.RUnlock()
	var out []Item
	for _, p := range l.Podcasts {
		for i := range p.Episodes {
			e := &p.Episodes[i]
			out = append(out, Item{Podcast: p, Episode: e, State: l.States[stateKey(Key{p.URL, e.GUID})]})
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Episode.Published.After(out[j].Episode.Published) })
	return out
}

// DownloadedItems lists episodes that are on disk, newest first.
func (l *Library) DownloadedItems() []Item {
	var out []Item
	for _, it := range l.AllItems() {
		if it.Downloaded() {
			out = append(out, it)
		}
	}
	return out
}

// Search finds episodes whose title, podcast or description match every
// word of q. Results are newest first.
func (l *Library) Search(q string) []Item {
	words := strings.Fields(strings.ToLower(q))
	if len(words) == 0 {
		return nil
	}
	var out []Item
	for _, it := range l.AllItems() {
		hay := strings.ToLower(it.Episode.Title + " " + it.Podcast.Title + " " + it.Episode.Description)
		ok := true
		for _, w := range words {
			if !strings.Contains(hay, w) {
				ok = false
				break
			}
		}
		if ok {
			out = append(out, it)
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// Queue

// QueueItems resolves the queue to items, dropping stale keys.
func (l *Library) QueueItems() []Item {
	l.mu.RLock()
	keys := append([]Key(nil), l.Queue...)
	l.mu.RUnlock()
	var out []Item
	for _, k := range keys {
		if it := l.Get(k); it != nil {
			out = append(out, *it)
		}
	}
	return out
}

// Enqueue appends a key (no duplicates).
func (l *Library) Enqueue(k Key) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, q := range l.Queue {
		if q == k {
			return
		}
	}
	l.Queue = append(l.Queue, k)
}

// EnqueueNext inserts a key at position pos.
func (l *Library) EnqueueAt(k Key, pos int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	q := l.Queue[:0]
	for _, x := range l.Queue {
		if x != k {
			q = append(q, x)
		}
	}
	if pos < 0 {
		pos = 0
	}
	if pos > len(q) {
		pos = len(q)
	}
	q = append(q[:pos], append([]Key{k}, q[pos:]...)...)
	l.Queue = q
}

// Dequeue removes a key.
func (l *Library) Dequeue(k Key) {
	l.mu.Lock()
	defer l.mu.Unlock()
	q := l.Queue[:0]
	for _, x := range l.Queue {
		if x != k {
			q = append(q, x)
		}
	}
	l.Queue = q
}

// SetQueue replaces the queue.
func (l *Library) SetQueue(keys []Key) {
	l.mu.Lock()
	l.Queue = append([]Key(nil), keys...)
	l.mu.Unlock()
}

// ---------------------------------------------------------------------------
// Voice profiles

// SaveVoices stores learned speakers for a podcast, merging by name.
func (l *Library) SaveVoices(p *Podcast, profiles []VoiceProfile) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, np := range profiles {
		if np.Talk < 15 || len(np.Centroid) == 0 {
			continue
		}
		merged := false
		for i, op := range p.Voices {
			if strings.EqualFold(op.Name, np.Name) {
				// weighted average keeps a long-known host stable
				w := op.Talk / (op.Talk + np.Talk)
				for j := range op.Centroid {
					if j < len(np.Centroid) {
						p.Voices[i].Centroid[j] = op.Centroid[j]*w + np.Centroid[j]*(1-w)
					}
				}
				p.Voices[i].F0 = op.F0*w + np.F0*(1-w)
				p.Voices[i].Talk += np.Talk
				merged = true
				break
			}
		}
		if !merged {
			p.Voices = append(p.Voices, np)
		}
	}
	if len(p.Voices) > 12 {
		sort.Slice(p.Voices, func(i, j int) bool { return p.Voices[i].Talk > p.Voices[j].Talk })
		p.Voices = p.Voices[:12]
	}
}

// RenameVoice renames a stored profile.
func (l *Library) RenameVoice(p *Podcast, from, to string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for i := range p.Voices {
		if p.Voices[i].Name == from {
			p.Voices[i].Name = to
		}
	}
}

// ForgetVoices clears learned speakers for a podcast.
func (l *Library) ForgetVoices(p *Podcast) {
	l.mu.Lock()
	p.Voices = nil
	l.mu.Unlock()
}
