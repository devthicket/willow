package node

import "github.com/devthicket/willow/internal/types"

// AnimationSequence defines a single named animation.
type AnimationSequence struct {
	Name   string
	Frames []types.TextureRegion
	FPS    float32
	Loop   bool
}

// AnimationPlayer manages multiple named animation sequences on a node.
type AnimationPlayer struct {
	node       *Node
	sequences  map[string]*AnimationSequence
	current    string
	frame      int
	elapsed    float32
	playing    bool
	onComplete func(name string)
}

// NewAnimationPlayer creates an AnimationPlayer bound to the given node.
func NewAnimationPlayer(n *Node) *AnimationPlayer {
	return &AnimationPlayer{
		node:      n,
		sequences: make(map[string]*AnimationSequence),
	}
}

// AddSequence registers a named animation sequence.
func (ap *AnimationPlayer) AddSequence(seq AnimationSequence) {
	s := seq // copy
	ap.sequences[seq.Name] = &s
}

// Play switches to the named sequence. No-op if already playing it.
// Does nothing if the sequence name is not registered.
func (ap *AnimationPlayer) Play(name string) {
	if ap.current == name && ap.playing {
		return
	}
	seq, ok := ap.sequences[name]
	if !ok || len(seq.Frames) == 0 {
		return
	}
	ap.current = name
	ap.frame = 0
	ap.elapsed = 0
	ap.playing = true
	ap.node.SetTextureRegion(seq.Frames[0])
}

// PlayFrom switches to the named sequence starting at a specific frame.
func (ap *AnimationPlayer) PlayFrom(name string, frame int) {
	seq, ok := ap.sequences[name]
	if !ok || len(seq.Frames) == 0 {
		return
	}
	ap.current = name
	if frame < 0 {
		frame = 0
	}
	if frame >= len(seq.Frames) {
		frame = len(seq.Frames) - 1
	}
	ap.frame = frame
	ap.elapsed = 0
	ap.playing = true
	ap.node.SetTextureRegion(seq.Frames[frame])
}

// Stop stops the current animation at the current frame.
func (ap *AnimationPlayer) Stop() {
	ap.playing = false
}

// Current returns the active sequence name.
func (ap *AnimationPlayer) Current() string {
	return ap.current
}

// Frame returns the current frame index.
func (ap *AnimationPlayer) Frame() int {
	return ap.frame
}

// IsPlaying reports whether the current sequence is still running.
func (ap *AnimationPlayer) IsPlaying() bool {
	return ap.playing
}

// OnComplete sets a callback invoked when a non-looping sequence finishes.
func (ap *AnimationPlayer) OnComplete(fn func(name string)) {
	ap.onComplete = fn
}

// Update advances the animation by dt seconds. Call from the game loop.
func (ap *AnimationPlayer) Update(dt float32) {
	if !ap.playing || ap.current == "" {
		return
	}
	seq, ok := ap.sequences[ap.current]
	if !ok || len(seq.Frames) == 0 {
		return
	}
	if seq.FPS <= 0 {
		return
	}

	ap.elapsed += dt
	frameDuration := 1.0 / seq.FPS

	for ap.elapsed >= frameDuration {
		ap.elapsed -= frameDuration
		ap.frame++

		if ap.frame >= len(seq.Frames) {
			if seq.Loop {
				ap.frame = 0
			} else {
				ap.frame = len(seq.Frames) - 1
				ap.playing = false
				ap.node.SetTextureRegion(seq.Frames[ap.frame])
				if ap.onComplete != nil {
					ap.onComplete(ap.current)
				}
				return
			}
		}

		ap.node.SetTextureRegion(seq.Frames[ap.frame])
	}
}
