// Package voices turns a stream of voice features into "orbs": one per
// distinct speaker, appearing as the conversation introduces them. It is a
// lightweight online speaker clustering: each voiced stretch of audio is
// summarised (timbre + pitch), compared against the known speakers, and
// either assigned to the closest one or, if it is far from all of them for
// long enough, spawns a new orb. When a transcript with speaker labels is
// available the orbs follow it directly, and labels name the orbs.
package voices

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/Cid-Emmerich/SopeBox/internal/dsp"
	"github.com/Cid-Emmerich/SopeBox/internal/store"
	"github.com/Cid-Emmerich/SopeBox/internal/transcript"
)

// Options are the user-tunable isolation settings.
type Options struct {
	Mode        string  // "auto", "transcript", "acoustic"
	Sensitivity float64 // 0..1: how readily a new orb appears
	Smoothing   float64 // 0..0.95: feature smoothing
	Falloff     float64 // per-frame decay of an idle orb
	MaxOrbs     int
}

// Orb is one detected speaker.
type Orb struct {
	ID       int
	Name     string
	Named    bool      // name came from a transcript / user, not "Voice n"
	Level    float64   // smoothed activity 0..1 (drives the visualizer)
	Spectrum []float64 // this speaker's own radial spectrum
	Peaks    []float64
	Centroid []float64 // timbre + pitch embedding
	F0       float64   // running mean pitch
	Talk     float64   // seconds of speech
	Active   bool      // speaking right now
	Born     float64   // time the orb appeared (seconds of session)
	Last     float64   // last time active
	Spawn    float64   // 0..1 spawn animation
	votes    map[string]int
	// physics state owned by the visualizer
	X, Y, VX, VY float64
	Placed       bool
}

// Tracker maintains the orbs.
type Tracker struct {
	Opts Options
	Orbs []*Orb

	nextID  int
	time    float64
	seg     segment
	silence float64
	tr      *transcript.Transcript
	trNames map[string]*Orb
	current *Orb
	persons []string
}

type segment struct {
	n      int
	emb    []float64
	f0log  float64
	frames int
	far    int // consecutive frames far from every orb
	orb    *Orb
	ema    []float64
}

const embDim = dsp.NumBands + 2

// New creates a tracker.
func New(o Options) *Tracker {
	return &Tracker{Opts: o, trNames: map[string]*Orb{}}
}

// Reset forgets every orb (new episode).
func (t *Tracker) Reset() {
	t.Orbs = nil
	t.nextID = 0
	t.seg = segment{}
	t.current = nil
	t.trNames = map[string]*Orb{}
	t.tr = nil
	t.time = 0
}

// SetTranscript attaches speaker labels for the episode.
func (t *Tracker) SetTranscript(tr *transcript.Transcript) { t.tr = tr }

// SetPersons supplies host / guest names from the feed for naming.
func (t *Tracker) SetPersons(names []string) { t.persons = names }

// Persons returns the feed's declared people.
func (t *Tracker) Persons() []string { return t.persons }

// UsingTranscript reports whether orbs currently follow transcript labels.
func (t *Tracker) UsingTranscript() bool {
	if t.tr == nil || !t.tr.HasSpeakers() {
		return false
	}
	return t.Opts.Mode == "transcript" || t.Opts.Mode == "auto"
}

// Current returns the orb speaking now (nil when silent).
func (t *Tracker) Current() *Orb { return t.current }

// Import seeds orbs from stored profiles so known hosts are recognised.
func (t *Tracker) Import(profiles []store.VoiceProfile) {
	for _, p := range profiles {
		if len(p.Centroid) != embDim {
			continue
		}
		o := t.newOrb(p.Name, true)
		o.Centroid = append([]float64(nil), p.Centroid...)
		o.F0 = p.F0
		o.Spawn = 1
		o.Level = 0
		t.trNames[strings.ToLower(p.Name)] = o
	}
}

// Export returns profiles for orbs worth remembering.
func (t *Tracker) Export() []store.VoiceProfile {
	var out []store.VoiceProfile
	for _, o := range t.Orbs {
		if o.Talk < 15 || len(o.Centroid) != embDim {
			continue
		}
		out = append(out, store.VoiceProfile{Name: o.Name, Centroid: append([]float64(nil), o.Centroid...), F0: o.F0, Talk: o.Talk})
	}
	return out
}

func (t *Tracker) newOrb(name string, named bool) *Orb {
	t.nextID++
	if name == "" {
		name = fmt.Sprintf("Voice %d", t.nextID)
	}
	o := &Orb{ID: t.nextID, Name: name, Named: named, Born: t.time, Last: t.time, votes: map[string]int{}}
	t.Orbs = append(t.Orbs, o)
	return o
}

// Rename sets a user-chosen name on an orb.
func (t *Tracker) Rename(o *Orb, name string) {
	if o == nil {
		return
	}
	delete(t.trNames, strings.ToLower(o.Name))
	o.Name = name
	o.Named = true
	t.trNames[strings.ToLower(name)] = o
}

// Merge folds orb b into a (used when the user says two orbs are the same).
func (t *Tracker) Merge(a, b *Orb) {
	if a == nil || b == nil || a == b {
		return
	}
	w := a.Talk / (a.Talk + b.Talk + 1e-9)
	for i := range a.Centroid {
		if i < len(b.Centroid) {
			a.Centroid[i] = a.Centroid[i]*w + b.Centroid[i]*(1-w)
		}
	}
	a.Talk += b.Talk
	t.remove(b)
}

func (t *Tracker) remove(b *Orb) {
	for i, o := range t.Orbs {
		if o == b {
			t.Orbs = append(t.Orbs[:i], t.Orbs[i+1:]...)
			break
		}
	}
	for k, o := range t.trNames {
		if o == b {
			delete(t.trNames, k)
		}
	}
	if t.current == b {
		t.current = nil
	}
}

// ---------------------------------------------------------------------------
// Per-frame update

// Update advances the tracker by one frame. f is the current voice
// feature, spectrum the radial spectrum bands to hand to the active orb,
// pos the playback position (for transcripts) and dt the frame time.
func (t *Tracker) Update(f dsp.Features, spectrum []float64, pos, dt float64, playing bool) {
	t.time += dt
	for _, o := range t.Orbs {
		o.Active = false
		if o.Spawn < 1 {
			o.Spawn = math.Min(1, o.Spawn+dt*2.2)
		}
	}
	speaking := playing && f.Level > 0.008
	if t.UsingTranscript() {
		t.updateTranscript(f, spectrum, pos, dt, speaking)
	} else {
		t.updateAcoustic(f, spectrum, pos, dt, speaking)
	}
	// decay idle orbs
	for _, o := range t.Orbs {
		if !o.Active {
			o.Level = math.Max(0, o.Level-t.Opts.Falloff)
			for i := range o.Spectrum {
				o.Spectrum[i] = math.Max(0, o.Spectrum[i]-t.Opts.Falloff*1.5)
			}
		}
		for i := range o.Peaks {
			if i < len(o.Spectrum) && o.Spectrum[i] >= o.Peaks[i] {
				o.Peaks[i] = o.Spectrum[i]
			} else {
				o.Peaks[i] = math.Max(0, o.Peaks[i]-t.Opts.Falloff*0.5)
			}
		}
	}
}

func (t *Tracker) activate(o *Orb, f dsp.Features, spectrum []float64, dt float64) {
	o.Active = true
	o.Last = t.time
	o.Talk += dt
	lv := math.Min(1, f.Level*6)
	s := t.Opts.Smoothing
	if lv > o.Level {
		o.Level = o.Level*s + lv*(1-s)
	} else {
		o.Level = math.Max(lv, o.Level-t.Opts.Falloff)
	}
	if len(o.Spectrum) != len(spectrum) {
		o.Spectrum = make([]float64, len(spectrum))
		o.Peaks = make([]float64, len(spectrum))
	}
	for i, v := range spectrum {
		if v > o.Spectrum[i] {
			o.Spectrum[i] = o.Spectrum[i]*s*0.6 + v*(1-s*0.6)
		} else {
			o.Spectrum[i] = math.Max(v, o.Spectrum[i]-t.Opts.Falloff*1.5)
		}
	}
	t.current = o
}

// --- transcript driven ------------------------------------------------------

func (t *Tracker) updateTranscript(f dsp.Features, spectrum []float64, pos, dt float64, speaking bool) {
	name := t.tr.SpeakerAt(pos)
	if name == "" || !speaking {
		t.current = nil
		return
	}
	o := t.trNames[strings.ToLower(name)]
	if o == nil {
		o = t.newOrb(name, true)
		t.trNames[strings.ToLower(name)] = o
	}
	// keep learning the acoustic centroid so the profile can be saved
	if f.Voiced {
		emb := embed(f)
		if o.Centroid == nil {
			o.Centroid = emb
		} else {
			for i := range o.Centroid {
				o.Centroid[i] = o.Centroid[i]*0.98 + emb[i]*0.02
			}
		}
		if o.F0 == 0 {
			o.F0 = f.F0
		} else {
			o.F0 = o.F0*0.98 + f.F0*0.02
		}
	}
	t.activate(o, f, spectrum, dt)
}

// --- acoustic clustering ----------------------------------------------------

func embed(f dsp.Features) []float64 {
	e := make([]float64, embDim)
	copy(e, f.Bands[:])
	e[dsp.NumBands] = math.Log2(math.Max(f.F0, 50) / 100) // octaves above 100 Hz
	e[dsp.NumBands+1] = math.Log2(math.Max(f.Centroid, 200) / 1000)
	return e
}

// distance between two embeddings, roughly 0..1.
func distance(a, b []float64) float64 {
	if len(a) != embDim || len(b) != embDim {
		return 1
	}
	// timbre: normalised euclidean over the band shape
	d := 0.0
	for i := 0; i < dsp.NumBands; i++ {
		x := a[i] - b[i]
		d += x * x
	}
	timbre := math.Sqrt(d/dsp.NumBands) / 0.55               // ~0.55 log10 units is a big difference
	pitch := math.Abs(a[dsp.NumBands]-b[dsp.NumBands]) / 0.5 // half an octave
	cent := math.Abs(a[dsp.NumBands+1]-b[dsp.NumBands+1]) / 0.8
	return math.Min(1.5, 0.5*timbre+0.4*pitch+0.1*cent)
}

func (t *Tracker) threshold() float64 {
	// sensitivity 0 → only very different voices split; 1 → eager
	return 0.95 - 0.6*t.Opts.Sensitivity
}

func (t *Tracker) spawnFrames() int {
	// how long a voice must stay "far" before it earns its own orb
	return int(50 - 36*t.Opts.Sensitivity)
}

func (t *Tracker) updateAcoustic(f dsp.Features, spectrum []float64, pos, dt float64, speaking bool) {
	if !speaking || !f.Voiced {
		t.silence += dt
		if t.silence > 0.35 {
			t.endSegment()
		}
		if t.current != nil && speaking && t.silence < 0.35 {
			// unvoiced consonants inside a phrase still belong to the speaker
			t.activate(t.current, f, spectrum, dt)
		} else {
			t.current = nil
		}
		return
	}
	t.silence = 0
	emb := embed(f)
	s := &t.seg
	if s.ema == nil {
		s.ema = append([]float64(nil), emb...)
	} else {
		a := 1 - t.Opts.Smoothing
		for i := range s.ema {
			s.ema[i] = s.ema[i]*(1-a) + emb[i]*a
		}
	}
	if s.emb == nil {
		s.emb = make([]float64, embDim)
	}
	for i := range s.emb {
		s.emb[i] += emb[i]
	}
	s.f0log += math.Log2(math.Max(f.F0, 50))
	s.n++
	s.frames++

	// running decision once we have a little evidence
	if s.n >= 6 {
		mean := make([]float64, embDim)
		for i := range mean {
			mean[i] = 0.5*s.emb[i]/float64(s.n) + 0.5*s.ema[i]
		}
		best, bd := t.nearest(mean)
		th := t.threshold()
		switch {
		case best != nil && bd < th:
			s.far = 0
			if s.orb == nil || s.orb != best {
				// switch mid-segment only if clearly closer
				if s.orb == nil || bd < distance(mean, s.orb.Centroid)-0.08 {
					s.orb = best
				}
			}
		case len(t.Orbs) == 0:
			s.orb = t.newOrb("", false)
			s.orb.Centroid = mean
			s.orb.F0 = math.Pow(2, s.f0log/float64(s.n))
			s.far = 0
		default:
			s.far++
			if s.far >= t.spawnFrames() {
				if len(t.Orbs) < t.Opts.MaxOrbs {
					s.orb = t.newOrb("", false)
					s.orb.Centroid = mean
					s.orb.F0 = math.Pow(2, s.f0log/float64(s.n))
				} else {
					s.orb = best
				}
				s.far = 0
			} else if s.orb == nil {
				s.orb = best
			}
		}
	}
	if s.orb != nil {
		t.activate(s.orb, f, spectrum, dt)
		t.vote(s.orb, pos)
	}
}

func (t *Tracker) nearest(emb []float64) (*Orb, float64) {
	var best *Orb
	bd := math.Inf(1)
	for _, o := range t.Orbs {
		if o.Centroid == nil {
			continue
		}
		d := distance(emb, o.Centroid)
		if d < bd {
			bd, best = d, o
		}
	}
	return best, bd
}

// endSegment folds the finished segment into its orb's centroid.
func (t *Tracker) endSegment() {
	s := &t.seg
	if s.orb != nil && s.n >= 6 {
		lr := 0.15
		if s.orb.Talk > 120 {
			lr = 0.05 // long-known voices move slowly
		}
		for i := range s.orb.Centroid {
			s.orb.Centroid[i] = s.orb.Centroid[i]*(1-lr) + (s.emb[i]/float64(s.n))*lr
		}
		f0 := math.Pow(2, s.f0log/float64(s.n))
		s.orb.F0 = s.orb.F0*(1-lr) + f0*lr
	}
	t.seg = segment{}
	t.current = nil
}

// vote labels an acoustic orb with the transcript speaker heard at pos.
func (t *Tracker) vote(o *Orb, pos float64) {
	if t.tr == nil || !t.tr.HasSpeakers() || o.Named {
		return
	}
	name := t.tr.SpeakerAt(pos)
	if name == "" {
		return
	}
	o.votes[name]++
	if o.votes[name] > 90 { // ~3 seconds of agreement
		total := 0
		for _, v := range o.votes {
			total += v
		}
		if float64(o.votes[name])/float64(total) > 0.6 {
			o.Name = name
			o.Named = true
		}
	}
}

// Sorted returns orbs oldest first (stable display order).
func (t *Tracker) Sorted() []*Orb {
	out := append([]*Orb(nil), t.Orbs...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// Visible returns orbs that have spoken in this episode or are named.
func (t *Tracker) Visible() []*Orb {
	var out []*Orb
	for _, o := range t.Sorted() {
		if o.Talk > 0.5 || o.Level > 0 {
			out = append(out, o)
		}
	}
	return out
}

// Prune drops orbs that spoke for under two seconds and have been quiet
// for a while (misfires from a cough or a jingle).
func (t *Tracker) Prune() {
	keep := t.Orbs[:0]
	for _, o := range t.Orbs {
		if o.Named || o.Talk >= 2 || t.time-o.Last < 20 || o.Active {
			keep = append(keep, o)
		}
	}
	t.Orbs = keep
}
