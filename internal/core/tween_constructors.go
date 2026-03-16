package core

import (
	"github.com/devthicket/willow/internal/node"
	"github.com/devthicket/willow/internal/types"
	"github.com/tanema/gween"
)

// autoRegister registers the tween with the node's scene if available.
func autoRegister(g *TweenGroup, n *node.Node) {
	if n.Scene_ != nil {
		g.Managed = true
		n.Scene_.(*Scene).RegisterTween(g)
	}
}

// TweenPosition animates node.X_ and node.Y_ to the given target.
func TweenPosition(n *node.Node, toX, toY float64, cfg types.TweenConfig) *TweenGroup {
	fn := types.TweenEase(cfg)
	g := &TweenGroup{Count: 2, Target: n}
	g.Tweens[0] = gween.New(float32(n.X_), float32(toX), cfg.Duration, fn)
	g.Tweens[1] = gween.New(float32(n.Y_), float32(toY), cfg.Duration, fn)
	g.Fields[0] = &n.X_
	g.Fields[1] = &n.Y_
	autoRegister(g, n)
	return g
}

// TweenScale animates node.ScaleX_ and node.ScaleY_ to the given target.
func TweenScale(n *node.Node, toSX, toSY float64, cfg types.TweenConfig) *TweenGroup {
	fn := types.TweenEase(cfg)
	g := &TweenGroup{Count: 2, Target: n}
	g.Tweens[0] = gween.New(float32(n.ScaleX_), float32(toSX), cfg.Duration, fn)
	g.Tweens[1] = gween.New(float32(n.ScaleY_), float32(toSY), cfg.Duration, fn)
	g.Fields[0] = &n.ScaleX_
	g.Fields[1] = &n.ScaleY_
	autoRegister(g, n)
	return g
}

// TweenColor animates all four components of node.Color_ to the target color.
func TweenColor(n *node.Node, to types.Color, cfg types.TweenConfig) *TweenGroup {
	fn := types.TweenEase(cfg)
	g := &TweenGroup{Count: 4, Target: n}
	g.Tweens[0] = gween.New(float32(n.Color_.R()), float32(to.R()), cfg.Duration, fn)
	g.Tweens[1] = gween.New(float32(n.Color_.G()), float32(to.G()), cfg.Duration, fn)
	g.Tweens[2] = gween.New(float32(n.Color_.B()), float32(to.B()), cfg.Duration, fn)
	g.Tweens[3] = gween.New(float32(n.Color_.A()), float32(to.A()), cfg.Duration, fn)
	rp, gp, bp, ap := types.ColorFieldPtrs(&n.Color_)
	g.Fields[0] = rp
	g.Fields[1] = gp
	g.Fields[2] = bp
	g.Fields[3] = ap
	autoRegister(g, n)
	return g
}

// TweenAlpha animates node.Alpha_ to the target value.
func TweenAlpha(n *node.Node, to float64, cfg types.TweenConfig) *TweenGroup {
	fn := types.TweenEase(cfg)
	g := &TweenGroup{Count: 1, Target: n}
	g.Tweens[0] = gween.New(float32(n.Alpha_), float32(to), cfg.Duration, fn)
	g.Fields[0] = &n.Alpha_
	autoRegister(g, n)
	return g
}

// TweenRotation animates node.Rotation_ to the target value.
func TweenRotation(n *node.Node, to float64, cfg types.TweenConfig) *TweenGroup {
	fn := types.TweenEase(cfg)
	g := &TweenGroup{Count: 1, Target: n}
	g.Tweens[0] = gween.New(float32(n.Rotation_), float32(to), cfg.Duration, fn)
	g.Fields[0] = &n.Rotation_
	autoRegister(g, n)
	return g
}
