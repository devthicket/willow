// CustomPaint demonstrates Node.CustomPaint and the Painter facade. A single
// host node emits a procedural starburst as one batched mesh: N triangular
// spokes radiating from the node's origin, rotating over time. Because the
// whole shape is submitted as one CommandMesh, adding more spokes does not
// add more pipeline entries.
package main

import (
	"flag"
	"log"
	"math"

	"github.com/devthicket/willow"
	"github.com/hajimehoshi/ebiten/v2"
)

const (
	windowTitle = "Willow  -  CustomPaint Example"
	screenW     = 640
	screenH     = 480
	numSpokes   = 24
	spokeLen    = 100
	spokeWidth  = 6
)

func main() {
	autotest := flag.String("autotest", "", "path to test script JSON (run and exit)")
	flag.Parse()

	scene := willow.NewScene()
	scene.ClearColor = willow.RGB(0.05, 0.05, 0.10)

	host := willow.NewContainer("starburst")
	host.SetPosition(screenW/2, screenH/2)
	scene.Root.AddChild(host)

	verts := make([]ebiten.Vertex, numSpokes*3)
	inds := make([]uint16, numSpokes*3)
	for i := range numSpokes {
		base := uint16(i * 3)
		inds[i*3+0] = base + 0
		inds[i*3+1] = base + 1
		inds[i*3+2] = base + 2
	}

	angle := 0.0
	host.OnUpdate = func(dt float64) {
		angle += dt * 1.2
	}

	willow.SetCustomPaint(host, func(p *willow.Painter, treeOrder *int) {
		// Pre-transform vertices to screen space using the host's world transform.
		wt := p.WorldTransform()
		a, b := float32(wt[0]), float32(wt[1])
		c, d := float32(wt[2]), float32(wt[3])
		ox, oy := float32(wt[4]), float32(wt[5])
		toScreen := func(x, y float32) (float32, float32) {
			return a*x + c*y + ox, b*x + d*y + oy
		}

		for i := range numSpokes {
			theta := angle + float64(i)*2*math.Pi/float64(numSpokes)
			cs, sn := math.Cos(theta), math.Sin(theta)
			tipX, tipY := float32(cs*spokeLen), float32(sn*spokeLen)
			perpX, perpY := float32(-sn*spokeWidth), float32(cs*spokeWidth)

			t0x, t0y := toScreen(tipX, tipY)
			t1x, t1y := toScreen(perpX, perpY)
			t2x, t2y := toScreen(-perpX, -perpY)

			base := i * 3
			verts[base+0] = ebiten.Vertex{
				DstX: t0x, DstY: t0y,
				ColorR: 1.0, ColorG: 0.7, ColorB: 0.3, ColorA: 1,
			}
			verts[base+1] = ebiten.Vertex{
				DstX: t1x, DstY: t1y,
				ColorR: 0.4, ColorG: 0.5, ColorB: 1.0, ColorA: 1,
			}
			verts[base+2] = ebiten.Vertex{
				DstX: t2x, DstY: t2y,
				ColorR: 0.4, ColorG: 0.5, ColorB: 1.0, ColorA: 1,
			}
		}

		p.AppendTriangles(willow.TrianglesPaint{
			Verts: verts,
			Inds:  inds,
			Image: willow.WhitePixel,
		}, treeOrder)
	})

	if err := willow.Run(scene, willow.RunConfig{
		Title:        windowTitle,
		Width:        screenW,
		Height:       screenH,
		ShowFPS:      true,
		AutoTestPath: *autotest,
	}); err != nil {
		log.Fatal(err)
	}
}
