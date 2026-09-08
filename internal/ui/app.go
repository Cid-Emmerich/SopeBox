// Package ui is the terminal interface: views, key handling and drawing.
package ui

import (
	"fmt"
	"math"
	"os"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"

	"github.com/Cid-Emmerich/SopeBox/internal/art"
	"github.com/Cid-Emmerich/SopeBox/internal/audio"
	"github.com/Cid-Emmerich/SopeBox/internal/captions"
	"github.com/Cid-Emmerich/SopeBox/internal/config"
	"github.com/Cid-Emmerich/SopeBox/internal/download"
	"github.com/Cid-Emmerich/SopeBox/internal/paint"
	"github.com/Cid-Emmerich/SopeBox/internal/store"
	"github.com/Cid-Emmerich/SopeBox/internal/theme"
	"github.com/Cid-Emmerich/SopeBox/internal/transcript"
	"github.com/Cid-Emmerich/SopeBox/internal/vis"
	"github.com/Cid-Emmerich/SopeBox/internal/voices"
)

// View identifies a screen.
type View int

const (
	ViewNow View = iota
	ViewPodcasts
	ViewQueue
	ViewDownloads
	ViewSearch
	ViewSettings
	numViews
)

var tabNames = []string{"Now Playing", "Podcasts", "Queue", "Downloads", "Search", "Settings"}

// IconModes lists the icon rendering modes in cycling order.
var IconModes = []string{"blocks", "ascii", "kitty"}

// events posted from background goroutines
type toastEvent struct {
	msg   string
	isErr bool
}

type iconEvent struct {
	key  string
	icon *art.Icon
	note string
}

type transcriptEvent struct {
	key store.Key
	tr  *transcript.Transcript
	ch  []transcript.Chapter
	err error
}

type progressEvent struct {
	phase string
	frac  float64
	key   store.Key
}

type refreshEvent struct {
	fresh int
	errs  int
	done  bool
}

type searchEvent struct {
	results []art.Result
	err     error
}

type redrawEvent struct{}

// prompt is a one-line text input overlay.
type prompt struct {
	active bool
	label  string
	text   string
	hint   string
	onDone func(string)
}

// App is the whole interactive player.
type App struct {
	scr tcell.Screen
	cfg *config.Config
	lib *store.Library
	pl  *audio.Player
	dl  *download.Manager

	view       View
	help       bool
	helpScroll int
	quit       bool
	prompt     prompt

	// theme
	themeNames []string
	themeIdx   int
	th         theme.Theme
	matchTh    theme.Theme
	haveMatch  bool

	// visualizer & voices
	visIdx    int
	visOpts   vis.Options
	visStart  time.Time
	lastTick  time.Time
	lastPrune time.Time
	canvas    *vis.Canvas
	tracker   *voices.Tracker
	level     float64
	words     []vis.FlyWord
	lastSeg   int
	lastWord  int

	// captions
	emoji      *captions.Emoji
	tr         *transcript.Transcript
	chapters   []transcript.Chapter
	trStatus   string // "loading", "none", "ok", "transcribing"
	trProgress string
	trBusy     map[store.Key]bool

	// playback
	cur        *store.Item
	lastSave   time.Time
	lastPosSav time.Time
	posSaved   float64

	// icons
	icons      map[string]*art.Icon
	iconBusy   map[string]bool
	cellCache  map[string][][]paint.Cell
	kittyID    int
	kittyDrawn bool

	// views
	pv        podcastsView
	qCursor   int
	qScroll   int
	dCursor   int
	dScroll   int
	sv        searchView
	setCur    int
	setScroll int

	// refresh
	refreshing bool
	busy       string

	// toast
	toast     string
	toastErr  bool
	toastTill time.Time

	mouseSeekRow int
	mouseSeekX0  int
	mouseSeekX1  int
}

// New builds the app. The player must already be started.
func New(cfg *config.Config, lib *store.Library, pl *audio.Player) *App {
	a := &App{
		cfg:        cfg,
		lib:        lib,
		pl:         pl,
		themeNames: theme.Names(),
		visOpts:    vis.OptionsFromConfig(*cfg),
		visIdx:     vis.Index(cfg.Vis),
		visStart:   time.Now(),
		lastTick:   time.Now(),
		emoji:      captions.NewEmoji(),
		icons:      map[string]*art.Icon{},
		iconBusy:   map[string]bool{},
		cellCache:  map[string][][]paint.Cell{},
		trBusy:     map[store.Key]bool{},
		kittyID:    int(time.Now().UnixNano()%9000) + 100,
		lastSeg:    -1,
		lastWord:   -1,
	}
	a.tracker = voices.New(voices.Options{Mode: cfg.VoiceMode, Sensitivity: cfg.VoiceSensitivity,
		Smoothing: cfg.VoiceSmoothing, Falloff: cfg.VoiceFalloff, MaxOrbs: cfg.VoiceMax})
	a.setTheme(cfg.Theme)
	if cfg.IconMode == "kitty" && !art.KittySupported() {
		cfg.IconMode = "blocks"
	}
	a.pv.init(a)
	a.sv.init()
	return a
}

func (a *App) setTheme(name string) {
	a.themeIdx = 0
	for i, n := range a.themeNames {
		if n == name {
			a.themeIdx = i
		}
	}
	a.applyTheme()
}

func (a *App) applyTheme() {
	name := a.themeNames[a.themeIdx]
	if name == "match" {
		if a.haveMatch {
			a.th = a.matchTh
		} else {
			a.th = theme.Get("sopebox")
			a.th.Name = "match"
		}
		return
	}
	a.th = theme.Get(name)
}

// post sends an event to the UI loop from any goroutine.
func (a *App) post(v interface{}) {
	if a.scr != nil {
		a.scr.PostEvent(tcell.NewEventInterrupt(v))
	}
}

// Run starts the event loop and blocks until quit.
func (a *App) Run(startView View) error {
	scr, err := tcell.NewScreen()
	if err != nil {
		return err
	}
	if err := scr.Init(); err != nil {
		return err
	}
	return a.RunWith(scr, startView)
}

// RunWith runs the event loop on an already-initialised screen.
func (a *App) RunWith(scr tcell.Screen, startView View) error {
	a.scr = scr
	scr.EnableMouse(tcell.MouseButtonEvents)
	scr.HideCursor()
	scr.Clear()
	a.view = startView

	a.dl = download.New(a.lib, a.cfg.DownloadDir, a.cfg.Concurrency, func() { a.post(redrawEvent{}) })

	events := make(chan tcell.Event, 64)
	go func() {
		for {
			ev := scr.PollEvent()
			if ev == nil {
				return
			}
			events <- ev
		}
	}()

	a.pl.OnChange(func() { a.post(redrawEvent{}) })
	a.pl.OnEnd(func(t *audio.Track) { a.post(t) })

	if a.cfg.RefreshStart && len(a.lib.Podcasts) > 0 {
		a.refreshAll()
	}
	if a.cur == nil && a.lib.Last != nil {
		if it := a.lib.Get(*a.lib.Last); it != nil {
			a.load(*it, true)
		}
	}

	fps := a.cfg.VisFPS
	if fps < 5 {
		fps = 5
	}
	if fps > 60 {
		fps = 60
	}
	ticker := time.NewTicker(time.Second / time.Duration(fps))
	defer ticker.Stop()
	saver := time.NewTicker(30 * time.Second)
	defer saver.Stop()

	a.draw()
	for !a.quit {
		select {
		case ev := <-events:
			a.handle(ev)
			for len(events) > 0 && !a.quit {
				a.handle(<-events)
			}
		case <-ticker.C:
			a.tick()
		case <-saver.C:
			a.savePosition()
			go a.lib.Save() // marshalling a big library off the UI thread
		}
		if a.quit {
			break
		}
		a.draw()
	}
	a.kittyClear()
	scr.Fini()
	a.shutdown()
	return nil
}

func (a *App) shutdown() {
	a.savePosition()
	a.saveVoices()
	if a.cur != nil {
		k := a.cur.Key()
		a.lib.Last = &k
	}
	if err := a.lib.Save(); err != nil {
		fmt.Fprintln(os.Stderr, "sopebox: could not save library:", err)
	}
	a.saveConfig()
}

func (a *App) saveConfig() {
	c := a.cfg
	st := a.pl.Status()
	c.Theme = a.themeNames[a.themeIdx]
	c.Vis = vis.Registry[a.visIdx].Name()
	a.visOpts.ApplyTo(c)
	c.Volume = st.Volume
	c.Speed = st.Speed
	c.VoiceMode = a.tracker.Opts.Mode
	c.VoiceSensitivity = a.tracker.Opts.Sensitivity
	c.VoiceSmoothing = a.tracker.Opts.Smoothing
	c.VoiceFalloff = a.tracker.Opts.Falloff
	c.VoiceMax = a.tracker.Opts.MaxOrbs
	if err := c.Save(); err != nil {
		fmt.Fprintln(os.Stderr, "sopebox: could not save config:", err)
	}
}

// handle dispatches one tcell event.
func (a *App) handle(ev tcell.Event) {
	switch e := ev.(type) {
	case *tcell.EventResize:
		a.scr.Sync()
		a.kittyClear()
		a.cellCache = map[string][][]paint.Cell{}
	case *tcell.EventKey:
		a.handleKey(e)
	case *tcell.EventMouse:
		a.handleMouse(e)
	case *tcell.EventInterrupt:
		switch d := e.Data().(type) {
		case redrawEvent:
		case playRequest:
			a.play(d.item)
		case *audio.Track:
			a.onTrackEnd(d)
		case toastEvent:
			a.busy = ""
			a.showToast(d.msg, d.isErr)
		case iconEvent:
			a.onIcon(d)
		case transcriptEvent:
			a.onTranscript(d)
		case progressEvent:
			if a.cur != nil && d.key == a.cur.Key() {
				a.trProgress = fmt.Sprintf("%s %d%%", d.phase, int(d.frac*100))
			}
		case refreshEvent:
			if d.done {
				a.refreshing = false
				msg := fmt.Sprintf("refreshed: %d new episodes", d.fresh)
				if d.errs > 0 {
					msg += fmt.Sprintf(", %d feeds failed", d.errs)
				}
				a.showToast(msg, d.errs > 0 && d.fresh == 0)
				a.pv.rebuild(a)
				if a.cfg.AutoDownload > 0 || a.anyFeedAuto() {
					q, r := a.dl.Apply(download.Policy{Latest: a.cfg.AutoDownload, KeepLatest: a.cfg.KeepLatest})
					if q > 0 || r > 0 {
						a.showToast(fmt.Sprintf("auto-download: %d queued, %d removed", q, r), false)
					}
				}
				_ = a.lib.Save()
			}
		case searchEvent:
			a.sv.busy = false
			if d.err != nil {
				a.showToast("search: "+d.err.Error(), true)
			} else {
				a.sv.results = d.results
				a.sv.cursor = 0
			}
		}
	}
}

func (a *App) anyFeedAuto() bool {
	for _, p := range a.lib.Podcasts {
		if p.AutoDownload > 0 {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// Per-frame update: voices, words, emoji

func (a *App) tick() {
	now := time.Now()
	dt := now.Sub(a.lastTick).Seconds()
	a.lastTick = now
	if dt > 0.2 {
		dt = 0.2
	}
	st := a.pl.Status()
	playing := st.Playing && !st.Loading
	feat := a.pl.Analyzer.Voice()
	spec := a.pl.Analyzer.Spectrum(32, 0, true, a.visOpts.Gain)
	a.tracker.Update(feat, spec, st.Position, dt, playing)
	lv := math.Min(1, feat.Level*5)
	if lv > a.level {
		a.level = a.level*a.visOpts.Smoothing + lv*(1-a.visOpts.Smoothing)
	} else {
		a.level = math.Max(lv, a.level-a.visOpts.Falloff)
	}
	if now.Sub(a.lastPrune) > 10*time.Second {
		a.lastPrune = now
		a.tracker.Prune()
	}
	// words & emoji
	if a.tr != nil && playing {
		seg, in := a.tr.At(st.Position)
		if seg >= 0 && in {
			w := a.tr.WordAt(seg, st.Position)
			if seg != a.lastSeg || w != a.lastWord {
				if w >= 0 && (seg != a.lastSeg || w > a.lastWord) {
					a.onWord(seg, w, feat.Level)
				}
				a.lastSeg, a.lastWord = seg, w
			}
		}
	}
	// age flying words
	keep := a.words[:0]
	for _, w := range a.words {
		w.Age += dt
		if w.Age < 4 {
			keep = append(keep, w)
		}
	}
	a.words = keep
	a.emoji.Step(dt)
	// periodic position bookkeeping
	if st.Track != nil && now.Sub(a.lastPosSav) > time.Second {
		a.lastPosSav = now
		a.savePosition()
	}
}

func (a *App) onWord(seg, w int, level float64) {
	s := a.tr.Segments[seg]
	if w >= len(s.Words) {
		return
	}
	word := s.Words[w].Text
	orbID := 0
	if o := a.tracker.Current(); o != nil {
		orbID = o.ID
	}
	loud := math.Min(1, level*6)
	if strings.HasSuffix(word, "!") || (len(word) > 2 && strings.ToUpper(word) == word && strings.ContainsAny(word, "ABCDEFGHIJKLMNOPQRSTUVWXYZ")) {
		loud = 1
	}
	a.words = append(a.words, vis.FlyWord{Text: word, Orb: orbID, Loud: loud, Seed: float64((seg*31+w*17)%97) / 97})
	if len(a.words) > 40 {
		a.words = a.words[len(a.words)-40:]
	}
	if a.cfg.Emoji {
		if idx := captions.Trigger(word); idx >= 0 {
			a.emoji.Fire(idx, strings.Trim(word, ".,!?;:\"'()[]"))
		}
	}
}

// ---------------------------------------------------------------------------
// Playback orchestration

// play starts an episode from its saved position.
func (a *App) play(it store.Item) { a.load(it, false) }

func (a *App) load(it store.Item, pausedStart bool) {
	a.savePosition()
	a.saveVoices()
	st := a.lib.State(it.Key())
	it.State = st
	a.cur = &it
	a.tracker.Reset()
	a.tracker.Import(it.Podcast.Voices)
	var names []string
	for _, p := range it.Episode.Persons {
		names = append(names, p.Name)
	}
	for _, p := range it.Podcast.Persons {
		names = append(names, p.Name)
	}
	a.tracker.SetPersons(dedupe(names))
	a.words = nil
	a.lastSeg, a.lastWord = -1, -1
	a.emoji.Clear()
	a.tr, a.chapters = nil, nil
	a.trStatus = "loading"
	a.trProgress = ""
	start := st.Position
	if it.Episode.Duration > 0 && start > it.Episode.Duration-20 {
		start = 0
	}
	a.pl.Play(&audio.Track{ID: it.Episode.GUID, Title: it.Episode.Title, Podcast: it.Podcast.Title,
		Source: it.Source(), Duration: it.Episode.Duration, Start: start})
	if pausedStart {
		a.pl.SetPaused(true)
	}
	a.requestIcon(it.Podcast, false)
	a.loadTranscript(it, false)
	if it.Episode.Chapters != "" {
		go func(k store.Key, u string) {
			ch, err := transcript.FetchChapters(u)
			if err == nil {
				a.post(transcriptEvent{key: k, ch: ch})
			}
		}(it.Key(), it.Episode.Chapters)
	}
	k := it.Key()
	a.lib.Last = &k
}

func dedupe(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range in {
		if s == "" || seen[strings.ToLower(s)] {
			continue
		}
		seen[strings.ToLower(s)] = true
		out = append(out, s)
	}
	return out
}

func (a *App) savePosition() {
	if a.cur == nil {
		return
	}
	st := a.pl.Status()
	if st.Track == nil {
		return
	}
	s := a.lib.State(a.cur.Key())
	s.Position = st.Position
	if st.Duration > 0 && st.Position > st.Duration-15 {
		s.Played = true
	}
}

func (a *App) saveVoices() {
	if a.cur == nil {
		return
	}
	if profiles := a.tracker.Export(); len(profiles) > 0 {
		a.lib.SaveVoices(a.cur.Podcast, profiles)
	}
}

func (a *App) onTrackEnd(t *audio.Track) {
	if a.cur == nil || a.cur.Episode.GUID != t.ID {
		return
	}
	s := a.lib.State(a.cur.Key())
	s.Played = true
	s.Position = 0
	a.saveVoices()
	if !a.cfg.Continue {
		return
	}
	if next := a.nextInQueue(); next != nil {
		a.lib.Dequeue(a.cur.Key())
		a.play(*next)
	} else {
		a.lib.Dequeue(a.cur.Key())
		a.showToast("end of queue", false)
	}
}

// nextInQueue finds the episode after the current one in the queue, or
// the head of the queue when the current episode is not in it.
func (a *App) nextInQueue() *store.Item {
	q := a.lib.QueueItems()
	if len(q) == 0 {
		return nil
	}
	if a.cur != nil {
		for i, it := range q {
			if it.Key() == a.cur.Key() {
				if i+1 < len(q) {
					return &q[i+1]
				}
				return nil
			}
		}
	}
	return &q[0]
}

func (a *App) playNext() {
	if next := a.nextInQueue(); next != nil {
		if a.cur != nil {
			a.lib.Dequeue(a.cur.Key())
		}
		a.play(*next)
		return
	}
	a.showToast("queue is empty", false)
}

func (a *App) playPrev() {
	st := a.pl.Status()
	if st.Position > 5 {
		a.pl.SeekTo(0)
		return
	}
	q := a.lib.QueueItems()
	if a.cur != nil {
		for i, it := range q {
			if it.Key() == a.cur.Key() && i > 0 {
				a.play(q[i-1])
				return
			}
		}
	}
	a.pl.SeekTo(0)
}

// ---------------------------------------------------------------------------
// Transcripts

func (a *App) loadTranscript(it store.Item, force bool) {
	k := it.Key()
	if a.trBusy[k] {
		return
	}
	a.trBusy[k] = true
	ep := *it.Episode
	cacheKey := transcript.CachePath(a.cfg.CacheDir, it.Podcast.URL+"|"+ep.GUID)
	go func() {
		if !force {
			if t, err := transcript.LoadCached(cacheKey); err == nil && len(t.Segments) > 0 {
				a.post(transcriptEvent{key: k, tr: t})
				return
			}
		}
		t, err := transcript.FromFeed(&ep)
		if err == nil {
			_ = transcript.SaveCached(cacheKey, t)
		}
		a.post(transcriptEvent{key: k, tr: t, err: err})
	}()
}

func (a *App) onTranscript(e transcriptEvent) {
	if e.ch != nil {
		if a.cur != nil && e.key == a.cur.Key() {
			a.chapters = e.ch
		}
		return
	}
	delete(a.trBusy, e.key)
	if a.cur == nil || e.key != a.cur.Key() {
		return
	}
	if e.tr != nil && len(e.tr.Segments) > 0 {
		a.tr = e.tr
		a.trStatus = "ok"
		a.tracker.SetTranscript(e.tr)
		a.lib.State(e.key).Transcript = e.tr.Source
		a.lastSeg, a.lastWord = -1, -1
		if e.tr.HasSpeakers() {
			a.showToast(fmt.Sprintf("captions loaded (%s) with %d named speakers", e.tr.Source, len(e.tr.Speakers)), false)
		} else {
			a.showToast("captions loaded ("+e.tr.Source+")", false)
		}
		return
	}
	a.trStatus = "none"
	a.trProgress = ""
	if e.err != nil && !strings.Contains(e.err.Error(), "offers no transcript") {
		a.showToast("captions: "+e.err.Error(), true)
	}
	// automatic whisper
	if a.cfg.AutoTranscribe == "always" || (a.cfg.AutoTranscribe == "downloaded" && a.cur.Downloaded()) {
		a.transcribe()
	}
}

// transcribe runs whisper.cpp on the current episode in the background.
func (a *App) transcribe() {
	if a.cur == nil {
		a.showToast("nothing playing", true)
		return
	}
	it := *a.cur
	k := it.Key()
	if a.trBusy[k] {
		a.showToast("already transcribing", false)
		return
	}
	if transcript.FindWhisper(a.cfg.WhisperBin) == "" {
		a.showToast("whisper.cpp not found — brew install whisper-cpp", true)
		return
	}
	if !it.Downloaded() {
		a.showToast("download the episode first (d), then transcribe", true)
		return
	}
	a.trBusy[k] = true
	a.trStatus = "transcribing"
	a.trProgress = "starting"
	cacheKey := transcript.CachePath(a.cfg.CacheDir, it.Podcast.URL+"|"+it.Episode.GUID)
	path := it.State.Path
	go func() {
		t, err := transcript.Transcribe(path, transcript.Options{Bin: a.cfg.WhisperBin, Model: a.cfg.WhisperModel, CacheDir: a.cfg.CacheDir,
			Progress: func(ph string, f float64) { a.post(progressEvent{phase: ph, frac: f, key: k}) }})
		if err == nil {
			_ = transcript.SaveCached(cacheKey, t)
		}
		a.post(transcriptEvent{key: k, tr: t, err: err})
	}()
}

// ---------------------------------------------------------------------------
// Icons

func (a *App) icon(p *store.Podcast) *art.Icon {
	if p == nil {
		return nil
	}
	if ic, ok := a.icons[p.URL]; ok {
		return ic
	}
	a.requestIcon(p, false)
	return nil
}

// requestIcon loads a podcast's icon from cache or from the feed's image.
func (a *App) requestIcon(p *store.Podcast, online bool) {
	if p == nil || a.iconBusy[p.URL] {
		return
	}
	a.iconBusy[p.URL] = true
	a.icons[p.URL] = nil
	key, img, title, author := p.URL, p.Image, p.Title, p.Author
	cache := a.cfg.CacheDir
	go func() {
		if !online {
			if ic := art.LoadCached(cache, key); ic != nil {
				a.post(iconEvent{key: key, icon: ic})
				return
			}
			if img != "" {
				if raw, err := art.Get(img); err == nil {
					if im, err := art.Decode(raw); err == nil {
						path, _ := art.Store(cache, key, raw)
						a.post(iconEvent{key: key, icon: &art.Icon{Image: im, Path: path, Source: "feed"}})
						return
					}
				}
			}
		}
		raw, src, err := art.FindIcon(title, author)
		if err != nil {
			a.post(iconEvent{key: key, note: "no icon found: " + err.Error()})
			return
		}
		im, err := art.Decode(raw)
		if err != nil {
			a.post(iconEvent{key: key, note: "icon image not readable"})
			return
		}
		path, _ := art.Store(cache, key, raw)
		a.post(iconEvent{key: key, icon: &art.Icon{Image: im, Path: path, Source: src}, note: "icon from " + src})
	}()
}

func (a *App) onIcon(e iconEvent) {
	delete(a.iconBusy, e.key)
	if e.icon != nil {
		a.icons[e.key] = e.icon
		for k := range a.cellCache {
			if strings.HasPrefix(k, e.key+"|") {
				delete(a.cellCache, k)
			}
		}
		if p := a.lib.Find(e.key); p != nil {
			p.IconPath = e.icon.Path
			p.IconSource = e.icon.Source
		}
		if a.cur != nil && a.cur.Podcast.URL == e.key {
			pal := art.ExtractPalette(e.icon.Image)
			a.matchTh = art.MatchTheme(pal, theme.Get("sopebox"))
			a.haveMatch = true
			a.applyTheme()
		}
		a.kittyClear()
	} else {
		delete(a.icons, e.key)
	}
	if e.note != "" {
		a.showToast(e.note, e.icon == nil)
	}
}

// iconCells renders an icon at a size, cached.
func (a *App) iconCells(key string, ic *art.Icon, w, h int) [][]paint.Cell {
	ck := fmt.Sprintf("%s|%s|%d|%d", key, a.cfg.IconMode, w, h)
	if c, ok := a.cellCache[ck]; ok {
		return c
	}
	var cells [][]paint.Cell
	if a.cfg.IconMode == "ascii" {
		cells = art.ASCII(ic.Image, w, h, "standard", a.th.Name != "mono")
	} else {
		cells = art.Blocks(ic.Image, w, h)
	}
	a.cellCache[ck] = cells
	return cells
}

func (a *App) kittyClear() {
	if a.kittyDrawn {
		os.Stdout.WriteString(art.KittyDelete(a.kittyID))
		a.kittyDrawn = false
	}
}

// ---------------------------------------------------------------------------
// Feeds

// refreshAll re-fetches every podcast in the background.
func (a *App) refreshAll() {
	if a.refreshing {
		return
	}
	a.refreshing = true
	pods := a.lib.Sorted()
	go func() {
		fresh, errs := 0, 0
		sem := make(chan struct{}, 4)
		done := make(chan [2]int, len(pods))
		for _, p := range pods {
			sem <- struct{}{}
			go func(p *store.Podcast) {
				defer func() { <-sem }()
				n, err := a.lib.Refresh(p)
				r := [2]int{len(n), 0}
				if err != nil {
					r[1] = 1
				}
				done <- r
			}(p)
		}
		for range pods {
			r := <-done
			fresh += r[0]
			errs += r[1]
			a.post(refreshEvent{fresh: fresh, errs: errs})
		}
		a.post(refreshEvent{fresh: fresh, errs: errs, done: true})
	}()
}

func (a *App) refreshOne(p *store.Podcast) {
	if p == nil {
		return
	}
	a.busy = "refreshing " + p.Title
	go func() {
		n, err := a.lib.Refresh(p)
		if err != nil {
			a.post(toastEvent{"refresh failed: " + err.Error(), true})
			return
		}
		_ = a.lib.Save()
		a.post(toastEvent{fmt.Sprintf("%s: %d new episodes", p.Title, len(n)), false})
	}()
}

// subscribe adds a feed URL in the background.
func (a *App) subscribe(url string) {
	url = strings.TrimSpace(url)
	if url == "" {
		return
	}
	a.busy = "subscribing"
	a.showToast("fetching "+url+"…", false)
	go func() {
		p, err := a.lib.Subscribe(url)
		if err != nil {
			a.post(toastEvent{"subscribe failed: " + err.Error(), true})
			return
		}
		_ = a.lib.Save()
		a.post(toastEvent{fmt.Sprintf("subscribed to %s (%d episodes)", p.Title, len(p.Episodes)), false})
	}()
}

func (a *App) showToast(msg string, isErr bool) {
	a.toast = msg
	a.toastErr = isErr
	a.toastTill = time.Now().Add(4 * time.Second)
}

// startPrompt opens the text input overlay.
func (a *App) startPrompt(label, initial, hint string, onDone func(string)) {
	a.prompt = prompt{active: true, label: label, text: initial, hint: hint, onDone: onDone}
}
