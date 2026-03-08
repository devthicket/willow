package types

import "testing"

func TestRGB(t *testing.T) {
	c := RGB(1, 0, 0)
	if c.R() != 1 || c.G() != 0 || c.B() != 0 || c.A() != 1 {
		t.Errorf("RGB(1,0,0) = %v, want r=1,g=0,b=0,a=1", c)
	}
}

func TestRGBA(t *testing.T) {
	c := RGBA(0.5, 0.5, 0.5, 0.5)
	if c.R() != 0.5 || c.A() != 0.5 {
		t.Errorf("RGBA(0.5,0.5,0.5,0.5) = %v", c)
	}
}

func TestColor_RGBA8(t *testing.T) {
	c := RGB(1, 0, 0)
	r, g, b, a := c.RGBA8()
	if r != 255 || g != 0 || b != 0 || a != 255 {
		t.Errorf("RGBA8() = (%d,%d,%d,%d), want (255,0,0,255)", r, g, b, a)
	}
}

func TestColorFromRGBA(t *testing.T) {
	c := ColorFromRGBA(255, 128, 0, 255)
	if c.R() != 1 || c.A() != 1 {
		t.Errorf("ColorFromRGBA(255,128,0,255).R()=%f, A()=%f", c.R(), c.A())
	}
	if c.G() < 0.5 || c.G() > 0.51 {
		t.Errorf("ColorFromRGBA(255,128,0,255).G()=%f, want ~0.502", c.G())
	}
}

func TestColorFromHSV(t *testing.T) {
	c := ColorFromHSV(0, 1, 1) // pure red
	if c.R() != 1 || c.G() != 0 || c.B() != 0 {
		t.Errorf("ColorFromHSV(0,1,1) = r=%f g=%f b=%f, want r=1,g=0,b=0", c.R(), c.G(), c.B())
	}
}

func TestRect_Contains(t *testing.T) {
	r := Rect{X: 10, Y: 10, Width: 100, Height: 50}
	if !r.Contains(50, 30) {
		t.Error("Contains(50,30) should be true")
	}
	if r.Contains(5, 5) {
		t.Error("Contains(5,5) should be false")
	}
	// Edge is inside
	if !r.Contains(10, 10) {
		t.Error("Contains(10,10) edge should be true")
	}
}

func TestRect_Intersects(t *testing.T) {
	a := Rect{X: 0, Y: 0, Width: 100, Height: 100}
	b := Rect{X: 50, Y: 50, Width: 100, Height: 100}
	if !a.Intersects(b) {
		t.Error("overlapping rects should intersect")
	}
	c := Rect{X: 200, Y: 200, Width: 10, Height: 10}
	if a.Intersects(c) {
		t.Error("non-overlapping rects should not intersect")
	}
}

func TestClamp01(t *testing.T) {
	if Clamp01(-0.5) != 0 {
		t.Error("Clamp01(-0.5) should be 0")
	}
	if Clamp01(1.5) != 1 {
		t.Error("Clamp01(1.5) should be 1")
	}
	if Clamp01(0.5) != 0.5 {
		t.Error("Clamp01(0.5) should be 0.5")
	}
}

func TestBlendMode_Values(t *testing.T) {
	// Verify enum ordering is stable
	if BlendNormal != 0 {
		t.Errorf("BlendNormal = %d, want 0", BlendNormal)
	}
	if BlendNone != 7 {
		t.Errorf("BlendNone = %d, want 7", BlendNone)
	}
}

func TestNodeType_Values(t *testing.T) {
	if NodeTypeContainer != 0 {
		t.Errorf("NodeTypeContainer = %d, want 0", NodeTypeContainer)
	}
	if NodeTypeText != 4 {
		t.Errorf("NodeTypeText = %d, want 4", NodeTypeText)
	}
}

func TestTextureRegion_ZeroValue(t *testing.T) {
	var r TextureRegion
	if r.Page != 0 || r.Width != 0 || r.Rotated {
		t.Error("zero TextureRegion should have all zero fields")
	}
}
