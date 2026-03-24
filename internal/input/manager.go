package input

import (
	"math"

	"github.com/devthicket/willow/internal/node"
	"github.com/devthicket/willow/internal/types"
	"github.com/hajimehoshi/ebiten/v2"
)

// --- Constants ---

const (
	MaxPointers         = 10  // pointer 0 = mouse, 1-9 = touch
	DefaultDragDeadZone = 4.0 // pixels
)

// --- Function pointers (wired by root/core) ---

var (
	// ScreenToWorldFn converts screen coordinates to world coordinates.
	// Set by root to use the primary camera's ScreenToWorld.
	ScreenToWorldFn func(sx, sy float64) (float64, float64)

	// NodeDimensionsFn returns the display dimensions for a node.
	// Set by root (delegates to camera.NodeDimensions or render.NodeDimensions).
	NodeDimensionsFn func(n *node.Node) (w, h float64)

	// EmitInteractionEventFn bridges input events to the ECS entity store.
	// Set by root; nil when no EntityStore is configured.
	EmitInteractionEventFn func(eventType types.EventType, n *node.Node, wx, wy, lx, ly float64,
		button types.MouseButton, mods types.KeyModifiers, drag node.DragContext, pinch node.PinchContext)

	// RebuildSortedChildrenFn sorts a node's children by ZIndex.
	// Set by root (delegates to render.RebuildSortedChildren).
	RebuildSortedChildrenFn func(n *node.Node)
)

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
	HitNode     *node.Node // exported for testing
	hoverNode   *node.Node
	dragging    bool
	button      types.MouseButton
}

// --- Pinch state ---

type pinchState struct {
	Active       bool // exported for testing
	Pointer0     int  // exported for testing
	Pointer1     int
	initialDist  float64
	initialAngle float64
	prevDist     float64
	prevAngle    float64
}

// --- Handler types ---

type pointerHandler struct {
	id uint32
	fn func(node.PointerContext)
}

type clickHandler struct {
	id uint32
	fn func(node.ClickContext)
}

type dragHandler struct {
	id uint32
	fn func(node.DragContext)
}

type pinchHandler struct {
	id uint32
	fn func(node.PinchContext)
}

// HandlerRegistry stores event handler slices for all event types.
type HandlerRegistry struct {
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
	reg   *HandlerRegistry
	event types.EventType
}

// Remove unregisters this callback.
func (h CallbackHandle) Remove() {
	if h.reg == nil {
		return
	}
	switch h.event {
	case types.EventPointerDown:
		h.reg.pointerDown = removePointerHandler(h.reg.pointerDown, h.id)
	case types.EventPointerUp:
		h.reg.pointerUp = removePointerHandler(h.reg.pointerUp, h.id)
	case types.EventPointerMove:
		h.reg.pointerMove = removePointerHandler(h.reg.pointerMove, h.id)
	case types.EventPointerEnter:
		h.reg.pointerEnter = removePointerHandler(h.reg.pointerEnter, h.id)
	case types.EventPointerLeave:
		h.reg.pointerLeave = removePointerHandler(h.reg.pointerLeave, h.id)
	case types.EventClick:
		h.reg.click = removeClickHandler(h.reg.click, h.id)
	case types.EventBgClick:
		h.reg.bgClick = removeClickHandler(h.reg.bgClick, h.id)
	case types.EventDragStart:
		h.reg.dragStart = removeDragHandler(h.reg.dragStart, h.id)
	case types.EventDrag:
		h.reg.drag = removeDragHandler(h.reg.drag, h.id)
	case types.EventDragEnd:
		h.reg.dragEnd = removeDragHandler(h.reg.dragEnd, h.id)
	case types.EventPinch:
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

// --- Manager ---

// Manager owns all input state. Replaces the input-related fields on Scene.
type Manager struct {
	Handlers     HandlerRegistry
	Captured     [MaxPointers]*node.Node
	Pointers     [MaxPointers]pointerState
	HitBuf       []*node.Node
	DragDeadZone float64
	TouchMap     [MaxPointers]ebiten.TouchID
	TouchUsed    [MaxPointers]bool
	PrevTouchIDs []ebiten.TouchID
	Pinch        pinchState
	InjectQueue  []SyntheticPointerEvent
}

// NewManager creates a Manager with default settings.
func NewManager() *Manager {
	return &Manager{
		DragDeadZone: DefaultDragDeadZone,
	}
}

// --- Pointer state queries ---

// PointerPosition returns the last known pointer position in screen space
// for the mouse pointer (pointer 0).
func (m *Manager) PointerPosition() (x, y float64) {
	ps := &m.Pointers[0]
	return ps.lastScreenX, ps.lastScreenY
}

// IsPointerDown reports whether the given mouse button is currently pressed.
// This checks the mouse pointer (pointer 0) state.
func (m *Manager) IsPointerDown(button types.MouseButton) bool {
	ps := &m.Pointers[0]
	return ps.down && ps.button == button
}

// --- Event registration ---

func (m *Manager) OnPointerDown(fn func(node.PointerContext)) CallbackHandle {
	m.Handlers.nextID++
	id := m.Handlers.nextID
	m.Handlers.pointerDown = append(m.Handlers.pointerDown, pointerHandler{id: id, fn: fn})
	return CallbackHandle{id: id, reg: &m.Handlers, event: types.EventPointerDown}
}

func (m *Manager) OnPointerUp(fn func(node.PointerContext)) CallbackHandle {
	m.Handlers.nextID++
	id := m.Handlers.nextID
	m.Handlers.pointerUp = append(m.Handlers.pointerUp, pointerHandler{id: id, fn: fn})
	return CallbackHandle{id: id, reg: &m.Handlers, event: types.EventPointerUp}
}

func (m *Manager) OnPointerMove(fn func(node.PointerContext)) CallbackHandle {
	m.Handlers.nextID++
	id := m.Handlers.nextID
	m.Handlers.pointerMove = append(m.Handlers.pointerMove, pointerHandler{id: id, fn: fn})
	return CallbackHandle{id: id, reg: &m.Handlers, event: types.EventPointerMove}
}

func (m *Manager) OnPointerEnter(fn func(node.PointerContext)) CallbackHandle {
	m.Handlers.nextID++
	id := m.Handlers.nextID
	m.Handlers.pointerEnter = append(m.Handlers.pointerEnter, pointerHandler{id: id, fn: fn})
	return CallbackHandle{id: id, reg: &m.Handlers, event: types.EventPointerEnter}
}

func (m *Manager) OnPointerLeave(fn func(node.PointerContext)) CallbackHandle {
	m.Handlers.nextID++
	id := m.Handlers.nextID
	m.Handlers.pointerLeave = append(m.Handlers.pointerLeave, pointerHandler{id: id, fn: fn})
	return CallbackHandle{id: id, reg: &m.Handlers, event: types.EventPointerLeave}
}

func (m *Manager) OnClick(fn func(node.ClickContext)) CallbackHandle {
	m.Handlers.nextID++
	id := m.Handlers.nextID
	m.Handlers.click = append(m.Handlers.click, clickHandler{id: id, fn: fn})
	return CallbackHandle{id: id, reg: &m.Handlers, event: types.EventClick}
}

func (m *Manager) OnBackgroundClick(fn func(node.ClickContext)) CallbackHandle {
	m.Handlers.nextID++
	id := m.Handlers.nextID
	m.Handlers.bgClick = append(m.Handlers.bgClick, clickHandler{id: id, fn: fn})
	return CallbackHandle{id: id, reg: &m.Handlers, event: types.EventBgClick}
}

func (m *Manager) OnDragStart(fn func(node.DragContext)) CallbackHandle {
	m.Handlers.nextID++
	id := m.Handlers.nextID
	m.Handlers.dragStart = append(m.Handlers.dragStart, dragHandler{id: id, fn: fn})
	return CallbackHandle{id: id, reg: &m.Handlers, event: types.EventDragStart}
}

func (m *Manager) OnDrag(fn func(node.DragContext)) CallbackHandle {
	m.Handlers.nextID++
	id := m.Handlers.nextID
	m.Handlers.drag = append(m.Handlers.drag, dragHandler{id: id, fn: fn})
	return CallbackHandle{id: id, reg: &m.Handlers, event: types.EventDrag}
}

func (m *Manager) OnDragEnd(fn func(node.DragContext)) CallbackHandle {
	m.Handlers.nextID++
	id := m.Handlers.nextID
	m.Handlers.dragEnd = append(m.Handlers.dragEnd, dragHandler{id: id, fn: fn})
	return CallbackHandle{id: id, reg: &m.Handlers, event: types.EventDragEnd}
}

func (m *Manager) OnPinch(fn func(node.PinchContext)) CallbackHandle {
	m.Handlers.nextID++
	id := m.Handlers.nextID
	m.Handlers.pinch = append(m.Handlers.pinch, pinchHandler{id: id, fn: fn})
	return CallbackHandle{id: id, reg: &m.Handlers, event: types.EventPinch}
}

// CapturePointer routes all events for pointerID to the given node.
func (m *Manager) CapturePointer(pointerID int, n *node.Node) {
	if pointerID >= 0 && pointerID < MaxPointers {
		m.Captured[pointerID] = n
	}
}

// ReleasePointer stops routing events for pointerID to a captured node.
func (m *Manager) ReleasePointer(pointerID int) {
	if pointerID >= 0 && pointerID < MaxPointers {
		m.Captured[pointerID] = nil
	}
}

// SetDragDeadZone sets the minimum movement in pixels before a drag starts.
func (m *Manager) SetDragDeadZone(pixels float64) {
	m.DragDeadZone = pixels
}

// --- Input processing ---

// ReadModifiers reads the current keyboard modifier state.
func ReadModifiers() types.KeyModifiers {
	var mods types.KeyModifiers
	if ebiten.IsKeyPressed(ebiten.KeyShift) || ebiten.IsKeyPressed(ebiten.KeyShiftLeft) || ebiten.IsKeyPressed(ebiten.KeyShiftRight) {
		mods |= types.ModShift
	}
	if ebiten.IsKeyPressed(ebiten.KeyControl) || ebiten.IsKeyPressed(ebiten.KeyControlLeft) || ebiten.IsKeyPressed(ebiten.KeyControlRight) {
		mods |= types.ModCtrl
	}
	if ebiten.IsKeyPressed(ebiten.KeyAlt) || ebiten.IsKeyPressed(ebiten.KeyAltLeft) || ebiten.IsKeyPressed(ebiten.KeyAltRight) {
		mods |= types.ModAlt
	}
	if ebiten.IsKeyPressed(ebiten.KeyMeta) || ebiten.IsKeyPressed(ebiten.KeyMetaLeft) || ebiten.IsKeyPressed(ebiten.KeyMetaRight) {
		mods |= types.ModMeta
	}
	return mods
}

// screenToWorld converts screen coords to world coords using the function pointer.
func screenToWorld(sx, sy float64) (float64, float64) {
	if ScreenToWorldFn != nil {
		return ScreenToWorldFn(sx, sy)
	}
	return sx, sy
}

// ProcessInput handles all mouse and touch input for a frame.
// root is the scene graph root for hit testing.
func (m *Manager) ProcessInput(root *node.Node) {
	mods := ReadModifiers()

	if !m.processInjectedInput(root, mods) {
		m.processMousePointer(root, mods)
	}
	m.processTouchPointers(root, mods)
	m.detectPinch(mods)
}

func (m *Manager) processMousePointer(root *node.Node, mods types.KeyModifiers) {
	mx, my := ebiten.CursorPosition()
	sx, sy := float64(mx), float64(my)
	wx, wy := screenToWorld(sx, sy)

	var pressed bool
	var button types.MouseButton
	left := ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)
	right := ebiten.IsMouseButtonPressed(ebiten.MouseButtonRight)
	middle := ebiten.IsMouseButtonPressed(ebiten.MouseButtonMiddle)

	if left || right || middle {
		pressed = true
		if left {
			button = types.MouseButtonLeft
		} else if right {
			button = types.MouseButtonRight
		} else {
			button = types.MouseButtonMiddle
		}
	}

	m.processPointer(root, 0, wx, wy, sx, sy, pressed, button, mods)
}

func (m *Manager) processTouchPointers(root *node.Node, mods types.KeyModifiers) {
	touchIDs := ebiten.AppendTouchIDs(m.PrevTouchIDs[:0])
	m.PrevTouchIDs = touchIDs

	var activeSlots [MaxPointers]bool
	for _, tid := range touchIDs {
		slot := m.touchSlot(tid)
		if slot < 0 {
			continue
		}
		activeSlots[slot] = true

		tx, ty := ebiten.TouchPosition(tid)
		stx, sty := float64(tx), float64(ty)
		wx, wy := screenToWorld(stx, sty)
		m.processPointer(root, slot, wx, wy, stx, sty, true, types.MouseButtonLeft, mods)
	}

	for i := 1; i < MaxPointers; i++ {
		if m.TouchUsed[i] && !activeSlots[i] {
			ps := &m.Pointers[i]
			if ps.down {
				m.processPointer(root, i, ps.lastX, ps.lastY, ps.lastScreenX, ps.lastScreenY, false, types.MouseButtonLeft, mods)
			}
			m.TouchUsed[i] = false
			m.TouchMap[i] = 0
		}
	}
}

func (m *Manager) touchSlot(tid ebiten.TouchID) int {
	for i := 1; i < MaxPointers; i++ {
		if m.TouchUsed[i] && m.TouchMap[i] == tid {
			return i
		}
	}
	for i := 1; i < MaxPointers; i++ {
		if !m.TouchUsed[i] {
			m.TouchUsed[i] = true
			m.TouchMap[i] = tid
			return i
		}
	}
	return -1
}

// ProcessPointer is the exported version of processPointer (for testing and bridging).
func (m *Manager) ProcessPointer(root *node.Node, pointerID int, wx, wy, sx, sy float64, pressed bool, button types.MouseButton, mods types.KeyModifiers) {
	m.processPointer(root, pointerID, wx, wy, sx, sy, pressed, button, mods)
}

// processPointer runs the pointer state machine for a single pointer.
func (m *Manager) processPointer(root *node.Node, pointerID int, wx, wy, sx, sy float64, pressed bool, button types.MouseButton, mods types.KeyModifiers) {
	ps := &m.Pointers[pointerID]

	var target *node.Node
	if m.Captured[pointerID] != nil {
		target = m.Captured[pointerID]
	} else {
		target = m.hitTest(root, wx, wy)
	}

	if ps.hoverNode != nil && ps.hoverNode.Disposed {
		ps.hoverNode = nil
	}

	if target != ps.hoverNode {
		if ps.hoverNode != nil {
			m.firePointerLeave(ps.hoverNode, pointerID, wx, wy, button, mods)
		}
		if target != nil {
			m.firePointerEnter(target, pointerID, wx, wy, button, mods)
		}
		ps.hoverNode = target
	}

	if pressed && !ps.down {
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
		ps.HitNode = target
		ps.dragging = false

		m.firePointerDown(target, pointerID, wx, wy, ps.button, mods)
	} else if !pressed && ps.down {
		sdx := sx - ps.lastScreenX
		sdy := sy - ps.lastScreenY
		if ps.dragging {
			m.fireDragEnd(ps.HitNode, pointerID, wx, wy, ps.startX, ps.startY,
				wx-ps.lastX, wy-ps.lastY, sdx, sdy, ps.button, mods)
		} else if ps.HitNode != nil && ps.HitNode == target {
			m.fireClick(target, pointerID, wx, wy, ps.button, mods)
		} else if ps.HitNode == nil && target == nil {
			m.fireBackgroundClick(pointerID, wx, wy, ps.button, mods)
		}

		m.firePointerUp(target, pointerID, wx, wy, ps.button, mods)

		m.Captured[pointerID] = nil
		ps.down = false
		ps.HitNode = nil
		ps.dragging = false
	} else if pressed && ps.down {
		if wx != ps.lastX || wy != ps.lastY || sx != ps.lastScreenX || sy != ps.lastScreenY {
			sdx := sx - ps.lastScreenX
			sdy := sy - ps.lastScreenY
			if !ps.dragging {
				dx := wx - ps.startX
				dy := wy - ps.startY
				if math.Sqrt(dx*dx+dy*dy) > m.DragDeadZone {
					ps.dragging = true
					m.fireDragStart(ps.HitNode, pointerID, wx, wy, ps.startX, ps.startY,
						wx-ps.startX, wy-ps.startY, sx-ps.screenX, sy-ps.screenY, ps.button, mods)
				}
			}
			if ps.dragging {
				m.fireDrag(ps.HitNode, pointerID, wx, wy, ps.startX, ps.startY,
					wx-ps.lastX, wy-ps.lastY, sdx, sdy, ps.button, mods)
			}
		}
		ps.lastX = wx
		ps.lastY = wy
		ps.lastScreenX = sx
		ps.lastScreenY = sy
	} else if !pressed && !ps.down {
		if wx != ps.lastX || wy != ps.lastY {
			m.firePointerMove(target, pointerID, wx, wy, button, mods)
			ps.lastX = wx
			ps.lastY = wy
		}
	}
}

// --- Pinch detection ---

func (m *Manager) detectPinch(mods types.KeyModifiers) {
	var active [MaxPointers]bool
	var count int
	for i := 1; i < MaxPointers; i++ {
		if m.Pointers[i].down {
			active[i] = true
			count++
		}
	}

	if count == 2 {
		var p0, p1 int
		found := 0
		for i := 1; i < MaxPointers; i++ {
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

		ps0 := &m.Pointers[p0]
		ps1 := &m.Pointers[p1]

		cx := (ps0.lastX + ps1.lastX) / 2
		cy := (ps0.lastY + ps1.lastY) / 2
		dx := ps1.lastX - ps0.lastX
		dy := ps1.lastY - ps0.lastY
		dist := math.Sqrt(dx*dx + dy*dy)
		angle := math.Atan2(dy, dx)

		if !m.Pinch.Active {
			m.Pinch.Active = true
			m.Pinch.Pointer0 = p0
			m.Pinch.Pointer1 = p1
			m.Pinch.initialDist = dist
			m.Pinch.initialAngle = angle
			m.Pinch.prevDist = dist
			m.Pinch.prevAngle = angle

			ps0.dragging = false
			ps1.dragging = false
		} else {
			scale := 1.0
			if m.Pinch.initialDist > 0 {
				scale = dist / m.Pinch.initialDist
			}
			scaleDelta := 0.0
			if m.Pinch.prevDist > 0 {
				scaleDelta = dist/m.Pinch.prevDist - 1.0
			}
			rotation := angle - m.Pinch.initialAngle
			rotDelta := angle - m.Pinch.prevAngle

			ctx := node.PinchContext{
				CenterX:    cx,
				CenterY:    cy,
				Scale:      scale,
				ScaleDelta: scaleDelta,
				Rotation:   rotation,
				RotDelta:   rotDelta,
			}
			m.firePinch(ctx, mods)

			m.Pinch.prevDist = dist
			m.Pinch.prevAngle = angle
		}

		ps0.dragging = false
		ps1.dragging = false
	} else if m.Pinch.Active {
		m.Pinch.Active = false
	}
}
