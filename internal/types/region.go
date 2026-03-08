package types

// TextureRegion describes a sub-rectangle within an atlas page.
// Value type (32 bytes) — stored directly on Node, no pointer.
type TextureRegion struct {
	Page      uint16 // atlas page index (references Scene.pages)
	X, Y      uint16 // top-left corner of the sub-image rect within the atlas page
	Width     uint16 // width of the sub-image rect (may differ from OriginalW if trimmed)
	Height    uint16 // height of the sub-image rect (may differ from OriginalH if trimmed)
	OriginalW uint16 // untrimmed sprite width as authored
	OriginalH uint16 // untrimmed sprite height as authored
	OffsetX   int16  // horizontal trim offset from TexturePacker
	OffsetY   int16  // vertical trim offset from TexturePacker
	Rotated   bool   // true if the region is stored 90 degrees clockwise in the atlas
}
