package core

import (
	"testing"

	"github.com/devthicket/willow/internal/node"
	"github.com/devthicket/willow/internal/particle"
	"github.com/devthicket/willow/internal/types"
)

// buildEmitterNode returns a particle node with an active emitter that spawns
// 1 particle per 1/60s frame so a single Update call produces Alive=1.
func buildEmitterNode(name string) *node.Node {
	n := node.NewNode(name, types.NodeTypeParticleEmitter)
	n.Emitter = particle.NewEmitter(particle.EmitterConfig{
		MaxParticles: 16,
		EmitRate:     60,
		Lifetime:     types.Range{Min: 1, Max: 1},
		StartScale:   types.Range{Min: 1, Max: 1},
		EndScale:     types.Range{Min: 1, Max: 1},
		StartAlpha:   types.Range{Min: 1, Max: 1},
		EndAlpha:     types.Range{Min: 1, Max: 1},
		StartColor:   types.RGB(1, 1, 1),
		EndColor:     types.RGB(1, 1, 1),
	})
	n.Emitter.Start()
	return n
}

func TestUpdateNodesAndParticles_ParticlesPauseWhenInvisible(t *testing.T) {
	n := buildEmitterNode("emitter")
	n.Visible_ = false
	UpdateNodesAndParticles(n, 1.0/60.0, true)
	if n.Emitter.AliveCount() != 0 {
		t.Errorf("emitter on invisible node should not tick; alive = %d, want 0", n.Emitter.AliveCount())
	}
}

func TestUpdateNodesAndParticles_ParticlesPauseWhenAncestorInvisible(t *testing.T) {
	parent := node.NewNode("parent", types.NodeTypeContainer)
	parent.Visible_ = false
	child := buildEmitterNode("emitter")
	parent.AddChild(child)

	UpdateNodesAndParticles(parent, 1.0/60.0, true)
	if child.Emitter.AliveCount() != 0 {
		t.Errorf("emitter under invisible ancestor should not tick; alive = %d, want 0", child.Emitter.AliveCount())
	}
}

func TestUpdateNodesAndParticles_ParticlesRunWhenVisible(t *testing.T) {
	n := buildEmitterNode("emitter")
	UpdateNodesAndParticles(n, 1.0/60.0, true)
	if n.Emitter.AliveCount() != 1 {
		t.Errorf("visible emitter should spawn 1 particle/frame at 60Hz; alive = %d, want 1", n.Emitter.AliveCount())
	}
}

func TestUpdateNodesAndParticles_SimulateWhileHiddenOverridesVisibilityGate(t *testing.T) {
	n := buildEmitterNode("emitter")
	n.Visible_ = false
	n.Emitter.Config.SimulateWhileHidden = true
	UpdateNodesAndParticles(n, 1.0/60.0, true)
	if n.Emitter.AliveCount() != 1 {
		t.Errorf("emitter with SimulateWhileHidden=true should tick while invisible; alive = %d, want 1", n.Emitter.AliveCount())
	}
}

func TestUpdateNodesAndParticles_OnUpdateFiresOnInvisibleSubtree(t *testing.T) {
	parent := node.NewNode("parent", types.NodeTypeContainer)
	parent.Visible_ = false
	child := node.NewNode("child", types.NodeTypeContainer)
	parent.AddChild(child)

	ticks := 0
	child.OnUpdate = func(dt float64) { ticks++ }

	UpdateNodesAndParticles(parent, 1.0/60.0, true)
	if ticks != 1 {
		t.Errorf("OnUpdate should fire on hidden subtree; ticks = %d, want 1", ticks)
	}
}
