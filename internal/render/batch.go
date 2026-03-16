package render

import (
	"image"

	"github.com/devthicket/willow/internal/mesh"
	"github.com/devthicket/willow/internal/node"
	"github.com/devthicket/willow/internal/particle"
	"github.com/devthicket/willow/internal/text"
	"github.com/devthicket/willow/internal/types"
	"github.com/hajimehoshi/ebiten/v2"
)

// SubmitBatches iterates sorted commands and submits draw calls to the target.
func (p *Pipeline) SubmitBatches(target *ebiten.Image) {
	if len(p.Commands) == 0 {
		return
	}

	if p.BatchMode == BatchModeCoalesced {
		p.SubmitBatchesCoalesced(target)
		return
	}

	var op ebiten.DrawImageOptions
	for i := range p.Commands {
		cmd := &p.Commands[i]
		switch cmd.Type {
		case CommandSprite:
			submitSprite(target, cmd, &op)
		case CommandParticle:
			submitParticles(target, cmd, &op)
		case CommandMesh:
			p.submitMesh(target, cmd)
		case CommandTilemap:
			p.submitTilemap(target, cmd)
		case CommandSDF:
			p.submitSDF(target, cmd)
		case CommandBitmapText:
			p.submitBitmapText(target, cmd)
		}
	}
}

// submitSprite draws a single sprite command using DrawImage.
func submitSprite(target *ebiten.Image, cmd *RenderCommand, op *ebiten.DrawImageOptions) {
	if cmd.DirectImage != nil {
		op.GeoM.Reset()
		op.GeoM.Concat(CommandGeoM(cmd))
		op.ColorScale.Reset()
		a := cmd.Color.A
		if a == 0 && cmd.Color.R == 0 && cmd.Color.G == 0 && cmd.Color.B == 0 {
			op.ColorScale.Scale(1, 1, 1, 1)
		} else {
			op.ColorScale.Scale(cmd.Color.R*a, cmd.Color.G*a, cmd.Color.B*a, a)
		}
		op.Blend = cmd.BlendMode.EbitenBlend()
		op.Filter = ebiten.FilterNearest
		target.DrawImage(cmd.DirectImage, op)
		return
	}

	r := &cmd.TextureRegion
	page := resolveAtlasPage(r.Page)
	if page == nil {
		return
	}

	var subRect image.Rectangle
	if r.Rotated {
		subRect = image.Rect(int(r.X), int(r.Y), int(r.X)+int(r.Height), int(r.Y)+int(r.Width))
	} else {
		subRect = image.Rect(int(r.X), int(r.Y), int(r.X)+int(r.Width), int(r.Y)+int(r.Height))
	}
	subImg := page.SubImage(subRect).(*ebiten.Image)

	op.GeoM.Reset()
	if r.Rotated {
		op.GeoM.Rotate(-1.5707963267948966) // -π/2
		op.GeoM.Translate(0, float64(r.Width))
	}
	if r.OffsetX != 0 || r.OffsetY != 0 {
		op.GeoM.Translate(float64(r.OffsetX), float64(r.OffsetY))
	}
	op.GeoM.Concat(CommandGeoM(cmd))

	op.ColorScale.Reset()
	a := cmd.Color.A
	if a == 0 && cmd.Color.R == 0 && cmd.Color.G == 0 && cmd.Color.B == 0 {
		op.ColorScale.Scale(1, 1, 1, 1)
	} else {
		op.ColorScale.Scale(cmd.Color.R*a, cmd.Color.G*a, cmd.Color.B*a, a)
	}
	op.Blend = cmd.BlendMode.EbitenBlend()
	target.DrawImage(subImg, op)
}

// submitParticles draws all alive particles using DrawImage (immediate mode).
func submitParticles(target *ebiten.Image, cmd *RenderCommand, op *ebiten.DrawImageOptions) {
	e := cmd.Emitter
	if e == nil || e.AliveCount() == 0 {
		return
	}

	r := &cmd.TextureRegion
	var subImg *ebiten.Image
	if cmd.DirectImage != nil {
		subImg = cmd.DirectImage
	} else {
		page := resolveAtlasPage(r.Page)
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

	baseGeoM := CommandGeoM(cmd)

	for i := 0; i < e.AliveCount(); i++ {
		px, py, pScale, pAlpha, pColorR, pColorG, pColorB := e.ParticleRenderData(i)

		op.GeoM.Reset()
		if r.Rotated {
			op.GeoM.Rotate(-1.5707963267948966)
			op.GeoM.Translate(0, float64(r.Width))
		}
		if r.OffsetX != 0 || r.OffsetY != 0 {
			op.GeoM.Translate(float64(r.OffsetX), float64(r.OffsetY))
		}
		op.GeoM.Translate(-float64(r.OriginalW)/2, -float64(r.OriginalH)/2)
		op.GeoM.Scale(float64(pScale), float64(pScale))
		op.GeoM.Translate(float64(r.OriginalW)/2, float64(r.OriginalH)/2)
		op.GeoM.Translate(px, py)
		op.GeoM.Concat(baseGeoM)

		cr := pColorR * cmd.Color.R
		cg := pColorG * cmd.Color.G
		cb := pColorB * cmd.Color.B
		ca := pAlpha * cmd.Color.A
		op.ColorScale.Reset()
		op.ColorScale.Scale(cr*ca, cg*ca, cb*ca, ca)
		op.Blend = cmd.BlendMode.EbitenBlend()
		target.DrawImage(subImg, op)
	}
}

// submitTilemap draws a tilemap layer command using DrawTriangles.
func (p *Pipeline) submitTilemap(target *ebiten.Image, cmd *RenderCommand) {
	if cmd.TilemapImage == nil || len(cmd.TilemapVerts) == 0 || len(cmd.TilemapInds) == 0 {
		return
	}
	var triOp ebiten.DrawTrianglesOptions
	triOp.Blend = cmd.BlendMode.EbitenBlend()
	triOp.ColorScaleMode = ebiten.ColorScaleModePremultipliedAlpha
	target.DrawTriangles(cmd.TilemapVerts, cmd.TilemapInds, cmd.TilemapImage, &triOp)
}

// submitMesh draws a mesh command using DrawTriangles.
func (p *Pipeline) submitMesh(target *ebiten.Image, cmd *RenderCommand) {
	if cmd.MeshImage == nil || len(cmd.MeshVerts) == 0 || len(cmd.MeshInds) == 0 {
		return
	}
	var triOp ebiten.DrawTrianglesOptions
	triOp.Blend = cmd.BlendMode.EbitenBlend()
	target.DrawTriangles(cmd.MeshVerts, cmd.MeshInds, cmd.MeshImage, &triOp)
}

// --- Coalesced batching ---

func (p *Pipeline) SubmitBatchesCoalesced(target *ebiten.Image) {
	p.BatchVerts = p.BatchVerts[:0]
	p.BatchInds = p.BatchInds[:0]

	var currentKey BatchKey
	inRun := false
	var op ebiten.DrawImageOptions

	for i := range p.Commands {
		cmd := &p.Commands[i]

		switch cmd.Type {
		case CommandSprite:
			if cmd.DirectImage != nil {
				p.flushSpriteBatch(target, currentKey)
				inRun = false
				submitSprite(target, cmd, &op)
				continue
			}
			key := CommandBatchKey(cmd)
			if inRun && key != currentKey {
				p.flushSpriteBatch(target, currentKey)
			}
			currentKey = key
			inRun = true
			p.AppendSpriteQuad(cmd)

		case CommandParticle:
			p.flushSpriteBatch(target, currentKey)
			inRun = false
			p.submitParticlesBatched(target, cmd)

		case CommandMesh:
			p.flushSpriteBatch(target, currentKey)
			inRun = false
			p.submitMesh(target, cmd)

		case CommandTilemap:
			p.flushSpriteBatch(target, currentKey)
			inRun = false
			p.submitTilemap(target, cmd)

		case CommandSDF:
			p.flushSpriteBatch(target, currentKey)
			inRun = false
			p.submitSDF(target, cmd)

		case CommandBitmapText:
			p.flushSpriteBatch(target, currentKey)
			inRun = false
			p.submitBitmapText(target, cmd)
		}
	}

	p.flushSpriteBatch(target, currentKey)
}

// AppendSpriteQuad appends 4 vertices and 6 indices for a single atlas sprite.
func (p *Pipeline) AppendSpriteQuad(cmd *RenderCommand) {
	r := &cmd.TextureRegion
	t := &cmd.Transform

	ox := float32(r.OffsetX)
	oy := float32(r.OffsetY)
	w := float32(r.Width)
	h := float32(r.Height)

	a, b, c, d, tx, ty := t[0], t[1], t[2], t[3], t[4], t[5]

	x0, y0 := ox, oy
	x1, y1 := ox+w, oy
	x2, y2 := ox, oy+h
	x3, y3 := ox+w, oy+h

	var sx0, sy0, sx1, sy1, sx2, sy2, sx3, sy3 float32
	if r.Rotated {
		rx := float32(r.X)
		ry := float32(r.Y)
		rh := float32(r.Height)
		rw := float32(r.Width)
		sx0, sy0 = rx+rh, ry
		sx1, sy1 = rx+rh, ry+rw
		sx2, sy2 = rx, ry
		sx3, sy3 = rx, ry+rw
	} else {
		rx := float32(r.X)
		ry := float32(r.Y)
		rw := float32(r.Width)
		rh := float32(r.Height)
		sx0, sy0 = rx, ry
		sx1, sy1 = rx+rw, ry
		sx2, sy2 = rx, ry+rh
		sx3, sy3 = rx+rw, ry+rh
	}

	var cr, cg, cb, ca float32
	ca = cmd.Color.A
	if ca == 0 && cmd.Color.R == 0 && cmd.Color.G == 0 && cmd.Color.B == 0 {
		cr, cg, cb, ca = 1, 1, 1, 1
	} else {
		cr = cmd.Color.R * ca
		cg = cmd.Color.G * ca
		cb = cmd.Color.B * ca
	}

	base := uint32(len(p.BatchVerts))

	p.BatchVerts = append(p.BatchVerts,
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

	p.BatchInds = append(p.BatchInds,
		base+0, base+1, base+2,
		base+1, base+3, base+2,
	)
}

// flushSpriteBatch submits accumulated vertices as a single DrawTriangles32 call.
func (p *Pipeline) flushSpriteBatch(target *ebiten.Image, key BatchKey) {
	if len(p.BatchVerts) == 0 {
		return
	}

	page := resolveAtlasPage(key.Page)
	if page == nil {
		p.BatchVerts = p.BatchVerts[:0]
		p.BatchInds = p.BatchInds[:0]
		return
	}

	var triOp ebiten.DrawTrianglesOptions
	triOp.Blend = key.Blend.EbitenBlend()
	triOp.ColorScaleMode = ebiten.ColorScaleModePremultipliedAlpha

	target.DrawTriangles32(p.BatchVerts, p.BatchInds, page, &triOp)

	p.BatchVerts = p.BatchVerts[:0]
	p.BatchInds = p.BatchInds[:0]
}

// submitSDF draws SDF text glyphs by transforming local-space vertices to screen
// space, then calling DrawTrianglesShader with the SDF shader.
func (p *Pipeline) submitSDF(target *ebiten.Image, cmd *RenderCommand) {
	if cmd.SdfAtlasImg == nil || cmd.SdfShader == nil || cmd.SdfVertCount == 0 || cmd.SdfIndCount == 0 {
		return
	}

	t := &cmd.Transform
	a, b, c, d, tx, ty := t[0], t[1], t[2], t[3], t[4], t[5]

	ca := cmd.Color.A
	var cr, cg, cb float32
	if ca == 0 && cmd.Color.R == 0 && cmd.Color.G == 0 && cmd.Color.B == 0 {
		cr, cg, cb, ca = 1, 1, 1, 1
	} else {
		cr = cmd.Color.R * ca
		cg = cmd.Color.G * ca
		cb = cmd.Color.B * ca
	}

	vc := cmd.SdfVertCount
	ic := cmd.SdfIndCount

	if cap(p.BatchVerts) < vc {
		p.BatchVerts = make([]ebiten.Vertex, vc)
	}
	p.BatchVerts = p.BatchVerts[:vc]

	for i := 0; i < vc; i++ {
		sv := &cmd.SdfVerts[i]
		dx := sv.DstX
		dy := sv.DstY
		p.BatchVerts[i] = ebiten.Vertex{
			DstX:   a*dx + c*dy + tx,
			DstY:   b*dx + d*dy + ty,
			SrcX:   sv.SrcX,
			SrcY:   sv.SrcY,
			ColorR: cr,
			ColorG: cg,
			ColorB: cb,
			ColorA: ca,
		}
	}

	opts := &ebiten.DrawTrianglesShaderOptions{
		Uniforms: cmd.SdfUniforms,
		Images:   [4]*ebiten.Image{cmd.SdfAtlasImg},
	}
	target.DrawTrianglesShader(p.BatchVerts[:vc], cmd.SdfInds[:ic], cmd.SdfShader, opts)

	p.BatchVerts = p.BatchVerts[:0]
}

// submitBitmapText draws pixel-perfect bitmap font glyphs.
func (p *Pipeline) submitBitmapText(target *ebiten.Image, cmd *RenderCommand) {
	if cmd.BmpImage == nil || cmd.BmpVertCount == 0 || cmd.BmpIndCount == 0 {
		return
	}

	t := &cmd.Transform
	a, b, c, d, tx, ty := t[0], t[1], t[2], t[3], t[4], t[5]

	ca := cmd.Color.A
	var cr, cg, cb float32
	if ca == 0 && cmd.Color.R == 0 && cmd.Color.G == 0 && cmd.Color.B == 0 {
		cr, cg, cb, ca = 1, 1, 1, 1
	} else {
		cr = cmd.Color.R * ca
		cg = cmd.Color.G * ca
		cb = cmd.Color.B * ca
	}

	vc := cmd.BmpVertCount
	ic := cmd.BmpIndCount

	if cap(p.BatchVerts) < vc {
		p.BatchVerts = make([]ebiten.Vertex, vc)
	}
	p.BatchVerts = p.BatchVerts[:vc]

	for i := 0; i < vc; i++ {
		sv := &cmd.BmpVerts[i]
		dx := sv.DstX
		dy := sv.DstY
		p.BatchVerts[i] = ebiten.Vertex{
			DstX:   a*dx + c*dy + tx,
			DstY:   b*dx + d*dy + ty,
			SrcX:   sv.SrcX,
			SrcY:   sv.SrcY,
			ColorR: cr,
			ColorG: cg,
			ColorB: cb,
			ColorA: ca,
		}
	}

	var triOp ebiten.DrawTrianglesOptions
	triOp.Blend = cmd.BlendMode.EbitenBlend()
	triOp.ColorScaleMode = ebiten.ColorScaleModePremultipliedAlpha
	triOp.Filter = ebiten.FilterNearest
	target.DrawTriangles(p.BatchVerts[:vc], cmd.BmpInds[:ic], cmd.BmpImage, &triOp)

	p.BatchVerts = p.BatchVerts[:0]
}

// submitParticlesBatched draws all alive particles using a single DrawTriangles32 call.
func (p *Pipeline) submitParticlesBatched(target *ebiten.Image, cmd *RenderCommand) {
	e := cmd.Emitter
	if e == nil || e.AliveCount() == 0 {
		return
	}

	r := &cmd.TextureRegion

	var srcImg *ebiten.Image
	var su0, sv0, su1, sv1 float32

	if cmd.DirectImage != nil {
		srcImg = cmd.DirectImage
		b := srcImg.Bounds()
		su0, sv0 = float32(b.Min.X), float32(b.Min.Y)
		su1, sv1 = float32(b.Max.X), float32(b.Max.Y)
	} else {
		srcImg = resolveAtlasPage(r.Page)
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

	bt := &cmd.Transform
	ba, bb, bc, bd, btx, bty := float64(bt[0]), float64(bt[1]), float64(bt[2]), float64(bt[3]), float64(bt[4]), float64(bt[5])

	ow := float64(r.OriginalW)
	oh := float64(r.OriginalH)
	halfW := ow / 2
	halfH := oh / 2
	offX := float64(r.OffsetX)
	offY := float64(r.OffsetY)

	var psx, psy [4]float32
	if cmd.DirectImage != nil {
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
	if cmd.DirectImage != nil {
		qw = float64(su1 - su0)
		qh = float64(sv1 - sv0)
	} else {
		qw = float64(r.Width)
		qh = float64(r.Height)
	}

	p.BatchVerts = p.BatchVerts[:0]
	p.BatchInds = p.BatchInds[:0]

	for i := 0; i < e.AliveCount(); i++ {
		px, py, pScale, pAlpha, pColorR, pColorG, pColorB := e.ParticleRenderData(i)

		ps := float64(pScale)
		localTx := (offX-halfW)*ps + halfW + px
		localTy := (offY-halfH)*ps + halfH + py

		fa := ba * ps
		fb := bb * ps
		fc := bc * ps
		fd := bd * ps
		ftx := ba*localTx + bc*localTy + btx
		fty := bb*localTx + bd*localTy + bty

		ca := pAlpha * cmd.Color.A
		cr := pColorR * cmd.Color.R * ca
		cg := pColorG * cmd.Color.G * ca
		cb := pColorB * cmd.Color.B * ca

		base := uint32(len(p.BatchVerts))

		p.BatchVerts = append(p.BatchVerts,
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

		p.BatchInds = append(p.BatchInds,
			base+0, base+1, base+2,
			base+1, base+3, base+2,
		)
	}

	if len(p.BatchVerts) == 0 {
		return
	}

	var triOp ebiten.DrawTrianglesOptions
	triOp.Blend = cmd.BlendMode.EbitenBlend()
	triOp.ColorScaleMode = ebiten.ColorScaleModePremultipliedAlpha

	target.DrawTriangles32(p.BatchVerts, p.BatchInds, srcImg, &triOp)

	p.BatchVerts = p.BatchVerts[:0]
	p.BatchInds = p.BatchInds[:0]
}

// --- Atlas page resolution helper ---

func resolveAtlasPage(pageIdx uint16) *ebiten.Image {
	if pageIdx == MagentaPlaceholderPage {
		if EnsureMagentaImageFn != nil {
			return EnsureMagentaImageFn()
		}
		return nil
	}
	if AtlasPageFn != nil {
		return AtlasPageFn(int(pageIdx))
	}
	return nil
}

// --- Emit helpers (for subtree rendering) ---

// EmitNodeCommand emits a render command for a single node at the given transform.
func EmitNodeCommand(p *Pipeline, n *node.Node, transform [6]float64, alpha float64, treeOrder *int) {
	if !n.Renderable_ {
		return
	}
	t32 := Affine32(transform)
	switch n.Type {
	case types.NodeTypeSprite:
		*treeOrder++
		cmd := RenderCommand{
			Type:        CommandSprite,
			Transform:   t32,
			Color:       Color32{float32(n.Color_.R()), float32(n.Color_.G()), float32(n.Color_.B()), float32(n.Color_.A() * alpha)},
			BlendMode:   n.BlendMode_,
			RenderLayer: n.RenderLayer,
			GlobalOrder: n.GlobalOrder,
			TreeOrder:   *treeOrder,
		}
		if n.CustomImage_ != nil {
			cmd.DirectImage = n.CustomImage_
		} else {
			cmd.TextureRegion = n.TextureRegion_
		}
		p.Commands = append(p.Commands, cmd)

	case types.NodeTypeMesh:
		if n.Mesh == nil || len(n.Mesh.Vertices) == 0 || len(n.Mesh.Indices) == 0 {
			return
		}
		tintColor := types.RGBA(n.Color_.R(), n.Color_.G(), n.Color_.B(), n.Color_.A()*alpha)
		dst := mesh.EnsureTransformedVerts(n)
		mesh.TransformVertices(n.Mesh.Vertices, dst, transform, tintColor)
		*treeOrder++
		p.Commands = append(p.Commands, RenderCommand{
			Type:        CommandMesh,
			Transform:   t32,
			BlendMode:   n.BlendMode_,
			RenderLayer: n.RenderLayer,
			GlobalOrder: n.GlobalOrder,
			TreeOrder:   *treeOrder,
			MeshVerts:   dst,
			MeshInds:    n.Mesh.Indices,
			MeshImage:   n.Mesh.Image,
		})

	case types.NodeTypeParticleEmitter:
		if n.Emitter != nil && n.Emitter.AliveCount() > 0 {
			*treeOrder++
			particleTransform := transform
			ws := n.Emitter.Config.WorldSpace
			if ws {
				particleTransform = p.ViewTransform
			}
			p.Commands = append(p.Commands, RenderCommand{
				Type:               CommandParticle,
				Transform:          Affine32(particleTransform),
				TextureRegion:      n.TextureRegion_,
				DirectImage:        n.CustomImage_,
				Color:              Color32{float32(n.Color_.R()), float32(n.Color_.G()), float32(n.Color_.B()), float32(n.Color_.A() * alpha)},
				BlendMode:          n.BlendMode_,
				RenderLayer:        n.RenderLayer,
				GlobalOrder:        n.GlobalOrder,
				TreeOrder:          *treeOrder,
				Emitter:            n.Emitter,
				WorldSpaceParticle: ws,
			})
		}

	case types.NodeTypeText:
		if n.TextBlock != nil && n.TextBlock.Font != nil {
			if text.IsPixelFont(n.TextBlock.Font) {
				p.Commands = EmitPixelTextCommand(n.TextBlock, n, transform, p.Commands, treeOrder)
			} else {
				p.Commands = EmitSDFTextCommand(n.TextBlock, n, transform, p.Commands, treeOrder)
			}
		}
	}
}

// importParticle is not used directly but keeps the particle import active.
var _ = (*particle.Emitter)(nil)
