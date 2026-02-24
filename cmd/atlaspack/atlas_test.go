package main

import (
	"encoding/json"
	"image"
	"image/color"
	"testing"
)

func TestComposeAtlas_SingleSprite(t *testing.T) {
	src := solidNRGBA(4, 4, color.NRGBA{R: 255, A: 255})
	sprites := []Sprite{{
		Name:      "red",
		Image:     src,
		PackX:     2,
		PackY:     3,
		OriginalW: 4,
		OriginalH: 4,
	}}
	atlas := ComposeAtlas(sprites, 16, 16)

	// Verify red pixels at expected location.
	for y := 3; y < 7; y++ {
		for x := 2; x < 6; x++ {
			c := atlas.NRGBAAt(x, y)
			if c.R != 255 || c.A != 255 {
				t.Errorf("pixel (%d,%d) = %v, want red", x, y, c)
			}
		}
	}
	// Verify a corner is transparent.
	c := atlas.NRGBAAt(0, 0)
	if c.A != 0 {
		t.Errorf("pixel (0,0) alpha = %d, want 0", c.A)
	}
}

func TestComposeAtlas_RotatedSprite(t *testing.T) {
	// 2x3 image: top-left red, bottom-right blue.
	src := image.NewNRGBA(image.Rect(0, 0, 2, 3))
	src.SetNRGBA(0, 0, color.NRGBA{R: 255, A: 255}) // red at (0,0)
	src.SetNRGBA(1, 2, color.NRGBA{B: 255, A: 255}) // blue at (1,2)

	sprites := []Sprite{{
		Name:      "test",
		Image:     src,
		PackX:     0,
		PackY:     0,
		Rotated:   true,
		OriginalW: 2,
		OriginalH: 3,
	}}
	// Rotated 90° CW: 2x3 → 3x2 in atlas.
	atlas := ComposeAtlas(sprites, 8, 8)

	// 90° CW rotation: (sx, sy) → (srcH-1-sy, sx)
	// Red (0,0) → (2, 0). Blue (1,2) → (0, 1).
	red := atlas.NRGBAAt(2, 0)
	if red.R != 255 || red.A != 255 {
		t.Errorf("expected red at (2,0), got %v", red)
	}
	blue := atlas.NRGBAAt(0, 1)
	if blue.B != 255 || blue.A != 255 {
		t.Errorf("expected blue at (0,1), got %v", blue)
	}
}

func TestGenerateJSON_BasicFormat(t *testing.T) {
	sprites := []Sprite{{
		Name:      "hero",
		Image:     image.NewNRGBA(image.Rect(0, 0, 64, 64)),
		PackX:     0,
		PackY:     0,
		OriginalW: 64,
		OriginalH: 64,
	}}
	data, err := GenerateJSON(sprites, 256, 256, "atlas.png")
	if err != nil {
		t.Fatalf("GenerateJSON: %v", err)
	}

	var result jsonAtlas
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if result.Meta.Image != "atlas.png" {
		t.Errorf("meta.image = %q, want atlas.png", result.Meta.Image)
	}
	if result.Meta.Size.W != 256 || result.Meta.Size.H != 256 {
		t.Errorf("meta.size = %dx%d, want 256x256", result.Meta.Size.W, result.Meta.Size.H)
	}

	f, ok := result.Frames["hero"]
	if !ok {
		t.Fatal("frame 'hero' not found")
	}
	if f.Frame.W != 64 || f.Frame.H != 64 {
		t.Errorf("frame size = %dx%d, want 64x64", f.Frame.W, f.Frame.H)
	}
	if f.Rotated {
		t.Error("should not be rotated")
	}
	if f.Trimmed {
		t.Error("should not be trimmed")
	}
	if f.SourceSize.W != 64 || f.SourceSize.H != 64 {
		t.Errorf("sourceSize = %dx%d, want 64x64", f.SourceSize.W, f.SourceSize.H)
	}
}

func TestGenerateJSON_Rotated(t *testing.T) {
	// Original 32x48, rotated in atlas → frame.w=48, frame.h=32
	sprites := []Sprite{{
		Name:      "tall",
		Image:     image.NewNRGBA(image.Rect(0, 0, 32, 48)),
		PackX:     10,
		PackY:     20,
		Rotated:   true,
		OriginalW: 32,
		OriginalH: 48,
	}}
	data, err := GenerateJSON(sprites, 512, 512, "out.png")
	if err != nil {
		t.Fatalf("GenerateJSON: %v", err)
	}

	var result jsonAtlas
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	f := result.Frames["tall"]
	if !f.Rotated {
		t.Error("should be rotated")
	}
	// Rotated: h becomes w, w becomes h.
	if f.Frame.W != 48 || f.Frame.H != 32 {
		t.Errorf("frame size = %dx%d, want 48x32", f.Frame.W, f.Frame.H)
	}
	if f.Frame.X != 10 || f.Frame.Y != 20 {
		t.Errorf("frame pos = (%d,%d), want (10,20)", f.Frame.X, f.Frame.Y)
	}
	if f.SourceSize.W != 32 || f.SourceSize.H != 48 {
		t.Errorf("sourceSize = %dx%d, want 32x48", f.SourceSize.W, f.SourceSize.H)
	}
}

func TestGenerateJSON_Trimmed(t *testing.T) {
	sprites := []Sprite{{
		Name:      "trimmed",
		Image:     image.NewNRGBA(image.Rect(0, 0, 60, 58)),
		PackX:     100,
		PackY:     50,
		Trimmed:   true,
		OffsetX:   2,
		OffsetY:   3,
		OriginalW: 64,
		OriginalH: 64,
	}}
	data, err := GenerateJSON(sprites, 1024, 1024, "atlas.png")
	if err != nil {
		t.Fatalf("GenerateJSON: %v", err)
	}

	var result jsonAtlas
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	f := result.Frames["trimmed"]
	if !f.Trimmed {
		t.Error("should be trimmed")
	}
	if f.Frame.X != 100 || f.Frame.Y != 50 {
		t.Errorf("frame pos = (%d,%d), want (100,50)", f.Frame.X, f.Frame.Y)
	}
	if f.Frame.W != 60 || f.Frame.H != 58 {
		t.Errorf("frame size = %dx%d, want 60x58", f.Frame.W, f.Frame.H)
	}
	if f.SpriteSourceSize.X != 2 || f.SpriteSourceSize.Y != 3 {
		t.Errorf("spriteSourceSize offset = (%d,%d), want (2,3)", f.SpriteSourceSize.X, f.SpriteSourceSize.Y)
	}
	if f.SourceSize.W != 64 || f.SourceSize.H != 64 {
		t.Errorf("sourceSize = %dx%d, want 64x64", f.SourceSize.W, f.SourceSize.H)
	}
}

func TestGenerateJSON_MatchesWillowSchema(t *testing.T) {
	// Produce JSON that mimics the test fixture in willow/atlas_test.go and
	// verify it can be round-tripped through the same structure.
	sprites := []Sprite{
		{
			Name:      "hero.png",
			Image:     image.NewNRGBA(image.Rect(0, 0, 64, 64)),
			PackX:     0,
			PackY:     0,
			OriginalW: 64,
			OriginalH: 64,
		},
		{
			Name:      "trimmed.png",
			Image:     image.NewNRGBA(image.Rect(0, 0, 60, 58)),
			PackX:     100,
			PackY:     50,
			Trimmed:   true,
			OffsetX:   2,
			OffsetY:   3,
			OriginalW: 64,
			OriginalH: 64,
		},
		{
			Name:      "rotated.png",
			Image:     image.NewNRGBA(image.Rect(0, 0, 32, 48)),
			PackX:     200,
			PackY:     0,
			Rotated:   true,
			OriginalW: 32,
			OriginalH: 48,
		},
	}
	data, err := GenerateJSON(sprites, 1024, 1024, "atlas.png")
	if err != nil {
		t.Fatalf("GenerateJSON: %v", err)
	}

	// Parse through a generic map to verify top-level structure.
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := raw["frames"]; !ok {
		t.Error("missing 'frames' key")
	}
	if _, ok := raw["meta"]; !ok {
		t.Error("missing 'meta' key")
	}

	// Parse and check rotated entry matches Willow's expected format.
	var atlas jsonAtlas
	if err := json.Unmarshal(data, &atlas); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	r := atlas.Frames["rotated.png"]
	if r.Frame.W != 48 || r.Frame.H != 32 {
		t.Errorf("rotated frame = %dx%d, want 48x32", r.Frame.W, r.Frame.H)
	}
	if r.SourceSize.W != 32 || r.SourceSize.H != 48 {
		t.Errorf("rotated sourceSize = %dx%d, want 32x48", r.SourceSize.W, r.SourceSize.H)
	}
}
