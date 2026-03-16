package integration

import (
	"math"
	"testing"

	"github.com/devthicket/willow/internal/node"
	"github.com/devthicket/willow/internal/render"

	. "github.com/devthicket/willow"
)

// traverseScene runs the render pipeline traverse without needing an ebiten screen.
func traverseScene(s *Scene) {
	s.Pipeline.Commands = s.Pipeline.Commands[:0]
	s.Pipeline.CommandsDirtyThisFrame = false
	node.UpdateWorldTransform(s.Root, node.IdentityTransform, 1.0, false, false)
	s.Pipeline.ViewTransform = node.IdentityTransform
	treeOrder := 0
	s.Pipeline.Traverse(s.Root, &treeOrder)
}

// traverseSceneWithCamera runs traverse with a camera's view transform.
func traverseSceneWithCamera(s *Scene, cam *Camera) {
	s.Pipeline.Commands = s.Pipeline.Commands[:0]
	s.Pipeline.CommandsDirtyThisFrame = false
	node.UpdateWorldTransform(s.Root, node.IdentityTransform, 1.0, false, false)
	if cam != nil {
		s.Pipeline.ViewTransform = cam.ComputeViewMatrix()
	} else {
		s.Pipeline.ViewTransform = node.IdentityTransform
	}
	treeOrder := 0
	s.Pipeline.Traverse(s.Root, &treeOrder)
}

// assertNear fails the test if |got-want| > 0.05.
func assertNear(t *testing.T, name string, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 0.05 {
		t.Errorf("%s = %f, want %f", name, got, want)
	}
}

// approxEqual returns true if |a-b| < eps.
func approxEqual(a, b, eps float64) bool {
	return math.Abs(a-b) < eps
}

const epsilon = 0.001

// assertVertexNear checks that a float32 is close to the expected value.
func assertVertexNear(t *testing.T, label string, got, want float32) {
	t.Helper()
	if diff := got - want; diff > 0.001 || diff < -0.001 {
		t.Errorf("%s = %f, want %f", label, got, want)
	}
}

// buildSpriteScene creates a scene with count sprite nodes.
func buildSpriteScene(count int) *Scene {
	s := NewScene()
	for i := 0; i < count; i++ {
		sp := NewSprite("", TextureRegion{Width: 32, Height: 32})
		s.Root.AddChild(sp)
	}
	return s
}

// mergeSort is a convenience wrapper for render.MergeSort.
func mergeSort(cmds []RenderCommand, buf *[]RenderCommand) {
	render.MergeSort(cmds, buf)
}
