package types

// Lerp linearly interpolates between a and b by t.
func Lerp(a, b, t float64) float64 {
	return a + (b-a)*t
}

// Lerp32 linearly interpolates between a and b by t (float32).
func Lerp32(a, b, t float32) float32 {
	return a + (b-a)*t
}
