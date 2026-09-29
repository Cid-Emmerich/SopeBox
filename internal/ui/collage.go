package ui

import (
	"context"
	"errors"
	"fmt"
	"image"
	"io"
	"os"
	"strings"
	"time"

	"github.com/Cid-Emmerich/SopeBox/internal/art"
	"github.com/Cid-Emmerich/SopeBox/internal/download"
	"github.com/Cid-Emmerich/SopeBox/internal/mentions"
	"github.com/Cid-Emmerich/SopeBox/internal/store"
	"github.com/Cid-Emmerich/SopeBox/internal/transcript"
	"github.com/Cid-Emmerich/SopeBox/internal/vis"
)

// CollageModels are the Claude models offered in Settings, best first.
var CollageModels = []string{"claude-opus-5-5", "claude-sonnet-5-5", "claude-haiku-4-5"}

type mentionsEvent struct {
	key    store.Key
	tl     *mentions.Timeline
	err    error
	cached bool
}

type wikiEvent struct {
	title string
	entry *mentions.Entry
	img   image.Image
}

// prepEvent reports a background collage preparation step.
type prepEvent struct {
	key   store.Key
	tr    *transcript.Transcript
	tl    *mentions.Timeline
	phase string
	err   error
	done  bool
}

// wikiItem is a Wikipedia lookup in the UI's memory.
type wikiItem struct {
	entry   *mentions.Entry
	img     image.Image
	loading bool
}

// cacheKeyOf is the key transcripts and timelines are cached under.
func cacheKeyOf(it store.Item) string { return it.Podcast.URL + "|" + it.Episode.GUID }

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

func (a *App) collageOn() bool { return vis.Registry[a.visIdx].Name() == "collage" }

// collageWanted reports whether the playing episode should get a timeline.
func (a *App) collageWanted() bool {
	return a.cur != nil && (a.collageOn() || a.cur.Podcast.Collage)
}

func (a *App) apiKey() string {
	if k := os.Getenv("ANTHROPIC_API_KEY"); k != "" {
		return k
	}
	return a.cfg.AnthropicKey
}

func episodeInfo(it store.Item) mentions.Episode {
	var hosts []string
	for _, p := range it.Episode.Persons {
		hosts = append(hosts, p.Name)
	}
	for _, p := range it.Podcast.Persons {
		hosts = append(hosts, p.Name)
	}
	return mentions.Episode{Podcast: it.Podcast.Title, Title: it.Episode.Title,
		Description: it.Episode.Description, Hosts: dedupe(hosts)}
}

// ---------------------------------------------------------------------------
// The playing episode's timeline

// ensureMentions moves the playing episode one step closer to having a
// timeline: load it from the cache, wait for (or start) a transcript, or
// ask Claude. Safe to call often.
func (a *App) ensureMentions() {
	if a.cur == nil || a.tl != nil {
		return
	}
	it := *a.cur
	k := it.Key()
	if a.tlBusy[k] {
		return
	}
	path := mentions.CachePath(a.cfg.CacheDir, cacheKeyOf(it))
	if fileExists(path) {
		a.tlBusy[k] = true
		a.tlStatus = "loading"
		tr := a.tr
		go func() {
			tl, err := mentions.LoadCached(path)
			if err == nil {
				tl.Rebuild(tr) // no-op until the transcript is known
			}
			a.post(mentionsEvent{key: k, tl: tl, err: err, cached: true})
		}()
		return
	}
	if !a.collageWanted() || a.tlStatus == "failed" {
		return
	}
	if a.tr == nil {
		a.tlStatus = "waiting"
		// the collage needs words: transcribe a downloaded episode right
		// away (once; a failure is shown and T retries by hand)
		if !a.trAuto && a.trStatus == "none" && it.Downloaded() && !a.trBusy[k] && transcript.FindWhisper(a.cfg.WhisperBin) != "" {
			a.trAuto = true
			a.transcribe()
		}
		return
	}
	if !mentions.HaveCredentials(a.apiKey()) {
		a.tlStatus = "nokey"
		return
	}
	a.extract(it, a.tr)
}

// extract asks Claude for the timeline in the background.
func (a *App) extract(it store.Item, tr *transcript.Transcript) {
	k := it.Key()
	a.tlBusy[k] = true
	a.tlStatus = "extracting"
	a.tlErr = ""
	ep, key, model := episodeInfo(it), a.apiKey(), a.cfg.CollageModel
	path := mentions.CachePath(a.cfg.CacheDir, cacheKeyOf(it))
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
		defer cancel()
		tl, err := mentions.Extract(ctx, key, model, ep, tr)
		if err == nil {
			_ = mentions.SaveCached(path, tl)
		}
		a.post(mentionsEvent{key: k, tl: tl, err: err})
	}()
}

// redoMentions throws away the playing episode's timeline and asks again.
func (a *App) redoMentions() {
	if a.cur == nil {
		a.showToast("nothing playing", true)
		return
	}
	if a.tlBusy[a.cur.Key()] {
		a.showToast("already reading this episode", false)
		return
	}
	os.Remove(mentions.CachePath(a.cfg.CacheDir, cacheKeyOf(*a.cur)))
	a.tl = nil
	a.tlStatus, a.tlErr = "", ""
	if a.tr == nil {
		a.showToast("the collage needs a transcript first", true)
	}
	if !a.collageOn() {
		a.setVis("collage")
	}
	a.ensureMentions()
}

func (a *App) onMentions(e mentionsEvent) {
	delete(a.tlBusy, e.key)
	if a.cur == nil || e.key != a.cur.Key() {
		return
	}
	if e.err != nil {
		a.tlStatus = "failed"
		a.tlErr = shortErr(e.err)
		if !e.cached {
			a.showToast("collage: "+a.tlErr, true)
		}
		return
	}
	a.tl = e.tl
	a.tlStatus = "ok"
	if !e.cached {
		a.showToast(fmt.Sprintf("collage: %d people, places and things found (%s tokens in, %s out)",
			e.tl.Distinct(), kilo(e.tl.InputTokens), kilo(e.tl.OutputTokens)), false)
	}
}

func kilo(n int64) string {
	if n >= 1000 {
		return fmt.Sprintf("%.1fk", float64(n)/1000)
	}
	return fmt.Sprint(n)
}

// shortErr trims API errors to their first line.
func shortErr(err error) string {
	s := err.Error()
	if i := strings.IndexByte(s, '\n'); i > 0 {
		s = s[:i]
	}
	if r := []rune(s); len(r) > 160 {
		s = string(r[:159]) + "…"
	}
	return s
}

// ---------------------------------------------------------------------------
// Wikipedia

const wikiParallel = 3

// wikiFor returns what is known about a mention's article, starting a
// lookup when it hasn't been asked for yet. nil means "no article".
func (a *App) wikiFor(m mentions.Mention) *wikiItem {
	if m.Wiki == "" {
		return nil
	}
	k := strings.ToLower(m.Wiki)
	if w, ok := a.wiki[k]; ok {
		return w
	}
	w := &wikiItem{loading: true}
	a.wiki[k] = w
	a.wikiQueue = append(a.wikiQueue, m.Wiki)
	a.pumpWiki()
	return w
}

func (a *App) pumpWiki() {
	for a.wikiInFlight < wikiParallel && len(a.wikiQueue) > 0 {
		title := a.wikiQueue[0]
		a.wikiQueue = a.wikiQueue[1:]
		a.wikiInFlight++
		cache := a.cfg.CacheDir
		go func() {
			ev := wikiEvent{title: title}
			if e, err := mentions.Lookup(cache, title); err == nil {
				ev.entry = e
				if !e.Missing && e.ImageURL != "" {
					ev.img, _ = mentions.Image(cache, e)
				}
			}
			a.post(ev)
		}()
	}
}

func (a *App) onWiki(e wikiEvent) {
	a.wikiInFlight--
	if w := a.wiki[strings.ToLower(e.title)]; w != nil {
		w.entry, w.img, w.loading = e.entry, e.img, false
	}
	a.pumpWiki()
}

// ---------------------------------------------------------------------------
// Building a frame

// buildCollage turns the timeline at the current position into tiles.
func (a *App) buildCollage(pos float64) *vis.Collage {
	c := &a.collage
	c.Tiles = c.Tiles[:0]
	c.Status, c.Detail, c.Footer = "", "", ""
	c.Kitty = a.collageKitty()
	if a.tl == nil && time.Since(a.tlCheck) > time.Second {
		// catches a download finishing, a transcript arriving, a key being set
		a.tlCheck = time.Now()
		a.ensureMentions()
	}
	if a.tl != nil {
		for _, m := range a.tl.Recent(pos, 5) {
			t := vis.CollageTile{Key: m.Key(), Name: m.Name, Kind: m.Kind, Ago: pos - m.At}
			if w := a.wikiFor(m); w != nil {
				t.Img, t.Loading = w.img, w.loading
				if w.entry != nil {
					t.Desc = w.entry.Description
				}
			}
			c.Tiles = append(c.Tiles, t)
		}
		// fetch what's about to be mentioned so it appears on the word
		for _, m := range a.tl.Between(pos, pos+90) {
			a.wikiFor(m)
		}
		if len(c.Tiles) == 0 {
			c.Status = "pictures appear here as people and places are mentioned"
			c.Detail = fmt.Sprintf("%d to come in this episode", a.tl.Distinct())
		} else {
			c.Footer = fmt.Sprintf("%d in this episode · pictures from Wikipedia", a.tl.Distinct())
		}
	} else {
		c.Status, c.Detail = a.collageStatus()
	}
	if a.prepBusy {
		c.Footer = fitRunes("preparing "+a.prepTitle, 40) + " · " + a.prepPhase
	}
	return c
}

// collageStatus explains why there is no timeline yet.
func (a *App) collageStatus() (string, string) {
	switch a.tlStatus {
	case "loading":
		return "loading…", ""
	case "extracting":
		return "Claude is reading the transcript…", "about a minute for an hour of conversation; pictures follow as it's played"
	case "failed":
		return "couldn't find who and what is mentioned", a.tlErr + " · press P to try again"
	case "nokey":
		return "the collage needs a Claude API key", "set ANTHROPIC_API_KEY, or add anthropic_api_key = … to " + a.cfg.ConfigPath
	}
	switch a.trStatus {
	case "loading":
		return "looking for a transcript…", ""
	case "transcribing":
		return "transcribing with whisper.cpp · " + a.trProgress, "the pictures come once the words are known"
	}
	if transcript.FindWhisper(a.cfg.WhisperBin) == "" {
		return "the collage needs a transcript", "this feed has none; install whisper.cpp (brew install whisper-cpp) to make one"
	}
	if a.cur != nil && !a.cur.Downloaded() {
		return "the collage needs a transcript", "press d to download this episode; it is transcribed with whisper.cpp as soon as it lands"
	}
	return "the collage needs a transcript", "press T to transcribe this episode"
}

func fitRunes(s string, w int) string {
	r := []rune(s)
	if len(r) <= w {
		return s
	}
	return string(r[:w-1]) + "…"
}

// ---------------------------------------------------------------------------
// Pictures in the terminal (Kitty graphics)

func (a *App) collageKitty() bool { return a.canKitty && a.cfg.CollageImages == "kitty" }

// kittyTiles keeps the collage's pictures uploaded to the terminal and
// moves them only when the layout changes, so frames stay cheap.
type kittyTiles struct {
	base  int
	next  int
	ids   map[string]int
	imgs  map[string]image.Image
	shown map[string][4]int
}

func newKittyTiles(base int) kittyTiles {
	return kittyTiles{base: base, ids: map[string]int{}, imgs: map[string]image.Image{}, shown: map[string][4]int{}}
}

// sync makes the terminal show exactly the wanted placements. ox, oy is
// the screen position of the canvas.
func (k *kittyTiles) sync(w io.Writer, want []vis.Placement, ox, oy int) {
	var sb strings.Builder
	keep := map[string]bool{}
	for _, p := range want {
		if p.W <= 0 || p.H <= 0 || p.Img == nil {
			continue
		}
		keep[p.Key] = true
		id, ok := k.ids[p.Key]
		if !ok || k.imgs[p.Key] != p.Img {
			if !ok {
				k.next++
				id = k.base + k.next
				k.ids[p.Key] = id
			}
			sb.WriteString(art.KittyTransmit(p.Img, id))
			k.imgs[p.Key] = p.Img
			delete(k.shown, p.Key)
		}
		r := [4]int{ox + p.X, oy + p.Y, p.W, p.H}
		if k.shown[p.Key] != r {
			fmt.Fprintf(&sb, "\x1b[%d;%dH", r[1]+1, r[0]+1)
			sb.WriteString(art.KittyPlace(id, p.W, p.H))
			k.shown[p.Key] = r
		}
	}
	for key := range k.shown {
		if !keep[key] {
			sb.WriteString(art.KittyUnplace(k.ids[key]))
			delete(k.shown, key)
		}
	}
	if sb.Len() > 0 {
		// save and restore the cursor so tcell's idea of it stays true
		io.WriteString(w, "\x1b7"+sb.String()+"\x1b8")
	}
}

// hide takes every picture off the screen (they stay uploaded).
func (k *kittyTiles) hide(w io.Writer) {
	if len(k.shown) == 0 {
		return
	}
	k.sync(w, nil, 0, 0)
}

// forget deletes every uploaded picture from the terminal.
func (k *kittyTiles) forget(w io.Writer) {
	var sb strings.Builder
	for _, id := range k.ids {
		sb.WriteString(art.KittyDelete(id))
	}
	if sb.Len() > 0 {
		io.WriteString(w, sb.String())
	}
	*k = newKittyTiles(k.base)
}

// drawCollageKitty runs after the screen is shown.
func (a *App) drawCollageKitty() {
	if a.view == ViewNow && !a.help && !a.prompt.active && a.cur != nil && a.collageOn() && a.collage.Kitty {
		a.kt.sync(a.kittyOut, a.collage.Place, a.collageOrigin[0], a.collageOrigin[1])
		return
	}
	a.kt.hide(a.kittyOut)
}

// ---------------------------------------------------------------------------
// Background preparation for podcasts marked with P

// prepNewest is how many of a marked podcast's newest episodes are
// prepared; older downloads get a collage when they are played.
const prepNewest = 3

// prepTick starts preparing the next downloaded episode of a marked
// podcast that has no timeline yet: transcript, then Claude, then pictures.
func (a *App) prepTick() {
	if a.prepBusy || !mentions.HaveCredentials(a.apiKey()) {
		return
	}
	for _, p := range a.lib.Sorted() {
		if !p.Collage {
			continue
		}
		for i, it := range a.lib.Items(p) {
			if i >= prepNewest {
				break
			}
			k := it.Key()
			if a.prepDone[k] || a.trBusy[k] || a.tlBusy[k] || !it.Downloaded() {
				continue
			}
			if fileExists(mentions.CachePath(a.cfg.CacheDir, cacheKeyOf(it))) {
				a.prepDone[k] = true
				continue
			}
			a.startPrep(it)
			return
		}
	}
}

func (a *App) startPrep(it store.Item) {
	k := it.Key()
	a.prepBusy = true
	a.prepKey = k
	a.prepTitle = it.Episode.Title
	a.prepPhase = "starting"
	a.trBusy[k] = true
	a.tlBusy[k] = true
	ep := *it.Episode
	info, key, model := episodeInfo(it), a.apiKey(), a.cfg.CollageModel
	cache, bin, wmodel, audioPath := a.cfg.CacheDir, a.cfg.WhisperBin, a.cfg.WhisperModel, it.State.Path
	ck := cacheKeyOf(it)
	go func() {
		trPath := transcript.CachePath(cache, ck)
		tr, err := transcript.LoadCached(trPath)
		if err != nil || len(tr.Segments) == 0 {
			a.post(prepEvent{key: k, phase: "looking for a transcript"})
			tr, err = transcript.FromFeed(&ep)
			if err != nil {
				if transcript.FindWhisper(bin) == "" {
					a.post(prepEvent{key: k, err: errors.New("no transcript in the feed and whisper.cpp isn't installed")})
					return
				}
				tr, err = transcript.Transcribe(audioPath, transcript.Options{Bin: bin, Model: wmodel, CacheDir: cache,
					Progress: func(ph string, f float64) { a.post(progressEvent{phase: ph, frac: f, key: k}) }})
				if err != nil {
					a.post(prepEvent{key: k, err: err})
					return
				}
			}
			_ = transcript.SaveCached(trPath, tr)
		}
		a.post(prepEvent{key: k, tr: tr, phase: "Claude is reading it"})
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
		defer cancel()
		tl, err := mentions.Extract(ctx, key, model, info, tr)
		if err != nil {
			a.post(prepEvent{key: k, err: err})
			return
		}
		_ = mentions.SaveCached(mentions.CachePath(cache, ck), tl)
		a.post(prepEvent{key: k, tl: tl, phase: "fetching pictures"})
		mentions.Prefetch(cache, tl, nil)
		a.post(prepEvent{key: k, done: true})
	}()
}

func (a *App) onPrep(e prepEvent) {
	cur := a.cur != nil && a.cur.Key() == e.key
	if e.phase != "" {
		a.prepPhase = e.phase
	}
	if e.tr != nil {
		delete(a.trBusy, e.key)
		if cur && a.tr == nil {
			a.onTranscript(transcriptEvent{key: e.key, tr: e.tr})
		}
	}
	if e.tl != nil {
		delete(a.tlBusy, e.key)
		if cur {
			a.onMentions(mentionsEvent{key: e.key, tl: e.tl})
		}
	}
	if e.err != nil {
		delete(a.trBusy, e.key)
		delete(a.tlBusy, e.key)
		if cur && a.trStatus == "transcribing" {
			a.trStatus, a.trProgress = "none", ""
		}
		if cur && a.tl == nil {
			a.tlStatus, a.tlErr = "failed", shortErr(e.err)
		}
		a.showToast("collage prep ("+fitRunes(a.prepTitle, 30)+"): "+shortErr(e.err), true)
	}
	if e.err != nil || e.done {
		a.prepBusy = false
		a.prepDone[e.key] = true
		if e.done {
			a.showToast("collage ready: "+a.prepTitle, false)
		}
		a.prepTick()
	}
}

// toggleCollagePrep marks a podcast for background collage preparation.
func (a *App) toggleCollagePrep(p *store.Podcast) {
	p.Collage = !p.Collage
	if !p.Collage {
		a.showToast(p.Title+": collage prep off", false)
		return
	}
	msg := p.Title + ": new episodes are transcribed and get a collage in the background"
	if p.AutoDownload == 0 || (p.AutoDownload < 0 && a.cfg.AutoDownload == 0) {
		p.AutoDownload = 1
		a.dl.Apply(download.Policy{Latest: a.cfg.AutoDownload, KeepLatest: a.cfg.KeepLatest})
		msg += " (auto-download set to the newest 1)"
	}
	if !mentions.HaveCredentials(a.apiKey()) {
		a.showToast(p.Title+": collage prep on, but set ANTHROPIC_API_KEY first", true)
		return
	}
	a.showToast(msg, false)
	a.prepTick()
}
