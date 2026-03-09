package node

import (
	"testing"

	"github.com/phanxgames/willow/internal/types"
)

func TestNewNode(t *testing.T) {
	n := NewNode("test", types.NodeTypeSprite)
	if n.Name != "test" {
		t.Errorf("Name = %q, want %q", n.Name, "test")
	}
	if n.Type != types.NodeTypeSprite {
		t.Errorf("Type = %d, want %d", n.Type, types.NodeTypeSprite)
	}
	if n.ID == 0 {
		t.Error("ID should be non-zero")
	}
	if n.ScaleX_ != 1 || n.ScaleY_ != 1 {
		t.Errorf("Scale = (%f, %f), want (1, 1)", n.ScaleX_, n.ScaleY_)
	}
	if !n.Visible_ || !n.Renderable_ {
		t.Error("should be visible and renderable by default")
	}
}

func TestNode_AddRemoveChild(t *testing.T) {
	parent := NewNode("parent", types.NodeTypeContainer)
	child := NewNode("child", types.NodeTypeSprite)
	parent.AddChild(child)
	if len(parent.Children_) != 1 {
		t.Errorf("children count = %d, want 1", len(parent.Children_))
	}
	if child.Parent != parent {
		t.Error("child.Parent should be parent")
	}
	parent.RemoveChild(child)
	if len(parent.Children_) != 0 {
		t.Errorf("children count after remove = %d, want 0", len(parent.Children_))
	}
	if child.Parent != nil {
		t.Error("child.Parent should be nil after remove")
	}
}

func TestNode_AddChild_Reparent(t *testing.T) {
	a := NewNode("a", types.NodeTypeContainer)
	b := NewNode("b", types.NodeTypeContainer)
	child := NewNode("child", types.NodeTypeSprite)
	a.AddChild(child)
	b.AddChild(child) // should auto-remove from a
	if len(a.Children_) != 0 {
		t.Error("a should have 0 children after reparent")
	}
	if len(b.Children_) != 1 {
		t.Error("b should have 1 child after reparent")
	}
	if child.Parent != b {
		t.Error("child.Parent should be b")
	}
}

func TestNode_AddChild_CyclePanics(t *testing.T) {
	parent := NewNode("parent", types.NodeTypeContainer)
	child := NewNode("child", types.NodeTypeContainer)
	parent.AddChild(child)

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic on cycle")
		}
	}()
	child.AddChild(parent) // should panic
}

func TestNode_RemoveChildren(t *testing.T) {
	parent := NewNode("parent", types.NodeTypeContainer)
	for i := 0; i < 5; i++ {
		parent.AddChild(NewNode("child", types.NodeTypeSprite))
	}
	parent.RemoveChildren()
	if len(parent.Children_) != 0 {
		t.Errorf("children count = %d, want 0", len(parent.Children_))
	}
}

func TestNode_FindChild(t *testing.T) {
	parent := NewNode("parent", types.NodeTypeContainer)
	a := NewNode("alpha", types.NodeTypeSprite)
	b := NewNode("beta", types.NodeTypeSprite)
	parent.AddChild(a)
	parent.AddChild(b)

	found := parent.FindChild("alpha")
	if found != a {
		t.Error("FindChild('alpha') should return a")
	}
	found = parent.FindChild("gamma")
	if found != nil {
		t.Error("FindChild('gamma') should return nil")
	}
	found = parent.FindChild("al%")
	if found != a {
		t.Error("FindChild('al%') should match alpha")
	}
}

func TestNode_FindDescendant(t *testing.T) {
	root := NewNode("root", types.NodeTypeContainer)
	child := NewNode("child", types.NodeTypeContainer)
	grandchild := NewNode("target", types.NodeTypeSprite)
	root.AddChild(child)
	child.AddChild(grandchild)

	found := root.FindDescendant("target")
	if found != grandchild {
		t.Error("FindDescendant should find grandchild")
	}
}

func TestNode_Dispose(t *testing.T) {
	parent := NewNode("parent", types.NodeTypeContainer)
	child := NewNode("child", types.NodeTypeSprite)
	parent.AddChild(child)
	child.Dispose()
	if !child.IsDisposed() {
		t.Error("child should be disposed")
	}
	if len(parent.Children_) != 0 {
		t.Error("parent should have 0 children after child dispose")
	}
}

func TestNode_SetPosition(t *testing.T) {
	n := NewNode("n", types.NodeTypeSprite)
	n.SetPosition(100, 200)
	if n.X() != 100 || n.Y() != 200 {
		t.Errorf("position = (%f, %f), want (100, 200)", n.X(), n.Y())
	}
	if !n.TransformDirty {
		t.Error("TransformDirty should be true after SetPosition")
	}
}

func TestNode_SetAlpha(t *testing.T) {
	n := NewNode("n", types.NodeTypeSprite)
	n.SetAlpha(0.5)
	if n.Alpha() != 0.5 {
		t.Errorf("alpha = %f, want 0.5", n.Alpha())
	}
	if !n.AlphaDirty {
		t.Error("AlphaDirty should be true after SetAlpha")
	}
}

func TestNode_SetZIndex(t *testing.T) {
	parent := NewNode("parent", types.NodeTypeContainer)
	child := NewNode("child", types.NodeTypeSprite)
	parent.AddChild(child)
	parent.ChildrenSorted = true
	child.SetZIndex(5)
	if child.ZIndex() != 5 {
		t.Errorf("ZIndex = %d, want 5", child.ZIndex())
	}
	if parent.ChildrenSorted {
		t.Error("parent.ChildrenSorted should be false after SetZIndex")
	}
}

// --- CacheAsTexture tests migrated from archived rendertarget_test.go ---

func TestSetCacheAsTextureEnabled(t *testing.T) {
	n := NewNode("s", types.NodeTypeSprite)
	n.SetCacheAsTexture(true)
	if !n.CacheEnabled {
		t.Error("cacheEnabled should be true")
	}
	if !n.CacheDirty {
		t.Error("cacheDirty should be true after enabling")
	}
}

func TestSetCacheAsTextureDisabled(t *testing.T) {
	n := NewNode("s", types.NodeTypeSprite)
	n.SetCacheAsTexture(true)
	n.SetCacheAsTexture(false)
	if n.CacheEnabled {
		t.Error("cacheEnabled should be false after disabling")
	}
	if n.CacheDirty {
		t.Error("cacheDirty should be false after disabling")
	}
	if n.CacheTexture != nil {
		t.Error("cacheTexture should be nil after disabling")
	}
}

func TestSetCacheAsTextureIdempotent(t *testing.T) {
	n := NewNode("s", types.NodeTypeSprite)
	n.SetCacheAsTexture(true)
	n.CacheDirty = false
	n.SetCacheAsTexture(true) // should be no-op
	if n.CacheDirty {
		t.Error("setting cache to same value should not reset dirty flag")
	}
}

func TestInvalidateCache(t *testing.T) {
	n := NewNode("s", types.NodeTypeSprite)
	n.SetCacheAsTexture(true)
	n.CacheDirty = false
	n.InvalidateCache()
	if !n.CacheDirty {
		t.Error("cacheDirty should be true after InvalidateCache")
	}
}

func TestInvalidateCacheNoCacheNoOp(t *testing.T) {
	n := NewNode("s", types.NodeTypeSprite)
	n.InvalidateCache() // cache not enabled - should be no-op
	if n.CacheDirty {
		t.Error("InvalidateCache on non-cached node should not set dirty")
	}
}

func TestIsCacheEnabled(t *testing.T) {
	n := NewNode("s", types.NodeTypeSprite)
	if n.IsCacheEnabled() {
		t.Error("should be false by default")
	}
	n.SetCacheAsTexture(true)
	if !n.IsCacheEnabled() {
		t.Error("should be true after enabling")
	}
}
