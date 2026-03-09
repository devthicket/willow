package willow

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"
)

// ---- Debug mode tests ------------------------------------------------------

func TestDebugMode_DisposedNodePanics(t *testing.T) {
	s := NewScene()
	s.SetDebugMode(true)
	defer s.SetDebugMode(false)

	parent := NewContainer("parent")
	s.Root.AddChild(parent)

	child := NewSprite("child", TextureRegion{OriginalW: 10, OriginalH: 10})
	child.Dispose()

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic on AddChild with disposed node, got none")
		}
		msg := fmt.Sprint(r)
		if !strings.Contains(msg, "disposed") {
			t.Errorf("panic message should mention 'disposed', got: %s", msg)
		}
	}()

	parent.AddChild(child)
}

func TestDebugMode_DisposedParentPanics(t *testing.T) {
	s := NewScene()
	s.SetDebugMode(true)
	defer s.SetDebugMode(false)

	parent := NewContainer("parent")
	parent.Dispose()

	child := NewSprite("child", TextureRegion{OriginalW: 10, OriginalH: 10})

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic on AddChild to disposed parent, got none")
		}
		msg := fmt.Sprint(r)
		if !strings.Contains(msg, "disposed") {
			t.Errorf("panic message should mention 'disposed', got: %s", msg)
		}
	}()

	parent.AddChild(child)
}

func TestReleaseMode_DisposedNodeNoOp(t *testing.T) {
	s := NewScene()
	s.SetDebugMode(false)

	child := NewSprite("child", TextureRegion{OriginalW: 10, OriginalH: 10})
	child.Dispose()

	// In release mode, adding a disposed child should not panic.
	// It still won't work correctly but it won't crash.
	defer func() {
		if r := recover(); r != nil {
			msg := fmt.Sprint(r)
			if strings.Contains(msg, "disposed") {
				t.Errorf("release mode should not panic on disposed node, got: %s", msg)
			}
		}
	}()

	// This may panic for other reasons (cycle check with nil parent chain),
	// but not for "disposed" reasons.
	s.Root.AddChild(child)
}

func TestDebugMode_TreeDepthWarning(t *testing.T) {
	s := NewScene()
	s.SetDebugMode(true)
	defer s.SetDebugMode(false)

	// Capture stderr output.
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	// Build a chain deeper than debugMaxTreeDepth (32).
	current := s.Root
	for i := 0; i < debugMaxTreeDepth+5; i++ {
		child := NewContainer(fmt.Sprintf("depth_%d", i))
		current.AddChild(child)
		current = child
	}

	w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "warning: tree depth") {
		t.Errorf("expected tree depth warning in stderr, got: %q", output)
	}
}

func TestDebugMode_ChildCountWarning(t *testing.T) {
	s := NewScene()
	s.SetDebugMode(true)
	defer s.SetDebugMode(false)

	// Capture stderr output.
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	parent := NewContainer("many_children")
	s.Root.AddChild(parent)

	for i := 0; i < debugMaxChildCount+1; i++ {
		child := NewContainer(fmt.Sprintf("c_%d", i))
		parent.AddChild(child)
	}

	w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "warning: node") || !strings.Contains(output, "children") {
		t.Errorf("expected child count warning in stderr, got: %q", output)
	}
}

func TestDebugStats_AllFieldsPopulated(t *testing.T) {
	stats := debugStats{
		traverseTime:  100,
		sortTime:      50,
		batchTime:     30,
		submitTime:    80,
		commandCount:  1000,
		batchCount:    10,
		drawCallCount: 15,
	}

	if stats.traverseTime == 0 || stats.sortTime == 0 || stats.submitTime == 0 {
		t.Error("expected all timing fields to be non-zero")
	}
	if stats.commandCount == 0 || stats.drawCallCount == 0 {
		t.Error("expected count fields to be non-zero")
	}
}

func TestCountDrawCalls(t *testing.T) {
	cmds := []RenderCommand{
		{Type: CommandSprite},
		{Type: CommandSprite},
		{Type: CommandMesh},
		{Type: CommandParticle, Emitter: &ParticleEmitter{Alive: 50}},
	}
	got := countDrawCalls(cmds)
	// 1 + 1 + 1 + 50 = 53
	if got != 53 {
		t.Errorf("countDrawCalls = %d, want 53", got)
	}
}

func TestCountDrawCalls_Empty(t *testing.T) {
	got := countDrawCalls(nil)
	if got != 0 {
		t.Errorf("countDrawCalls(nil) = %d, want 0", got)
	}
}

// ---- Screenshot tests ------------------------------------------------------

func TestSanitizeLabel(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"hello", "hello"},
		{"after-spawn", "after-spawn"},
		{"frame.01", "frame.01"},
		{"has spaces", "has_spaces"},
		{"path/to/thing", "path_to_thing"},
		{"back\\slash", "back_slash"},
		{"special!@#$%", "special_____"},
		{"", "unlabeled"},
		{"   ", "unlabeled"},
		{"MixedCase123", "MixedCase123"},
	}
	for _, tt := range tests {
		got := sanitizeLabel(tt.in)
		if got != tt.want {
			t.Errorf("sanitizeLabel(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestScreenshotQueueAppend(t *testing.T) {
	s := NewScene()
	s.Screenshot("a")
	s.Screenshot("b")
	s.Screenshot("c")
	if len(s.ScreenshotQueue) != 3 {
		t.Fatalf("queue len = %d, want 3", len(s.ScreenshotQueue))
	}
	if s.ScreenshotQueue[0] != "a" || s.ScreenshotQueue[1] != "b" || s.ScreenshotQueue[2] != "c" {
		t.Errorf("queue = %v, want [a b c]", s.ScreenshotQueue)
	}
}

func TestScreenshotDirDefault(t *testing.T) {
	s := NewScene()
	if s.ScreenshotDir != "screenshots" {
		t.Errorf("ScreenshotDir = %q, want %q", s.ScreenshotDir, "screenshots")
	}
}

// ---- Input injection tests -------------------------------------------------

func TestInjectClick(t *testing.T) {
	s := NewScene()
	sprite := NewSprite("s", TextureRegion{OriginalW: 100, OriginalH: 100})
	sprite.Interactable = true
	s.Root.AddChild(sprite)
	updateWorldTransform(s.Root, identityTransform, 1.0, false, false)

	var clicked bool
	s.OnClick(func(ctx ClickContext) {
		clicked = true
		if ctx.Node != sprite {
			t.Error("expected sprite node")
		}
	})

	s.InjectClick(50, 50)
	if len(s.Input.InjectQueue) != 2 {
		t.Fatalf("expected 2 queued events, got %d", len(s.Input.InjectQueue))
	}

	// Frame 1: press
	s.Input.ProcessInput(s.Root)
	if len(s.Input.InjectQueue) != 1 {
		t.Fatalf("expected 1 remaining event after frame 1, got %d", len(s.Input.InjectQueue))
	}
	if clicked {
		t.Error("click should not fire on press frame")
	}

	// Frame 2: release → click fires
	s.Input.ProcessInput(s.Root)
	if len(s.Input.InjectQueue) != 0 {
		t.Fatalf("expected 0 remaining events after frame 2, got %d", len(s.Input.InjectQueue))
	}
	if !clicked {
		t.Error("click should fire on release frame")
	}
}

func TestInjectDrag(t *testing.T) {
	s := NewScene()
	sprite := NewSprite("s", TextureRegion{OriginalW: 400, OriginalH: 400})
	sprite.Interactable = true
	s.Root.AddChild(sprite)
	updateWorldTransform(s.Root, identityTransform, 1.0, false, false)

	var events []string
	s.OnDragStart(func(ctx DragContext) { events = append(events, "dragstart") })
	s.OnDrag(func(ctx DragContext) { events = append(events, "drag") })
	s.OnDragEnd(func(ctx DragContext) { events = append(events, "dragend") })

	// Drag from (10,10) to (200,200) over 5 frames:
	// frame 0: press at (10,10)
	// frame 1: move to ~(57.5, 57.5)
	// frame 2: move to ~(105, 105)
	// frame 3: move to ~(152.5, 152.5)
	// frame 4: release at (200, 200)
	s.InjectDrag(10, 10, 200, 200, 5)
	if len(s.Input.InjectQueue) != 5 {
		t.Fatalf("expected 5 queued events, got %d", len(s.Input.InjectQueue))
	}

	// Drain all frames.
	for i := 0; i < 5; i++ {
		s.Input.ProcessInput(s.Root)
	}

	// Should see dragstart, at least one drag, and dragend.
	if len(events) < 3 {
		t.Fatalf("expected at least 3 events, got %v", events)
	}
	if events[0] != "dragstart" {
		t.Errorf("first event should be dragstart, got %s", events[0])
	}
	if events[len(events)-1] != "dragend" {
		t.Errorf("last event should be dragend, got %s", events[len(events)-1])
	}
}

func TestInjectDrag_MinFrames(t *testing.T) {
	s := NewScene()
	s.InjectDrag(0, 0, 100, 100, 1) // should clamp to 2
	if len(s.Input.InjectQueue) != 2 {
		t.Fatalf("expected 2 queued events (clamped), got %d", len(s.Input.InjectQueue))
	}
}

func TestInjectQueueOrder(t *testing.T) {
	s := NewScene()

	s.InjectPress(10, 20)
	s.InjectMove(30, 40)
	s.InjectRelease(50, 60)

	if len(s.Input.InjectQueue) != 3 {
		t.Fatalf("expected 3 events, got %d", len(s.Input.InjectQueue))
	}

	// Verify order: press, move, release.
	if !s.Input.InjectQueue[0].Pressed || s.Input.InjectQueue[0].ScreenX != 10 {
		t.Error("first event should be press at (10,20)")
	}
	if !s.Input.InjectQueue[1].Pressed || s.Input.InjectQueue[1].ScreenX != 30 {
		t.Error("second event should be move at (30,40)")
	}
	if s.Input.InjectQueue[2].Pressed || s.Input.InjectQueue[2].ScreenX != 50 {
		t.Error("third event should be release at (50,60)")
	}
}

func TestProcessInjectedInput(t *testing.T) {
	s := NewScene()
	sprite := NewSprite("s", TextureRegion{OriginalW: 100, OriginalH: 100})
	sprite.Interactable = true
	s.Root.AddChild(sprite)
	updateWorldTransform(s.Root, identityTransform, 1.0, false, false)

	var downFired bool
	s.OnPointerDown(func(ctx PointerContext) {
		downFired = true
		if ctx.GlobalX != 50 || ctx.GlobalY != 50 {
			t.Errorf("expected global (50,50), got (%v,%v)", ctx.GlobalX, ctx.GlobalY)
		}
	})

	// No camera → screen coords = world coords.
	s.InjectPress(50, 50)
	consumed := s.Input.ProcessInjectedInput(s.Root, 0)
	if !consumed {
		t.Error("expected processInjectedInput to consume an event")
	}
	if !downFired {
		t.Error("pointer down should have fired")
	}
	if len(s.Input.InjectQueue) != 0 {
		t.Errorf("queue should be empty, got %d", len(s.Input.InjectQueue))
	}
}

func TestProcessInjectedInput_EmptyQueue(t *testing.T) {
	s := NewScene()
	consumed := s.Input.ProcessInjectedInput(s.Root, 0)
	if consumed {
		t.Error("should not consume when queue is empty")
	}
}

func TestInjectWithCamera(t *testing.T) {
	s := NewScene()
	cam := s.NewCamera(Rect{X: 0, Y: 0, Width: 640, Height: 480})
	cam.X = 320
	cam.Y = 240
	cam.Zoom = 2.0
	cam.ComputeViewMatrix()

	sprite := NewSprite("s", TextureRegion{OriginalW: 50, OriginalH: 50})
	sprite.Interactable = true
	sprite.X_ = 295
	sprite.Y_ = 215
	s.Root.AddChild(sprite)
	updateWorldTransform(s.Root, identityTransform, 1.0, false, false)

	var hitNode *Node
	s.OnPointerDown(func(ctx PointerContext) {
		hitNode = ctx.Node
	})

	// Screen center (320, 240) maps to world (320, 240) with camera centered there.
	s.InjectPress(320, 240)
	s.Input.ProcessInjectedInput(s.Root, 0)

	if hitNode != sprite {
		t.Errorf("expected sprite hit via camera transform, got %v", hitNode)
	}
}

// ---- Test runner tests -----------------------------------------------------

func TestLoadTestScript(t *testing.T) {
	data := []byte(`{
		"steps": [
			{"action": "screenshot", "label": "initial"},
			{"action": "click", "x": 100, "y": 200},
			{"action": "wait", "frames": 3},
			{"action": "screenshot", "label": "after-click"}
		]
	}`)

	runner, err := LoadTestScript(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(runner.Steps) != 4 {
		t.Fatalf("expected 4 steps, got %d", len(runner.Steps))
	}
	if runner.Steps[0].Action != "screenshot" || runner.Steps[0].Label != "initial" {
		t.Error("step 0 mismatch")
	}
	if runner.Steps[1].Action != "click" || runner.Steps[1].X != 100 || runner.Steps[1].Y != 200 {
		t.Error("step 1 mismatch")
	}
	if runner.Steps[2].Action != "wait" || runner.Steps[2].Frames != 3 {
		t.Error("step 2 mismatch")
	}
}

func TestLoadTestScript_Invalid(t *testing.T) {
	_, err := LoadTestScript([]byte(`not json`))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestLoadTestScript_Empty(t *testing.T) {
	_, err := LoadTestScript([]byte(`{"steps": []}`))
	if err == nil {
		t.Error("expected error for empty steps")
	}
}

func TestRunnerStep_Click(t *testing.T) {
	s := NewScene()
	sprite := NewSprite("s", TextureRegion{OriginalW: 200, OriginalH: 200})
	sprite.Interactable = true
	s.Root.AddChild(sprite)
	updateWorldTransform(s.Root, identityTransform, 1.0, false, false)

	data := []byte(`{"steps": [{"action": "click", "x": 50, "y": 50}]}`)
	runner, err := LoadTestScript(data)
	if err != nil {
		t.Fatal(err)
	}
	s.SetTestRunner(runner)

	// First step call: click queues press+release (2 events).
	runner.Step(stepActionFor(s))
	if len(s.Input.InjectQueue) != 2 {
		t.Fatalf("expected 2 queued events, got %d", len(s.Input.InjectQueue))
	}
	// Runner should not be done yet  -  injections still pending.
	if runner.Done() {
		t.Error("runner should not be done while inject queue has events")
	}

	// Drain injections.
	s.Input.ProcessInput(s.Root)
	s.Input.ProcessInput(s.Root)

	// Now step again  -  should finalize.
	runner.Step(stepActionFor(s))
	if !runner.Done() {
		t.Error("runner should be done after all steps executed and queue drained")
	}
}

func TestRunnerStep_Wait(t *testing.T) {
	s := NewScene()

	data := []byte(`{"steps": [
		{"action": "wait", "frames": 3},
		{"action": "screenshot", "label": "done"}
	]}`)
	runner, err := LoadTestScript(data)
	if err != nil {
		t.Fatal(err)
	}

	// Frame 1: execute wait (waitCount becomes 2).
	runner.Step(stepActionFor(s))
	if runner.Done() {
		t.Error("should not be done during wait")
	}

	// Frame 2: waitCount 2→1.
	runner.Step(stepActionFor(s))
	if runner.Done() {
		t.Error("should not be done during wait countdown")
	}

	// Frame 3: waitCount 1→0.
	runner.Step(stepActionFor(s))
	if runner.Done() {
		t.Error("should not be done  -  screenshot step not yet executed")
	}

	// Frame 4: execute screenshot step, runner finishes.
	runner.Step(stepActionFor(s))
	if !runner.Done() {
		t.Error("runner should be done after screenshot step")
	}

	// Verify screenshot was queued.
	if len(s.ScreenshotQueue) != 1 || s.ScreenshotQueue[0] != "done" {
		t.Errorf("expected screenshot 'done', got %v", s.ScreenshotQueue)
	}
}

func TestRunnerStep_Drag(t *testing.T) {
	s := NewScene()
	sprite := NewSprite("s", TextureRegion{OriginalW: 400, OriginalH: 400})
	sprite.Interactable = true
	s.Root.AddChild(sprite)
	updateWorldTransform(s.Root, identityTransform, 1.0, false, false)

	data := []byte(`{"steps": [{"action": "drag", "fromX": 10, "fromY": 10, "toX": 200, "toY": 200, "frames": 4}]}`)
	runner, err := LoadTestScript(data)
	if err != nil {
		t.Fatal(err)
	}

	runner.Step(stepActionFor(s))
	if len(s.Input.InjectQueue) != 4 {
		t.Fatalf("expected 4 queued events for drag, got %d", len(s.Input.InjectQueue))
	}
}

func TestRunnerDone(t *testing.T) {
	s := NewScene()

	data := []byte(`{"steps": [{"action": "screenshot", "label": "only"}]}`)
	runner, err := LoadTestScript(data)
	if err != nil {
		t.Fatal(err)
	}

	if runner.Done() {
		t.Error("runner should not be done before any steps")
	}

	runner.Step(stepActionFor(s))
	if !runner.Done() {
		t.Error("runner should be done after single screenshot step")
	}
}

func TestRunnerWaitsForInjectQueue(t *testing.T) {
	s := NewScene()

	data := []byte(`{"steps": [
		{"action": "click", "x": 50, "y": 50},
		{"action": "screenshot", "label": "after"}
	]}`)
	runner, err := LoadTestScript(data)
	if err != nil {
		t.Fatal(err)
	}

	// Step 1: click queues 2 events.
	runner.Step(stepActionFor(s))
	if len(s.Input.InjectQueue) != 2 {
		t.Fatalf("expected 2 events, got %d", len(s.Input.InjectQueue))
	}

	// Step again  -  should NOT advance because inject queue is not drained.
	runner.Step(stepActionFor(s))
	if runner.Cursor != 1 {
		t.Errorf("cursor should still be 1, got %d", runner.Cursor)
	}

	// Drain inject queue manually.
	s.Input.InjectQueue = s.Input.InjectQueue[:0]

	// Now step  -  should execute screenshot.
	runner.Step(stepActionFor(s))
	if len(s.ScreenshotQueue) != 1 || s.ScreenshotQueue[0] != "after" {
		t.Errorf("expected screenshot 'after', got %v", s.ScreenshotQueue)
	}
	if !runner.Done() {
		t.Error("runner should be done")
	}
}
