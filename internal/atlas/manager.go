package atlas

import "github.com/hajimehoshi/ebiten/v2"

// Manager is the global singleton that owns atlas page images.
// Pages are shared across Scenes, enabling fonts and atlases to be created
// independently of any particular Scene. Single-threaded (no sync needed).
type Manager struct {
	pages    []*ebiten.Image
	nextPage int
	refs     []int
	static   []bool
}

var globalManager *Manager

// GlobalManager returns the global Manager singleton, lazily initialised.
func GlobalManager() *Manager {
	if globalManager == nil {
		globalManager = &Manager{}
	}
	return globalManager
}

// Page returns the atlas page image at the given index, or nil if out of range.
func (am *Manager) Page(index int) *ebiten.Image {
	if index < 0 || index >= len(am.pages) {
		return nil
	}
	return am.pages[index]
}

// PageCount returns the number of page slots currently allocated.
func (am *Manager) PageCount() int {
	return len(am.pages)
}

// NextPage returns the next available page index (without allocating it).
func (am *Manager) NextPage() int {
	return am.nextPage
}

// AllocPage reserves and returns the next available page index.
func (am *Manager) AllocPage() int {
	idx := am.nextPage
	am.nextPage++
	return idx
}

// RegisterPage stores an atlas page image at the given index, growing
// internal slices as needed.
func (am *Manager) RegisterPage(index int, img *ebiten.Image) {
	for len(am.pages) <= index {
		am.pages = append(am.pages, nil)
	}
	am.pages[index] = img

	// Keep nextPage past all registered pages so dynamic allocation
	// (TTF/SDF cached images) never overwrites them.
	if index+1 > am.nextPage {
		am.nextPage = index + 1
	}

	// Grow auxiliary slices to match.
	for len(am.refs) <= index {
		am.refs = append(am.refs, 0)
	}
	for len(am.static) <= index {
		am.static = append(am.static, false)
	}
}

// Retain increments the reference count for the given page index.
func (am *Manager) Retain(index int) {
	for len(am.refs) <= index {
		am.refs = append(am.refs, 0)
	}
	am.refs[index]++
}

// Release decrements the reference count for the given page index.
// Does not go below zero.
func (am *Manager) Release(index int) {
	if index < 0 || index >= len(am.refs) {
		return
	}
	if am.refs[index] > 0 {
		am.refs[index]--
	}
}

// SetStatic marks a page as permanent so Cleanup never deallocates it.
func (am *Manager) SetStatic(index int) {
	for len(am.static) <= index {
		am.static = append(am.static, false)
	}
	am.static[index] = true
}

// Cleanup deallocates dynamic pages with zero references.
// Static pages and pages with non-zero refs are left untouched.
func (am *Manager) Cleanup() {
	for i := range am.pages {
		if am.pages[i] == nil {
			continue
		}
		if i < len(am.static) && am.static[i] {
			continue
		}
		if i < len(am.refs) && am.refs[i] > 0 {
			continue
		}
		am.pages[i].Deallocate()
		am.pages[i] = nil
	}
}

// ResetGlobalManager resets the global singleton for test isolation.
func ResetGlobalManager() {
	globalManager = nil
}
