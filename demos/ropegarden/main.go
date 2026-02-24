// Rope Garden  -  a cable-untangling puzzle.
//
// Eight color-coded cables stretch between two columns of sockets. Each cable
// has a left peg and a right peg. The pegs start shuffled onto wrong sockets,
// creating a tangle. Drag each peg to the socket that matches its color to
// straighten the cable. Solve all eight to win.
//
// Demonstrates: NewRope, catenary curves, pointer-bound endpoints, draggable
// nodes with snap-to-socket behavior, and runtime texture swapping.
package main

import (
	"log"
	"math"
	"math/rand/v2"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/phanxgames/willow"
)

// ---- configuration --------------------------------------------------------

const (
	screenW = 1280
	screenH = 720

	numCables = 8    // number of cables (and sockets per side)
	columnGap = 800  // horizontal distance between left and right socket columns
	rowGap    = 72.0 // vertical spacing between sockets
	snapDist  = 50.0 // how close a peg must be to snap to a socket

	socketRadius = 14.0
	pegRadius    = 18.0
	ropeWidth    = 6
	ropeTexW     = 1024 // rope texture width in pixels (covers UV path length)
)

// ---- colors ---------------------------------------------------------------

var colors = [numCables]willow.Color{
	{R: 0.95, G: 0.30, B: 0.25, A: 1}, // red
	{R: 0.25, G: 0.65, B: 0.95, A: 1}, // blue
	{R: 0.35, G: 0.88, B: 0.40, A: 1}, // green
	{R: 0.95, G: 0.75, B: 0.15, A: 1}, // gold
	{R: 0.75, G: 0.35, B: 0.90, A: 1}, // purple
	{R: 0.95, G: 0.50, B: 0.15, A: 1}, // orange
	{R: 0.20, G: 0.85, B: 0.80, A: 1}, // teal
	{R: 0.90, G: 0.35, B: 0.60, A: 1}, // pink
}

// ---- data model -----------------------------------------------------------

// peg is a draggable endpoint of a cable.
type peg struct {
	node   *willow.Node
	pos    willow.Vec2 // current world position (ropes read this via pointer)
	socket int         // which socket index this peg sits on, or -1 if floating
	home   int         // which socket index solves this peg
}

// cable connects two pegs with a visible rope.
type cable struct {
	rope      *willow.Rope
	node      *willow.Node
	start     willow.Vec2   // rope reads &start  -  must be stable pointer
	end       willow.Vec2   // rope reads &end    -  must be stable pointer
	normalImg *ebiten.Image // dimmed texture (unsolved)
	solvedImg *ebiten.Image // bright green texture (solved)
	solved    bool
}

// puzzle holds all game state.
type puzzle struct {
	scene *willow.Scene

	// Socket positions (static). Index 0..numCables-1.
	leftSockets  [numCables]willow.Vec2
	rightSockets [numCables]willow.Vec2

	// Pegs (one per socket per side). leftPegs[i] belongs to cable i.
	leftPegs  [numCables]peg
	rightPegs [numCables]peg

	// Cables. cable[i] connects leftPegs[i] to rightPegs[i].
	cables [numCables]*cable

	time        float64
	solvedCount int
	wonAt       float64 // timestamp when all cables solved (0 = not yet)
}

// ---- main -----------------------------------------------------------------

func main() {
	scene := willow.NewScene()
	scene.ClearColor = willow.Color{R: 0.06, G: 0.05, B: 0.10, A: 1}

	// Camera centered on screen so world origin = top-left corner.
	cam := scene.NewCamera(willow.Rect{Width: screenW, Height: screenH})
	cam.X = screenW / 2
	cam.Y = screenH / 2
	cam.Invalidate()

	p := &puzzle{scene: scene}
	p.layoutSockets()
	p.createSocketSprites()
	p.createCables()
	p.createPegs()

	scene.SetUpdateFunc(p.update)

	if err := willow.Run(scene, willow.RunConfig{
		Title:   "Willow  -  Rope Garden",
		Width:   screenW,
		Height:  screenH,
		ShowFPS: true,
	}); err != nil {
		log.Fatal(err)
	}
}

// ---- setup ----------------------------------------------------------------

// layoutSockets computes the x,y position of every socket.
func (p *puzzle) layoutSockets() {
	gridH := float64(numCables-1) * rowGap
	leftX := float64(screenW-columnGap) / 2
	rightX := leftX + columnGap
	topY := (screenH - gridH) / 2

	for i := range numCables {
		y := topY + float64(i)*rowGap
		p.leftSockets[i] = willow.Vec2{X: leftX, Y: y}
		p.rightSockets[i] = willow.Vec2{X: rightX, Y: y}
	}
}

// createSocketSprites adds visual indicators for each socket. The colored ring
// hints which peg belongs there; the dark inner dot marks the snap target.
func (p *puzzle) createSocketSprites() {
	for i := range numCables {
		p.addSocketSprite(p.leftSockets[i], colors[i])
		p.addSocketSprite(p.rightSockets[i], colors[i])
	}
}

func (p *puzzle) addSocketSprite(pos willow.Vec2, col willow.Color) {
	// Colored ring (hint).
	ring := willow.NewSprite("ring", willow.TextureRegion{})
	ring.ScaleX = (socketRadius + 4) * 2
	ring.ScaleY = (socketRadius + 4) * 2
	ring.PivotX = 0.5
	ring.PivotY = 0.5
	ring.X = pos.X
	ring.Y = pos.Y
	ring.Color = col
	ring.Alpha = 0.35
	p.scene.Root().AddChild(ring)

	// Inner dot.
	dot := willow.NewSprite("socket", willow.TextureRegion{})
	dot.ScaleX = socketRadius * 2
	dot.ScaleY = socketRadius * 2
	dot.PivotX = 0.5
	dot.PivotY = 0.5
	dot.X = pos.X
	dot.Y = pos.Y
	dot.Color = willow.Color{R: 0.18, G: 0.18, B: 0.22, A: 1}
	p.scene.Root().AddChild(dot)
}

// createCables builds the ropes. Added before pegs so pegs render on top.
//
// IMPORTANT: each cable is heap-allocated (*cable) so the Rope's Start/End
// pointers (&c.start, &c.end) remain stable. If cable were a value in an
// array, resizing or copying the array would invalidate those pointers.
func (p *puzzle) createCables() {
	solvedColor := willow.Color{R: 0.3, G: 1.0, B: 0.5, A: 1}

	for i := range numCables {
		c := &cable{
			start:     p.leftSockets[i],
			end:       p.rightSockets[i],
			normalImg: makeRopeTexture(colors[i], 0.6),
			solvedImg: makeRopeTexture(solvedColor, 1.0),
		}

		r := willow.NewRope("cable", c.normalImg, nil, willow.RopeConfig{
			Width:     ropeWidth,
			JoinMode:  willow.RopeJoinBevel,
			CurveMode: willow.RopeCurveCatenary,
			Segments:  24,
			Sag:       25,
			Start:     &c.start, // Rope reads these by pointer each Update().
			End:       &c.end,
		})
		c.rope = r
		c.node = r.Node()
		p.scene.Root().AddChild(r.Node())

		p.cables[i] = c
	}
}

// createPegs builds draggable endpoints and shuffles them onto wrong sockets.
func (p *puzzle) createPegs() {
	// A derangement guarantees every peg starts on the wrong socket.
	leftOrder := derangement(numCables)
	rightOrder := derangement(numCables)

	for i := range numCables {
		p.leftPegs[i] = p.makePeg(i, leftOrder[i], p.leftSockets[:])
		p.rightPegs[i] = p.makePeg(i, rightOrder[i], p.rightSockets[:])
	}
}

// makePeg creates one draggable peg for cable cableIdx, initially placed on
// startIdx within the given socket column. Its home (solution) socket is
// sockets[cableIdx].
func (p *puzzle) makePeg(cableIdx, startIdx int, sockets []willow.Vec2) peg {
	startPos := sockets[startIdx]
	col := colors[cableIdx]

	sp := willow.NewSprite("peg", willow.TextureRegion{})
	sp.ScaleX = pegRadius * 2
	sp.ScaleY = pegRadius * 2
	sp.PivotX = 0.5
	sp.PivotY = 0.5
	sp.X = startPos.X
	sp.Y = startPos.Y
	sp.Color = col

	pg := peg{
		node:   sp,
		pos:    willow.Vec2{X: startPos.X, Y: startPos.Y},
		socket: startIdx,
		home:   cableIdx,
	}

	// We need a stable pointer for the closures below. The peg will be stored
	// in either p.leftPegs or p.rightPegs (fixed-size arrays, no realloc).
	// Find which array and index by comparing the socket column pointer.
	isLeft := &sockets[0] == &p.leftSockets[0]

	sp.OnDrag(func(ctx willow.DragContext) {
		sp.X += ctx.DeltaX
		sp.Y += ctx.DeltaY
		sp.Invalidate()

		pg := p.pegPtr(cableIdx, isLeft)
		pg.pos.X = sp.X
		pg.pos.Y = sp.Y
		pg.socket = -1
	})

	sp.OnDragEnd(func(ctx willow.DragContext) {
		pg := p.pegPtr(cableIdx, isLeft)
		best, bestDist := -1, snapDist

		for si := range numCables {
			sock := sockets[si]
			dx := sock.X - sp.X
			dy := sock.Y - sp.Y
			dist := math.Sqrt(dx*dx + dy*dy)
			if dist < bestDist && !p.socketTaken(si, cableIdx, isLeft) {
				bestDist = dist
				best = si
			}
		}

		if best >= 0 {
			sp.X = sockets[best].X
			sp.Y = sockets[best].Y
			sp.Invalidate()
			pg.pos.X = sp.X
			pg.pos.Y = sp.Y
			pg.socket = best
		}

		p.checkSolved()
	})

	p.scene.Root().AddChild(sp)
	return pg
}

// pegPtr returns a pointer to a peg by cable index and side.
func (p *puzzle) pegPtr(cableIdx int, isLeft bool) *peg {
	if isLeft {
		return &p.leftPegs[cableIdx]
	}
	return &p.rightPegs[cableIdx]
}

// socketTaken reports whether socket si on the given side is occupied by
// another cable's peg.
func (p *puzzle) socketTaken(si, excludeCable int, isLeft bool) bool {
	pegs := &p.rightPegs
	if isLeft {
		pegs = &p.leftPegs
	}
	for i := range numCables {
		if i == excludeCable {
			continue
		}
		if pegs[i].socket == si {
			return true
		}
	}
	return false
}

// ---- solve check ----------------------------------------------------------

func (p *puzzle) checkSolved() {
	p.solvedCount = 0
	for i, c := range p.cables {
		wasSolved := c.solved
		c.solved = p.leftPegs[i].socket == p.leftPegs[i].home &&
			p.rightPegs[i].socket == p.rightPegs[i].home

		if c.solved {
			p.solvedCount++
		}

		// Swap rope texture when solved state changes.
		if c.solved != wasSolved {
			if c.solved {
				c.node.SetCustomImage(c.solvedImg)
			} else {
				c.node.SetCustomImage(c.normalImg)
			}
		}
	}

	if p.solvedCount == numCables && p.wonAt == 0 {
		p.wonAt = p.time
	} else if p.solvedCount < numCables {
		p.wonAt = 0
	}
}

// ---- update ---------------------------------------------------------------

func (p *puzzle) update() error {
	p.time += 1.0 / float64(ebiten.TPS())

	// Sync rope endpoints to peg positions.
	for i, c := range p.cables {
		c.start.X = p.leftPegs[i].pos.X
		c.start.Y = p.leftPegs[i].pos.Y
		c.end.X = p.rightPegs[i].pos.X
		c.end.Y = p.rightPegs[i].pos.Y
		c.rope.Update()
	}

	// Victory celebration: gentle pulse on all cables and pegs.
	if p.wonAt > 0 {
		elapsed := p.time - p.wonAt
		pulse := 0.7 + 0.3*math.Sin(elapsed*4.0)
		for _, c := range p.cables {
			c.node.Alpha = pulse
			c.node.Invalidate()
		}
		for i := range numCables {
			breathe := pegRadius*2 + 4*math.Sin(elapsed*3.0+float64(i)*0.4)
			for _, pg := range []*peg{&p.leftPegs[i], &p.rightPegs[i]} {
				pg.node.ScaleX = breathe
				pg.node.ScaleY = breathe
				pg.node.Invalidate()
			}
		}
	}

	return nil
}

// ---- helpers --------------------------------------------------------------

// derangement returns a permutation of [0..n) where no element maps to itself.
// This guarantees every peg starts on a wrong socket.
func derangement(n int) []int {
	for {
		perm := rand.Perm(n)
		fixed := false
		for i, v := range perm {
			if i == v {
				fixed = true
				break
			}
		}
		if !fixed {
			return perm
		}
	}
}

// makeRopeTexture creates a narrow image used as the rope's texture. The
// vertical gradient gives the rope a rounded, 3D look. SrcX tiles along the
// rope's cumulative path length, so the texture width must cover the longest
// expected rope.
func makeRopeTexture(col willow.Color, brightness float64) *ebiten.Image {
	img := ebiten.NewImage(ropeTexW, ropeWidth)
	for y := range ropeWidth {
		center := float64(ropeWidth) / 2
		dist := math.Abs(float64(y)-center) / center
		b := brightness * (1.0 - dist*0.5)
		r := uint8(clamp01(col.R*b) * 255)
		g := uint8(clamp01(col.G*b) * 255)
		bl := uint8(clamp01(col.B*b) * 255)
		for x := range ropeTexW {
			img.Set(x, y, &pixelColor{r, g, bl, 255})
		}
	}
	return img
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// pixelColor implements color.Color for setting individual pixels.
type pixelColor struct{ r, g, b, a uint8 }

func (c *pixelColor) RGBA() (uint32, uint32, uint32, uint32) {
	return uint32(c.r) * 257, uint32(c.g) * 257, uint32(c.b) * 257, uint32(c.a) * 257
}
