// Package download runs the episode download queue and the automatic
// "keep the N newest episodes" policy.
package download

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Cid-Emmerich/SopeBox/internal/art"
	"github.com/Cid-Emmerich/SopeBox/internal/store"
)

// JobState is where a download is in its life.
type JobState int

const (
	Queued JobState = iota
	Running
	Done
	Failed
	Cancelled
)

func (s JobState) String() string {
	return [...]string{"queued", "downloading", "done", "failed", "cancelled"}[s]
}

// Job is one episode download.
type Job struct {
	Key     store.Key
	Title   string
	Podcast string
	URL     string
	Path    string
	Total   int64
	Done    int64
	State   JobState
	Err     string
	Auto    bool
	Started time.Time
	cancel  chan struct{}
}

// Progress is 0..1 (or -1 when the size is unknown).
func (j *Job) Progress() float64 {
	if j.Total <= 0 {
		return -1
	}
	return float64(j.Done) / float64(j.Total)
}

// Manager owns the queue.
type Manager struct {
	mu      sync.Mutex
	lib     *store.Library
	dir     string
	conc    int
	jobs    []*Job
	running int
	onEvent func()
	client  *http.Client
}

// New creates a manager. onEvent is called (from any goroutine) whenever a
// job changes state so the UI can redraw.
func New(lib *store.Library, dir string, concurrency int, onEvent func()) *Manager {
	if concurrency < 1 {
		concurrency = 1
	}
	return &Manager{lib: lib, dir: dir, conc: concurrency, onEvent: onEvent,
		client: &http.Client{Timeout: 0}}
}

// SetDir changes the download folder for future downloads.
func (m *Manager) SetDir(d string) { m.mu.Lock(); m.dir = d; m.mu.Unlock() }

// SetConcurrency changes how many downloads run at once.
func (m *Manager) SetConcurrency(n int) {
	m.mu.Lock()
	if n < 1 {
		n = 1
	}
	m.conc = n
	m.mu.Unlock()
	m.pump()
}

func (m *Manager) fire() {
	if m.onEvent != nil {
		m.onEvent()
	}
}

// Jobs returns a snapshot of the queue, newest first.
func (m *Manager) Jobs() []Job {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]Job, 0, len(m.jobs))
	for i := len(m.jobs) - 1; i >= 0; i-- {
		out = append(out, *m.jobs[i])
	}
	return out
}

// Active counts queued + running jobs.
func (m *Manager) Active() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	n := 0
	for _, j := range m.jobs {
		if j.State == Queued || j.State == Running {
			n++
		}
	}
	return n
}

// JobFor returns the job for a key (nil if none).
func (m *Manager) JobFor(k store.Key) *Job {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := len(m.jobs) - 1; i >= 0; i-- {
		if m.jobs[i].Key == k {
			j := *m.jobs[i]
			return &j
		}
	}
	return nil
}

// Enqueue schedules an episode for download. Already downloaded or already
// queued episodes are ignored.
func (m *Manager) Enqueue(it store.Item, auto bool) error {
	if it.Downloaded() {
		return errors.New("already downloaded")
	}
	if it.Episode.URL == "" {
		return errors.New("episode has no audio url")
	}
	m.mu.Lock()
	for _, j := range m.jobs {
		if j.Key == it.Key() && (j.State == Queued || j.State == Running) {
			m.mu.Unlock()
			return errors.New("already queued")
		}
	}
	j := &Job{Key: it.Key(), Title: it.Episode.Title, Podcast: it.Podcast.Title, URL: it.Episode.URL,
		Path: m.pathFor(it), Total: it.Episode.Bytes, Auto: auto, cancel: make(chan struct{})}
	m.jobs = append(m.jobs, j)
	if len(m.jobs) > 200 {
		m.jobs = m.jobs[len(m.jobs)-200:]
	}
	m.mu.Unlock()
	m.fire()
	m.pump()
	return nil
}

// Cancel stops a queued or running job.
func (m *Manager) Cancel(k store.Key) {
	m.mu.Lock()
	for _, j := range m.jobs {
		if j.Key == k && (j.State == Queued || j.State == Running) {
			if j.State == Queued {
				j.State = Cancelled
			}
			select {
			case <-j.cancel:
			default:
				close(j.cancel)
			}
		}
	}
	m.mu.Unlock()
	m.fire()
}

// Clear drops finished, failed and cancelled jobs from the list.
func (m *Manager) Clear() {
	m.mu.Lock()
	keep := m.jobs[:0]
	for _, j := range m.jobs {
		if j.State == Queued || j.State == Running {
			keep = append(keep, j)
		}
	}
	m.jobs = keep
	m.mu.Unlock()
	m.fire()
}

func (m *Manager) pathFor(it store.Item) string {
	ext := extFor(it.Episode.URL, it.Episode.Type)
	date := ""
	if !it.Episode.Published.IsZero() {
		date = it.Episode.Published.Format("2006-01-02") + " - "
	}
	name := date + sanitize(it.Episode.Title)
	if len(name) > 120 {
		name = name[:120]
	}
	return filepath.Join(m.dir, sanitize(it.Podcast.Title), name+ext)
}

func extFor(u, typ string) string {
	if i := strings.Index(u, "?"); i >= 0 {
		u = u[:i]
	}
	ext := strings.ToLower(filepath.Ext(u))
	switch ext {
	case ".mp3", ".m4a", ".aac", ".ogg", ".opus", ".wav", ".flac", ".mp4", ".m4b":
		return ext
	}
	switch {
	case strings.Contains(typ, "mp4"), strings.Contains(typ, "m4a"), strings.Contains(typ, "aac"):
		return ".m4a"
	case strings.Contains(typ, "ogg"), strings.Contains(typ, "opus"):
		return ".ogg"
	}
	return ".mp3"
}

func sanitize(s string) string {
	s = strings.Map(func(r rune) rune {
		switch r {
		case '/', '\\', ':', '*', '?', '"', '<', '>', '|', '\x00':
			return '-'
		}
		return r
	}, s)
	s = strings.Join(strings.Fields(s), " ")
	s = strings.Trim(s, ". ")
	if s == "" {
		s = "untitled"
	}
	return s
}

// pump starts queued jobs while there is capacity.
func (m *Manager) pump() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, j := range m.jobs {
		if m.running >= m.conc {
			return
		}
		if j.State == Queued {
			j.State = Running
			j.Started = time.Now()
			m.running++
			go m.run(j)
		}
	}
}

func (m *Manager) run(j *Job) {
	err := m.fetch(j)
	m.mu.Lock()
	m.running--
	switch {
	case err == nil:
		j.State = Done
		j.Done = j.Total
		st := m.lib.State(j.Key)
		st.Path = j.Path
		st.Downloaded = time.Now()
		st.Auto = j.Auto
	case errors.Is(err, errCancelled):
		j.State = Cancelled
		os.Remove(j.Path + ".part")
	default:
		j.State = Failed
		j.Err = err.Error()
		os.Remove(j.Path + ".part")
	}
	m.mu.Unlock()
	m.fire()
	_ = m.lib.Save()
	m.pump()
}

var errCancelled = errors.New("cancelled")

func (m *Manager) fetch(j *Job) error {
	if err := os.MkdirAll(filepath.Dir(j.Path), 0o755); err != nil {
		return err
	}
	req, err := http.NewRequest("GET", j.URL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", art.UserAgent)
	resp, err := m.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	if resp.ContentLength > 0 {
		m.mu.Lock()
		j.Total = resp.ContentLength
		m.mu.Unlock()
	}
	tmp := j.Path + ".part"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	buf := make([]byte, 256<<10)
	last := time.Now()
	for {
		select {
		case <-j.cancel:
			f.Close()
			return errCancelled
		default:
		}
		n, rerr := resp.Body.Read(buf)
		if n > 0 {
			if _, werr := f.Write(buf[:n]); werr != nil {
				f.Close()
				return werr
			}
			m.mu.Lock()
			j.Done += int64(n)
			m.mu.Unlock()
			if time.Since(last) > 250*time.Millisecond {
				last = time.Now()
				m.fire()
			}
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			f.Close()
			return rerr
		}
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, j.Path)
}

// ---------------------------------------------------------------------------
// Automatic downloads

// Policy describes the automatic download settings.
type Policy struct {
	Latest     int // global "download the N most recent" (0 = off)
	KeepLatest int // delete auto-downloads older than the N newest (0 = keep)
}

// Choices lists the auto-download counts in cycling order.
var Choices = []int{0, 1, 3, 5, 10}

// Apply enqueues the newest episodes of every podcast according to the
// policy and removes old automatic downloads. Returns how many were queued.
func (m *Manager) Apply(pol Policy) (queued, removed int) {
	for _, p := range m.lib.Sorted() {
		n := p.AutoDownload
		if n < 0 {
			n = pol.Latest
		}
		items := m.lib.Items(p)
		for i, it := range items {
			if i >= n {
				break
			}
			if !it.Downloaded() {
				if err := m.Enqueue(it, true); err == nil {
					queued++
				}
			}
		}
		if pol.KeepLatest > 0 {
			var auto []store.Item
			for _, it := range items {
				if it.Downloaded() && it.State != nil && it.State.Auto {
					auto = append(auto, it)
				}
			}
			sort.SliceStable(auto, func(i, j int) bool { return auto[i].Episode.Published.After(auto[j].Episode.Published) })
			for i := pol.KeepLatest; i < len(auto); i++ {
				if m.Remove(auto[i]) == nil {
					removed++
				}
			}
		}
	}
	return
}

// Remove deletes a downloaded file and clears its state.
func (m *Manager) Remove(it store.Item) error {
	if it.State == nil || it.State.Path == "" {
		return errors.New("not downloaded")
	}
	path := it.State.Path
	err := os.Remove(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	it.State.Path = ""
	it.State.Auto = false
	it.State.Downloaded = time.Time{}
	// prune empty podcast folders (fails harmlessly when not empty)
	os.Remove(filepath.Dir(path))
	return nil
}
