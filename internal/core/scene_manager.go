package core

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

// Transition controls the visual effect between scene changes.
type Transition interface {
	Duration() float32
	Update(dt float32)
	Draw(screen *ebiten.Image, progress float32)
	Done() bool
}

// SceneManager manages a stack of scenes with optional transitions.
type SceneManager struct {
	stack      []*Scene
	transition Transition
	// transState: 0 = idle, 1 = fading out, 2 = fading in
	transState   int
	transElapsed float32
	transDur     float32
	pendingScene *Scene
	pendingOp    int // 1 = push, 2 = replace, 3 = pop
}

// NewSceneManager creates a SceneManager with the given initial scene.
func NewSceneManager(initial *Scene) *SceneManager {
	sm := &SceneManager{
		stack: []*Scene{initial},
	}
	if initial.OnEnterFn != nil {
		initial.OnEnterFn()
	}
	return sm
}

// Push adds a scene on top of the stack with an optional transition.
func (sm *SceneManager) Push(scene *Scene) {
	if sm.transState != 0 {
		return // transition in progress
	}
	if sm.transition != nil {
		sm.pendingScene = scene
		sm.pendingOp = 1
		sm.startTransitionOut()
	} else {
		sm.doPush(scene)
	}
}

// Pop removes the top scene and returns to the previous one.
// Panics if only one scene remains.
func (sm *SceneManager) Pop() {
	if len(sm.stack) <= 1 {
		panic("SceneManager.Pop: cannot pop the last scene")
	}
	if sm.transState != 0 {
		return
	}
	if sm.transition != nil {
		sm.pendingOp = 3
		sm.startTransitionOut()
	} else {
		sm.doPop()
	}
}

// Replace swaps the current scene for a new one (no stack growth).
func (sm *SceneManager) Replace(scene *Scene) {
	if sm.transState != 0 {
		return
	}
	if sm.transition != nil {
		sm.pendingScene = scene
		sm.pendingOp = 2
		sm.startTransitionOut()
	} else {
		sm.doReplace(scene)
	}
}

// SetTransition configures the transition for the next scene change.
// Pass nil to disable. Consumed after one use.
func (sm *SceneManager) SetTransition(t Transition) {
	sm.transition = t
}

// Current returns the active scene (top of stack).
func (sm *SceneManager) Current() *Scene {
	if len(sm.stack) == 0 {
		return nil
	}
	return sm.stack[len(sm.stack)-1]
}

// Update advances the active scene and any in-progress transition.
func (sm *SceneManager) Update() {
	dt := float32(1.0 / float64(ebiten.TPS()))

	switch sm.transState {
	case 1: // fading out — old scene frozen, just advance transition
		sm.transElapsed += dt
		sm.transition.Update(dt)
		if sm.transElapsed >= sm.transDur/2 {
			sm.executeOp()
			sm.transState = 2
		}
	case 2: // fading in — new scene updates normally
		sm.transElapsed += dt
		sm.transition.Update(dt)
		sm.Current().Update()
		if sm.transition.Done() {
			sm.transState = 0
			sm.transition = nil // consumed
			sm.pendingScene = nil
		}
	default:
		cur := sm.Current()
		if cur != nil {
			cur.Update()
		}
	}
}

// Draw renders the active scene and any transition overlay.
func (sm *SceneManager) Draw(screen *ebiten.Image) {
	cur := sm.Current()
	if cur != nil {
		cur.Draw(screen)
	}
	if sm.transState != 0 && sm.transition != nil {
		progress := sm.transElapsed / sm.transDur
		if progress > 1 {
			progress = 1
		}
		sm.transition.Draw(screen, progress)
	}
}

func (sm *SceneManager) startTransitionOut() {
	sm.transState = 1
	sm.transElapsed = 0
	sm.transDur = sm.transition.Duration()
}

func (sm *SceneManager) executeOp() {
	switch sm.pendingOp {
	case 1:
		sm.doPush(sm.pendingScene)
	case 2:
		sm.doReplace(sm.pendingScene)
	case 3:
		sm.doPop()
	}
	sm.pendingOp = 0
}

func (sm *SceneManager) doPush(scene *Scene) {
	cur := sm.Current()
	if cur != nil && cur.OnExitFn != nil {
		cur.OnExitFn()
	}
	sm.stack = append(sm.stack, scene)
	if scene.OnEnterFn != nil {
		scene.OnEnterFn()
	}
}

func (sm *SceneManager) doPop() {
	cur := sm.Current()
	if cur != nil && cur.OnExitFn != nil {
		cur.OnExitFn()
	}
	sm.stack = sm.stack[:len(sm.stack)-1]
	next := sm.Current()
	if next != nil && next.OnEnterFn != nil {
		next.OnEnterFn()
	}
}

func (sm *SceneManager) doReplace(scene *Scene) {
	cur := sm.Current()
	if cur != nil && cur.OnExitFn != nil {
		cur.OnExitFn()
	}
	sm.stack[len(sm.stack)-1] = scene
	if scene.OnEnterFn != nil {
		scene.OnEnterFn()
	}
}

// --- FadeTransition ---

// FadeTransition fades through a solid color (typically black).
type FadeTransition struct {
	dur     float32
	elapsed float32
	col     color.Color
	done    bool
}

// NewFadeTransition creates a fade transition with the given duration and color.
func NewFadeTransition(duration float32, c color.Color) *FadeTransition {
	return &FadeTransition{dur: duration, col: c}
}

func (f *FadeTransition) Duration() float32 { return f.dur }

func (f *FadeTransition) Update(dt float32) {
	f.elapsed += dt
	if f.elapsed >= f.dur {
		f.done = true
	}
}

func (f *FadeTransition) Draw(screen *ebiten.Image, progress float32) {
	// Fade out: alpha goes 0→1 in first half, 1→0 in second half.
	var alpha float32
	if progress < 0.5 {
		alpha = progress * 2
	} else {
		alpha = (1.0 - progress) * 2
	}
	if alpha < 0 {
		alpha = 0
	}
	if alpha > 1 {
		alpha = 1
	}

	r, g, b, _ := f.col.RGBA()
	r8 := uint8(r >> 8)
	g8 := uint8(g >> 8)
	b8 := uint8(b >> 8)
	a8 := uint8(float32(255) * alpha)

	overlay := ebiten.NewImage(screen.Bounds().Dx(), screen.Bounds().Dy())
	overlay.Fill(color.RGBA{R: r8, G: g8, B: b8, A: a8})
	screen.DrawImage(overlay, nil)
}

func (f *FadeTransition) Done() bool { return f.done }
