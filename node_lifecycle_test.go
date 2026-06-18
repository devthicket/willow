package willow

import (
	"strings"
	"testing"
)

// OnEnter fires when a node enters the live scene tree; OnReady fires once the
// first time; OnExit fires when it leaves. Re-adding fires OnEnter again but not
// OnReady.
func TestNodeLifecycleHooks(t *testing.T) {
	scene := NewScene()
	var ev []string
	child := NewContainer("child")
	child.OnEnter = func() { ev = append(ev, "enter") }
	child.OnReady = func() { ev = append(ev, "ready") }
	child.OnExit = func() { ev = append(ev, "exit") }

	scene.Root.AddChild(child)    // enter, ready
	scene.Root.RemoveChild(child) // exit
	scene.Root.AddChild(child)    // enter (no second ready)

	got := strings.Join(ev, ",")
	want := "enter,ready,exit,enter"
	if got != want {
		t.Fatalf("hook sequence = %q, want %q", got, want)
	}
}

// Adding/removing a subtree fires hooks for every node: enter top-down (parent
// before child), exit bottom-up (child before parent).
func TestNodeLifecycleSubtreeOrder(t *testing.T) {
	scene := NewScene()
	var ev []string
	parent := NewContainer("parent")
	child := NewContainer("child")
	parent.AddChild(child) // parent not in scene yet -> no hooks fire

	parent.OnEnter = func() { ev = append(ev, "parent-enter") }
	child.OnEnter = func() { ev = append(ev, "child-enter") }
	parent.OnExit = func() { ev = append(ev, "parent-exit") }
	child.OnExit = func() { ev = append(ev, "child-exit") }

	scene.Root.AddChild(parent)    // enter: parent then child
	scene.Root.RemoveChild(parent) // exit: child then parent

	got := strings.Join(ev, ",")
	want := "parent-enter,child-enter,child-exit,parent-exit"
	if got != want {
		t.Fatalf("subtree hook order = %q, want %q", got, want)
	}
}

// A node built and attached under a parent that is not yet in the scene gets no
// hooks until the whole subtree is added to the live tree.
func TestNodeLifecycleNoFireBeforeScene(t *testing.T) {
	var fired bool
	parent := NewContainer("parent")
	child := NewContainer("child")
	child.OnEnter = func() { fired = true }
	parent.AddChild(child)
	if fired {
		t.Fatal("OnEnter fired while detached from any scene")
	}
}
