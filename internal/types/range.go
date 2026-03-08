package types

// Range is a general-purpose min/max range.
// Used by the particle system (EmitterConfig) and potentially other systems.
type Range struct {
	Min, Max float64
}
