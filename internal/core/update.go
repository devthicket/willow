package core

import (
	"github.com/phanxgames/willow/internal/node"
	"github.com/phanxgames/willow/internal/types"
)

// UpdateNodesAndParticles walks the tree depth-first, calling OnUpdate callbacks
// and ticking particle emitters.
func UpdateNodesAndParticles(n *node.Node, dt float64) {
	if !n.Visible_ {
		return
	}
	if n.OnUpdate != nil {
		n.OnUpdate(dt)
	}
	if n.Type == types.NodeTypeParticleEmitter && n.Emitter != nil {
		if n.Emitter.Config.WorldSpace {
			n.Emitter.WorldX = n.WorldTransform[4]
			n.Emitter.WorldY = n.WorldTransform[5]
		}
		n.Emitter.Update(dt)
		if n.Emitter.Alive > 0 {
			if node.InvalidateAncestorCacheFn != nil {
				node.InvalidateAncestorCacheFn(n)
			}
		}
	}
	for _, child := range n.Children_ {
		UpdateNodesAndParticles(child, dt)
	}
}
