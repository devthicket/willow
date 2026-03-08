package types

import "math/rand/v2"

// Range is a general-purpose min/max range.
// Used by the particle system (EmitterConfig) and potentially other systems.
type Range struct {
	Min, Max float64
}

// Random returns a random float64 in [Min, Max].
func (r Range) Random() float64 {
	if r.Min == r.Max {
		return r.Min
	}
	return r.Min + rand.Float64()*(r.Max-r.Min)
}
