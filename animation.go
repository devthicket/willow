package willow

import (
	"github.com/phanxgames/willow/internal/types"
	"github.com/tanema/gween"
)

// TweenConfig holds the duration and easing function for a tween.
// A nil Ease defaults to ease.Linear. Duration is in seconds; 0 means instant.
// Aliased from internal/types.TweenConfig.
type TweenConfig = types.TweenConfig

// tweenEase returns cfg.Ease if non-nil, otherwise ease.Linear.
func tweenEase(cfg TweenConfig) func(t, b, c, d float32) float32 {
	return types.TweenEase(cfg)
}

// TweenGroup animates up to 4 float64 fields on a Node simultaneously.
// Create one via the convenience constructors (TweenPosition, TweenScale,
// TweenColor, TweenAlpha, TweenRotation). When the target node belongs to a
// Scene, the tween is automatically managed: Scene.Update() ticks it each
// frame and removes it when done. Manual Update(dt) calls are safe but become
// no-ops for managed tweens to prevent double-ticking.
type TweenGroup struct {
	tweens [4]*gween.Tween
	count  int
	fields [4]*float64
	target *Node
	// Done is true when all tweens in the group have finished, the target
	// node has been disposed, or Cancel was called.
	Done      bool
	managed   bool // true when auto-registered with a Scene
	cancelled bool // set by Cancel(); tween finishes on next tick
}

// tick is the internal method that does the actual tween work.
func (g *TweenGroup) tick(dt float32) {
	if g.Done {
		return
	}

	if g.cancelled {
		g.Done = true
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

// Update advances all tweens by dt seconds, writes values to the target fields,
// and marks the node dirty. For managed tweens (auto-registered with a Scene),
// this is a no-op since Scene.Update() handles ticking.
func (g *TweenGroup) Update(dt float32) {
	if g.managed {
		return
	}
	g.tick(dt)
}

// Cancel marks the tween for removal on the next tick.
func (g *TweenGroup) Cancel() {
	g.cancelled = true
}

// autoRegister registers the tween with the node's scene if available.
func (g *TweenGroup) autoRegister(node *Node) {
	if node.Scene_ != nil {
		g.managed = true
		node.Scene_.(*Scene).registerTween(g)
	}
}

// TweenPosition creates a TweenGroup that animates node.X and node.Y to the
// given target coordinates over the specified duration using the easing function.
func TweenPosition(node *Node, toX, toY float64, cfg TweenConfig) *TweenGroup {
	fn := tweenEase(cfg)
	g := &TweenGroup{count: 2, target: node}
	g.tweens[0] = gween.New(float32(node.X_), float32(toX), cfg.Duration, fn)
	g.tweens[1] = gween.New(float32(node.Y_), float32(toY), cfg.Duration, fn)
	g.fields[0] = &node.X_
	g.fields[1] = &node.Y_
	g.autoRegister(node)
	return g
}

// TweenScale creates a TweenGroup that animates node.ScaleX and node.ScaleY to
// the given target values over the specified duration using the easing function.
func TweenScale(node *Node, toSX, toSY float64, cfg TweenConfig) *TweenGroup {
	fn := tweenEase(cfg)
	g := &TweenGroup{count: 2, target: node}
	g.tweens[0] = gween.New(float32(node.ScaleX_), float32(toSX), cfg.Duration, fn)
	g.tweens[1] = gween.New(float32(node.ScaleY_), float32(toSY), cfg.Duration, fn)
	g.fields[0] = &node.ScaleX_
	g.fields[1] = &node.ScaleY_
	g.autoRegister(node)
	return g
}

// TweenColor creates a TweenGroup that animates all four components of
// node.Color (R, G, B, A) to the target color over the specified duration.
func TweenColor(node *Node, to Color, cfg TweenConfig) *TweenGroup {
	fn := tweenEase(cfg)
	g := &TweenGroup{count: 4, target: node}
	g.tweens[0] = gween.New(float32(node.Color_.R()), float32(to.R()), cfg.Duration, fn)
	g.tweens[1] = gween.New(float32(node.Color_.G()), float32(to.G()), cfg.Duration, fn)
	g.tweens[2] = gween.New(float32(node.Color_.B()), float32(to.B()), cfg.Duration, fn)
	g.tweens[3] = gween.New(float32(node.Color_.A()), float32(to.A()), cfg.Duration, fn)
	rp, gp, bp, ap := types.ColorFieldPtrs(&node.Color_)
	g.fields[0] = rp
	g.fields[1] = gp
	g.fields[2] = bp
	g.fields[3] = ap
	g.autoRegister(node)
	return g
}

// TweenAlpha creates a TweenGroup that animates node.Alpha to the target value
// over the specified duration using the easing function.
func TweenAlpha(node *Node, to float64, cfg TweenConfig) *TweenGroup {
	fn := tweenEase(cfg)
	g := &TweenGroup{count: 1, target: node}
	g.tweens[0] = gween.New(float32(node.Alpha_), float32(to), cfg.Duration, fn)
	g.fields[0] = &node.Alpha_
	g.autoRegister(node)
	return g
}

// TweenRotation creates a TweenGroup that animates node.Rotation to the target
// value over the specified duration using the easing function.
func TweenRotation(node *Node, to float64, cfg TweenConfig) *TweenGroup {
	fn := tweenEase(cfg)
	g := &TweenGroup{count: 1, target: node}
	g.tweens[0] = gween.New(float32(node.Rotation_), float32(to), cfg.Duration, fn)
	g.fields[0] = &node.Rotation_
	g.autoRegister(node)
	return g
}
