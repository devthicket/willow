package input

import (
	"testing"

	"github.com/devthicket/willow/internal/node"
	"github.com/devthicket/willow/internal/types"
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

// --- helpers ---

// testRect implements types.HitShape for testing.
type testRect struct {
	x, y, w, h float64
}

func (r testRect) Contains(x, y float64) bool {
	return x >= r.x && x <= r.x+r.w && y >= r.y && y <= r.y+r.h
}

// setupIdentityTransforms sets ScreenToWorldFn and NodeDimensionsFn to identity/fixed
// values so that processPointer works without a real camera. Returns a cleanup function.
func setupIdentityTransforms(dimW, dimH float64) func() {
	oldSTW := ScreenToWorldFn
	oldDim := NodeDimensionsFn
	oldRebuild := RebuildSortedChildrenFn
	oldEmit := EmitInteractionEventFn

	ScreenToWorldFn = func(sx, sy float64) (float64, float64) { return sx, sy }
	NodeDimensionsFn = func(n *node.Node) (float64, float64) { return dimW, dimH }
	RebuildSortedChildrenFn = nil
	EmitInteractionEventFn = nil

	return func() {
		ScreenToWorldFn = oldSTW
		NodeDimensionsFn = oldDim
		RebuildSortedChildrenFn = oldRebuild
		EmitInteractionEventFn = oldEmit
	}
}

// makeInteractableNode creates a visible, interactable sprite node.
func makeInteractableNode(name string) *node.Node {
	n := node.NewNode(name, types.NodeTypeSprite)
	n.Interactable = true
	n.Visible_ = true
	return n
}

// makeRoot creates a visible, interactable container with children.
func makeRoot(children ...*node.Node) *node.Node {
	root := node.NewNode("root", types.NodeTypeContainer)
	root.Interactable = true
	root.Visible_ = true
	root.Children_ = children
	return root
}

// --- ProcessInjectedInput tests ---

func TestProcessInjectedInput_PressChangesState(t *testing.T) {
	cleanup := setupIdentityTransforms(100, 100)
	defer cleanup()

	m := NewManager()
	child := makeInteractableNode("btn")
	root := makeRoot(child)

	m.InjectPress(50, 50)
	consumed := m.ProcessInjectedInput(root, 0)
	if !consumed {
		t.Fatal("expected injected input to be consumed")
	}
	if !m.Pointers[0].down {
		t.Error("pointer should be down after injected press")
	}
	if m.Pointers[0].HitNode != child {
		t.Error("HitNode should be the child node")
	}
	if m.Pointers[0].startX != 50 || m.Pointers[0].startY != 50 {
		t.Error("start coordinates should match injected position")
	}
}

func TestProcessInjectedInput_EmptyQueue(t *testing.T) {
	m := NewManager()
	root := makeRoot()
	consumed := m.ProcessInjectedInput(root, 0)
	if consumed {
		t.Error("should return false for empty queue")
	}
}

func TestProcessInjectedInput_ConsumesOnePerCall(t *testing.T) {
	cleanup := setupIdentityTransforms(100, 100)
	defer cleanup()

	m := NewManager()
	root := makeRoot()

	m.InjectPress(10, 10)
	m.InjectRelease(10, 10)
	if len(m.InjectQueue) != 2 {
		t.Fatalf("expected 2 queued events, got %d", len(m.InjectQueue))
	}

	m.ProcessInjectedInput(root, 0)
	if len(m.InjectQueue) != 1 {
		t.Errorf("expected 1 remaining event, got %d", len(m.InjectQueue))
	}

	m.ProcessInjectedInput(root, 0)
	if len(m.InjectQueue) != 0 {
		t.Errorf("expected 0 remaining events, got %d", len(m.InjectQueue))
	}
}

// --- Click event firing tests ---

func TestClick_SceneHandler(t *testing.T) {
	cleanup := setupIdentityTransforms(100, 100)
	defer cleanup()

	m := NewManager()
	child := makeInteractableNode("btn")
	root := makeRoot(child)

	var got node.ClickContext
	var fired bool
	m.OnClick(func(ctx node.ClickContext) {
		fired = true
		got = ctx
	})

	// Frame 1: press
	m.ProcessPointer(root, 0, 50, 50, 50, 50, true, types.MouseButtonLeft, 0)
	// Frame 2: release on same node
	m.ProcessPointer(root, 0, 50, 50, 50, 50, false, types.MouseButtonLeft, 0)

	if !fired {
		t.Fatal("OnClick handler should have fired")
	}
	if got.Node != child {
		t.Error("click node should be the child")
	}
	if got.GlobalX != 50 || got.GlobalY != 50 {
		t.Errorf("click coords = (%v,%v), want (50,50)", got.GlobalX, got.GlobalY)
	}
	if got.Button != types.MouseButtonLeft {
		t.Error("button should be left")
	}
}

func TestClick_NodeCallback(t *testing.T) {
	cleanup := setupIdentityTransforms(100, 100)
	defer cleanup()

	m := NewManager()
	child := makeInteractableNode("btn")
	root := makeRoot(child)

	var fired bool
	child.EnsureCallbacks().OnClick = func(ctx node.ClickContext) {
		fired = true
	}

	m.ProcessPointer(root, 0, 50, 50, 50, 50, true, types.MouseButtonLeft, 0)
	m.ProcessPointer(root, 0, 50, 50, 50, 50, false, types.MouseButtonLeft, 0)

	if !fired {
		t.Fatal("node OnClick callback should have fired")
	}
}

func TestBackgroundClick(t *testing.T) {
	cleanup := setupIdentityTransforms(100, 100)
	defer cleanup()

	m := NewManager()
	// No children - clicking hits nothing
	root := makeRoot()

	var fired bool
	m.OnBackgroundClick(func(ctx node.ClickContext) {
		fired = true
	})

	// Press and release on empty area
	m.ProcessPointer(root, 0, 50, 50, 50, 50, true, types.MouseButtonLeft, 0)
	m.ProcessPointer(root, 0, 50, 50, 50, 50, false, types.MouseButtonLeft, 0)

	if !fired {
		t.Fatal("OnBackgroundClick should have fired when clicking on nothing")
	}
}

func TestClick_DoesNotFireOnDifferentNode(t *testing.T) {
	cleanup := setupIdentityTransforms(0, 0)
	defer cleanup()

	m := NewManager()
	// Two non-overlapping nodes using HitShape
	child1 := makeInteractableNode("c1")
	child1.HitShape = HitRect{X: 0, Y: 0, Width: 50, Height: 50}
	child2 := makeInteractableNode("c2")
	child2.HitShape = HitRect{X: 60, Y: 0, Width: 50, Height: 50}
	root := makeRoot(child1, child2)

	var clicked bool
	m.OnClick(func(ctx node.ClickContext) {
		clicked = true
	})

	// Press on child1, release on child2 (different hit target) => no click
	m.ProcessPointer(root, 0, 25, 25, 25, 25, true, types.MouseButtonLeft, 0)
	m.ProcessPointer(root, 0, 85, 25, 85, 25, false, types.MouseButtonLeft, 0)

	if clicked {
		t.Error("OnClick should not fire when press and release are on different nodes")
	}
}

// --- Drag detection tests ---

func TestDrag_DetectsAfterDeadzone(t *testing.T) {
	cleanup := setupIdentityTransforms(200, 200)
	defer cleanup()

	m := NewManager()
	m.SetDragDeadZone(5.0)
	child := makeInteractableNode("drag-target")
	root := makeRoot(child)

	var dragStartFired, dragFired, dragEndFired bool
	var dragStartCtx node.DragContext
	m.OnDragStart(func(ctx node.DragContext) {
		dragStartFired = true
		dragStartCtx = ctx
	})
	m.OnDrag(func(ctx node.DragContext) {
		dragFired = true
	})
	m.OnDragEnd(func(ctx node.DragContext) {
		dragEndFired = true
	})

	// Press at (50,50)
	m.ProcessPointer(root, 0, 50, 50, 50, 50, true, types.MouseButtonLeft, 0)

	// Move within deadzone (2 px) - should NOT trigger drag
	m.ProcessPointer(root, 0, 52, 50, 52, 50, true, types.MouseButtonLeft, 0)
	if dragStartFired {
		t.Error("drag should not start within deadzone")
	}

	// Move beyond deadzone (10 px)
	m.ProcessPointer(root, 0, 60, 50, 60, 50, true, types.MouseButtonLeft, 0)
	if !dragStartFired {
		t.Fatal("OnDragStart should fire after exceeding deadzone")
	}
	if dragStartCtx.StartX != 50 || dragStartCtx.StartY != 50 {
		t.Error("drag start coords should match original press")
	}
	if !dragFired {
		t.Error("OnDrag should fire on the same frame as OnDragStart")
	}

	// Continue drag
	dragFired = false
	m.ProcessPointer(root, 0, 70, 50, 70, 50, true, types.MouseButtonLeft, 0)
	if !dragFired {
		t.Error("OnDrag should continue firing on movement")
	}

	// Release
	m.ProcessPointer(root, 0, 70, 50, 70, 50, false, types.MouseButtonLeft, 0)
	if !dragEndFired {
		t.Fatal("OnDragEnd should fire on release after dragging")
	}
}

func TestDrag_NodeCallback(t *testing.T) {
	cleanup := setupIdentityTransforms(200, 200)
	defer cleanup()

	m := NewManager()
	m.SetDragDeadZone(2.0)
	child := makeInteractableNode("drag-node")
	root := makeRoot(child)

	var startFired, moveFired, endFired bool
	child.EnsureCallbacks().OnDragStart = func(ctx node.DragContext) { startFired = true }
	child.EnsureCallbacks().OnDrag = func(ctx node.DragContext) { moveFired = true }
	child.EnsureCallbacks().OnDragEnd = func(ctx node.DragContext) { endFired = true }

	m.ProcessPointer(root, 0, 50, 50, 50, 50, true, types.MouseButtonLeft, 0)
	m.ProcessPointer(root, 0, 60, 60, 60, 60, true, types.MouseButtonLeft, 0)
	m.ProcessPointer(root, 0, 70, 70, 70, 70, false, types.MouseButtonLeft, 0)

	if !startFired {
		t.Error("node OnDragStart should have fired")
	}
	if !moveFired {
		t.Error("node OnDrag should have fired")
	}
	if !endFired {
		t.Error("node OnDragEnd should have fired")
	}
}

// --- Pointer enter/leave tests ---

func TestPointerEnterLeave_SceneHandlers(t *testing.T) {
	cleanup := setupIdentityTransforms(50, 50)
	defer cleanup()

	m := NewManager()
	child := makeInteractableNode("hover-target")
	root := makeRoot(child)

	var enterFired, leaveFired bool
	var enterNode, leaveNode *node.Node
	m.OnPointerEnter(func(ctx node.PointerContext) {
		enterFired = true
		enterNode = ctx.Node
	})
	m.OnPointerLeave(func(ctx node.PointerContext) {
		leaveFired = true
		leaveNode = ctx.Node
	})

	// Move over the child (not pressed, no button)
	m.ProcessPointer(root, 0, 25, 25, 25, 25, false, 0, 0)
	if !enterFired {
		t.Error("OnPointerEnter should fire when pointer enters a node")
	}
	if enterNode != child {
		t.Error("enter node should be the child")
	}

	// Move outside the child
	m.ProcessPointer(root, 0, 200, 200, 200, 200, false, 0, 0)
	if !leaveFired {
		t.Error("OnPointerLeave should fire when pointer leaves a node")
	}
	if leaveNode != child {
		t.Error("leave node should be the child we left")
	}
}

func TestPointerEnterLeave_NodeCallbacks(t *testing.T) {
	cleanup := setupIdentityTransforms(50, 50)
	defer cleanup()

	m := NewManager()
	child := makeInteractableNode("hover-node")
	root := makeRoot(child)

	var enterFired, leaveFired bool
	child.EnsureCallbacks().OnPointerEnter = func(ctx node.PointerContext) { enterFired = true }
	child.EnsureCallbacks().OnPointerLeave = func(ctx node.PointerContext) { leaveFired = true }

	m.ProcessPointer(root, 0, 25, 25, 25, 25, false, 0, 0)
	if !enterFired {
		t.Error("node OnPointerEnter should fire")
	}

	m.ProcessPointer(root, 0, 200, 200, 200, 200, false, 0, 0)
	if !leaveFired {
		t.Error("node OnPointerLeave should fire")
	}
}

func TestPointerEnterLeave_SwitchBetweenNodes(t *testing.T) {
	cleanup := setupIdentityTransforms(0, 0) // won't use dim fn; using HitShape
	defer cleanup()

	m := NewManager()
	child1 := makeInteractableNode("a")
	child1.HitShape = HitRect{X: 0, Y: 0, Width: 40, Height: 40}
	child2 := makeInteractableNode("b")
	child2.HitShape = HitRect{X: 50, Y: 0, Width: 40, Height: 40}
	root := makeRoot(child1, child2)

	var leaves, enters []*node.Node
	m.OnPointerEnter(func(ctx node.PointerContext) { enters = append(enters, ctx.Node) })
	m.OnPointerLeave(func(ctx node.PointerContext) { leaves = append(leaves, ctx.Node) })

	// Hover child1
	m.ProcessPointer(root, 0, 20, 20, 20, 20, false, 0, 0)
	// Hover child2
	m.ProcessPointer(root, 0, 70, 20, 70, 20, false, 0, 0)

	if len(enters) != 2 {
		t.Fatalf("expected 2 enter events, got %d", len(enters))
	}
	if enters[0] != child1 || enters[1] != child2 {
		t.Error("enter order should be child1 then child2")
	}
	if len(leaves) != 1 || leaves[0] != child1 {
		t.Error("should have one leave event for child1")
	}
}

func TestPointerLeave_DisposedNode(t *testing.T) {
	cleanup := setupIdentityTransforms(50, 50)
	defer cleanup()

	m := NewManager()
	child := makeInteractableNode("disposable")
	root := makeRoot(child)

	// Enter the node
	m.ProcessPointer(root, 0, 25, 25, 25, 25, false, 0, 0)

	// Dispose the node; hoverNode should be cleared on next call
	child.Disposed = true
	root.Children_ = nil

	var leaveFired bool
	m.OnPointerLeave(func(ctx node.PointerContext) { leaveFired = true })

	// Move to empty area — should not panic
	m.ProcessPointer(root, 0, 200, 200, 200, 200, false, 0, 0)
	if leaveFired {
		t.Error("should not fire leave for disposed node (hoverNode cleared to nil)")
	}
}

// --- HitRect tests ---

func TestHitRect_Contains(t *testing.T) {
	r := HitRect{X: 10, Y: 20, Width: 100, Height: 50}

	tests := []struct {
		x, y float64
		want bool
		name string
	}{
		{60, 45, true, "center"},
		{10, 20, true, "top-left corner"},
		{110, 70, true, "bottom-right corner"},
		{10, 70, true, "bottom-left corner"},
		{110, 20, true, "top-right corner"},
		{9, 45, false, "left of rect"},
		{111, 45, false, "right of rect"},
		{60, 19, false, "above rect"},
		{60, 71, false, "below rect"},
	}

	for _, tt := range tests {
		if got := r.Contains(tt.x, tt.y); got != tt.want {
			t.Errorf("HitRect.Contains(%v,%v) [%s] = %v, want %v", tt.x, tt.y, tt.name, got, tt.want)
		}
	}
}

func TestHitRect_ZeroSize(t *testing.T) {
	r := HitRect{X: 0, Y: 0, Width: 0, Height: 0}
	if !r.Contains(0, 0) {
		t.Error("zero-size rect at origin should contain (0,0)")
	}
	if r.Contains(1, 0) {
		t.Error("zero-size rect should not contain (1,0)")
	}
}

// --- HitCircle tests ---

func TestHitCircle_Contains(t *testing.T) {
	c := HitCircle{CenterX: 50, CenterY: 50, Radius: 25}

	tests := []struct {
		x, y float64
		want bool
		name string
	}{
		{50, 50, true, "center"},
		{75, 50, true, "right edge"},
		{25, 50, true, "left edge"},
		{50, 75, true, "bottom edge"},
		{50, 25, true, "top edge"},
		{76, 50, false, "just outside right"},
		{50, 76, false, "just outside bottom"},
		{0, 0, false, "far away"},
	}

	for _, tt := range tests {
		if got := c.Contains(tt.x, tt.y); got != tt.want {
			t.Errorf("HitCircle.Contains(%v,%v) [%s] = %v, want %v", tt.x, tt.y, tt.name, got, tt.want)
		}
	}
}

func TestHitCircle_ZeroRadius(t *testing.T) {
	c := HitCircle{CenterX: 10, CenterY: 10, Radius: 0}
	if !c.Contains(10, 10) {
		t.Error("zero-radius circle should contain its center")
	}
	if c.Contains(11, 10) {
		t.Error("zero-radius circle should not contain adjacent point")
	}
}

// --- HitPolygon tests ---

func TestHitPolygon_Triangle(t *testing.T) {
	p := HitPolygon{
		Points: []types.Vec2{
			{X: 0, Y: 0},
			{X: 100, Y: 0},
			{X: 50, Y: 100},
		},
	}

	if !p.Contains(50, 30) {
		t.Error("should contain center-ish point")
	}
	if p.Contains(-10, 50) {
		t.Error("should not contain point outside left")
	}
	if p.Contains(50, 110) {
		t.Error("should not contain point below")
	}
}

func TestHitPolygon_Square(t *testing.T) {
	p := HitPolygon{
		Points: []types.Vec2{
			{X: 0, Y: 0},
			{X: 100, Y: 0},
			{X: 100, Y: 100},
			{X: 0, Y: 100},
		},
	}

	if !p.Contains(50, 50) {
		t.Error("center should be inside")
	}
	if p.Contains(101, 50) {
		t.Error("outside right edge should not be inside")
	}
}

func TestHitPolygon_TooFewPoints(t *testing.T) {
	p := HitPolygon{Points: []types.Vec2{{X: 0, Y: 0}, {X: 1, Y: 1}}}
	if p.Contains(0, 0) {
		t.Error("polygon with < 3 points should never contain anything")
	}

	empty := HitPolygon{}
	if empty.Contains(0, 0) {
		t.Error("empty polygon should never contain anything")
	}
}

// --- Pointer capture tests ---

func TestCapturedPointer_RoutesToCapturedNode(t *testing.T) {
	cleanup := setupIdentityTransforms(0, 0)
	defer cleanup()

	m := NewManager()
	child1 := makeInteractableNode("c1")
	child1.HitShape = HitRect{X: 0, Y: 0, Width: 50, Height: 50}
	child2 := makeInteractableNode("c2")
	child2.HitShape = HitRect{X: 60, Y: 0, Width: 50, Height: 50}
	root := makeRoot(child1, child2)

	// Capture pointer to child1
	m.CapturePointer(0, child1)

	var downNode *node.Node
	m.OnPointerDown(func(ctx node.PointerContext) {
		downNode = ctx.Node
	})

	// Press at coords over child2 — should still route to child1 due to capture
	m.ProcessPointer(root, 0, 80, 25, 80, 25, true, types.MouseButtonLeft, 0)

	if downNode != child1 {
		t.Errorf("pointer down should route to captured node child1, got %v", downNode)
	}
}

func TestCapturedPointer_ReleasedOnPointerUp(t *testing.T) {
	cleanup := setupIdentityTransforms(50, 50)
	defer cleanup()

	m := NewManager()
	child := makeInteractableNode("btn")
	root := makeRoot(child)

	m.CapturePointer(0, child)
	if m.Captured[0] != child {
		t.Fatal("capture should be set")
	}

	// Press then release
	m.ProcessPointer(root, 0, 25, 25, 25, 25, true, types.MouseButtonLeft, 0)
	m.ProcessPointer(root, 0, 25, 25, 25, 25, false, types.MouseButtonLeft, 0)

	if m.Captured[0] != nil {
		t.Error("capture should be cleared after pointer up")
	}
}

// --- PointerDown / PointerUp tests ---

func TestPointerDown_SceneHandler(t *testing.T) {
	cleanup := setupIdentityTransforms(100, 100)
	defer cleanup()

	m := NewManager()
	child := makeInteractableNode("btn")
	root := makeRoot(child)

	var downFired bool
	m.OnPointerDown(func(ctx node.PointerContext) {
		downFired = true
	})

	m.ProcessPointer(root, 0, 50, 50, 50, 50, true, types.MouseButtonLeft, 0)
	if !downFired {
		t.Error("OnPointerDown should fire on press")
	}
}

func TestPointerUp_SceneHandler(t *testing.T) {
	cleanup := setupIdentityTransforms(100, 100)
	defer cleanup()

	m := NewManager()
	child := makeInteractableNode("btn")
	root := makeRoot(child)

	var upFired bool
	m.OnPointerUp(func(ctx node.PointerContext) {
		upFired = true
	})

	m.ProcessPointer(root, 0, 50, 50, 50, 50, true, types.MouseButtonLeft, 0)
	m.ProcessPointer(root, 0, 50, 50, 50, 50, false, types.MouseButtonLeft, 0)
	if !upFired {
		t.Error("OnPointerUp should fire on release")
	}
}

// --- PointerMove tests ---

func TestPointerMove_FiresOnMovementWhileNotPressed(t *testing.T) {
	cleanup := setupIdentityTransforms(100, 100)
	defer cleanup()

	m := NewManager()
	child := makeInteractableNode("target")
	root := makeRoot(child)

	var moveFired bool
	m.OnPointerMove(func(ctx node.PointerContext) {
		moveFired = true
	})

	// First position
	m.ProcessPointer(root, 0, 10, 10, 10, 10, false, 0, 0)
	// Move to new position (not pressed)
	m.ProcessPointer(root, 0, 20, 20, 20, 20, false, 0, 0)

	if !moveFired {
		t.Error("OnPointerMove should fire on movement while not pressed")
	}
}

func TestPointerMove_DoesNotFireWithoutMovement(t *testing.T) {
	cleanup := setupIdentityTransforms(100, 100)
	defer cleanup()

	m := NewManager()
	root := makeRoot()

	var moveFired bool
	m.OnPointerMove(func(ctx node.PointerContext) {
		moveFired = true
	})

	// Same position twice
	m.ProcessPointer(root, 0, 10, 10, 10, 10, false, 0, 0)
	moveFired = false
	m.ProcessPointer(root, 0, 10, 10, 10, 10, false, 0, 0)

	if moveFired {
		t.Error("OnPointerMove should not fire without movement")
	}
}

// --- Injected input integration tests ---

func TestInjectedClick_FiresClickCallback(t *testing.T) {
	cleanup := setupIdentityTransforms(100, 100)
	defer cleanup()

	m := NewManager()
	child := makeInteractableNode("btn")
	root := makeRoot(child)

	var clicked bool
	m.OnClick(func(ctx node.ClickContext) {
		clicked = true
	})

	m.InjectClick(50, 50)

	// Process press frame
	m.ProcessInjectedInput(root, 0)
	// Process release frame
	m.ProcessInjectedInput(root, 0)

	if !clicked {
		t.Error("injected click should fire OnClick handler")
	}
}

func TestInjectedDrag_FiresDragCallbacks(t *testing.T) {
	cleanup := setupIdentityTransforms(200, 200)
	defer cleanup()

	m := NewManager()
	m.SetDragDeadZone(2.0)
	child := makeInteractableNode("drag-target")
	root := makeRoot(child)

	var startFired, dragFired, endFired bool
	m.OnDragStart(func(ctx node.DragContext) { startFired = true })
	m.OnDrag(func(ctx node.DragContext) { dragFired = true })
	m.OnDragEnd(func(ctx node.DragContext) { endFired = true })

	m.InjectDrag(50, 50, 150, 150, 5)

	// Process all 5 frames
	for len(m.InjectQueue) > 0 {
		m.ProcessInjectedInput(root, 0)
	}

	if !startFired {
		t.Error("injected drag should fire OnDragStart")
	}
	if !dragFired {
		t.Error("injected drag should fire OnDrag")
	}
	if !endFired {
		t.Error("injected drag should fire OnDragEnd")
	}
}

// --- screenToWorld fallback test ---

func TestScreenToWorld_NilFallback(t *testing.T) {
	old := ScreenToWorldFn
	ScreenToWorldFn = nil
	defer func() { ScreenToWorldFn = old }()

	wx, wy := screenToWorld(123, 456)
	if wx != 123 || wy != 456 {
		t.Errorf("nil ScreenToWorldFn should pass through coords, got (%v,%v)", wx, wy)
	}
}

// --- CallbackHandle Remove for all event types ---

func TestCallbackHandle_RemoveAllTypes(t *testing.T) {
	m := NewManager()

	handles := []CallbackHandle{
		m.OnPointerDown(func(node.PointerContext) {}),
		m.OnPointerUp(func(node.PointerContext) {}),
		m.OnPointerMove(func(node.PointerContext) {}),
		m.OnPointerEnter(func(node.PointerContext) {}),
		m.OnPointerLeave(func(node.PointerContext) {}),
		m.OnClick(func(node.ClickContext) {}),
		m.OnBackgroundClick(func(node.ClickContext) {}),
		m.OnDragStart(func(node.DragContext) {}),
		m.OnDrag(func(node.DragContext) {}),
		m.OnDragEnd(func(node.DragContext) {}),
		m.OnPinch(func(node.PinchContext) {}),
	}

	for _, h := range handles {
		h.Remove()
	}

	if len(m.Handlers.pointerDown) != 0 {
		t.Error("pointerDown should be empty after remove")
	}
	if len(m.Handlers.pointerUp) != 0 {
		t.Error("pointerUp should be empty after remove")
	}
	if len(m.Handlers.pointerMove) != 0 {
		t.Error("pointerMove should be empty after remove")
	}
	if len(m.Handlers.pointerEnter) != 0 {
		t.Error("pointerEnter should be empty after remove")
	}
	if len(m.Handlers.pointerLeave) != 0 {
		t.Error("pointerLeave should be empty after remove")
	}
	if len(m.Handlers.click) != 0 {
		t.Error("click should be empty after remove")
	}
	if len(m.Handlers.bgClick) != 0 {
		t.Error("bgClick should be empty after remove")
	}
	if len(m.Handlers.dragStart) != 0 {
		t.Error("dragStart should be empty after remove")
	}
	if len(m.Handlers.drag) != 0 {
		t.Error("drag should be empty after remove")
	}
	if len(m.Handlers.dragEnd) != 0 {
		t.Error("dragEnd should be empty after remove")
	}
	if len(m.Handlers.pinch) != 0 {
		t.Error("pinch should be empty after remove")
	}
}

// --- InjectHover tests ---

func TestInjectHover(t *testing.T) {
	m := NewManager()
	m.InjectHover(100, 200)
	if len(m.InjectQueue) != 1 {
		t.Fatalf("expected 1 event, got %d", len(m.InjectQueue))
	}
	evt := m.InjectQueue[0]
	if evt.Pressed {
		t.Error("hover event should not be pressed")
	}
	if evt.ScreenX != 100 || evt.ScreenY != 200 {
		t.Error("hover coords mismatch")
	}
}

func TestInjectHover_TriggersEnterLeave(t *testing.T) {
	cleanup := setupIdentityTransforms(50, 50)
	defer cleanup()

	m := NewManager()
	child := makeInteractableNode("btn")
	root := makeRoot(child)

	var enterFired bool
	m.OnPointerEnter(func(ctx node.PointerContext) { enterFired = true })

	m.InjectHover(25, 25)
	m.ProcessInjectedInput(root, 0)

	if !enterFired {
		t.Error("InjectHover should trigger OnPointerEnter when hovering over a node")
	}
}

// --- HitTest nil result ---

func TestHitTest_NilOnMiss(t *testing.T) {
	cleanup := setupIdentityTransforms(50, 50)
	defer cleanup()

	m := NewManager()
	child := makeInteractableNode("small")
	root := makeRoot(child)

	hit := m.HitTest(root, 200, 200)
	if hit != nil {
		t.Error("hitTest should return nil when no node is hit")
	}
}

// --- InjectPress / InjectMove / InjectRelease field tests ---

func TestInjectPress_Fields(t *testing.T) {
	m := NewManager()
	m.InjectPress(42, 84)
	if len(m.InjectQueue) != 1 {
		t.Fatal("expected 1 event")
	}
	evt := m.InjectQueue[0]
	if !evt.Pressed {
		t.Error("press should be pressed")
	}
	if evt.Button != types.MouseButtonLeft {
		t.Error("press button should be left")
	}
	if evt.ScreenX != 42 || evt.ScreenY != 84 {
		t.Error("press coords mismatch")
	}
}

func TestInjectRelease_Fields(t *testing.T) {
	m := NewManager()
	m.InjectRelease(42, 84)
	if len(m.InjectQueue) != 1 {
		t.Fatal("expected 1 event")
	}
	evt := m.InjectQueue[0]
	if evt.Pressed {
		t.Error("release should not be pressed")
	}
}
