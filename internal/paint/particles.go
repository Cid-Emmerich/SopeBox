package paint

import "math/rand"

// Particle is one moving dot.
type Particle struct {
	X, Y, VX, VY float64
	Age, Life    float64
	Size         float64
	C            RGB
	Kind         int
	Spin         float64
}

// System holds particles and a random source.
type System struct {
	P   []Particle
	Rng *rand.Rand
	acc float64
}

// NewSystem creates a particle system with a fixed seed so runs look alike.
func NewSystem(seed int64) *System {
	return &System{Rng: rand.New(rand.NewSource(seed))}
}

// Rand returns a float in [lo,hi).
func (s *System) Rand(lo, hi float64) float64 { return lo + s.Rng.Float64()*(hi-lo) }

// Emit adds a particle.
func (s *System) Emit(p Particle) {
	if p.Life <= 0 {
		p.Life = 1
	}
	s.P = append(s.P, p)
}

// Every calls emit roughly rate times per second, with the time budget
// carried between frames so low frame rates still emit.
func (s *System) Every(dt, rate float64, emit func()) {
	s.acc += dt * rate
	for s.acc >= 1 {
		s.acc--
		emit()
	}
}

// Step advances particles: gravity in units/s², drag as fraction kept per s.
func (s *System) Step(dt, gravity, drag float64) {
	k := 1 - drag*dt
	if k < 0 {
		k = 0
	}
	n := 0
	for i := range s.P {
		p := &s.P[i]
		p.Age += dt
		if p.Age >= p.Life {
			continue
		}
		p.VY += gravity * dt
		p.VX *= k
		p.VY *= k
		p.X += p.VX * dt
		p.Y += p.VY * dt
		s.P[n] = *p
		n++
	}
	s.P = s.P[:n]
}

// T returns the particle's normalised age 0..1.
func (p *Particle) T() float64 { return Clamp01(p.Age / p.Life) }

// Clear removes every particle.
func (s *System) Clear() { s.P = s.P[:0] }
