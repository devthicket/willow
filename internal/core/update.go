package core

import (
	"github.com/devthicket/willow/internal/node"
	"github.com/devthicket/willow/internal/types"
)

// UpdateNodesAndParticles walks the tree depth-first, calling OnUpdate callbacks
// and ticking particle emitters. OnUpdate fires regardless of visibility (so
// tweens, AI, and gameplay logic keep running on hidden subtrees). Particle
// emitters pause when this node or any ancestor is invisible unless the
// emitter opts in via EmitterConfig.SimulateWhileHidden.
func UpdateNodesAndParticles(n *node.Node, dt float64, parentVisible bool) {
	visible := parentVisible && n.Visible_

	if n.OnUpdate != nil {
		n.OnUpdate(dt)
	}

	tickEmitter := n.Type == types.NodeTypeParticleEmitter && n.Emitter != nil &&
		(visible || n.Emitter.Config.SimulateWhileHidden)
	if tickEmitter {
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
		UpdateNodesAndParticles(child, dt, visible)
	}
}
