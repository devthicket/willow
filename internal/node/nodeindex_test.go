package node

import (
	"testing"

	"github.com/phanxgames/willow/internal/types"
)

func TestNodeIndex_AddAndFind(t *testing.T) {
	idx := NewNodeIndex()
	n := NewNode("hero", types.NodeTypeSprite)
	idx.Add(n, "player", "controllable")

	found := idx.FindByName("hero")
	if found != n {
		t.Error("FindByName should find hero")
	}

	tagged := idx.FindByTag("player")
	if len(tagged) != 1 || tagged[0] != n {
		t.Error("FindByTag should find hero by 'player' tag")
	}
}

func TestNodeIndex_Remove(t *testing.T) {
	idx := NewNodeIndex()
	n := NewNode("hero", types.NodeTypeSprite)
	idx.Add(n, "player")
	idx.Remove(n)

	found := idx.FindByName("hero")
	if found != nil {
		t.Error("FindByName should return nil after Remove")
	}
}

func TestNodeIndex_RemoveTag(t *testing.T) {
	idx := NewNodeIndex()
	n := NewNode("hero", types.NodeTypeSprite)
	idx.Add(n, "player", "controllable")
	idx.Remove(n, "player")

	tagged := idx.FindByTag("player")
	if len(tagged) != 0 {
		t.Error("FindByTag('player') should return empty after tag remove")
	}
	tagged = idx.FindByTag("controllable")
	if len(tagged) != 1 {
		t.Error("FindByTag('controllable') should still find node")
	}
}

func TestNodeIndex_WildcardTag(t *testing.T) {
	idx := NewNodeIndex()
	a := NewNode("enemy1", types.NodeTypeSprite)
	b := NewNode("enemy2", types.NodeTypeSprite)
	c := NewNode("friend", types.NodeTypeSprite)
	idx.Add(a, "enemy_ground")
	idx.Add(b, "enemy_air")
	idx.Add(c, "friend")

	tagged := idx.FindByTag("enemy%")
	if len(tagged) != 2 {
		t.Errorf("FindByTag('enemy%%') = %d nodes, want 2", len(tagged))
	}
}

func TestNodeIndex_FindByTags(t *testing.T) {
	idx := NewNodeIndex()
	n := NewNode("hero", types.NodeTypeSprite)
	idx.Add(n, "player", "controllable")

	found := idx.FindByTags("player", "controllable")
	if len(found) != 1 {
		t.Errorf("FindByTags = %d, want 1", len(found))
	}
	found = idx.FindByTags("player", "npc")
	if len(found) != 0 {
		t.Errorf("FindByTags('player', 'npc') = %d, want 0", len(found))
	}
}

func TestNodeIndex_CountByTag(t *testing.T) {
	idx := NewNodeIndex()
	for i := 0; i < 5; i++ {
		n := NewNode("enemy", types.NodeTypeSprite)
		idx.Add(n, "hostile")
	}
	if idx.CountByTag("hostile") != 5 {
		t.Errorf("CountByTag = %d, want 5", idx.CountByTag("hostile"))
	}
}
