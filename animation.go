package willow

import (
	"github.com/phanxgames/willow/internal/types"
	"github.com/tanema/gween"
	"github.com/tanema/gween/ease"
)

// TweenConfig holds the duration and easing function for a tween.
// A nil Ease defaults to ease.Linear. Duration is in seconds; 0 means instant.
type TweenConfig struct {
	Duration float32        // seconds; 0 = instant
	Ease     ease.TweenFunc // nil defaults to ease.Linear
}

// tweenEase returns cfg.Ease if non-nil, otherwise ease.Linear.
func tweenEase(cfg TweenConfig) ease.TweenFunc {
	if cfg.Ease != nil {
		return cfg.Ease
	}
	return ease.Linear
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
	if node.scene != nil {
		g.managed = true
		node.scene.registerTween(g)
	}
}

// TweenPosition creates a TweenGroup that animates node.X and node.Y to the
// given target coordinates over the specified duration using the easing function.
func TweenPosition(node *Node, toX, toY float64, cfg TweenConfig) *TweenGroup {
	fn := tweenEase(cfg)
	g := &TweenGroup{count: 2, target: node}
	g.tweens[0] = gween.New(float32(node.x), float32(toX), cfg.Duration, fn)
	g.tweens[1] = gween.New(float32(node.y), float32(toY), cfg.Duration, fn)
	g.fields[0] = &node.x
	g.fields[1] = &node.y
	g.autoRegister(node)
	return g
}

// TweenScale creates a TweenGroup that animates node.ScaleX and node.ScaleY to
// the given target values over the specified duration using the easing function.
func TweenScale(node *Node, toSX, toSY float64, cfg TweenConfig) *TweenGroup {
	fn := tweenEase(cfg)
	g := &TweenGroup{count: 2, target: node}
	g.tweens[0] = gween.New(float32(node.scaleX), float32(toSX), cfg.Duration, fn)
	g.tweens[1] = gween.New(float32(node.scaleY), float32(toSY), cfg.Duration, fn)
	g.fields[0] = &node.scaleX
	g.fields[1] = &node.scaleY
	g.autoRegister(node)
	return g
}

// TweenColor creates a TweenGroup that animates all four components of
// node.Color (R, G, B, A) to the target color over the specified duration.
func TweenColor(node *Node, to Color, cfg TweenConfig) *TweenGroup {
	fn := tweenEase(cfg)
	g := &TweenGroup{count: 4, target: node}
	g.tweens[0] = gween.New(float32(node.color.R()), float32(to.R()), cfg.Duration, fn)
	g.tweens[1] = gween.New(float32(node.color.G()), float32(to.G()), cfg.Duration, fn)
	g.tweens[2] = gween.New(float32(node.color.B()), float32(to.B()), cfg.Duration, fn)
	g.tweens[3] = gween.New(float32(node.color.A()), float32(to.A()), cfg.Duration, fn)
	rp, gp, bp, ap := types.ColorFieldPtrs(&node.color)
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
	g.tweens[0] = gween.New(float32(node.alpha), float32(to), cfg.Duration, fn)
	g.fields[0] = &node.alpha
	g.autoRegister(node)
	return g
}

// TweenRotation creates a TweenGroup that animates node.Rotation to the target
// value over the specified duration using the easing function.
func TweenRotation(node *Node, to float64, cfg TweenConfig) *TweenGroup {
	fn := tweenEase(cfg)
	g := &TweenGroup{count: 1, target: node}
	g.tweens[0] = gween.New(float32(node.rotation), float32(to), cfg.Duration, fn)
	g.fields[0] = &node.rotation
	g.autoRegister(node)
	return g
}
