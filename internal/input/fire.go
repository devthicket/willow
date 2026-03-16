package input

import (
	"github.com/devthicket/willow/internal/node"
	"github.com/devthicket/willow/internal/types"
)

// --- Event dispatch ---

func (m *Manager) firePointerDown(n *node.Node, pointerID int, wx, wy float64, button types.MouseButton, mods types.KeyModifiers) {
	var lx, ly float64
	var entityID uint32
	var userData any
	if n != nil {
		lx, ly = n.WorldToLocal(wx, wy)
		entityID = n.EntityID
		userData = n.UserData
	}
	ctx := node.PointerContext{
		Node: n, EntityID: entityID, UserData: userData,
		GlobalX: wx, GlobalY: wy, LocalX: lx, LocalY: ly,
		Button: button, PointerID: pointerID, Modifiers: mods,
	}
	for _, h := range m.Handlers.pointerDown {
		h.fn(ctx)
	}
	if n != nil && n.Callbacks != nil && n.Callbacks.OnPointerDown != nil {
		n.Callbacks.OnPointerDown(ctx)
	}
	if EmitInteractionEventFn != nil {
		EmitInteractionEventFn(types.EventPointerDown, n, wx, wy, lx, ly, button, mods, node.DragContext{}, node.PinchContext{})
	}
}

func (m *Manager) firePointerUp(n *node.Node, pointerID int, wx, wy float64, button types.MouseButton, mods types.KeyModifiers) {
	var lx, ly float64
	var entityID uint32
	var userData any
	if n != nil {
		lx, ly = n.WorldToLocal(wx, wy)
		entityID = n.EntityID
		userData = n.UserData
	}
	ctx := node.PointerContext{
		Node: n, EntityID: entityID, UserData: userData,
		GlobalX: wx, GlobalY: wy, LocalX: lx, LocalY: ly,
		Button: button, PointerID: pointerID, Modifiers: mods,
	}
	for _, h := range m.Handlers.pointerUp {
		h.fn(ctx)
	}
	if n != nil && n.Callbacks != nil && n.Callbacks.OnPointerUp != nil {
		n.Callbacks.OnPointerUp(ctx)
	}
	if EmitInteractionEventFn != nil {
		EmitInteractionEventFn(types.EventPointerUp, n, wx, wy, lx, ly, button, mods, node.DragContext{}, node.PinchContext{})
	}
}

func (m *Manager) firePointerMove(n *node.Node, pointerID int, wx, wy float64, button types.MouseButton, mods types.KeyModifiers) {
	var lx, ly float64
	var entityID uint32
	var userData any
	if n != nil {
		lx, ly = n.WorldToLocal(wx, wy)
		entityID = n.EntityID
		userData = n.UserData
	}
	ctx := node.PointerContext{
		Node: n, EntityID: entityID, UserData: userData,
		GlobalX: wx, GlobalY: wy, LocalX: lx, LocalY: ly,
		Button: button, PointerID: pointerID, Modifiers: mods,
	}
	for _, h := range m.Handlers.pointerMove {
		h.fn(ctx)
	}
	if n != nil && n.Callbacks != nil && n.Callbacks.OnPointerMove != nil {
		n.Callbacks.OnPointerMove(ctx)
	}
	if EmitInteractionEventFn != nil {
		EmitInteractionEventFn(types.EventPointerMove, n, wx, wy, lx, ly, button, mods, node.DragContext{}, node.PinchContext{})
	}
}

func (m *Manager) firePointerEnter(n *node.Node, pointerID int, wx, wy float64, button types.MouseButton, mods types.KeyModifiers) {
	var lx, ly float64
	var entityID uint32
	var userData any
	if n != nil {
		lx, ly = n.WorldToLocal(wx, wy)
		entityID = n.EntityID
		userData = n.UserData
	}
	ctx := node.PointerContext{
		Node: n, EntityID: entityID, UserData: userData,
		GlobalX: wx, GlobalY: wy, LocalX: lx, LocalY: ly,
		Button: button, PointerID: pointerID, Modifiers: mods,
	}
	for _, h := range m.Handlers.pointerEnter {
		h.fn(ctx)
	}
	if n != nil && n.Callbacks != nil && n.Callbacks.OnPointerEnter != nil {
		n.Callbacks.OnPointerEnter(ctx)
	}
	if EmitInteractionEventFn != nil {
		EmitInteractionEventFn(types.EventPointerEnter, n, wx, wy, lx, ly, button, mods, node.DragContext{}, node.PinchContext{})
	}
}

func (m *Manager) firePointerLeave(n *node.Node, pointerID int, wx, wy float64, button types.MouseButton, mods types.KeyModifiers) {
	var lx, ly float64
	var entityID uint32
	var userData any
	if n != nil {
		lx, ly = n.WorldToLocal(wx, wy)
		entityID = n.EntityID
		userData = n.UserData
	}
	ctx := node.PointerContext{
		Node: n, EntityID: entityID, UserData: userData,
		GlobalX: wx, GlobalY: wy, LocalX: lx, LocalY: ly,
		Button: button, PointerID: pointerID, Modifiers: mods,
	}
	for _, h := range m.Handlers.pointerLeave {
		h.fn(ctx)
	}
	if n != nil && n.Callbacks != nil && n.Callbacks.OnPointerLeave != nil {
		n.Callbacks.OnPointerLeave(ctx)
	}
	if EmitInteractionEventFn != nil {
		EmitInteractionEventFn(types.EventPointerLeave, n, wx, wy, lx, ly, button, mods, node.DragContext{}, node.PinchContext{})
	}
}

func (m *Manager) fireClick(n *node.Node, pointerID int, wx, wy float64, button types.MouseButton, mods types.KeyModifiers) {
	var lx, ly float64
	var entityID uint32
	var userData any
	if n != nil {
		lx, ly = n.WorldToLocal(wx, wy)
		entityID = n.EntityID
		userData = n.UserData
	}
	ctx := node.ClickContext{
		Node: n, EntityID: entityID, UserData: userData,
		GlobalX: wx, GlobalY: wy, LocalX: lx, LocalY: ly,
		Button: button, PointerID: pointerID, Modifiers: mods,
	}
	for _, h := range m.Handlers.click {
		h.fn(ctx)
	}
	if n != nil && n.Callbacks != nil && n.Callbacks.OnClick != nil {
		n.Callbacks.OnClick(ctx)
	}
	if EmitInteractionEventFn != nil {
		EmitInteractionEventFn(types.EventClick, n, wx, wy, lx, ly, button, mods, node.DragContext{}, node.PinchContext{})
	}
}

func (m *Manager) fireDragStart(n *node.Node, pointerID int, wx, wy, startX, startY, deltaX, deltaY, screenDX, screenDY float64, button types.MouseButton, mods types.KeyModifiers) {
	var lx, ly float64
	var entityID uint32
	var userData any
	if n != nil {
		lx, ly = n.WorldToLocal(wx, wy)
		entityID = n.EntityID
		userData = n.UserData
	}
	ctx := node.DragContext{
		Node: n, EntityID: entityID, UserData: userData,
		GlobalX: wx, GlobalY: wy, LocalX: lx, LocalY: ly,
		StartX: startX, StartY: startY, DeltaX: deltaX, DeltaY: deltaY,
		ScreenDeltaX: screenDX, ScreenDeltaY: screenDY,
		Button: button, PointerID: pointerID, Modifiers: mods,
	}
	for _, h := range m.Handlers.dragStart {
		h.fn(ctx)
	}
	if n != nil && n.Callbacks != nil && n.Callbacks.OnDragStart != nil {
		n.Callbacks.OnDragStart(ctx)
	}
	if EmitInteractionEventFn != nil {
		EmitInteractionEventFn(types.EventDragStart, n, wx, wy, lx, ly, button, mods, ctx, node.PinchContext{})
	}
}

func (m *Manager) fireDrag(n *node.Node, pointerID int, wx, wy, startX, startY, deltaX, deltaY, screenDX, screenDY float64, button types.MouseButton, mods types.KeyModifiers) {
	var lx, ly float64
	var entityID uint32
	var userData any
	if n != nil {
		lx, ly = n.WorldToLocal(wx, wy)
		entityID = n.EntityID
		userData = n.UserData
	}
	ctx := node.DragContext{
		Node: n, EntityID: entityID, UserData: userData,
		GlobalX: wx, GlobalY: wy, LocalX: lx, LocalY: ly,
		StartX: startX, StartY: startY, DeltaX: deltaX, DeltaY: deltaY,
		ScreenDeltaX: screenDX, ScreenDeltaY: screenDY,
		Button: button, PointerID: pointerID, Modifiers: mods,
	}
	for _, h := range m.Handlers.drag {
		h.fn(ctx)
	}
	if n != nil && n.Callbacks != nil && n.Callbacks.OnDrag != nil {
		n.Callbacks.OnDrag(ctx)
	}
	if EmitInteractionEventFn != nil {
		EmitInteractionEventFn(types.EventDrag, n, wx, wy, lx, ly, button, mods, ctx, node.PinchContext{})
	}
}

func (m *Manager) fireDragEnd(n *node.Node, pointerID int, wx, wy, startX, startY, deltaX, deltaY, screenDX, screenDY float64, button types.MouseButton, mods types.KeyModifiers) {
	var lx, ly float64
	var entityID uint32
	var userData any
	if n != nil {
		lx, ly = n.WorldToLocal(wx, wy)
		entityID = n.EntityID
		userData = n.UserData
	}
	ctx := node.DragContext{
		Node: n, EntityID: entityID, UserData: userData,
		GlobalX: wx, GlobalY: wy, LocalX: lx, LocalY: ly,
		StartX: startX, StartY: startY, DeltaX: deltaX, DeltaY: deltaY,
		ScreenDeltaX: screenDX, ScreenDeltaY: screenDY,
		Button: button, PointerID: pointerID, Modifiers: mods,
	}
	for _, h := range m.Handlers.dragEnd {
		h.fn(ctx)
	}
	if n != nil && n.Callbacks != nil && n.Callbacks.OnDragEnd != nil {
		n.Callbacks.OnDragEnd(ctx)
	}
	if EmitInteractionEventFn != nil {
		EmitInteractionEventFn(types.EventDragEnd, n, wx, wy, lx, ly, button, mods, ctx, node.PinchContext{})
	}
}

func (m *Manager) fireBackgroundClick(pointerID int, wx, wy float64, button types.MouseButton, mods types.KeyModifiers) {
	if len(m.Handlers.bgClick) == 0 {
		return
	}
	ctx := node.ClickContext{
		GlobalX: wx, GlobalY: wy,
		Button: button, PointerID: pointerID, Modifiers: mods,
	}
	for _, h := range m.Handlers.bgClick {
		h.fn(ctx)
	}
}

// FirePointerDown is the exported version of firePointerDown (for testing and bridging).
func (m *Manager) FirePointerDown(n *node.Node, pointerID int, wx, wy float64, button types.MouseButton, mods types.KeyModifiers) {
	m.firePointerDown(n, pointerID, wx, wy, button, mods)
}

// FireClick is the exported version of fireClick.
func (m *Manager) FireClick(n *node.Node, pointerID int, wx, wy float64, button types.MouseButton, mods types.KeyModifiers) {
	m.fireClick(n, pointerID, wx, wy, button, mods)
}

// FireDragStart is the exported version of fireDragStart.
func (m *Manager) FireDragStart(n *node.Node, pointerID int, wx, wy, startX, startY, deltaX, deltaY, screenDX, screenDY float64, button types.MouseButton, mods types.KeyModifiers) {
	m.fireDragStart(n, pointerID, wx, wy, startX, startY, deltaX, deltaY, screenDX, screenDY, button, mods)
}

// FireDrag is the exported version of fireDrag.
func (m *Manager) FireDrag(n *node.Node, pointerID int, wx, wy, startX, startY, deltaX, deltaY, screenDX, screenDY float64, button types.MouseButton, mods types.KeyModifiers) {
	m.fireDrag(n, pointerID, wx, wy, startX, startY, deltaX, deltaY, screenDX, screenDY, button, mods)
}

// FireDragEnd is the exported version of fireDragEnd.
func (m *Manager) FireDragEnd(n *node.Node, pointerID int, wx, wy, startX, startY, deltaX, deltaY, screenDX, screenDY float64, button types.MouseButton, mods types.KeyModifiers) {
	m.fireDragEnd(n, pointerID, wx, wy, startX, startY, deltaX, deltaY, screenDX, screenDY, button, mods)
}

// FirePinch is the exported version of firePinch.
func (m *Manager) FirePinch(ctx node.PinchContext, mods types.KeyModifiers) {
	m.firePinch(ctx, mods)
}

func (m *Manager) firePinch(ctx node.PinchContext, mods types.KeyModifiers) {
	for _, h := range m.Handlers.pinch {
		h.fn(ctx)
	}

	var pinchNode *node.Node
	if m.Pinch.Pointer0 > 0 && m.Pinch.Pointer0 < MaxPointers {
		pinchNode = m.Pointers[m.Pinch.Pointer0].HitNode
		if pinchNode != nil && pinchNode.Callbacks != nil && pinchNode.Callbacks.OnPinch != nil {
			pinchNode.Callbacks.OnPinch(ctx)
		}
	}
	if EmitInteractionEventFn != nil {
		EmitInteractionEventFn(types.EventPinch, pinchNode, ctx.CenterX, ctx.CenterY, 0, 0,
			types.MouseButtonLeft, mods, node.DragContext{}, ctx)
	}
}
