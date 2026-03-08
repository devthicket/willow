package willow

import (
	"math"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/tanema/gween/ease"
)

func approxEqual(a, b, eps float64) bool {
	return math.Abs(a-b) < eps
}

func TestCameraDefaults(t *testing.T) {
	cam := newCamera(Rect{X: 0, Y: 0, Width: 800, Height: 600})
	if cam.Zoom != 1.0 {
		t.Errorf("Zoom = %f, want 1.0", cam.Zoom)
	}
	if !cam.CullEnabled {
		t.Error("CullEnabled = false, want true")
	}
	if cam.Viewport.Width != 800 || cam.Viewport.Height != 600 {
		t.Errorf("Viewport = %v, want 800x600", cam.Viewport)
	}
}

func TestCameraIdentityViewMatrix(t *testing.T) {
	cam := newCamera(Rect{X: 0, Y: 0, Width: 800, Height: 600})
	vm := cam.computeViewMatrix()
	// At (0,0), zoom 1, no rotation:
	// viewMatrix should translate to viewport center (400, 300)
	sx, sy := transformPoint(vm, 0, 0)
	if !approxEqual(sx, 400, epsilon) || !approxEqual(sy, 300, epsilon) {
		t.Errorf("WorldToScreen(0,0) = (%f,%f), want (400,300)", sx, sy)
	}
}

func TestCameraTranslation(t *testing.T) {
	cam := newCamera(Rect{X: 0, Y: 0, Width: 800, Height: 600})
	cam.X = 100
	cam.Y = 50
	cam.dirty = true
	sx, sy := cam.WorldToScreen(100, 50)
	// Camera at (100,50) looking at (100,50) should map to viewport center
	if !approxEqual(sx, 400, epsilon) || !approxEqual(sy, 300, epsilon) {
		t.Errorf("WorldToScreen(100,50) with cam at (100,50) = (%f,%f), want (400,300)", sx, sy)
	}
}

func TestCameraZoom(t *testing.T) {
	cam := newCamera(Rect{X: 0, Y: 0, Width: 800, Height: 600})
	cam.Zoom = 2.0
	cam.dirty = true

	// At zoom 2, a point 1 unit from camera center should appear 2 pixels away
	sx1, _ := cam.WorldToScreen(1, 0)
	sx0, _ := cam.WorldToScreen(0, 0)
	screenDist := sx1 - sx0
	if !approxEqual(screenDist, 2.0, epsilon) {
		t.Errorf("zoom 2x: 1 world unit = %f screen pixels, want 2.0", screenDist)
	}
}

func TestCameraRotation90(t *testing.T) {
	cam := newCamera(Rect{X: 0, Y: 0, Width: 800, Height: 600})
	cam.Rotation = math.Pi / 2 // 90 degrees
	cam.dirty = true

	// World point (1, 0) with 90° camera rotation
	sx, sy := cam.WorldToScreen(1, 0)
	// Rotate(-π/2) maps (1,0)→(0,-1), then translate to viewport center (400,300)
	// Result: (400, 299)
	cx, cy := 400.0, 300.0
	if !approxEqual(sx, cx, epsilon) || !approxEqual(sy, cy-1, epsilon) {
		t.Errorf("90° rotation: WorldToScreen(1,0) = (%f,%f), want (%f,%f)", sx, sy, cx, cy-1)
	}
}

func TestScreenToWorldRoundtrip(t *testing.T) {
	cam := newCamera(Rect{X: 0, Y: 0, Width: 800, Height: 600})
	cam.X = 42
	cam.Y = -17
	cam.Zoom = 1.5
	cam.Rotation = 0.3
	cam.dirty = true

	origWX, origWY := 123.0, -456.0
	sx, sy := cam.WorldToScreen(origWX, origWY)
	wx, wy := cam.ScreenToWorld(sx, sy)

	if !approxEqual(wx, origWX, 1e-6) || !approxEqual(wy, origWY, 1e-6) {
		t.Errorf("roundtrip: got (%f,%f), want (%f,%f)", wx, wy, origWX, origWY)
	}
}

func TestVisibleBounds_Zoom1(t *testing.T) {
	cam := newCamera(Rect{X: 0, Y: 0, Width: 800, Height: 600})
	cam.X = 400
	cam.Y = 300
	cam.dirty = true
	bounds := cam.VisibleBounds()
	// Camera centered at (400,300), viewport 800x600, zoom 1: visible is (0,0)-(800,600)
	if !approxEqual(bounds.X, 0, 1e-6) || !approxEqual(bounds.Y, 0, 1e-6) {
		t.Errorf("VisibleBounds origin = (%f,%f), want (0,0)", bounds.X, bounds.Y)
	}
	if !approxEqual(bounds.Width, 800, 1e-6) || !approxEqual(bounds.Height, 600, 1e-6) {
		t.Errorf("VisibleBounds size = (%f,%f), want (800,600)", bounds.Width, bounds.Height)
	}
}

func TestVisibleBounds_Zoom2(t *testing.T) {
	cam := newCamera(Rect{X: 0, Y: 0, Width: 800, Height: 600})
	cam.X = 400
	cam.Y = 300
	cam.Zoom = 2.0
	cam.dirty = true
	bounds := cam.VisibleBounds()
	// Zoom 2 halves the visible area
	if !approxEqual(bounds.Width, 400, 1e-6) || !approxEqual(bounds.Height, 300, 1e-6) {
		t.Errorf("VisibleBounds at zoom 2 size = (%f,%f), want (400,300)", bounds.Width, bounds.Height)
	}
}

func TestCameraFollow(t *testing.T) {
	scene := NewScene()
	cam := scene.NewCamera(Rect{X: 0, Y: 0, Width: 800, Height: 600})

	target := NewSprite("target", TextureRegion{})
	target.X_ = 200
	target.Y_ = 150
	target.TransformDirty = false
	target.WorldTransform = [6]float64{1, 0, 0, 1, 200, 150}
	scene.Root().AddChild(target)

	cam.Follow(target, 0, 0, 1.0) // lerp=1 snaps immediately

	cam.update(1.0 / 60.0)
	if !approxEqual(cam.X, 200, epsilon) || !approxEqual(cam.Y, 150, epsilon) {
		t.Errorf("after follow snap: cam = (%f,%f), want (200,150)", cam.X, cam.Y)
	}
}

func TestCameraFollowLerp(t *testing.T) {
	cam := newCamera(Rect{X: 0, Y: 0, Width: 800, Height: 600})
	target := NewSprite("target", TextureRegion{})
	target.WorldTransform = [6]float64{1, 0, 0, 1, 100, 0}

	cam.Follow(target, 0, 0, 0.5)
	cam.update(1.0 / 60.0)
	// Should move halfway from 0 to 100
	if !approxEqual(cam.X, 50, epsilon) {
		t.Errorf("after lerp 0.5: cam.X = %f, want 50", cam.X)
	}
}

func TestCameraFollowWithOffset(t *testing.T) {
	cam := newCamera(Rect{X: 0, Y: 0, Width: 800, Height: 600})
	target := NewSprite("target", TextureRegion{})
	target.WorldTransform = [6]float64{1, 0, 0, 1, 100, 100}

	cam.Follow(target, 10, -20, 1.0)
	cam.update(1.0 / 60.0)
	if !approxEqual(cam.X, 110, epsilon) || !approxEqual(cam.Y, 80, epsilon) {
		t.Errorf("follow with offset: cam = (%f,%f), want (110,80)", cam.X, cam.Y)
	}
}

func TestCameraUnfollow(t *testing.T) {
	cam := newCamera(Rect{X: 0, Y: 0, Width: 800, Height: 600})
	target := NewSprite("target", TextureRegion{})
	target.WorldTransform = [6]float64{1, 0, 0, 1, 100, 100}

	cam.Follow(target, 0, 0, 1.0)
	cam.update(1.0 / 60.0)
	cam.Unfollow()

	// Move target, camera should not follow
	target.WorldTransform[4] = 500
	cam.update(1.0 / 60.0)
	if !approxEqual(cam.X, 100, epsilon) {
		t.Errorf("after unfollow: cam.X = %f, want 100", cam.X)
	}
}

func TestCameraScrollTo(t *testing.T) {
	cam := newCamera(Rect{X: 0, Y: 0, Width: 800, Height: 600})
	cam.ScrollTo(100, 200, TweenConfig{Duration: 1.0, Ease: ease.Linear})

	// Advance halfway
	cam.update(0.5)
	if !approxEqual(cam.X, 50, 1.0) || !approxEqual(cam.Y, 100, 1.0) {
		t.Errorf("scroll halfway: cam = (%f,%f), want ~(50,100)", cam.X, cam.Y)
	}

	// Advance to end
	cam.update(0.5)
	if !approxEqual(cam.X, 100, 1.0) || !approxEqual(cam.Y, 200, 1.0) {
		t.Errorf("scroll end: cam = (%f,%f), want ~(100,200)", cam.X, cam.Y)
	}

	// Tween should be cleared
	if cam.scrollTween != nil {
		t.Error("scrollTween not nil after completion")
	}
}

func TestCameraScrollToTile(t *testing.T) {
	cam := newCamera(Rect{X: 0, Y: 0, Width: 800, Height: 600})
	cam.ScrollToTile(3, 2, 32, 32, TweenConfig{Duration: 0.0001, Ease: ease.Linear})

	// tile center: (3*32+16, 2*32+16) = (112, 80)
	cam.update(1.0) // large dt to finish instantly
	if !approxEqual(cam.X, 112, 1.0) || !approxEqual(cam.Y, 80, 1.0) {
		t.Errorf("scrollToTile: cam = (%f,%f), want ~(112,80)", cam.X, cam.Y)
	}
}

func TestCameraBounds(t *testing.T) {
	cam := newCamera(Rect{X: 0, Y: 0, Width: 100, Height: 100})
	cam.SetBounds(Rect{X: 0, Y: 0, Width: 1000, Height: 1000})

	// Camera at (0,0) with viewport 100x100  -  min visible area is (50,50) center
	cam.X = 0
	cam.Y = 0
	cam.update(0)
	if cam.X < 50 || cam.Y < 50 {
		t.Errorf("bounds clamp min: cam = (%f,%f), want >= (50,50)", cam.X, cam.Y)
	}

	// Try to go past right edge
	cam.X = 999
	cam.Y = 999
	cam.dirty = true
	cam.update(0)
	if cam.X > 950 || cam.Y > 950 {
		t.Errorf("bounds clamp max: cam = (%f,%f), want <= (950,950)", cam.X, cam.Y)
	}
}

func TestCameraClearBounds(t *testing.T) {
	cam := newCamera(Rect{X: 0, Y: 0, Width: 100, Height: 100})
	cam.SetBounds(Rect{X: 0, Y: 0, Width: 1000, Height: 1000})
	cam.ClearBounds()

	cam.X = -999
	cam.Y = -999
	cam.update(0)
	// No clamping should occur
	if cam.X != -999 || cam.Y != -999 {
		t.Errorf("after ClearBounds: cam = (%f,%f), want (-999,-999)", cam.X, cam.Y)
	}
}

func TestCameraBoundsSmallWorld(t *testing.T) {
	cam := newCamera(Rect{X: 0, Y: 0, Width: 800, Height: 600})
	// World smaller than viewport  -  should center
	cam.SetBounds(Rect{X: 0, Y: 0, Width: 100, Height: 100})
	cam.X = 0
	cam.Y = 0
	cam.update(0)
	if !approxEqual(cam.X, 50, epsilon) || !approxEqual(cam.Y, 50, epsilon) {
		t.Errorf("small world center: cam = (%f,%f), want (50,50)", cam.X, cam.Y)
	}
}

// --- Culling tests ---

func TestWorldAABB(t *testing.T) {
	// Identity transform, 64x64 sprite
	aabb := worldAABB(identityTransform, 64, 64)
	if !approxEqual(aabb.X, 0, epsilon) || !approxEqual(aabb.Y, 0, epsilon) {
		t.Errorf("AABB origin = (%f,%f), want (0,0)", aabb.X, aabb.Y)
	}
	if !approxEqual(aabb.Width, 64, epsilon) || !approxEqual(aabb.Height, 64, epsilon) {
		t.Errorf("AABB size = (%f,%f), want (64,64)", aabb.Width, aabb.Height)
	}
}

func TestWorldAABB_Translated(t *testing.T) {
	transform := [6]float64{1, 0, 0, 1, 100, 200}
	aabb := worldAABB(transform, 32, 32)
	if !approxEqual(aabb.X, 100, epsilon) || !approxEqual(aabb.Y, 200, epsilon) {
		t.Errorf("translated AABB origin = (%f,%f), want (100,200)", aabb.X, aabb.Y)
	}
}

func TestWorldAABB_Rotated(t *testing.T) {
	// 45° rotation of a 100x100 sprite at origin
	cos45 := math.Cos(math.Pi / 4)
	sin45 := math.Sin(math.Pi / 4)
	transform := [6]float64{cos45, sin45, -sin45, cos45, 0, 0}
	aabb := worldAABB(transform, 100, 100)
	// A 100x100 square rotated 45° has AABB approximately 141x141
	expectedSize := 100 * math.Sqrt(2)
	if !approxEqual(aabb.Width, expectedSize, 0.01) || !approxEqual(aabb.Height, expectedSize, 0.01) {
		t.Errorf("rotated AABB size = (%f,%f), want ~(%f,%f)", aabb.Width, aabb.Height, expectedSize, expectedSize)
	}
}

func TestCulling_InsideViewport(t *testing.T) {
	n := NewSprite("visible", TextureRegion{Width: 64, Height: 64, OriginalW: 64, OriginalH: 64})
	n.WorldTransform = [6]float64{1, 0, 0, 1, 100, 100}
	viewport := Rect{X: 0, Y: 0, Width: 800, Height: 600}
	if shouldCull(n, n.WorldTransform, viewport) {
		t.Error("node inside viewport was culled")
	}
}

func TestCulling_OutsideViewport(t *testing.T) {
	n := NewSprite("outside", TextureRegion{Width: 64, Height: 64, OriginalW: 64, OriginalH: 64})
	n.WorldTransform = [6]float64{1, 0, 0, 1, -200, -200}
	viewport := Rect{X: 0, Y: 0, Width: 800, Height: 600}
	if !shouldCull(n, n.WorldTransform, viewport) {
		t.Error("node outside viewport was not culled")
	}
}

func TestCulling_ContainerNeverCulled(t *testing.T) {
	n := NewContainer("container")
	n.WorldTransform = [6]float64{1, 0, 0, 1, -9999, -9999}
	viewport := Rect{X: 0, Y: 0, Width: 800, Height: 600}
	if shouldCull(n, n.WorldTransform, viewport) {
		t.Error("container was culled")
	}
}

func TestCulling_TextNeverCulled(t *testing.T) {
	n := NewText("text", "hello", nil)
	n.WorldTransform = [6]float64{1, 0, 0, 1, -9999, -9999}
	viewport := Rect{X: 0, Y: 0, Width: 800, Height: 600}
	if shouldCull(n, n.WorldTransform, viewport) {
		t.Error("text node was culled")
	}
}

func TestCulling_IntegrationWithScene(t *testing.T) {
	scene := NewScene()
	cam := scene.NewCamera(Rect{X: 0, Y: 0, Width: 800, Height: 600})
	cam.X = 400
	cam.Y = 300
	cam.dirty = true

	// Sprite inside viewport
	visible := NewSprite("visible", TextureRegion{Width: 64, Height: 64, OriginalW: 64, OriginalH: 64, Page: 0})
	visible.X_ = 400
	visible.Y_ = 300
	scene.Root().AddChild(visible)

	// Sprite far outside viewport
	hidden := NewSprite("hidden", TextureRegion{Width: 64, Height: 64, OriginalW: 64, OriginalH: 64, Page: 0})
	hidden.X_ = 5000
	hidden.Y_ = 5000
	scene.Root().AddChild(hidden)

	// Register a dummy page so commands can be emitted
	page := ebiten.NewImage(1024, 1024)
	scene.RegisterPage(0, page)

	screen := ebiten.NewImage(800, 600)
	updateWorldTransform(scene.root, identityTransform, 1.0, false, false)
	scene.Draw(screen)

	// Only the visible sprite should produce a command
	if len(scene.commands) != 1 {
		t.Errorf("command count = %d, want 1 (visible only)", len(scene.commands))
	}
}

func TestCulling_DisabledShowsAll(t *testing.T) {
	scene := NewScene()
	cam := scene.NewCamera(Rect{X: 0, Y: 0, Width: 800, Height: 600})
	cam.X = 400
	cam.Y = 300
	cam.CullEnabled = false
	cam.dirty = true

	visible := NewSprite("s1", TextureRegion{Width: 64, Height: 64, OriginalW: 64, OriginalH: 64})
	visible.X_ = 400
	visible.Y_ = 300
	scene.Root().AddChild(visible)

	hidden := NewSprite("s2", TextureRegion{Width: 64, Height: 64, OriginalW: 64, OriginalH: 64})
	hidden.X_ = 5000
	hidden.Y_ = 5000
	scene.Root().AddChild(hidden)

	screen := ebiten.NewImage(800, 600)
	updateWorldTransform(scene.root, identityTransform, 1.0, false, false)
	scene.Draw(screen)

	if len(scene.commands) != 2 {
		t.Errorf("culling disabled: command count = %d, want 2", len(scene.commands))
	}
}

// --- Multi-camera tests ---

func TestSceneNewCamera(t *testing.T) {
	scene := NewScene()
	cam1 := scene.NewCamera(Rect{X: 0, Y: 0, Width: 400, Height: 300})
	cam2 := scene.NewCamera(Rect{X: 400, Y: 0, Width: 400, Height: 300})

	cams := scene.Cameras()
	if len(cams) != 2 {
		t.Fatalf("camera count = %d, want 2", len(cams))
	}
	if cams[0] != cam1 || cams[1] != cam2 {
		t.Error("cameras in wrong order")
	}
}

func TestSceneRemoveCamera(t *testing.T) {
	scene := NewScene()
	cam1 := scene.NewCamera(Rect{X: 0, Y: 0, Width: 400, Height: 300})
	scene.NewCamera(Rect{X: 400, Y: 0, Width: 400, Height: 300})

	scene.RemoveCamera(cam1)
	if len(scene.Cameras()) != 1 {
		t.Errorf("camera count after remove = %d, want 1", len(scene.Cameras()))
	}
}

func TestMultiCamera_BothRender(t *testing.T) {
	scene := NewScene()
	scene.NewCamera(Rect{X: 0, Y: 0, Width: 400, Height: 300})
	scene.NewCamera(Rect{X: 400, Y: 0, Width: 400, Height: 300})

	sprite := NewSprite("s", TextureRegion{Width: 32, Height: 32, OriginalW: 32, OriginalH: 32})
	scene.Root().AddChild(sprite)

	screen := ebiten.NewImage(800, 300)
	updateWorldTransform(scene.root, identityTransform, 1.0, false, false)
	scene.Draw(screen)
	// Both cameras should render the sprite  -  we can verify that Draw didn't panic.
	// Detailed multi-camera output verification would need pixel checks.
}

func TestSceneUpdateRunsCameraUpdates(t *testing.T) {
	scene := NewScene()
	cam := scene.NewCamera(Rect{X: 0, Y: 0, Width: 800, Height: 600})
	cam.ScrollTo(100, 0, TweenConfig{Duration: 1.0, Ease: ease.Linear})

	scene.Update() // Should advance scroll
	if cam.X == 0 {
		t.Error("Scene.Update() did not advance camera scroll")
	}
}

func TestNoCameraImplicitIdentity(t *testing.T) {
	scene := NewScene()
	sprite := NewSprite("s", TextureRegion{Width: 32, Height: 32, OriginalW: 32, OriginalH: 32})
	scene.Root().AddChild(sprite)

	screen := ebiten.NewImage(800, 600)
	updateWorldTransform(scene.root, identityTransform, 1.0, false, false)
	scene.Draw(screen) // Should not panic  -  uses implicit identity camera
}

// --- Camera Invalidate ---

func TestCameraInvalidate(t *testing.T) {
	cam := newCamera(Rect{X: 0, Y: 0, Width: 800, Height: 600})
	cam.computeViewMatrix()
	if cam.dirty {
		t.Error("camera should not be dirty after computeViewMatrix")
	}
	cam.Invalidate()
	if !cam.dirty {
		t.Error("camera should be dirty after Invalidate")
	}
}

// --- Benchmarks ---

func BenchmarkWorldAABB(b *testing.B) {
	transform := [6]float64{0.866, 0.5, -0.5, 0.866, 100, 200}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = worldAABB(transform, 64, 64)
	}
}

func BenchmarkCulling_10000Nodes(b *testing.B) {
	scene := NewScene()
	cam := scene.NewCamera(Rect{X: 0, Y: 0, Width: 800, Height: 600})
	cam.X = 400
	cam.Y = 300
	cam.dirty = true

	page := ebiten.NewImage(1024, 1024)
	scene.RegisterPage(0, page)

	for i := 0; i < 10000; i++ {
		s := NewSprite("s", TextureRegion{Width: 32, Height: 32, OriginalW: 32, OriginalH: 32, Page: 0})
		if i%2 == 0 {
			// Inside viewport
			s.X_ = float64(i%40) * 20
			s.Y_ = float64(i/40) * 20
		} else {
			// Outside viewport
			s.X_ = 5000
			s.Y_ = 5000
		}
		scene.Root().AddChild(s)
	}

	screen := ebiten.NewImage(800, 600)
	// Warm up: compute all transforms then draw
	updateWorldTransform(scene.root, identityTransform, 1.0, false, false)
	scene.Draw(screen)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		scene.Draw(screen)
	}
}
