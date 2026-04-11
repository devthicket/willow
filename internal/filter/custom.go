package filter

import "github.com/hajimehoshi/ebiten/v2"

// CustomShaderFilter wraps a user-provided Kage shader for spatial effects
// that require an offscreen render target (e.g. shaders that read neighbors,
// need padding, or use multiple passes). Images[0] is auto-filled with the
// source texture; the user may set Images[1] and Images[2] for additional
// textures.
type CustomShaderFilter struct {
	Shader   *ebiten.Shader
	Uniforms map[string]any
	Images   [3]*ebiten.Image
	padding  int
	shaderOp ebiten.DrawRectShaderOptions
}

// NewCustomShaderFilter creates a custom shader filter with the given shader and padding.
func NewCustomShaderFilter(shader *ebiten.Shader, padding int) *CustomShaderFilter {
	return &CustomShaderFilter{
		Shader:   shader,
		Uniforms: make(map[string]any),
		padding:  padding,
	}
}

// Apply runs the user-provided Kage shader with src as Images[0].
func (f *CustomShaderFilter) Apply(src, dst *ebiten.Image) {
	bounds := src.Bounds()
	f.shaderOp.Images[0] = src
	f.shaderOp.Images[1] = f.Images[1]
	f.shaderOp.Images[2] = f.Images[2]
	f.shaderOp.Uniforms = f.Uniforms
	dst.DrawRectShader(bounds.Dx(), bounds.Dy(), f.Shader, &f.shaderOp)
}

// Padding returns the padding value set at construction time.
func (f *CustomShaderFilter) Padding() int { return f.padding }

// CustomDrawShaderFilter wraps a user-provided Kage shader for per-pixel
// effects that can be applied at draw time without an offscreen render target.
// On leaf sprites this is drawn inline via DrawTrianglesShader, and
// consecutive sprites sharing the same filter instance are batched into a
// single draw call.
//
// Use this for per-pixel transforms (color grading, palette effects, etc.)
// that don't read neighboring pixels and don't need padding.
// For spatial effects that read neighbors, use CustomShaderFilter instead.
type CustomDrawShaderFilter struct {
	Shader   *ebiten.Shader
	Uniforms map[string]any
	Images   [3]*ebiten.Image
	shaderOp ebiten.DrawRectShaderOptions
}

// NewCustomDrawShaderFilter creates a draw-time custom shader filter.
// The shader must be a per-pixel effect (no neighbor reads, zero padding).
// If extra Images are used, they must match the atlas page size or the
// filter will fall back to the offscreen RT path on non-leaf nodes.
func NewCustomDrawShaderFilter(shader *ebiten.Shader) *CustomDrawShaderFilter {
	return &CustomDrawShaderFilter{
		Shader:   shader,
		Uniforms: make(map[string]any),
	}
}

// Apply runs the shader with src as Images[0] (offscreen RT fallback path).
func (f *CustomDrawShaderFilter) Apply(src, dst *ebiten.Image) {
	bounds := src.Bounds()
	f.shaderOp.Images[0] = src
	f.shaderOp.Images[1] = f.Images[1]
	f.shaderOp.Images[2] = f.Images[2]
	f.shaderOp.Uniforms = f.Uniforms
	dst.DrawRectShader(bounds.Dx(), bounds.Dy(), f.Shader, &f.shaderOp)
}

// Padding returns 0; draw-time filters don't expand bounds.
func (f *CustomDrawShaderFilter) Padding() int { return 0 }

// DrawShader returns the user-provided shader.
func (f *CustomDrawShaderFilter) DrawShader() *ebiten.Shader {
	return f.Shader
}

// DrawUniforms returns the user-provided uniforms.
func (f *CustomDrawShaderFilter) DrawUniforms() map[string]any {
	return f.Uniforms
}

// DrawImages returns the user-provided extra textures.
func (f *CustomDrawShaderFilter) DrawImages() [3]*ebiten.Image {
	return f.Images
}
