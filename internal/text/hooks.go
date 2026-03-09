package text

import "github.com/hajimehoshi/ebiten/v2"

// RegisterPageFn registers an atlas page image. Wired by root.
var RegisterPageFn func(pageIndex int, img *ebiten.Image)

// NextPageFn returns the next available atlas page index. Wired by root.
var NextPageFn func() int

// AllocPageFn allocates and returns a new atlas page index. Wired by root.
var AllocPageFn func() int
