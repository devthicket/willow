package willow

import (
	"testing"
)

func makeNode(name string) *Node {
	return NewContainer(name)
}

// --- Add / Remove ---

func TestNodeIndex_AddAndFindByName(t *testing.T) {
	idx := NewNodeIndex()
	n := makeNode("hero")
	idx.Add(n)

	if got := idx.FindByName("hero"); got != n {
		t.Fatalf("FindByName returned %v, want %v", got, n)
	}
}

func TestNodeIndex_AddWithTags(t *testing.T) {
	idx := NewNodeIndex()
	n := makeNode("goblin")
	idx.Add(n, "enemy", "melee")

	if got := idx.FindByTag("enemy"); len(got) != 1 || got[0] != n {
		t.Fatalf("FindByTag(enemy) = %v", got)
	}
	if got := idx.FindByTag("melee"); len(got) != 1 || got[0] != n {
		t.Fatalf("FindByTag(melee) = %v", got)
	}
}

func TestNodeIndex_AddDuplicateTagIgnored(t *testing.T) {
	idx := NewNodeIndex()
	n := makeNode("goblin")
	idx.Add(n, "enemy")
	idx.Add(n, "enemy") // duplicate

	if got := idx.FindByTag("enemy"); len(got) != 1 {
		t.Fatalf("duplicate tag created %d entries, want 1", len(got))
	}
}

func TestNodeIndex_AddDuplicateNodeNotReregistered(t *testing.T) {
	idx := NewNodeIndex()
	n := makeNode("hero")
	idx.Add(n)
	idx.Add(n, "player")

	// Should still only appear once in name map
	if got := idx.FindAllByName("hero"); len(got) != 1 {
		t.Fatalf("name map has %d entries, want 1", len(got))
	}
	// Tag should be added
	if got := idx.FindByTag("player"); len(got) != 1 {
		t.Fatalf("tag not added on second Add")
	}
}

func TestNodeIndex_RemoveEntirely(t *testing.T) {
	idx := NewNodeIndex()
	n := makeNode("goblin")
	idx.Add(n, "enemy", "melee")
	idx.Remove(n)

	if got := idx.FindByName("goblin"); got != nil {
		t.Fatal("node still findable by name after Remove")
	}
	if got := idx.FindByTag("enemy"); len(got) != 0 {
		t.Fatal("node still findable by tag after Remove")
	}
	if got := idx.CountByTag("melee"); got != 0 {
		t.Fatal("tag count nonzero after Remove")
	}
}

func TestNodeIndex_RemoveSpecificTags(t *testing.T) {
	idx := NewNodeIndex()
	n := makeNode("goblin")
	idx.Add(n, "enemy", "melee", "stunned")

	idx.Remove(n, "stunned")

	if got := idx.FindByTag("stunned"); len(got) != 0 {
		t.Fatal("stunned tag still present after Remove")
	}
	// Other tags and name remain
	if got := idx.FindByTag("enemy"); len(got) != 1 {
		t.Fatal("enemy tag lost")
	}
	if got := idx.FindByName("goblin"); got != n {
		t.Fatal("name lost")
	}
}

func TestNodeIndex_RemoveUnregisteredNodeNoOp(t *testing.T) {
	idx := NewNodeIndex()
	n := makeNode("ghost")
	idx.Remove(n) // should not panic
}

func TestNodeIndex_RemoveNonexistentTagNoOp(t *testing.T) {
	idx := NewNodeIndex()
	n := makeNode("goblin")
	idx.Add(n, "enemy")
	idx.Remove(n, "nonexistent") // should not panic
}

// --- FindByName ---

func TestNodeIndex_FindByNameMiss(t *testing.T) {
	idx := NewNodeIndex()
	if got := idx.FindByName("nobody"); got != nil {
		t.Fatal("expected nil for missing name")
	}
}

func TestNodeIndex_FindAllByName(t *testing.T) {
	idx := NewNodeIndex()
	a := makeNode("enemy_grunt")
	b := makeNode("enemy_grunt")
	c := makeNode("enemy_boss")
	idx.Add(a)
	idx.Add(b)
	idx.Add(c)

	got := idx.FindAllByName("enemy_grunt")
	if len(got) != 2 {
		t.Fatalf("FindAllByName returned %d, want 2", len(got))
	}
}

// --- FindByTag ---

func TestNodeIndex_FindByTagMiss(t *testing.T) {
	idx := NewNodeIndex()
	if got := idx.FindByTag("nothing"); len(got) != 0 {
		t.Fatalf("expected empty for missing tag, got %d", len(got))
	}
}

func TestNodeIndex_FindByTagMultipleNodes(t *testing.T) {
	idx := NewNodeIndex()
	a := makeNode("a")
	b := makeNode("b")
	c := makeNode("c")
	idx.Add(a, "enemy")
	idx.Add(b, "enemy")
	idx.Add(c, "ally")

	got := idx.FindByTag("enemy")
	if len(got) != 2 {
		t.Fatalf("FindByTag(enemy) returned %d, want 2", len(got))
	}
}

// --- FindByTags (intersection) ---

func TestNodeIndex_FindByTags(t *testing.T) {
	idx := NewNodeIndex()
	a := makeNode("a")
	b := makeNode("b")
	c := makeNode("c")
	idx.Add(a, "enemy", "flying")
	idx.Add(b, "enemy", "ground")
	idx.Add(c, "ally", "flying")

	got := idx.FindByTags("enemy", "flying")
	if len(got) != 1 || got[0] != a {
		t.Fatalf("FindByTags returned %v, want [a]", got)
	}
}

func TestNodeIndex_FindByTagsEmpty(t *testing.T) {
	idx := NewNodeIndex()
	if got := idx.FindByTags(); got != nil {
		t.Fatal("FindByTags with no args should return nil")
	}
}

func TestNodeIndex_FindByTagsNoMatch(t *testing.T) {
	idx := NewNodeIndex()
	n := makeNode("a")
	idx.Add(n, "enemy")

	got := idx.FindByTags("enemy", "flying")
	if len(got) != 0 {
		t.Fatal("expected no match")
	}
}

func TestNodeIndex_FindByTagsFirstTagMissing(t *testing.T) {
	idx := NewNodeIndex()
	n := makeNode("a")
	idx.Add(n, "enemy")

	got := idx.FindByTags("nonexistent", "enemy")
	if len(got) != 0 {
		t.Fatal("expected no match when first tag missing")
	}
}

// --- Wildcards ---

func TestNodeIndex_FindByNamePrefix(t *testing.T) {
	idx := NewNodeIndex()
	a := makeNode("enemy_01")
	b := makeNode("enemy_02")
	c := makeNode("ally_01")
	idx.Add(a)
	idx.Add(b)
	idx.Add(c)

	got := idx.FindAllByName("enemy%")
	if len(got) != 2 {
		t.Fatalf("prefix match returned %d, want 2", len(got))
	}
}

func TestNodeIndex_FindByNameSuffix(t *testing.T) {
	idx := NewNodeIndex()
	a := makeNode("final_boss")
	b := makeNode("mini_boss")
	c := makeNode("boss_helper")
	idx.Add(a)
	idx.Add(b)
	idx.Add(c)

	got := idx.FindAllByName("%boss")
	if len(got) != 2 {
		t.Fatalf("suffix match returned %d, want 2", len(got))
	}
	// Verify boss_helper is NOT included
	for _, n := range got {
		if n.Name == "boss_helper" {
			t.Fatal("suffix match incorrectly included boss_helper")
		}
	}
}

func TestNodeIndex_FindByNameContains(t *testing.T) {
	idx := NewNodeIndex()
	a := makeNode("big_enemy_01")
	b := makeNode("enemy")
	c := makeNode("ally_01")
	idx.Add(a)
	idx.Add(b)
	idx.Add(c)

	got := idx.FindAllByName("%enemy%")
	if len(got) != 2 {
		t.Fatalf("contains match returned %d, want 2", len(got))
	}
}

func TestNodeIndex_FindByNameWildcardFirstMatch(t *testing.T) {
	idx := NewNodeIndex()
	a := makeNode("enemy_01")
	b := makeNode("enemy_02")
	idx.Add(a)
	idx.Add(b)

	got := idx.FindByName("enemy%")
	if got == nil {
		t.Fatal("wildcard FindByName returned nil")
	}
	if got.Name != "enemy_01" && got.Name != "enemy_02" {
		t.Fatalf("unexpected node: %s", got.Name)
	}
}

func TestNodeIndex_FindByTagWildcard(t *testing.T) {
	idx := NewNodeIndex()
	a := makeNode("a")
	b := makeNode("b")
	c := makeNode("c")
	idx.Add(a, "team_red")
	idx.Add(b, "team_blue")
	idx.Add(c, "enemy")

	got := idx.FindByTag("team%")
	if len(got) != 2 {
		t.Fatalf("tag wildcard returned %d, want 2", len(got))
	}
}

func TestNodeIndex_FindByTagWildcardDeduplicates(t *testing.T) {
	idx := NewNodeIndex()
	n := makeNode("a")
	idx.Add(n, "team_red", "team_blue")

	got := idx.FindByTag("team%")
	if len(got) != 1 {
		t.Fatalf("wildcard should deduplicate, got %d", len(got))
	}
}

// --- CountByTag ---

func TestNodeIndex_CountByTag(t *testing.T) {
	idx := NewNodeIndex()
	idx.Add(makeNode("a"), "enemy")
	idx.Add(makeNode("b"), "enemy")
	idx.Add(makeNode("c"), "ally")

	if got := idx.CountByTag("enemy"); got != 2 {
		t.Fatalf("CountByTag = %d, want 2", got)
	}
	if got := idx.CountByTag("missing"); got != 0 {
		t.Fatalf("CountByTag(missing) = %d, want 0", got)
	}
}

// --- Each / EachByTag ---

func TestNodeIndex_Each(t *testing.T) {
	idx := NewNodeIndex()
	idx.Add(makeNode("a"))
	idx.Add(makeNode("b"))
	idx.Add(makeNode("c"))

	count := 0
	idx.Each(func(n *Node) bool {
		count++
		return true
	})
	if count != 3 {
		t.Fatalf("Each visited %d, want 3", count)
	}
}

func TestNodeIndex_EachEarlyStop(t *testing.T) {
	idx := NewNodeIndex()
	idx.Add(makeNode("a"))
	idx.Add(makeNode("b"))
	idx.Add(makeNode("c"))

	count := 0
	idx.Each(func(n *Node) bool {
		count++
		return false // stop after first
	})
	if count != 1 {
		t.Fatalf("Each did not stop early, visited %d", count)
	}
}

func TestNodeIndex_EachByTag(t *testing.T) {
	idx := NewNodeIndex()
	idx.Add(makeNode("a"), "enemy")
	idx.Add(makeNode("b"), "enemy")
	idx.Add(makeNode("c"), "ally")

	count := 0
	idx.EachByTag("enemy", func(n *Node) bool {
		count++
		return true
	})
	if count != 2 {
		t.Fatalf("EachByTag visited %d, want 2", count)
	}
}

func TestNodeIndex_EachByTagEarlyStop(t *testing.T) {
	idx := NewNodeIndex()
	idx.Add(makeNode("a"), "enemy")
	idx.Add(makeNode("b"), "enemy")

	count := 0
	idx.EachByTag("enemy", func(n *Node) bool {
		count++
		return false
	})
	if count != 1 {
		t.Fatalf("EachByTag did not stop early, visited %d", count)
	}
}

func TestNodeIndex_EachByTagMissing(t *testing.T) {
	idx := NewNodeIndex()
	count := 0
	idx.EachByTag("nothing", func(n *Node) bool {
		count++
		return true
	})
	if count != 0 {
		t.Fatal("EachByTag on missing tag should not iterate")
	}
}

// --- parsePattern / matchPattern ---

func TestParsePattern(t *testing.T) {
	tests := []struct {
		pattern string
		inner   string
		mode    wildMode
	}{
		{"exact", "exact", wildNone},
		{"foo%", "foo", wildPrefix},
		{"%bar", "bar", wildSuffix},
		{"%mid%", "mid", wildContains},
		{"%", "", wildSuffix},    // lone % = suffix with empty inner
		{"%%", "", wildContains}, // %% = contains empty (matches everything)
	}
	for _, tt := range tests {
		inner, mode := parsePattern(tt.pattern)
		if inner != tt.inner || mode != tt.mode {
			t.Errorf("parsePattern(%q) = (%q, %d), want (%q, %d)",
				tt.pattern, inner, mode, tt.inner, tt.mode)
		}
	}
}

func TestMatchPattern(t *testing.T) {
	tests := []struct {
		s     string
		inner string
		mode  wildMode
		want  bool
	}{
		// exact
		{"hello", "hello", wildNone, true},
		{"hello", "world", wildNone, false},
		// prefix
		{"enemy_01", "enemy", wildPrefix, true},
		{"ally_01", "enemy", wildPrefix, false},
		// suffix
		{"final_boss", "boss", wildSuffix, true},
		{"boss_helper", "boss", wildSuffix, false},
		// contains
		{"big_enemy_01", "enemy", wildContains, true},
		{"ally_01", "enemy", wildContains, false},
		{"enemy", "enemy", wildContains, true},
	}
	for _, tt := range tests {
		got := matchPattern(tt.s, tt.inner, tt.mode)
		if got != tt.want {
			t.Errorf("matchPattern(%q, %q, %d) = %v, want %v",
				tt.s, tt.inner, tt.mode, got, tt.want)
		}
	}
}

// --- Tag as state toggle ---

func TestNodeIndex_TagAsState(t *testing.T) {
	idx := NewNodeIndex()
	n := makeNode("enemy")
	idx.Add(n, "enemy")

	// Stun
	idx.Add(n, "stunned")
	if got := idx.CountByTag("stunned"); got != 1 {
		t.Fatal("stunned tag not applied")
	}

	// Unstun
	idx.Remove(n, "stunned")
	if got := idx.CountByTag("stunned"); got != 0 {
		t.Fatal("stunned tag not removed")
	}

	// Node still registered
	if got := idx.FindByTag("enemy"); len(got) != 1 {
		t.Fatal("node lost from enemy tag")
	}
}

// --- Multiple nodes same name ---

func TestNodeIndex_MultipleNodesSameName(t *testing.T) {
	idx := NewNodeIndex()
	a := makeNode("grunt")
	b := makeNode("grunt")
	idx.Add(a, "enemy")
	idx.Add(b, "enemy")

	// FindByName returns first
	got := idx.FindByName("grunt")
	if got == nil {
		t.Fatal("FindByName returned nil")
	}

	// FindAllByName returns both
	all := idx.FindAllByName("grunt")
	if len(all) != 2 {
		t.Fatalf("FindAllByName returned %d, want 2", len(all))
	}

	// Remove one, other remains
	idx.Remove(a)
	if got := idx.FindByName("grunt"); got != b {
		t.Fatal("wrong node remained after Remove")
	}
	if got := idx.CountByTag("enemy"); got != 1 {
		t.Fatal("tag count wrong after partial Remove")
	}
}
