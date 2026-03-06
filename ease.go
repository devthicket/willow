package willow

import "github.com/tanema/gween/ease"

// EaseFunc is the signature for easing functions used by TweenConfig.
// This is a re-export of ease.TweenFunc so users don't need to import
// the gween/ease package directly.
type EaseFunc = ease.TweenFunc

// Easing functions re-exported from gween/ease for discoverability.
// Type "willow.Ease" in your editor to see all available options.
var (
	EaseLinear EaseFunc = ease.Linear

	EaseInQuad    EaseFunc = ease.InQuad
	EaseOutQuad   EaseFunc = ease.OutQuad
	EaseInOutQuad EaseFunc = ease.InOutQuad
	EaseOutInQuad EaseFunc = ease.OutInQuad

	EaseInCubic    EaseFunc = ease.InCubic
	EaseOutCubic   EaseFunc = ease.OutCubic
	EaseInOutCubic EaseFunc = ease.InOutCubic
	EaseOutInCubic EaseFunc = ease.OutInCubic

	EaseInQuart    EaseFunc = ease.InQuart
	EaseOutQuart   EaseFunc = ease.OutQuart
	EaseInOutQuart EaseFunc = ease.InOutQuart
	EaseOutInQuart EaseFunc = ease.OutInQuart

	EaseInQuint    EaseFunc = ease.InQuint
	EaseOutQuint   EaseFunc = ease.OutQuint
	EaseInOutQuint EaseFunc = ease.InOutQuint
	EaseOutInQuint EaseFunc = ease.OutInQuint

	EaseInSine    EaseFunc = ease.InSine
	EaseOutSine   EaseFunc = ease.OutSine
	EaseInOutSine EaseFunc = ease.InOutSine
	EaseOutInSine EaseFunc = ease.OutInSine

	EaseInExpo    EaseFunc = ease.InExpo
	EaseOutExpo   EaseFunc = ease.OutExpo
	EaseInOutExpo EaseFunc = ease.InOutExpo
	EaseOutInExpo EaseFunc = ease.OutInExpo

	EaseInCirc    EaseFunc = ease.InCirc
	EaseOutCirc   EaseFunc = ease.OutCirc
	EaseInOutCirc EaseFunc = ease.InOutCirc
	EaseOutInCirc EaseFunc = ease.OutInCirc

	EaseInElastic    EaseFunc = ease.InElastic
	EaseOutElastic   EaseFunc = ease.OutElastic
	EaseInOutElastic EaseFunc = ease.InOutElastic
	EaseOutInElastic EaseFunc = ease.OutInElastic

	EaseInBack    EaseFunc = ease.InBack
	EaseOutBack   EaseFunc = ease.OutBack
	EaseInOutBack EaseFunc = ease.InOutBack
	EaseOutInBack EaseFunc = ease.OutInBack

	EaseInBounce    EaseFunc = ease.InBounce
	EaseOutBounce   EaseFunc = ease.OutBounce
	EaseInOutBounce EaseFunc = ease.InOutBounce
	EaseOutInBounce EaseFunc = ease.OutInBounce
)
