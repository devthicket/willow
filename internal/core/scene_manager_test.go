package core

import (
	"image/color"
	"testing"

	"github.com/devthicket/willow/internal/node"
	"github.com/devthicket/willow/internal/types"
)

func newTestScene(name string) *Scene {
	root := node.NewNode(name, types.NodeTypeContainer)
	return NewScene(root)
}

func TestSceneManagerCurrent(t *testing.T) {
	s := newTestScene("main")
	sm := NewSceneManager(s)
	if sm.Current() != s {
		t.Error("Current should return initial scene")
	}
}

func TestSceneManagerPush(t *testing.T) {
	s1 := newTestScene("s1")
	s2 := newTestScene("s2")
	sm := NewSceneManager(s1)
	sm.Push(s2)
	if sm.Current() != s2 {
		t.Error("Current should return pushed scene")
	}
}

func TestSceneManagerPop(t *testing.T) {
	s1 := newTestScene("s1")
	s2 := newTestScene("s2")
	sm := NewSceneManager(s1)
	sm.Push(s2)
	sm.Pop()
	if sm.Current() != s1 {
		t.Error("Current should return s1 after pop")
	}
}

func TestSceneManagerPopPanicsOnLast(t *testing.T) {
	s := newTestScene("main")
	sm := NewSceneManager(s)
	defer func() {
		if r := recover(); r == nil {
			t.Error("Pop on last scene should panic")
		}
	}()
	sm.Pop()
}

func TestSceneManagerReplace(t *testing.T) {
	s1 := newTestScene("s1")
	s2 := newTestScene("s2")
	sm := NewSceneManager(s1)
	sm.Replace(s2)
	if sm.Current() != s2 {
		t.Error("Current should return replaced scene")
	}
	if len(sm.stack) != 1 {
		t.Errorf("stack len = %d, want 1", len(sm.stack))
	}
}

func TestSceneManagerLifecycleHooks(t *testing.T) {
	var log []string
	s1 := newTestScene("s1")
	s1.SetOnEnter(func() { log = append(log, "s1-enter") })
	s1.SetOnExit(func() { log = append(log, "s1-exit") })
	s2 := newTestScene("s2")
	s2.SetOnEnter(func() { log = append(log, "s2-enter") })
	s2.SetOnExit(func() { log = append(log, "s2-exit") })

	sm := NewSceneManager(s1) // triggers s1-enter
	sm.Push(s2)               // triggers s1-exit, s2-enter

	expected := []string{"s1-enter", "s1-exit", "s2-enter"}
	if len(log) != len(expected) {
		t.Fatalf("log = %v, want %v", log, expected)
	}
	for i, e := range expected {
		if log[i] != e {
			t.Errorf("log[%d] = %q, want %q", i, log[i], e)
		}
	}
}

func TestFadeTransitionDuration(t *testing.T) {
	ft := NewFadeTransition(0.5, color.Black)
	if ft.Duration() != 0.5 {
		t.Errorf("Duration = %f, want 0.5", ft.Duration())
	}
}

func TestFadeTransitionDone(t *testing.T) {
	ft := NewFadeTransition(0.1, color.Black)
	if ft.Done() {
		t.Error("should not be done initially")
	}
	for i := 0; i < 20; i++ {
		ft.Update(0.01)
	}
	if !ft.Done() {
		t.Error("should be done after duration elapsed")
	}
}
