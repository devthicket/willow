package types

import (
	"github.com/tanema/gween/ease"
)

// TweenConfig holds the duration and easing function for a tween.
// A nil Ease defaults to ease.Linear. Duration is in seconds; 0 means instant.
type TweenConfig struct {
	Duration float32        // seconds; 0 = instant
	Ease     ease.TweenFunc // nil defaults to ease.Linear
}

// TweenEase returns cfg.Ease if non-nil, otherwise ease.Linear.
func TweenEase(cfg TweenConfig) ease.TweenFunc {
	if cfg.Ease != nil {
		return cfg.Ease
	}
	return ease.Linear
}
