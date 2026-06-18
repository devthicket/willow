//go:build physics

package physics

import "github.com/jakecoffman/cp/v2"

// Config carries the tunables a physics parent needs at construction time.
//
// Iterations of 0 leaves cp's default (10) in place. SleepTime of 0 disables
// idle-body sleeping entirely (cp's default is to never sleep).
type Config struct {
	Gravity    cp.Vector
	Iterations int
	SleepTime  float64
}
