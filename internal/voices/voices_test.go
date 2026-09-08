package voices

import (
	"math"
	"testing"

	"github.com/Cid-Emmerich/SopeBox/internal/dsp"
	"github.com/Cid-Emmerich/SopeBox/internal/transcript"
)

func feat(f0 float64, tilt float64) dsp.Features {
	f := dsp.Features{Level: 0.1, Voiced: true, Voicing: 0.8, F0: f0, Centroid: 800 + tilt*400}
	for i := range f.Bands {
		f.Bands[i] = tilt * (float64(i)/dsp.NumBands - 0.5)
	}
	return f
}

func TestAcousticSplitsTwoVoices(t *testing.T) {
	tr := New(Options{Mode: "acoustic", Sensitivity: 0.5, Smoothing: 0.5, Falloff: 0.05, MaxOrbs: 6})
	spec := make([]float64, 32)
	dt := 1.0 / 30
	// voice A talks for 4 seconds
	for i := 0; i < 120; i++ {
		tr.Update(feat(110, 1.0), spec, float64(i)*dt, dt, true)
	}
	if len(tr.Orbs) != 1 {
		t.Fatalf("after A: orbs = %d", len(tr.Orbs))
	}
	// silence
	for i := 0; i < 20; i++ {
		tr.Update(dsp.Features{}, spec, 4, dt, true)
	}
	// voice B, very different pitch and timbre
	for i := 0; i < 120; i++ {
		tr.Update(feat(220, -1.0), spec, 5+float64(i)*dt, dt, true)
	}
	if len(tr.Orbs) != 2 {
		t.Fatalf("after B: orbs = %d", len(tr.Orbs))
	}
	if tr.Current() == nil || tr.Current().ID != 2 {
		t.Errorf("current = %+v", tr.Current())
	}
	// A returns and should be recognised, not spawn a third orb
	for i := 0; i < 20; i++ {
		tr.Update(dsp.Features{}, spec, 9, dt, true)
	}
	for i := 0; i < 120; i++ {
		tr.Update(feat(112, 0.95), spec, 10+float64(i)*dt, dt, true)
	}
	if len(tr.Orbs) != 2 || tr.Current().ID != 1 {
		t.Errorf("A not recognised: orbs=%d current=%v", len(tr.Orbs), tr.Current())
	}
	// profiles are only exported once a voice has talked for a while
	for i := 0; i < 600; i++ {
		tr.Update(feat(112, 0.95), spec, 14+float64(i)*dt, dt, true)
	}
	if p := tr.Export(); len(p) != 1 || p[0].Talk < 15 {
		t.Errorf("export = %+v", p)
	}
}

func TestLowSensitivityMerges(t *testing.T) {
	tr := New(Options{Mode: "acoustic", Sensitivity: 0.0, Smoothing: 0.5, Falloff: 0.05, MaxOrbs: 6})
	spec := make([]float64, 32)
	dt := 1.0 / 30
	for i := 0; i < 90; i++ {
		tr.Update(feat(120, 0.5), spec, float64(i)*dt, dt, true)
	}
	for i := 0; i < 20; i++ {
		tr.Update(dsp.Features{}, spec, 4, dt, true)
	}
	for i := 0; i < 90; i++ {
		tr.Update(feat(135, 0.4), spec, 5+float64(i)*dt, dt, true)
	}
	if len(tr.Orbs) != 1 {
		t.Errorf("similar voices split at sensitivity 0: orbs = %d", len(tr.Orbs))
	}
}

func TestTranscriptMode(t *testing.T) {
	tr := New(Options{Mode: "auto", Sensitivity: 0.5, Smoothing: 0.5, Falloff: 0.05, MaxOrbs: 6})
	tr.SetTranscript(&transcript.Transcript{Segments: []transcript.Segment{
		{Start: 0, End: 5, Speaker: "Adam", Text: "hi"},
		{Start: 5, End: 10, Speaker: "John", Text: "hello"},
	}, Speakers: []string{"Adam", "John"}})
	if !tr.UsingTranscript() {
		t.Fatal("not using transcript")
	}
	spec := make([]float64, 32)
	tr.Update(feat(120, 0), spec, 1, 0.03, true)
	tr.Update(feat(120, 0), spec, 6, 0.03, true)
	if len(tr.Orbs) != 2 || tr.Orbs[0].Name != "Adam" || tr.Orbs[1].Name != "John" {
		t.Errorf("orbs = %+v", tr.Orbs)
	}
	if !tr.Orbs[1].Active || tr.Orbs[0].Active {
		t.Error("wrong active orb")
	}
	// decay
	for i := 0; i < 40; i++ {
		tr.Update(dsp.Features{}, spec, 12, 0.03, true)
	}
	if tr.Orbs[1].Level > 0.01 {
		t.Errorf("level did not decay: %v", tr.Orbs[1].Level)
	}
}

func TestDistance(t *testing.T) {
	a := embed(feat(110, 1))
	if d := distance(a, a); d != 0 {
		t.Errorf("self distance %v", d)
	}
	if d := distance(a, embed(feat(220, -1))); d < 0.8 {
		t.Errorf("different voices too close: %v", d)
	}
	if d := distance(a, embed(feat(115, 0.9))); d > 0.35 {
		t.Errorf("same voice too far: %v", d)
	}
	_ = math.Pi
}
