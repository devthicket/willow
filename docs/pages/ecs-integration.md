# ECS Integration

Willow bridges its scene graph with Entity Component Systems (ECS) through the `EntityStore` interface and `InteractionEvent` type.

## EntityStore Interface

```go
type EntityStore interface {
    EmitEvent(event InteractionEvent)
}
```

Register an entity store with the scene:

```go
scene.SetEntityStore(store)
```

When a node with a non-zero `EntityID` receives an interaction (click, drag, etc.), the scene forwards an `InteractionEvent` to the entity store.

## InteractionEvent

```go
type InteractionEvent struct {
    Type         EventType
    EntityID     uint32
    GlobalX      float64
    GlobalY      float64
    LocalX       float64
    LocalY       float64
    Button       MouseButton
    Modifiers    KeyModifiers
    // Drag fields (EventDragStart, EventDrag, EventDragEnd)
    StartX, StartY             float64
    DeltaX, DeltaY             float64
    ScreenDeltaX, ScreenDeltaY float64
    // Pinch fields (EventPinch)
    Scale, ScaleDelta   float64
    Rotation, RotDelta  float64
}
```

## Linking Nodes to Entities

Set `EntityID` on any interactable node:

```go
sprite := willow.NewSprite("enemy", region)
sprite.EntityID = entityID  // your ECS entity identifier
sprite.Interactable = true
sprite.HitShape = willow.HitRect{X: 0, Y: 0, Width: 32, Height: 32}
scene.Root().AddChild(sprite)
```

When this sprite is clicked, dragged, etc., an `InteractionEvent` with the matching `EntityID` is emitted through the entity store.

## Donburi Adapter

Willow ships a ready-made adapter for [Donburi](https://github.com/yohamta/donburi) ECS in the `willow/ecs` submodule:

```go
import (
    "github.com/phanxgames/willow/ecs"
    "github.com/yohamta/donburi"
    "github.com/yohamta/donburi/features/events"
)

world := donburi.NewWorld()
store := ecs.NewDonburiStore(world)
scene.SetEntityStore(store)
```

### Consuming Events

Subscribe to interaction events using Donburi's event system:

```go
events.Subscribe(world, ecs.InteractionEventType,
    func(w donburi.World, event willow.InteractionEvent) {
        switch event.Type {
        case willow.EventClick:
            fmt.Printf("Entity %d clicked at (%.0f, %.0f)\n",
                event.EntityID, event.GlobalX, event.GlobalY)
        case willow.EventDrag:
            fmt.Printf("Entity %d dragged by (%.1f, %.1f)\n",
                event.EntityID, event.DeltaX, event.DeltaY)
        }
    },
)

// Process events each frame
events.ProcessAllEvents(world)
```

### Installation

```bash
go get github.com/phanxgames/willow/ecs@latest
```

## Custom EntityStore

Implement the `EntityStore` interface for any ECS framework:

```go
type MyStore struct {
    events []willow.InteractionEvent
}

func (s *MyStore) EmitEvent(event willow.InteractionEvent) {
    s.events = append(s.events, event)
}

// Process events in your game loop
func (s *MyStore) Process() {
    for _, e := range s.events {
        // handle events
    }
    s.events = s.events[:0]
}
```

## Related

- [Events & Callbacks](?page=events-and-callbacks)  -  the callback system and event types that feed into ECS
- [Input & Hit Testing](?page=input-hit-testing-and-gestures)  -  making nodes interactive with `Interactable` and `HitShape`
- [Nodes](?page=nodes)  -  `EntityID` and `UserData` fields on nodes
