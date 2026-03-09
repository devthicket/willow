package willow

import (
	"github.com/phanxgames/willow/internal/core"
	"github.com/phanxgames/willow/internal/types"
	"github.com/tanema/gween"
)

// TweenConfig holds the duration and easing function for a tween.
// A nil Ease defaults to ease.Linear. Duration is in seconds; 0 means instant.
// Aliased from internal/types.TweenConfig.
type TweenConfig = types.TweenConfig

// TweenGroup animates up to 4 float64 fields on a Node simultaneously.
// Create one via the convenience constructors (TweenPosition, TweenScale,
// TweenColor, TweenAlpha, TweenRotation). When the target node belongs to a
// Scene, the tween is automatically managed: Scene.Update() ticks it each
// frame and removes it when done. Manual Update(dt) calls are safe but become
// no-ops for managed tweens to prevent double-ticking.
// Aliased from internal/core.TweenGroup.
type TweenGroup = core.TweenGroup

// tweenEase returns cfg.Ease if non-nil, otherwise ease.Linear.
func tweenEase(cfg TweenConfig) func(t, b, c, d float32) float32 {
	return types.TweenEase(cfg)
}

// autoRegister registers the tween with the node's scene if available.
func autoRegister(g *TweenGroup, n *Node) {
	if n.Scene_ != nil {
		g.Managed = true
		n.Scene_.(*Scene).registerTween(g)
	}
}

// TweenPosition creates a TweenGroup that animates node.X and node.Y to the
// given target coordinates over the specified duration using the easing function.
func TweenPosition(node *Node, toX, toY float64, cfg TweenConfig) *TweenGroup {
	fn := tweenEase(cfg)
	g := &TweenGroup{Count: 2, Target: node}
	g.Tweens[0] = gween.New(float32(node.X_), float32(toX), cfg.Duration, fn)
	g.Tweens[1] = gween.New(float32(node.Y_), float32(toY), cfg.Duration, fn)
	g.Fields[0] = &node.X_
	g.Fields[1] = &node.Y_
	autoRegister(g, node)
	return g
}

// TweenScale creates a TweenGroup that animates node.ScaleX and node.ScaleY to
// the given target values over the specified duration using the easing function.
func TweenScale(node *Node, toSX, toSY float64, cfg TweenConfig) *TweenGroup {
	fn := tweenEase(cfg)
	g := &TweenGroup{Count: 2, Target: node}
	g.Tweens[0] = gween.New(float32(node.ScaleX_), float32(toSX), cfg.Duration, fn)
	g.Tweens[1] = gween.New(float32(node.ScaleY_), float32(toSY), cfg.Duration, fn)
	g.Fields[0] = &node.ScaleX_
	g.Fields[1] = &node.ScaleY_
	autoRegister(g, node)
	return g
}

// TweenColor creates a TweenGroup that animates all four components of
// node.Color (R, G, B, A) to the target color over the specified duration.
func TweenColor(node *Node, to Color, cfg TweenConfig) *TweenGroup {
	fn := tweenEase(cfg)
	g := &TweenGroup{Count: 4, Target: node}
	g.Tweens[0] = gween.New(float32(node.Color_.R()), float32(to.R()), cfg.Duration, fn)
	g.Tweens[1] = gween.New(float32(node.Color_.G()), float32(to.G()), cfg.Duration, fn)
	g.Tweens[2] = gween.New(float32(node.Color_.B()), float32(to.B()), cfg.Duration, fn)
	g.Tweens[3] = gween.New(float32(node.Color_.A()), float32(to.A()), cfg.Duration, fn)
	rp, gp, bp, ap := types.ColorFieldPtrs(&node.Color_)
	g.Fields[0] = rp
	g.Fields[1] = gp
	g.Fields[2] = bp
	g.Fields[3] = ap
	autoRegister(g, node)
	return g
}

// TweenAlpha creates a TweenGroup that animates node.Alpha to the target value
// over the specified duration using the easing function.
func TweenAlpha(node *Node, to float64, cfg TweenConfig) *TweenGroup {
	fn := tweenEase(cfg)
	g := &TweenGroup{Count: 1, Target: node}
	g.Tweens[0] = gween.New(float32(node.Alpha_), float32(to), cfg.Duration, fn)
	g.Fields[0] = &node.Alpha_
	autoRegister(g, node)
	return g
}

// TweenRotation creates a TweenGroup that animates node.Rotation to the target
// value over the specified duration using the easing function.
func TweenRotation(node *Node, to float64, cfg TweenConfig) *TweenGroup {
	fn := tweenEase(cfg)
	g := &TweenGroup{Count: 1, Target: node}
	g.Tweens[0] = gween.New(float32(node.Rotation_), float32(to), cfg.Duration, fn)
	g.Fields[0] = &node.Rotation_
	autoRegister(g, node)
	return g
}
