package core

import (
	"encoding/json"
	"fmt"
)

// TestStep represents a single action in a test script.
type TestStep struct {
	Action string  `json:"action"`
	Label  string  `json:"label,omitempty"`
	Text   string  `json:"text,omitempty"`
	Key    string  `json:"key,omitempty"`
	X      float64 `json:"x,omitempty"`
	Y      float64 `json:"y,omitempty"`
	FromX  float64 `json:"fromX,omitempty"`
	FromY  float64 `json:"fromY,omitempty"`
	ToX    float64 `json:"toX,omitempty"`
	ToY    float64 `json:"toY,omitempty"`
	Frames int     `json:"frames,omitempty"`

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
	Steps []TestStep `json:"steps"`
}

// TestRunner sequences injected input events and screenshots across frames.
type TestRunner struct {
	Steps     []TestStep
	Cursor    int
	WaitCount int
	IsDone    bool
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
	return &TestRunner{Steps: script.Steps}, nil
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
	QueueLen    func() int
}

// Step advances the test runner by one frame.
func (r *TestRunner) Step(a StepAction) {
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
	case "click":
		a.InjectClick(st.X, st.Y)
	case "drag":
		frames := st.Frames
		if frames < 2 {
			frames = 2
		}
		a.InjectDrag(st.FromX, st.FromY, st.ToX, st.ToY, frames)
	case "move":
		if a.InjectMove != nil {
			a.InjectMove(st.X, st.Y)
		}
	case "type":
		if a.InjectText != nil {
			a.InjectText(st.Text)
		}
	case "key":
		if a.InjectKey != nil {
			a.InjectKey(st.Key)
		}
	case "start_gif":
		if a.StartGif != nil {
			a.StartGif(st.Label, GifConfig{
				X: int(st.X), Y: int(st.Y),
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
		if st.Frames > 0 {
			r.WaitCount = st.Frames - 1
		}
	}

	if r.Cursor >= len(r.Steps) && r.WaitCount == 0 && a.QueueLen() == 0 {
		r.IsDone = true
	}
}
