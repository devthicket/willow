package willow

import "github.com/hajimehoshi/ebiten/v2"

// --- SDF Kage shader sources ---

// sdfShaderSrc is the single-channel SDF fragment shader.
// Reads distance from the alpha channel of the atlas glyph.
// Produces fill, outline, glow, and drop shadow in a single pass.
const sdfShaderSrc = `//kage:unit pixels
package main

// Edge threshold for fill (0.5 = exact edge in normalized distance).
var Threshold float
// Smoothing width in distance-field units (controls anti-aliasing sharpness).
var Smoothing float
// Outline width in distance-field units (0 = no outline).
var OutlineWidth float
// Outline color (premultiplied RGBA).
var OutlineColor vec4
// Glow width in distance-field units (0 = no glow).
var GlowWidth float
// Glow color (premultiplied RGBA).
var GlowColor vec4
// Shadow offset in pixels.
var ShadowOffset vec2
// Shadow color (premultiplied RGBA).
var ShadowColor vec4
// Shadow softness in distance-field units.
var ShadowSoftness float
// Fill color (premultiplied RGBA).
var FillColor vec4

func Fragment(dst vec4, src vec2, color vec4) vec4 {
	dist := imageSrc0At(src).a

	// Combine per-pixel derivatives with scale-aware uniform for optimal smoothing.
	// fwidth gives per-texel sharpness; Smoothing provides the scale-aware floor.
	sm := max(fwidth(dist), Smoothing)

	// --- Shadow ---
	var result vec4
	if ShadowColor.a > 0 && (ShadowOffset.x != 0 || ShadowOffset.y != 0 || ShadowSoftness > 0) {
		shadowDist := imageSrc0At(src - ShadowOffset).a
		shadowAlpha := smoothstep(Threshold - ShadowSoftness - sm, Threshold - ShadowSoftness + sm, shadowDist)
		result = ShadowColor * shadowAlpha
	}

	// --- Glow ---
	if GlowColor.a > 0 && GlowWidth > 0 {
		glowEdge := Threshold - OutlineWidth - GlowWidth
		glowAlpha := smoothstep(glowEdge - sm, glowEdge + sm, dist)
		result = mix(result, GlowColor, GlowColor.a * glowAlpha)
	}

	// --- Outline ---
	if OutlineColor.a > 0 && OutlineWidth > 0 {
		outerEdge := Threshold - OutlineWidth
		outlineAlpha := smoothstep(outerEdge - sm, outerEdge + sm, dist)
		result = mix(result, OutlineColor, OutlineColor.a * outlineAlpha)
	}

	// --- Fill ---
	fillAlpha := smoothstep(Threshold - sm, Threshold + sm, dist)
	result = mix(result, FillColor, FillColor.a * fillAlpha)

	// Fade all effects to transparent at the SDF range boundary (dist near 0)
	// to prevent visible rectangles where the distance field runs out of data.
	result *= smoothstep(0, sm*3, dist)

	return result
}
`

// msdfShaderSrc is the multi-channel SDF fragment shader.
// Computes median(R, G, B) for the distance value, then applies the same
// threshold logic as the single-channel shader.
const msdfShaderSrc = `//kage:unit pixels
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

func median3(a, b, c float) float {
	return max(min(a, b), min(max(a, b), c))
}

func sampleDist(pos vec2) float {
	s := imageSrc0At(pos)
	return median3(s.r, s.g, s.b)
}

func Fragment(dst vec4, src vec2, color vec4) vec4 {
	dist := sampleDist(src)

	// Combine per-pixel derivatives with scale-aware uniform for optimal smoothing.
	sm := max(fwidth(dist), Smoothing)

	// --- Shadow ---
	var result vec4
	if ShadowColor.a > 0 && (ShadowOffset.x != 0 || ShadowOffset.y != 0 || ShadowSoftness > 0) {
		shadowDist := sampleDist(src - ShadowOffset)
		shadowAlpha := smoothstep(Threshold - ShadowSoftness - sm, Threshold - ShadowSoftness + sm, shadowDist)
		result = ShadowColor * shadowAlpha
	}

	// --- Glow ---
	if GlowColor.a > 0 && GlowWidth > 0 {
		glowEdge := Threshold - OutlineWidth - GlowWidth
		glowAlpha := smoothstep(glowEdge - sm, glowEdge + sm, dist)
		result = mix(result, GlowColor, GlowColor.a * glowAlpha)
	}

	// --- Outline ---
	if OutlineColor.a > 0 && OutlineWidth > 0 {
		outerEdge := Threshold - OutlineWidth
		outlineAlpha := smoothstep(outerEdge - sm, outerEdge + sm, dist)
		result = mix(result, OutlineColor, OutlineColor.a * outlineAlpha)
	}

	// --- Fill ---
	fillAlpha := smoothstep(Threshold - sm, Threshold + sm, dist)
	result = mix(result, FillColor, FillColor.a * fillAlpha)

	// Fade all effects to transparent at the SDF range boundary (dist near 0)
	// to prevent visible rectangles where the distance field runs out of data.
	result *= smoothstep(0, sm*3, dist)

	return result
}
`

// --- Lazy shader compilation ---

var (
	sdfShader  *ebiten.Shader
	msdfShader *ebiten.Shader
)

func ensureSDFShader() *ebiten.Shader {
	if sdfShader == nil {
		s, err := ebiten.NewShader([]byte(sdfShaderSrc))
		if err != nil {
			panic("willow: failed to compile SDF shader: " + err.Error())
		}
		sdfShader = s
	}
	return sdfShader
}

func ensureMSDFShader() *ebiten.Shader {
	if msdfShader == nil {
		s, err := ebiten.NewShader([]byte(msdfShaderSrc))
		if err != nil {
			panic("willow: failed to compile MSDF shader: " + err.Error())
		}
		msdfShader = s
	}
	return msdfShader
}
