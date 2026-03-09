package node

// NodeCallbacks holds per-node event handler functions. Allocated on
// first callback setter call. Most nodes never receive callbacks.
type NodeCallbacks struct {
	OnPointerDown  func(PointerContext)
	OnPointerUp    func(PointerContext)
	OnPointerMove  func(PointerContext)
	OnClick        func(ClickContext)
	OnDragStart    func(DragContext)
	OnDrag         func(DragContext)
	OnDragEnd      func(DragContext)
	OnPinch        func(PinchContext)
	OnPointerEnter func(PointerContext)
	OnPointerLeave func(PointerContext)
}

// EnsureCallbacks lazily allocates the callback struct.
func (n *Node) EnsureCallbacks() *NodeCallbacks {
	if n.Callbacks == nil {
		n.Callbacks = &NodeCallbacks{}
	}
	return n.Callbacks
}

// HasOnPointerDown reports whether a pointer-down callback is registered.
func (n *Node) HasOnPointerDown() bool {
	return n.Callbacks != nil && n.Callbacks.OnPointerDown != nil
}

// HasOnPointerUp reports whether a pointer-up callback is registered.
func (n *Node) HasOnPointerUp() bool {
	return n.Callbacks != nil && n.Callbacks.OnPointerUp != nil
}

// HasOnPointerMove reports whether a pointer-move callback is registered.
func (n *Node) HasOnPointerMove() bool {
	return n.Callbacks != nil && n.Callbacks.OnPointerMove != nil
}

// HasOnClick reports whether a click callback is registered.
func (n *Node) HasOnClick() bool {
	return n.Callbacks != nil && n.Callbacks.OnClick != nil
}

// HasOnDragStart reports whether a drag-start callback is registered.
func (n *Node) HasOnDragStart() bool {
	return n.Callbacks != nil && n.Callbacks.OnDragStart != nil
}

// HasOnDrag reports whether a drag callback is registered.
func (n *Node) HasOnDrag() bool {
	return n.Callbacks != nil && n.Callbacks.OnDrag != nil
}

// HasOnDragEnd reports whether a drag-end callback is registered.
func (n *Node) HasOnDragEnd() bool {
	return n.Callbacks != nil && n.Callbacks.OnDragEnd != nil
}

// HasOnPinch reports whether a pinch callback is registered.
func (n *Node) HasOnPinch() bool {
	return n.Callbacks != nil && n.Callbacks.OnPinch != nil
}

// HasOnPointerEnter reports whether a pointer-enter callback is registered.
func (n *Node) HasOnPointerEnter() bool {
	return n.Callbacks != nil && n.Callbacks.OnPointerEnter != nil
}

// HasOnPointerLeave reports whether a pointer-leave callback is registered.
func (n *Node) HasOnPointerLeave() bool {
	return n.Callbacks != nil && n.Callbacks.OnPointerLeave != nil
}
