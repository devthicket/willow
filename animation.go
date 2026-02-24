package willow

import (
	"github.com/tanema/gween"
	"github.com/tanema/gween/ease"
)

// TweenGroup animates up to 4 float64 fields on a Node simultaneously.
// Create one via the convenience constructors (TweenPosition, TweenScale,
// TweenColor) and call Update(dt) each frame. The group auto-applies values
// and marks the node dirty. If the target node is disposed, the group stops
// immediately.
//
// There is no global animation manager  -  users call Update themselves.
type TweenGroup struct {
	tweens [4]*gween.Tween
	count  int
	fields [4]*float64
	target *Node
	// Done is true when all tweens in the group have finished or the target
	// node has been disposed.
	Done bool
}

// Update advances all tweens by dt seconds, writes values to the target fields,
// and marks the node dirty. If the target node has been disposed, Done is set
// to true and no writes occur.
func (g *TweenGroup) Update(dt float32) {
	if g.Done {
		return
	}

	if g.target != nil && g.target.IsDisposed() {
		g.Done = true
		return
	}

	allDone := true
	for i := 0; i < g.count; i++ {
		val, finished := g.tweens[i].Update(dt)
		*g.fields[i] = float64(val)
		if !finished {
			allDone = false
		}
	}
	g.Done = allDone

	if g.target != nil {
		g.target.Invalidate()
	}
}

// TweenPosition creates a TweenGroup that animates node.X and node.Y to the
// given target coordinates over the specified duration using the easing function.
func TweenPosition(node *Node, toX, toY float64, duration float32, fn ease.TweenFunc) *TweenGroup {
	g := &TweenGroup{count: 2, target: node}
	g.tweens[0] = gween.New(float32(node.X), float32(toX), duration, fn)
	g.tweens[1] = gween.New(float32(node.Y), float32(toY), duration, fn)
	g.fields[0] = &node.X
	g.fields[1] = &node.Y
	return g
}

// TweenScale creates a TweenGroup that animates node.ScaleX and node.ScaleY to
// the given target values over the specified duration using the easing function.
func TweenScale(node *Node, toSX, toSY float64, duration float32, fn ease.TweenFunc) *TweenGroup {
	g := &TweenGroup{count: 2, target: node}
	g.tweens[0] = gween.New(float32(node.ScaleX), float32(toSX), duration, fn)
	g.tweens[1] = gween.New(float32(node.ScaleY), float32(toSY), duration, fn)
	g.fields[0] = &node.ScaleX
	g.fields[1] = &node.ScaleY
	return g
}

// TweenColor creates a TweenGroup that animates all four components of
// node.Color (R, G, B, A) to the target color over the specified duration.
func TweenColor(node *Node, to Color, duration float32, fn ease.TweenFunc) *TweenGroup {
	g := &TweenGroup{count: 4, target: node}
	g.tweens[0] = gween.New(float32(node.Color.R), float32(to.R), duration, fn)
	g.tweens[1] = gween.New(float32(node.Color.G), float32(to.G), duration, fn)
	g.tweens[2] = gween.New(float32(node.Color.B), float32(to.B), duration, fn)
	g.tweens[3] = gween.New(float32(node.Color.A), float32(to.A), duration, fn)
	g.fields[0] = &node.Color.R
	g.fields[1] = &node.Color.G
	g.fields[2] = &node.Color.B
	g.fields[3] = &node.Color.A
	return g
}

// TweenAlpha creates a TweenGroup that animates node.Alpha to the target value
// over the specified duration using the easing function.
func TweenAlpha(node *Node, to float64, duration float32, fn ease.TweenFunc) *TweenGroup {
	g := &TweenGroup{count: 1, target: node}
	g.tweens[0] = gween.New(float32(node.Alpha), float32(to), duration, fn)
	g.fields[0] = &node.Alpha
	return g
}

// TweenRotation creates a TweenGroup that animates node.Rotation to the target
// value over the specified duration using the easing function.
func TweenRotation(node *Node, to float64, duration float32, fn ease.TweenFunc) *TweenGroup {
	g := &TweenGroup{count: 1, target: node}
	g.tweens[0] = gween.New(float32(node.Rotation), float32(to), duration, fn)
	g.fields[0] = &node.Rotation
	return g
}
