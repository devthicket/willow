package main

import (
	"encoding/json"
	"image"
	"path/filepath"
)

// Sprite holds all data needed to composite one sprite into the atlas and
// generate its JSON entry.
type Sprite struct {
	Name      string
	Image     *image.NRGBA // trimmed (or original) pixels
	PackX     int          // top-left X in atlas
	PackY     int          // top-left Y in atlas
	Rotated   bool         // placed with 90° CW rotation
	Trimmed   bool
	OffsetX   int // trim offset X in original coordinate space
	OffsetY   int // trim offset Y in original coordinate space
	OriginalW int // untrimmed width
	OriginalH int // untrimmed height
}

// ComposeAtlas draws all sprites onto a new NRGBA image of the given dimensions.
func ComposeAtlas(sprites []Sprite, w, h int) *image.NRGBA {
	atlas := image.NewNRGBA(image.Rect(0, 0, w, h))
	for i := range sprites {
		s := &sprites[i]
		if s.Rotated {
			blitRotated90CW(atlas, s.Image, s.PackX, s.PackY)
		} else {
			blit(atlas, s.Image, s.PackX, s.PackY)
		}
	}
	return atlas
}

// blit copies src pixels into dst at (dstX, dstY) row-by-row.
func blit(dst *image.NRGBA, src *image.NRGBA, dstX, dstY int) {
	srcBounds := src.Bounds()
	srcW := srcBounds.Dx()
	srcH := srcBounds.Dy()
	for row := 0; row < srcH; row++ {
		srcOff := src.PixOffset(srcBounds.Min.X, srcBounds.Min.Y+row)
		dstOff := dst.PixOffset(dstX, dstY+row)
		copy(dst.Pix[dstOff:dstOff+srcW*4], src.Pix[srcOff:srcOff+srcW*4])
	}
}

// blitRotated90CW copies src pixels into dst rotated 90° clockwise.
// A WxH source becomes HxW in the destination.
// Original pixel (sx, sy) maps to destination (dstX + srcH-1-sy, dstY + sx).
func blitRotated90CW(dst *image.NRGBA, src *image.NRGBA, dstX, dstY int) {
	srcBounds := src.Bounds()
	srcW := srcBounds.Dx()
	srcH := srcBounds.Dy()
	for sy := 0; sy < srcH; sy++ {
		for sx := 0; sx < srcW; sx++ {
			srcOff := src.PixOffset(srcBounds.Min.X+sx, srcBounds.Min.Y+sy)
			dstOff := dst.PixOffset(dstX+srcH-1-sy, dstY+sx)
			copy(dst.Pix[dstOff:dstOff+4], src.Pix[srcOff:srcOff+4])
		}
	}
}

// --- JSON output types (match Willow's atlas.go schema) ---

type jsonRect struct {
	X int `json:"x"`
	Y int `json:"y"`
	W int `json:"w"`
	H int `json:"h"`
}

type jsonSize struct {
	W int `json:"w"`
	H int `json:"h"`
}

type jsonFrame struct {
	Frame            jsonRect `json:"frame"`
	Rotated          bool     `json:"rotated"`
	Trimmed          bool     `json:"trimmed"`
	SpriteSourceSize jsonRect `json:"spriteSourceSize"`
	SourceSize       jsonSize `json:"sourceSize"`
}

type jsonAtlas struct {
	Frames map[string]jsonFrame `json:"frames"`
	Meta   jsonMeta             `json:"meta"`
}

type jsonMeta struct {
	Image string   `json:"image"`
	Size  jsonSize `json:"size"`
}

// GenerateJSON produces TexturePacker-compatible hash-format JSON.
func GenerateJSON(sprites []Sprite, w, h int, imageFile string) ([]byte, error) {
	frames := make(map[string]jsonFrame, len(sprites))
	for i := range sprites {
		s := &sprites[i]

		// Frame dimensions: if rotated, w/h are swapped (h becomes w, w becomes h).
		frameW := s.Image.Bounds().Dx()
		frameH := s.Image.Bounds().Dy()
		if s.Rotated {
			frameW, frameH = frameH, frameW
		}

		frames[s.Name] = jsonFrame{
			Frame: jsonRect{
				X: s.PackX,
				Y: s.PackY,
				W: frameW,
				H: frameH,
			},
			Rotated: s.Rotated,
			Trimmed: s.Trimmed,
			SpriteSourceSize: jsonRect{
				X: s.OffsetX,
				Y: s.OffsetY,
				W: frameW,
				H: frameH,
			},
			SourceSize: jsonSize{
				W: s.OriginalW,
				H: s.OriginalH,
			},
		}
	}

	atlas := jsonAtlas{
		Frames: frames,
		Meta: jsonMeta{
			Image: filepath.Base(imageFile),
			Size:  jsonSize{W: w, H: h},
		},
	}

	return json.MarshalIndent(atlas, "", "  ")
}
