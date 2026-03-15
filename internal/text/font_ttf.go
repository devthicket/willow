package text

// NewFontFromTTFOpts generates an SDF font from TTF/OTF data using explicit
// SDFGenOptions. PageIndex is overwritten by an allocated page.
func NewFontFromTTFOpts(ttfData []byte, opts SDFGenOptions) (*DistanceFieldFont, error) {
	opts.PageIndex = uint16(AllocPageFn())
	sdfFont, atlasImg, _, err := LoadDistanceFieldFontFromTTF(ttfData, opts)
	if err != nil {
		return nil, err
	}
	RegisterPageFn(int(opts.PageIndex), atlasImg)
	sdfFont.SetTTFData(ttfData)
	return sdfFont, nil
}

// NewMSDFFontFromTTFOpts generates an MSDF font from TTF/OTF data using
// explicit SDFGenOptions. PageIndex is overwritten by an allocated page.
//
// EXPERIMENTAL — MSDF produces inconsistent rendering quality compared to SDF.
// Multi-contour glyphs (0, 8, @, etc.) exhibit anti-aliasing artifacts and
// channel divergence that degrades text appearance. Use NewFontFromTTFOpts
// (single-channel SDF) for production text rendering.
func NewMSDFFontFromTTFOpts(ttfData []byte, opts SDFGenOptions) (*DistanceFieldFont, error) {
	opts.PageIndex = uint16(AllocPageFn())
	msdfFont, atlasImg, _, err := LoadDistanceFieldFontFromTTFMSDF(ttfData, opts)
	if err != nil {
		return nil, err
	}
	RegisterPageFn(int(opts.PageIndex), atlasImg)
	msdfFont.SetTTFData(ttfData)
	return msdfFont, nil
}

// NewMSDFFontFromTTF generates an MSDF font from TTF/OTF data. MSDF encodes
// per-channel directional distances into R, G, B, enabling sharper corners and
// crisper edges than single-channel SDF at all display sizes.
//
// EXPERIMENTAL — MSDF produces inconsistent rendering quality compared to SDF.
// Multi-contour glyphs (0, 8, @, etc.) exhibit anti-aliasing artifacts and
// channel divergence that degrades text appearance. Use NewFontFromTTF
// (single-channel SDF) for production text rendering.
func NewMSDFFontFromTTF(ttfData []byte, size float64) (*DistanceFieldFont, error) {
	pageIndex := AllocPageFn()
	msdfFont, atlasImg, _, err := LoadDistanceFieldFontFromTTFMSDF(ttfData, SDFGenOptions{
		Size:      size,
		PageIndex: uint16(pageIndex),
	})
	if err != nil {
		return nil, err
	}
	RegisterPageFn(pageIndex, atlasImg)
	msdfFont.SetTTFData(ttfData)
	return msdfFont, nil
}

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
