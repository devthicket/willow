package node

import (
	"testing"

	"github.com/devthicket/willow/internal/types"
)

func TestWorldPosition(t *testing.T) {
	n := NewNode("test", types.NodeTypeSprite)
	n.WorldTransform = [6]float64{1, 0, 0, 1, 42, 99}
	x, y := n.WorldPosition()
	if x != 42 || y != 99 {
		t.Errorf("WorldPosition = (%f, %f), want (42, 99)", x, y)
	}
}

func TestGetWorldAlpha(t *testing.T) {
	n := NewNode("test", types.NodeTypeSprite)
	n.WorldAlpha = 0.75
	if n.GetWorldAlpha() != 0.75 {
		t.Errorf("GetWorldAlpha = %f, want 0.75", n.GetWorldAlpha())
	}
}

func TestWorldBoundsSprite(t *testing.T) {
	n := NewNode("test", types.NodeTypeSprite)
	n.TextureRegion_ = types.TextureRegion{OriginalW: 64, OriginalH: 32}
	n.WorldTransform = [6]float64{1, 0, 0, 1, 100, 200}
	bounds := n.WorldBounds()
	if bounds.X != 100 || bounds.Y != 200 || bounds.Width != 64 || bounds.Height != 32 {
		t.Errorf("WorldBounds = %v, want {100, 200, 64, 32}", bounds)
	}
}

func TestWorldBoundsContainer(t *testing.T) {
	n := NewNode("test", types.NodeTypeContainer)
	n.WorldTransform = [6]float64{1, 0, 0, 1, 50, 60}
	bounds := n.WorldBounds()
	if bounds.X != 50 || bounds.Y != 60 || bounds.Width != 0 || bounds.Height != 0 {
		t.Errorf("container WorldBounds = %v, want {50, 60, 0, 0}", bounds)
	}
}
