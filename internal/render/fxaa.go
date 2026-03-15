package render

import (
	"github.com/hajimehoshi/ebiten/v2"
)

// FXAAConfig holds tunable parameters for the FXAA post-process pass.
type FXAAConfig struct {
	// SubpixQuality controls sub-pixel aliasing removal (0.0 = off, 1.0 = max). Default: 0.75.
	SubpixQuality float32
	// EdgeThreshold is the minimum local contrast required to trigger AA.
	// Lower catches more edges but increases blur. Default: 0.166.
	EdgeThreshold float32
	// EdgeThresholdMin skips AA on pixels below this darkness threshold.
	// Prevents noise in shadowed areas. Default: 0.0312.
	EdgeThresholdMin float32
}

// DefaultFXAAConfig returns an FXAAConfig with the FXAA 3.11 quality-15 defaults.
func DefaultFXAAConfig() FXAAConfig {
	return FXAAConfig{
		SubpixQuality:    0.75,
		EdgeThreshold:    0.166,
		EdgeThresholdMin: 0.0312,
	}
}

// fxaaShaderSrc is a FXAA 3.11 quality-15 implementation in Kage.
// All sampling is in pixel coordinates (//kage:unit pixels).
// SubpixQuality, EdgeThreshold, and EdgeThresholdMin are passed as uniforms.
const fxaaShaderSrc = `//kage:unit pixels
package main

var SubpixQuality    float
var EdgeThreshold    float
var EdgeThresholdMin float

func luma(c vec4) float {
	return sqrt(dot(c.rgb, vec3(0.299, 0.587, 0.114)))
}

func Fragment(dst vec4, src vec2, color vec4) vec4 {
	colorC := imageSrc0UnsafeAt(src)
	lumaC  := luma(colorC)
	lumaN  := luma(imageSrc0UnsafeAt(src + vec2( 0, -1)))
	lumaS  := luma(imageSrc0UnsafeAt(src + vec2( 0,  1)))
	lumaW  := luma(imageSrc0UnsafeAt(src + vec2(-1,  0)))
	lumaE  := luma(imageSrc0UnsafeAt(src + vec2( 1,  0)))

	lumaMax   := max(lumaC, max(max(lumaN, lumaS), max(lumaW, lumaE)))
	lumaMin   := min(lumaC, min(min(lumaN, lumaS), min(lumaW, lumaE)))
	lumaRange := lumaMax - lumaMin

	if lumaRange < max(EdgeThresholdMin, lumaMax*EdgeThreshold) {
		return colorC
	}

	lumaNW := luma(imageSrc0UnsafeAt(src + vec2(-1, -1)))
	lumaNE := luma(imageSrc0UnsafeAt(src + vec2( 1, -1)))
	lumaSW := luma(imageSrc0UnsafeAt(src + vec2(-1,  1)))
	lumaSE := luma(imageSrc0UnsafeAt(src + vec2( 1,  1)))

	// Sub-pixel aliasing factor.
	lumaL    := (2.0*(lumaN+lumaS+lumaW+lumaE) + lumaNW+lumaNE+lumaSW+lumaSE) * (1.0 / 12.0)
	subpixOff := clamp(abs(lumaL-lumaC)/lumaRange, 0.0, 1.0)
	subpixOff  = (-2.0*subpixOff + 3.0) * subpixOff * subpixOff
	subpixOff  = subpixOff * subpixOff * SubpixQuality

	// Determine edge orientation.
	edgeH := abs(lumaNW+-2.0*lumaN+lumaNE) +
		abs(lumaW+-2.0*lumaC+lumaE)*2.0 +
		abs(lumaSW+-2.0*lumaS+lumaSE)
	edgeV := abs(lumaNW+-2.0*lumaW+lumaSW) +
		abs(lumaN+-2.0*lumaC+lumaS)*2.0 +
		abs(lumaNE+-2.0*lumaE+lumaSE)
	isHorz := edgeH >= edgeV

	// Perpendicular neighbor lumas.
	// For horizontal edge: luma1=-Y (north), luma2=+Y (south).
	// For vertical edge:   luma1=-X (west),  luma2=+X (east).
	var luma1, luma2 float
	if isHorz {
		luma1 = lumaN
		luma2 = lumaS
	} else {
		luma1 = lumaW
		luma2 = lumaE
	}
	grad1   := abs(luma1 - lumaC)
	grad2   := abs(luma2 - lumaC)
	gradMax := 0.25 * max(grad1, grad2)

	// stepLen=+1 is in the +perpendicular direction.
	// When luma1 (negative direction) is steeper, negate to walk that way.
	var stepLen float = 1.0
	lumaAvg := 0.5 * (luma2 + lumaC)
	if grad1 >= grad2 {
		stepLen = -1.0
		lumaAvg = 0.5 * (luma1 + lumaC)
	}

	// Starting UV for the edge walk: offset 0.5 px perpendicular to edge.
	var walkPos vec2
	var step vec2
	if isHorz {
		walkPos = vec2(src.x, src.y+stepLen*0.5)
		step    = vec2(1, 0)
	} else {
		walkPos = vec2(src.x+stepLen*0.5, src.y)
		step    = vec2(0, 1)
	}

	// Walk along the edge in both directions to find endpoints.
	p1 := walkPos - step
	p2 := walkPos + step
	end1 := luma(imageSrc0UnsafeAt(p1)) - lumaAvg
	end2 := luma(imageSrc0UnsafeAt(p2)) - lumaAvg
	done1 := abs(end1) >= gradMax
	done2 := abs(end2) >= gradMax

	// 8 iterations, unrolled — steps: 1,1,1.5,2,2,2,4,8
	if !done1 { p1 -= step;     end1 = luma(imageSrc0UnsafeAt(p1)) - lumaAvg; done1 = abs(end1) >= gradMax }
	if !done2 { p2 += step;     end2 = luma(imageSrc0UnsafeAt(p2)) - lumaAvg; done2 = abs(end2) >= gradMax }
	if !done1 { p1 -= step;     end1 = luma(imageSrc0UnsafeAt(p1)) - lumaAvg; done1 = abs(end1) >= gradMax }
	if !done2 { p2 += step;     end2 = luma(imageSrc0UnsafeAt(p2)) - lumaAvg; done2 = abs(end2) >= gradMax }
	if !done1 { p1 -= step*1.5; end1 = luma(imageSrc0UnsafeAt(p1)) - lumaAvg; done1 = abs(end1) >= gradMax }
	if !done2 { p2 += step*1.5; end2 = luma(imageSrc0UnsafeAt(p2)) - lumaAvg; done2 = abs(end2) >= gradMax }
	if !done1 { p1 -= step*2.0; end1 = luma(imageSrc0UnsafeAt(p1)) - lumaAvg; done1 = abs(end1) >= gradMax }
	if !done2 { p2 += step*2.0; end2 = luma(imageSrc0UnsafeAt(p2)) - lumaAvg; done2 = abs(end2) >= gradMax }
	if !done1 { p1 -= step*2.0; end1 = luma(imageSrc0UnsafeAt(p1)) - lumaAvg; done1 = abs(end1) >= gradMax }
	if !done2 { p2 += step*2.0; end2 = luma(imageSrc0UnsafeAt(p2)) - lumaAvg; done2 = abs(end2) >= gradMax }
	if !done1 { p1 -= step*2.0; end1 = luma(imageSrc0UnsafeAt(p1)) - lumaAvg; done1 = abs(end1) >= gradMax }
	if !done2 { p2 += step*2.0; end2 = luma(imageSrc0UnsafeAt(p2)) - lumaAvg; done2 = abs(end2) >= gradMax }
	if !done1 { p1 -= step*4.0; end1 = luma(imageSrc0UnsafeAt(p1)) - lumaAvg; done1 = abs(end1) >= gradMax }
	if !done2 { p2 += step*4.0; end2 = luma(imageSrc0UnsafeAt(p2)) - lumaAvg; done2 = abs(end2) >= gradMax }
	if !done1 { p1 -= step*8.0 }
	if !done2 { p2 += step*8.0 }
	_ = done1
	_ = done2

	// Distances from center to each endpoint along the edge.
	var dst1, dst2 float
	if isHorz {
		dst1 = src.x - p1.x
		dst2 = p2.x - src.x
	} else {
		dst1 = src.y - p1.y
		dst2 = p2.y - src.y
	}

	// Check that the endpoint luma confirms the edge direction.
	lumaCLessAvg := lumaC < lumaAvg
	good1 := (end1 < 0) != lumaCLessAvg
	good2 := (end2 < 0) != lumaCLessAvg

	// Pixel offset toward the edge midpoint (0 if no good span found).
	var pixOffset float
	if (dst1 < dst2 && good1) || (dst2 <= dst1 && good2) {
		pixOffset = 0.5 - min(dst1, dst2)/(dst1+dst2)
	}
	pixOffset = max(pixOffset, subpixOff)

	var finalPos vec2
	if isHorz {
		finalPos = vec2(src.x, src.y+pixOffset*stepLen)
	} else {
		finalPos = vec2(src.x+pixOffset*stepLen, src.y)
	}
	return imageSrc0UnsafeAt(finalPos)
}
`

var fxaaShader *ebiten.Shader

// fxaaUniforms is a persistent map reused every frame to avoid allocation.
var fxaaUniforms = map[string]any{
	"SubpixQuality":    float32(0),
	"EdgeThreshold":    float32(0),
	"EdgeThresholdMin": float32(0),
}

// EnsureFXAAShader lazily compiles and returns the FXAA screen shader.
func EnsureFXAAShader() *ebiten.Shader {
	if fxaaShader == nil {
		s, err := ebiten.NewShader([]byte(fxaaShaderSrc))
		if err != nil {
			panic("willow: failed to compile FXAA shader: " + err.Error())
		}
		fxaaShader = s
	}
	return fxaaShader
}

// DrawFinalScreenFXAA applies the FXAA shader when drawing the final screen.
func DrawFinalScreenFXAA(screen ebiten.FinalScreen, offscreen *ebiten.Image, geoM ebiten.GeoM, cfg FXAAConfig) {
	shader := EnsureFXAAShader()
	fxaaUniforms["SubpixQuality"] = cfg.SubpixQuality
	fxaaUniforms["EdgeThreshold"] = cfg.EdgeThreshold
	fxaaUniforms["EdgeThresholdMin"] = cfg.EdgeThresholdMin
	b := offscreen.Bounds()
	var op ebiten.DrawRectShaderOptions
	op.GeoM = geoM
	op.Images[0] = offscreen
	op.Uniforms = fxaaUniforms
	screen.DrawRectShader(b.Dx(), b.Dy(), shader, &op)
}
