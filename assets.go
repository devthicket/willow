package willow

import _ "embed"

// GofontBundle is the pre-baked .fontbundle for the Go Regular font family.
// Pass it to [NewFontFamilyFromFontBundle] to get a ready-to-use FontFamily.
//
//	font, err := willow.NewFontFamilyFromFontBundle(willow.GofontBundle)
//
//go:embed assets/fonts/gofont.fontbundle
var GofontBundle []byte
