package input

import (
	"github.com/phanxgames/willow/internal/node"
	"github.com/phanxgames/willow/internal/types"
)

// SyntheticPointerEvent represents a single injected pointer event.
// Screen coordinates are used (matching what an AI sees in screenshots)
// and converted to world coordinates via ScreenToWorldFn.
type SyntheticPointerEvent struct {
	ScreenX float64
	ScreenY float64
	Pressed bool
	Button  types.MouseButton
}

// InjectPress queues a pointer press event at the given screen coordinates.
func (m *Manager) InjectPress(x, y float64) {
	m.InjectQueue = append(m.InjectQueue, SyntheticPointerEvent{
		ScreenX: x, ScreenY: y,
		Pressed: true,
		Button:  types.MouseButtonLeft,
	})
}

// InjectMove queues a pointer move event at the given screen coordinates
// with the button held down.
func (m *Manager) InjectMove(x, y float64) {
	m.InjectQueue = append(m.InjectQueue, SyntheticPointerEvent{
		ScreenX: x, ScreenY: y,
		Pressed: true,
		Button:  types.MouseButtonLeft,
	})
}

// InjectRelease queues a pointer release event at the given screen coordinates.
func (m *Manager) InjectRelease(x, y float64) {
	m.InjectQueue = append(m.InjectQueue, SyntheticPointerEvent{
		ScreenX: x, ScreenY: y,
		Pressed: false,
		Button:  types.MouseButtonLeft,
	})
}

// InjectHover queues a free pointer move (no button held) at the given
// screen coordinates. Use this to trigger OnPointerEnter / OnPointerLeave
// callbacks without pressing a button.
func (m *Manager) InjectHover(x, y float64) {
	m.InjectQueue = append(m.InjectQueue, SyntheticPointerEvent{
		ScreenX: x, ScreenY: y,
		Pressed: false,
	})
}

// InjectClick is a convenience that queues a press followed by a release
// at the same screen coordinates. Consumes two frames.
func (m *Manager) InjectClick(x, y float64) {
	m.InjectPress(x, y)
	m.InjectRelease(x, y)
}

// InjectDrag queues a full drag sequence: press at (fromX, fromY),
// linearly interpolated moves over frames-2 intermediate frames, and
// release at (toX, toY). Minimum frames is 2.
func (m *Manager) InjectDrag(fromX, fromY, toX, toY float64, frames int) {
	if frames < 2 {
		frames = 2
	}
	m.InjectPress(fromX, fromY)
	steps := frames - 2
	for i := 1; i <= steps; i++ {
		t := float64(i) / float64(steps+1)
		x := fromX + (toX-fromX)*t
		y := fromY + (toY-fromY)*t
		m.InjectMove(x, y)
	}
	m.InjectRelease(toX, toY)
}

// ProcessInjectedInput pops one event from the inject queue, converts
// screen→world via ScreenToWorldFn, and feeds it through processPointer.
// Returns true if an event was consumed (real mouse input should be skipped).
func (m *Manager) ProcessInjectedInput(root *node.Node, mods types.KeyModifiers) bool {
	return m.processInjectedInput(root, mods)
}

// processInjectedInput is the internal implementation.
func (m *Manager) processInjectedInput(root *node.Node, mods types.KeyModifiers) bool {
	if len(m.InjectQueue) == 0 {
		return false
	}
	evt := m.InjectQueue[0]
	copy(m.InjectQueue, m.InjectQueue[1:])
	m.InjectQueue = m.InjectQueue[:len(m.InjectQueue)-1]

	wx, wy := screenToWorld(evt.ScreenX, evt.ScreenY)
	m.processPointer(root, 0, wx, wy, evt.ScreenX, evt.ScreenY, evt.Pressed, evt.Button, mods)
	return true
}
