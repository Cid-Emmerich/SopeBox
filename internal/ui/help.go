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
		"# Playback",
		"space / enter\tplay or pause",
		"← / →\tskip back / forward (shift: 4x further)",
		"n / b\tnext / previous episode in the queue",
		"s / S\tfaster / slower (pitch is preserved)",
		"↑ ↓  + -\tvolume (now-playing view)",
		"m\tmute",
		"d\tdownload the playing episode",
		"o\tjump to the playing episode in the podcast list",
		"",
		"# Views",
		"1 … 6\tnow playing / podcasts / queue / downloads / search / settings",
		"tab / shift+tab\tnext / previous view",
		"esc\tback to now playing",
		"ctrl+k  ?\tthis help",
		"q  ctrl+c\tquit (settings, positions and learned voices are saved)",
		"",
		"# Voices (now-playing view)",
		"{ / }\tvoice sensitivity: lower merges similar voices, higher splits them into more orbs",
		"( / )\tvoice smoothing: how much each voice fingerprint is averaged",
		"< / >\tvoice falloff: how quickly a silent orb dims",
		"M\tisolation mode: auto → transcript → acoustic",
		"N\tname the voice that is speaking (remembered for this podcast)",
		"B\tname it from the people declared in the feed",
		"K\tmerge the speaking voice into the previous one",
		"X\tclear the voices found in this episode",
		"F\tforget every learned voice for this podcast",
		"L\tspeaker names under the visualizer on / off",
		"",
		"# Visualizer",
		"v / V\tnext / previous style: orbs, constellation, halo, ribbon, wordflow, pulse, talktime, bars, collage",
		"t / T\tnext / previous colour theme ('match' follows the podcast icon)",
		"g / G\tgradient: theme, horizontal, rainbow, spectrum, fire, ice, neon, heat, mono, pastel, matrix",
		"i\torb fill: braille, dots, rings, petals, ascii, block",
		"x\tpeak dots on / off",
		"p\torb physics (drift and repel) on / off",
		"w / W\trotation faster / slower",
		"e / E\torbs bigger / smaller",
		", / .\tsmoothing less / more",
		"[ / ]\tgain lower / higher",
		"; / '\tfalloff slower / faster",
		"R\treset visualizer and voice tuning",
		"",
		"# Collage (pictures of who and what is mentioned)",
		"P\tread the transcript with Claude again (now-playing view)",
		"P\tprepare collages for this podcast's new episodes in the background (podcast list)",
		"",
		"# Captions",
		"c\tcaptions pane on / off",
		"C\tcaptions on the right / at the bottom",
		"u / U\tcaptions pane narrower / wider",
		"j\tanimated emoji on / off",
		"J\temoji style: blocks, braille, ascii, chunky",
		"k\temoji colour: emoji, theme, rainbow, fire, ice, neon, matrix, mono",
		"T\ttranscribe the (downloaded) episode locally with whisper.cpp",
		"r\treload captions from the feed",
		"",
		"# Podcasts",
		"↑ ↓  j k\tmove",
		"← →  h l\tpodcasts pane / episodes pane (l again cycles: podcasts → all episodes → downloaded)",
		"enter\tplay",
		"e / E\tadd to the queue / play next",
		"d\tdownload (on a podcast: its 3 newest episodes)",
		"D  delete\tdelete a download (or cancel one in progress)",
		"m\tmark played / unplayed",
		"/\tfilter episodes by words (esc clears)",
		"a\tadd a podcast by feed URL",
		"x\tunsubscribe (asks for confirmation)",
		"r / R\trefresh this podcast / every podcast",
		"i / I\tfind an icon online (iTunes) / reload the icon from the feed",
		"A\tauto-download for this podcast: global → off → 1 → 3 → 5 → 10",
		"P\tcollage prep on / off: new downloads are transcribed and read by Claude",
		"c / C\ticons on / off · icon style: blocks, ascii, kitty",
		"o\tjump to the playing episode",
		"",
		"# Queue",
		"enter\tplay",
		"x  delete\tremove",
		"J / K\tmove the episode down / up",
		"C\tclear the queue",
		"",
		"# Downloads",
		"x\tcancel",
		"C\tclear finished",
		"a\tapply the auto-download policy now",
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
