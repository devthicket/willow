package core

import (
	"github.com/devthicket/willow/internal/node"
	"github.com/tanema/gween"
)

// TweenGroup animates up to 4 float64 fields on a Node simultaneously.
// Create via root convenience constructors (TweenPosition, TweenScale, etc.).
type TweenGroup struct {
	Tweens [4]*gween.Tween
	Count  int
	Fields [4]*float64
	Target *node.Node
	Done   bool
	// Managed is true when auto-registered with a Scene.
	Managed   bool
	Cancelled bool
}

// Tick is the internal method that does the actual tween work.
func (g *TweenGroup) Tick(dt float32) {
	if g.Done {
		return
	}

	if g.Cancelled {
		g.Done = true
		return
	}

	if g.Target != nil && g.Target.IsDisposed() {
		g.Done = true
		return
	}

	allDone := true
	for i := 0; i < g.Count; i++ {
		val, finished := g.Tweens[i].Update(dt)
		*g.Fields[i] = float64(val)
		if !finished {
			allDone = false
		}
	}
	g.Done = allDone

	if g.Target != nil {
		g.Target.Invalidate()
	}
}

// Update advances all tweens by dt seconds. For managed tweens (auto-registered
// with a Scene), this is a no-op since Scene.Update() handles ticking.
func (g *TweenGroup) Update(dt float32) {
	if g.Managed {
		return
	}
	g.Tick(dt)
}

// Cancel marks the tween for removal on the next tick.
func (g *TweenGroup) Cancel() {
	g.Cancelled = true
}
