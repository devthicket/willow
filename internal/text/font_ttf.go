package text

// NewFontFromTTF generates an SDF font from TTF/OTF data, registers the atlas
// page via function pointers, and returns the font ready to use.
func NewFontFromTTF(ttfData []byte, size float64) (*DistanceFieldFont, error) {
	pageIndex := AllocPageFn()
	sdfFont, atlasImg, _, err := LoadDistanceFieldFontFromTTF(ttfData, SDFGenOptions{
		Size:      size,
		PageIndex: uint16(pageIndex),
	})
	if err != nil {
		return nil, err
	}
	RegisterPageFn(pageIndex, atlasImg)
	sdfFont.SetTTFData(ttfData)
	return sdfFont, nil
}
