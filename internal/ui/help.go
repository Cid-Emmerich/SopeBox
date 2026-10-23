package ui

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"

	"github.com/Cid-Emmerich/SopeBox/internal/captions"
	"github.com/Cid-Emmerich/SopeBox/internal/glyph"
)

// helpLines is the single source of truth for the ctrl+k help menu.
// Lines starting with "# " are section headers; others are "keys<TAB>description".
func helpLines() []string {
	return []string{
		"# Everywhere",
		"space\tplay or pause",
		"1 … 6\tnow playing / podcasts / queue / downloads / search / settings",
		"tab / shift+tab\tnext / previous view",
		"↑ ↓  ← →  enter\tmove, change and choose",
		"esc\tback to now playing",
		"ctrl+k  ?\tthis help",
		"q  ctrl+c\tquit (settings, positions and learned voices are saved)",
		"",
		"# Now playing",
		"space / enter\tplay or pause",
		"← / →\tskip back / forward (shift: 4x further)",
		"↑ / ↓\tvolume",
		"v\tnext visualizer: constellation, wordflow, talktime, bars",
		"t\tnext colour theme ('match' follows the podcast icon)",
		"c\tcaptions pane on / off",
		"{ / }\tvoice sensitivity: lower merges similar voices, higher splits them into more orbs",
		"N\tname the voice that is speaking (remembered for this podcast)",
		"d\tdownload the playing episode",
		"T\ttranscribe the (downloaded) episode locally with whisper.cpp",
		"",
		"# Podcasts",
		"← →\tpodcasts pane / episodes pane",
		"enter\tplay",
		"e\tadd to the queue",
		"d\tdownload (on a podcast: its 3 newest episodes)",
		"D  delete\tdelete a download (or cancel one in progress)",
		"m\tmark played / unplayed",
		"a\tadd a podcast by feed URL",
		"r\trefresh this podcast",
		"i\tfind an icon online (iTunes)",
		"/\tfilter episodes by words (esc clears)",
		"A\tauto-download for this podcast: global → off → 1 → 3 → 5 → 10",
		"x\tunsubscribe (asks for confirmation)",
		"",
		"# Queue",
		"enter\tplay",
		"x  delete\tremove",
		"J / K\tmove the episode down / up",
		"C\tclear the queue",
		"",
		"# Downloads",
		"enter\tplay",
		"x\tcancel",
		"C\tclear finished",
		"",
		"# Settings (6)",
		"↑ / ↓\tchoose a setting",
		"← / →\tchange it",
		"enter\tchange it, or run an action (next episode, merge voice, reload captions, …)",
		"\tspeed, mute, the visualizer and voice tuning, captions, emoji, and icons all live here",
		"",
		"# Command line",
		"sopebox\topen the player",
		"sopebox add <feed url>\tsubscribe",
		"sopebox search <words>\tsearch iTunes and pick a podcast to subscribe to",
		"sopebox import castero\tbring in castero subscriptions (also: import opml <file>)",
		"sopebox export <file.opml>\tsave subscriptions",
		"sopebox play <words>\tplay the best matching episode",
		"sopebox download [podcast] [n]\tdownload the n newest episodes",
		"sopebox transcribe <words>\ttranscribe a downloaded episode with whisper.cpp",
		"sopebox refresh\trefresh every feed",
		"sopebox path <dir>\tset the download folder",
		"sopebox -t <theme> -v <style>\tstart with a theme and a visualizer",
		"",
		fmt.Sprintf("mouse\tclick tabs, click the progress bar to seek, scroll lists · %d trigger words animate %d emoji", captions.TriggerCount(), len(glyph.Registry)),
	}
}

func (a *App) drawHelp(w, h int) {
	bw := min(w-4, 96)
	bh := h - 2
	x0 := (w - bw) / 2
	y0 := 1
	bg := tcell.StyleDefault.Background(tc(a.th.Select))
	for y := y0; y < y0+bh; y++ {
		a.fillRow(y, x0, x0+bw, bg)
	}
	a.drawBox(x0, y0, bw, bh, a.th.Accent, "SopeBox keys (↑↓ scroll, esc close)")
	lines := helpLines()
	rows := bh - 2
	if a.helpScroll > len(lines)-rows {
		a.helpScroll = max(0, len(lines)-rows)
	}
	keyW := 30
	for i := 0; i < rows && a.helpScroll+i < len(lines); i++ {
		l := lines[a.helpScroll+i]
		y := y0 + 1 + i
		if strings.HasPrefix(l, "# ") {
			a.puts(x0+2, y, l[2:], bg.Foreground(tc(a.th.Accent)).Bold(true), bw-4)
			continue
		}
		k, d, _ := strings.Cut(l, "\t")
		a.puts(x0+2, y, fit(k, keyW), bg.Foreground(tc(a.th.Text)), keyW)
		a.puts(x0+2+keyW, y, fit(d, bw-4-keyW), bg.Foreground(tc(a.th.Muted)), bw-4-keyW)
	}
}
