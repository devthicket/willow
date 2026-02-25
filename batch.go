package willow

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
)

// batchKey groups render commands that can be submitted in a single draw call.
type batchKey struct {
	targetID uint16
	shaderID uint16
	blend    BlendMode
	page     uint16
}

func commandBatchKey(cmd *RenderCommand) batchKey {
	return batchKey{
		targetID: cmd.TargetID,
		shaderID: cmd.ShaderID,
		blend:    cmd.BlendMode,
		page:     cmd.TextureRegion.Page,
	}
}

// submitBatches iterates sorted commands, groups them by batch key, and submits
// draw calls to the target image.
func (s *Scene) submitBatches(target *ebiten.Image) {
	if len(s.commands) == 0 {
		return
	}

	var op ebiten.DrawImageOptions

	for i := range s.commands {
		cmd := &s.commands[i]

		switch cmd.Type {
		case CommandSprite:
			s.submitSprite(target, cmd, &op)
		case CommandParticle:
			s.submitParticles(target, cmd, &op)
		case CommandMesh:
			s.submitMesh(target, cmd)
		case CommandTilemap:
			s.submitTilemap(target, cmd)
		}
	}
}

// submitSprite draws a single sprite command using DrawImage.
func (s *Scene) submitSprite(target *ebiten.Image, cmd *RenderCommand, op *ebiten.DrawImageOptions) {
	// Direct image path: draw a pre-rendered offscreen texture directly.
	if cmd.directImage != nil {
		op.GeoM.Reset()
		op.GeoM.Concat(commandGeoM(cmd))
		op.ColorScale.Reset()
		a := cmd.Color.A
		if a == 0 && cmd.Color.R == 0 && cmd.Color.G == 0 && cmd.Color.B == 0 {
			a = 1
			op.ColorScale.Scale(1, 1, 1, 1)
		} else {
			op.ColorScale.Scale(cmd.Color.R*a, cmd.Color.G*a, cmd.Color.B*a, a)
		}
		op.Blend = cmd.BlendMode.EbitenBlend()
		op.Filter = ebiten.FilterLinear
		target.DrawImage(cmd.directImage, op)
		return
	}

	r := &cmd.TextureRegion

	// Resolve the atlas page image.
	var page *ebiten.Image
	if r.Page == magentaPlaceholderPage {
		page = ensureMagentaImage()
	} else {
		page = atlasManager().Page(int(r.Page))
	}
	if page == nil {
		return
	}

	// Compute SubImage rect
	var subRect image.Rectangle
	if r.Rotated {
		subRect = image.Rect(int(r.X), int(r.Y), int(r.X)+int(r.Height), int(r.Y)+int(r.Width))
	} else {
		subRect = image.Rect(int(r.X), int(r.Y), int(r.X)+int(r.Width), int(r.Y)+int(r.Height))
	}
	subImg := page.SubImage(subRect).(*ebiten.Image)

	op.GeoM.Reset()

	// Handle rotated regions: rotate -90° and shift
	if r.Rotated {
		// Rotated regions in atlas are stored rotated 90° CW.
		// To draw correctly: rotate -90° (CCW) then shift right by height.
		op.GeoM.Rotate(-1.5707963267948966) // -π/2
		op.GeoM.Translate(0, float64(r.Width))
	}

	// Apply trim offset
	if r.OffsetX != 0 || r.OffsetY != 0 {
		op.GeoM.Translate(float64(r.OffsetX), float64(r.OffsetY))
	}

	// Apply world transform
	op.GeoM.Concat(commandGeoM(cmd))

	// Apply premultiplied color scale
	op.ColorScale.Reset()
	a := cmd.Color.A
	if a == 0 && cmd.Color.R == 0 && cmd.Color.G == 0 && cmd.Color.B == 0 {
		a = 1
		op.ColorScale.Scale(1, 1, 1, 1)
	} else {
		op.ColorScale.Scale(cmd.Color.R*a, cmd.Color.G*a, cmd.Color.B*a, a)
	}

	op.Blend = cmd.BlendMode.EbitenBlend()

	target.DrawImage(subImg, op)
}

// submitParticles draws all alive particles for a CommandParticle command.
func (s *Scene) submitParticles(target *ebiten.Image, cmd *RenderCommand, op *ebiten.DrawImageOptions) {
	e := cmd.emitter
	if e == nil || e.alive == 0 {
		return
	}

	r := &cmd.TextureRegion

	// Resolve the particle texture. A directImage (e.g. WhitePixel) takes
	// priority; otherwise fall back to an atlas page.
	var subImg *ebiten.Image
	if cmd.directImage != nil {
		subImg = cmd.directImage
	} else {
		var page *ebiten.Image
		if r.Page == magentaPlaceholderPage {
			page = ensureMagentaImage()
		} else {
			page = atlasManager().Page(int(r.Page))
		}
		if page == nil {
			return
		}
		var subRect image.Rectangle
		if r.Rotated {
			subRect = image.Rect(int(r.X), int(r.Y), int(r.X)+int(r.Height), int(r.Y)+int(r.Width))
		} else {
			subRect = image.Rect(int(r.X), int(r.Y), int(r.X)+int(r.Width), int(r.Y)+int(r.Height))
		}
		subImg = page.SubImage(subRect).(*ebiten.Image)
	}

	// Transform for positioning: world transform for attached, view-only for world-space.
	baseGeoM := commandGeoM(cmd)

	for i := 0; i < e.alive; i++ {
		p := &e.particles[i]

		op.GeoM.Reset()

		// Handle rotated regions.
		if r.Rotated {
			op.GeoM.Rotate(-1.5707963267948966) // -π/2
			op.GeoM.Translate(0, float64(r.Width))
		}

		// Apply trim offset.
		if r.OffsetX != 0 || r.OffsetY != 0 {
			op.GeoM.Translate(float64(r.OffsetX), float64(r.OffsetY))
		}

		// Per-particle scale (around sprite center).
		op.GeoM.Translate(-float64(r.OriginalW)/2, -float64(r.OriginalH)/2)
		op.GeoM.Scale(float64(p.scale), float64(p.scale))
		op.GeoM.Translate(float64(r.OriginalW)/2, float64(r.OriginalH)/2)

		// Per-particle translation.
		// For world-space particles, (p.x, p.y) are absolute world coords.
		// For attached particles, (p.x, p.y) are relative to the emitter.
		op.GeoM.Translate(p.x, p.y)

		// Apply base transform (emitter world transform or view-only transform).
		op.GeoM.Concat(baseGeoM)

		// Per-particle color: particle color * emitter node color, particle alpha * emitter worldAlpha.
		cr := p.colorR * cmd.Color.R
		cg := p.colorG * cmd.Color.G
		cb := p.colorB * cmd.Color.B
		ca := p.alpha * cmd.Color.A
		op.ColorScale.Reset()
		op.ColorScale.Scale(cr*ca, cg*ca, cb*ca, ca)

		op.Blend = cmd.BlendMode.EbitenBlend()

		target.DrawImage(subImg, op)
	}
}

// submitTilemap draws a tilemap layer command using DrawTriangles.
func (s *Scene) submitTilemap(target *ebiten.Image, cmd *RenderCommand) {
	if cmd.tilemapImage == nil || len(cmd.tilemapVerts) == 0 || len(cmd.tilemapInds) == 0 {
		return
	}
	var triOp ebiten.DrawTrianglesOptions
	triOp.Blend = cmd.BlendMode.EbitenBlend()
	triOp.ColorScaleMode = ebiten.ColorScaleModePremultipliedAlpha
	target.DrawTriangles(cmd.tilemapVerts, cmd.tilemapInds, cmd.tilemapImage, &triOp)
}

// submitMesh draws a mesh command using DrawTriangles.
func (s *Scene) submitMesh(target *ebiten.Image, cmd *RenderCommand) {
	if cmd.meshImage == nil || len(cmd.meshVerts) == 0 || len(cmd.meshInds) == 0 {
		return
	}

	var triOp ebiten.DrawTrianglesOptions
	triOp.Blend = cmd.BlendMode.EbitenBlend()

	target.DrawTriangles(cmd.meshVerts, cmd.meshInds, cmd.meshImage, &triOp)
}

// commandGeoM converts a command's [6]float64 transform into an ebiten.GeoM.
func commandGeoM(cmd *RenderCommand) ebiten.GeoM {
	var m ebiten.GeoM
	m.SetElement(0, 0, float64(cmd.Transform[0]))
	m.SetElement(1, 0, float64(cmd.Transform[1]))
	m.SetElement(0, 1, float64(cmd.Transform[2]))
	m.SetElement(1, 1, float64(cmd.Transform[3]))
	m.SetElement(0, 2, float64(cmd.Transform[4]))
	m.SetElement(1, 2, float64(cmd.Transform[5]))
	return m
}

// --- Coalesced batching (BatchModeCoalesced) ---

// submitBatchesCoalesced iterates sorted commands, coalescing consecutive
// same-key atlas sprites into a single DrawTriangles32 call.
func (s *Scene) submitBatchesCoalesced(target *ebiten.Image) {
	if len(s.commands) == 0 {
		return
	}

	s.batchVerts = s.batchVerts[:0]
	s.batchInds = s.batchInds[:0]

	var currentKey batchKey
	inRun := false
	var op ebiten.DrawImageOptions

	for i := range s.commands {
		cmd := &s.commands[i]

		switch cmd.Type {
		case CommandSprite:
			if cmd.directImage != nil {
				// Direct-image sprites cannot be coalesced (different source images).
				s.flushSpriteBatch(target, currentKey)
				inRun = false
				s.submitSprite(target, cmd, &op)
				continue
			}

			key := commandBatchKey(cmd)
			if inRun && key != currentKey {
				s.flushSpriteBatch(target, currentKey)
			}
			currentKey = key
			inRun = true
			s.appendSpriteQuad(cmd)

		case CommandParticle:
			s.flushSpriteBatch(target, currentKey)
			inRun = false
			s.submitParticlesBatched(target, cmd)

		case CommandMesh:
			s.flushSpriteBatch(target, currentKey)
			inRun = false
			s.submitMesh(target, cmd)

		case CommandTilemap:
			s.flushSpriteBatch(target, currentKey)
			inRun = false
			s.submitTilemap(target, cmd)
		}
	}

	s.flushSpriteBatch(target, currentKey)
}

// appendSpriteQuad appends 4 vertices and 6 indices for a single atlas sprite.
func (s *Scene) appendSpriteQuad(cmd *RenderCommand) {
	r := &cmd.TextureRegion
	t := &cmd.Transform // [a, b, c, d, tx, ty]

	// Local quad corners before world transform.
	ox := float32(r.OffsetX)
	oy := float32(r.OffsetY)
	w := float32(r.Width)
	h := float32(r.Height)

	// Affine transform components.
	a, b, c, d, tx, ty := t[0], t[1], t[2], t[3], t[4], t[5]

	// Precompute local corner positions: TL, TR, BL, BR.
	x0, y0 := ox, oy     // TL
	x1, y1 := ox+w, oy   // TR
	x2, y2 := ox, oy+h   // BL
	x3, y3 := ox+w, oy+h // BR

	// Source UVs (pixel coordinates on the atlas page).
	var sx0, sy0, sx1, sy1, sx2, sy2, sx3, sy3 float32
	if r.Rotated {
		rx := float32(r.X)
		ry := float32(r.Y)
		rh := float32(r.Height) // stored width in atlas
		rw := float32(r.Width)  // stored height in atlas
		sx0, sy0 = rx+rh, ry    // TL
		sx1, sy1 = rx+rh, ry+rw // TR
		sx2, sy2 = rx, ry       // BL
		sx3, sy3 = rx, ry+rw    // BR
	} else {
		rx := float32(r.X)
		ry := float32(r.Y)
		rw := float32(r.Width)
		rh := float32(r.Height)
		sx0, sy0 = rx, ry       // TL
		sx1, sy1 = rx+rw, ry    // TR
		sx2, sy2 = rx, ry+rh    // BL
		sx3, sy3 = rx+rw, ry+rh // BR
	}

	// Premultiplied RGBA. Zero-color sentinel → opaque white.
	var cr, cg, cb, ca float32
	ca = cmd.Color.A
	if ca == 0 && cmd.Color.R == 0 && cmd.Color.G == 0 && cmd.Color.B == 0 {
		cr, cg, cb, ca = 1, 1, 1, 1
	} else {
		cr = cmd.Color.R * ca
		cg = cmd.Color.G * ca
		cb = cmd.Color.B * ca
	}

	base := uint32(len(s.batchVerts))

	// Inline 4 vertex computations (no loop, no intermediate arrays).
	s.batchVerts = append(s.batchVerts,
		ebiten.Vertex{
			DstX: a*x0 + c*y0 + tx, DstY: b*x0 + d*y0 + ty,
			SrcX: sx0, SrcY: sy0,
			ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca,
		},
		ebiten.Vertex{
			DstX: a*x1 + c*y1 + tx, DstY: b*x1 + d*y1 + ty,
			SrcX: sx1, SrcY: sy1,
			ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca,
		},
		ebiten.Vertex{
			DstX: a*x2 + c*y2 + tx, DstY: b*x2 + d*y2 + ty,
			SrcX: sx2, SrcY: sy2,
			ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca,
		},
		ebiten.Vertex{
			DstX: a*x3 + c*y3 + tx, DstY: b*x3 + d*y3 + ty,
			SrcX: sx3, SrcY: sy3,
			ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca,
		},
	)

	// Two triangles: TL-TR-BL, TR-BR-BL
	s.batchInds = append(s.batchInds,
		base+0, base+1, base+2,
		base+1, base+3, base+2,
	)
}

// flushSpriteBatch submits accumulated vertices as a single DrawTriangles32 call.
func (s *Scene) flushSpriteBatch(target *ebiten.Image, key batchKey) {
	if len(s.batchVerts) == 0 {
		return
	}

	var page *ebiten.Image
	if key.page == magentaPlaceholderPage {
		page = ensureMagentaImage()
	} else {
		page = atlasManager().Page(int(key.page))
	}
	if page == nil {
		s.batchVerts = s.batchVerts[:0]
		s.batchInds = s.batchInds[:0]
		return
	}

	var triOp ebiten.DrawTrianglesOptions
	triOp.Blend = key.blend.EbitenBlend()
	triOp.ColorScaleMode = ebiten.ColorScaleModePremultipliedAlpha

	target.DrawTriangles32(s.batchVerts, s.batchInds, page, &triOp)

	s.batchVerts = s.batchVerts[:0]
	s.batchInds = s.batchInds[:0]
}

// submitParticlesBatched draws all alive particles using a single DrawTriangles32 call.
func (s *Scene) submitParticlesBatched(target *ebiten.Image, cmd *RenderCommand) {
	e := cmd.emitter
	if e == nil || e.alive == 0 {
		return
	}

	r := &cmd.TextureRegion

	// Resolve the source image and UV bounds.
	var srcImg *ebiten.Image
	var su0, sv0, su1, sv1 float32

	if cmd.directImage != nil {
		srcImg = cmd.directImage
		b := srcImg.Bounds()
		su0, sv0 = float32(b.Min.X), float32(b.Min.Y)
		su1, sv1 = float32(b.Max.X), float32(b.Max.Y)
	} else {
		if r.Page == magentaPlaceholderPage {
			srcImg = ensureMagentaImage()
		} else {
			srcImg = atlasManager().Page(int(r.Page))
		}
		if srcImg == nil {
			return
		}
		if r.Rotated {
			su0, sv0 = float32(r.X), float32(r.Y)
			su1, sv1 = float32(r.X)+float32(r.Height), float32(r.Y)+float32(r.Width)
		} else {
			su0, sv0 = float32(r.X), float32(r.Y)
			su1, sv1 = float32(r.X)+float32(r.Width), float32(r.Y)+float32(r.Height)
		}
	}

	// Base transform components (widen to float64 for particle position math).
	bt := &cmd.Transform
	ba, bb, bc, bd, btx, bty := float64(bt[0]), float64(bt[1]), float64(bt[2]), float64(bt[3]), float64(bt[4]), float64(bt[5])

	ow := float64(r.OriginalW)
	oh := float64(r.OriginalH)
	halfW := ow / 2
	halfH := oh / 2
	offX := float64(r.OffsetX)
	offY := float64(r.OffsetY)

	// Precompute emitter-constant UV coordinates and quad dimensions outside the particle loop.
	var psx, psy [4]float32
	if cmd.directImage != nil {
		psx = [4]float32{su0, su1, su0, su1}
		psy = [4]float32{sv0, sv0, sv1, sv1}
	} else if r.Rotated {
		psx = [4]float32{su1, su1, su0, su0}
		psy = [4]float32{sv0, sv1, sv0, sv1}
	} else {
		psx = [4]float32{su0, su1, su0, su1}
		psy = [4]float32{sv0, sv0, sv1, sv1}
	}

	var qw, qh float64
	if cmd.directImage != nil {
		qw = float64(su1 - su0)
		qh = float64(sv1 - sv0)
	} else {
		qw = float64(r.Width)
		qh = float64(r.Height)
	}

	s.batchVerts = s.batchVerts[:0]
	s.batchInds = s.batchInds[:0]

	for i := 0; i < e.alive; i++ {
		p := &e.particles[i]

		// Build per-particle local→world transform.
		ps := float64(p.scale)
		localTx := (offX-halfW)*ps + halfW + p.x
		localTy := (offY-halfH)*ps + halfH + p.y

		// Concat with base: M_base * M_local
		fa := ba * ps
		fb := bb * ps
		fc := bc * ps
		fd := bd * ps
		ftx := ba*localTx + bc*localTy + btx
		fty := bb*localTx + bd*localTy + bty

		// Per-particle color
		ca := p.alpha * cmd.Color.A
		cr := p.colorR * cmd.Color.R * ca
		cg := p.colorG * cmd.Color.G * ca
		cb := p.colorB * cmd.Color.B * ca

		base := uint32(len(s.batchVerts))

		// Inline 4 vertices: corners at (0,0), (qw,0), (0,qh), (qw,qh).
		// qw/qh are emitter-constant, hoisted outside the particle loop.
		s.batchVerts = append(s.batchVerts,
			ebiten.Vertex{
				DstX: float32(ftx), DstY: float32(fty),
				SrcX: psx[0], SrcY: psy[0],
				ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca,
			},
			ebiten.Vertex{
				DstX: float32(fa*qw + ftx), DstY: float32(fb*qw + fty),
				SrcX: psx[1], SrcY: psy[1],
				ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca,
			},
			ebiten.Vertex{
				DstX: float32(fc*qh + ftx), DstY: float32(fd*qh + fty),
				SrcX: psx[2], SrcY: psy[2],
				ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca,
			},
			ebiten.Vertex{
				DstX: float32(fa*qw + fc*qh + ftx), DstY: float32(fb*qw + fd*qh + fty),
				SrcX: psx[3], SrcY: psy[3],
				ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca,
			},
		)

		s.batchInds = append(s.batchInds,
			base+0, base+1, base+2,
			base+1, base+3, base+2,
		)
	}

	if len(s.batchVerts) == 0 {
		return
	}

	var triOp ebiten.DrawTrianglesOptions
	triOp.Blend = cmd.BlendMode.EbitenBlend()
	triOp.ColorScaleMode = ebiten.ColorScaleModePremultipliedAlpha

	target.DrawTriangles32(s.batchVerts, s.batchInds, srcImg, &triOp)

	s.batchVerts = s.batchVerts[:0]
	s.batchInds = s.batchInds[:0]
}
