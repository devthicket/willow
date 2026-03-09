package willow

import "testing"

func TestNewScene(t *testing.T) {
	s := NewScene()
	if s.Root == nil {
		t.Fatal("root should not be nil")
	}
	if s.Root.Name != "root" {
		t.Errorf("root.Name = %q, want %q", s.Root.Name, "root")
	}
	if s.Root.Type != NodeTypeContainer {
		t.Errorf("root.Type = %d, want NodeTypeContainer", s.Root.Type)
	}
}

func TestSceneRoot(t *testing.T) {
	s := NewScene()
	r := Root(s)
	if r == nil {
		t.Error("Root(s) should return the root node")
	}
	if r != s.Root {
		t.Error("Root(s) should return the internal root node")
	}
}

func TestSceneSetEntityStore(t *testing.T) {
	s := NewScene()
	s.SetEntityStore(nil) // should not panic
}

func TestSceneSetDebugMode(t *testing.T) {
	s := NewScene()
	s.SetDebugMode(true)
	if !s.Debug {
		t.Error("debug should be true")
	}
	s.SetDebugMode(false)
	if s.Debug {
		t.Error("debug should be false")
	}
}

func TestSceneRegisterPage(t *testing.T) {
	resetAtlasManager()
	defer resetAtlasManager()

	s := NewScene()
	s.RegisterPage(0, nil)
	s.RegisterPage(2, nil)
	am := atlasManager()
	if am.PageCount() != 3 {
		t.Errorf("PageCount = %d, want 3", am.PageCount())
	}
}
