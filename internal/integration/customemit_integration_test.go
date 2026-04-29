package integration

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	. "github.com/devthicket/willow"
)

// TestCustomEmit_ReplacesDefault verifies that when a handler does NOT call
// EmitDefault, the host node's default render command is suppressed and only
// the user-appended commands remain.
func TestCustomEmit_ReplacesDefault(t *testing.T) {
	s := NewScene()
	host := NewSprite("host", TextureRegion{Width: 32, Height: 32})
	s.Root.AddChild(host)

	verts := []ebiten.Vertex{
		{DstX: 0, DstY: 0}, {DstX: 10, DstY: 0}, {DstX: 0, DstY: 10},
	}
	inds := []uint16{0, 1, 2}

	called := 0
	SetCustomEmit(host, func(e *Emitter, treeOrder *int) {
		called++
		e.AppendTriangles(TrianglesEmit{Verts: verts, Inds: inds, Image: WhitePixel}, treeOrder)
	})

	traverseScene(s)

	if called != 1 {
		t.Fatalf("handler called %d times, want 1", called)
	}
	if len(s.Pipeline.Commands) != 1 {
		t.Fatalf("commands = %d, want 1 (default sprite suppressed)", len(s.Pipeline.Commands))
	}
	if s.Pipeline.Commands[0].Type != CommandMesh {
		t.Errorf("Type = %d, want CommandMesh", s.Pipeline.Commands[0].Type)
	}
}

// TestCustomEmit_EmitDefaultAppends verifies EmitDefault triggers the host's
// normal render emit alongside any custom-appended commands.
func TestCustomEmit_EmitDefaultAppends(t *testing.T) {
	s := NewScene()
	host := NewSprite("host", TextureRegion{Width: 32, Height: 32})
	s.Root.AddChild(host)

	verts := []ebiten.Vertex{
		{DstX: 0, DstY: 0}, {DstX: 10, DstY: 0}, {DstX: 0, DstY: 10},
	}
	inds := []uint16{0, 1, 2}

	SetCustomEmit(host, func(e *Emitter, treeOrder *int) {
		e.EmitDefault(treeOrder)
		e.AppendTriangles(TrianglesEmit{Verts: verts, Inds: inds, Image: WhitePixel}, treeOrder)
	})

	traverseScene(s)

	if len(s.Pipeline.Commands) != 2 {
		t.Fatalf("commands = %d, want 2 (sprite + mesh)", len(s.Pipeline.Commands))
	}
	if s.Pipeline.Commands[0].Type != CommandSprite {
		t.Errorf("cmd[0] Type = %d, want CommandSprite", s.Pipeline.Commands[0].Type)
	}
	if s.Pipeline.Commands[1].Type != CommandMesh {
		t.Errorf("cmd[1] Type = %d, want CommandMesh", s.Pipeline.Commands[1].Type)
	}
	if s.Pipeline.Commands[1].TreeOrder <= s.Pipeline.Commands[0].TreeOrder {
		t.Errorf("TreeOrder not monotonic: cmd[0]=%d cmd[1]=%d",
			s.Pipeline.Commands[0].TreeOrder, s.Pipeline.Commands[1].TreeOrder)
	}
}

// TestSetCustomEmit_NilClearsHandler verifies passing fn==nil removes a
// previously installed handler so default emit resumes.
func TestSetCustomEmit_NilClearsHandler(t *testing.T) {
	s := NewScene()
	host := NewSprite("host", TextureRegion{Width: 32, Height: 32})
	s.Root.AddChild(host)

	SetCustomEmit(host, func(e *Emitter, treeOrder *int) {
		// Replace the default with nothing.
	})
	traverseScene(s)
	if len(s.Pipeline.Commands) != 0 {
		t.Fatalf("with handler: commands = %d, want 0", len(s.Pipeline.Commands))
	}

	SetCustomEmit(host, nil)
	traverseScene(s)
	if len(s.Pipeline.Commands) != 1 {
		t.Fatalf("after clear: commands = %d, want 1 (default sprite restored)", len(s.Pipeline.Commands))
	}
}

// TestCustomEmit_IsBuildingCache_TrueInCacheBuild guards against the regression
// where Emitter.IsBuildingCache always returned false inside a CacheAsTree
// build. The custom-emit child runs underneath a cached container; on the
// build pass IsBuildingCache must be true so handlers can skip animated
// content. On a normal pass it must be false.
func TestCustomEmit_IsBuildingCache_TrueInCacheBuild(t *testing.T) {
	s := NewScene()
	container := NewContainer("c")
	host := NewSprite("host", TextureRegion{Width: 32, Height: 32})
	container.AddChild(host)
	s.Root.AddChild(container)

	var seen []bool
	SetCustomEmit(host, func(e *Emitter, treeOrder *int) {
		seen = append(seen, e.IsBuildingCache())
		e.EmitDefault(treeOrder)
	})

	// First traverse with cache enabled triggers the build pass.
	container.SetCacheAsTree(true, CacheTreeManual)
	traverseScene(s)
	// Second traverse replays from cache; the handler should NOT be invoked
	// for cached children (cache replay doesn't re-run children's emit).
	traverseScene(s)

	if len(seen) == 0 {
		t.Fatal("handler never called")
	}
	if !seen[0] {
		t.Errorf("first call: IsBuildingCache = false, want true (inside cache build)")
	}
	for i, b := range seen[1:] {
		if b {
			t.Errorf("call[%d]: IsBuildingCache = true outside a build pass", i+1)
		}
	}
}

// TestCustomEmit_AppendTrianglesStampsNodeIDInCacheBuild verifies that when a
// handler appends triangles inside a CacheAsTree build pass, the resulting
// command has EmittingNodeID set to the host node's ID — so the cache
// invalidator can attribute the command back to its source node.
func TestCustomEmit_AppendTrianglesStampsNodeIDInCacheBuild(t *testing.T) {
	s := NewScene()
	container := NewContainer("c")
	host := NewSprite("host", TextureRegion{Width: 32, Height: 32})
	container.AddChild(host)
	s.Root.AddChild(container)

	verts := []ebiten.Vertex{
		{DstX: 0, DstY: 0}, {DstX: 10, DstY: 0}, {DstX: 0, DstY: 10},
	}
	inds := []uint16{0, 1, 2}

	SetCustomEmit(host, func(e *Emitter, treeOrder *int) {
		e.AppendTriangles(TrianglesEmit{Verts: verts, Inds: inds, Image: WhitePixel}, treeOrder)
	})

	container.SetCacheAsTree(true, CacheTreeManual)
	traverseScene(s) // build pass

	var found *RenderCommand
	for i := range s.Pipeline.Commands {
		if s.Pipeline.Commands[i].Type == CommandMesh {
			found = &s.Pipeline.Commands[i]
			break
		}
	}
	if found == nil {
		t.Fatal("no CommandMesh emitted")
	}
	if found.EmittingNodeID != host.ID {
		t.Errorf("EmittingNodeID = %d, want %d (host.ID)", found.EmittingNodeID, host.ID)
	}
}

// TestCustomEmit_AppendTrianglesNoNodeIDOutsideBuild verifies the inverse:
// commands appended outside a cache build do NOT get EmittingNodeID stamped.
// (Only cache-build emissions need attribution; stamping otherwise would be
// noise.)
func TestCustomEmit_AppendTrianglesNoNodeIDOutsideBuild(t *testing.T) {
	s := NewScene()
	host := NewSprite("host", TextureRegion{Width: 32, Height: 32})
	s.Root.AddChild(host)

	verts := []ebiten.Vertex{
		{DstX: 0, DstY: 0}, {DstX: 10, DstY: 0}, {DstX: 0, DstY: 10},
	}
	inds := []uint16{0, 1, 2}

	SetCustomEmit(host, func(e *Emitter, treeOrder *int) {
		e.AppendTriangles(TrianglesEmit{Verts: verts, Inds: inds, Image: WhitePixel}, treeOrder)
	})

	traverseScene(s)

	if len(s.Pipeline.Commands) != 1 {
		t.Fatalf("commands = %d, want 1", len(s.Pipeline.Commands))
	}
	if got := s.Pipeline.Commands[0].EmittingNodeID; got != 0 {
		t.Errorf("EmittingNodeID = %d, want 0 outside cache build", got)
	}
}
