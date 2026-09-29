// Command sopebox is a terminal podcast player with a voice-aware
// visualizer: every speaker gets their own radial spectrum orb, and the
// captions pane animates an emoji whenever a fitting word is spoken.
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Cid-Emmerich/SopeBox/internal/art"
	"github.com/Cid-Emmerich/SopeBox/internal/audio"
	"github.com/Cid-Emmerich/SopeBox/internal/config"
	"github.com/Cid-Emmerich/SopeBox/internal/download"
	"github.com/Cid-Emmerich/SopeBox/internal/feed"
	"github.com/Cid-Emmerich/SopeBox/internal/mentions"
	"github.com/Cid-Emmerich/SopeBox/internal/store"
	"github.com/Cid-Emmerich/SopeBox/internal/theme"
	"github.com/Cid-Emmerich/SopeBox/internal/transcript"
	"github.com/Cid-Emmerich/SopeBox/internal/ui"
	"github.com/Cid-Emmerich/SopeBox/internal/vis"
)

func usage() {
	fmt.Println(`sopebox — a terminal podcast player with a voice-aware visualizer

usage:
  sopebox                         open the player
  sopebox add <feed url>          subscribe to a podcast
  sopebox search <words>          search iTunes and pick a podcast to add
  sopebox import castero          import subscriptions from castero
  sopebox import opml <file>      import an OPML file
  sopebox export <file.opml>      export subscriptions
  sopebox refresh                 refresh every feed
  sopebox list                    list subscriptions
  sopebox play <words>            play the best matching episode
  sopebox download [podcast] [n]  download the n newest episodes (default 3)
  sopebox transcribe <words>      transcribe a downloaded episode with whisper.cpp
  sopebox collage <words>         find who and what an episode mentions (Claude) and
                                  fetch their pictures, ready for the collage style
                                  (reuses what Claude found before; --fresh asks again)
  sopebox path <dir>              set the download folder
  sopebox config                  print the config file location

options:
  -t <theme>   start with a theme: ` + strings.Join(theme.Names(), ", ") + `
  -v <style>   start with a visualizer: ` + strings.Join(vis.Names(), ", ") + `
  -n           don't refresh feeds on start`)
}

func die(err error) {
	fmt.Fprintln(os.Stderr, "sopebox:", err)
	os.Exit(1)
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "sopebox: config:", err)
	}
	args := os.Args[1:]
	var rest []string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-t", "--theme":
			if i+1 < len(args) {
				cfg.Theme = args[i+1]
				i++
			}
		case "-v", "--vis":
			if i+1 < len(args) {
				cfg.Vis = args[i+1]
				i++
			}
		case "-n", "--no-refresh":
			cfg.RefreshStart = false
		case "-h", "--help", "help":
			usage()
			return
		default:
			rest = append(rest, args[i])
		}
	}
	lib, err := store.Open(cfg.LibraryPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "sopebox: library:", err)
	}
	if len(rest) > 0 {
		if done := command(&cfg, lib, rest); done {
			return
		}
	}
	runUI(&cfg, lib, rest)
}

func runUI(cfg *config.Config, lib *store.Library, rest []string) {
	if !audio.HaveFFmpeg() {
		die(fmt.Errorf("ffmpeg is required (brew install ffmpeg)"))
	}
	pl := audio.New()
	if err := pl.Start(); err != nil {
		die(fmt.Errorf("audio: %w", err))
	}
	defer pl.Close()
	pl.SetVolume(cfg.Volume)
	pl.SetSpeed(cfg.Speed)
	app := ui.New(cfg, lib, pl)
	start := ui.ViewNow
	if len(lib.Podcasts) == 0 {
		start = ui.ViewSearch
	} else if lib.Last == nil {
		start = ui.ViewPodcasts
	}
	if err := app.Run(start); err != nil {
		die(err)
	}
}

// command handles subcommands; returns true when the program should exit.
func command(cfg *config.Config, lib *store.Library, args []string) bool {
	cmd, rest := args[0], args[1:]
	switch cmd {
	case "path":
		if len(rest) == 0 {
			fmt.Println(cfg.DownloadDir)
			return true
		}
		p, _ := filepath.Abs(config.ExpandHome(rest[0]))
		cfg.DownloadDir = p
		if err := cfg.Save(); err != nil {
			die(err)
		}
		fmt.Println("downloads will go to", p)
		return true
	case "config":
		fmt.Println(cfg.ConfigPath)
		return true
	case "add":
		if len(rest) == 0 {
			die(fmt.Errorf("usage: sopebox add <feed url>"))
		}
		for _, u := range rest {
			p, err := lib.Subscribe(u)
			if err != nil {
				fmt.Fprintln(os.Stderr, "sopebox:", u+":", err)
				continue
			}
			fmt.Printf("subscribed to %s (%d episodes)\n", p.Title, len(p.Episodes))
		}
		must(lib.Save())
		return true
	case "search":
		q := strings.Join(rest, " ")
		res, err := art.SearchPodcasts(q, 15)
		if err != nil {
			die(err)
		}
		for i, r := range res {
			fmt.Printf("%2d. %s — %s (%d episodes)\n    %s\n", i+1, r.Title, r.Author, r.Count, r.FeedURL)
		}
		fmt.Print("subscribe to which number? (enter to skip) ")
		var n int
		if _, err := fmt.Scanln(&n); err == nil && n >= 1 && n <= len(res) {
			p, err := lib.Subscribe(res[n-1].FeedURL)
			if err != nil {
				die(err)
			}
			must(lib.Save())
			fmt.Printf("subscribed to %s (%d episodes)\n", p.Title, len(p.Episodes))
		}
		return true
	case "import":
		var subs []feed.Sub
		var err error
		switch {
		case len(rest) == 0 || rest[0] == "castero":
			db := ""
			if len(rest) > 1 {
				db = rest[1]
			}
			subs, err = feed.ImportCastero(db)
		case rest[0] == "opml":
			if len(rest) < 2 {
				die(fmt.Errorf("usage: sopebox import opml <file>"))
			}
			subs, err = feed.ImportOPML(rest[1])
		default:
			subs, err = feed.ImportOPML(rest[0])
		}
		if err != nil {
			die(err)
		}
		n := 0
		for _, s := range subs {
			if lib.Find(s.URL) == nil {
				lib.AddPlaceholder(s.URL, s.Title)
				n++
			}
		}
		must(lib.Save())
		fmt.Printf("imported %d subscriptions (%d new); fetching feeds…\n", len(subs), n)
		refresh(lib)
		return true
	case "export":
		if len(rest) == 0 {
			die(fmt.Errorf("usage: sopebox export <file.opml>"))
		}
		var subs []feed.Sub
		for _, p := range lib.Sorted() {
			subs = append(subs, feed.Sub{Title: p.Title, URL: p.URL})
		}
		must(feed.ExportOPML(rest[0], subs))
		fmt.Printf("wrote %d subscriptions to %s\n", len(subs), rest[0])
		return true
	case "refresh":
		refresh(lib)
		return true
	case "list", "ls":
		for _, p := range lib.Sorted() {
			fmt.Printf("%-40s %5d episodes  %s\n", fitStr(p.Title, 40), len(p.Episodes), p.URL)
		}
		return true
	case "download", "dl":
		n := 3
		var words []string
		for _, r := range rest {
			var k int
			if _, err := fmt.Sscanf(r, "%d", &k); err == nil {
				n = k
			} else {
				words = append(words, r)
			}
		}
		pods := lib.Sorted()
		if len(words) > 0 {
			q := strings.ToLower(strings.Join(words, " "))
			var m []*store.Podcast
			for _, p := range pods {
				if strings.Contains(strings.ToLower(p.Title), q) {
					m = append(m, p)
				}
			}
			pods = m
		}
		if len(pods) == 0 {
			die(fmt.Errorf("no matching podcast"))
		}
		dl := download.New(lib, cfg.DownloadDir, cfg.Concurrency, nil)
		queued := 0
		for _, p := range pods {
			for i, it := range lib.Items(p) {
				if i >= n {
					break
				}
				if err := dl.Enqueue(it, false); err == nil {
					queued++
					fmt.Println("queued:", p.Title, "›", it.Episode.Title)
				}
			}
		}
		if queued == 0 {
			fmt.Println("nothing to download")
			return true
		}
		for dl.Active() > 0 {
			time.Sleep(300 * time.Millisecond)
			var running []string
			for _, j := range dl.Jobs() {
				if j.State == download.Running {
					if j.Total > 0 {
						running = append(running, fmt.Sprintf("%s %d%%", fitStr(j.Title, 30), int(j.Progress()*100)))
					} else {
						running = append(running, fitStr(j.Title, 30))
					}
				}
			}
			fmt.Printf("\r\x1b[K%s", strings.Join(running, " · "))
		}
		fmt.Println()
		for _, j := range dl.Jobs() {
			fmt.Printf("%-12s %s\n", j.State.String(), j.Path)
		}
		must(lib.Save())
		return true
	case "play":
		// handled by the UI: leave the words for it
		return false
	case "transcribe":
		q := strings.ToLower(strings.Join(rest, " "))
		var target *store.Item
		for _, it := range lib.DownloadedItems() {
			if strings.Contains(strings.ToLower(it.Podcast.Title+" "+it.Episode.Title), q) {
				target = &it
				break
			}
		}
		if target == nil {
			die(fmt.Errorf("no downloaded episode matches %q", q))
		}
		fmt.Println("transcribing:", target.Podcast.Title, "›", target.Episode.Title)
		t, err := transcript.Transcribe(target.State.Path, transcript.Options{Bin: cfg.WhisperBin, Model: cfg.WhisperModel, CacheDir: cfg.CacheDir,
			Progress: func(ph string, f float64) { fmt.Printf("\r%s %3d%%", ph, int(f*100)) }})
		fmt.Println()
		if err != nil {
			die(err)
		}
		must(transcript.SaveCached(transcript.CachePath(cfg.CacheDir, target.Podcast.URL+"|"+target.Episode.GUID), t))
		fmt.Printf("done: %d caption segments\n", len(t.Segments))
		return true
	case "collage":
		collage(cfg, lib, rest)
		return true
	}
	// unknown word: treat as "play <words>"
	return false
}

func refresh(lib *store.Library) {
	pods := lib.Sorted()
	type res struct {
		p   *store.Podcast
		n   int
		err error
	}
	out := make(chan res, len(pods))
	sem := make(chan struct{}, 4)
	for _, p := range pods {
		sem <- struct{}{}
		go func(p *store.Podcast) {
			defer func() { <-sem }()
			n, err := lib.Refresh(p)
			out <- res{p, len(n), err}
		}(p)
	}
	var rows []res
	for range pods {
		rows = append(rows, <-out)
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].p.Title < rows[j].p.Title })
	for _, r := range rows {
		if r.err != nil {
			fmt.Printf("  ✗ %-40s %v\n", fitStr(r.p.Title, 40), r.err)
		} else {
			fmt.Printf("  ✓ %-40s %d new\n", fitStr(r.p.Title, 40), r.n)
		}
	}
	must(lib.Save())
}

func fitStr(s string, w int) string {
	r := []rune(s)
	if len(r) <= w {
		return s
	}
	return string(r[:w-1]) + "…"
}

func must(err error) {
	if err != nil {
		die(err)
	}
}

// collage prepares an episode for the collage visualizer: transcript
// (cached, from the feed, or whisper), then Claude, then pictures.
func collage(cfg *config.Config, lib *store.Library, args []string) {
	fresh := false
	var words []string
	for _, a := range args {
		if a == "--fresh" {
			fresh = true
		} else {
			words = append(words, a)
		}
	}
	q := strings.ToLower(strings.Join(words, " "))
	if q == "" {
		die(fmt.Errorf("usage: sopebox collage <episode words>"))
	}
	var target *store.Item
	for _, p := range lib.Sorted() {
		for _, it := range lib.Items(p) {
			if strings.Contains(strings.ToLower(p.Title+" "+it.Episode.Title), q) {
				target = &it
				break
			}
		}
		if target != nil {
			break
		}
	}
	if target == nil {
		die(fmt.Errorf("no episode matches %q", q))
	}
	it := *target
	key := os.Getenv("ANTHROPIC_API_KEY")
	if key == "" {
		key = cfg.AnthropicKey
	}
	fmt.Println("episode:", it.Podcast.Title, "›", it.Episode.Title)
	ck := it.Podcast.URL + "|" + it.Episode.GUID
	tlPath := mentions.CachePath(cfg.CacheDir, ck)
	cached, cacheErr := mentions.LoadCached(tlPath)
	if (fresh || cacheErr != nil) && !mentions.HaveCredentials(key) {
		die(fmt.Errorf("set ANTHROPIC_API_KEY (or anthropic_api_key in %s) first", cfg.ConfigPath))
	}
	trPath := transcript.CachePath(cfg.CacheDir, ck)
	tr, err := transcript.LoadCached(trPath)
	if err != nil || len(tr.Segments) == 0 {
		if tr, err = transcript.FromFeed(it.Episode); err != nil {
			if !it.Downloaded() {
				die(fmt.Errorf("no transcript in the feed; download the episode first (sopebox download) so whisper can make one"))
			}
			fmt.Println("transcribing with whisper.cpp…")
			tr, err = transcript.Transcribe(it.State.Path, transcript.Options{Bin: cfg.WhisperBin, Model: cfg.WhisperModel, CacheDir: cfg.CacheDir,
				Progress: func(ph string, f float64) { fmt.Printf("\r%s %3d%%", ph, int(f*100)) }})
			fmt.Println()
			if err != nil {
				die(err)
			}
		}
		must(transcript.SaveCached(trPath, tr))
	}
	fmt.Printf("transcript: %s, %d lines\n", tr.Source, len(tr.Segments))
	if !fresh && cacheErr == nil {
		cached.Rebuild(tr)
		must(mentions.SaveCached(tlPath, cached))
		fmt.Printf("reusing what %s found before (no charge; --fresh asks again): %d mentions of %d people, places and things\n",
			cached.Model, len(cached.Mentions), cached.Distinct())
		printTimeline(cached)
		prefetch(cfg, cached)
		return
	}
	fmt.Printf("asking %s who and what is mentioned…\n", cfg.CollageModel)
	start := time.Now()
	info := mentions.Episode{Podcast: it.Podcast.Title, Title: it.Episode.Title, Description: it.Episode.Description}
	for _, p := range append(it.Episode.Persons, it.Podcast.Persons...) {
		info.Hosts = append(info.Hosts, p.Name)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()
	tl, err := mentions.Extract(ctx, key, cfg.CollageModel, info, tr)
	if err != nil {
		die(err)
	}
	must(mentions.SaveCached(tlPath, tl))
	fmt.Printf("%d mentions of %d people, places and things in %s (%d tokens in, %d out)\n",
		len(tl.Mentions), tl.Distinct(), time.Since(start).Round(time.Second), tl.InputTokens, tl.OutputTokens)
	printTimeline(tl)
	prefetch(cfg, tl)
}

func printTimeline(tl *mentions.Timeline) {
	for _, m := range tl.Mentions {
		wiki := m.Wiki
		if wiki == "" {
			wiki = "—"
		}
		fmt.Printf("  %8s  %-7s %-34s %s\n", fmtClock(m.At), m.Kind, fitStr(m.Name, 34), wiki)
	}
}

func prefetch(cfg *config.Config, tl *mentions.Timeline) {
	fmt.Println("fetching pictures from Wikipedia…")
	found, pics := mentions.Prefetch(cfg.CacheDir, tl, nil)
	fmt.Printf("done: %d articles, %d pictures. Play it and press v until the style is collage.\n", found, pics)
}

func fmtClock(s float64) string {
	t := int(s)
	if t >= 3600 {
		return fmt.Sprintf("%d:%02d:%02d", t/3600, t/60%60, t%60)
	}
	return fmt.Sprintf("%d:%02d", t/60, t%60)
}
