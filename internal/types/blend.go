package types

import "github.com/hajimehoshi/ebiten/v2"

// BlendMode selects a compositing operation. Each maps to a specific ebiten.Blend value.
type BlendMode uint8

const (
	BlendNormal   BlendMode = iota // source-over (standard alpha blending)
	BlendAdd                       // additive / lighter
	BlendMultiply                  // multiply (source * destination; only darkens)
	BlendScreen                    // screen (1 - (1-src)*(1-dst); only brightens)
	BlendErase                     // destination-out (punch transparent holes)
	BlendMask                      // clip destination to source alpha
	BlendBelow                     // destination-over (draw behind existing content)
	BlendNone                      // opaque copy (skip blending)
)

// EbitenBlend returns the ebiten.Blend value corresponding to this BlendMode.
func (b BlendMode) EbitenBlend() ebiten.Blend {
	switch b {
	case BlendNormal:
		return ebiten.BlendSourceOver
	case BlendAdd:
		return ebiten.BlendLighter
	case BlendMultiply:
		return ebiten.Blend{
			BlendFactorSourceRGB:        ebiten.BlendFactorDestinationColor,
			BlendFactorSourceAlpha:      ebiten.BlendFactorDestinationAlpha,
			BlendFactorDestinationRGB:   ebiten.BlendFactorOneMinusSourceAlpha,
			BlendFactorDestinationAlpha: ebiten.BlendFactorOneMinusSourceAlpha,
			BlendOperationRGB:           ebiten.BlendOperationAdd,
			BlendOperationAlpha:         ebiten.BlendOperationAdd,
		}
	case BlendScreen:
		return ebiten.Blend{
			BlendFactorSourceRGB:        ebiten.BlendFactorOne,
			BlendFactorSourceAlpha:      ebiten.BlendFactorOne,
			BlendFactorDestinationRGB:   ebiten.BlendFactorOneMinusSourceColor,
			BlendFactorDestinationAlpha: ebiten.BlendFactorOneMinusSourceAlpha,
			BlendOperationRGB:           ebiten.BlendOperationAdd,
			BlendOperationAlpha:         ebiten.BlendOperationAdd,
		}
	case BlendErase:
		return ebiten.BlendDestinationOut
	case BlendMask:
		return ebiten.Blend{
			BlendFactorSourceRGB:        ebiten.BlendFactorZero,
			BlendFactorSourceAlpha:      ebiten.BlendFactorZero,
			BlendFactorDestinationRGB:   ebiten.BlendFactorSourceAlpha,
			BlendFactorDestinationAlpha: ebiten.BlendFactorSourceAlpha,
			BlendOperationRGB:           ebiten.BlendOperationAdd,
			BlendOperationAlpha:         ebiten.BlendOperationAdd,
		}
	case BlendBelow:
		return ebiten.BlendDestinationOver
	case BlendNone:
		return ebiten.BlendCopy
	default:
		return ebiten.BlendSourceOver
	}
}
