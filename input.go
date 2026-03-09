package willow

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/phanxgames/willow/internal/camera"
	"github.com/phanxgames/willow/internal/render"
)

// --- Constants ---

const (
	maxPointers         = 10  // pointer 0 = mouse, 1-9 = touch
	defaultDragDeadZone = 4.0 // pixels
)

// --- Built-in HitShape types ---

// HitRect is an axis-aligned rectangular hit area in local coordinates.
type HitRect struct {
	X, Y, Width, Height float64
}

// Contains reports whether (x, y) lies inside the rectangle.
func (r HitRect) Contains(x, y float64) bool {
	return x >= r.X && x <= r.X+r.Width &&
		y >= r.Y && y <= r.Y+r.Height
}

// HitCircle is a circular hit area in local coordinates.
type HitCircle struct {
	CenterX, CenterY, Radius float64
}

// Contains reports whether (x, y) lies inside or on the circle.
func (c HitCircle) Contains(x, y float64) bool {
	dx := x - c.CenterX
	dy := y - c.CenterY
	return dx*dx+dy*dy <= c.Radius*c.Radius
}

// HitPolygon is a convex polygon hit area in local coordinates.
// Points must define a convex polygon in either winding order.
// Concave polygons will produce incorrect results.
type HitPolygon struct {
	Points []Vec2
}

// Contains reports whether (x, y) lies inside a convex polygon using cross-product sign test.
func (p HitPolygon) Contains(x, y float64) bool {
	n := len(p.Points)
	if n < 3 {
		return false
	}

	// Check that the point is on the same side of every edge.
	var positive, negative bool
	for i := 0; i < n; i++ {
		x1 := p.Points[i].X
		y1 := p.Points[i].Y
		j := (i + 1) % n
		x2 := p.Points[j].X
		y2 := p.Points[j].Y

		cross := (x2-x1)*(y-y1) - (y2-y1)*(x-x1)
		if cross > 0 {
			positive = true
		} else if cross < 0 {
			negative = true
		}
		if positive && negative {
			return false
		}
	}
	return true
}

// --- Per-pointer state ---

type pointerState struct {
	down        bool
	startX      float64
	startY      float64
	lastX       float64
	lastY       float64
	screenX     float64
	screenY     float64
	lastScreenX float64
	lastScreenY float64
	hitNode     *Node
	hoverNode   *Node // last node the pointer was hovering over (for enter/leave)
	dragging    bool
	button      MouseButton // button captured at press time
}

// --- Pinch state ---

type pinchState struct {
	active       bool
	pointer0     int
	pointer1     int
	initialDist  float64
	initialAngle float64
	prevDist     float64
	prevAngle    float64
}

// --- Handler registry ---

type pointerHandler struct {
	id uint32
	fn func(PointerContext)
}

type clickHandler struct {
	id uint32
	fn func(ClickContext)
}

type dragHandler struct {
	id uint32
	fn func(DragContext)
}

type pinchHandler struct {
	id uint32
	fn func(PinchContext)
}

type handlerRegistry struct {
	pointerDown  []pointerHandler
	pointerUp    []pointerHandler
	pointerMove  []pointerHandler
	pointerEnter []pointerHandler
	pointerLeave []pointerHandler
	click        []clickHandler
	bgClick      []clickHandler
	dragStart    []dragHandler
	drag         []dragHandler
	dragEnd      []dragHandler
	pinch        []pinchHandler
	nextID       uint32
}

// CallbackHandle allows removing a registered scene-level callback.
type CallbackHandle struct {
	id    uint32
	reg   *handlerRegistry
	event EventType
}

// Remove unregisters this callback so it no longer fires.
// The entry is removed from the slice to avoid nil iteration waste.
func (h CallbackHandle) Remove() {
	if h.reg == nil {
		return
	}
	switch h.event {
	case EventPointerDown:
		h.reg.pointerDown = removePointerHandler(h.reg.pointerDown, h.id)
	case EventPointerUp:
		h.reg.pointerUp = removePointerHandler(h.reg.pointerUp, h.id)
	case EventPointerMove:
		h.reg.pointerMove = removePointerHandler(h.reg.pointerMove, h.id)
	case EventPointerEnter:
		h.reg.pointerEnter = removePointerHandler(h.reg.pointerEnter, h.id)
	case EventPointerLeave:
		h.reg.pointerLeave = removePointerHandler(h.reg.pointerLeave, h.id)
	case EventClick:
		h.reg.click = removeClickHandler(h.reg.click, h.id)
	case eventBgClick:
		h.reg.bgClick = removeClickHandler(h.reg.bgClick, h.id)
	case EventDragStart:
		h.reg.dragStart = removeDragHandler(h.reg.dragStart, h.id)
	case EventDrag:
		h.reg.drag = removeDragHandler(h.reg.drag, h.id)
	case EventDragEnd:
		h.reg.dragEnd = removeDragHandler(h.reg.dragEnd, h.id)
	case EventPinch:
		h.reg.pinch = removePinchHandler(h.reg.pinch, h.id)
	}
}

func removePointerHandler(s []pointerHandler, id uint32) []pointerHandler {
	for i := range s {
		if s[i].id == id {
			copy(s[i:], s[i+1:])
			s[len(s)-1] = pointerHandler{}
			return s[:len(s)-1]
		}
	}
	return s
}

func removeClickHandler(s []clickHandler, id uint32) []clickHandler {
	for i := range s {
		if s[i].id == id {
			copy(s[i:], s[i+1:])
			s[len(s)-1] = clickHandler{}
			return s[:len(s)-1]
		}
	}
	return s
}

func removeDragHandler(s []dragHandler, id uint32) []dragHandler {
	for i := range s {
		if s[i].id == id {
			copy(s[i:], s[i+1:])
			s[len(s)-1] = dragHandler{}
			return s[:len(s)-1]
		}
	}
	return s
}

func removePinchHandler(s []pinchHandler, id uint32) []pinchHandler {
	for i := range s {
		if s[i].id == id {
			copy(s[i:], s[i+1:])
			s[len(s)-1] = pinchHandler{}
			return s[:len(s)-1]
		}
	}
	return s
}

// --- Scene-level event registration ---

// OnPointerDown registers a scene-level callback for pointer down events.
// Multiple registrations are additive; scene-level handlers fire before
// per-node callbacks. Use CallbackHandle.Remove to unregister.
func (s *Scene) OnPointerDown(fn func(PointerContext)) CallbackHandle {
	s.handlers.nextID++
	id := s.handlers.nextID
	s.handlers.pointerDown = append(s.handlers.pointerDown, pointerHandler{id: id, fn: fn})
	return CallbackHandle{id: id, reg: &s.handlers, event: EventPointerDown}
}

// OnPointerUp registers a scene-level callback for pointer up events.
// Multiple registrations are additive. Use CallbackHandle.Remove to unregister.
func (s *Scene) OnPointerUp(fn func(PointerContext)) CallbackHandle {
	s.handlers.nextID++
	id := s.handlers.nextID
	s.handlers.pointerUp = append(s.handlers.pointerUp, pointerHandler{id: id, fn: fn})
	return CallbackHandle{id: id, reg: &s.handlers, event: EventPointerUp}
}

// OnPointerMove registers a scene-level callback for pointer move events.
// Multiple registrations are additive. Use CallbackHandle.Remove to unregister.
func (s *Scene) OnPointerMove(fn func(PointerContext)) CallbackHandle {
	s.handlers.nextID++
	id := s.handlers.nextID
	s.handlers.pointerMove = append(s.handlers.pointerMove, pointerHandler{id: id, fn: fn})
	return CallbackHandle{id: id, reg: &s.handlers, event: EventPointerMove}
}

// OnPointerEnter registers a scene-level callback for pointer enter events.
// Fired when the pointer moves over a new node (or from nil to a node).
func (s *Scene) OnPointerEnter(fn func(PointerContext)) CallbackHandle {
	s.handlers.nextID++
	id := s.handlers.nextID
	s.handlers.pointerEnter = append(s.handlers.pointerEnter, pointerHandler{id: id, fn: fn})
	return CallbackHandle{id: id, reg: &s.handlers, event: EventPointerEnter}
}

// OnPointerLeave registers a scene-level callback for pointer leave events.
// Fired when the pointer leaves a node (moves to a different node or to empty space).
func (s *Scene) OnPointerLeave(fn func(PointerContext)) CallbackHandle {
	s.handlers.nextID++
	id := s.handlers.nextID
	s.handlers.pointerLeave = append(s.handlers.pointerLeave, pointerHandler{id: id, fn: fn})
	return CallbackHandle{id: id, reg: &s.handlers, event: EventPointerLeave}
}

// OnClick registers a scene-level callback for click events.
// Multiple registrations are additive. Use CallbackHandle.Remove to unregister.
func (s *Scene) OnClick(fn func(ClickContext)) CallbackHandle {
	s.handlers.nextID++
	id := s.handlers.nextID
	s.handlers.click = append(s.handlers.click, clickHandler{id: id, fn: fn})
	return CallbackHandle{id: id, reg: &s.handlers, event: EventClick}
}

// OnDragStart registers a scene-level callback for drag start events.
// Multiple registrations are additive. Use CallbackHandle.Remove to unregister.
func (s *Scene) OnDragStart(fn func(DragContext)) CallbackHandle {
	s.handlers.nextID++
	id := s.handlers.nextID
	s.handlers.dragStart = append(s.handlers.dragStart, dragHandler{id: id, fn: fn})
	return CallbackHandle{id: id, reg: &s.handlers, event: EventDragStart}
}

// OnDrag registers a scene-level callback for drag events.
// Multiple registrations are additive. Use CallbackHandle.Remove to unregister.
func (s *Scene) OnDrag(fn func(DragContext)) CallbackHandle {
	s.handlers.nextID++
	id := s.handlers.nextID
	s.handlers.drag = append(s.handlers.drag, dragHandler{id: id, fn: fn})
	return CallbackHandle{id: id, reg: &s.handlers, event: EventDrag}
}

// OnDragEnd registers a scene-level callback for drag end events.
// Multiple registrations are additive. Use CallbackHandle.Remove to unregister.
func (s *Scene) OnDragEnd(fn func(DragContext)) CallbackHandle {
	s.handlers.nextID++
	id := s.handlers.nextID
	s.handlers.dragEnd = append(s.handlers.dragEnd, dragHandler{id: id, fn: fn})
	return CallbackHandle{id: id, reg: &s.handlers, event: EventDragEnd}
}

// OnPinch registers a scene-level callback for pinch events.
// Multiple registrations are additive. Use CallbackHandle.Remove to unregister.
func (s *Scene) OnPinch(fn func(PinchContext)) CallbackHandle {
	s.handlers.nextID++
	id := s.handlers.nextID
	s.handlers.pinch = append(s.handlers.pinch, pinchHandler{id: id, fn: fn})
	return CallbackHandle{id: id, reg: &s.handlers, event: EventPinch}
}

// OnBackgroundClick registers a scene-level callback that fires when a click
// lands on empty space (no interactive node was hit). Use CallbackHandle.Remove
// to unregister.
func (s *Scene) OnBackgroundClick(fn func(ClickContext)) CallbackHandle {
	s.handlers.nextID++
	id := s.handlers.nextID
	s.handlers.bgClick = append(s.handlers.bgClick, clickHandler{id: id, fn: fn})
	return CallbackHandle{id: id, reg: &s.handlers, event: eventBgClick}
}

// CapturePointer routes all events for pointerID to the given node.
func (s *Scene) CapturePointer(pointerID int, node *Node) {
	if pointerID >= 0 && pointerID < maxPointers {
		s.captured[pointerID] = node
	}
}

// ReleasePointer stops routing events for pointerID to a captured node.
func (s *Scene) ReleasePointer(pointerID int) {
	if pointerID >= 0 && pointerID < maxPointers {
		s.captured[pointerID] = nil
	}
}

// SetDragDeadZone sets the minimum movement in pixels before a drag starts.
func (s *Scene) SetDragDeadZone(pixels float64) {
	s.dragDeadZone = pixels
}

// --- Hit testing ---

// nodeContainsLocal tests whether (lx, ly) falls inside a node's hit region.
// Uses HitShape if set; otherwise derives AABB from node dimensions.
// Containers with no HitShape are not hit-testable.
func nodeContainsLocal(n *Node, lx, ly float64) bool {
	if n.HitShape != nil {
		return n.HitShape.Contains(lx, ly)
	}
	w, h := camera.NodeDimensions(n)
	if w == 0 && h == 0 {
		return false
	}
	return lx >= 0 && lx <= w && ly >= 0 && ly <= h
}

// collectInteractable walks the tree in painter order (DFS, ZIndex-sorted),
// appending interactable leaf nodes to buf. Skips Visible=false or
// Interactable=false subtrees.
func (s *Scene) collectInteractable(n *Node, buf []*Node) []*Node {
	if !n.Visible_ || !n.Interactable {
		return buf
	}

	// Add this node if it's potentially hit-testable (has shape or dimensions).
	if n.HitShape != nil || n.Type != NodeTypeContainer {
		buf = append(buf, n)
	}

	if len(n.Children_) == 0 {
		return buf
	}

	children := n.Children_
	if !n.ChildrenSorted {
		render.RebuildSortedChildren(n)
	}
	if n.SortedChildren != nil {
		children = n.SortedChildren
	}
	for _, child := range children {
		buf = s.collectInteractable(child, buf)
	}
	return buf
}

// hitTest finds the topmost interactable node at (worldX, worldY).
// Returns nil if nothing is hit.
func (s *Scene) hitTest(worldX, worldY float64) *Node {
	s.hitBuf = s.collectInteractable(s.root, s.hitBuf[:0])

	// Iterate backward (reverse painter order): topmost visual node first.
	for i := len(s.hitBuf) - 1; i >= 0; i-- {
		n := s.hitBuf[i]
		lx, ly := n.WorldToLocal(worldX, worldY)
		if nodeContainsLocal(n, lx, ly) {
			return n
		}
	}
	return nil
}

// --- Input processing ---

// readModifiers reads the current keyboard modifier state.
func readModifiers() KeyModifiers {
	var mods KeyModifiers
	if ebiten.IsKeyPressed(ebiten.KeyShift) || ebiten.IsKeyPressed(ebiten.KeyShiftLeft) || ebiten.IsKeyPressed(ebiten.KeyShiftRight) {
		mods |= ModShift
	}
	if ebiten.IsKeyPressed(ebiten.KeyControl) || ebiten.IsKeyPressed(ebiten.KeyControlLeft) || ebiten.IsKeyPressed(ebiten.KeyControlRight) {
		mods |= ModCtrl
	}
	if ebiten.IsKeyPressed(ebiten.KeyAlt) || ebiten.IsKeyPressed(ebiten.KeyAltLeft) || ebiten.IsKeyPressed(ebiten.KeyAltRight) {
		mods |= ModAlt
	}
	if ebiten.IsKeyPressed(ebiten.KeyMeta) || ebiten.IsKeyPressed(ebiten.KeyMetaLeft) || ebiten.IsKeyPressed(ebiten.KeyMetaRight) {
		mods |= ModMeta
	}
	return mods
}

// processInput is called from Scene.Update() to handle all mouse and touch input.
// World transforms are already refreshed at the start of Scene.Update().
func (s *Scene) processInput() {
	mods := readModifiers()

	// Primary camera for screen-to-world conversion.
	var cam *Camera
	if len(s.cameras) > 0 {
		cam = s.cameras[0]
		cam.ComputeViewMatrix()
	}

	if !s.processInjectedInput(cam, mods) {
		s.processMousePointer(cam, mods)
	}
	s.processTouchPointers(cam, mods)
	s.detectPinch(mods)
}

// screenToWorld converts screen coordinates to world coordinates using the primary camera.
func screenToWorld(cam *Camera, sx, sy float64) (float64, float64) {
	if cam != nil {
		return cam.ScreenToWorld(sx, sy)
	}
	return sx, sy
}

// processMousePointer handles mouse input (pointer 0).
func (s *Scene) processMousePointer(cam *Camera, mods KeyModifiers) {
	mx, my := ebiten.CursorPosition()
	sx, sy := float64(mx), float64(my)
	wx, wy := screenToWorld(cam, sx, sy)

	// Detect which button is pressed. If pointer is already down, use the
	// stored button to avoid changing mid-interaction.
	var pressed bool
	var button MouseButton
	left := ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)
	right := ebiten.IsMouseButtonPressed(ebiten.MouseButtonRight)
	middle := ebiten.IsMouseButtonPressed(ebiten.MouseButtonMiddle)

	if left || right || middle {
		pressed = true
		if left {
			button = MouseButtonLeft
		} else if right {
			button = MouseButtonRight
		} else {
			button = MouseButtonMiddle
		}
	}

	s.processPointer(0, wx, wy, sx, sy, pressed, button, mods)
}

// processTouchPointers handles touch input (pointers 1-9).
func (s *Scene) processTouchPointers(cam *Camera, mods KeyModifiers) {
	touchIDs := ebiten.AppendTouchIDs(s.prevTouchIDs[:0])
	s.prevTouchIDs = touchIDs

	// Mark all touch slots as unused this frame.
	var activeSlots [maxPointers]bool
	for _, tid := range touchIDs {
		slot := s.touchSlot(tid)
		if slot < 0 {
			continue
		}
		activeSlots[slot] = true

		tx, ty := ebiten.TouchPosition(tid)
		stx, sty := float64(tx), float64(ty)
		wx, wy := screenToWorld(cam, stx, sty)
		s.processPointer(slot, wx, wy, stx, sty, true, MouseButtonLeft, mods)
	}

	// Release any touch slots that are no longer active.
	for i := 1; i < maxPointers; i++ {
		if s.touchUsed[i] && !activeSlots[i] {
			ps := &s.pointers[i]
			if ps.down {
				s.processPointer(i, ps.lastX, ps.lastY, ps.lastScreenX, ps.lastScreenY, false, MouseButtonLeft, mods)
			}
			s.touchUsed[i] = false
			s.touchMap[i] = 0
		}
	}
}

// touchSlot maps an ebiten.TouchID to a pointer slot (1-9).
// Returns the existing slot or allocates a new one. Returns -1 if full.
func (s *Scene) touchSlot(tid ebiten.TouchID) int {
	// Check existing mapping.
	for i := 1; i < maxPointers; i++ {
		if s.touchUsed[i] && s.touchMap[i] == tid {
			return i
		}
	}
	// Allocate new slot.
	for i := 1; i < maxPointers; i++ {
		if !s.touchUsed[i] {
			s.touchUsed[i] = true
			s.touchMap[i] = tid
			return i
		}
	}
	return -1
}

// processPointer runs the pointer state machine for a single pointer.
// sx, sy are the raw screen-space coordinates (before camera transform).
func (s *Scene) processPointer(pointerID int, wx, wy, sx, sy float64, pressed bool, button MouseButton, mods KeyModifiers) {
	ps := &s.pointers[pointerID]

	// Determine target node: captured node or hit test.
	var target *Node
	if s.captured[pointerID] != nil {
		target = s.captured[pointerID]
	} else {
		target = s.hitTest(wx, wy)
	}

	// Clear stale hover reference if the node was disposed.
	if ps.hoverNode != nil && ps.hoverNode.Disposed {
		ps.hoverNode = nil
	}

	// Fire hover enter/leave when the hovered node changes.
	if target != ps.hoverNode {
		if ps.hoverNode != nil {
			s.firePointerLeave(ps.hoverNode, pointerID, wx, wy, button, mods)
		}
		if target != nil {
			s.firePointerEnter(target, pointerID, wx, wy, button, mods)
		}
		ps.hoverNode = target
	}

	if pressed && !ps.down {
		// Just pressed  -  capture button for the duration of this interaction.
		ps.down = true
		ps.button = button
		ps.startX = wx
		ps.startY = wy
		ps.lastX = wx
		ps.lastY = wy
		ps.screenX = sx
		ps.screenY = sy
		ps.lastScreenX = sx
		ps.lastScreenY = sy
		ps.hitNode = target
		ps.dragging = false

		s.firePointerDown(target, pointerID, wx, wy, ps.button, mods)
	} else if !pressed && ps.down {
		// Just released  -  use button from press start.
		sdx := sx - ps.lastScreenX
		sdy := sy - ps.lastScreenY
		if ps.dragging {
			s.fireDragEnd(ps.hitNode, pointerID, wx, wy, ps.startX, ps.startY,
				wx-ps.lastX, wy-ps.lastY, sdx, sdy, ps.button, mods)
		} else if ps.hitNode != nil && ps.hitNode == target {
			s.fireClick(target, pointerID, wx, wy, ps.button, mods)
		} else if ps.hitNode == nil && target == nil {
			s.fireBackgroundClick(pointerID, wx, wy, ps.button, mods)
		}

		s.firePointerUp(target, pointerID, wx, wy, ps.button, mods)

		// Auto-release capture.
		s.captured[pointerID] = nil
		ps.down = false
		ps.hitNode = nil
		ps.dragging = false
	} else if pressed && ps.down {
		// Held down, possibly moved  -  use button from press start.
		if wx != ps.lastX || wy != ps.lastY || sx != ps.lastScreenX || sy != ps.lastScreenY {
			sdx := sx - ps.lastScreenX
			sdy := sy - ps.lastScreenY
			if !ps.dragging {
				dx := wx - ps.startX
				dy := wy - ps.startY
				if math.Sqrt(dx*dx+dy*dy) > s.dragDeadZone {
					ps.dragging = true
					s.fireDragStart(ps.hitNode, pointerID, wx, wy, ps.startX, ps.startY,
						wx-ps.startX, wy-ps.startY, sx-ps.screenX, sy-ps.screenY, ps.button, mods)
				}
			}
			if ps.dragging {
				s.fireDrag(ps.hitNode, pointerID, wx, wy, ps.startX, ps.startY,
					wx-ps.lastX, wy-ps.lastY, sdx, sdy, ps.button, mods)
			}
		}
		ps.lastX = wx
		ps.lastY = wy
		ps.lastScreenX = sx
		ps.lastScreenY = sy
	} else if !pressed && !ps.down {
		// Hover move.
		if wx != ps.lastX || wy != ps.lastY {
			s.firePointerMove(target, pointerID, wx, wy, button, mods)
			ps.lastX = wx
			ps.lastY = wy
		}
	}
}

// --- Pinch detection ---

func (s *Scene) detectPinch(mods KeyModifiers) {
	// Count active touch pointers.
	var active [maxPointers]bool
	var count int
	for i := 1; i < maxPointers; i++ {
		if s.pointers[i].down {
			active[i] = true
			count++
		}
	}

	if count == 2 {
		// Find the two active pointers.
		var p0, p1 int
		found := 0
		for i := 1; i < maxPointers; i++ {
			if active[i] {
				if found == 0 {
					p0 = i
				} else {
					p1 = i
				}
				found++
				if found == 2 {
					break
				}
			}
		}

		ps0 := &s.pointers[p0]
		ps1 := &s.pointers[p1]

		cx := (ps0.lastX + ps1.lastX) / 2
		cy := (ps0.lastY + ps1.lastY) / 2
		dx := ps1.lastX - ps0.lastX
		dy := ps1.lastY - ps0.lastY
		dist := math.Sqrt(dx*dx + dy*dy)
		angle := math.Atan2(dy, dx)

		if !s.pinch.active {
			// Start pinch.
			s.pinch.active = true
			s.pinch.pointer0 = p0
			s.pinch.pointer1 = p1
			s.pinch.initialDist = dist
			s.pinch.initialAngle = angle
			s.pinch.prevDist = dist
			s.pinch.prevAngle = angle

			// Suppress drag for the two pinch pointers.
			ps0.dragging = false
			ps1.dragging = false
		} else {
			scale := 1.0
			if s.pinch.initialDist > 0 {
				scale = dist / s.pinch.initialDist
			}
			scaleDelta := 0.0
			if s.pinch.prevDist > 0 {
				scaleDelta = dist/s.pinch.prevDist - 1.0
			}
			rotation := angle - s.pinch.initialAngle
			rotDelta := angle - s.pinch.prevAngle

			ctx := PinchContext{
				CenterX:    cx,
				CenterY:    cy,
				Scale:      scale,
				ScaleDelta: scaleDelta,
				Rotation:   rotation,
				RotDelta:   rotDelta,
			}
			s.firePinch(ctx, mods)

			s.pinch.prevDist = dist
			s.pinch.prevAngle = angle
		}

		// Suppress drag events for pinch pointers.
		ps0.dragging = false
		ps1.dragging = false
	} else if s.pinch.active {
		// Pinch ended.
		s.pinch.active = false
	}
}

// --- Event dispatch ---

func (s *Scene) firePointerDown(node *Node, pointerID int, wx, wy float64, button MouseButton, mods KeyModifiers) {
	var lx, ly float64
	var entityID uint32
	var userData any
	if node != nil {
		lx, ly = node.WorldToLocal(wx, wy)
		entityID = node.EntityID
		userData = node.UserData
	}
	ctx := PointerContext{
		Node: node, EntityID: entityID, UserData: userData,
		GlobalX: wx, GlobalY: wy, LocalX: lx, LocalY: ly,
		Button: button, PointerID: pointerID, Modifiers: mods,
	}
	// Scene-level handlers first.
	for _, h := range s.handlers.pointerDown {
		h.fn(ctx)
	}
	// Per-node callback.
	if node != nil && node.Callbacks != nil && node.Callbacks.OnPointerDown != nil {
		node.Callbacks.OnPointerDown(ctx)
	}
	// ECS bridge.
	s.emitInteractionEvent(EventPointerDown, node, wx, wy, lx, ly, button, mods, DragContext{}, PinchContext{})
}

func (s *Scene) firePointerUp(node *Node, pointerID int, wx, wy float64, button MouseButton, mods KeyModifiers) {
	var lx, ly float64
	var entityID uint32
	var userData any
	if node != nil {
		lx, ly = node.WorldToLocal(wx, wy)
		entityID = node.EntityID
		userData = node.UserData
	}
	ctx := PointerContext{
		Node: node, EntityID: entityID, UserData: userData,
		GlobalX: wx, GlobalY: wy, LocalX: lx, LocalY: ly,
		Button: button, PointerID: pointerID, Modifiers: mods,
	}
	for _, h := range s.handlers.pointerUp {
		h.fn(ctx)
	}
	if node != nil && node.Callbacks != nil && node.Callbacks.OnPointerUp != nil {
		node.Callbacks.OnPointerUp(ctx)
	}
	s.emitInteractionEvent(EventPointerUp, node, wx, wy, lx, ly, button, mods, DragContext{}, PinchContext{})
}

func (s *Scene) firePointerMove(node *Node, pointerID int, wx, wy float64, button MouseButton, mods KeyModifiers) {
	var lx, ly float64
	var entityID uint32
	var userData any
	if node != nil {
		lx, ly = node.WorldToLocal(wx, wy)
		entityID = node.EntityID
		userData = node.UserData
	}
	ctx := PointerContext{
		Node: node, EntityID: entityID, UserData: userData,
		GlobalX: wx, GlobalY: wy, LocalX: lx, LocalY: ly,
		Button: button, PointerID: pointerID, Modifiers: mods,
	}
	for _, h := range s.handlers.pointerMove {
		h.fn(ctx)
	}
	if node != nil && node.Callbacks != nil && node.Callbacks.OnPointerMove != nil {
		node.Callbacks.OnPointerMove(ctx)
	}
	s.emitInteractionEvent(EventPointerMove, node, wx, wy, lx, ly, button, mods, DragContext{}, PinchContext{})
}

func (s *Scene) firePointerEnter(node *Node, pointerID int, wx, wy float64, button MouseButton, mods KeyModifiers) {
	var lx, ly float64
	var entityID uint32
	var userData any
	if node != nil {
		lx, ly = node.WorldToLocal(wx, wy)
		entityID = node.EntityID
		userData = node.UserData
	}
	ctx := PointerContext{
		Node: node, EntityID: entityID, UserData: userData,
		GlobalX: wx, GlobalY: wy, LocalX: lx, LocalY: ly,
		Button: button, PointerID: pointerID, Modifiers: mods,
	}
	for _, h := range s.handlers.pointerEnter {
		h.fn(ctx)
	}
	if node != nil && node.Callbacks != nil && node.Callbacks.OnPointerEnter != nil {
		node.Callbacks.OnPointerEnter(ctx)
	}
	s.emitInteractionEvent(EventPointerEnter, node, wx, wy, lx, ly, button, mods, DragContext{}, PinchContext{})
}

func (s *Scene) firePointerLeave(node *Node, pointerID int, wx, wy float64, button MouseButton, mods KeyModifiers) {
	var lx, ly float64
	var entityID uint32
	var userData any
	if node != nil {
		lx, ly = node.WorldToLocal(wx, wy)
		entityID = node.EntityID
		userData = node.UserData
	}
	ctx := PointerContext{
		Node: node, EntityID: entityID, UserData: userData,
		GlobalX: wx, GlobalY: wy, LocalX: lx, LocalY: ly,
		Button: button, PointerID: pointerID, Modifiers: mods,
	}
	for _, h := range s.handlers.pointerLeave {
		h.fn(ctx)
	}
	if node != nil && node.Callbacks != nil && node.Callbacks.OnPointerLeave != nil {
		node.Callbacks.OnPointerLeave(ctx)
	}
	s.emitInteractionEvent(EventPointerLeave, node, wx, wy, lx, ly, button, mods, DragContext{}, PinchContext{})
}

func (s *Scene) fireClick(node *Node, pointerID int, wx, wy float64, button MouseButton, mods KeyModifiers) {
	var lx, ly float64
	var entityID uint32
	var userData any
	if node != nil {
		lx, ly = node.WorldToLocal(wx, wy)
		entityID = node.EntityID
		userData = node.UserData
	}
	ctx := ClickContext{
		Node: node, EntityID: entityID, UserData: userData,
		GlobalX: wx, GlobalY: wy, LocalX: lx, LocalY: ly,
		Button: button, PointerID: pointerID, Modifiers: mods,
	}
	for _, h := range s.handlers.click {
		h.fn(ctx)
	}
	if node != nil && node.Callbacks != nil && node.Callbacks.OnClick != nil {
		node.Callbacks.OnClick(ctx)
	}
	s.emitInteractionEvent(EventClick, node, wx, wy, lx, ly, button, mods, DragContext{}, PinchContext{})
}

func (s *Scene) fireDragStart(node *Node, pointerID int, wx, wy, startX, startY, deltaX, deltaY, screenDX, screenDY float64, button MouseButton, mods KeyModifiers) {
	var lx, ly float64
	var entityID uint32
	var userData any
	if node != nil {
		lx, ly = node.WorldToLocal(wx, wy)
		entityID = node.EntityID
		userData = node.UserData
	}
	ctx := DragContext{
		Node: node, EntityID: entityID, UserData: userData,
		GlobalX: wx, GlobalY: wy, LocalX: lx, LocalY: ly,
		StartX: startX, StartY: startY, DeltaX: deltaX, DeltaY: deltaY,
		ScreenDeltaX: screenDX, ScreenDeltaY: screenDY,
		Button: button, PointerID: pointerID, Modifiers: mods,
	}
	for _, h := range s.handlers.dragStart {
		h.fn(ctx)
	}
	if node != nil && node.Callbacks != nil && node.Callbacks.OnDragStart != nil {
		node.Callbacks.OnDragStart(ctx)
	}
	s.emitInteractionEvent(EventDragStart, node, wx, wy, lx, ly, button, mods, ctx, PinchContext{})
}

func (s *Scene) fireDrag(node *Node, pointerID int, wx, wy, startX, startY, deltaX, deltaY, screenDX, screenDY float64, button MouseButton, mods KeyModifiers) {
	var lx, ly float64
	var entityID uint32
	var userData any
	if node != nil {
		lx, ly = node.WorldToLocal(wx, wy)
		entityID = node.EntityID
		userData = node.UserData
	}
	ctx := DragContext{
		Node: node, EntityID: entityID, UserData: userData,
		GlobalX: wx, GlobalY: wy, LocalX: lx, LocalY: ly,
		StartX: startX, StartY: startY, DeltaX: deltaX, DeltaY: deltaY,
		ScreenDeltaX: screenDX, ScreenDeltaY: screenDY,
		Button: button, PointerID: pointerID, Modifiers: mods,
	}
	for _, h := range s.handlers.drag {
		h.fn(ctx)
	}
	if node != nil && node.Callbacks != nil && node.Callbacks.OnDrag != nil {
		node.Callbacks.OnDrag(ctx)
	}
	s.emitInteractionEvent(EventDrag, node, wx, wy, lx, ly, button, mods, ctx, PinchContext{})
}

func (s *Scene) fireDragEnd(node *Node, pointerID int, wx, wy, startX, startY, deltaX, deltaY, screenDX, screenDY float64, button MouseButton, mods KeyModifiers) {
	var lx, ly float64
	var entityID uint32
	var userData any
	if node != nil {
		lx, ly = node.WorldToLocal(wx, wy)
		entityID = node.EntityID
		userData = node.UserData
	}
	ctx := DragContext{
		Node: node, EntityID: entityID, UserData: userData,
		GlobalX: wx, GlobalY: wy, LocalX: lx, LocalY: ly,
		StartX: startX, StartY: startY, DeltaX: deltaX, DeltaY: deltaY,
		ScreenDeltaX: screenDX, ScreenDeltaY: screenDY,
		Button: button, PointerID: pointerID, Modifiers: mods,
	}
	for _, h := range s.handlers.dragEnd {
		h.fn(ctx)
	}
	if node != nil && node.Callbacks != nil && node.Callbacks.OnDragEnd != nil {
		node.Callbacks.OnDragEnd(ctx)
	}
	s.emitInteractionEvent(EventDragEnd, node, wx, wy, lx, ly, button, mods, ctx, PinchContext{})
}

func (s *Scene) fireBackgroundClick(pointerID int, wx, wy float64, button MouseButton, mods KeyModifiers) {
	if len(s.handlers.bgClick) == 0 {
		return
	}
	ctx := ClickContext{
		GlobalX: wx, GlobalY: wy,
		Button: button, PointerID: pointerID, Modifiers: mods,
	}
	for _, h := range s.handlers.bgClick {
		h.fn(ctx)
	}
}

func (s *Scene) firePinch(ctx PinchContext, mods KeyModifiers) {
	for _, h := range s.handlers.pinch {
		h.fn(ctx)
	}

	// Fire per-node OnPinch on the hit node of the first pinch pointer.
	var pinchNode *Node
	if s.pinch.pointer0 > 0 && s.pinch.pointer0 < maxPointers {
		pinchNode = s.pointers[s.pinch.pointer0].hitNode
		if pinchNode != nil && pinchNode.Callbacks != nil && pinchNode.Callbacks.OnPinch != nil {
			pinchNode.Callbacks.OnPinch(ctx)
		}
	}
	// Pinch is a global gesture  -  always emit to EntityStore (node may be nil).
	s.emitInteractionEvent(EventPinch, pinchNode, ctx.CenterX, ctx.CenterY, 0, 0,
		MouseButtonLeft, mods, DragContext{}, ctx)
}

// --- ECS bridge ---

func (s *Scene) emitInteractionEvent(eventType EventType, node *Node, wx, wy, lx, ly float64,
	button MouseButton, mods KeyModifiers, drag DragContext, pinch PinchContext) {
	if s.store == nil {
		return
	}
	// Pinch is a global gesture  -  emit even without a hit node or EntityID.
	if eventType != EventPinch && (node == nil || node.EntityID == 0) {
		return
	}
	var entityID uint32
	if node != nil {
		entityID = node.EntityID
	}
	s.store.EmitEvent(InteractionEvent{
		Type:         eventType,
		EntityID:     entityID,
		GlobalX:      wx,
		GlobalY:      wy,
		LocalX:       lx,
		LocalY:       ly,
		Button:       button,
		Modifiers:    mods,
		StartX:       drag.StartX,
		StartY:       drag.StartY,
		DeltaX:       drag.DeltaX,
		DeltaY:       drag.DeltaY,
		ScreenDeltaX: drag.ScreenDeltaX,
		ScreenDeltaY: drag.ScreenDeltaY,
		Scale:        pinch.Scale,
		ScaleDelta:   pinch.ScaleDelta,
		Rotation:     pinch.Rotation,
		RotDelta:     pinch.RotDelta,
	})
}
