package text

// Font is the interface for text measurement and layout.
// Implemented by SpriteFont and PixelFont.
//
// All measurements are in native atlas pixels. To get display-sized values,
// use TextBlock.MeasureDisplay or scale manually by TextBlock.FontScale().
type Font interface {
	// MeasureString returns the pixel width and height of the rendered text
	// in native atlas pixels, accounting for newlines and the font's line height.
	MeasureString(text string) (width, height float64)
	// LineHeight returns the vertical distance between baselines in native atlas pixels.
	LineHeight() float64
}
