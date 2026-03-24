# Camera Shake

## Summary

Add screen shake support to `Camera`. This is a standard 2D game-feel feature that every action game needs — damage feedback, explosions, impacts, environmental rumble.

## Current State

`Camera` supports follow, scroll-to animation, bounds clamping, zoom, and rotation. No shake or trauma system exists. Users would have to manually offset `Camera.X`/`Camera.Y` each frame and restore them, which fights with follow and bounds clamping.

## Proposed API

```go
// Shake applies a one-shot shake with the given intensity and duration.
// Intensity is the maximum pixel displacement; duration is in seconds.
func (c *Camera) Shake(intensity, duration float32)

// SetTrauma adds trauma (0–1) that decays over time. Shake magnitude = trauma².
// Repeated hits accumulate trauma naturally. This is the preferred API for
// gameplay — call it on damage, impacts, etc.
func (c *Camera) SetTrauma(amount float32)
```

## Design Notes

- **Trauma model preferred** over raw shake: `magnitude = trauma²` gives natural falloff. Multiple hits stack. See Squirrel Eiserloh's GDC talk "Math for Game Programmers: Juicing Your Cameras With Math".
- **Offset applied in Update(), before bounds clamping** — shake should respect camera bounds so the player never sees outside the world.
- **Does not modify X/Y directly** — use a separate `shakeOffsetX`/`shakeOffsetY` applied in `ComputeViewMatrix()` so follow/scroll/bounds aren't corrupted.
- **Noise-based displacement** (Perlin or dual-sine) looks better than random jitter. Ludum uses dual-frequency sine; even a simple `sin(t * freq1) + sin(t * freq2)` with different frequencies for X/Y is effective.
- **Decay**: Linear trauma decay (`trauma -= decayRate * dt`) is simple and works. Configurable decay rate with a sensible default (e.g. 1.0/s = full trauma decays in 1 second).

## Implementation Scope

Small — ~30-50 lines in `camera.go`. Fields: `trauma`, `traumaDecay`, `shakeIntensity`, `shakeTime`. Changes to `Update()` and `ComputeViewMatrix()`.

## Prior Art

- Ludum (`camera.go:227-253`): Dual-sine shake with duration-based decay. Hardcoded 1/60s tick. Simpler but less flexible than trauma model.
- Godot: `Camera2D` has built-in `noise`-based shake via `PhantomCameraShake`.
