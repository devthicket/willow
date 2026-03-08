package willow

import (
	"math"
	"testing"
)

// --- HitShape tests ---

func TestHitRectContains(t *testing.T) {
	r := HitRect{X: 10, Y: 20, Width: 100, Height: 50}

	tests := []struct {
		name string
		x, y float64
		want bool
	}{
		{"inside", 50, 40, true},
		{"top-left corner", 10, 20, true},
		{"bottom-right corner", 110, 70, true},
		{"outside left", 5, 40, false},
		{"outside right", 115, 40, false},
		{"outside top", 50, 15, false},
		{"outside bottom", 50, 75, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := r.Contains(tt.x, tt.y); got != tt.want {
				t.Errorf("HitRect.Contains(%v, %v) = %v, want %v", tt.x, tt.y, got, tt.want)
			}
		})
	}
}

func TestHitCircleContains(t *testing.T) {
	c := HitCircle{CenterX: 50, CenterY: 50, Radius: 25}

	tests := []struct {
		name string
		x, y float64
		want bool
	}{
		{"center", 50, 50, true},
		{"on circumference", 75, 50, true},
		{"inside", 60, 50, true},
		{"outside", 80, 50, false},
		{"outside diagonal", 70, 70, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := c.Contains(tt.x, tt.y); got != tt.want {
				t.Errorf("HitCircle.Contains(%v, %v) = %v, want %v", tt.x, tt.y, got, tt.want)
			}
		})
	}
}

func TestHitPolygonContains(t *testing.T) {
	// Square polygon: (0,0), (100,0), (100,100), (0,100)
	p := HitPolygon{Points: []Vec2{
		{X: 0, Y: 0}, {X: 100, Y: 0}, {X: 100, Y: 100}, {X: 0, Y: 100},
	}}

	tests := []struct {
		name string
		x, y float64
		want bool
	}{
		{"inside", 50, 50, true},
		{"on edge", 0, 50, true},
		{"corner", 0, 0, true},
		{"outside", -1, 50, false},
		{"outside far", 200, 200, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := p.Contains(tt.x, tt.y); got != tt.want {
				t.Errorf("HitPolygon.Contains(%v, %v) = %v, want %v", tt.x, tt.y, got, tt.want)
			}
		})
	}

	// Triangle
	tri := HitPolygon{Points: []Vec2{
		{X: 0, Y: 0}, {X: 100, Y: 0}, {X: 50, Y: 100},
	}}
	if !tri.Contains(50, 50) {
		t.Error("triangle should contain its center")
	}
	if tri.Contains(-10, 50) {
		t.Error("triangle should not contain point far left")
	}

	// Degenerate (< 3 points)
	degen := HitPolygon{Points: []Vec2{{X: 0, Y: 0}, {X: 1, Y: 1}}}
	if degen.Contains(0, 0) {
		t.Error("degenerate polygon should not contain anything")
	}
}

func TestHitPolygonContains_ReversedWinding(t *testing.T) {
	// Same square but clockwise winding.
	p := HitPolygon{Points: []Vec2{
		{X: 0, Y: 100}, {X: 100, Y: 100}, {X: 100, Y: 0}, {X: 0, Y: 0},
	}}
	if !p.Contains(50, 50) {
		t.Error("reversed winding polygon should still contain center point")
	}
	if p.Contains(-1, 50) {
		t.Error("reversed winding polygon should not contain outside point")
	}
}

// --- nodeContainsLocal tests ---

func TestNodeContainsLocal_WithHitShape(t *testing.T) {
	n := NewSprite("test", TextureRegion{OriginalW: 64, OriginalH: 64})
	n.HitShape = HitCircle{CenterX: 32, CenterY: 32, Radius: 16}

	if !nodeContainsLocal(n, 32, 32) {
		t.Error("should contain center of circle")
	}
	if nodeContainsLocal(n, 0, 0) {
		t.Error("should not contain corner outside circle")
	}
}

func TestNodeContainsLocal_DefaultAABB(t *testing.T) {
	n := NewSprite("test", TextureRegion{OriginalW: 100, OriginalH: 50})

	if !nodeContainsLocal(n, 50, 25) {
		t.Error("should contain center of sprite AABB")
	}
	if !nodeContainsLocal(n, 0, 0) {
		t.Error("should contain top-left corner")
	}
	if nodeContainsLocal(n, -1, 25) {
		t.Error("should not contain point outside left")
	}
	if nodeContainsLocal(n, 101, 25) {
		t.Error("should not contain point outside right")
	}
}

func TestNodeContainsLocal_ContainerNoHitShape(t *testing.T) {
	n := NewContainer("box")
	if nodeContainsLocal(n, 0, 0) {
		t.Error("container without HitShape should not be hit-testable")
	}
}

func TestNodeContainsLocal_ContainerWithHitShape(t *testing.T) {
	n := NewContainer("box")
	n.HitShape = HitRect{X: 0, Y: 0, Width: 100, Height: 100}
	if !nodeContainsLocal(n, 50, 50) {
		t.Error("container with HitShape should be hit-testable")
	}
}

// --- Hit test traversal tests ---

func TestHitTest_TopmostNode(t *testing.T) {
	s := NewScene()
	// Two overlapping sprites at origin.
	a := NewSprite("a", TextureRegion{OriginalW: 100, OriginalH: 100})
	a.Interactable = true
	b := NewSprite("b", TextureRegion{OriginalW: 100, OriginalH: 100})
	b.Interactable = true

	s.Root().AddChild(a)
	s.Root().AddChild(b)

	// Refresh transforms.
	updateWorldTransform(s.root, identityTransform, 1.0, false, false)

	hit := s.hitTest(50, 50)
	if hit != b {
		t.Errorf("expected topmost node b, got %v", hit)
	}
}

func TestHitTest_SkipsInvisible(t *testing.T) {
	s := NewScene()
	a := NewSprite("a", TextureRegion{OriginalW: 100, OriginalH: 100})
	a.Interactable = true
	b := NewSprite("b", TextureRegion{OriginalW: 100, OriginalH: 100})
	b.Interactable = true
	b.visible = false

	s.Root().AddChild(a)
	s.Root().AddChild(b)
	updateWorldTransform(s.root, identityTransform, 1.0, false, false)

	hit := s.hitTest(50, 50)
	if hit != a {
		t.Errorf("expected node a (b is invisible), got %v", hit)
	}
}

func TestHitTest_SkipsNonInteractable(t *testing.T) {
	s := NewScene()
	a := NewSprite("a", TextureRegion{OriginalW: 100, OriginalH: 100})
	a.Interactable = true
	b := NewSprite("b", TextureRegion{OriginalW: 100, OriginalH: 100})
	// b.Interactable is false by default

	s.Root().AddChild(a)
	s.Root().AddChild(b)
	updateWorldTransform(s.root, identityTransform, 1.0, false, false)

	hit := s.hitTest(50, 50)
	if hit != a {
		t.Errorf("expected node a (b is not interactable), got %v", hit)
	}
}

func TestHitTest_RespectsZIndex(t *testing.T) {
	s := NewScene()
	a := NewSprite("a", TextureRegion{OriginalW: 100, OriginalH: 100})
	a.Interactable = true
	a.SetZIndex(10) // higher ZIndex → rendered later → on top

	b := NewSprite("b", TextureRegion{OriginalW: 100, OriginalH: 100})
	b.Interactable = true
	b.SetZIndex(0)

	s.Root().AddChild(a)
	s.Root().AddChild(b)
	updateWorldTransform(s.root, identityTransform, 1.0, false, false)

	hit := s.hitTest(50, 50)
	if hit != a {
		t.Errorf("expected node a (higher ZIndex), got %v", hit)
	}
}

func TestHitTest_Miss(t *testing.T) {
	s := NewScene()
	a := NewSprite("a", TextureRegion{OriginalW: 100, OriginalH: 100})
	a.Interactable = true

	s.Root().AddChild(a)
	updateWorldTransform(s.root, identityTransform, 1.0, false, false)

	hit := s.hitTest(200, 200)
	if hit != nil {
		t.Errorf("expected nil, got %v", hit)
	}
}

func TestHitTest_TransformedNode(t *testing.T) {
	s := NewScene()
	a := NewSprite("a", TextureRegion{OriginalW: 100, OriginalH: 100})
	a.Interactable = true
	a.x = 200
	a.y = 200

	s.Root().AddChild(a)
	updateWorldTransform(s.root, identityTransform, 1.0, false, false)

	// Point at origin should miss.
	if s.hitTest(50, 50) != nil {
		t.Error("expected miss at origin")
	}
	// Point at (250, 250) should hit.
	if s.hitTest(250, 250) != a {
		t.Error("expected hit at (250, 250)")
	}
}

func TestHitTest_RotatedNode(t *testing.T) {
	s := NewScene()
	a := NewSprite("a", TextureRegion{OriginalW: 100, OriginalH: 100})
	a.Interactable = true
	a.pivotX = 50
	a.pivotY = 50
	a.x = 50
	a.y = 50
	a.rotation = math.Pi / 4 // 45 degrees

	s.Root().AddChild(a)
	updateWorldTransform(s.root, identityTransform, 1.0, false, false)

	// Center should still hit.
	if s.hitTest(50, 50) != a {
		t.Error("center of rotated node should hit")
	}
}

// --- Callback dispatch tests ---

func TestSceneLevelCallback_PointerDown(t *testing.T) {
	s := NewScene()
	sprite := NewSprite("s", TextureRegion{OriginalW: 100, OriginalH: 100})
	sprite.Interactable = true
	s.Root().AddChild(sprite)
	updateWorldTransform(s.root, identityTransform, 1.0, false, false)

	var called bool
	s.OnPointerDown(func(ctx PointerContext) {
		called = true
		if ctx.Node != sprite {
			t.Error("expected hit sprite")
		}
	})

	s.firePointerDown(sprite, 0, 50, 50, MouseButtonLeft, 0)
	if !called {
		t.Error("scene-level pointer down callback not fired")
	}
}

func TestPerNodeCallback_PointerDown(t *testing.T) {
	s := NewScene()
	sprite := NewSprite("s", TextureRegion{OriginalW: 100, OriginalH: 100})
	sprite.Interactable = true
	s.Root().AddChild(sprite)
	updateWorldTransform(s.root, identityTransform, 1.0, false, false)

	var nodeCalled bool
	sprite.OnPointerDown(func(ctx PointerContext) {
		nodeCalled = true
	})

	s.firePointerDown(sprite, 0, 50, 50, MouseButtonLeft, 0)
	if !nodeCalled {
		t.Error("per-node pointer down callback not fired")
	}
}

func TestCallbackOrder_SceneThenNode(t *testing.T) {
	s := NewScene()
	sprite := NewSprite("s", TextureRegion{OriginalW: 100, OriginalH: 100})
	sprite.Interactable = true
	s.Root().AddChild(sprite)
	updateWorldTransform(s.root, identityTransform, 1.0, false, false)

	var order []string
	s.OnPointerDown(func(ctx PointerContext) {
		order = append(order, "scene")
	})
	sprite.OnPointerDown(func(ctx PointerContext) {
		order = append(order, "node")
	})

	s.firePointerDown(sprite, 0, 50, 50, MouseButtonLeft, 0)
	if len(order) != 2 || order[0] != "scene" || order[1] != "node" {
		t.Errorf("expected [scene node], got %v", order)
	}
}

func TestCallbackHandle_Remove(t *testing.T) {
	s := NewScene()

	count := 0
	handle := s.OnPointerDown(func(ctx PointerContext) {
		count++
	})

	s.firePointerDown(nil, 0, 0, 0, MouseButtonLeft, 0)
	if count != 1 {
		t.Fatalf("expected count 1, got %d", count)
	}

	handle.Remove()
	s.firePointerDown(nil, 0, 0, 0, MouseButtonLeft, 0)
	if count != 1 {
		t.Fatalf("expected count still 1 after Remove, got %d", count)
	}
}

// --- Pointer capture tests ---

func TestPointerCapture(t *testing.T) {
	s := NewScene()
	a := NewSprite("a", TextureRegion{OriginalW: 100, OriginalH: 100})
	a.Interactable = true
	a.x = 0
	b := NewSprite("b", TextureRegion{OriginalW: 100, OriginalH: 100})
	b.Interactable = true
	b.x = 200

	s.Root().AddChild(a)
	s.Root().AddChild(b)
	updateWorldTransform(s.root, identityTransform, 1.0, false, false)

	// Capture pointer to b.
	s.CapturePointer(0, b)

	// Hit test at a's location should still return a, but captured should be b.
	target := s.hitTest(50, 50)
	if target != a {
		t.Error("hitTest should return a")
	}
	if s.captured[0] != b {
		t.Error("captured should be b")
	}

	// Process a pointer press  -  the captured node b should receive the event.
	var receivedNode *Node
	s.OnPointerDown(func(ctx PointerContext) {
		receivedNode = ctx.Node
	})
	s.processPointer(0, 50, 50, 50, 50, true, MouseButtonLeft, 0)
	if receivedNode != b {
		t.Errorf("expected captured node b, got %v", receivedNode)
	}

	// Release capture.
	s.ReleasePointer(0)
	if s.captured[0] != nil {
		t.Error("captured should be nil after release")
	}
}

// --- Drag detection tests ---

func TestDragDetection(t *testing.T) {
	s := NewScene()
	sprite := NewSprite("s", TextureRegion{OriginalW: 100, OriginalH: 100})
	sprite.Interactable = true
	s.Root().AddChild(sprite)
	updateWorldTransform(s.root, identityTransform, 1.0, false, false)

	var events []string
	s.OnDragStart(func(ctx DragContext) { events = append(events, "dragstart") })
	s.OnDrag(func(ctx DragContext) { events = append(events, "drag") })
	s.OnDragEnd(func(ctx DragContext) { events = append(events, "dragend") })

	// Press at (50, 50).
	s.processPointer(0, 50, 50, 50, 50, true, MouseButtonLeft, 0)

	// Move within dead zone  -  no drag.
	s.processPointer(0, 52, 52, 52, 52, true, MouseButtonLeft, 0)
	if len(events) != 0 {
		t.Fatalf("expected no events within dead zone, got %v", events)
	}

	// Move beyond dead zone.
	s.processPointer(0, 60, 50, 60, 50, true, MouseButtonLeft, 0)
	if len(events) != 2 || events[0] != "dragstart" || events[1] != "drag" {
		t.Fatalf("expected [dragstart drag], got %v", events)
	}

	// Continue dragging.
	events = events[:0]
	s.processPointer(0, 70, 50, 70, 50, true, MouseButtonLeft, 0)
	if len(events) != 1 || events[0] != "drag" {
		t.Fatalf("expected [drag], got %v", events)
	}

	// Release.
	events = events[:0]
	s.processPointer(0, 70, 50, 70, 50, false, MouseButtonLeft, 0)
	if len(events) != 1 || events[0] != "dragend" {
		t.Fatalf("expected [dragend], got %v", events)
	}
}

func TestClickDetection(t *testing.T) {
	s := NewScene()
	sprite := NewSprite("s", TextureRegion{OriginalW: 100, OriginalH: 100})
	sprite.Interactable = true
	s.Root().AddChild(sprite)
	updateWorldTransform(s.root, identityTransform, 1.0, false, false)

	var clicked bool
	s.OnClick(func(ctx ClickContext) {
		clicked = true
		if ctx.Node != sprite {
			t.Error("expected sprite node")
		}
	})

	// Press and release at same location within dead zone.
	s.processPointer(0, 50, 50, 50, 50, true, MouseButtonLeft, 0)
	s.processPointer(0, 50, 50, 50, 50, false, MouseButtonLeft, 0)
	if !clicked {
		t.Error("expected click event")
	}
}

func TestClickNotFiredOnDrag(t *testing.T) {
	s := NewScene()
	sprite := NewSprite("s", TextureRegion{OriginalW: 100, OriginalH: 100})
	sprite.Interactable = true
	s.Root().AddChild(sprite)
	updateWorldTransform(s.root, identityTransform, 1.0, false, false)

	var clicked bool
	s.OnClick(func(ctx ClickContext) { clicked = true })

	// Press, drag beyond dead zone, release.
	s.processPointer(0, 50, 50, 50, 50, true, MouseButtonLeft, 0)
	s.processPointer(0, 60, 50, 60, 50, true, MouseButtonLeft, 0)
	s.processPointer(0, 60, 50, 60, 50, false, MouseButtonLeft, 0)
	if clicked {
		t.Error("click should not fire after drag")
	}
}

func TestClickNotFiredOnDifferentNode(t *testing.T) {
	s := NewScene()
	a := NewSprite("a", TextureRegion{OriginalW: 50, OriginalH: 100})
	a.Interactable = true
	b := NewSprite("b", TextureRegion{OriginalW: 50, OriginalH: 100})
	b.Interactable = true
	b.x = 50

	s.Root().AddChild(a)
	s.Root().AddChild(b)
	updateWorldTransform(s.root, identityTransform, 1.0, false, false)

	var clicked bool
	s.OnClick(func(ctx ClickContext) { clicked = true })

	// Press on a, release on b (within dead zone distance but different node).
	s.processPointer(0, 25, 50, 25, 50, true, MouseButtonLeft, 0)
	s.processPointer(0, 75, 50, 75, 50, false, MouseButtonLeft, 0)
	if clicked {
		t.Error("click should not fire when press and release are on different nodes")
	}
}

// --- Context coordinate tests ---

func TestContextCoordinates(t *testing.T) {
	s := NewScene()
	sprite := NewSprite("s", TextureRegion{OriginalW: 100, OriginalH: 100})
	sprite.Interactable = true
	sprite.x = 50
	sprite.y = 50
	s.Root().AddChild(sprite)
	updateWorldTransform(s.root, identityTransform, 1.0, false, false)

	s.OnPointerDown(func(ctx PointerContext) {
		if ctx.GlobalX != 75 || ctx.GlobalY != 75 {
			t.Errorf("expected global (75,75), got (%v,%v)", ctx.GlobalX, ctx.GlobalY)
		}
		// Local should be offset by the node's position.
		if ctx.LocalX != 25 || ctx.LocalY != 25 {
			t.Errorf("expected local (25,25), got (%v,%v)", ctx.LocalX, ctx.LocalY)
		}
	})

	s.firePointerDown(sprite, 0, 75, 75, MouseButtonLeft, 0)
}

// --- SetDragDeadZone test ---

func TestSetDragDeadZone(t *testing.T) {
	s := NewScene()
	sprite := NewSprite("s", TextureRegion{OriginalW: 100, OriginalH: 100})
	sprite.Interactable = true
	s.Root().AddChild(sprite)
	updateWorldTransform(s.root, identityTransform, 1.0, false, false)

	s.SetDragDeadZone(20)

	var dragStarted bool
	s.OnDragStart(func(ctx DragContext) { dragStarted = true })

	// Press.
	s.processPointer(0, 50, 50, 50, 50, true, MouseButtonLeft, 0)
	// Move 10 pixels  -  should NOT start drag with 20px dead zone.
	s.processPointer(0, 60, 50, 60, 50, true, MouseButtonLeft, 0)
	if dragStarted {
		t.Error("drag should not start within 20px dead zone")
	}

	// Move 25 pixels from start  -  should start drag.
	s.processPointer(0, 75, 50, 75, 50, true, MouseButtonLeft, 0)
	if !dragStarted {
		t.Error("drag should start beyond 20px dead zone")
	}
}

// --- Independent Scenes test ---

func TestIndependentScenes(t *testing.T) {
	s1 := NewScene()
	s2 := NewScene()

	sprite1 := NewSprite("s1", TextureRegion{OriginalW: 100, OriginalH: 100})
	sprite1.Interactable = true
	s1.Root().AddChild(sprite1)

	sprite2 := NewSprite("s2", TextureRegion{OriginalW: 100, OriginalH: 100})
	sprite2.Interactable = true
	s2.Root().AddChild(sprite2)

	updateWorldTransform(s1.root, identityTransform, 1.0, false, false)
	updateWorldTransform(s2.root, identityTransform, 1.0, false, false)

	var count1, count2 int
	s1.OnPointerDown(func(ctx PointerContext) { count1++ })
	s2.OnPointerDown(func(ctx PointerContext) { count2++ })

	s1.firePointerDown(sprite1, 0, 50, 50, MouseButtonLeft, 0)
	if count1 != 1 || count2 != 0 {
		t.Errorf("expected s1=1 s2=0, got s1=%d s2=%d", count1, count2)
	}

	s2.firePointerDown(sprite2, 0, 50, 50, MouseButtonLeft, 0)
	if count1 != 1 || count2 != 1 {
		t.Errorf("expected s1=1 s2=1, got s1=%d s2=%d", count1, count2)
	}
}

// --- ECS bridge test ---

type mockStore struct {
	events []InteractionEvent
}

func (m *mockStore) EmitEvent(e InteractionEvent) {
	m.events = append(m.events, e)
}

func TestECSBridge(t *testing.T) {
	s := NewScene()
	store := &mockStore{}
	s.SetEntityStore(store)

	sprite := NewSprite("s", TextureRegion{OriginalW: 100, OriginalH: 100})
	sprite.Interactable = true
	sprite.EntityID = 42
	s.Root().AddChild(sprite)
	updateWorldTransform(s.root, identityTransform, 1.0, false, false)

	s.firePointerDown(sprite, 0, 50, 50, MouseButtonLeft, 0)

	if len(store.events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(store.events))
	}
	e := store.events[0]
	if e.Type != EventPointerDown || e.EntityID != 42 {
		t.Errorf("unexpected event: %+v", e)
	}
}

func TestECSBridge_NoEntity(t *testing.T) {
	s := NewScene()
	store := &mockStore{}
	s.SetEntityStore(store)

	sprite := NewSprite("s", TextureRegion{OriginalW: 100, OriginalH: 100})
	sprite.Interactable = true
	// EntityID is 0  -  should not emit.
	s.Root().AddChild(sprite)
	updateWorldTransform(s.root, identityTransform, 1.0, false, false)

	s.firePointerDown(sprite, 0, 50, 50, MouseButtonLeft, 0)
	if len(store.events) != 0 {
		t.Errorf("expected 0 events for node without EntityID, got %d", len(store.events))
	}
}

func TestECSBridge_DragFields(t *testing.T) {
	s := NewScene()
	store := &mockStore{}
	s.SetEntityStore(store)

	sprite := NewSprite("s", TextureRegion{OriginalW: 100, OriginalH: 100})
	sprite.Interactable = true
	sprite.EntityID = 7
	s.Root().AddChild(sprite)
	updateWorldTransform(s.root, identityTransform, 1.0, false, false)

	ctx := DragContext{
		StartX: 10, StartY: 20,
		DeltaX: 5, DeltaY: -3,
	}
	s.fireDrag(sprite, 0, 50, 60, ctx.StartX, ctx.StartY, ctx.DeltaX, ctx.DeltaY, 0, 0, MouseButtonLeft, ModShift)

	if len(store.events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(store.events))
	}
	e := store.events[0]
	if e.Type != EventDrag {
		t.Errorf("Type = %d, want EventDrag", e.Type)
	}
	if e.EntityID != 7 {
		t.Errorf("EntityID = %d, want 7", e.EntityID)
	}
	if e.StartX != 10 || e.StartY != 20 {
		t.Errorf("StartX/Y = (%v,%v), want (10,20)", e.StartX, e.StartY)
	}
	if e.DeltaX != 5 || e.DeltaY != -3 {
		t.Errorf("DeltaX/Y = (%v,%v), want (5,-3)", e.DeltaX, e.DeltaY)
	}
	if e.Modifiers != ModShift {
		t.Errorf("Modifiers = %d, want ModShift", e.Modifiers)
	}
}

func TestECSBridge_PinchFields(t *testing.T) {
	s := NewScene()
	store := &mockStore{}
	s.SetEntityStore(store)

	sprite := NewSprite("s", TextureRegion{OriginalW: 100, OriginalH: 100})
	sprite.Interactable = true
	sprite.EntityID = 99
	s.Root().AddChild(sprite)
	updateWorldTransform(s.root, identityTransform, 1.0, false, false)

	// Set up pinch state so firePinch finds a hit node.
	s.pinch.active = true
	s.pinch.pointer0 = 1
	s.pointers[1].hitNode = sprite

	s.firePinch(PinchContext{
		Scale: 2.0, ScaleDelta: 0.5,
		Rotation: 1.57, RotDelta: 0.1,
		CenterX: 40, CenterY: 60,
	}, 0)

	if len(store.events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(store.events))
	}
	e := store.events[0]
	if e.Type != EventPinch {
		t.Errorf("Type = %d, want EventPinch", e.Type)
	}
	if e.EntityID != 99 {
		t.Errorf("EntityID = %d, want 99", e.EntityID)
	}
	if e.Scale != 2.0 || e.ScaleDelta != 0.5 {
		t.Errorf("Scale = %v, ScaleDelta = %v", e.Scale, e.ScaleDelta)
	}
	if e.Rotation != 1.57 || e.RotDelta != 0.1 {
		t.Errorf("Rotation = %v, RotDelta = %v", e.Rotation, e.RotDelta)
	}
	if e.GlobalX != 40 || e.GlobalY != 60 {
		t.Errorf("GlobalX/Y = (%v,%v), want (40,60)", e.GlobalX, e.GlobalY)
	}
}

func TestECSBridge_PinchWithoutEntityID(t *testing.T) {
	s := NewScene()
	store := &mockStore{}
	s.SetEntityStore(store)

	sprite := NewSprite("s", TextureRegion{OriginalW: 100, OriginalH: 100})
	sprite.Interactable = true
	// EntityID is 0  -  pinch events should still emit.
	s.Root().AddChild(sprite)
	updateWorldTransform(s.root, identityTransform, 1.0, false, false)

	s.pinch.active = true
	s.pinch.pointer0 = 1
	s.pointers[1].hitNode = sprite

	s.firePinch(PinchContext{Scale: 1.5}, 0)

	if len(store.events) != 1 {
		t.Fatalf("expected 1 pinch event even without EntityID, got %d", len(store.events))
	}
	if store.events[0].Type != EventPinch {
		t.Errorf("Type = %d, want EventPinch", store.events[0].Type)
	}
	if store.events[0].EntityID != 0 {
		t.Errorf("EntityID = %d, want 0", store.events[0].EntityID)
	}
}

func TestECSBridge_PinchNoHitNode(t *testing.T) {
	s := NewScene()
	store := &mockStore{}
	s.SetEntityStore(store)

	// No hit node  -  pinch should still emit with EntityID=0.
	s.pinch.active = true
	s.pinch.pointer0 = 1

	s.firePinch(PinchContext{Scale: 3.0, CenterX: 100, CenterY: 200}, 0)

	if len(store.events) != 1 {
		t.Fatalf("expected 1 pinch event without hit node, got %d", len(store.events))
	}
	e := store.events[0]
	if e.Type != EventPinch || e.EntityID != 0 {
		t.Errorf("unexpected event: %+v", e)
	}
}

func TestECSBridge_NoStore(t *testing.T) {
	s := NewScene()
	// No store set  -  should not panic.
	sprite := NewSprite("s", TextureRegion{OriginalW: 100, OriginalH: 100})
	sprite.Interactable = true
	sprite.EntityID = 1
	s.Root().AddChild(sprite)
	updateWorldTransform(s.root, identityTransform, 1.0, false, false)

	s.firePointerDown(sprite, 0, 50, 50, MouseButtonLeft, 0)
	s.fireClick(sprite, 0, 50, 50, MouseButtonLeft, 0)
	s.fireDragStart(sprite, 0, 50, 50, 50, 50, 0, 0, 0, 0, MouseButtonLeft, 0)
	s.fireDrag(sprite, 0, 60, 60, 50, 50, 10, 10, 10, 10, MouseButtonLeft, 0)
	s.fireDragEnd(sprite, 0, 60, 60, 50, 50, 10, 10, 10, 10, MouseButtonLeft, 0)
	s.firePinch(PinchContext{Scale: 1.0}, 0)
	// If we reach here without panic, test passes.
}

// --- Per-node OnPinch test ---

func TestPerNodeOnPinch(t *testing.T) {
	s := NewScene()
	sprite := NewSprite("s", TextureRegion{OriginalW: 100, OriginalH: 100})
	sprite.Interactable = true
	s.Root().AddChild(sprite)
	updateWorldTransform(s.root, identityTransform, 1.0, false, false)

	var called bool
	sprite.OnPinch(func(ctx PinchContext) {
		called = true
	})

	// Simulate pinch state.
	s.pinch.active = true
	s.pinch.pointer0 = 1
	s.pointers[1].hitNode = sprite

	s.firePinch(PinchContext{Scale: 1.5, ScaleDelta: 0.1}, 0)
	if !called {
		t.Error("per-node OnPinch not fired")
	}
}

// --- Hover test ---

func TestHoverMove(t *testing.T) {
	s := NewScene()
	sprite := NewSprite("s", TextureRegion{OriginalW: 100, OriginalH: 100})
	sprite.Interactable = true
	s.Root().AddChild(sprite)
	updateWorldTransform(s.root, identityTransform, 1.0, false, false)

	var moveCalled bool
	s.OnPointerMove(func(ctx PointerContext) {
		moveCalled = true
		if ctx.Node != sprite {
			t.Error("expected sprite node on hover")
		}
	})

	// Hover (not pressed) with position change.
	s.processPointer(0, 50, 50, 50, 50, false, MouseButtonLeft, 0)
	if !moveCalled {
		t.Error("pointer move callback not fired on hover")
	}
}

// --- collectInteractable tests ---

func TestCollectInteractable_SkipsInvisibleSubtree(t *testing.T) {
	s := NewScene()
	container := NewContainer("c")
	container.Interactable = true
	container.visible = false

	child := NewSprite("child", TextureRegion{OriginalW: 100, OriginalH: 100})
	child.Interactable = true
	container.AddChild(child)

	s.Root().AddChild(container)
	updateWorldTransform(s.root, identityTransform, 1.0, false, false)

	buf := s.collectInteractable(s.root, nil)
	for _, n := range buf {
		if n == child {
			t.Error("invisible subtree children should not be collected")
		}
	}
}

func TestCollectInteractable_SkipsNonInteractableSubtree(t *testing.T) {
	s := NewScene()
	container := NewContainer("c")
	container.Interactable = false

	child := NewSprite("child", TextureRegion{OriginalW: 100, OriginalH: 100})
	child.Interactable = true
	container.AddChild(child)

	s.Root().AddChild(container)
	updateWorldTransform(s.root, identityTransform, 1.0, false, false)

	buf := s.collectInteractable(s.root, nil)
	for _, n := range buf {
		if n == child {
			t.Error("non-interactable subtree children should not be collected")
		}
	}
}

// --- Auto-release capture on pointer up ---

func TestAutoReleaseCapture(t *testing.T) {
	s := NewScene()
	sprite := NewSprite("s", TextureRegion{OriginalW: 100, OriginalH: 100})
	sprite.Interactable = true
	s.Root().AddChild(sprite)
	updateWorldTransform(s.root, identityTransform, 1.0, false, false)

	s.CapturePointer(0, sprite)
	if s.captured[0] != sprite {
		t.Fatal("capture not set")
	}

	// Press and release.
	s.processPointer(0, 50, 50, 50, 50, true, MouseButtonLeft, 0)
	s.processPointer(0, 50, 50, 50, 50, false, MouseButtonLeft, 0)

	if s.captured[0] != nil {
		t.Error("capture should be auto-released on pointer up")
	}
}

// --- Multiple handlers test ---

func TestMultipleSceneHandlers(t *testing.T) {
	s := NewScene()
	var count int
	s.OnPointerDown(func(ctx PointerContext) { count++ })
	s.OnPointerDown(func(ctx PointerContext) { count++ })
	s.OnPointerDown(func(ctx PointerContext) { count++ })

	s.firePointerDown(nil, 0, 0, 0, MouseButtonLeft, 0)
	if count != 3 {
		t.Errorf("expected 3, got %d", count)
	}
}

// --- Benchmarks ---

func BenchmarkHitTest_1000Nodes(b *testing.B) {
	s := NewScene()
	for i := 0; i < 1000; i++ {
		n := NewSprite("n", TextureRegion{OriginalW: 10, OriginalH: 10})
		n.Interactable = true
		n.x = float64(i%100) * 12
		n.y = float64(i/100) * 12
		s.Root().AddChild(n)
	}
	updateWorldTransform(s.root, identityTransform, 1.0, false, false)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		s.hitTest(500, 50)
	}
}

func BenchmarkDispatch_10Handlers(b *testing.B) {
	s := NewScene()
	for i := 0; i < 10; i++ {
		s.OnPointerDown(func(ctx PointerContext) {})
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		s.firePointerDown(nil, 0, 0, 0, MouseButtonLeft, 0)
	}
}
