package filter

import "github.com/hajimehoshi/ebiten/v2"

// CustomShaderFilter wraps a user-provided Kage shader, exposing Ebitengine's
// shader system directly. Images[0] is auto-filled with the source texture;
// the user may set Images[1] and Images[2] for additional textures.
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
