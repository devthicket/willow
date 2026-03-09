package willow

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// --- Constructor defaults ---

func TestNewContainerDefaults(t *testing.T) {
	n := NewContainer("test")
	assertNodeDefaults(t, n, "test", NodeTypeContainer)
}

func TestNewSpriteDefaults(t *testing.T) {
	region := TextureRegion{Width: 32, Height: 32, OriginalW: 32, OriginalH: 32}
	n := NewSprite("spr", region)
	assertNodeDefaults(t, n, "spr", NodeTypeSprite)
	if n.TextureRegion_ != region {
		t.Errorf("TextureRegion = %v, want %v", n.TextureRegion_, region)
	}
}

func TestNewMeshDefaults(t *testing.T) {
	verts := []ebiten.Vertex{{DstX: 0, DstY: 0}}
	inds := []uint16{0}
	n := NewMesh("mesh", nil, verts, inds)
	assertNodeDefaults(t, n, "mesh", NodeTypeMesh)
	if len(n.Mesh.Vertices) != 1 || len(n.Mesh.Indices) != 1 {
		t.Errorf("Vertices/Indices not set")
	}
}

func TestNewParticleEmitterDefaults(t *testing.T) {
	n := NewParticleEmitter("emitter", EmitterConfig{})
	assertNodeDefaults(t, n, "emitter", NodeTypeParticleEmitter)
}

func TestNewTextDefaults(t *testing.T) {
	n := NewText("text", "hello", nil)
	assertNodeDefaults(t, n, "text", NodeTypeText)
}

func assertNodeDefaults(t *testing.T, n *Node, name string, typ NodeType) {
	t.Helper()
	if n.ID == 0 {
		t.Error("ID should be non-zero")
	}
	if n.Name != name {
		t.Errorf("Name = %q, want %q", n.Name, name)
	}
	if n.Type != typ {
		t.Errorf("Type = %d, want %d", n.Type, typ)
	}
	if n.ScaleX_ != 1 || n.ScaleY_ != 1 {
		t.Errorf("Scale = (%v, %v), want (1, 1)", n.ScaleX_, n.ScaleY_)
	}
	if n.Alpha_ != 1 {
		t.Errorf("Alpha = %v, want 1", n.Alpha_)
	}
	if n.Color_ != RGBA(1, 1, 1, 1) {
		t.Errorf("Color = %v, want white", n.Color_)
	}
	if !n.Visible_ {
		t.Error("Visible should be true")
	}
	if !n.Renderable_ {
		t.Error("Renderable should be true")
	}
	if !n.TransformDirty {
		t.Error("transformDirty should be true")
	}
}

// --- Unique IDs ---

func TestUniqueIDs(t *testing.T) {
	a := NewContainer("a")
	b := NewContainer("b")
	c := NewSprite("c", TextureRegion{})
	if a.ID == b.ID || b.ID == c.ID || a.ID == c.ID {
		t.Errorf("IDs should be unique: %d, %d, %d", a.ID, b.ID, c.ID)
	}
}
