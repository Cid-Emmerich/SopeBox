package dsp

import (
	"math"
	"math/cmplx"
)

// NumBands is the size of the spectral-shape part of a voice feature.
const NumBands = 20

// Features describes the sound in the most recent analysis window in the
// terms a speaker tracker cares about.
type Features struct {
	Level    float64           // RMS 0..1
	Voiced   bool              // periodic (a voice, not silence/noise/music)
	Voicing  float64           // 0..1 autocorrelation strength
	F0       float64           // fundamental frequency in Hz (0 when unvoiced)
	Bands    [NumBands]float64 // mean-normalised log band energies (timbre)
	Centroid float64           // spectral centroid in Hz
}

// Voice computes speaker features from the newest samples.
func (a *Analyzer) Voice() Features {
	var f Features
	a.mu.Lock()
	rate := a.rate
	n := a.size
	start := (a.pos - n + n) % n
	mono := make([]float64, n)
	sum := 0.0
	for i := 0; i < n; i++ {
		s := a.ring[(start+i)%n]
		v := (s[0] + s[1]) / 2
		mono[i] = v
		sum += v * v
	}
	a.mu.Unlock()
	f.Level = math.Sqrt(sum / float64(n))
	if f.Level < 0.004 {
		return f
	}

	// --- pitch by normalised autocorrelation ------------------------------
	minLag := rate / 400
	maxLag := rate / 70
	if maxLag > n/2 {
		maxLag = n / 2
	}
	// remove DC
	mean := 0.0
	for _, v := range mono {
		mean += v
	}
	mean /= float64(n)
	e0 := 0.0
	for i := range mono {
		mono[i] -= mean
		e0 += mono[i] * mono[i]
	}
	bestLag, best := 0, 0.0
	// coarse search with step 2 then refine
	for lag := minLag; lag <= maxLag; lag += 2 {
		c, e1 := 0.0, 0.0
		for i := 0; i+lag < n; i++ {
			c += mono[i] * mono[i+lag]
			e1 += mono[i+lag] * mono[i+lag]
		}
		if e1 > 0 {
			c /= math.Sqrt(e0 * e1)
		}
		if c > best {
			best, bestLag = c, lag
		}
	}
	if bestLag > 0 {
		for lag := bestLag - 1; lag <= bestLag+1; lag++ {
			if lag < minLag || lag > maxLag {
				continue
			}
			c, e1 := 0.0, 0.0
			for i := 0; i+lag < n; i++ {
				c += mono[i] * mono[i+lag]
				e1 += mono[i+lag] * mono[i+lag]
			}
			if e1 > 0 {
				c /= math.Sqrt(e0 * e1)
			}
			if c > best {
				best, bestLag = c, lag
			}
		}
	}
	f.Voicing = best
	if best > 0.45 && bestLag > 0 {
		f.Voiced = true
		f.F0 = float64(rate) / float64(bestLag)
	}

	// --- spectral shape ---------------------------------------------------
	for i := 0; i < n; i++ {
		a.buf[i] = complex(mono[i]*a.window[i], 0)
	}
	fft(a.buf)
	half := n / 2
	binHz := float64(rate) / float64(n)
	lo, hi := 100.0, 5000.0
	totalE, weighted := 0.0, 0.0
	for i := 1; i < half; i++ {
		m := cmplx.Abs(a.buf[i])
		e := m * m
		totalE += e
		weighted += e * float64(i) * binHz
	}
	if totalE > 0 {
		f.Centroid = weighted / totalE
	}
	meanLog := 0.0
	for b := 0; b < NumBands; b++ {
		f0 := lo * math.Pow(hi/lo, float64(b)/NumBands)
		f1 := lo * math.Pow(hi/lo, float64(b+1)/NumBands)
		i0, i1 := int(f0/binHz), int(f1/binHz)
		if i1 <= i0 {
			i1 = i0 + 1
		}
		if i1 > half {
			i1 = half
		}
		e := 0.0
		for i := i0; i < i1; i++ {
			m := cmplx.Abs(a.buf[i])
			e += m * m
		}
		e /= float64(i1 - i0)
		f.Bands[b] = math.Log10(e + 1e-9)
		meanLog += f.Bands[b]
	}
	meanLog /= NumBands
	for b := range f.Bands {
		f.Bands[b] -= meanLog
	}
	return f
}
