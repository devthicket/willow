# Lighting

<p align="center">
  <img src="gif/lights.gif" alt="Lighting demo" width="400">
</p>

`LightLayer` provides a simple 2D lighting system. It renders to an offscreen texture and composites over the scene using multiply blending  -  areas without lights appear dark.

## Creating a Light Layer

```go
lightLayer := willow.NewLightLayer(800, 600, 0.7)
// width, height: should match your game viewport
// ambientAlpha: 0 = fully lit, 1 = fully dark
scene.Root().AddChild(lightLayer.Node())
```

The ambient alpha controls the base darkness level. At `0.7`, the scene is 70% dark where no lights shine.

## Adding Lights

```go
light := &willow.Light{
    X:         400,
    Y:         300,
    Radius:    150,
    Intensity: 1.0,
    Enabled:   true,
    Color:     willow.RGB(1, 0.9, 0.7),
}
lightLayer.AddLight(light)
```

### Light Fields

| Field | Type | Description |
|-------|------|-------------|
| `X`, `Y` | `float64` | Position in light layer's local space |
| `Radius` | `float64` | Light radius (drawn diameter = `Radius * 2`) |
| `Rotation` | `float64` | Radians (relevant for texture lights) |
| `Intensity` | `float64` | Brightness `[0,1]` |
| `Enabled` | `bool` | Toggle on/off |
| `Color` | `Color` | Tint color (white = neutral) |
| `TextureRegion` | `TextureRegion` | Use a sprite instead of feathered circle |
| `Target` | `*Node` | Follow this node's position |
| `OffsetX`, `OffsetY` | `float64` | Offset from target's pivot |

## Following Nodes

Attach a light to a node so it moves automatically:

```go
light.Target = playerNode
light.OffsetX = 0
light.OffsetY = -10  // slightly above the player
```

## Texture Lights

Use a custom texture instead of the default feathered circle:

```go
light.TextureRegion = atlas.Region("flashlight_cone")
light.Rotation = math.Pi / 4   // angled
```

Register atlas pages with the light layer:

```go
lightLayer.SetPages(atlas.Pages)
```

## Circle Light Cache

Pre-generate a circle light texture at a specific radius for reuse:

```go
lightLayer.SetCircleRadius(200)
```

## Redrawing

Call `Redraw()` every frame to update the light layer texture:

```go
scene.SetUpdateFunc(func() error {
    lightLayer.Redraw()
    return nil
})
```

## Managing Lights

```go
lightLayer.RemoveLight(light)
lightLayer.ClearLights()
lights := lightLayer.Lights()  // read-only slice
```

## Ambient Alpha

```go
lightLayer.SetAmbientAlpha(0.9)  // very dark
alpha := lightLayer.AmbientAlpha()
```

## Access the Underlying Texture

```go
rt := lightLayer.RenderTexture()
```

## Cleanup

```go
lightLayer.Dispose()
```

## Example

```go
scene := willow.NewScene()

// Game content
bg := willow.NewSprite("bg", atlas.Region("dungeon"))
scene.Root().AddChild(bg)

// Light layer on top
ll := willow.NewLightLayer(800, 600, 0.8)
scene.Root().AddChild(ll.Node())

// Player torch
torch := &willow.Light{
    Radius:    120,
    Intensity: 1.0,
    Enabled:   true,
    Color:     willow.RGB(1, 0.85, 0.6),
    Target:    playerNode,
}
ll.AddLight(torch)

// Ambient campfire
campfire := &willow.Light{
    X: 400, Y: 300,
    Radius:    80,
    Intensity: 0.8,
    Enabled:   true,
    Color:     willow.RGB(1, 0.6, 0.2),
}
ll.AddLight(campfire)

scene.SetUpdateFunc(func() error {
    ll.Redraw()
    return nil
})
```

## Next Steps

- [Clipping & Masks](?page=clipping-and-masks)  -  alpha-based masking and clipping
- [Post-Processing Filters](?page=post-processing-filters)  -  shader-based visual effects

## Related

- [Offscreen Rendering](?page=offscreen-rendering)  -  light layer uses a RenderTexture internally
- [Sprites](?page=sprites)  -  texture lights use TextureRegion
