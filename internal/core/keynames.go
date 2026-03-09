package core

import "github.com/hajimehoshi/ebiten/v2"

// keyNameMap maps JSON-friendly key names to ebiten key constants.
var keyNameMap = map[string]ebiten.Key{
	"enter":     ebiten.KeyEnter,
	"backspace": ebiten.KeyBackspace,
	"delete":    ebiten.KeyDelete,
	"escape":    ebiten.KeyEscape,
	"tab":       ebiten.KeyTab,
	"space":     ebiten.KeySpace,
	"left":      ebiten.KeyLeft,
	"right":     ebiten.KeyRight,
	"up":        ebiten.KeyUp,
	"down":      ebiten.KeyDown,
	"home":      ebiten.KeyHome,
	"end":       ebiten.KeyEnd,
	"shift":     ebiten.KeyShift,
	"control":   ebiten.KeyControl,
	"meta":      ebiten.KeyMeta,
	"a":         ebiten.KeyA,
	"b":         ebiten.KeyB,
	"c":         ebiten.KeyC,
	"d":         ebiten.KeyD,
	"e":         ebiten.KeyE,
	"f":         ebiten.KeyF,
	"v":         ebiten.KeyV,
	"x":         ebiten.KeyX,
	"z":         ebiten.KeyZ,
}

// KeyFromName returns the ebiten.Key for a JSON key name.
// Returns -1 if the name is not recognized.
func KeyFromName(name string) ebiten.Key {
	if k, ok := keyNameMap[name]; ok {
		return k
	}
	return -1
}
