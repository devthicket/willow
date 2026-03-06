# Getting Started

## Installation

```bash
go get github.com/phanxgames/willow@latest
```

Willow requires **Go 1.24+** and [Ebitengine](https://ebitengine.org/) v2.

## Two Paths: Run Wrapper vs Custom Game Loop

There are two ways to integrate Willow  -  both are first-class.

### Path 1: `willow.Run()`  -  Quick Setup

Call `willow.Run(scene, config)` and Willow handles the window, game loop, and calling `Update`/`Draw` for you. Best for getting started quickly or projects that don't need custom Ebitengine lifecycle control.

```go
package main

import (
    "log"

    "github.com/phanxgames/willow"
)

func main() {
    scene := willow.NewScene()

    sprite := willow.NewSprite("hero", willow.TextureRegion{})
    sprite.SetSize(40, 40)
    sprite.SetColor(willow.RGBA(0.3, 0.7, 1, 1))
    sprite.SetPosition(300, 220)
    scene.Root().AddChild(sprite)

    if err := willow.Run(scene, willow.RunConfig{
        Title:  "My Game",
        Width:  640,
        Height: 480,
    }); err != nil {
        log.Fatal(err)
    }
}
```

This creates a 640x480 window with a blue square. `NewSprite` with an empty `TextureRegion{}` creates a [solid-color sprite](?page=solid-color-sprites)  -  set `SetColor()` for the fill and `SetSize()` for the dimensions.

Use `SetUpdateFunc` to attach game logic without a custom struct:

```go
scene.SetUpdateFunc(func() error {
    sprite.SetX(sprite.X() + 1) // move right every frame
    return nil
})
```

### Path 2: Custom Game Loop  -  Full Control

Implement `ebiten.Game` yourself and call `scene.Update()` and `scene.Draw(screen)` directly. This gives you full control over the Ebitengine lifecycle  -  useful when you need custom `Layout` logic, multiple scenes, or integration with other Ebitengine features.

```go
package main

import (
    "github.com/hajimehoshi/ebiten/v2"
    "github.com/phanxgames/willow"
)

type Game struct {
    scene *willow.Scene
}

func (g *Game) Update() error {
    g.scene.Update()
    return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
    g.scene.Draw(screen)
}

func (g *Game) Layout(w, h int) (int, int) {
    return w, h
}

func main() {
    scene := willow.NewScene()
    // ... set up your scene ...

    ebiten.SetWindowSize(640, 480)
    ebiten.SetWindowTitle("Custom Loop")
    ebiten.RunGame(&Game{scene: scene})
}
```

## Next Steps

- [Scene](?page=scene)  -  scene configuration, batch modes, and RunConfig
- [Nodes](?page=nodes)  -  node types, tree manipulation, and visual properties
- [Sprites](?page=sprites)  -  display images and load atlases
- [Camera & Viewport](?page=camera-and-viewport)  -  set up viewport and scrolling

## Related

- [What is Willow?](?page=what-is-willow)  -  overview of features and design
- [Architecture](?page=architecture)  -  how the render pipeline works under the hood
- [Examples](?page=examples)  -  runnable demos with GIFs and source code
