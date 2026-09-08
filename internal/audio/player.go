// Package audio is SopeBox's playback engine: it decodes episodes (local
// files or streaming URLs) through ffmpeg, handles speed, seeking and the
// play queue, and feeds every sample to the analyzer for the visualizers.
package audio

import (
	"errors"
	"math"
	"sync"
	"time"

	"github.com/gopxl/beep/v2"
	"github.com/gopxl/beep/v2/speaker"

	"github.com/Cid-Emmerich/SopeBox/internal/dsp"
)

// SampleRate is the fixed output rate.
const SampleRate beep.SampleRate = 44100

// Track is what the player needs to know about an episode.
type Track struct {
	ID       string // library key, opaque to the player
	Title    string
	Podcast  string
	Source   string  // local path or http(s) URL
	Duration float64 // seconds if known (0 = probe)
	Start    float64 // resume position
}

// Status is a snapshot for the UI.
type Status struct {
	Track    *Track
	Playing  bool
	Paused   bool
	Loading  bool
	Position float64
	Duration float64
	Volume   float64
	Muted    bool
	Speed    float64
	Buffered float64 // seconds of audio decoded ahead (streams)
	Error    string
	Stream   bool // playing from a URL rather than a file
}

// Player owns the speaker and one decoded voice.
type Player struct {
	mu sync.Mutex

	Analyzer *dsp.Analyzer

	track   *Track
	src     *streamer
	paused  bool
	loading bool
	volume  float64
	muted   bool
	speed   float64
	lastErr string
	done    bool
	gen     int

	onEnd    func(*Track)
	onChange func()
	inited   bool
}

// New creates a player. Call Start before playing.
func New() *Player {
	return &Player{
		Analyzer: dsp.NewAnalyzer(2048, int(SampleRate)),
		volume:   0.8,
		speed:    1,
	}
}

// Start opens the audio device.
func (p *Player) Start() error {
	if err := speaker.Init(SampleRate, SampleRate.N(60*time.Millisecond)); err != nil {
		return err
	}
	p.inited = true
	speaker.Play(p)
	go p.watch()
	return nil
}

// Close stops audio.
func (p *Player) Close() {
	p.mu.Lock()
	if p.src != nil {
		p.src.Close()
		p.src = nil
	}
	p.mu.Unlock()
	if p.inited {
		speaker.Close()
	}
}

// OnEnd registers a callback fired (from a goroutine) when a track ends.
func (p *Player) OnEnd(f func(*Track)) { p.onEnd = f }

// OnChange registers a callback fired when the track or state changes.
func (p *Player) OnChange(f func()) { p.onChange = f }

func (p *Player) fire() {
	if p.onChange != nil {
		go p.onChange()
	}
}

// Stream implements beep.Streamer.
func (p *Player) Stream(samples [][2]float64) (int, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for i := range samples {
		samples[i] = [2]float64{}
	}
	if p.paused || p.src == nil || p.done {
		p.Analyzer.Push(samples)
		return len(samples), true
	}
	n, ok := p.src.Stream(samples)
	if !ok || n < len(samples) {
		if p.src.Err() != nil {
			p.lastErr = p.src.Err().Error()
		}
		if p.src.starved() {
			// network stall: keep the stream open, output silence
			for i := n; i < len(samples); i++ {
				samples[i] = [2]float64{}
			}
		} else {
			p.done = true
		}
	}
	// analyzer sees the un-attenuated mix so voice features ignore volume
	p.Analyzer.Push(samples)
	vol := p.volume * p.volume
	if p.muted {
		vol = 0
	}
	for i := range samples {
		samples[i][0] = clampSample(samples[i][0] * vol)
		samples[i][1] = clampSample(samples[i][1] * vol)
	}
	return len(samples), true
}

// Err implements beep.Streamer.
func (p *Player) Err() error { return nil }

func clampSample(x float64) float64 {
	if x > 1 {
		return 1
	}
	if x < -1 {
		return -1
	}
	return x
}

func (p *Player) watch() {
	t := time.NewTicker(100 * time.Millisecond)
	defer t.Stop()
	for range t.C {
		p.mu.Lock()
		if p.done && p.track != nil && p.src != nil {
			tr := p.track
			p.src.Close()
			p.src = nil
			p.done = false
			p.mu.Unlock()
			if p.onEnd != nil {
				p.onEnd(tr)
			}
			continue
		}
		p.mu.Unlock()
	}
}

// Play starts a track (replacing the current one).
func (p *Player) Play(t *Track) {
	p.mu.Lock()
	p.gen++
	gen := p.gen
	if p.src != nil {
		p.src.Close()
		p.src = nil
	}
	p.track = t
	p.loading = true
	p.done = false
	p.paused = false
	p.lastErr = ""
	speed := p.speed
	p.mu.Unlock()
	p.Analyzer.Clear()
	p.fire()
	go func() {
		s, err := open(t.Source, t.Start, speed, t.Duration)
		p.mu.Lock()
		defer p.mu.Unlock()
		if p.gen != gen {
			if s != nil {
				s.Close()
			}
			return
		}
		p.loading = false
		if err != nil {
			p.lastErr = err.Error()
			p.done = true
			p.fire()
			return
		}
		p.src = s
		if s.length > 0 {
			t.Duration = s.length
		}
		p.fire()
	}()
}

// Stop ends playback.
func (p *Player) Stop() {
	p.mu.Lock()
	p.gen++
	if p.src != nil {
		p.src.Close()
		p.src = nil
	}
	p.track = nil
	p.done = false
	p.loading = false
	p.mu.Unlock()
	p.Analyzer.Clear()
	p.fire()
}

// TogglePause flips pause.
func (p *Player) TogglePause() {
	p.mu.Lock()
	if p.track != nil {
		p.paused = !p.paused
	}
	p.mu.Unlock()
	p.fire()
}

// SetPaused sets the pause state.
func (p *Player) SetPaused(v bool) {
	p.mu.Lock()
	p.paused = v
	p.mu.Unlock()
	p.fire()
}

// Seek moves by delta seconds.
func (p *Player) Seek(delta float64) {
	p.mu.Lock()
	if p.src == nil {
		p.mu.Unlock()
		return
	}
	pos := p.src.Position() + delta
	p.mu.Unlock()
	p.SeekTo(pos)
}

// SeekTo jumps to an absolute position.
func (p *Player) SeekTo(pos float64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.src == nil {
		return
	}
	if pos < 0 {
		pos = 0
	}
	if p.src.length > 0 && pos > p.src.length-1 {
		pos = p.src.length - 1
	}
	p.src.Seek(pos, p.speed)
	p.done = false
}

// SetSpeed changes playback rate (pitch preserved). Restarts decoding at
// the current position.
func (p *Player) SetSpeed(s float64) float64 {
	if s < 0.5 {
		s = 0.5
	}
	if s > 3 {
		s = 3
	}
	s = math.Round(s*20) / 20
	p.mu.Lock()
	p.speed = s
	if p.src != nil {
		p.src.Seek(p.src.Position(), s)
	}
	p.mu.Unlock()
	p.fire()
	return s
}

// SpeedDelta nudges the speed.
func (p *Player) SpeedDelta(d float64) float64 {
	p.mu.Lock()
	s := p.speed
	p.mu.Unlock()
	return p.SetSpeed(s + d)
}

// VolumeDelta changes the volume.
func (p *Player) VolumeDelta(d float64) {
	p.mu.Lock()
	p.volume = math.Max(0, math.Min(1, p.volume+d))
	p.muted = false
	p.mu.Unlock()
}

// SetVolume sets the volume 0..1.
func (p *Player) SetVolume(v float64) {
	p.mu.Lock()
	p.volume = math.Max(0, math.Min(1, v))
	p.mu.Unlock()
}

// ToggleMute flips mute.
func (p *Player) ToggleMute() {
	p.mu.Lock()
	p.muted = !p.muted
	p.mu.Unlock()
}

// Current returns the track being played (nil if none).
func (p *Player) Current() *Track {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.track
}

// Status returns a snapshot.
func (p *Player) Status() Status {
	p.mu.Lock()
	defer p.mu.Unlock()
	st := Status{Track: p.track, Paused: p.paused, Loading: p.loading, Volume: p.volume, Muted: p.muted, Speed: p.speed, Error: p.lastErr}
	if p.track != nil {
		st.Duration = p.track.Duration
		st.Position = p.track.Start
		st.Stream = isURL(p.track.Source)
	}
	if p.src != nil {
		st.Position = p.src.Position()
		if p.src.length > 0 {
			st.Duration = p.src.length
		}
		st.Playing = !p.paused && !p.done
		st.Buffered = p.src.buffered()
	}
	return st
}

// ErrNoTrack is returned by operations that need a track.
var ErrNoTrack = errors.New("nothing playing")
