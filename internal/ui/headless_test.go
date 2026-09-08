package ui

import (
	"os"
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"

	"github.com/Cid-Emmerich/SopeBox/internal/audio"
	"github.com/Cid-Emmerich/SopeBox/internal/config"
	"github.com/Cid-Emmerich/SopeBox/internal/store"
)

func TestHeadlessRun(t *testing.T) {
	if os.Getenv("SOPEBOX_HEADLESS") == "" {
		t.Skip("set SOPEBOX_HEADLESS=1")
	}
	cfg, _ := config.Load()
	cfg.RefreshStart = false
	lib, err := store.Open(cfg.LibraryPath)
	if err != nil {
		t.Fatal(err)
	}
	pl := audio.New()
	if err := pl.Start(); err != nil {
		t.Skip("no audio device:", err)
	}
	defer pl.Close()
	pl.SetVolume(0.05)
	a := New(&cfg, lib, pl)
	scr := tcell.NewSimulationScreen("")
	if err := scr.Init(); err != nil {
		t.Fatal(err)
	}
	scr.SetSize(150, 42)
	go func() {
		time.Sleep(300 * time.Millisecond)
		// prefer a podcast whose newest episode ships a transcript
		var target *store.Item
		for _, p := range lib.Sorted() {
			if items := lib.Items(p); len(items) > 0 && len(items[0].Episode.Transcripts) > 0 {
				target = &items[0]
				break
			}
		}
		if target == nil {
			if items := lib.DownloadedItems(); len(items) > 0 {
				target = &items[0]
			}
		}
		if target != nil {
			scr.PostEvent(tcell.NewEventInterrupt(playRequest{*target}))
		}
		time.Sleep(6 * time.Second)
		for i := 0; i < 12; i++ {
			scr.InjectKey(tcell.KeyRight, 0, 0) // skip ~6 minutes past the intro
			time.Sleep(150 * time.Millisecond)
		}
		time.Sleep(12 * time.Second)
		t.Log("\nNOW PLAYING\n" + dump(scr))
		scr.InjectKey(tcell.KeyRune, 'v', 0)
		time.Sleep(700 * time.Millisecond)
		t.Log("\nCONSTELLATION\n" + dump(scr))
		scr.InjectKey(tcell.KeyRune, 'v', 0)
		scr.InjectKey(tcell.KeyRune, 'v', 0)
		time.Sleep(700 * time.Millisecond)
		t.Log("\nRIBBON\n" + dump(scr))
		scr.InjectKey(tcell.KeyRune, '2', 0)
		time.Sleep(1500 * time.Millisecond)
		t.Log("\nPODCASTS\n" + dump(scr))
		scr.InjectKey(tcell.KeyRune, 'q', 0)
	}()
	if err := a.RunWith(scr, ViewNow); err != nil {
		t.Fatal(err)
	}
	st := pl.Status()
	t.Logf("status: pos=%.1f dur=%.1f err=%q orbs=%d tr=%v", st.Position, st.Duration, st.Error, len(a.tracker.Orbs), a.trStatus)
}
