package core

import (
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
)

// keyNameMap maps JSON-friendly key names to ebiten key constants.
// All lookups are lowercased so "W", "w", and "Tab" all work.
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
	"g":         ebiten.KeyG,
	"h":         ebiten.KeyH,
	"i":         ebiten.KeyI,
	"j":         ebiten.KeyJ,
	"k":         ebiten.KeyK,
	"l":         ebiten.KeyL,
	"m":         ebiten.KeyM,
	"n":         ebiten.KeyN,
	"o":         ebiten.KeyO,
	"p":         ebiten.KeyP,
	"q":         ebiten.KeyQ,
	"r":         ebiten.KeyR,
	"s":         ebiten.KeyS,
	"t":         ebiten.KeyT,
	"u":         ebiten.KeyU,
	"v":         ebiten.KeyV,
	"w":         ebiten.KeyW,
	"x":         ebiten.KeyX,
	"y":         ebiten.KeyY,
	"z":         ebiten.KeyZ,
	"0":         ebiten.Key0,
	"1":         ebiten.Key1,
	"2":         ebiten.Key2,
	"3":         ebiten.Key3,
	"4":         ebiten.Key4,
	"5":         ebiten.Key5,
	"6":         ebiten.Key6,
	"7":         ebiten.Key7,
	"8":         ebiten.Key8,
	"9":         ebiten.Key9,
	"f1":        ebiten.KeyF1,
	"f2":        ebiten.KeyF2,
	"f3":        ebiten.KeyF3,
	"f4":        ebiten.KeyF4,
	"f5":        ebiten.KeyF5,
	"f6":        ebiten.KeyF6,
	"f7":        ebiten.KeyF7,
	"f8":        ebiten.KeyF8,
	"f9":        ebiten.KeyF9,
	"f10":       ebiten.KeyF10,
	"f11":       ebiten.KeyF11,
	"f12":       ebiten.KeyF12,
}

// KeyFromName returns the ebiten.Key for a JSON key name.
// Lookup is case-insensitive.  Returns -1 if the name is not recognized.
func KeyFromName(name string) ebiten.Key {
	if k, ok := keyNameMap[strings.ToLower(name)]; ok {
		return k
	}
	return -1
}
