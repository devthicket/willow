package input

import (
	"testing"

	"github.com/phanxgames/willow/internal/node"
	"github.com/phanxgames/willow/internal/types"
)

func TestNewManager(t *testing.T) {
	m := NewManager()
	if m.DragDeadZone != DefaultDragDeadZone {
		t.Errorf("DragDeadZone = %f, want %f", m.DragDeadZone, DefaultDragDeadZone)
	}
}

func TestCaptureReleasePointer(t *testing.T) {
	m := NewManager()
	n := node.NewNode("test", types.NodeTypeSprite)

	m.CapturePointer(0, n)
	if m.Captured[0] != n {
		t.Error("expected captured node")
	}
	m.ReleasePointer(0)
	if m.Captured[0] != nil {
		t.Error("expected nil after release")
	}
}

func TestCapturePointer_OutOfBounds(t *testing.T) {
	m := NewManager()
	n := node.NewNode("test", types.NodeTypeSprite)
	m.CapturePointer(-1, n)
	m.CapturePointer(MaxPointers, n)
	m.ReleasePointer(-1)
	m.ReleasePointer(MaxPointers)
}

func TestSetDragDeadZone(t *testing.T) {
	m := NewManager()
	m.SetDragDeadZone(10.0)
	if m.DragDeadZone != 10.0 {
		t.Errorf("DragDeadZone = %f, want 10", m.DragDeadZone)
	}
}

func TestCallbackHandle_Remove(t *testing.T) {
	m := NewManager()
	h := m.OnClick(func(ctx node.ClickContext) {})

	if len(m.Handlers.click) != 1 {
		t.Fatalf("expected 1 click handler, got %d", len(m.Handlers.click))
	}

	h.Remove()
	if len(m.Handlers.click) != 0 {
		t.Error("click handler should be removed")
	}
}

func TestCallbackHandle_RemoveNilReg(t *testing.T) {
	h := CallbackHandle{}
	h.Remove() // should not panic
}

func TestNodeContainsLocal_HitShape(t *testing.T) {
	n := node.NewNode("test", types.NodeTypeSprite)
	n.HitShape = testRect{0, 0, 100, 100}

	if !nodeContainsLocal(n, 50, 50) {
		t.Error("should contain (50,50)")
	}
	if nodeContainsLocal(n, 150, 50) {
		t.Error("should not contain (150,50)")
	}
}

func TestNodeContainsLocal_NoDimFn(t *testing.T) {
	old := NodeDimensionsFn
	NodeDimensionsFn = nil
	defer func() { NodeDimensionsFn = old }()

	n := node.NewNode("test", types.NodeTypeSprite)
	if nodeContainsLocal(n, 0, 0) {
		t.Error("should return false when NodeDimensionsFn is nil")
	}
}

func TestNodeContainsLocal_DimFn(t *testing.T) {
	old := NodeDimensionsFn
	NodeDimensionsFn = func(n *node.Node) (float64, float64) { return 50, 50 }
	defer func() { NodeDimensionsFn = old }()

	n := node.NewNode("test", types.NodeTypeSprite)
	if !nodeContainsLocal(n, 25, 25) {
		t.Error("should contain (25,25)")
	}
	if nodeContainsLocal(n, 60, 25) {
		t.Error("should not contain (60,25)")
	}
}

func TestCollectInteractable_SkipsInvisible(t *testing.T) {
	root := node.NewNode("root", types.NodeTypeContainer)
	root.Interactable = true
	root.Visible_ = true

	child := node.NewNode("child", types.NodeTypeSprite)
	child.Interactable = true
	child.Visible_ = false
	root.Children_ = []*node.Node{child}

	buf := collectInteractable(root, nil)
	for _, n := range buf {
		if n == child {
			t.Error("invisible child should not be collected")
		}
	}
}

func TestCollectInteractable_SkipsNonInteractable(t *testing.T) {
	root := node.NewNode("root", types.NodeTypeContainer)
	root.Interactable = true
	root.Visible_ = true

	child := node.NewNode("child", types.NodeTypeSprite)
	child.Interactable = false
	child.Visible_ = true
	root.Children_ = []*node.Node{child}

	buf := collectInteractable(root, nil)
	for _, n := range buf {
		if n == child {
			t.Error("non-interactable child should not be collected")
		}
	}
}

func TestHitTest_ReverseOrder(t *testing.T) {
	old := NodeDimensionsFn
	NodeDimensionsFn = func(n *node.Node) (float64, float64) { return 100, 100 }
	defer func() { NodeDimensionsFn = old }()

	m := NewManager()
	root := node.NewNode("root", types.NodeTypeContainer)
	root.Interactable = true
	root.Visible_ = true

	c1 := node.NewNode("c1", types.NodeTypeSprite)
	c1.Interactable = true
	c1.Visible_ = true

	c2 := node.NewNode("c2", types.NodeTypeSprite)
	c2.Interactable = true
	c2.Visible_ = true

	root.Children_ = []*node.Node{c1, c2}

	hit := m.hitTest(root, 50, 50)
	if hit != c2 {
		t.Errorf("expected c2 (topmost), got %v", hit)
	}
}

func TestInjectClick(t *testing.T) {
	m := NewManager()
	m.InjectClick(100, 200)
	if len(m.InjectQueue) != 2 {
		t.Fatalf("expected 2 events, got %d", len(m.InjectQueue))
	}
	if !m.InjectQueue[0].Pressed {
		t.Error("first event should be press")
	}
	if m.InjectQueue[1].Pressed {
		t.Error("second event should be release")
	}
}

func TestInjectDrag(t *testing.T) {
	m := NewManager()
	m.InjectDrag(0, 0, 100, 100, 5)
	// 1 press + 3 moves + 1 release = 5
	if len(m.InjectQueue) != 5 {
		t.Fatalf("expected 5 events, got %d", len(m.InjectQueue))
	}
	if !m.InjectQueue[0].Pressed {
		t.Error("first should be press")
	}
	if m.InjectQueue[4].Pressed {
		t.Error("last should be release")
	}
}

func TestInjectDrag_MinFrames(t *testing.T) {
	m := NewManager()
	m.InjectDrag(0, 0, 100, 100, 1) // should clamp to 2
	if len(m.InjectQueue) != 2 {
		t.Fatalf("expected 2 events (min), got %d", len(m.InjectQueue))
	}
}

func TestOnHandlerRegistration(t *testing.T) {
	m := NewManager()

	m.OnPointerDown(func(node.PointerContext) {})
	m.OnPointerUp(func(node.PointerContext) {})
	m.OnPointerMove(func(node.PointerContext) {})
	m.OnPointerEnter(func(node.PointerContext) {})
	m.OnPointerLeave(func(node.PointerContext) {})
	m.OnClick(func(node.ClickContext) {})
	m.OnBackgroundClick(func(node.ClickContext) {})
	m.OnDragStart(func(node.DragContext) {})
	m.OnDrag(func(node.DragContext) {})
	m.OnDragEnd(func(node.DragContext) {})
	m.OnPinch(func(node.PinchContext) {})

	if len(m.Handlers.pointerDown) != 1 {
		t.Error("expected 1 pointerDown handler")
	}
	if len(m.Handlers.pinch) != 1 {
		t.Error("expected 1 pinch handler")
	}
	if m.Handlers.nextID != 11 {
		t.Errorf("nextID = %d, want 11", m.Handlers.nextID)
	}
}

func TestSyntheticPointerEvent_Fields(t *testing.T) {
	evt := SyntheticPointerEvent{
		ScreenX: 100, ScreenY: 200,
		Pressed: true,
		Button:  types.MouseButtonRight,
	}
	if evt.ScreenX != 100 || evt.ScreenY != 200 {
		t.Error("screen coords mismatch")
	}
	if !evt.Pressed {
		t.Error("should be pressed")
	}
	if evt.Button != types.MouseButtonRight {
		t.Error("button should be right")
	}
}

// testRect implements types.HitShape for testing.
type testRect struct {
	x, y, w, h float64
}

func (r testRect) Contains(x, y float64) bool {
	return x >= r.x && x <= r.x+r.w && y >= r.y && y <= r.y+r.h
}
