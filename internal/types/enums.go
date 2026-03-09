package types

// NodeType distinguishes rendering behavior for a Node.
type NodeType uint8

const (
	NodeTypeContainer       NodeType = iota // group node with no visual output
	NodeTypeSprite                          // renders a TextureRegion or custom image
	NodeTypeMesh                            // renders arbitrary triangles via DrawTriangles
	NodeTypeParticleEmitter                 // CPU-simulated particle system
	NodeTypeText                            // renders text via DistanceFieldFont
)

// EventType identifies a kind of interaction event.
type EventType uint8

const (
	EventPointerDown  EventType = iota // fires when a pointer button is pressed
	EventPointerUp                     // fires when a pointer button is released
	EventPointerMove                   // fires when the pointer moves (hover, no button)
	EventClick                         // fires on press then release over the same node
	EventDragStart                     // fires when movement exceeds the drag dead zone
	EventDrag                          // fires each frame while dragging
	EventDragEnd                       // fires when the pointer is released after dragging
	EventPinch                         // fires during a two-finger pinch/rotate gesture
	EventPointerEnter                  // fires when the pointer enters a node's bounds
	EventPointerLeave                  // fires when the pointer leaves a node's bounds
	EventBgClick                       // internal: background click (no node hit)
)

// MouseButton identifies a mouse button.
type MouseButton uint8

const (
	MouseButtonLeft   MouseButton = iota // primary (left) mouse button
	MouseButtonRight                     // secondary (right) mouse button
	MouseButtonMiddle                    // middle mouse button (scroll wheel click)
)

// KeyModifiers is a bitmask of keyboard modifier keys.
// Values can be combined with bitwise OR (e.g. ModShift | ModCtrl).
type KeyModifiers uint8

const (
	ModShift KeyModifiers = 1 << iota // Shift key
	ModCtrl                           // Control key
	ModAlt                            // Alt / Option key
	ModMeta                           // Meta / Command / Windows key
)

// TextAlign controls horizontal text alignment within a TextBlock.
type TextAlign uint8

const (
	TextAlignLeft   TextAlign = iota // align text to the left edge (default)
	TextAlignCenter                  // center text horizontally
	TextAlignRight                   // align text to the right edge
)

// CacheTreeMode controls how a cached subtree invalidates.
type CacheTreeMode uint8

const (
	// CacheTreeManual requires the user to call InvalidateCacheTree() when
	// the subtree changes. Zero overhead on setters. Best for large tilemaps.
	CacheTreeManual CacheTreeMode = iota + 1

	// CacheTreeAuto automatically invalidates when any setter is called on
	// the cached node or its descendants. Small overhead per setter call
	// (one pointer chase + bool write).
	CacheTreeAuto
)

// HitShape is implemented by custom hit-test shapes attached to a Node.
type HitShape interface {
	// Contains reports whether the local-space point (x, y) is inside the shape.
	Contains(x, y float64) bool
}
