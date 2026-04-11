package core

import (
	"testing"
)

func TestLoadTestScript_Valid(t *testing.T) {
	script := `{
		"steps": [
			{"action": "screenshot", "label": "initial"},
			{"action": "wait", "frames": 5},
			{"action": "click", "x": 100, "y": 200}
		]
	}`
	runner, err := LoadTestScript([]byte(script))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(runner.Steps) != 3 {
		t.Fatalf("expected 3 steps, got %d", len(runner.Steps))
	}
	if runner.Steps[0].Action != "screenshot" {
		t.Errorf("step 0 action = %q, want %q", runner.Steps[0].Action, "screenshot")
	}
	if runner.Steps[0].Label != "initial" {
		t.Errorf("step 0 label = %q, want %q", runner.Steps[0].Label, "initial")
	}
	if runner.Steps[1].Frames != 5 {
		t.Errorf("step 1 frames = %d, want 5", runner.Steps[1].Frames)
	}
	if runner.Steps[2].X.F() != 100 || runner.Steps[2].Y.F() != 200 {
		t.Errorf("step 2 coords = (%v, %v), want (100, 200)", runner.Steps[2].X.F(), runner.Steps[2].Y.F())
	}
	if runner.Done() {
		t.Error("runner should not be done before any steps execute")
	}
}

func TestLoadTestScript_ShowCursor(t *testing.T) {
	script := `{"show_cursor": true, "steps": [{"action": "wait", "frames": 1}]}`
	runner, err := LoadTestScript([]byte(script))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !runner.ShowCursor {
		t.Error("ShowCursor should be true")
	}
}

func TestLoadTestScript_EmptySteps(t *testing.T) {
	script := `{"steps": []}`
	_, err := LoadTestScript([]byte(script))
	if err == nil {
		t.Fatal("expected error for empty steps")
	}
}

func TestLoadTestScript_InvalidJSON(t *testing.T) {
	_, err := LoadTestScript([]byte(`{not json`))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestLoadTestScript_MissingSteps(t *testing.T) {
	_, err := LoadTestScript([]byte(`{}`))
	if err == nil {
		t.Fatal("expected error when steps field is missing")
	}
}

func TestLoadTestScript_BareArrayFails(t *testing.T) {
	// Bare arrays are not valid; must be {"steps": [...]}
	_, err := LoadTestScript([]byte(`[{"action": "wait", "frames": 1}]`))
	if err == nil {
		t.Fatal("expected error for bare array (not wrapped in steps object)")
	}
}

// noopStepAction returns a StepAction with no-op callbacks for testing.
func noopStepAction() StepAction {
	return StepAction{
		Screenshot:  func(string) {},
		StartGif:    func(string, GifConfig) {},
		StopGif:     func() {},
		InjectClick: func(float64, float64) {},
		InjectDrag:  func(float64, float64, float64, float64, int) {},
		InjectMove:  func(float64, float64) {},
		InjectText:  func(string) {},
		InjectKey:   func(string) {},
		KeyDown:     func(string) {},
		KeyUp:       func(string) {},
		QueueLen:    func() int { return 0 },
		PixelCheck:  func(int, int, string) {},
	}
}

func TestRunner_WaitAndDone(t *testing.T) {
	script := `{"steps": [{"action": "wait", "frames": 3}]}`
	runner, err := LoadTestScript([]byte(script))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	a := noopStepAction()

	// Should take 3 frames of waiting, then mark done on the next step.
	for i := 0; i < 3; i++ {
		if runner.Done() {
			t.Fatalf("runner done too early at frame %d", i)
		}
		runner.Step(a)
	}
	// After 3 wait frames, the cursor advances past the last step.
	runner.Step(a) // This processes the step after wait completes.
	if !runner.Done() {
		t.Error("runner should be done after all steps execute")
	}
}

func TestRunner_ScreenshotCapture(t *testing.T) {
	script := `{"steps": [{"action": "screenshot", "label": "test-shot"}]}`
	runner, err := LoadTestScript([]byte(script))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var capturedLabel string
	a := noopStepAction()
	a.Screenshot = func(label string) { capturedLabel = label }

	runner.Step(a)
	if capturedLabel != "test-shot" {
		t.Errorf("screenshot label = %q, want %q", capturedLabel, "test-shot")
	}
}

func TestRunner_ClickInjection(t *testing.T) {
	script := `{"steps": [{"action": "click", "x": 50, "y": 75}]}`
	runner, err := LoadTestScript([]byte(script))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var clickX, clickY float64
	a := noopStepAction()
	a.InjectClick = func(x, y float64) { clickX, clickY = x, y }

	runner.Step(a)
	if clickX != 50 || clickY != 75 {
		t.Errorf("click at (%v, %v), want (50, 75)", clickX, clickY)
	}
}

func TestRunner_TypeInjection(t *testing.T) {
	script := `{"steps": [{"action": "type", "text": "hello"}]}`
	runner, err := LoadTestScript([]byte(script))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var typedText string
	a := noopStepAction()
	a.InjectText = func(text string) { typedText = text }

	runner.Step(a)
	if typedText != "hello" {
		t.Errorf("typed text = %q, want %q", typedText, "hello")
	}
}

func TestRunner_MovePointer(t *testing.T) {
	script := `{"steps": [{"action": "move", "x": 200, "y": 300}]}`
	runner, err := LoadTestScript([]byte(script))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	a := noopStepAction()
	runner.Step(a)

	if !runner.Pointer.Active {
		t.Error("pointer should be active after move")
	}
	if runner.Pointer.X != 200 || runner.Pointer.Y != 300 {
		t.Errorf("pointer at (%v, %v), want (200, 300)", runner.Pointer.X, runner.Pointer.Y)
	}
	if runner.Pointer.Pressed {
		t.Error("pointer should not be pressed after move")
	}
}

func TestRunner_MultiStep(t *testing.T) {
	script := `{
		"steps": [
			{"action": "screenshot", "label": "before"},
			{"action": "click", "x": 10, "y": 20},
			{"action": "screenshot", "label": "after"}
		]
	}`
	runner, err := LoadTestScript([]byte(script))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	labels := []string{}
	a := noopStepAction()
	a.Screenshot = func(label string) { labels = append(labels, label) }

	// Step through all actions.
	for !runner.Done() {
		runner.Step(a)
	}

	if len(labels) != 2 {
		t.Fatalf("expected 2 screenshots, got %d", len(labels))
	}
	if labels[0] != "before" || labels[1] != "after" {
		t.Errorf("labels = %v, want [before, after]", labels)
	}
}

func TestRunner_RangeFloat(t *testing.T) {
	script := `{"steps": [{"action": "click", "x": [50, 100], "y": 200}]}`
	runner, err := LoadTestScript([]byte(script))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	x := runner.Steps[0].X.F()
	if x < 50 || x > 100 {
		t.Errorf("RangeFloat x = %v, want between 50 and 100", x)
	}
	if runner.Steps[0].Y.F() != 200 {
		t.Errorf("fixed y = %v, want 200", runner.Steps[0].Y.F())
	}
}

func TestMsToFrames(t *testing.T) {
	cases := []struct {
		ms     int
		expect int
	}{
		{1000, 60},  // 1 second = 60 frames
		{500, 30},   // 0.5 second = 30 frames
		{16, 1},     // ~1 frame
		{0, 1},      // minimum 1
		{100, 6},    // 0.1s = 6 frames
	}
	for _, tc := range cases {
		got := msToFrames(tc.ms)
		if got != tc.expect {
			t.Errorf("msToFrames(%d) = %d, want %d", tc.ms, got, tc.expect)
		}
	}
}
