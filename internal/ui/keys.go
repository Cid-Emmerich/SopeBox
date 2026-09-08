package ui

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"

	"github.com/Cid-Emmerich/SopeBox/internal/art"
	"github.com/Cid-Emmerich/SopeBox/internal/config"
	"github.com/Cid-Emmerich/SopeBox/internal/download"
	"github.com/Cid-Emmerich/SopeBox/internal/store"
	"github.com/Cid-Emmerich/SopeBox/internal/vis"
	"github.com/Cid-Emmerich/SopeBox/internal/voices"
)

func (a *App) handleKey(e *tcell.EventKey) {
	key, r := e.Key(), e.Rune()

	// Text prompt swallows everything.
	if a.prompt.active {
		switch key {
		case tcell.KeyEscape:
			a.prompt.active = false
		case tcell.KeyEnter:
			p := a.prompt
			a.prompt.active = false
			if p.onDone != nil {
				p.onDone(strings.TrimSpace(p.text))
			}
		case tcell.KeyBackspace, tcell.KeyBackspace2:
			if rs := []rune(a.prompt.text); len(rs) > 0 {
				a.prompt.text = string(rs[:len(rs)-1])
			}
		case tcell.KeyCtrlU:
			a.prompt.text = ""
		case tcell.KeyCtrlC:
			a.quit = true
		case tcell.KeyRune:
			a.prompt.text += string(r)
		}
		return
	}

	// Help overlay.
	if a.help {
		switch {
		case key == tcell.KeyCtrlK, key == tcell.KeyEscape, r == 'q', r == '?', key == tcell.KeyEnter:
			a.help = false
		case key == tcell.KeyDown, r == 'j':
			a.helpScroll++
		case key == tcell.KeyUp, r == 'k':
			if a.helpScroll > 0 {
				a.helpScroll--
			}
		case key == tcell.KeyPgDn:
			a.helpScroll += 10
		case key == tcell.KeyPgUp:
			a.helpScroll = max(0, a.helpScroll-10)
		case key == tcell.KeyCtrlC:
			a.quit = true
		}
		return
	}

	// Typing modes.
	if a.view == ViewPodcasts && a.pv.typing {
		switch key {
		case tcell.KeyEscape:
			a.pv.filter = ""
			a.pv.typing = false
			a.pv.rebuildEpisodes(a)
		case tcell.KeyEnter:
			a.pv.typing = false
		case tcell.KeyBackspace, tcell.KeyBackspace2:
			if rs := []rune(a.pv.filter); len(rs) > 0 {
				a.pv.filter = string(rs[:len(rs)-1])
				a.pv.rebuildEpisodes(a)
			}
		case tcell.KeyDown, tcell.KeyCtrlN:
			a.pv.move(a, 1)
		case tcell.KeyUp, tcell.KeyCtrlP:
			a.pv.move(a, -1)
		case tcell.KeyCtrlC:
			a.quit = true
		case tcell.KeyRune:
			a.pv.filter += string(r)
			a.pv.eCursor = 0
			a.pv.rebuildEpisodes(a)
		}
		return
	}
	if a.view == ViewSearch && a.sv.typing {
		switch key {
		case tcell.KeyEscape:
			if a.sv.query == "" {
				a.switchView(ViewPodcasts)
			} else {
				a.sv.typing = false
			}
		case tcell.KeyEnter:
			a.runSearch()
		case tcell.KeyBackspace, tcell.KeyBackspace2:
			if rs := []rune(a.sv.query); len(rs) > 0 {
				a.sv.query = string(rs[:len(rs)-1])
			}
		case tcell.KeyCtrlU:
			a.sv.query = ""
		case tcell.KeyDown, tcell.KeyTab:
			if len(a.sv.results) > 0 {
				a.sv.typing = false
			}
		case tcell.KeyCtrlC:
			a.quit = true
		case tcell.KeyCtrlK:
			a.help = true
		case tcell.KeyRune:
			a.sv.query += string(r)
		}
		return
	}

	switch key {
	case tcell.KeyCtrlK:
		a.help = true
		a.helpScroll = 0
		return
	case tcell.KeyCtrlC:
		a.quit = true
		return
	case tcell.KeyTab:
		a.switchView((a.view + 1) % numViews)
		return
	case tcell.KeyBacktab:
		a.switchView((a.view + numViews - 1) % numViews)
		return
	case tcell.KeyEscape:
		if a.view != ViewNow {
			a.switchView(ViewNow)
		}
		a.toast = ""
		return
	case tcell.KeyLeft:
		a.leftRight(-1, e.Modifiers())
		return
	case tcell.KeyRight:
		a.leftRight(1, e.Modifiers())
		return
	case tcell.KeyUp:
		a.listMove(-1)
		return
	case tcell.KeyDown:
		a.listMove(1)
		return
	case tcell.KeyPgUp:
		a.listMove(-10)
		return
	case tcell.KeyPgDn:
		a.listMove(10)
		return
	case tcell.KeyHome:
		a.listMove(-1 << 20)
		return
	case tcell.KeyEnd:
		a.listMove(1 << 20)
		return
	case tcell.KeyEnter:
		a.activate()
		return
	case tcell.KeyDelete:
		a.deleteKey()
		return
	case tcell.KeyRune:
	default:
		return
	}

	// --- Rune keys: global -------------------------------------------------
	switch r {
	case 'q':
		a.quit = true
		return
	case '?':
		a.help = true
		a.helpScroll = 0
		return
	case '1', '2', '3', '4', '5', '6':
		a.switchView(View(r - '1'))
		return
	case ' ':
		if a.cur == nil {
			if it := a.pv.selected(); it != nil && a.view == ViewPodcasts {
				a.play(*it)
				return
			}
		}
		a.pl.TogglePause()
		return
	case '+', '=':
		a.pl.VolumeDelta(0.05)
		return
	case '-', '_':
		a.pl.VolumeDelta(-0.05)
		return
	case 'm':
		if a.view == ViewPodcasts {
			a.togglePlayed()
			return
		}
		a.pl.ToggleMute()
		return
	case 's':
		a.showToast(fmt.Sprintf("speed %.2fx", a.pl.SpeedDelta(0.1)), false)
		return
	case 'S':
		a.showToast(fmt.Sprintf("speed %.2fx", a.pl.SpeedDelta(-0.1)), false)
		return
	case 'n':
		a.playNext()
		return
	case 'b':
		a.playPrev()
		return
	case 't':
		a.cycleTheme(1)
		return
	case 'T':
		if a.view == ViewNow {
			a.transcribe()
		} else {
			a.cycleTheme(-1)
		}
		return
	case 'v':
		a.cycleVis(1)
		return
	case 'V':
		a.cycleVis(-1)
		return
	}

	switch a.view {
	case ViewNow:
		a.nowKey(r)
	case ViewPodcasts:
		a.podcastsKey(r)
	case ViewQueue:
		a.queueKey(r)
	case ViewDownloads:
		a.downloadsKey(r)
	case ViewSearch:
		a.searchKey(r)
	case ViewSettings:
		a.settingsKey(r)
	}
}

func (a *App) switchView(v View) {
	a.view = v
	a.kittyClear()
	a.cellCache = map[string][][]cellRow{}
	if v == ViewPodcasts {
		a.pv.rebuild(a)
	}
	if v == ViewSearch && len(a.sv.results) == 0 {
		a.sv.typing = true
	}
}

func (a *App) leftRight(dir int, mod tcell.ModMask) {
	switch a.view {
	case ViewPodcasts:
		if a.pv.mode == 0 {
			if dir < 0 {
				a.pv.focus = 0
			} else {
				a.pv.focus = 1
			}
			a.kittyClear()
		}
	case ViewSettings:
		items := a.settings()
		if a.setCur < len(items) {
			items[a.setCur].cycle(dir)
		}
	default:
		secs := float64(a.cfg.SkipFwd)
		if dir < 0 {
			secs = -float64(a.cfg.SkipBack)
		}
		if mod&tcell.ModShift != 0 {
			secs *= 4
		}
		a.pl.Seek(secs)
	}
}

func (a *App) listMove(d int) {
	switch a.view {
	case ViewNow:
		if d < 0 {
			a.pl.VolumeDelta(0.05)
		} else if d > 0 {
			a.pl.VolumeDelta(-0.05)
		}
	case ViewPodcasts:
		if d > 1000 || d < -1000 {
			if a.pv.focus == 0 && a.pv.mode == 0 {
				a.pv.pCursor = 0
				if d > 0 {
					a.pv.pCursor = max(0, len(a.pv.pods)-1)
				}
				a.pv.rebuildEpisodes(a)
			} else {
				a.pv.eCursor = 0
				if d > 0 {
					a.pv.eCursor = max(0, len(a.pv.items)-1)
				}
			}
			return
		}
		a.pv.move(a, d)
	case ViewQueue:
		n := len(a.lib.Queue)
		a.qCursor = clampi(a.qCursor+d, 0, max(0, n-1))
	case ViewDownloads:
		n := len(a.dl.Jobs())
		a.dCursor = clampi(a.dCursor+d, 0, max(0, n-1))
	case ViewSearch:
		if a.sv.typing && d > 0 && len(a.sv.results) > 0 {
			a.sv.typing = false
			return
		}
		if a.sv.cursor == 0 && d < 0 {
			a.sv.typing = true
			return
		}
		a.sv.cursor = clampi(a.sv.cursor+d, 0, max(0, len(a.sv.results)-1))
	case ViewSettings:
		a.setCur = clampi(a.setCur+d, 0, len(a.settings())-1)
	}
}

func (a *App) activate() {
	switch a.view {
	case ViewNow:
		a.pl.TogglePause()
	case ViewPodcasts:
		if a.pv.focus == 0 && a.pv.mode == 0 {
			a.pv.focus = 1
			return
		}
		if it := a.pv.selected(); it != nil {
			a.play(*it)
			a.switchView(ViewNow)
		}
	case ViewQueue:
		q := a.lib.QueueItems()
		if a.qCursor < len(q) {
			a.play(q[a.qCursor])
			a.switchView(ViewNow)
		}
	case ViewDownloads:
		jobs := a.dl.Jobs()
		if a.dCursor < len(jobs) && jobs[a.dCursor].State == download.Done {
			if it := a.lib.Get(jobs[a.dCursor].Key); it != nil {
				a.play(*it)
				a.switchView(ViewNow)
			}
		}
	case ViewSearch:
		if a.sv.typing {
			a.runSearch()
			return
		}
		if a.sv.cursor < len(a.sv.results) {
			r := a.sv.results[a.sv.cursor]
			if r.FeedURL == "" {
				a.showToast("this result has no feed url", true)
				return
			}
			a.subscribe(r.FeedURL)
		}
	case ViewSettings:
		items := a.settings()
		if a.setCur < len(items) {
			items[a.setCur].cycle(1)
		}
	}
}

func (a *App) deleteKey() {
	switch a.view {
	case ViewQueue:
		a.queueKey('x')
	case ViewPodcasts:
		a.podcastsKey('D')
	}
}

// --- now playing --------------------------------------------------------------

func (a *App) nowKey(r rune) {
	o := &a.visOpts
	vo := &a.tracker.Opts
	switch r {
	case 'c':
		a.cfg.Captions = !a.cfg.Captions
		a.showToast("captions "+onOff(a.cfg.Captions), false)
	case 'C':
		a.cfg.CaptionSide = cycleStr(a.cfg.CaptionSide, []string{"right", "bottom"}, 1)
		a.showToast("captions on the "+a.cfg.CaptionSide, false)
	case 'u':
		a.cfg.CaptionSize = clampi(a.cfg.CaptionSize-5, 20, 70)
	case 'U':
		a.cfg.CaptionSize = clampi(a.cfg.CaptionSize+5, 20, 70)
	case 'j':
		a.cfg.Emoji = !a.cfg.Emoji
		a.showToast("animated emoji "+onOff(a.cfg.Emoji), false)
	case 'J':
		a.cfg.EmojiStyle = cycleStr(a.cfg.EmojiStyle, []string{"blocks", "braille", "ascii", "chunky"}, 1)
		a.showToast("emoji style: "+a.cfg.EmojiStyle, false)
	case 'k':
		a.cfg.EmojiColour = cycleStr(a.cfg.EmojiColour, []string{"emoji", "theme", "rainbow", "fire", "ice", "neon", "matrix", "mono"}, 1)
		a.showToast("emoji colour: "+a.cfg.EmojiColour, false)
	case 'L':
		o.Names = !o.Names
	case 'g':
		o.Gradient = cycleStr(o.Gradient, vis.GradientNames, 1)
		a.showToast("gradient: "+o.Gradient, false)
	case 'G':
		o.Gradient = cycleStr(o.Gradient, vis.GradientNames, -1)
		a.showToast("gradient: "+o.Gradient, false)
	case 'i':
		o.Fill = cycleStr(o.Fill, vis.FillNames, 1)
		a.showToast("fill: "+o.Fill, false)
	case 'x':
		o.Peaks = !o.Peaks
	case 'y':
		o.Mirror = !o.Mirror
	case 'p':
		o.Physics = !o.Physics
		a.showToast("orb physics "+onOff(o.Physics), false)
	case 'w':
		o.Rotate = clampF(o.Rotate+0.05, -2, 2)
		a.showToast(fmt.Sprintf("rotation %.2f", o.Rotate), false)
	case 'W':
		o.Rotate = clampF(o.Rotate-0.05, -2, 2)
		a.showToast(fmt.Sprintf("rotation %.2f", o.Rotate), false)
	case 'e':
		o.OrbSize = clampF(o.OrbSize+0.1, 0.4, 2.5)
		a.showToast(fmt.Sprintf("orb size %.1f", o.OrbSize), false)
	case 'E':
		o.OrbSize = clampF(o.OrbSize-0.1, 0.4, 2.5)
		a.showToast(fmt.Sprintf("orb size %.1f", o.OrbSize), false)
	case ',':
		o.Smoothing = clampF(o.Smoothing-0.05, 0, 0.95)
		a.showToast(fmt.Sprintf("smoothing %.2f", o.Smoothing), false)
	case '.':
		o.Smoothing = clampF(o.Smoothing+0.05, 0, 0.95)
		a.showToast(fmt.Sprintf("smoothing %.2f", o.Smoothing), false)
	case '[':
		o.Gain = clampF(o.Gain-0.1, 0.1, 10)
		a.showToast(fmt.Sprintf("gain %.1f", o.Gain), false)
	case ']':
		o.Gain = clampF(o.Gain+0.1, 0.1, 10)
		a.showToast(fmt.Sprintf("gain %.1f", o.Gain), false)
	case ';':
		o.Falloff = clampF(o.Falloff-0.01, 0.01, 1)
		a.showToast(fmt.Sprintf("falloff %.2f", o.Falloff), false)
	case '\'':
		o.Falloff = clampF(o.Falloff+0.01, 0.01, 1)
		a.showToast(fmt.Sprintf("falloff %.2f", o.Falloff), false)
	case 'R':
		def := config.Default()
		a.visOpts = vis.OptionsFromConfig(def)
		a.visOpts.Names = o.Names
		vo.Sensitivity, vo.Smoothing, vo.Falloff = def.VoiceSensitivity, def.VoiceSmoothing, def.VoiceFalloff
		a.showToast("visualizer and voice settings reset", false)
	// voice isolation
	case '{':
		vo.Sensitivity = clampF(vo.Sensitivity-0.05, 0, 1)
		a.showToast(fmt.Sprintf("voice sensitivity %.2f (lower = fewer orbs)", vo.Sensitivity), false)
	case '}':
		vo.Sensitivity = clampF(vo.Sensitivity+0.05, 0, 1)
		a.showToast(fmt.Sprintf("voice sensitivity %.2f (higher = more orbs)", vo.Sensitivity), false)
	case '(':
		vo.Smoothing = clampF(vo.Smoothing-0.05, 0, 0.95)
		a.showToast(fmt.Sprintf("voice smoothing %.2f", vo.Smoothing), false)
	case ')':
		vo.Smoothing = clampF(vo.Smoothing+0.05, 0, 0.95)
		a.showToast(fmt.Sprintf("voice smoothing %.2f", vo.Smoothing), false)
	case '<':
		vo.Falloff = clampF(vo.Falloff-0.01, 0.005, 0.5)
		a.showToast(fmt.Sprintf("voice falloff %.3f", vo.Falloff), false)
	case '>':
		vo.Falloff = clampF(vo.Falloff+0.01, 0.005, 0.5)
		a.showToast(fmt.Sprintf("voice falloff %.3f", vo.Falloff), false)
	case 'M':
		vo.Mode = cycleStr(vo.Mode, []string{"auto", "transcript", "acoustic"}, 1)
		a.showToast("voice mode: "+vo.Mode, false)
	case 'N':
		a.renameVoice()
	case 'B':
		a.nameFromPersons()
	case 'K':
		a.mergeVoice()
	case 'X':
		a.tracker.Reset()
		if a.cur != nil {
			a.tracker.SetTranscript(a.tr)
		}
		a.showToast("voices cleared for this episode", false)
	case 'F':
		if a.cur != nil {
			a.lib.ForgetVoices(a.cur.Podcast)
			a.tracker.Reset()
			a.tracker.SetTranscript(a.tr)
			a.showToast("forgot learned voices for "+a.cur.Podcast.Title, false)
		}
	case 'd':
		if a.cur != nil {
			a.download(*a.cur)
		}
	case 'o':
		a.pv.jumpToPlaying(a)
		a.switchView(ViewPodcasts)
	case 'r':
		if a.cur != nil {
			a.loadTranscript(*a.cur, true)
			a.showToast("reloading captions", false)
		}
	}
}

func (a *App) currentOrb() *voices.Orb {
	if o := a.tracker.Current(); o != nil {
		return o
	}
	// most recently active
	var best *voices.Orb
	for _, o := range a.tracker.Visible() {
		if best == nil || o.Last > best.Last {
			best = o
		}
	}
	return best
}

func (a *App) renameVoice() {
	o := a.currentOrb()
	if o == nil {
		a.showToast("no voice to name yet", true)
		return
	}
	hint := "enter = save · esc = cancel"
	if ps := a.tracker.Persons(); len(ps) > 0 {
		hint = "people in this feed: " + strings.Join(ps, ", ") + " (B cycles them)"
	}
	a.startPrompt("name for "+o.Name, "", hint, func(s string) {
		if s == "" {
			return
		}
		a.tracker.Rename(o, s)
		if a.cur != nil {
			a.lib.SaveVoices(a.cur.Podcast, a.tracker.Export())
		}
		a.showToast("voice named "+s, false)
	})
}

// nameFromPersons cycles the current orb's name through the feed's people.
func (a *App) nameFromPersons() {
	o := a.currentOrb()
	ps := a.tracker.Persons()
	if o == nil || len(ps) == 0 {
		a.showToast("no people declared in this feed — use N to type a name", true)
		return
	}
	idx := -1
	for i, p := range ps {
		if strings.EqualFold(p, o.Name) {
			idx = i
		}
	}
	name := ps[(idx+1)%len(ps)]
	a.tracker.Rename(o, name)
	a.showToast("voice named "+name, false)
}

func (a *App) mergeVoice() {
	o := a.currentOrb()
	vis := a.tracker.Visible()
	if o == nil || len(vis) < 2 {
		a.showToast("need two voices to merge", true)
		return
	}
	var other *voices.Orb
	for _, v := range vis {
		if v != o && (other == nil || v.Last > other.Last) {
			other = v
		}
	}
	a.tracker.Merge(other, o)
	a.showToast(fmt.Sprintf("merged %s into %s", o.Name, other.Name), false)
}

func (a *App) download(it store.Item) {
	if err := a.dl.Enqueue(it, false); err != nil {
		a.showToast("download: "+err.Error(), true)
		return
	}
	a.showToast("downloading "+it.Episode.Title, false)
}

// --- podcasts -----------------------------------------------------------------

func (a *App) podcastsKey(r rune) {
	v := &a.pv
	switch r {
	case 'j':
		v.move(a, 1)
	case 'k':
		v.move(a, -1)
	case 'h':
		if v.mode == 0 {
			v.focus = 0
		}
	case 'l':
		if v.mode == 0 && v.focus == 0 {
			v.focus = 1
		} else {
			v.mode = (v.mode + 1) % 3
			v.eCursor, v.eScroll = 0, 0
			v.rebuildEpisodes(a)
			a.showToast("listing: "+listModes[v.mode], false)
		}
	case '/':
		v.typing = true
		if v.mode == 0 {
			v.focus = 1
		}
	case 'e':
		if it := v.selected(); it != nil {
			a.lib.Enqueue(it.Key())
			a.showToast("queued: "+it.Episode.Title, false)
		}
	case 'E':
		if it := v.selected(); it != nil {
			pos := 0
			if a.cur != nil {
				for i, q := range a.lib.QueueItems() {
					if q.Key() == a.cur.Key() {
						pos = i + 1
					}
				}
			}
			a.lib.EnqueueAt(it.Key(), pos)
			a.showToast("playing next: "+it.Episode.Title, false)
		}
	case 'd':
		if v.focus == 0 && v.mode == 0 {
			if p := v.selectedPodcast(a); p != nil {
				n := 0
				for i, it := range a.lib.Items(p) {
					if i >= 3 {
						break
					}
					if a.dl.Enqueue(it, false) == nil {
						n++
					}
				}
				a.showToast(fmt.Sprintf("downloading the %d newest episodes of %s", n, p.Title), false)
			}
			return
		}
		if it := v.selected(); it != nil {
			a.download(*it)
		}
	case 'D':
		if it := v.selected(); it != nil && it.Downloaded() {
			if err := a.dl.Remove(*it); err != nil {
				a.showToast("delete: "+err.Error(), true)
			} else {
				a.showToast("deleted download", false)
				v.rebuildEpisodes(a)
			}
		} else if it != nil {
			a.dl.Cancel(it.Key())
		}
	case 'a':
		a.startPrompt("add podcast feed", "", "paste an RSS feed URL", func(s string) { a.subscribe(s) })
	case 'r':
		a.refreshOne(v.selectedPodcast(a))
	case 'R':
		a.refreshAll()
	case 'i':
		if p := v.selectedPodcast(a); p != nil {
			a.showToast("searching for an icon: "+p.Title+"…", false)
			a.requestIcon(p, true)
		}
	case 'I':
		if p := v.selectedPodcast(a); p != nil {
			delete(a.icons, p.URL)
			a.requestIcon(p, false)
		}
	case 'A':
		if p := v.selectedPodcast(a); p != nil {
			choices := append([]int{-1}, download.Choices...)
			p.AutoDownload = cycleInt(p.AutoDownload, choices, 1)
			switch p.AutoDownload {
			case -1:
				a.showToast(p.Title+": auto-download follows the global setting", false)
			case 0:
				a.showToast(p.Title+": auto-download off", false)
			default:
				a.showToast(fmt.Sprintf("%s: keep the %d newest episodes downloaded", p.Title, p.AutoDownload), false)
				a.dl.Apply(download.Policy{Latest: a.cfg.AutoDownload, KeepLatest: a.cfg.KeepLatest})
			}
		}
	case 'x':
		if p := v.selectedPodcast(a); p != nil && v.focus == 0 && v.mode == 0 {
			a.startPrompt("unsubscribe from "+p.Title+"?", "", "type yes to confirm", func(s string) {
				if strings.ToLower(s) == "yes" || strings.ToLower(s) == "y" {
					a.lib.Unsubscribe(p.URL)
					v.rebuild(a)
					a.showToast("unsubscribed from "+p.Title, false)
				}
			})
		}
	case 'o':
		v.jumpToPlaying(a)
	case 'z':
		v.filter = ""
		v.rebuildEpisodes(a)
	case 'c':
		a.cfg.ShowIcons = !a.cfg.ShowIcons
		a.kittyClear()
	case 'C':
		a.cfg.IconMode = cycleStr(a.cfg.IconMode, IconModes, 1)
		if a.cfg.IconMode == "kitty" && !art.KittySupported() {
			a.cfg.IconMode = "blocks"
		}
		a.kittyClear()
		a.showToast("icon style: "+a.cfg.IconMode, false)
	}
}

func (a *App) togglePlayed() {
	if it := a.pv.selected(); it != nil {
		s := a.lib.State(it.Key())
		s.Played = !s.Played
		if s.Played {
			s.Position = 0
		}
		a.pv.rebuildEpisodes(a)
	}
}

// --- queue ----------------------------------------------------------------------

func (a *App) queueKey(r rune) {
	q := a.lib.QueueItems()
	switch r {
	case 'j':
		a.listMove(1)
	case 'k':
		a.listMove(-1)
	case 'x':
		if a.qCursor < len(q) {
			a.lib.Dequeue(q[a.qCursor].Key())
		}
	case 'J', 'K':
		if len(q) < 2 || a.qCursor >= len(q) {
			return
		}
		to := a.qCursor + 1
		if r == 'K' {
			to = a.qCursor - 1
		}
		if to < 0 || to >= len(q) {
			return
		}
		keys := make([]store.Key, len(q))
		for i, it := range q {
			keys[i] = it.Key()
		}
		keys[a.qCursor], keys[to] = keys[to], keys[a.qCursor]
		a.lib.SetQueue(keys)
		a.qCursor = to
	case 'C':
		a.lib.SetQueue(nil)
		a.showToast("queue cleared", false)
	case 'd':
		if a.qCursor < len(q) {
			a.download(q[a.qCursor])
		}
	case 'g', 'o':
		for i, it := range q {
			if a.cur != nil && it.Key() == a.cur.Key() {
				a.qCursor = i
			}
		}
	}
}

// --- downloads -------------------------------------------------------------------

func (a *App) downloadsKey(r rune) {
	jobs := a.dl.Jobs()
	switch r {
	case 'j':
		a.listMove(1)
	case 'k':
		a.listMove(-1)
	case 'x':
		if a.dCursor < len(jobs) {
			a.dl.Cancel(jobs[a.dCursor].Key)
		}
	case 'C':
		a.dl.Clear()
	case 'a':
		q, rm := a.dl.Apply(download.Policy{Latest: a.cfg.AutoDownload, KeepLatest: a.cfg.KeepLatest})
		a.showToast(fmt.Sprintf("auto-download applied: %d queued, %d removed", q, rm), false)
	case 'e':
		if a.dCursor < len(jobs) {
			a.lib.Enqueue(jobs[a.dCursor].Key)
			a.showToast("queued", false)
		}
	}
}

// --- search --------------------------------------------------------------------

func (a *App) searchKey(r rune) {
	switch r {
	case 'j':
		a.listMove(1)
	case 'k':
		a.listMove(-1)
	case '/', 'i':
		a.sv.typing = true
	}
}

// --- settings ------------------------------------------------------------------

func (a *App) settingsKey(r rune) {
	switch r {
	case 'j':
		a.listMove(1)
	case 'k':
		a.listMove(-1)
	case 'h':
		a.leftRight(-1, 0)
	case 'l':
		a.leftRight(1, 0)
	}
}

// --- cycles ---------------------------------------------------------------------

func (a *App) cycleTheme(d int) {
	a.themeIdx = (a.themeIdx + d + len(a.themeNames)) % len(a.themeNames)
	a.applyTheme()
	a.cellCache = map[string][][]cellRow{}
	a.showToast("theme: "+a.th.Name, false)
}

func (a *App) cycleVis(d int) {
	a.visIdx = (a.visIdx + d + len(vis.Registry)) % len(vis.Registry)
	v := vis.Registry[a.visIdx]
	a.showToast("visualizer: "+v.Name()+" — "+v.Describe(), false)
}

// --- mouse -------------------------------------------------------------------

func (a *App) handleMouse(e *tcell.EventMouse) {
	x, y := e.Position()
	btn := e.Buttons()
	if a.help || a.prompt.active {
		if btn&tcell.Button1 != 0 {
			a.help = false
		}
		return
	}
	switch {
	case btn&tcell.WheelUp != 0:
		a.listMove(-1)
		if a.view == ViewNow {
			a.pl.VolumeDelta(0.05)
		}
	case btn&tcell.WheelDown != 0:
		a.listMove(1)
		if a.view == ViewNow {
			a.pl.VolumeDelta(-0.05)
		}
	case btn&tcell.Button1 != 0:
		if y == 0 {
			for i, pos := range tabPositions(a.scrW()) {
				if x >= pos[0] && x < pos[1] {
					a.switchView(View(i))
				}
			}
			return
		}
		if a.view == ViewNow && y == a.mouseSeekRow && a.mouseSeekX1 > a.mouseSeekX0 {
			st := a.pl.Status()
			if st.Duration > 0 {
				frac := float64(x-a.mouseSeekX0) / float64(a.mouseSeekX1-a.mouseSeekX0)
				a.pl.SeekTo(clampF(frac, 0, 1) * st.Duration)
			}
		}
	}
}

func (a *App) scrW() int {
	w, _ := a.scr.Size()
	return w
}
