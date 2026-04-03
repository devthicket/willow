package core

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
)

// RangeFloat is a float64 that can be unmarshaled from either a number or a
// [min, max] array. When an array is provided, the value is randomly chosen
// in that range at parse time.
type RangeFloat float64

func (r *RangeFloat) UnmarshalJSON(data []byte) error {
	// Try number first.
	var v float64
	if err := json.Unmarshal(data, &v); err == nil {
		*r = RangeFloat(v)
		return nil
	}
	// Try [min, max] array.
	var arr [2]float64
	if err := json.Unmarshal(data, &arr); err != nil {
		return fmt.Errorf("expected number or [min, max]: %s", string(data))
	}
	*r = RangeFloat(arr[0] + rand.Float64()*(arr[1]-arr[0]))
	return nil
}

// F returns the float64 value.
func (r RangeFloat) F() float64 { return float64(r) }

// TestStep represents a single action in a test script.
type TestStep struct {
	Action string     `json:"action"`
	Label  string     `json:"label,omitempty"`
	Text   string     `json:"text,omitempty"`
	Key    string     `json:"key,omitempty"`
	X      RangeFloat `json:"x,omitempty"`
	Y      RangeFloat `json:"y,omitempty"`
	FromX  RangeFloat `json:"fromX,omitempty"`
	FromY  RangeFloat `json:"fromY,omitempty"`
	ToX    RangeFloat `json:"toX,omitempty"`
	ToY    RangeFloat `json:"toY,omitempty"`
	Frames int        `json:"frames,omitempty"`
	Ms     int        `json:"ms,omitempty"`    // duration in ms (converted to frames at 60fps)
	Tween  int        `json:"tween,omitempty"` // duration in ms for smooth move
	Delay  int        `json:"delay,omitempty"` // per-character delay in ms for type action
	Color  string     `json:"color,omitempty"` // expected color for pixelcheck (e.g. "red", "blue", "#FF0000")

	// GIF config fields (start_gif action only).
	Inset       int  `json:"inset,omitempty"`
	Width       int  `json:"width,omitempty"`
	Height      int  `json:"height,omitempty"`
	FPS         int  `json:"fps,omitempty"`
	MaxWidth    int  `json:"maxWidth,omitempty"`
	MaxHeight   int  `json:"maxHeight,omitempty"`
	MaxColors   int  `json:"maxColors,omitempty"`
	DropEvery   int  `json:"dropEvery,omitempty"`
	NoDedup     bool `json:"noDedup,omitempty"`
	NoDiff      bool `json:"noDiff,omitempty"`
	NoDownscale bool `json:"noDownscale,omitempty"`
}

type testScript struct {
	ShowCursor bool       `json:"show_cursor,omitempty"`
	Steps      []TestStep `json:"steps"`
}

// PointerOverride is a persistent pointer state that the input manager reads
// each frame in place of real mouse input. Set by "press", "release", and
// "move" test actions.
type PointerOverride struct {
	Active  bool
	X, Y    float64
	Pressed bool
}

// moveTween tracks an in-progress smooth cursor movement.
type moveTween struct {
	active       bool
	fromX, fromY float64
	toX, toY     float64
	frame        int
	totalFrames  int
	pressed      bool // preserve pressed state during tween
}

// TestRunner sequences injected input events and screenshots across frames.
type TestRunner struct {
	Steps      []TestStep
	Cursor     int
	WaitCount  int
	IsDone     bool
	ShowCursor bool

	// Auto-release keys: key action does key_down + auto key_up after 2 frames.
	pendingUps []pendingKeyUp

	// Pointer holds the persistent pointer state for press/release/move actions.
	Pointer PointerOverride

	// tween holds an in-progress smooth cursor movement.
	tween moveTween
}

type pendingKeyUp struct {
	key    string
	frames int
}

// LoadTestScript parses a JSON test script and returns a TestRunner.
func LoadTestScript(jsonData []byte) (*TestRunner, error) {
	var script testScript
	if err := json.Unmarshal(jsonData, &script); err != nil {
		return nil, fmt.Errorf("parse test script: %w", err)
	}
	if len(script.Steps) == 0 {
		return nil, fmt.Errorf("parse test script: no steps")
	}
	return &TestRunner{Steps: script.Steps, ShowCursor: script.ShowCursor}, nil
}

// Done reports whether all steps have been executed.
func (r *TestRunner) Done() bool {
	return r.IsDone
}

// StepAction is the interface that Scene uses to execute test steps.
// The Scene provides screenshot, inject, and queue access.
type StepAction struct {
	Screenshot  func(label string)
	StartGif    func(label string, cfg GifConfig)
	StopGif     func()
	InjectClick func(x, y float64)
	InjectDrag  func(fromX, fromY, toX, toY float64, frames int)
	InjectMove  func(x, y float64)
	InjectText  func(text string)
	InjectKey   func(key string)
	KeyDown     func(key string)
	KeyUp       func(key string)
	QueueLen    func() int
	PixelCheck  func(x, y int, color string) // reads pixel at (x,y) and checks against expected color
}

// msToFrames converts milliseconds to frames at 60fps. Returns at least 1.
func msToFrames(ms int) int {
	f := int(math.Round(float64(ms) / 1000.0 * 60.0))
	if f < 1 {
		return 1
	}
	return f
}

// resolveFrames returns the frame count from either Ms or Frames fields.
// Ms takes priority if set.
func (st *TestStep) resolveFrames() int {
	if st.Ms > 0 {
		return msToFrames(st.Ms)
	}
	return st.Frames
}

// easeInOutCubic provides smooth acceleration and deceleration.
func easeInOutCubic(t float64) float64 {
	if t < 0.5 {
		return 4 * t * t * t
	}
	return 1 - math.Pow(-2*t+2, 3)/2
}

// Step advances the test runner by one frame.
func (r *TestRunner) Step(a StepAction) {
	// Tick move tween — this runs every frame regardless of other state.
	if r.tween.active {
		r.tween.frame++
		if r.tween.frame >= r.tween.totalFrames {
			// Tween complete — snap to destination.
			r.Pointer.X = r.tween.toX
			r.Pointer.Y = r.tween.toY
			r.tween.active = false
			if r.tween.pressed {
				// Drag complete — release the pointer.
				r.Pointer.Pressed = false
			}
		} else {
			t := float64(r.tween.frame) / float64(r.tween.totalFrames)
			if !r.tween.pressed {
				t = easeInOutCubic(t) // ease for moves, linear for drags
			}
			r.Pointer.X = r.tween.fromX + (r.tween.toX-r.tween.fromX)*t
			r.Pointer.Y = r.tween.fromY + (r.tween.toY-r.tween.fromY)*t
		}
		return // tween consumes this frame
	}

	// Tick pending auto-release keys.
	for i := len(r.pendingUps) - 1; i >= 0; i-- {
		r.pendingUps[i].frames--
		if r.pendingUps[i].frames <= 0 {
			if a.KeyUp != nil {
				a.KeyUp(r.pendingUps[i].key)
			}
			r.pendingUps = append(r.pendingUps[:i], r.pendingUps[i+1:]...)
		}
	}

	if r.IsDone {
		return
	}
	if a.QueueLen() > 0 {
		return
	}
	if r.WaitCount > 0 {
		r.WaitCount--
		return
	}
	if r.Cursor >= len(r.Steps) {
		r.IsDone = true
		return
	}

	st := r.Steps[r.Cursor]
	r.Cursor++

	switch st.Action {
	case "screenshot":
		a.Screenshot(st.Label)
	case "pixelcheck":
		if a.PixelCheck != nil {
			a.PixelCheck(int(st.X.F()), int(st.Y.F()), st.Color)
		}
	case "click":
		a.InjectClick(st.X.F(), st.Y.F())
	case "press":
		r.Pointer = PointerOverride{Active: true, X: st.X.F(), Y: st.Y.F(), Pressed: true}
	case "release":
		r.Pointer = PointerOverride{Active: true, X: st.X.F(), Y: st.Y.F(), Pressed: false}
	case "press_down":
		r.Pointer.Pressed = true
	case "press_up":
		r.Pointer.Pressed = false
	case "move":
		if st.Tween > 0 {
			// Smooth move: interpolate from current position over tween ms.
			totalFrames := int(math.Round(float64(st.Tween) / 1000.0 * 60.0))
			if totalFrames < 2 {
				totalFrames = 2
			}
			fromX, fromY := r.Pointer.X, r.Pointer.Y
			if !r.Pointer.Active {
				fromX, fromY = st.X.F(), st.Y.F() // no prior position, start at dest
			}
			r.Pointer.Active = true
			r.Pointer.Pressed = false
			r.tween = moveTween{
				active:      true,
				fromX:       fromX,
				fromY:       fromY,
				toX:         st.X.F(),
				toY:         st.Y.F(),
				frame:       0,
				totalFrames: totalFrames,
				pressed:     r.Pointer.Pressed,
			}
		} else {
			r.Pointer = PointerOverride{Active: true, X: st.X.F(), Y: st.Y.F(), Pressed: false}
		}
	case "drag":
		frames := st.resolveFrames()
		if frames < 2 {
			frames = 2
		}
		// Use a tween so the virtual cursor dot follows the drag visually.
		r.Pointer = PointerOverride{Active: true, X: st.FromX.F(), Y: st.FromY.F(), Pressed: true}
		r.tween = moveTween{
			active:      true,
			fromX:       st.FromX.F(),
			fromY:       st.FromY.F(),
			toX:         st.ToX.F(),
			toY:         st.ToY.F(),
			frame:       0,
			totalFrames: frames,
			pressed:     true,
		}
	case "type":
		if a.InjectText != nil {
			if st.Delay > 0 && len(st.Text) > 1 {
				// Expand into individual characters with delays between them.
				// Insert remaining chars + waits after the current cursor position.
				runes := []rune(st.Text)
				a.InjectText(string(runes[0]))
				extra := make([]TestStep, 0, (len(runes)-1)*2)
				delayFrames := msToFrames(st.Delay)
				for _, ch := range runes[1:] {
					extra = append(extra, TestStep{Action: "wait", Frames: delayFrames})
					extra = append(extra, TestStep{Action: "type", Text: string(ch)})
				}
				// Splice extra steps into the Steps slice at the current cursor.
				tail := make([]TestStep, len(r.Steps[r.Cursor:]))
				copy(tail, r.Steps[r.Cursor:])
				r.Steps = append(r.Steps[:r.Cursor], append(extra, tail...)...)
			} else {
				a.InjectText(st.Text)
			}
		}
	case "key":
		if a.InjectKey != nil {
			a.InjectKey(st.Key)
		}
		if a.KeyDown != nil {
			a.KeyDown(st.Key)
			r.pendingUps = append(r.pendingUps, pendingKeyUp{key: st.Key, frames: 2})
		}
	case "key_down":
		if a.KeyDown != nil {
			a.KeyDown(st.Key)
		}
	case "key_up":
		if a.KeyUp != nil {
			a.KeyUp(st.Key)
		}
	case "start_gif":
		if a.StartGif != nil {
			a.StartGif(st.Label, GifConfig{
				X: int(st.X.F()), Y: int(st.Y.F()),
				Width: st.Width, Height: st.Height,
				Inset:       st.Inset,
				FPS:         st.FPS,
				MaxWidth:    st.MaxWidth,
				MaxHeight:   st.MaxHeight,
				MaxColors:   st.MaxColors,
				DropEvery:   st.DropEvery,
				NoDedup:     st.NoDedup,
				NoDiff:      st.NoDiff,
				NoDownscale: st.NoDownscale,
			})
		}
	case "stop_gif":
		if a.StopGif != nil {
			a.StopGif()
		}
	case "wait":
		f := st.resolveFrames()
		if f > 0 {
			r.WaitCount = f - 1
		}
	}

	if r.Cursor >= len(r.Steps) && r.WaitCount == 0 && a.QueueLen() == 0 {
		r.IsDone = true
	}
}
