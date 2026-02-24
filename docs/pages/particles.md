# Particles

<p align="center">
  <img src="gif/particles.gif" alt="Particles demo" width="400">
</p>

Willow includes a CPU-simulated particle system. Particles are configured via `EmitterConfig` and attached to the scene graph as `NodeTypeParticleEmitter` nodes.

## EmitterConfig

```go
type EmitterConfig struct {
    MaxParticles int           // pool size (0 defaults to 128)
    EmitRate     float64       // particles per second
    Lifetime     willow.Range  // seconds each particle lives
    Speed        willow.Range  // initial speed in pixels/sec
    Angle        willow.Range  // emission direction in radians
    StartScale   willow.Range  // scale at birth
    EndScale     willow.Range  // scale at death
    StartAlpha   willow.Range  // alpha at birth
    EndAlpha     willow.Range  // alpha at death
    Gravity      willow.Vec2   // acceleration in pixels/sec^2
    StartColor   willow.Color  // tint at birth
    EndColor     willow.Color  // tint at death
    Region       willow.TextureRegion  // particle sprite
    BlendMode    willow.BlendMode
    WorldSpace   bool          // particles keep world position when emitted
}
```

The `Range` type has `Min` and `Max` fields  -  each particle picks a random value in that range at spawn.

## Creating a Particle Emitter

```go
cfg := willow.EmitterConfig{
    MaxParticles: 500,
    EmitRate:     100,
    Lifetime:     willow.Range{Min: 0.5, Max: 1.5},
    Speed:        willow.Range{Min: 50, Max: 150},
    Angle:        willow.Range{Min: 0, Max: 2 * math.Pi},
    StartScale:   willow.Range{Min: 0.5, Max: 1.0},
    EndScale:     willow.Range{Min: 0, Max: 0.2},
    StartAlpha:   willow.Range{Min: 0.8, Max: 1.0},
    EndAlpha:     willow.Range{Min: 0, Max: 0},
    Gravity:      willow.Vec2{X: 0, Y: 200},
    StartColor:   willow.Color{R: 1, G: 0.8, B: 0.2, A: 1},
    EndColor:     willow.Color{R: 1, G: 0.2, B: 0, A: 1},
    Region:       atlas.Region("particle"),
    BlendMode:    willow.BlendAdd,
}

emitter := willow.NewParticleEmitter("sparks", cfg)
emitter.X = 400
emitter.Y = 300
scene.Root().AddChild(emitter)
```

## Controlling the Emitter

Access the emitter API through `node.Emitter`:

```go
emitter.Emitter.Start()      // begin emitting
emitter.Emitter.Stop()       // stop emitting (existing particles finish)
emitter.Emitter.Reset()      // stop and kill all particles
emitter.Emitter.IsActive()   // check if emitting
emitter.Emitter.AliveCount() // current live particle count
```

## Live Tuning

Modify the config at runtime:

```go
cfg := emitter.Emitter.Config()
cfg.EmitRate = 200
cfg.Gravity.Y = 400
```

Changes take effect on the next `Update()`.

## World-Space vs. Local-Space

By default (`WorldSpace: false`), particles move with their emitter node. Set `WorldSpace: true` for particles that remain at their world-space emission point (e.g., a trail behind a moving character).

## Simulation Timing

Particle simulation runs during `scene.Update()`, not `Draw()`. This ensures consistent behavior regardless of frame rate.

## Solid-Color Particles

Use `TextureRegion{}` for untextured particles  -  they use the 1x1 WhitePixel and can be tinted with `StartColor`/`EndColor`:

```go
cfg := willow.EmitterConfig{
    Region:     willow.TextureRegion{},
    StartColor: willow.Color{R: 1, G: 1, B: 1, A: 1},
    EndColor:   willow.Color{R: 0, G: 0, B: 1, A: 0},
    // ...
}
```

## Next Steps

- [Mesh & Distortion](?page=meshes)  -  deformable vertex geometry and distortion grids
- [Ropes](?page=ropes)  -  textured strips along curved paths

## Related

- [Tweens & Animation](?page=tweens-and-animation)  -  animate emitter properties
- [Solid-Color Sprites](?page=solid-color-sprites)  -  WhitePixel used by untextured particles
- [Nodes](?page=nodes)  -  particle emitter is a node type
