package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"

	"github.com/Cid-Emmerich/SopeBox/internal/paint"
)

func tc(c paint.RGB) tcell.Color { return tcell.NewRGBColor(int32(c.R), int32(c.G), int32(c.B)) }

func (a *App) st(fg paint.RGB) tcell.Style { return tcell.StyleDefault.Foreground(tc(fg)) }

// puts writes a string clipped to maxW cells and returns the width used.
func (a *App) puts(x, y int, s string, style tcell.Style, maxW int) int {
	i := 0
	for _, r := range s {
		if i >= maxW {
			break
		}
		a.scr.SetContent(x+i, y, r, nil, style)
		i++
	}
	return i
}

func fit(s string, w int) string {
	rs := []rune(s)
	if len(rs) <= w {
		return s
	}
	if w <= 1 {
		return string(rs[:max(w, 0)])
	}
	return string(rs[:w-1]) + "…"
}

func pad(s string, w int) string {
	n := len([]rune(s))
	if n >= w {
		return fit(s, w)
	}
	return s + strings.Repeat(" ", w-n)
}

func fmtTime(sec float64) string {
	if sec < 0 || sec != sec {
		sec = 0
	}
	s := int(sec + 0.5)
	if s >= 3600 {
		return fmt.Sprintf("%d:%02d:%02d", s/3600, (s%3600)/60, s%60)
	}
	return fmt.Sprintf("%d:%02d", s/60, s%60)
}

func fmtBytes(b int64) string {
	switch {
	case b >= 1<<30:
		return fmt.Sprintf("%.1f GB", float64(b)/(1<<30))
	case b >= 1<<20:
		return fmt.Sprintf("%.0f MB", float64(b)/(1<<20))
	case b >= 1<<10:
		return fmt.Sprintf("%.0f KB", float64(b)/(1<<10))
	}
	return fmt.Sprintf("%d B", b)
}

func fmtDate(t time.Time) string {
	if t.IsZero() {
		return "          "
	}
	return t.Format("2006-01-02")
}

func (a *App) fillRow(y, x0, x1 int, style tcell.Style) {
	for x := x0; x < x1; x++ {
		a.scr.SetContent(x, y, ' ', nil, style)
	}
}

// draw renders the whole screen.
func (a *App) draw() {
	a.scr.Clear()
	w, h := a.scr.Size()
	if w < 40 || h < 10 {
		a.puts(0, 0, "window too small", a.st(a.th.Warn), w)
		a.scr.Show()
		return
	}
	a.drawHeader(w)
	switch a.view {
	case ViewNow:
		a.drawNow(w, h)
	case ViewPodcasts:
		a.drawPodcasts(w, h)
	case ViewQueue:
		a.drawQueue(w, h)
	case ViewDownloads:
		a.drawDownloads(w, h)
	case ViewSearch:
		a.drawSearch(w, h)
	case ViewSettings:
		a.drawSettings(w, h)
	}
	a.drawBottomLine(w, h)
	if a.help {
		a.drawHelp(w, h)
	}
	if a.prompt.active {
		a.drawPrompt(w, h)
	}
	a.scr.Show()
	a.drawKitty(w, h)
}

// tabPositions returns [start,end) x ranges of the header tabs.
func tabPositions(w int) [][2]int {
	x := 9
	var out [][2]int
	for _, n := range tabNames {
		width := len(n) + 4
		out = append(out, [2]int{x, x + width})
		x += width + 1
	}
	return out
}

func (a *App) drawHeader(w int) {
	a.fillRow(0, 0, w, tcell.StyleDefault)
	a.puts(1, 0, "SopeBox", a.st(a.th.Accent).Bold(true), w)
	compact := w < 90
	for i, pos := range tabPositions(w) {
		style := a.st(a.th.Muted)
		if View(i) == a.view && !a.help {
			style = a.st(a.th.Text).Background(tc(a.th.Select)).Bold(true)
		}
		label := fmt.Sprintf(" %d %s ", i+1, tabNames[i])
		if compact {
			label = fmt.Sprintf(" %d ", i+1)
			pos = [2]int{9 + i*4, 12 + i*4}
		}
		a.fillRow(0, pos[0], pos[1], style)
		a.puts(pos[0], 0, label, style, pos[1]-pos[0])
	}
	right := "ctrl+k help"
	if a.refreshing {
		right = "refreshing… · " + right
	} else if a.busy != "" {
		right = a.busy + "… · " + right
	} else if n := a.dl.Active(); n > 0 {
		right = fmt.Sprintf("↓ %d · %s", n, right)
	}
	a.puts(w-len([]rune(right))-1, 0, right, a.st(a.th.Muted), w)
}

func (a *App) drawBottomLine(w, h int) {
	y := h - 1
	a.fillRow(y, 0, w, tcell.StyleDefault)
	if a.toast != "" && time.Now().Before(a.toastTill) {
		col := a.th.Accent
		if a.toastErr {
			col = a.th.Warn
		}
		a.puts(1, y, fit(a.toast, w-2), a.st(col), w-2)
		return
	}
	var hint string
	switch a.view {
	case ViewNow:
		hint = "space play/pause · ←/→ skip · v style · t theme · c captions · { } voice sensitivity · N name voice · d download · T transcribe"
	case ViewPodcasts:
		hint = "enter play · e queue · d download · a add feed · r refresh · i find icon · / filter · A auto-download · x unsubscribe"
	case ViewQueue:
		hint = "enter play · x remove · J/K move · C clear"
	case ViewDownloads:
		hint = "enter play · x cancel · C clear finished · settings: auto-download in 6"
	case ViewSearch:
		hint = "type to search iTunes · enter subscribe · esc back"
	case ViewSettings:
		hint = "↑/↓ choose · ←/→ enter change"
	}
	a.puts(1, y, fit(hint, w-2), a.st(a.th.Muted), w-2)
}

func (a *App) drawBox(x, y, w, h int, col paint.RGB, title string) {
	style := a.st(col)
	if w < 2 || h < 2 {
		return
	}
	for i := 0; i < w; i++ {
		a.scr.SetContent(x+i, y, '─', nil, style)
		a.scr.SetContent(x+i, y+h-1, '─', nil, style)
	}
	for j := 0; j < h; j++ {
		a.scr.SetContent(x, y+j, '│', nil, style)
		a.scr.SetContent(x+w-1, y+j, '│', nil, style)
	}
	a.scr.SetContent(x, y, '╭', nil, style)
	a.scr.SetContent(x+w-1, y, '╮', nil, style)
	a.scr.SetContent(x, y+h-1, '╰', nil, style)
	a.scr.SetContent(x+w-1, y+h-1, '╯', nil, style)
	if title != "" {
		a.puts(x+2, y, " "+fit(title, w-6)+" ", a.st(a.th.Muted), w-4)
	}
}

// blit copies a cell grid to the screen.
func (a *App) blit(cells []paint.Cell, cw, ch, x0, y0 int, defFg paint.RGB) {
	for y := 0; y < ch; y++ {
		for x := 0; x < cw; x++ {
			c := cells[y*cw+x]
			if c.Ch == 0 {
				continue
			}
			fg := c.Fg
			if fg == (paint.RGB{}) && !c.HasBg {
				fg = defFg
			}
			style := tcell.StyleDefault.Foreground(tc(fg))
			if c.HasBg {
				style = style.Background(tc(c.Bg))
			}
			a.scr.SetContent(x0+x, y0+y, c.Ch, nil, style)
		}
	}
}

// blitRows draws a [][]Cell grid.
func (a *App) blitRows(cells [][]paint.Cell, x0, y0 int) {
	for y, row := range cells {
		for x, c := range row {
			fg := c.Fg
			if fg == (paint.RGB{}) && !c.HasBg {
				fg = a.th.Text
			}
			style := tcell.StyleDefault.Foreground(tc(fg))
			if c.HasBg {
				style = style.Background(tc(c.Bg))
			}
			a.scr.SetContent(x0+x, y0+y, c.Ch, nil, style)
		}
	}
}

func (a *App) drawPrompt(w, h int) {
	pw := min(w-4, 70)
	ph := 5
	x := (w - pw) / 2
	y := (h - ph) / 2
	for j := 0; j < ph; j++ {
		a.fillRow(y+j, x, x+pw, tcell.StyleDefault.Background(tc(a.th.Select)))
	}
	a.drawBox(x, y, pw, ph, a.th.Accent, a.prompt.label)
	inner := tcell.StyleDefault.Background(tc(a.th.Select)).Foreground(tc(a.th.Text))
	txt := a.prompt.text
	if len([]rune(txt)) > pw-6 {
		txt = string([]rune(txt)[len([]rune(txt))-(pw-6):])
	}
	a.puts(x+2, y+2, "> "+txt+"▏", inner, pw-4)
	if a.prompt.hint != "" {
		a.puts(x+2, y+3, fit(a.prompt.hint, pw-4), tcell.StyleDefault.Background(tc(a.th.Select)).Foreground(tc(a.th.Muted)), pw-4)
	}
}

// listWindow keeps cursor visible and returns the first index to draw.
func listWindow(cursor, scroll, n, rows int) int {
	if rows < 1 {
		return 0
	}
	if cursor < scroll {
		scroll = cursor
	}
	if cursor >= scroll+rows {
		scroll = cursor - rows + 1
	}
	if scroll > n-rows {
		scroll = n - rows
	}
	if scroll < 0 {
		scroll = 0
	}
	return scroll
}

// progressBar renders a bar of width w with fraction f.
func (a *App) progressBar(x, y, w int, f float64, col paint.RGB) {
	if w < 1 {
		return
	}
	if f < 0 {
		f = 0
	}
	if f > 1 {
		f = 1
	}
	filled := int(f * float64(w))
	for i := 0; i < w; i++ {
		switch {
		case i < filled:
			a.scr.SetContent(x+i, y, '━', nil, a.st(col))
		case i == filled && f > 0 && f < 1:
			a.scr.SetContent(x+i, y, '╸', nil, a.st(col))
		default:
			a.scr.SetContent(x+i, y, '─', nil, a.st(a.th.Select))
		}
	}
}

// cellRow is one row of rendered icon cells.
type cellRow = paint.Cell
