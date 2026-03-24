# Scene Manager

## Summary

Add a scene manager that handles switching between scenes (title screen → gameplay → pause → game over) with optional transitions (fade, wipe, etc.). Currently `willow.Run()` takes a single `Scene` with no built-in way to switch.

## Current State

`willow.Run(scene, cfg)` wraps a single `Scene` in a `gameShell` and calls `ebiten.RunGame()`. Users who need multiple scenes must manage the swap themselves — replacing the root node tree, cameras, and state manually. There's no lifecycle (enter/exit), no transition effects, and no scene stack for push/pop patterns (e.g. pause overlay).

## Proposed API

```go
// SceneManager manages a stack of scenes with optional transitions.
type SceneManager struct {
    // ...
}

func NewSceneManager(initial *Scene) *SceneManager

// Push adds a scene on top of the stack. The current scene's OnExit is called,
// then the new scene's OnEnter. If a transition is set, it plays between them.
func (sm *SceneManager) Push(scene *Scene)

// Pop removes the top scene and returns to the previous one.
// Panics if only one scene remains.
func (sm *SceneManager) Pop()

// Replace swaps the current scene for a new one (no stack growth).
func (sm *SceneManager) Replace(scene *Scene)

// SetTransition configures the transition effect for the next scene change.
// Pass nil to disable transitions.
func (sm *SceneManager) SetTransition(t Transition)

// Current returns the active scene.
func (sm *SceneManager) Current() *Scene
```

### Scene Lifecycle Hooks

```go
// Optional interfaces a Scene (or a user wrapper) can implement:

// OnEnter is called when the scene becomes active (after transition completes).
scene.SetOnEnter(func() { ... })

// OnExit is called when the scene is being left (before transition starts).
scene.SetOnExit(func() { ... })
```

### Transitions

```go
// Transition controls the visual effect between scene changes.
type Transition interface {
    // Duration returns total transition time in seconds.
    Duration() float32
    // Update advances the transition. Progress goes 0→1.
    Update(dt float32)
    // Draw renders the transition overlay on top of the screen.
    Draw(screen *ebiten.Image, progress float32)
    // Done reports whether the transition has completed.
    Done() bool
}

// Built-in transitions:
func FadeTransition(duration float32, color color.Color) Transition
```

## Design Notes

- **SceneManager replaces Scene in `willow.Run()`**: `Run()` should accept either a `*Scene` or a `*SceneManager`. Or `SceneManager` wraps a `Scene` and implements the same update/draw contract.
- **Transitions are optional**: Default is instant switch. `SetTransition()` applies to the next change only (not sticky). For sticky transitions, add `SetDefaultTransition()`.
- **Fade-out then fade-in**: During fade-out, the old scene is frozen (no updates). During fade-in, the new scene updates normally. This matches user expectations.
- **Stack-based for overlay patterns**: Push a pause menu on top of gameplay; pop to resume. The scene below stays in memory but doesn't update.
- **Resource cleanup**: `OnExit` is the place to release textures, stop audio, etc. The manager doesn't own resource lifecycle — scenes do.
- **Keep it simple**: No coroutines, no async loading screens (that's a user concern). Just push/pop/replace with lifecycle hooks and an optional visual transition.

## Implementation Scope

Medium — ~100-150 lines. SceneManager struct, stack management, transition state machine, integration with `gameShell`. FadeTransition is ~30 lines.

## Prior Art

- Ludum (`scene_manager.go`): Stack-based push/pop/replace with fade overlay. Transition freezes old scene during fade-out, runs new scene during fade-in. Simple and correct. Hardcodes 1/60s tick.
- Ebitengine examples: The `scenes` example uses a manual state machine with no transitions.
- Godot `SceneTree`: Full-featured with autoloads, groups, signals. Way more than needed here.
