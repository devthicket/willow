# Animation Player

## Summary

Add a higher-level animation sequence manager for sprite-based frame animation. This manages multiple named animation sequences (idle, walk, attack, etc.) on a single node, with switching, looping, and completion callbacks.

## Current State

Willow handles sprite rendering, atlas regions, and the render pipeline — but animation playback (advancing frames over time, managing multiple sequences per entity) is left to the user. Every game with animated sprites needs this; it should be a first-class feature.

## Proposed API

```go
// AnimationSequence defines a single named animation.
type AnimationSequence struct {
    Name     string
    Frames   []types.TextureRegion  // ordered frames from atlas
    FPS      float32                // playback speed
    Loop     bool                   // loop or one-shot
}

// AnimationPlayer manages multiple sequences on a node.
type AnimationPlayer struct {
    // ...
}

func NewAnimationPlayer(node *node.Node) *AnimationPlayer

// AddSequence registers a named animation.
func (ap *AnimationPlayer) AddSequence(seq AnimationSequence)

// Play switches to the named sequence. No-op if already playing it.
func (ap *AnimationPlayer) Play(name string)

// PlayFrom switches and starts from a specific frame.
func (ap *AnimationPlayer) PlayFrom(name string, frame int)

// Current returns the active sequence name.
func (ap *AnimationPlayer) Current() string

// IsPlaying reports whether the current sequence is still running (always true for looping).
func (ap *AnimationPlayer) IsPlaying() bool

// OnComplete is called when a non-looping sequence finishes.
// Callback receives the sequence name.
func (ap *AnimationPlayer) OnComplete(fn func(name string))

// Update advances the animation by dt. Called from the game loop.
func (ap *AnimationPlayer) Update(dt float32)
```

## Design Notes

- **Atlas-native**: Frames are `TextureRegion` references into an existing atlas — no separate image loading, preserves batching.
- **Caller-driven Update()**: Not tied to the scene graph's update cycle. The user calls `Update(dt)` so they control timing (pause, slow-mo, etc.).
- **Play() is idempotent**: Calling `Play("walk")` when already playing "walk" does nothing — no restart, no flicker. Use `PlayFrom("walk", 0)` to force restart.
- **Frame change applies the region to the node**: When the frame advances, the player sets the node's texture region directly. This keeps the node as the single source of truth for rendering.
- **No blending/transitions**: Frame-by-frame sprite animation doesn't need crossfade. Keep it simple.
- **Per-sequence FPS**: Different animations can run at different speeds (idle=6fps, attack=12fps).

## Implementation Scope

Small-medium — ~80-120 lines. Core struct + Update logic + sequence switching. No new dependencies.

## Prior Art

- Ludum (`animation_player.go`): Map of named `AnimationSet` sequences, `SetCurrentAnimation(id)` switching. Uses `FIXED_DELTA` (hardcoded 1/60) instead of dt — less flexible.
- Godot `AnimationPlayer`: Full timeline system with tracks, blending, root motion. Overkill for frame-by-frame sprites but the named-sequence + play/stop API is the right model.
- Aseprite tag system: Artists export named animation tags — the player should map 1:1 to these.
