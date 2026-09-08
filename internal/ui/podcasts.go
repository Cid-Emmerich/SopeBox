package ui

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"

	"github.com/Cid-Emmerich/SopeBox/internal/art"
	"github.com/Cid-Emmerich/SopeBox/internal/captions"
	"github.com/Cid-Emmerich/SopeBox/internal/download"
	"github.com/Cid-Emmerich/SopeBox/internal/paint"
	"github.com/Cid-Emmerich/SopeBox/internal/store"
)

// podcastsView is the castero-style browser: podcasts on the left,
// episodes on the right, details underneath.
type podcastsView struct {
	pods     []*store.Podcast
	items    []store.Item
	focus    int // 0 podcasts, 1 episodes
	pCursor  int
	pScroll  int
	eCursor  int
	eScroll  int
	filter   string
	typing   bool
	mode     int // 0 by podcast, 1 all recent, 2 downloaded
	kittyBox [4]int
	lastPod  *store.Podcast
}

var listModes = []string{"podcasts", "all episodes", "downloaded"}

func (v *podcastsView) init(a *App) { v.rebuild(a) }

// rebuild refreshes the podcast and episode lists.
func (v *podcastsView) rebuild(a *App) {
	v.pods = a.lib.Sorted()
	if v.pCursor >= len(v.pods) {
		v.pCursor = max(0, len(v.pods)-1)
	}
	v.rebuildEpisodes(a)
}

func (v *podcastsView) selectedPodcast(a *App) *store.Podcast {
	if v.mode != 0 {
		if it := v.selected(); it != nil {
			return it.Podcast
		}
		return nil
	}
	if v.pCursor < len(v.pods) {
		return v.pods[v.pCursor]
	}
	return nil
}

func (v *podcastsView) rebuildEpisodes(a *App) {
	var items []store.Item
	switch v.mode {
	case 1:
		items = a.lib.AllItems()
		if len(items) > 2000 {
			items = items[:2000]
		}
	case 2:
		items = a.lib.DownloadedItems()
	default:
		if p := v.selectedPodcast(a); p != nil {
			items = a.lib.Items(p)
		}
	}
	if f := strings.ToLower(strings.TrimSpace(v.filter)); f != "" {
		words := strings.Fields(f)
		var out []store.Item
		for _, it := range items {
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
		items = out
	}
	v.items = items
	if v.eCursor >= len(v.items) {
		v.eCursor = max(0, len(v.items)-1)
	}
}

func (v *podcastsView) selected() *store.Item {
	if v.eCursor < len(v.items) {
		it := v.items[v.eCursor]
		return &it
	}
	return nil
}

func (v *podcastsView) move(a *App, d int) {
	if v.focus == 0 && v.mode == 0 {
		if len(v.pods) == 0 {
			return
		}
		v.pCursor = clampi(v.pCursor+d, 0, len(v.pods)-1)
		v.eCursor, v.eScroll = 0, 0
		v.rebuildEpisodes(a)
		a.kittyClear()
		return
	}
	if len(v.items) == 0 {
		return
	}
	v.eCursor = clampi(v.eCursor+d, 0, len(v.items)-1)
	a.kittyClear()
}

func clampi(x, lo, hi int) int {
	if x < lo {
		return lo
	}
	if x > hi {
		return hi
	}
	return x
}

// jumpToPlaying moves the cursor to the playing episode.
func (v *podcastsView) jumpToPlaying(a *App) {
	if a.cur == nil {
		return
	}
	if v.mode == 0 {
		for i, p := range v.pods {
			if p.URL == a.cur.Podcast.URL {
				v.pCursor = i
			}
		}
		v.rebuildEpisodes(a)
	}
	for i, it := range v.items {
		if it.Key() == a.cur.Key() {
			v.eCursor = i
			v.focus = 1
		}
	}
}

// ---------------------------------------------------------------------------

func (a *App) drawPodcasts(w, h int) {
	v := &a.pv
	top := 1
	detailH := 8
	if h < 24 {
		detailH = 5
	}
	listH := h - 2 - top - detailH
	leftW := w / 3
	if leftW < 24 {
		leftW = 24
	}
	if v.mode != 0 {
		leftW = 0
	}
	rightX := leftW
	rightW := w - rightX

	if v.mode == 0 {
		a.drawPodcastList(0, top, leftW, listH)
	}
	a.drawEpisodeList(rightX, top, rightW, listH)
	a.drawDetails(0, top+listH, w, detailH)
}

func (a *App) drawPodcastList(x, y, w, h int) {
	v := &a.pv
	title := fmt.Sprintf("podcasts (%d)", len(v.pods))
	col := a.th.Select
	if v.focus == 0 {
		col = a.th.Accent
	}
	a.drawBox(x, y, w, h, col, title)
	rowH := 1
	iconW := 0
	if a.cfg.ShowIcons && a.cfg.IconMode != "kitty" {
		rowH = 2
		iconW = 4
	}
	rows := (h - 2) / rowH
	if rows < 1 {
		return
	}
	v.pScroll = listWindow(v.pCursor, v.pScroll, len(v.pods), rows)
	if len(v.pods) == 0 {
		a.puts(x+2, y+2, "no podcasts", a.st(a.th.Muted), w-4)
		a.puts(x+2, y+3, "a: add feed url", a.st(a.th.Muted), w-4)
		a.puts(x+2, y+4, "5: search online", a.st(a.th.Muted), w-4)
		return
	}
	for i := 0; i < rows && v.pScroll+i < len(v.pods); i++ {
		p := v.pods[v.pScroll+i]
		ry := y + 1 + i*rowH
		sel := v.pScroll+i == v.pCursor
		base := tcell.StyleDefault
		if sel {
			base = base.Background(tc(a.th.Select))
			for k := 0; k < rowH; k++ {
				a.fillRow(ry+k, x+1, x+w-1, base)
			}
		}
		tx := x + 2
		if iconW > 0 {
			if ic := a.icon(p); ic != nil {
				cells := a.iconCells(p.URL, ic, iconW, rowH)
				a.blitRows(cells, x+1, ry)
			} else {
				ph := "░░░░"
				for k := 0; k < rowH; k++ {
					a.puts(x+1, ry+k, ph, base.Foreground(tc(a.th.Select)), iconW)
				}
			}
			tx = x + 1 + iconW + 1
		}
		tw := x + w - 1 - tx
		nameStyle := base.Foreground(tc(a.th.Text))
		if sel {
			nameStyle = nameStyle.Bold(true)
		}
		if p.Error != "" {
			nameStyle = base.Foreground(tc(a.th.Warn))
		}
		marker := ""
		if a.cur != nil && a.cur.Podcast.URL == p.URL {
			marker = "▶ "
		}
		a.puts(tx, ry, fit(marker+p.Title, tw), nameStyle, tw)
		if rowH == 2 {
			sub := fmt.Sprintf("%d episodes", len(p.Episodes))
			if p.Author != "" {
				sub = p.Author + " · " + sub
			}
			if n := p.AutoDownload; n > 0 {
				sub += fmt.Sprintf(" · auto %d", n)
			}
			a.puts(tx, ry+1, fit(sub, tw), base.Foreground(tc(a.th.Muted)), tw)
		}
	}
}

func (a *App) drawEpisodeList(x, y, w, h int) {
	v := &a.pv
	var title string
	switch v.mode {
	case 1:
		title = fmt.Sprintf("all episodes (%d)", len(v.items))
	case 2:
		title = fmt.Sprintf("downloaded (%d)", len(v.items))
	default:
		if p := v.selectedPodcast(a); p != nil {
			title = fmt.Sprintf("%s (%d)", p.Title, len(v.items))
		} else {
			title = "episodes"
		}
	}
	if v.filter != "" || v.typing {
		title += " · filter: " + v.filter
		if v.typing {
			title += "▏"
		}
	}
	col := a.th.Select
	if v.focus == 1 || v.mode != 0 {
		col = a.th.Accent
	}
	a.drawBox(x, y, w, h, col, title)
	rows := h - 2
	if rows < 1 {
		return
	}
	v.eScroll = listWindow(v.eCursor, v.eScroll, len(v.items), rows)
	if len(v.items) == 0 {
		msg := "no episodes"
		if v.mode == 2 {
			msg = "nothing downloaded yet — press d on an episode"
		}
		a.puts(x+2, y+2, fit(msg, w-4), a.st(a.th.Muted), w-4)
		return
	}
	showPod := v.mode != 0
	for i := 0; i < rows && v.eScroll+i < len(v.items); i++ {
		it := v.items[v.eScroll+i]
		ry := y + 1 + i
		sel := v.eScroll+i == v.eCursor
		base := tcell.StyleDefault
		if sel {
			base = base.Background(tc(a.th.Select))
			a.fillRow(ry, x+1, x+w-1, base)
		}
		// status glyph
		mark, mcol := "·", a.th.Select
		st := it.State
		playing := a.cur != nil && a.cur.Key() == it.Key()
		switch {
		case playing:
			mark, mcol = "▶", a.th.Accent
		case st != nil && st.Played:
			mark, mcol = "✓", a.th.Muted
		case st != nil && st.Position > 30:
			mark, mcol = "◐", a.th.Secondary
		default:
			mark, mcol = "●", a.th.Tertiary
		}
		a.puts(x+2, ry, mark, base.Foreground(tc(mcol)), 1)
		dl := " "
		dcol := a.th.Muted
		if it.Downloaded() {
			dl, dcol = "↓", a.th.Tertiary
		} else if j := a.dl.JobFor(it.Key()); j != nil && (j.State == download.Queued || j.State == download.Running) {
			dl, dcol = "⇣", a.th.Secondary
		}
		a.puts(x+4, ry, dl, base.Foreground(tc(dcol)), 1)
		date := fmtDate(it.Episode.Published)
		a.puts(x+6, ry, date, base.Foreground(tc(a.th.Muted)), 10)
		dur := ""
		if it.Episode.Duration > 0 {
			dur = fmtTime(it.Episode.Duration)
		}
		tx := x + 17
		tw := x + w - 2 - tx - len(dur) - 1
		name := it.Episode.Title
		if showPod {
			name = it.Podcast.Title + " › " + name
		}
		tstyle := base.Foreground(tc(a.th.Text))
		if st != nil && st.Played && !playing {
			tstyle = base.Foreground(tc(a.th.Muted))
		}
		if sel {
			tstyle = tstyle.Bold(true)
		}
		a.puts(tx, ry, fit(name, tw), tstyle, tw)
		a.puts(x+w-2-len(dur), ry, dur, base.Foreground(tc(a.th.Muted)), len(dur))
	}
}

// drawDetails shows the selected episode's metadata with the podcast icon.
func (a *App) drawDetails(x, y, w, h int) {
	v := &a.pv
	a.drawBox(x, y, w, h, a.th.Select, "details")
	v.kittyBox = [4]int{}
	ih := h - 2
	if ih < 1 {
		return
	}
	p := v.selectedPodcast(a)
	if p == nil {
		return
	}
	// icon on the left
	iconW := ih * 2
	ix := x + 2
	tx := ix
	if a.cfg.ShowIcons && iconW > 0 {
		if a.cfg.IconMode == "kitty" {
			v.kittyBox = [4]int{ix, y + 1, iconW, ih}
			tx = ix + iconW + 2
		} else if ic := a.icon(p); ic != nil {
			cw, ch := art.Fit(ic.Image, iconW, ih)
			a.blitRows(a.iconCells(p.URL, ic, cw, ch), ix, y+1)
			tx = ix + cw + 2
		} else {
			a.puts(ix, y+1+ih/2, fit("no icon (i)", iconW), a.st(a.th.Select), iconW)
			tx = ix + iconW + 2
		}
	}
	tw := x + w - 2 - tx
	if tw < 10 {
		return
	}
	var lines []struct {
		s     string
		style tcell.Style
	}
	add := func(s string, st tcell.Style) {
		lines = append(lines, struct {
			s     string
			style tcell.Style
		}{s, st})
	}
	it := v.selected()
	if it != nil && (v.focus == 1 || v.mode != 0) {
		e := it.Episode
		add(e.Title, a.st(a.th.Accent).Bold(true))
		meta := []string{it.Podcast.Title}
		if !e.Published.IsZero() {
			meta = append(meta, e.Published.Format("Mon 2 Jan 2006"))
		}
		if e.Duration > 0 {
			meta = append(meta, fmtTime(e.Duration))
		}
		if e.Bytes > 0 {
			meta = append(meta, fmtBytes(e.Bytes))
		}
		if e.Number > 0 {
			meta = append(meta, fmt.Sprintf("ep %d", e.Number))
		}
		add(strings.Join(meta, " · "), a.st(a.th.Muted))
		var flags []string
		if it.Downloaded() {
			flags = append(flags, "downloaded")
		}
		if len(e.Transcripts) > 0 {
			flags = append(flags, "transcript in feed")
		}
		if e.Chapters != "" {
			flags = append(flags, "chapters")
		}
		if len(e.Persons) > 0 {
			var names []string
			for _, p := range e.Persons {
				names = append(names, p.Name)
			}
			flags = append(flags, "with "+strings.Join(names, ", "))
		}
		if it.State != nil && it.State.Position > 30 && !it.State.Played {
			flags = append(flags, "resume at "+fmtTime(it.State.Position))
		}
		if len(flags) > 0 {
			add(strings.Join(flags, " · "), a.st(a.th.Tertiary))
		}
		for _, l := range captions.WrapText(e.Description, tw) {
			add(l, a.st(a.th.Text))
		}
	} else {
		add(p.Title, a.st(a.th.Accent).Bold(true))
		meta := []string{}
		if p.Author != "" {
			meta = append(meta, p.Author)
		}
		meta = append(meta, fmt.Sprintf("%d episodes", len(p.Episodes)))
		if len(p.Categories) > 0 {
			meta = append(meta, strings.Join(p.Categories, ", "))
		}
		if !p.LastRefresh.IsZero() {
			meta = append(meta, "refreshed "+p.LastRefresh.Format("2 Jan 15:04"))
		}
		add(strings.Join(meta, " · "), a.st(a.th.Muted))
		auto := "auto-download: global"
		if p.AutoDownload >= 0 {
			auto = fmt.Sprintf("auto-download: %d", p.AutoDownload)
			if p.AutoDownload == 0 {
				auto = "auto-download: off"
			}
		}
		extras := []string{auto}
		if len(p.Voices) > 0 {
			var names []string
			for _, vp := range p.Voices {
				names = append(names, vp.Name)
			}
			extras = append(extras, "known voices: "+strings.Join(names, ", "))
		}
		if len(p.Persons) > 0 {
			var names []string
			for _, pp := range p.Persons {
				names = append(names, pp.Name)
			}
			extras = append(extras, "people: "+strings.Join(names, ", "))
		}
		if p.IconSource != "" {
			extras = append(extras, "icon: "+p.IconSource)
		}
		if p.Error != "" {
			add("error: "+p.Error, a.st(a.th.Warn))
		}
		add(strings.Join(extras, " · "), a.st(a.th.Tertiary))
		for _, l := range captions.WrapText(p.Description, tw) {
			add(l, a.st(a.th.Text))
		}
	}
	for i, l := range lines {
		if i >= ih {
			break
		}
		a.puts(tx, y+1+i, fit(l.s, tw), l.style, tw)
	}
}

var _ = paint.RGB{}
