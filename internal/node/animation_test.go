package node

import (
	"testing"

	"github.com/devthicket/willow/internal/types"
)

func makeFrames(n int) []types.TextureRegion {
	frames := make([]types.TextureRegion, n)
	for i := range frames {
		frames[i] = types.TextureRegion{X: uint16(i * 32), Width: 32, Height: 32, OriginalW: 32, OriginalH: 32}
	}
	return frames
}

func TestAnimationPlayerPlay(t *testing.T) {
	n := NewNode("test", types.NodeTypeSprite)
	ap := NewAnimationPlayer(n)
	frames := makeFrames(4)
	ap.AddSequence(AnimationSequence{Name: "walk", Frames: frames, FPS: 10, Loop: true})
	ap.Play("walk")

	if ap.Current() != "walk" {
		t.Errorf("Current = %q, want walk", ap.Current())
	}
	if !ap.IsPlaying() {
		t.Error("should be playing")
	}
	if n.TextureRegion_ != frames[0] {
		t.Error("initial frame not set")
	}
}

func TestAnimationPlayerAdvance(t *testing.T) {
	n := NewNode("test", types.NodeTypeSprite)
	ap := NewAnimationPlayer(n)
	frames := makeFrames(4)
	ap.AddSequence(AnimationSequence{Name: "walk", Frames: frames, FPS: 10, Loop: true})
	ap.Play("walk")

	// At 10 FPS, each frame is 0.1s. Advance 0.15s to reach frame 1
	ap.Update(0.15)
	if ap.Frame() != 1 {
		t.Errorf("frame = %d, want 1", ap.Frame())
	}
	if n.TextureRegion_ != frames[1] {
		t.Error("texture region not updated to frame 1")
	}
}

func TestAnimationPlayerLoop(t *testing.T) {
	n := NewNode("test", types.NodeTypeSprite)
	ap := NewAnimationPlayer(n)
	frames := makeFrames(3)
	ap.AddSequence(AnimationSequence{Name: "idle", Frames: frames, FPS: 10, Loop: true})
	ap.Play("idle")

	// Advance past all frames (0.35s at 10 FPS = 3.5 frames)
	ap.Update(0.35)
	if !ap.IsPlaying() {
		t.Error("looping animation should still be playing")
	}
	// Should have wrapped around
	if ap.Frame() >= len(frames) {
		t.Errorf("frame %d should be < %d", ap.Frame(), len(frames))
	}
}

func TestAnimationPlayerOneShotComplete(t *testing.T) {
	n := NewNode("test", types.NodeTypeSprite)
	ap := NewAnimationPlayer(n)
	frames := makeFrames(3)
	ap.AddSequence(AnimationSequence{Name: "attack", Frames: frames, FPS: 10, Loop: false})

	var completed string
	ap.OnComplete(func(name string) { completed = name })
	ap.Play("attack")

	// Advance past all frames
	ap.Update(0.5)
	if ap.IsPlaying() {
		t.Error("one-shot should have stopped")
	}
	if completed != "attack" {
		t.Errorf("OnComplete name = %q, want attack", completed)
	}
	if ap.Frame() != 2 {
		t.Errorf("should be on last frame, got %d", ap.Frame())
	}
}

func TestAnimationPlayerPlayIdempotent(t *testing.T) {
	n := NewNode("test", types.NodeTypeSprite)
	ap := NewAnimationPlayer(n)
	frames := makeFrames(4)
	ap.AddSequence(AnimationSequence{Name: "walk", Frames: frames, FPS: 10, Loop: true})
	ap.Play("walk")
	ap.Update(0.15) // advance to frame 1

	// Playing same animation should not reset
	ap.Play("walk")
	if ap.Frame() != 1 {
		t.Errorf("Play same animation reset frame to %d, want 1", ap.Frame())
	}
}

func TestAnimationPlayerPlayFrom(t *testing.T) {
	n := NewNode("test", types.NodeTypeSprite)
	ap := NewAnimationPlayer(n)
	frames := makeFrames(4)
	ap.AddSequence(AnimationSequence{Name: "walk", Frames: frames, FPS: 10, Loop: true})

	ap.PlayFrom("walk", 2)
	if ap.Frame() != 2 {
		t.Errorf("frame = %d, want 2", ap.Frame())
	}
	if n.TextureRegion_ != frames[2] {
		t.Error("PlayFrom didn't set correct texture region")
	}
}

func TestAnimationPlayerStop(t *testing.T) {
	n := NewNode("test", types.NodeTypeSprite)
	ap := NewAnimationPlayer(n)
	frames := makeFrames(4)
	ap.AddSequence(AnimationSequence{Name: "walk", Frames: frames, FPS: 10, Loop: true})
	ap.Play("walk")
	ap.Stop()
	if ap.IsPlaying() {
		t.Error("should not be playing after Stop")
	}
}

func TestAnimationPlayerUnknownSequence(t *testing.T) {
	n := NewNode("test", types.NodeTypeSprite)
	ap := NewAnimationPlayer(n)
	ap.Play("nonexistent")
	if ap.IsPlaying() {
		t.Error("should not play nonexistent sequence")
	}
}
