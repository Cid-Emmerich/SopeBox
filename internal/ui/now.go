package ui

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"

	"github.com/Cid-Emmerich/SopeBox/internal/art"
	"github.com/Cid-Emmerich/SopeBox/internal/audio"
	"github.com/Cid-Emmerich/SopeBox/internal/captions"
	"github.com/Cid-Emmerich/SopeBox/internal/paint"
	"github.com/Cid-Emmerich/SopeBox/internal/transcript"
	"github.com/Cid-Emmerich/SopeBox/internal/vis"
)

// drawNow renders the now-playing screen: visualizer, captions, names,
// progress and status.
func (a *App) drawNow(w, h int) {
	st := a.pl.Status()
	top := 1
	mainH := h - 5 // rows 1 .. h-5
	progressRow := h - 3
	a.mouseSeekRow = progressRow

	visX, visW, visH := 1, w-2, mainH
	capX, capY, capW, capH := 0, 0, 0, 0
	if a.cfg.Captions && w >= 70 {
		if a.cfg.CaptionSide == "bottom" {
			capH = mainH * a.cfg.CaptionSize / 100
			if capH < 6 {
				capH = 6
			}
			visH = mainH - capH
			capX, capY, capW = 1, top+visH, w-2
		} else {
			capW = w * a.cfg.CaptionSize / 100
			if capW < 26 {
				capW = 26
			}
			visW = w - 2 - capW
			capX, capY, capH = 1+visW, top, mainH
		}
	}
	if a.cur == nil {
		a.drawIdle(w, top, visW, visH)
	} else {
		a.drawVisualizer(visX, top, visW, visH, st)
	}
	if capW > 0 {
		a.drawCaptions(capX, capY, capW, capH, st)
	}
	a.drawProgress(w, progressRow, st)
	a.drawStatus(w, h-2, st)
}

func (a *App) drawIdle(w, top, vw, vh int) {
	lines := []string{
		"nothing playing",
		"",
		"2        browse your podcasts",
		"5        search for podcasts to subscribe to",
		"enter    play the selected episode",
		"ctrl+k   every keyboard shortcut",
	}
	if len(a.lib.Podcasts) == 0 {
		lines = []string{
			"no podcasts yet",
			"",
			"5              search iTunes and subscribe",
			"a              add a feed by URL",
			"sopebox import castero|opml   bring in existing subscriptions",
		}
	}
	y := top + (vh-len(lines))/2
	for i, l := range lines {
		style := a.st(a.th.Muted)
		if i == 0 {
			style = a.st(a.th.Accent).Bold(true)
		}
		a.puts(1+(vw-len(l))/2, y+i, l, style, vw)
	}
}

func (a *App) drawVisualizer(x0, top, cw, ch int, st audio.Status) {
	namesRow := 0
	if a.visOpts.Names {
		namesRow = 1
	}
	vh := ch - 1 - namesRow // one row for the title
	if vh < 3 {
		vh = 3
	}
	if a.canvas == nil || a.canvas.W != cw || a.canvas.H != vh {
		a.canvas = vis.NewCanvas(cw, vh)
	}
	a.canvas.Clear()
	f := &vis.Frame{
		Analyzer: a.pl.Analyzer,
		Tracker:  a.tracker,
		Opts:     &a.visOpts,
		Theme:    a.th,
		Time:     time.Since(a.visStart).Seconds(),
		DT:       1.0 / float64(max(a.cfg.VisFPS, 5)),
		Playing:  st.Playing,
		Words:    a.words,
		Level:    a.level,
	}
	vis.Registry[a.visIdx].Draw(a.canvas, f)
	a.blit(a.canvas.Cells, cw, vh, x0, top+1, a.th.Text)

	// title row
	title := a.cur.Episode.Title
	a.puts(x0+1, top, fit(title, cw-2), a.st(a.th.Text).Bold(true), cw-2)
	name := vis.Registry[a.visIdx].Name()
	mode := "acoustic"
	if a.tracker.UsingTranscript() {
		mode = "transcript"
	}
	tag := fmt.Sprintf("%s · voices %s", name, mode)
	if len([]rune(title))+len(tag)+4 < cw {
		a.puts(x0+cw-len(tag)-1, top, tag, a.st(a.th.Muted), cw)
	}
	// names row
	if namesRow == 1 {
		a.drawNames(x0, top+1+vh, cw)
	}
}

// drawNames lists every voice with a marker on the one speaking.
func (a *App) drawNames(x0, y, w int) {
	orbs := a.tracker.Visible()
	x := x0 + 1
	if len(orbs) == 0 {
		a.puts(x, y, "voices appear here as people speak", a.st(a.th.Select), w-2)
		return
	}
	for i, o := range orbs {
		col := a.th.Voice(i)
		mark := "○ "
		style := a.st(paint.Mix(col, a.th.Select, 0.45))
		if o.Active {
			mark = "● "
			style = a.st(paint.Mix(col, paint.White, 0.25)).Bold(true)
		} else if o.Level > 0.05 {
			style = a.st(col)
		}
		s := mark + o.Name
		if x+len([]rune(s))+3 > x0+w {
			a.puts(x, y, "…", a.st(a.th.Muted), 2)
			break
		}
		a.puts(x, y, s, style, w)
		x += len([]rune(s)) + 3
		_ = i
	}
	// isolation summary at the right
	sum := fmt.Sprintf("sens %.2f · smooth %.2f · falloff %.3f", a.tracker.Opts.Sensitivity, a.tracker.Opts.Smoothing, a.tracker.Opts.Falloff)
	if x+len(sum)+2 < x0+w {
		a.puts(x0+w-len(sum)-1, y, sum, a.st(a.th.Select), w)
	}
}

// ---------------------------------------------------------------------------
// Captions pane

func (a *App) drawCaptions(x, y, w, h int, st audio.Status) {
	title := "captions"
	if a.tr != nil {
		title = "captions · " + a.tr.Source
	}
	a.drawBox(x, y, w, h, a.th.Select, title)
	ix, iy, iw, ih := x+1, y+1, w-2, h-2
	if iw < 10 || ih < 3 {
		return
	}
	if a.cur == nil {
		a.puts(ix+1, iy+1, "captions and animated emoji appear here", a.st(a.th.Muted), iw-2)
		return
	}
	// emoji panel at the top
	ey := iy
	if a.cfg.Emoji {
		ew := min(iw-2, 26)
		eh := ew / 2
		if eh > ih/2 {
			eh = ih / 2
			ew = eh * 2
		}
		if eh >= 4 {
			ex := ix + (iw-ew)/2
			if cv := a.emoji.Render(ew, eh, a.cfg.EmojiStyle, a.cfg.EmojiColour, a.th.Palette(), !st.Playing); cv != nil {
				a.blit(cv.Cells, ew, eh, ex, ey, a.th.Text)
				lbl := "“" + a.emoji.Word() + "”"
				if d := a.emoji.Def(); d != nil {
					lbl += "  " + d.Emoji
				}
				a.puts(ix+(iw-len([]rune(lbl)))/2, ey+eh, lbl, a.st(a.th.Muted), iw)
			} else {
				hint := "emoji animate when a spoken word matches one"
				a.puts(ix+(iw-len(hint))/2, ey+eh/2, fit(hint, iw), a.st(a.th.Select), iw)
			}
			ey += eh + 2
		}
	}
	// footer: chapter / progress
	bottom := iy + ih
	if a.trStatus == "transcribing" {
		msg := "whisper: " + a.trProgress
		a.puts(ix+1, bottom-1, fit(msg, iw-2), a.st(a.th.Secondary), iw-2)
		bottom--
	}
	if len(a.chapters) > 0 {
		if ci := transcript.ChapterAt(a.chapters, st.Position); ci >= 0 {
			ch := fmt.Sprintf("chapter %d/%d · %s", ci+1, len(a.chapters), a.chapters[ci].Title)
			a.puts(ix+1, bottom-1, fit(ch, iw-2), a.st(a.th.Tertiary), iw-2)
			bottom--
		}
	}
	// caption text
	if a.tr == nil {
		var lines []string
		switch a.trStatus {
		case "loading":
			lines = []string{"looking for a transcript…"}
		case "transcribing":
			lines = []string{"transcribing with whisper.cpp…", "", "captions appear when it finishes;", "keep listening in the meantime."}
		default:
			lines = []string{"no transcript for this episode.", ""}
			if transcript.FindWhisper(a.cfg.WhisperBin) != "" {
				if a.cur.Downloaded() {
					lines = append(lines, "press T to transcribe it locally", "with whisper.cpp ("+a.cfg.WhisperModel+").")
				} else {
					lines = append(lines, "press d to download it, then T to", "transcribe locally with whisper.cpp.")
				}
			} else {
				lines = append(lines, "install whisper.cpp to generate one:", "  brew install whisper-cpp", "then press T while an episode is", "downloaded.")
			}
		}
		for i, l := range lines {
			if ey+i >= bottom {
				break
			}
			a.puts(ix+1, ey+i, fit(l, iw-2), a.st(a.th.Muted), iw-2)
		}
		return
	}
	a.drawCaptionText(ix+1, ey, iw-2, bottom-ey, st.Position)
}

// drawCaptionText shows the current segment with the spoken word lit, a few
// past lines above and the next line below.
func (a *App) drawCaptionText(x, y, w, h int, pos float64) {
	if h < 1 {
		return
	}
	cur, in := a.tr.At(pos)
	if cur < 0 {
		cur = 0
	}
	orbCol := func(name string) paint.RGB {
		for i, o := range a.tracker.Visible() {
			if strings.EqualFold(o.Name, name) {
				return a.th.Voice(i)
			}
		}
		// colour by speaker index in transcript
		for i, s := range a.tr.Speakers {
			if s == name {
				return a.th.Voice(i)
			}
		}
		return a.th.Secondary
	}
	type row struct {
		spans   captions.Line
		seg     int
		speaker string
	}
	var rows []row
	build := func(si int) []row {
		s := a.tr.Segments[si]
		words := make([]string, len(s.Words))
		for i, wd := range s.Words {
			words[i] = wd.Text
		}
		prefix := ""
		if s.Speaker != "" && (si == 0 || a.tr.Segments[si-1].Speaker != s.Speaker) {
			prefix = s.Speaker + ":"
		}
		var out []row
		for _, l := range captions.Wrap(prefix, words, w) {
			out = append(out, row{spans: l, seg: si, speaker: s.Speaker})
		}
		return out
	}
	// current segment rows
	curRows := build(cur)
	// past rows fill the space above so the current segment sits ~60% down
	want := h*3/5 - len(curRows)
	var past []row
	for si := cur - 1; si >= 0 && len(past) < want && cur-si < 8; si-- {
		past = append(build(si), past...)
	}
	if len(past) > want && want > 0 {
		past = past[len(past)-want:]
	} else if want <= 0 {
		past = nil
	}
	rows = append(rows, past...)
	curStart := len(rows)
	rows = append(rows, curRows...)
	for si := cur + 1; si < len(a.tr.Segments) && len(rows) < h; si++ {
		rows = append(rows, build(si)...)
	}
	if len(rows) > h {
		rows = rows[:h]
	}
	wordIdx := a.tr.WordAt(cur, pos)
	for i, r := range rows {
		cx := x
		for _, sp := range r.spans {
			var style tcell.Style
			switch {
			case sp.Word < 0:
				style = a.st(orbCol(r.speaker)).Bold(true)
			case r.seg < cur:
				style = a.st(paint.Mix(a.th.Text, a.th.Select, 0.55))
			case r.seg > cur:
				style = a.st(a.th.Muted)
			case in && sp.Word == wordIdx:
				style = a.st(paint.Mix(orbCol(r.speaker), paint.White, 0.5)).Bold(true)
			case sp.Word < wordIdx || !in:
				style = a.st(a.th.Text)
			default:
				style = a.st(paint.Mix(a.th.Text, a.th.Muted, 0.5))
			}
			if i >= curStart && i < curStart+len(curRows) && sp.Word < 0 {
				style = style.Underline(true)
			}
			cx += a.puts(cx, y+i, sp.Text, style, x+w-cx)
		}
	}
}

// ---------------------------------------------------------------------------
// Progress and status rows

func (a *App) drawProgress(w, y int, st audio.Status) {
	left := fmtTime(st.Position)
	right := fmtTime(st.Duration)
	if st.Track == nil {
		left, right = "0:00", "0:00"
	}
	if st.Duration > 0 && st.Track != nil {
		right = "-" + fmtTime(st.Duration-st.Position) + " / " + fmtTime(st.Duration)
	}
	x0 := len(left) + 2
	x1 := w - len(right) - 2
	a.puts(1, y, left, a.st(a.th.Muted), w)
	a.puts(w-len(right)-1, y, right, a.st(a.th.Muted), w)
	a.mouseSeekX0, a.mouseSeekX1 = x0, x1
	bw := x1 - x0
	if bw < 1 {
		return
	}
	frac := 0.0
	if st.Duration > 0 {
		frac = st.Position / st.Duration
	}
	col := a.th.Accent
	if o := a.tracker.Current(); o != nil {
		for i, v := range a.tracker.Visible() {
			if v == o {
				col = a.th.Voice(i)
			}
		}
	}
	a.progressBar(x0, y, bw, frac, col)
	// chapter ticks
	if len(a.chapters) > 0 && st.Duration > 0 {
		for _, ch := range a.chapters {
			cx := x0 + int(ch.Start/st.Duration*float64(bw))
			if cx >= x0 && cx < x1 {
				a.scr.SetContent(cx, y, '┼', nil, a.st(a.th.Muted))
			}
		}
	}
}

func (a *App) drawStatus(w, y int, st audio.Status) {
	var state string
	switch {
	case st.Track == nil:
		state = "■ stopped"
	case st.Loading:
		state = "◌ loading"
	case st.Paused:
		state = "⏸ paused"
	case st.Playing:
		state = "▶ playing"
	default:
		state = "◌ buffering"
	}
	vol := fmt.Sprintf("vol %d%%", int(st.Volume*100+0.5))
	if st.Muted {
		vol = "muted"
	}
	speed := fmt.Sprintf("%.2fx", st.Speed)
	src := ""
	if st.Track != nil {
		if st.Stream {
			src = "streaming"
			if a.cur != nil {
				if j := a.dl.JobFor(a.cur.Key()); j != nil && j.State < 2 {
					src = fmt.Sprintf("streaming · downloading %d%%", int(j.Progress()*100))
				}
			}
		} else {
			src = "downloaded"
		}
	}
	parts := []string{state, speed, vol}
	if src != "" {
		parts = append(parts, src)
	}
	if a.cur != nil {
		parts = append(parts, a.cur.Podcast.Title)
	}
	x := 1
	for i, p := range parts {
		style := a.st(a.th.Muted)
		if i == 0 {
			style = a.st(a.th.Accent)
		}
		x += a.puts(x, y, p, style, w-x-1)
		if i < len(parts)-1 {
			x += a.puts(x, y, " · ", a.st(a.th.Select), w-x-1)
		}
	}
	right := fmt.Sprintf("theme %s · %s", a.th.Name, vis.Registry[a.visIdx].Name())
	if q := len(a.lib.Queue); q > 0 {
		right = fmt.Sprintf("queue %d · ", q) + right
	}
	if x+len(right)+2 < w {
		a.puts(w-len(right)-1, y, right, a.st(a.th.Muted), w)
	}
}

// drawKitty paints the selected podcast's icon with the Kitty protocol.
func (a *App) drawKitty(w, h int) {
	if a.cfg.IconMode != "kitty" || a.view != ViewPodcasts || a.pv.kittyBox[2] == 0 {
		return
	}
	p := a.pv.selectedPodcast(a)
	ic := a.icon(p)
	if ic == nil {
		return
	}
	box := a.pv.kittyBox
	if a.kittyDrawn {
		return
	}
	fmt.Fprintf(os.Stdout, "\x1b[%d;%dH", box[1]+1, box[0]+1)
	os.Stdout.WriteString(art.KittyImage(ic.Image, box[2], box[3], a.kittyID))
	a.kittyDrawn = true
}
