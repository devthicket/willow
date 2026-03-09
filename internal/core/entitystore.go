package core

import "github.com/phanxgames/willow/internal/types"

// EntityStore is the interface for optional ECS integration.
// When set on a Scene via SetEntityStore, interaction events on nodes with
// a non-zero EntityID are forwarded to the ECS.
type EntityStore interface {
	// EmitEvent delivers an interaction event to the ECS.
	EmitEvent(event InteractionEvent)
}

// InteractionEvent carries interaction data for the ECS bridge.
type InteractionEvent struct {
	Type      types.EventType    // which kind of interaction occurred
	EntityID  uint32             // the ECS entity associated with the hit node
	GlobalX   float64            // pointer X in world coordinates
	GlobalY   float64            // pointer Y in world coordinates
	LocalX    float64            // pointer X in the hit node's local coordinates
	LocalY    float64            // pointer Y in the hit node's local coordinates
	Button    types.MouseButton  // which mouse button is involved
	Modifiers types.KeyModifiers // keyboard modifier keys held during the event
	// Drag fields (valid for EventDragStart, EventDrag, EventDragEnd)
	StartX       float64 // world X where the drag began
	StartY       float64 // world Y where the drag began
	DeltaX       float64 // X movement since the previous drag event
	DeltaY       float64 // Y movement since the previous drag event
	ScreenDeltaX float64 // X movement in screen pixels since the previous drag event
	ScreenDeltaY float64 // Y movement in screen pixels since the previous drag event
	// Pinch fields (valid for EventPinch)
	Scale      float64 // cumulative scale factor since pinch start
	ScaleDelta float64 // frame-to-frame scale change
	Rotation   float64 // cumulative rotation in radians since pinch start
	RotDelta   float64 // frame-to-frame rotation change in radians
}
