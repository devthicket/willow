package node

import (
	"math"
	"testing"

	"github.com/devthicket/willow/internal/types"
)

func TestDistanceBetween(t *testing.T) {
	a := NewNode("a", types.NodeTypeSprite)
	b := NewNode("b", types.NodeTypeSprite)
	a.WorldTransform = [6]float64{1, 0, 0, 1, 0, 0}
	b.WorldTransform = [6]float64{1, 0, 0, 1, 3, 4}
	dist := DistanceBetween(a, b)
	if math.Abs(dist-5.0) > 1e-9 {
		t.Errorf("distance = %f, want 5.0", dist)
	}
}

func TestDistanceBetweenSamePoint(t *testing.T) {
	a := NewNode("a", types.NodeTypeSprite)
	b := NewNode("b", types.NodeTypeSprite)
	a.WorldTransform = [6]float64{1, 0, 0, 1, 10, 20}
	b.WorldTransform = [6]float64{1, 0, 0, 1, 10, 20}
	if DistanceBetween(a, b) != 0 {
		t.Error("distance between same point should be 0")
	}
}

func TestDirectionBetween(t *testing.T) {
	a := NewNode("a", types.NodeTypeSprite)
	b := NewNode("b", types.NodeTypeSprite)
	a.WorldTransform = [6]float64{1, 0, 0, 1, 0, 0}
	b.WorldTransform = [6]float64{1, 0, 0, 1, 10, 0}
	dx, dy := DirectionBetween(a, b)
	if math.Abs(dx-1.0) > 1e-9 || math.Abs(dy) > 1e-9 {
		t.Errorf("direction = (%f, %f), want (1, 0)", dx, dy)
	}
}

func TestDirectionBetweenSamePoint(t *testing.T) {
	a := NewNode("a", types.NodeTypeSprite)
	b := NewNode("b", types.NodeTypeSprite)
	a.WorldTransform = [6]float64{1, 0, 0, 1, 5, 5}
	b.WorldTransform = [6]float64{1, 0, 0, 1, 5, 5}
	dx, dy := DirectionBetween(a, b)
	if dx != 0 || dy != 0 {
		t.Errorf("direction between same point = (%f, %f), want (0, 0)", dx, dy)
	}
}

func TestDirectionBetweenDiagonal(t *testing.T) {
	a := NewNode("a", types.NodeTypeSprite)
	b := NewNode("b", types.NodeTypeSprite)
	a.WorldTransform = [6]float64{1, 0, 0, 1, 0, 0}
	b.WorldTransform = [6]float64{1, 0, 0, 1, 1, 1}
	dx, dy := DirectionBetween(a, b)
	expected := 1.0 / math.Sqrt(2)
	if math.Abs(dx-expected) > 1e-9 || math.Abs(dy-expected) > 1e-9 {
		t.Errorf("direction = (%f, %f), want (%f, %f)", dx, dy, expected, expected)
	}
}
