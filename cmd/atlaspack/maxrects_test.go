package main

import "testing"

func TestPack_SingleSprite(t *testing.T) {
	p := NewMaxRectsPacker(256, 256, 0, false)
	results, err := p.Pack([]PackInput{{ID: 0, Width: 64, Height: 64}})
	if err != nil {
		t.Fatalf("Pack: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	r := results[0]
	if r.X != 0 || r.Y != 0 {
		t.Errorf("position = (%d, %d), want (0, 0)", r.X, r.Y)
	}
	if r.Width != 64 || r.Height != 64 {
		t.Errorf("size = %dx%d, want 64x64", r.Width, r.Height)
	}
	if r.Rotated {
		t.Error("should not be rotated")
	}
}

func TestPack_TwoSprites(t *testing.T) {
	p := NewMaxRectsPacker(256, 256, 0, false)
	results, err := p.Pack([]PackInput{
		{ID: 0, Width: 128, Height: 128},
		{ID: 1, Width: 64, Height: 64},
	})
	if err != nil {
		t.Fatalf("Pack: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("got %d results, want 2", len(results))
	}
	// Verify no overlap.
	for i := 0; i < len(results); i++ {
		for j := i + 1; j < len(results); j++ {
			a, b := results[i], results[j]
			if a.X < b.X+b.Width && a.X+a.Width > b.X &&
				a.Y < b.Y+b.Height && a.Y+a.Height > b.Y {
				t.Errorf("sprites %d and %d overlap", a.ID, b.ID)
			}
		}
	}
}

func TestPack_RotationWins(t *testing.T) {
	// 130x64 atlas, sprite is 64x128. Without rotation it doesn't fit.
	// Rotated to 128x64, it fits.
	p := NewMaxRectsPacker(130, 70, 0, true)
	results, err := p.Pack([]PackInput{{ID: 0, Width: 64, Height: 128}})
	if err != nil {
		t.Fatalf("Pack: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	if !results[0].Rotated {
		t.Error("should be rotated")
	}
	if results[0].Width != 128 || results[0].Height != 64 {
		t.Errorf("placed size = %dx%d, want 128x64", results[0].Width, results[0].Height)
	}
}

func TestPack_NoFitError(t *testing.T) {
	p := NewMaxRectsPacker(32, 32, 0, false)
	_, err := p.Pack([]PackInput{{ID: 0, Width: 64, Height: 64}})
	if err == nil {
		t.Fatal("expected error for sprite that doesn't fit")
	}
}

func TestPack_Padding(t *testing.T) {
	// Two 32x32 sprites with 2px padding in a 68x34 atlas.
	// 32+2 = 34, two side by side = 68. Should barely fit.
	p := NewMaxRectsPacker(68, 34, 2, false)
	results, err := p.Pack([]PackInput{
		{ID: 0, Width: 32, Height: 32},
		{ID: 1, Width: 32, Height: 32},
	})
	if err != nil {
		t.Fatalf("Pack: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("got %d results, want 2", len(results))
	}
	// Verify sprites don't share coordinates.
	if results[0].X == results[1].X && results[0].Y == results[1].Y {
		t.Error("sprites placed at same position")
	}
}

func TestPack_ManySmall(t *testing.T) {
	// 16 sprites of 16x16 in a 64x64 atlas (4x4 grid).
	p := NewMaxRectsPacker(64, 64, 0, false)
	inputs := make([]PackInput, 16)
	for i := range inputs {
		inputs[i] = PackInput{ID: i, Width: 16, Height: 16}
	}
	results, err := p.Pack(inputs)
	if err != nil {
		t.Fatalf("Pack: %v", err)
	}
	if len(results) != 16 {
		t.Fatalf("got %d results, want 16", len(results))
	}
	// Verify no overlaps.
	for i := 0; i < len(results); i++ {
		for j := i + 1; j < len(results); j++ {
			a, b := results[i], results[j]
			if a.X < b.X+b.Width && a.X+a.Width > b.X &&
				a.Y < b.Y+b.Height && a.Y+a.Height > b.Y {
				t.Errorf("sprites %d and %d overlap", a.ID, b.ID)
			}
		}
	}
}

func TestPack_NoRotationFlag(t *testing.T) {
	// Sprite only fits if rotated, but rotation is disabled.
	p := NewMaxRectsPacker(130, 70, 0, false)
	_, err := p.Pack([]PackInput{{ID: 0, Width: 64, Height: 128}})
	if err == nil {
		t.Fatal("expected error when rotation is disabled and sprite doesn't fit")
	}
}

func TestSortByAreaDesc(t *testing.T) {
	s := []PackInput{
		{ID: 0, Width: 4, Height: 4},   // area 16
		{ID: 1, Width: 10, Height: 10}, // area 100
		{ID: 2, Width: 8, Height: 8},   // area 64
	}
	sortByAreaDesc(s)
	if s[0].ID != 1 || s[1].ID != 2 || s[2].ID != 0 {
		t.Errorf("sort order = [%d, %d, %d], want [1, 2, 0]", s[0].ID, s[1].ID, s[2].ID)
	}
}
