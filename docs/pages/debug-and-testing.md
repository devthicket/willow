# Debug & Testing

Willow includes built-in tools for visual debugging, automated testing, and input simulation.

## Screenshot

Capture the rendered frame as a PNG file:

```go
scene.Screenshot("after-spawn")
// writes: screenshots/20260217_143052_after-spawn.png
```

This is the visual equivalent of `fmt.Println`  -  use it anywhere (`Update`, `Draw`, callbacks) to see what the viewport looks like at a specific point in time.

### Output Directory

```go
scene.ScreenshotDir = "debug_out"  // default: "screenshots"
```

### Behavior

- Multiple calls per frame are batched  -  pixels are read once, then written as separate PNGs
- Labels are sanitized for filenames (unsafe chars become `_`, empty becomes `unlabeled`)
- Errors go to stderr, never crash the application
- When not in use: one `len()` check per frame (zero cost)

## Input Injection

Inject synthetic pointer events for debugging, AI agents, or test harnesses. Events queue on the Scene and are consumed one per frame through the same input state machine as real mouse input.

```go
scene.InjectPress(x, y)              // queue a press at screen coords
scene.InjectMove(x, y)               // queue a move (button held)
scene.InjectRelease(x, y)            // queue a release
scene.InjectClick(x, y)              // press + release (2 frames)
scene.InjectDrag(fx, fy, tx, ty, 10) // full drag over 10 frames
```

### Important Notes

- Coordinates are **screen space** (what you see in screenshots), converted to world via the primary camera
- When the inject queue has events, real mouse input is skipped for that frame
- Touch and pinch continue normally alongside injected events

## TestRunner (JSON-Driven Visual Testing)

The `TestRunner` sequences injections and screenshots across frames from a JSON script:

```go
jsonData, _ := os.ReadFile("test_script.json")
runner, err := willow.LoadTestScript(jsonData)
scene.SetTestRunner(runner)

// In your game loop, check when done:
if runner.Done() {
    fmt.Println("Test script complete")
}
```

### JSON Format

```json
{
  "steps": [
    {"action": "screenshot", "label": "initial"},
    {"action": "click", "x": 100, "y": 200},
    {"action": "wait", "frames": 5},
    {"action": "screenshot", "label": "after-click"},
    {"action": "drag", "fromX": 100, "fromY": 200, "toX": 300, "toY": 400, "frames": 10},
    {"action": "screenshot", "label": "after-drag"}
  ]
}
```

### Actions

| Action | Fields | Description |
|--------|--------|-------------|
| `screenshot` | `label` | Capture current frame as PNG |
| `click` | `x`, `y` | Inject a press+release at coordinates |
| `drag` | `fromX`, `fromY`, `toX`, `toY`, `frames` | Inject a drag gesture over N frames |
| `wait` | `frames` | Wait N frames before next action |

### Use Cases

- **Regression testing**: capture screenshots at key states, compare across versions
- **AI agent testing**: script interactions without manual input
- **Bug reproduction**: record and replay interaction sequences

## Debug Mode

Enable debug overlays and additional logging:

```go
scene.SetDebugMode(true)
```

When debug mode is enabled, missing atlas regions log warnings and display magenta placeholder sprites.

## FPS Widget

Show an FPS/TPS counter (used automatically by `RunConfig.ShowFPS`):

```go
fps := willow.NewFPSWidget()
scene.Root().AddChild(fps)
// Updates every 0.5 seconds, renders at RenderLayer 255
```

## Next Steps

- [ECS Integration](?page=ecs-integration)  -  bridge scene graph interactions to an entity store
- [Sprite Packing (atlaspack)](?page=atlaspack)  -  pack PNG sprites into a texture atlas
- [Font Generation (fontgen)](?page=fontgen)  -  generate SDF font atlases from TTF/OTF files

## Related

- [Events & Callbacks](?page=events-and-callbacks)  -  the callback system that input injection feeds into
- [Scene](?page=scene)  -  `ScreenshotDir` field and `SetDebugMode`
- [Performance](?page=performance-overview)  -  benchmarking and optimization strategies
