package ecs

import (
	"github.com/phanxgames/willow"
	"testing"

	"github.com/yohamta/donburi"
	"github.com/yohamta/donburi/features/events"
)

func TestNewDonburiStore(t *testing.T) {
	world := donburi.NewWorld()
	store := NewDonburiStore(world)
	if store == nil {
		t.Fatal("NewDonburiStore returned nil")
	}
}

func TestDonburiStore_EmitEvent(t *testing.T) {
	world := donburi.NewWorld()
	store := NewDonburiStore(world)

	var received []willow.InteractionEvent
	InteractionEventType.Subscribe(world, func(w donburi.World, e willow.InteractionEvent) {
		received = append(received, e)
	})

	store.EmitEvent(willow.InteractionEvent{
		Type:     willow.EventPointerDown,
		EntityID: 42,
		GlobalX:  100,
		GlobalY:  200,
		Button:   willow.MouseButtonLeft,
	})

	store.EmitEvent(willow.InteractionEvent{
		Type:       willow.EventPinch,
		Scale:      2.0,
		ScaleDelta: 0.5,
	})

	// Events are queued  -  process them.
	InteractionEventType.ProcessEvents(world)

	if len(received) != 2 {
		t.Fatalf("expected 2 events, got %d", len(received))
	}

	e0 := received[0]
	if e0.Type != willow.EventPointerDown || e0.EntityID != 42 {
		t.Errorf("event 0: %+v", e0)
	}
	if e0.GlobalX != 100 || e0.GlobalY != 200 {
		t.Errorf("event 0 position: (%v,%v)", e0.GlobalX, e0.GlobalY)
	}

	e1 := received[1]
	if e1.Type != willow.EventPinch || e1.Scale != 2.0 {
		t.Errorf("event 1: %+v", e1)
	}
}

func TestDonburiStore_ImplementsEntityStore(t *testing.T) {
	world := donburi.NewWorld()
	var store willow.EntityStore = NewDonburiStore(world)
	_ = store // compile-time interface check
}

func TestDonburiStore_MultipleSubscribers(t *testing.T) {
	world := donburi.NewWorld()
	store := NewDonburiStore(world)

	var count1, count2 int
	InteractionEventType.Subscribe(world, func(w donburi.World, e willow.InteractionEvent) {
		count1++
	})
	InteractionEventType.Subscribe(world, func(w donburi.World, e willow.InteractionEvent) {
		count2++
	})

	store.EmitEvent(willow.InteractionEvent{Type: willow.EventClick})
	events.ProcessAllEvents(world)

	if count1 != 1 || count2 != 1 {
		t.Errorf("expected both subscribers called once, got %d and %d", count1, count2)
	}
}
