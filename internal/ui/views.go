package ui

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"

	"github.com/Cid-Emmerich/SopeBox/internal/art"
	"github.com/Cid-Emmerich/SopeBox/internal/download"
	"github.com/Cid-Emmerich/SopeBox/internal/transcript"
	"github.com/Cid-Emmerich/SopeBox/internal/vis"
)

// ---------------------------------------------------------------------------
// Queue

func (a *App) drawQueue(w, h int) {
	items := a.lib.QueueItems()
	top := 1
	lh := h - 2 - top
	a.drawBox(0, top, w, lh, a.th.Accent, fmt.Sprintf("queue (%d)", len(items)))
	rows := lh - 2
	if rows < 1 {
		return
	}
	if len(items) == 0 {
		a.puts(2, top+2, "the queue is empty — press e on an episode to add it, E to play it next", a.st(a.th.Muted), w-4)
		return
	}
	if a.qCursor >= len(items) {
		a.qCursor = len(items) - 1
	}
	a.qScroll = listWindow(a.qCursor, a.qScroll, len(items), rows)
	total := 0.0
	for _, it := range items {
		total += it.Episode.Duration
	}
	for i := 0; i < rows && a.qScroll+i < len(items); i++ {
		it := items[a.qScroll+i]
		ry := top + 1 + i
		sel := a.qScroll+i == a.qCursor
		base := tcell.StyleDefault
		if sel {
			base = base.Background(tc(a.th.Select))
			a.fillRow(ry, 1, w-1, base)
		}
		mark := " "
		if a.cur != nil && a.cur.Key() == it.Key() {
			mark = "▶"
		}
		a.puts(2, ry, fmt.Sprintf("%s %2d ", mark, a.qScroll+i+1), base.Foreground(tc(a.th.Accent)), 8)
		dur := fmtTime(it.Episode.Duration)
		name := it.Podcast.Title + " › " + it.Episode.Title
		a.puts(8, ry, fit(name, w-12-len(dur)), base.Foreground(tc(a.th.Text)), w-12-len(dur))
		a.puts(w-3-len(dur), ry, dur, base.Foreground(tc(a.th.Muted)), len(dur))
	}
	a.puts(2, top+lh-1, fmt.Sprintf(" %d episodes · %s ", len(items), fmtTime(total)), a.st(a.th.Muted), w-4)
}

// ---------------------------------------------------------------------------
// Downloads

func (a *App) drawDownloads(w, h int) {
	jobs := a.dl.Jobs()
	top := 1
	lh := h - 2 - top
	pol := fmt.Sprintf("auto-download latest %d · keep %d · %d at a time", a.cfg.AutoDownload, a.cfg.KeepLatest, a.cfg.Concurrency)
	if a.cfg.AutoDownload == 0 {
		pol = fmt.Sprintf("auto-download off · keep %d · %d at a time", a.cfg.KeepLatest, a.cfg.Concurrency)
	}
	a.drawBox(0, top, w, lh, a.th.Accent, "downloads · "+pol)
	rows := (lh - 2) / 2
	if rows < 1 {
		return
	}
	if len(jobs) == 0 {
		a.puts(2, top+2, "no downloads yet — press d on an episode, or set auto-download in settings", a.st(a.th.Muted), w-4)
		a.puts(2, top+3, "downloads go to "+a.cfg.DownloadDir, a.st(a.th.Muted), w-4)
		n := len(a.lib.DownloadedItems())
		a.puts(2, top+5, fmt.Sprintf("%d episodes are on disk (press l in podcasts to list them)", n), a.st(a.th.Muted), w-4)
		return
	}
	if a.dCursor >= len(jobs) {
		a.dCursor = len(jobs) - 1
	}
	a.dScroll = listWindow(a.dCursor, a.dScroll, len(jobs), rows)
	for i := 0; i < rows && a.dScroll+i < len(jobs); i++ {
		j := jobs[a.dScroll+i]
		ry := top + 1 + i*2
		sel := a.dScroll+i == a.dCursor
		base := tcell.StyleDefault
		if sel {
			base = base.Background(tc(a.th.Select))
			a.fillRow(ry, 1, w-1, base)
			a.fillRow(ry+1, 1, w-1, base)
		}
		col := a.th.Muted
		switch j.State {
		case download.Running:
			col = a.th.Accent
		case download.Done:
			col = a.th.Tertiary
		case download.Failed:
			col = a.th.Warn
		}
		state := j.State.String()
		if j.Auto {
			state += " · auto"
		}
		a.puts(2, ry, fit(j.Podcast+" › "+j.Title, w-6-len(state)), base.Foreground(tc(a.th.Text)), w-6-len(state))
		a.puts(w-2-len(state), ry, state, base.Foreground(tc(col)), len(state))
		info := ""
		switch j.State {
		case download.Running:
			if j.Total > 0 {
				info = fmt.Sprintf("%s / %s", fmtBytes(j.Done), fmtBytes(j.Total))
			} else {
				info = fmtBytes(j.Done)
			}
		case download.Failed:
			info = j.Err
		case download.Done:
			info = j.Path
		default:
			info = j.URL
		}
		bw := w/3 - 4
		if j.State == download.Running || j.State == download.Done {
			f := j.Progress()
			if j.State == download.Done {
				f = 1
			}
			a.progressBar(2, ry+1, bw, f, col)
			a.puts(4+bw, ry+1, fit(info, w-8-bw), base.Foreground(tc(a.th.Muted)), w-8-bw)
		} else {
			a.puts(2, ry+1, fit(info, w-4), base.Foreground(tc(a.th.Muted)), w-4)
		}
	}
}

// ---------------------------------------------------------------------------
// Search (iTunes)

type searchView struct {
	query   string
	typing  bool
	results []art.Result
	cursor  int
	scroll  int
	busy    bool
}

func (v *searchView) init() { v.typing = true }

func (a *App) drawSearch(w, h int) {
	v := &a.sv
	top := 1
	lh := h - 2 - top
	a.drawBox(0, top, w, lh, a.th.Accent, "search podcasts online (iTunes)")
	q := v.query
	if v.typing {
		q += "▏"
	}
	a.puts(2, top+1, "search: ", a.st(a.th.Muted), 10)
	a.puts(10, top+1, fit(q, w-12), a.st(a.th.Text).Bold(true), w-12)
	if v.busy {
		a.puts(2, top+2, "searching…", a.st(a.th.Secondary), w-4)
	}
	rows := (lh - 4) / 2
	if rows < 1 {
		return
	}
	if len(v.results) == 0 {
		if !v.busy {
			a.puts(2, top+3, "type a podcast name and press enter; enter again on a result subscribes to it", a.st(a.th.Muted), w-4)
			a.puts(2, top+4, "tip: paste a feed URL in podcasts with a, or run: sopebox import castero", a.st(a.th.Muted), w-4)
		}
		return
	}
	if v.cursor >= len(v.results) {
		v.cursor = len(v.results) - 1
	}
	v.scroll = listWindow(v.cursor, v.scroll, len(v.results), rows)
	for i := 0; i < rows && v.scroll+i < len(v.results); i++ {
		r := v.results[v.scroll+i]
		ry := top + 3 + i*2
		sel := v.scroll+i == v.cursor && !v.typing
		base := tcell.StyleDefault
		if sel {
			base = base.Background(tc(a.th.Select))
			a.fillRow(ry, 1, w-1, base)
			a.fillRow(ry+1, 1, w-1, base)
		}
		subbed := ""
		if a.lib.Find(r.FeedURL) != nil {
			subbed = " ✓ subscribed"
		}
		a.puts(2, ry, fit(r.Title, w-4-len(subbed)), base.Foreground(tc(a.th.Text)).Bold(true), w-4-len(subbed))
		a.puts(w-2-len(subbed), ry, subbed, base.Foreground(tc(a.th.Tertiary)), len(subbed))
		sub := fmt.Sprintf("%s · %s · %d episodes", r.Author, r.Genre, r.Count)
		a.puts(4, ry+1, fit(sub, w-6), base.Foreground(tc(a.th.Muted)), w-6)
	}
}

func (a *App) runSearch() {
	v := &a.sv
	q := strings.TrimSpace(v.query)
	if q == "" {
		return
	}
	v.busy = true
	v.typing = false
	go func() {
		res, err := art.SearchPodcasts(q, 25)
		a.post(searchEvent{results: res, err: err})
	}()
}

// ---------------------------------------------------------------------------
// Settings

type setting struct {
	name  string
	value func() string
	cycle func(dir int)
	help  string
}

func cycleInt(cur int, choices []int, dir int) int {
	idx := 0
	for i, c := range choices {
		if c == cur {
			idx = i
		}
	}
	idx = (idx + dir + len(choices)) % len(choices)
	return choices[idx]
}

func cycleStr(cur string, choices []string, dir int) string {
	idx := 0
	for i, c := range choices {
		if c == cur {
			idx = i
		}
	}
	idx = (idx + dir + len(choices)) % len(choices)
	return choices[idx]
}

func onOff(b bool) string {
	if b {
		return "on"
	}
	return "off"
}

func (a *App) settings() []setting {
	c := a.cfg
	return []setting{
		{"auto-download latest", func() string {
			if c.AutoDownload == 0 {
				return "off"
			}
			return fmt.Sprintf("%d most recent per podcast", c.AutoDownload)
		}, func(d int) { c.AutoDownload = cycleInt(c.AutoDownload, download.Choices, d) }, "like Apple Podcasts: keep the newest episodes of every show on disk (per-podcast override with A in the podcast list)"},
		{"keep only latest", func() string {
			if c.KeepLatest == 0 {
				return "keep everything"
			}
			return fmt.Sprintf("%d auto-downloads per podcast", c.KeepLatest)
		}, func(d int) { c.KeepLatest = cycleInt(c.KeepLatest, []int{0, 3, 5, 10, 25}, d) }, "delete older automatic downloads (manual downloads are never removed)"},
		{"parallel downloads", func() string { return fmt.Sprint(c.Concurrency) }, func(d int) {
			c.Concurrency = cycleInt(c.Concurrency, []int{1, 2, 3, 4}, d)
			a.dl.SetConcurrency(c.Concurrency)
		}, ""},
		{"download folder", func() string { return c.DownloadDir }, func(d int) {}, "change with: sopebox path <dir>"},
		{"refresh feeds on start", func() string { return onOff(c.RefreshStart) }, func(d int) { c.RefreshStart = !c.RefreshStart }, ""},
		{"continue with queue", func() string { return onOff(c.Continue) }, func(d int) { c.Continue = !c.Continue }, "play the next queued episode when one ends"},
		{"skip forward", func() string { return fmt.Sprintf("%ds", c.SkipFwd) }, func(d int) { c.SkipFwd = cycleInt(c.SkipFwd, []int{10, 15, 30, 45, 60}, d) }, "→ key"},
		{"skip back", func() string { return fmt.Sprintf("%ds", c.SkipBack) }, func(d int) { c.SkipBack = cycleInt(c.SkipBack, []int{5, 10, 15, 30}, d) }, "← key"},
		{"voice isolation mode", func() string { return a.tracker.Opts.Mode }, func(d int) {
			a.tracker.Opts.Mode = cycleStr(a.tracker.Opts.Mode, []string{"auto", "transcript", "acoustic"}, d)
		}, "auto follows transcript speaker labels when present, else listens; acoustic always listens"},
		{"voice sensitivity", func() string { return fmt.Sprintf("%.2f", a.tracker.Opts.Sensitivity) }, func(d int) {
			a.tracker.Opts.Sensitivity = clampF(a.tracker.Opts.Sensitivity+0.05*float64(d), 0, 1)
		}, "higher splits voices more eagerly (more orbs); lower merges similar voices. { } in now playing"},
		{"voice smoothing", func() string { return fmt.Sprintf("%.2f", a.tracker.Opts.Smoothing) }, func(d int) {
			a.tracker.Opts.Smoothing = clampF(a.tracker.Opts.Smoothing+0.05*float64(d), 0, 0.95)
		}, "how much each voice's fingerprint is averaged over time. ( ) in now playing"},
		{"voice falloff", func() string { return fmt.Sprintf("%.3f", a.tracker.Opts.Falloff) }, func(d int) {
			a.tracker.Opts.Falloff = clampF(a.tracker.Opts.Falloff+0.01*float64(d), 0.005, 0.5)
		}, "how fast an orb dims after its speaker stops. < > in now playing"},
		{"max voices", func() string { return fmt.Sprint(a.tracker.Opts.MaxOrbs) }, func(d int) {
			a.tracker.Opts.MaxOrbs = clampi(a.tracker.Opts.MaxOrbs+d, 1, 12)
		}, ""},
		{"captions pane", func() string { return onOff(c.Captions) }, func(d int) { c.Captions = !c.Captions }, "c in now playing"},
		{"captions side", func() string { return c.CaptionSide }, func(d int) { c.CaptionSide = cycleStr(c.CaptionSide, []string{"right", "bottom"}, d) }, ""},
		{"captions size", func() string { return fmt.Sprintf("%d%%", c.CaptionSize) }, func(d int) { c.CaptionSize = clampi(c.CaptionSize+5*d, 20, 70) }, ""},
		{"animated emoji", func() string { return onOff(c.Emoji) }, func(d int) { c.Emoji = !c.Emoji }, fmt.Sprintf("%d trigger words map to %d animated glyphs", 0, 0)},
		{"emoji style", func() string { return c.EmojiStyle }, func(d int) {
			c.EmojiStyle = cycleStr(c.EmojiStyle, []string{"blocks", "braille", "ascii", "chunky"}, d)
		}, ""},
		{"emoji colour", func() string { return c.EmojiColour }, func(d int) {
			c.EmojiColour = cycleStr(c.EmojiColour, []string{"emoji", "theme", "rainbow", "fire", "ice", "neon", "matrix", "mono"}, d)
		}, "emoji = natural colours, otherwise a gradient"},
		{"speaker names", func() string { return onOff(a.visOpts.Names) }, func(d int) { a.visOpts.Names = !a.visOpts.Names }, "names under the visualizer"},
		{"podcast icons", func() string { return onOff(c.ShowIcons) }, func(d int) { c.ShowIcons = !c.ShowIcons }, ""},
		{"icon style", func() string { return c.IconMode }, func(d int) {
			c.IconMode = cycleStr(c.IconMode, IconModes, d)
			if c.IconMode == "kitty" && !art.KittySupported() {
				c.IconMode = cycleStr(c.IconMode, IconModes, d)
			}
			a.kittyClear()
		}, "blocks = true colour half blocks, ascii, kitty = pixel-perfect in Ghostty/Kitty/WezTerm"},
		{"whisper model", func() string { return c.WhisperModel }, func(d int) { c.WhisperModel = cycleStr(c.WhisperModel, transcript.Models, d) }, "downloaded on first use to ~/.cache/sopebox/models; *.en are English-only and faster"},
		{"auto transcribe", func() string { return c.AutoTranscribe }, func(d int) {
			c.AutoTranscribe = cycleStr(c.AutoTranscribe, []string{"off", "downloaded", "always"}, d)
		}, "run whisper automatically when an episode without a feed transcript starts"},
		{"whisper.cpp", func() string {
			if p := transcript.FindWhisper(c.WhisperBin); p != "" {
				return p
			}
			return "not found (brew install whisper-cpp)"
		}, func(d int) {}, ""},
		{"theme", func() string { return a.themeNames[a.themeIdx] }, func(d int) { a.cycleTheme(d) }, ""},
		{"visualizer", func() string { return vis.Registry[a.visIdx].Name() }, func(d int) { a.cycleVis(d) }, ""},
		{"frames per second", func() string { return fmt.Sprint(c.VisFPS) }, func(d int) { c.VisFPS = cycleInt(c.VisFPS, []int{15, 20, 30, 45, 60}, d) }, "takes effect on restart"},
	}
}

func clampF(x, lo, hi float64) float64 {
	if x < lo {
		return lo
	}
	if x > hi {
		return hi
	}
	return x
}

func (a *App) drawSettings(w, h int) {
	items := a.settings()
	top := 1
	lh := h - 2 - top
	a.drawBox(0, top, w, lh, a.th.Accent, "settings")
	rows := lh - 3
	if rows < 1 {
		return
	}
	if a.setCur >= len(items) {
		a.setCur = len(items) - 1
	}
	a.setScroll = listWindow(a.setCur, a.setScroll, len(items), rows)
	nameW := 24
	for i := 0; i < rows && a.setScroll+i < len(items); i++ {
		s := items[a.setScroll+i]
		ry := top + 1 + i
		sel := a.setScroll+i == a.setCur
		base := tcell.StyleDefault
		if sel {
			base = base.Background(tc(a.th.Select))
			a.fillRow(ry, 1, w-1, base)
		}
		a.puts(2, ry, pad(s.name, nameW), base.Foreground(tc(a.th.Muted)), nameW)
		val := s.value()
		vs := base.Foreground(tc(a.th.Text))
		if sel {
			vs = base.Foreground(tc(a.th.Accent)).Bold(true)
			val = "‹ " + val + " ›"
		}
		a.puts(2+nameW, ry, fit(val, w-4-nameW), vs, w-4-nameW)
	}
	if a.setCur < len(items) {
		a.puts(2, top+lh-2, fit(items[a.setCur].help, w-4), a.st(a.th.Muted), w-4)
	}
}
