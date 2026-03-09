package core

import (
	"testing"

	"github.com/phanxgames/willow/internal/node"
	"github.com/phanxgames/willow/internal/render"
	"github.com/phanxgames/willow/internal/types"
)

func TestNewScene(t *testing.T) {
	root := node.NewNode("root", types.NodeTypeContainer)
	s := NewScene(root)
	if s.Root != root {
		t.Error("Root should match")
	}
	if s.ScreenshotDir != "screenshots" {
		t.Errorf("ScreenshotDir = %q, want %q", s.ScreenshotDir, "screenshots")
	}
	if s.Input == nil {
		t.Error("Input manager should be initialized")
	}
}

func TestNewCamera(t *testing.T) {
	root := node.NewNode("root", types.NodeTypeContainer)
	s := NewScene(root)
	cam := s.NewCamera(types.Rect{X: 0, Y: 0, Width: 800, Height: 600})
	if len(s.Cameras) != 1 {
		t.Fatalf("expected 1 camera, got %d", len(s.Cameras))
	}
	if cam.Viewport.Width != 800 || cam.Viewport.Height != 600 {
		t.Error("camera viewport mismatch")
	}
}

func TestRemoveCamera(t *testing.T) {
	root := node.NewNode("root", types.NodeTypeContainer)
	s := NewScene(root)
	cam := s.NewCamera(types.Rect{Width: 100, Height: 100})
	s.RemoveCamera(cam)
	if len(s.Cameras) != 0 {
		t.Error("camera should be removed")
	}
}

func TestTweenGroup_Tick(t *testing.T) {
	n := node.NewNode("test", types.NodeTypeSprite)
	val := 0.0
	g := &TweenGroup{
		Count:  0,
		Target: n,
	}
	g.Tick(0.1)
	if !g.Done {
		t.Error("tween with no sub-tweens should be done immediately")
	}
	_ = val
}

func TestTweenGroup_Cancel(t *testing.T) {
	g := &TweenGroup{}
	g.Cancel()
	g.Tick(0.1)
	if !g.Done {
		t.Error("cancelled tween should be done after tick")
	}
}

func TestTweenGroup_DisposedTarget(t *testing.T) {
	n := node.NewNode("test", types.NodeTypeSprite)
	n.Dispose()
	g := &TweenGroup{Count: 0, Target: n}
	g.Tick(0.1)
	if !g.Done {
		t.Error("tween with disposed target should be done")
	}
}

func TestTweenGroup_Update_Managed(t *testing.T) {
	g := &TweenGroup{Managed: true}
	g.Update(0.1) // should be no-op
	if g.Done {
		t.Error("managed tween Update should be no-op")
	}
}

func TestTickTweens(t *testing.T) {
	root := node.NewNode("root", types.NodeTypeContainer)
	s := NewScene(root)

	g1 := &TweenGroup{Done: false}
	g2 := &TweenGroup{Done: true} // already done
	g3 := &TweenGroup{Done: false}
	s.Tweens = []*TweenGroup{g1, g2, g3}

	s.TickTweens(0.016)

	// g2 was already done, g1 and g3 have no sub-tweens so become done
	// All should be compacted out
	if len(s.Tweens) != 0 {
		t.Errorf("expected 0 tweens remaining, got %d", len(s.Tweens))
	}
}

func TestRegisterTween(t *testing.T) {
	root := node.NewNode("root", types.NodeTypeContainer)
	s := NewScene(root)
	g := &TweenGroup{}
	s.RegisterTween(g)
	if len(s.Tweens) != 1 {
		t.Fatalf("expected 1 tween, got %d", len(s.Tweens))
	}
}

func TestScreenshot(t *testing.T) {
	root := node.NewNode("root", types.NodeTypeContainer)
	s := NewScene(root)
	s.Screenshot("test-label")
	if len(s.ScreenshotQueue) != 1 {
		t.Fatalf("expected 1 queued screenshot, got %d", len(s.ScreenshotQueue))
	}
	if s.ScreenshotQueue[0] != "test-label" {
		t.Errorf("label = %q, want %q", s.ScreenshotQueue[0], "test-label")
	}
}

func TestSanitizeLabel(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hello", "hello"},
		{"hello world", "hello_world"},
		{"  ", "unlabeled"},
		{"", "unlabeled"},
		{"test-123.png", "test-123.png"},
		{"a/b\\c:d", "a_b_c_d"},
	}
	for _, tt := range tests {
		got := SanitizeLabel(tt.input)
		if got != tt.want {
			t.Errorf("SanitizeLabel(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestCountBatches_Empty(t *testing.T) {
	if CountBatches(nil) != 0 {
		t.Error("expected 0 for empty commands")
	}
}

func TestCountBatches_Single(t *testing.T) {
	cmds := []render.RenderCommand{{Type: render.CommandSprite}}
	if CountBatches(cmds) != 1 {
		t.Errorf("expected 1 batch, got %d", CountBatches(cmds))
	}
}

func TestCountDrawCalls_Empty(t *testing.T) {
	if CountDrawCalls(nil) != 0 {
		t.Error("expected 0 for empty commands")
	}
}

func TestCountDrawCallsCoalesced_Empty(t *testing.T) {
	if CountDrawCallsCoalesced(nil) != 0 {
		t.Error("expected 0 for empty commands")
	}
}

func TestUpdateNodesAndParticles_CallsOnUpdate(t *testing.T) {
	root := node.NewNode("root", types.NodeTypeContainer)
	root.Visible_ = true
	called := false
	root.OnUpdate = func(dt float64) { called = true }
	UpdateNodesAndParticles(root, 0.016)
	if !called {
		t.Error("OnUpdate should have been called")
	}
}

func TestUpdateNodesAndParticles_SkipsInvisible(t *testing.T) {
	root := node.NewNode("root", types.NodeTypeContainer)
	root.Visible_ = false
	called := false
	root.OnUpdate = func(dt float64) { called = true }
	UpdateNodesAndParticles(root, 0.016)
	if called {
		t.Error("OnUpdate should not be called on invisible node")
	}
}

func TestLoadTestScript(t *testing.T) {
	json := []byte(`{"steps":[{"action":"screenshot","label":"test"},{"action":"wait","frames":3}]}`)
	runner, err := LoadTestScript(json)
	if err != nil {
		t.Fatal(err)
	}
	if len(runner.Steps) != 2 {
		t.Fatalf("expected 2 steps, got %d", len(runner.Steps))
	}
	if runner.Done() {
		t.Error("should not be done yet")
	}
}

func TestLoadTestScript_Empty(t *testing.T) {
	json := []byte(`{"steps":[]}`)
	_, err := LoadTestScript(json)
	if err == nil {
		t.Error("expected error for empty steps")
	}
}

func TestLoadTestScript_Invalid(t *testing.T) {
	_, err := LoadTestScript([]byte(`{invalid`))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestTestRunner_Step(t *testing.T) {
	runner := &TestRunner{
		Steps: []TestStep{
			{Action: "wait", Frames: 2},
			{Action: "screenshot", Label: "done"},
		},
	}

	screenshotLabels := []string{}
	action := StepAction{
		Screenshot:  func(l string) { screenshotLabels = append(screenshotLabels, l) },
		InjectClick: func(x, y float64) {},
		InjectDrag:  func(fx, fy, tx, ty float64, f int) {},
		QueueLen:    func() int { return 0 },
	}

	// Step 1: processes "wait 2", sets waitCount=1
	runner.Step(action)
	if runner.IsDone {
		t.Error("should not be done after first step")
	}

	// Step 2: wait countdown (waitCount goes 1→0)
	runner.Step(action)

	// Step 3: processes "screenshot"
	runner.Step(action)
	if len(screenshotLabels) != 1 || screenshotLabels[0] != "done" {
		t.Errorf("expected screenshot 'done', got %v", screenshotLabels)
	}
	if !runner.IsDone {
		t.Error("should be done after all steps")
	}
}
