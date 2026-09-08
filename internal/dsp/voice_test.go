package dsp

import (
	"math"
	"testing"
)

func TestVoicePitch(t *testing.T) {
	a := NewAnalyzer(2048, 44100)
	buf := make([][2]float64, 2048)
	for i := range buf {
		// a 150 Hz sawtooth-ish voice-like tone with harmonics
		x := 0.0
		for h := 1; h <= 6; h++ {
			x += math.Sin(2*math.Pi*150*float64(h)*float64(i)/44100) / float64(h)
		}
		buf[i] = [2]float64{x * 0.3, x * 0.3}
	}
	a.Push(buf)
	f := a.Voice()
	if !f.Voiced {
		t.Fatalf("not voiced: %+v", f)
	}
	if math.Abs(f.F0-150) > 8 {
		t.Errorf("f0 = %v", f.F0)
	}
	if f.Centroid < 150 || f.Centroid > 2000 {
		t.Errorf("centroid = %v", f.Centroid)
	}
	// silence
	a.Clear()
	if s := a.Voice(); s.Voiced || s.Level > 0.001 {
		t.Errorf("silence detected as voice: %+v", s)
	}
}
