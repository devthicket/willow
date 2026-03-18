package render

import (
	"math"

	"github.com/devthicket/willow/internal/node"
	"github.com/devthicket/willow/internal/text"
	"github.com/hajimehoshi/ebiten/v2"
)

// --- SDF Kage shader sources ---

const SdfShaderSrc = `//kage:unit pixels
package main

var Threshold float
var Smoothing float
var OutlineWidth float
var OutlineColor vec4
var GlowWidth float
var GlowColor vec4
var ShadowOffset vec2
var ShadowColor vec4
var ShadowSoftness float
var FillColor vec4

func Fragment(dst vec4, src vec2, color vec4) vec4 {
	dist := imageSrc0At(src).a

	sm := max(fwidth(dist), Smoothing)

	var result vec4
	if ShadowColor.a > 0 && (ShadowOffset.x != 0 || ShadowOffset.y != 0 || ShadowSoftness > 0) {
		shadowDist := imageSrc0At(src - ShadowOffset).a
		shadowAlpha := smoothstep(Threshold - ShadowSoftness - sm, Threshold - ShadowSoftness + sm, shadowDist)
		result = ShadowColor * shadowAlpha
	}

	if GlowColor.a > 0 && GlowWidth > 0 {
		glowEdge := Threshold - OutlineWidth - GlowWidth
		glowAlpha := smoothstep(glowEdge - sm, glowEdge + sm, dist)
		result = mix(result, GlowColor, GlowColor.a * glowAlpha)
	}

	if OutlineColor.a > 0 && OutlineWidth > 0 {
		outerEdge := Threshold - OutlineWidth
		outlineAlpha := smoothstep(outerEdge - sm, outerEdge + sm, dist)
		result = mix(result, OutlineColor, OutlineColor.a * outlineAlpha)
	}

	fillAlpha := smoothstep(Threshold - sm, Threshold + sm, dist)
	result = mix(result, FillColor, FillColor.a * fillAlpha)

	result *= smoothstep(0, Smoothing*2, dist)

	result *= color

	return result
}
`

// MsdfShaderSrc is the MSDF fragment shader.
//
// EXPERIMENTAL — This shader is used when IsMultiChannel() is true. MSDF font
// generation currently produces inconsistent quality (see msdf.go). The SDF
// shader (SdfShaderSrc) is recommended for production use.
const MsdfShaderSrc = `//kage:unit pixels
package main

var Threshold float
var Smoothing float
var ScreenPxRange float
var OutlineWidth float
var OutlineColor vec4
var GlowWidth float
var GlowColor vec4
var ShadowOffset vec2
var ShadowColor vec4
var ShadowSoftness float
var FillColor vec4

func median3(a, b, c float) float {
	return max(min(a, b), min(max(a, b), c))
}

func sampleDist(pos vec2) float {
	s := imageSrc0At(pos)
	return median3(s.r, s.g, s.b)
}

// screenPxDist converts a distance-field value to screen-pixel distance
// using the msdfgen-recommended approach. This avoids computing fwidth()
// on the median (which is noisy at channel crossings) and instead uses
// the known pixel range of the distance field.
func screenPxDist(dist float) float {
	return ScreenPxRange * (dist - Threshold)
}

func Fragment(dst vec4, src vec2, color vec4) vec4 {
	dist := sampleDist(src)

	// Convert to screen pixels for anti-aliasing.
	// screenPxDist = 0 at the edge, positive inside, negative outside.
	// The 1-pixel transition band gives clean anti-aliasing.
	spd := screenPxDist(dist)

	// Fallback: if ScreenPxRange is not set, use fwidth-based smoothing.
	sm := Smoothing
	if ScreenPxRange <= 0 {
		sm = max(fwidth(dist), Smoothing)
	}

	var result vec4
	if ShadowColor.a > 0 && (ShadowOffset.x != 0 || ShadowOffset.y != 0 || ShadowSoftness > 0) {
		shadowDist := sampleDist(src - ShadowOffset)
		if ScreenPxRange > 0 {
			shadowSpd := screenPxDist(shadowDist) + ScreenPxRange*ShadowSoftness
			shadowAlpha := smoothstep(-0.75, 0.75, shadowSpd)
			result = ShadowColor * shadowAlpha
		} else {
			shadowAlpha := smoothstep(Threshold - ShadowSoftness - sm, Threshold - ShadowSoftness + sm, shadowDist)
			result = ShadowColor * shadowAlpha
		}
	}

	if GlowColor.a > 0 && GlowWidth > 0 {
		glowEdge := Threshold - OutlineWidth - GlowWidth
		if ScreenPxRange > 0 {
			glowSpd := ScreenPxRange * (dist - glowEdge)
			glowAlpha := smoothstep(-0.75, 0.75, glowSpd)
			result = mix(result, GlowColor, GlowColor.a * glowAlpha)
		} else {
			glowAlpha := smoothstep(glowEdge - sm, glowEdge + sm, dist)
			result = mix(result, GlowColor, GlowColor.a * glowAlpha)
		}
	}

	if OutlineColor.a > 0 && OutlineWidth > 0 {
		outerEdge := Threshold - OutlineWidth
		if ScreenPxRange > 0 {
			outSpd := ScreenPxRange * (dist - outerEdge)
			outAlpha := smoothstep(-0.75, 0.75, outSpd)
			result = mix(result, OutlineColor, OutlineColor.a * outAlpha)
		} else {
			outlineAlpha := smoothstep(outerEdge - sm, outerEdge + sm, dist)
			result = mix(result, OutlineColor, OutlineColor.a * outlineAlpha)
		}
	}

	if ScreenPxRange > 0 {
		// Use smoothstep for a wider, smoother anti-aliased transition
		// (S-curve over 1.5 screen pixels instead of linear 1px clamp).
		// This prevents round glyphs like "0" from looking hard-edged.
		fillAlpha := smoothstep(-0.75, 0.75, spd)
		result = mix(result, FillColor, FillColor.a * fillAlpha)
		// Outer fade: suppress channel-crossing noise outside the shape.
		result *= clamp(spd*0.667 + 1.0, 0.0, 1.0)
	} else {
		fillAlpha := smoothstep(Threshold - sm, Threshold + sm, dist)
		result = mix(result, FillColor, FillColor.a * fillAlpha)
		result *= smoothstep(0, Smoothing*2, dist)
	}

	result *= color

	return result
}
`

// InitShaders eagerly compiles all built-in render shaders so that any
// compilation failure panics at startup rather than at an unpredictable frame.
func InitShaders() {
	EnsureSDFShader()
	EnsureMSDFShader()
	EnsureFXAAShader()
}

// --- Lazy shader compilation ---

var (
	sdfShader  *ebiten.Shader
	msdfShader *ebiten.Shader
)

// EnsureSDFShader lazily compiles and returns the single-channel SDF shader.
func EnsureSDFShader() *ebiten.Shader {
	if sdfShader == nil {
		s, err := ebiten.NewShader([]byte(SdfShaderSrc))
		if err != nil {
			panic("willow: failed to compile SDF shader: " + err.Error())
		}
		sdfShader = s
	}
	return sdfShader
}

// EnsureMSDFShader lazily compiles the multi-channel SDF shader.
func EnsureMSDFShader() *ebiten.Shader {
	if msdfShader == nil {
		s, err := ebiten.NewShader([]byte(MsdfShaderSrc))
		if err != nil {
			panic("willow: failed to compile MSDF shader: " + err.Error())
		}
		msdfShader = s
	}
	return msdfShader
}

// EnsureUniforms builds or updates the cached uniform map for the SDF shader.
func EnsureUniforms(tb *text.TextBlock, displayScale float64) {
	ff := tb.Font
	if ff == nil {
		return
	}

	distRange := ff.SDFDistanceRange(tb.FontSize, tb.Bold, tb.Italic)
	isMultiChannel := ff.SDFIsMultiChannel(tb.FontSize, tb.Bold, tb.Italic)
	sdfFontSize := ff.SDFFontSize(tb.FontSize, tb.Bold, tb.Italic)

	smoothMult := 1.5
	if tb.Sharpness > 0 {
		s := tb.Sharpness
		if s > 1 {
			s = 1
		}
		smoothMult = 1.5 - s*1.0
		if smoothMult < 0.5 {
			smoothMult = 0.5
		}
	}
	smoothing := float32(smoothMult / (distRange * displayScale))

	threshold := float32(0.5)
	if !isMultiChannel && tb.FontSize > 0 && tb.FontSize < 24 {
		t := tb.FontSize / 24
		threshold = float32(0.30 + 0.20*t)
	}

	fillColor := tb.Color
	fillPremul := [4]float32{
		float32(fillColor.R() * fillColor.A()),
		float32(fillColor.G() * fillColor.A()),
		float32(fillColor.B() * fillColor.A()),
		float32(fillColor.A()),
	}

	if tb.SdfUniforms == nil {
		tb.SdfUniforms = make(map[string]any, 11)
		tb.UniformsDirty = true
	}

	tb.SdfUniforms["Smoothing"] = smoothing

	var screenPxRange float32
	if isMultiChannel {
		fontScale := float64(1.0)
		if tb.FontSize > 0 && sdfFontSize > 0 {
			fontScale = tb.FontSize / sdfFontSize
		}
		screenPxRange = float32(distRange * displayScale * fontScale)
		if screenPxRange < 1.0 {
			screenPxRange = 1.0
		}
	}
	tb.SdfUniforms["ScreenPxRange"] = screenPxRange

	if !tb.UniformsDirty {
		return
	}
	tb.UniformsDirty = false

	tb.SdfUniforms["Threshold"] = threshold
	tb.SdfUniforms["FillColor"] = fillPremul[:]
	tb.SdfUniforms["OutlineWidth"] = float32(0)
	tb.SdfUniforms["OutlineColor"] = []float32{0, 0, 0, 0}
	tb.SdfUniforms["GlowWidth"] = float32(0)
	tb.SdfUniforms["GlowColor"] = []float32{0, 0, 0, 0}
	tb.SdfUniforms["ShadowOffset"] = []float32{0, 0}
	tb.SdfUniforms["ShadowColor"] = []float32{0, 0, 0, 0}
	tb.SdfUniforms["ShadowSoftness"] = float32(0)

	if tb.TextEffects != nil {
		e := tb.TextEffects
		tb.SdfUniforms["OutlineWidth"] = float32(e.OutlineWidth / distRange)
		tb.SdfUniforms["OutlineColor"] = []float32{
			float32(e.OutlineColor.R() * e.OutlineColor.A()),
			float32(e.OutlineColor.G() * e.OutlineColor.A()),
			float32(e.OutlineColor.B() * e.OutlineColor.A()),
			float32(e.OutlineColor.A()),
		}
		tb.SdfUniforms["GlowWidth"] = float32(e.GlowWidth / distRange)
		tb.SdfUniforms["GlowColor"] = []float32{
			float32(e.GlowColor.R() * e.GlowColor.A()),
			float32(e.GlowColor.G() * e.GlowColor.A()),
			float32(e.GlowColor.B() * e.GlowColor.A()),
			float32(e.GlowColor.A()),
		}
		tb.SdfUniforms["ShadowOffset"] = []float32{
			float32(e.ShadowOffset.X),
			float32(e.ShadowOffset.Y),
		}
		tb.SdfUniforms["ShadowColor"] = []float32{
			float32(e.ShadowColor.R() * e.ShadowColor.A()),
			float32(e.ShadowColor.G() * e.ShadowColor.A()),
			float32(e.ShadowColor.B() * e.ShadowColor.A()),
			float32(e.ShadowColor.A()),
		}
		tb.SdfUniforms["ShadowSoftness"] = float32(e.ShadowSoftness / distRange)
	}

	if isMultiChannel {
		tb.SdfShader = EnsureMSDFShader()
	} else {
		tb.SdfShader = EnsureSDFShader()
	}
}

// EmitSDFTextCommand emits a CommandSDF carrying local-space glyph quads.
func EmitSDFTextCommand(tb *text.TextBlock, n *node.Node, worldTransform [6]float64, commands []RenderCommand, treeOrder *int) []RenderCommand {
	if tb.Font == nil || tb.Font.IsPixelMode() {
		return commands
	}

	tb.Layout()
	if tb.MeasuredW == 0 || tb.MeasuredH == 0 {
		return commands
	}

	if tb.SdfVertCount == 0 || len(tb.SdfVerts) == 0 {
		lineH := tb.Font.SDFLineHeight(tb.FontSize, tb.Bold, tb.Italic)
		tb.RebuildLocalVerts(lineH)
	}
	if tb.SdfVertCount == 0 {
		return commands
	}

	fontScale := tb.FontScale()
	fst := [6]float64{fontScale, 0, 0, fontScale, 0, 0}
	scaledWT := node.MultiplyAffine(worldTransform, fst)

	page := tb.Font.SDFPage(tb.FontSize, tb.Bold, tb.Italic)
	atlasImg := resolveAtlasPage(page)
	if atlasImg == nil {
		return commands
	}

	displayScale := math.Sqrt(worldTransform[0]*worldTransform[0] + worldTransform[1]*worldTransform[1])
	if displayScale < 0.05 {
		displayScale = 0.05
	}
	EnsureUniforms(tb, displayScale)

	*treeOrder++
	commands = append(commands, RenderCommand{
		Type:      CommandSDF,
		Transform: Affine32(scaledWT),
		Color: Color32{
			float32(n.Color_.R()),
			float32(n.Color_.G()),
			float32(n.Color_.B()),
			float32(n.Color_.A() * n.WorldAlpha),
		},
		BlendMode:    n.BlendMode_,
		RenderLayer:  n.RenderLayer,
		GlobalOrder:  n.GlobalOrder,
		TreeOrder:    *treeOrder,
		SdfVerts:     tb.SdfVerts,
		SdfInds:      tb.SdfInds,
		SdfVertCount: tb.SdfVertCount,
		SdfIndCount:  tb.SdfIndCount,
		SdfShader:    tb.SdfShader,
		SdfAtlasImg:  atlasImg,
		SdfUniforms:  tb.SdfUniforms,
	})

	return commands
}

// EmitPixelTextCommand emits a CommandBitmapText for pixel-perfect bitmap font rendering.
func EmitPixelTextCommand(tb *text.TextBlock, n *node.Node, worldTransform [6]float64, commands []RenderCommand, treeOrder *int) []RenderCommand {
	if tb.Font == nil || !tb.Font.IsPixelMode() {
		return commands
	}

	tb.Layout()
	if tb.MeasuredW == 0 || tb.MeasuredH == 0 {
		return commands
	}

	if tb.SdfVertCount == 0 || len(tb.SdfVerts) == 0 {
		_, cellH := tb.Font.PixelCellSize()
		tb.RebuildLocalVerts(float64(cellH))
	}
	if tb.SdfVertCount == 0 {
		return commands
	}

	s := tb.FontScale()
	fst := [6]float64{s, 0, 0, s, 0, 0}
	scaledWT := node.MultiplyAffine(worldTransform, fst)

	atlasImg := resolveAtlasPage(tb.Font.PixelPage())
	if atlasImg == nil {
		return commands
	}

	*treeOrder++
	commands = append(commands, RenderCommand{
		Type:      CommandBitmapText,
		Transform: Affine32(scaledWT),
		Color: Color32{
			float32(tb.Color.R() * n.Color_.R()),
			float32(tb.Color.G() * n.Color_.G()),
			float32(tb.Color.B() * n.Color_.B()),
			float32(tb.Color.A() * n.Color_.A() * n.WorldAlpha),
		},
		BlendMode:    n.BlendMode_,
		RenderLayer:  n.RenderLayer,
		GlobalOrder:  n.GlobalOrder,
		TreeOrder:    *treeOrder,
		BmpVerts:     tb.SdfVerts,
		BmpInds:      tb.SdfInds,
		BmpVertCount: tb.SdfVertCount,
		BmpIndCount:  tb.SdfIndCount,
		BmpImage:     atlasImg,
	})

	return commands
}
