// Village — a low-poly software 3D village rendered in the PS1/N64 aesthetic.
//
// Architecture: a CPU rasterizer writes triangles into a pixel buffer which is
// displayed via SetCustomImage on a Willow sprite. Willow handles the window,
// game loop, and any future HUD nodes. The renderer knows nothing about Willow.
//
// Camera: fixed low-angle perspective, looking across the village rather than
// down at it. The low angle makes buildings feel tall and dramatic.
//
// Controls:
//
//	W/S or ↑/↓  – move forward/back
//	A/D          – strafe left/right
//	Q/E or ←/→  – turn left/right
//	R/F          – pitch up/down
package main

import (
	"bufio"
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"math"
	"os"
	"runtime"
	"strconv"
	"strings"

	"github.com/devthicket/willow"
	"github.com/hajimehoshi/ebiten/v2"
)

// runtimeCaller is an alias so dumpCamera can call runtime.Caller without
// the import being flagged as unused by a single-file go run.
var runtimeCaller = runtime.Caller

//go:embed *.scene
var sceneFS embed.FS

// ── Screen & render settings ──────────────────────────────────────────────────

const (
	screenW = 800
	screenH = 500
	fovDeg  = 65.0
	nearZ   = 0.2
)

var focal = float64(screenW/2) / math.Tan(fovDeg*math.Pi/360)

// ── Colour type ───────────────────────────────────────────────────────────────

type col3 struct{ r, g, b uint8 }

func c(r, g, b uint8) col3 { return col3{r, g, b} }

// ── Geometry types ────────────────────────────────────────────────────────────

type vec3 struct{ x, y, z float64 }

func (a vec3) sub(b vec3) vec3 { return vec3{a.x - b.x, a.y - b.y, a.z - b.z} }
func (a vec3) cross(b vec3) vec3 {
	return vec3{a.y*b.z - a.z*b.y, a.z*b.x - a.x*b.z, a.x*b.y - a.y*b.x}
}
func (a vec3) dot(b vec3) float64 { return a.x*b.x + a.y*b.y + a.z*b.z }

// A triangle with a flat colour.
type tri struct {
	v [3]vec3
	c col3
}

// quad splits a planar quad (v0,v1,v2,v3 counter-clockwise) into two triangles.
func quad(v0, v1, v2, v3 vec3, c col3) []tri {
	return []tri{
		{v: [3]vec3{v0, v1, v2}, c: c},
		{v: [3]vec3{v0, v2, v3}, c: c},
	}
}

// ── Box & roof geometry builders ──────────────────────────────────────────────

// box builds a solid box: a floor quad + 4 walls.
// Front = -Z face, Back = +Z face, Left = -X face, Right = +X face.
// top/bottom are the Y extents; the box sits on the ground.
func box(x0, z0, x1, z1, yBot, yTop float64, front, back, left, right, top col3) []tri {
	out := make([]tri, 0, 10)
	// top cap
	out = append(out, quad(
		vec3{x0, yTop, z0}, vec3{x1, yTop, z0},
		vec3{x1, yTop, z1}, vec3{x0, yTop, z1},
		top)...)
	// front wall (z = z0, faces -Z)
	out = append(out, quad(
		vec3{x0, yBot, z0}, vec3{x0, yTop, z0},
		vec3{x1, yTop, z0}, vec3{x1, yBot, z0},
		front)...)
	// back wall (z = z1, faces +Z)
	out = append(out, quad(
		vec3{x1, yBot, z1}, vec3{x1, yTop, z1},
		vec3{x0, yTop, z1}, vec3{x0, yBot, z1},
		back)...)
	// left wall (x = x0, faces -X)
	out = append(out, quad(
		vec3{x0, yBot, z1}, vec3{x0, yTop, z1},
		vec3{x0, yTop, z0}, vec3{x0, yBot, z0},
		left)...)
	// right wall (x = x1, faces +X)
	out = append(out, quad(
		vec3{x1, yBot, z0}, vec3{x1, yTop, z0},
		vec3{x1, yTop, z1}, vec3{x1, yBot, z1},
		right)...)
	return out
}

// pyramidRoof builds a hip/pyramid roof centred over (cx, cz) with base radius
// (hw, hd) at yBase and a peak at yPeak.
func pyramidRoof(cx, cz, hw, hd, yBase, yPeak float64, sideA, sideB, sideC, sideD col3) []tri {
	peak := vec3{cx, yPeak, cz}
	bl := vec3{cx - hw, yBase, cz - hd}
	br := vec3{cx + hw, yBase, cz - hd}
	tr := vec3{cx + hw, yBase, cz + hd}
	tl := vec3{cx - hw, yBase, cz + hd}
	return []tri{
		{v: [3]vec3{bl, br, peak}, c: sideA}, // front slope
		{v: [3]vec3{br, tr, peak}, c: sideB}, // right slope
		{v: [3]vec3{tr, tl, peak}, c: sideC}, // back slope
		{v: [3]vec3{tl, bl, peak}, c: sideD}, // left slope
	}
}

// ── Camera transform & projection ────────────────────────────────────────────

// worldToCam transforms a world-space point into camera space.
// Yaw rotates around Y (positive = camera turns right/clockwise from above).
// Pitch rotates around X (negative = camera tilts down).
func worldToCam(p vec3, pos vec3, yaw, pitch float64) vec3 {
	// Translate
	p.x -= pos.x
	p.y -= pos.y
	p.z -= pos.z

	// Yaw (rotate around Y): aligns camera forward with +Z in camera space
	cy, sy := math.Cos(yaw), math.Sin(yaw)
	x1 := p.x*cy - p.z*sy
	z1 := p.x*sy + p.z*cy

	// Pitch (rotate around X): tilts camera up/down
	cp, sp := math.Cos(pitch), math.Sin(pitch)
	y2 := p.y*cp - z1*sp
	z2 := p.y*sp + z1*cp

	return vec3{x1, y2, z2}
}

// sv holds a projected screen vertex. Coordinates are floating-point so the
// rasterizer can test pixel *centres* against the triangle edges — this is the
// standard fix for shared-edge flickering and eliminates z-fighting between
// the two triangles of every quad.
type sv struct{ x, y, iz float64 }

func project(cv vec3) sv {
	return sv{
		x:  cv.x/cv.z*focal + screenW/2,
		y:  -cv.y/cv.z*focal + screenH/2,
		iz: 1.0 / cv.z,
	}
}

// ── Near-plane clipping ───────────────────────────────────────────────────────
//
// Sutherland-Hodgman against z = nearZ. Returns 0–2 output triangles.

func lerpV(a, b vec3, t float64) vec3 {
	return vec3{a.x + t*(b.x-a.x), a.y + t*(b.y-a.y), a.z + t*(b.z-a.z)}
}

func clipTri(t tri, camPos vec3, yaw, pitch float64) []tri {
	// Transform all 3 verts to camera space
	cv := [3]vec3{
		worldToCam(t.v[0], camPos, yaw, pitch),
		worldToCam(t.v[1], camPos, yaw, pitch),
		worldToCam(t.v[2], camPos, yaw, pitch),
	}

	inside := [3]bool{cv[0].z >= nearZ, cv[1].z >= nearZ, cv[2].z >= nearZ}
	nIn := 0
	for _, b := range inside {
		if b {
			nIn++
		}
	}

	switch nIn {
	case 0:
		return nil
	case 3:
		// All in front — return as camera-space tri
		return []tri{{v: [3]vec3{cv[0], cv[1], cv[2]}, c: t.c}}
	case 1:
		// One vertex in front → clip to one triangle
		in, out0, out1 := -1, -1, -1
		for i := 0; i < 3; i++ {
			if inside[i] && in == -1 {
				in = i
			} else if !inside[i] && out0 == -1 {
				out0 = i
			} else if !inside[i] {
				out1 = i
			}
		}
		t0 := (nearZ - cv[in].z) / (cv[out0].z - cv[in].z)
		t1 := (nearZ - cv[in].z) / (cv[out1].z - cv[in].z)
		clip0 := lerpV(cv[in], cv[out0], t0)
		clip1 := lerpV(cv[in], cv[out1], t1)
		return []tri{{v: [3]vec3{cv[in], clip0, clip1}, c: t.c}}
	case 2:
		// Two verts in front → clip to a quad (two triangles)
		var out_, in0, in1 int
		out_ = -1
		for i := 0; i < 3; i++ {
			if !inside[i] {
				out_ = i
			}
		}
		in0, in1 = -1, -1
		for i := 0; i < 3; i++ {
			if inside[i] && in0 == -1 {
				in0 = i
			} else if inside[i] {
				in1 = i
			}
		}
		t0 := (nearZ - cv[out_].z) / (cv[in0].z - cv[out_].z)
		t1 := (nearZ - cv[out_].z) / (cv[in1].z - cv[out_].z)
		clip0 := lerpV(cv[out_], cv[in0], t0)
		clip1 := lerpV(cv[out_], cv[in1], t1)
		return []tri{
			{v: [3]vec3{cv[in0], clip0, cv[in1]}, c: t.c},
			{v: [3]vec3{clip0, clip1, cv[in1]}, c: t.c},
		}
	}
	return nil
}

// ── Rasterizer ────────────────────────────────────────────────────────────────

// edgeFn returns the signed area of the parallelogram formed by (a→b, a→p).
// Positive when p is to the left of the directed edge a→b.
func edgeFn(ax, ay, bx, by, px, py float64) float64 {
	return (bx-ax)*(py-ay) - (by-ay)*(px-ax)
}

func clampI(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// rasterize draws a camera-space triangle (already clipped) into the pixel buf.
// Pixel centres (x+0.5, y+0.5) are tested against the edges so that every
// pixel on a shared edge belongs to exactly one triangle — no flickering seams.
func rasterize(a, b, cc sv, color col3, pixels []byte, zbuf []float64) {
	minX := clampI(int(math.Floor(math.Min(a.x, math.Min(b.x, cc.x)))), 0, screenW-1)
	maxX := clampI(int(math.Ceil(math.Max(a.x, math.Max(b.x, cc.x)))), 0, screenW-1)
	minY := clampI(int(math.Floor(math.Min(a.y, math.Min(b.y, cc.y)))), 0, screenH-1)
	maxY := clampI(int(math.Ceil(math.Max(a.y, math.Max(b.y, cc.y)))), 0, screenH-1)
	if minX > maxX || minY > maxY {
		return
	}

	area := edgeFn(a.x, a.y, b.x, b.y, cc.x, cc.y)
	if math.Abs(area) < 0.5 {
		return
	}

	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			// Test the pixel centre, not its top-left corner.
			px, py := float64(x)+0.5, float64(y)+0.5
			w0 := edgeFn(b.x, b.y, cc.x, cc.y, px, py)
			w1 := edgeFn(cc.x, cc.y, a.x, a.y, px, py)
			w2 := edgeFn(a.x, a.y, b.x, b.y, px, py)

			if area > 0 {
				if w0 < 0 || w1 < 0 || w2 < 0 {
					continue
				}
			} else {
				if w0 > 0 || w1 > 0 || w2 > 0 {
					continue
				}
			}

			// Perspective-correct depth interpolation
			iz := (w0*a.iz + w1*b.iz + w2*cc.iz) / area
			idx := y*screenW + x
			if iz <= zbuf[idx] {
				continue
			}
			zbuf[idx] = iz

			i := idx * 4
			pixels[i] = color.r
			pixels[i+1] = color.g
			pixels[i+2] = color.b
			pixels[i+3] = 255
		}
	}
}

// ── Game ──────────────────────────────────────────────────────────────────────

type game struct {
	scene    []tri
	sky      col3
	pixels   []byte
	zbuf     []float64
	framebuf *ebiten.Image
	viewport *willow.Node
	willow   *willow.Scene

	// Camera
	camPos   vec3
	camYaw   float64
	camPitch float64

	// Scene switching
	sceneName  string
	tabWasDown bool

	// Mouse camera control
	mouseLastX int
	mouseLastY int
	mouseDrag  bool

	// F5 dump
	f5WasDown bool
}

func newGame() *game {
	g := &game{
		pixels:   make([]byte, screenW*screenH*4),
		zbuf:     make([]float64, screenW*screenH),
		framebuf: ebiten.NewImage(screenW, screenH),
	}

	g.willow = willow.NewScene()
	g.viewport = willow.NewSprite("viewport", willow.TextureRegion{})
	g.viewport.SetCustomImage(g.framebuf)
	g.willow.Root.AddChild(g.viewport)
	g.willow.SetUpdateFunc(g.update)

	g.loadNamedScene("tavern.scene")
	return g
}

func (g *game) loadNamedScene(name string) {
	tris, sky, camValid, camPos, camYaw, camPitch := loadScene(name)
	g.scene = tris
	g.sky = sky
	g.sceneName = name

	// Default camera fallback (overridden by the scene file if valid)
	g.camPos = vec3{-1, 2.8, 3.5}
	g.camYaw = math.Pi / 4
	g.camPitch = -0.22
	if camValid {
		g.camPos = camPos
		g.camYaw = camYaw
		g.camPitch = camPitch
	}

	g.willow.ClearColor = willow.RGB(
		float64(sky.r)/255,
		float64(sky.g)/255,
		float64(sky.b)/255,
	)
}

func (g *game) update() error {
	const moveSpeed = 0.1

	// ── Tab: toggle scene ────────────────────────────────────────────────────
	if g.willow.IsKeyPressed(ebiten.KeyTab) && !g.tabWasDown {
		next := "tavern.scene"
		if g.sceneName == "tavern.scene" {
			next = "village.scene"
		}
		g.loadNamedScene(next)
	}
	g.tabWasDown = g.willow.IsKeyPressed(ebiten.KeyTab)

	// ── F5: dump camera ──────────────────────────────────────────────────────
	if ebiten.IsKeyPressed(ebiten.KeyF5) && !g.f5WasDown {
		g.dumpCamera()
	}
	g.f5WasDown = ebiten.IsKeyPressed(ebiten.KeyF5)

	// ── Right-click drag: look (WoW style) ───────────────────────────────────
	mx, my := ebiten.CursorPosition()
	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonRight) {
		if g.mouseDrag {
			dx := mx - g.mouseLastX
			dy := my - g.mouseLastY
			g.camYaw += float64(dx) * 0.005
			g.camPitch += float64(dy) * 0.004
			g.camPitch = math.Max(-math.Pi/2+0.05, math.Min(math.Pi/2-0.05, g.camPitch))
		}
		g.mouseDrag = true
	} else {
		g.mouseDrag = false
	}
	g.mouseLastX, g.mouseLastY = mx, my

	// ── Scroll wheel: dolly along full look vector ───────────────────────────
	_, wy := ebiten.Wheel()
	if wy != 0 {
		cp, sp := math.Cos(g.camPitch), math.Sin(g.camPitch)
		cy, sy := math.Cos(g.camYaw), math.Sin(g.camYaw)
		speed := wy * 0.5
		g.camPos.x += sy * cp * speed
		g.camPos.y -= sp * speed
		g.camPos.z += cy * cp * speed
	}

	// ── WASD: move along full look vector (pitch included) ──────────────────
	cp, sp := math.Cos(g.camPitch), math.Sin(g.camPitch)
	sy, cy := math.Sin(g.camYaw), math.Cos(g.camYaw)
	if g.willow.IsKeyPressed(ebiten.KeyW) {
		g.camPos.x += sy * cp * moveSpeed
		g.camPos.y += sp * moveSpeed
		g.camPos.z += cy * cp * moveSpeed
	}
	if g.willow.IsKeyPressed(ebiten.KeyS) {
		g.camPos.x -= sy * cp * moveSpeed
		g.camPos.y -= sp * moveSpeed
		g.camPos.z -= cy * cp * moveSpeed
	}
	if g.willow.IsKeyPressed(ebiten.KeyA) {
		g.camPos.x -= cy * moveSpeed
		g.camPos.z += sy * moveSpeed
	}
	if g.willow.IsKeyPressed(ebiten.KeyD) {
		g.camPos.x += cy * moveSpeed
		g.camPos.z -= sy * moveSpeed
	}

	// ── Space / Shift: fly up / down ─────────────────────────────────────────
	if g.willow.IsKeyPressed(ebiten.KeySpace) {
		g.camPos.y += moveSpeed
	}
	if g.willow.IsKeyPressed(ebiten.KeyShiftLeft) || g.willow.IsKeyPressed(ebiten.KeyShiftRight) {
		g.camPos.y -= moveSpeed
	}

	g.render()
	return nil
}

func (g *game) dumpCamera() {
	line := fmt.Sprintf("camera  pos=%.3f,%.3f,%.3f  yaw=%.4f  pitch=%.4f\n",
		g.camPos.x, g.camPos.y, g.camPos.z, g.camYaw, g.camPitch)
	// Write next to the source file so it's easy to find regardless of launch dir.
	_, file, _, _ := runtimeCaller(0)
	path := file[:len(file)-len("main.go")] + "cam_dump.txt"
	if err := os.WriteFile(path, []byte(line), 0644); err != nil {
		fmt.Fprintln(os.Stderr, "cam dump error:", err)
	} else {
		fmt.Print("cam dumped to ", path, ": ", line)
	}
}

func (g *game) render() {
	px := g.pixels
	zb := g.zbuf

	// Clear pixel buffer to sky colour
	sr, sg, sb := g.sky.r, g.sky.g, g.sky.b
	for i := 0; i < screenW*screenH; i++ {
		j := i * 4
		px[j], px[j+1], px[j+2], px[j+3] = sr, sg, sb, 255
	}
	// Clear z-buffer
	for i := range zb {
		zb[i] = 0
	}

	// Process every triangle: clip against near plane, project, rasterize.
	for _, t := range g.scene {
		clipped := clipTri(t, g.camPos, g.camYaw, g.camPitch)
		for _, ct := range clipped {
			// ct.v is already in camera space from clipTri
			sa := project(ct.v[0])
			sb2 := project(ct.v[1])
			sc := project(ct.v[2])
			rasterize(sa, sb2, sc, ct.c, px, zb)
		}
	}

	g.framebuf.WritePixels(px)
}

// ── Scene parser ──────────────────────────────────────────────────────────────

type shapeDef struct {
	defaults map[string]float64
	body     []string
}

type sceneParser struct {
	pal  map[string]col3
	defs map[string]*shapeDef
	out  []tri
	cam  struct {
		pos   vec3
		yaw   float64
		pitch float64
		valid bool
	}
}

func loadScene(path string) (tris []tri, skyCol col3, camValid bool, camPos vec3, camYaw, camPitch float64) {
	p := &sceneParser{
		pal:  make(map[string]col3),
		defs: make(map[string]*shapeDef),
	}
	if err := p.parseFile(sceneFS, path); err != nil {
		panic(fmt.Sprintf("scene %q: %v", path, err))
	}
	sky := col3{0x12, 0x10, 0x2a}
	if c, ok := p.pal["sky"]; ok {
		sky = c
	}
	return p.out, sky, p.cam.valid, p.cam.pos, p.cam.yaw, p.cam.pitch
}

func (p *sceneParser) parseFile(fsys fs.FS, path string) error {
	f, err := fsys.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	inDef := false
	var defName, defHeader string
	var defBody []string
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if inDef {
			if line == "}" {
				inDef = false
				p.registerDef(defName, defHeader, defBody)
				defBody = nil
			} else {
				defBody = append(defBody, line)
			}
			continue
		}
		if idx := strings.Index(line, " = #"); idx > 0 {
			name := strings.TrimSpace(line[:idx])
			hex := strings.TrimSpace(line[idx+4:])
			if ci := strings.Index(hex, " #"); ci >= 0 {
				hex = strings.TrimSpace(hex[:ci])
			}
			c, err := parseHexColor(hex)
			if err != nil {
				return fmt.Errorf("palette %q: %v", name, err)
			}
			p.pal[name] = c
			continue
		}
		if strings.HasPrefix(line, "def ") {
			rest := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(line[4:]), "{"))
			fields := strings.Fields(rest)
			defName = fields[0]
			defHeader = rest
			inDef = true
			continue
		}
		if err := p.parseLine(line, nil, 0, 0); err != nil {
			return fmt.Errorf("line %q: %v", line, err)
		}
	}
	return scanner.Err()
}

func (p *sceneParser) registerDef(name, header string, body []string) {
	fields := strings.Fields(header)
	defaults := make(map[string]float64)
	for _, f := range fields[1:] {
		kv := strings.SplitN(f, "=", 2)
		if len(kv) == 2 {
			if v, err := strconv.ParseFloat(kv[1], 64); err == nil {
				defaults[kv[0]] = v
			}
		}
	}
	p.defs[name] = &shapeDef{defaults: defaults, body: body}
}

func (p *sceneParser) parseLine(line string, params map[string]float64, cx, cz float64) error {
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return nil
	}
	args := parseKVArgs(fields[1:])
	switch fields[0] {
	case "camera":
		return p.doCamera(args)
	case "quad":
		return p.doQuad(args, params, cx, cz)
	case "fill":
		return p.doFill(args, params, cx, cz)
	case "box":
		return p.doBox(args, params, cx, cz)
	case "roof":
		return p.doRoof(args, params, cx, cz)
	case "ngon":
		return p.doNgon(args, params, cx, cz)
	case "disk":
		return p.doDisk(args, params, cx, cz)
	default:
		if def, ok := p.defs[fields[0]]; ok {
			return p.instantiate(def, args, params)
		}
		return fmt.Errorf("unknown keyword %q", fields[0])
	}
}

func parseKVArgs(fields []string) map[string]string {
	m := make(map[string]string, len(fields))
	for _, f := range fields {
		kv := strings.SplitN(f, "=", 2)
		if len(kv) == 2 {
			m[kv[0]] = kv[1]
		}
	}
	return m
}

func (p *sceneParser) instantiate(def *shapeDef, args map[string]string, outerParams map[string]float64) error {
	params := make(map[string]float64, len(def.defaults))
	for k, v := range def.defaults {
		params[k] = v
	}
	for k, v := range args {
		if k == "cx" {
			continue
		}
		if fv, err := strconv.ParseFloat(v, 64); err == nil {
			params[k] = fv
		}
	}
	cx, cz := 0.0, 0.0
	if s, ok := args["cx"]; ok {
		parts := strings.SplitN(s, ",", 2)
		if len(parts) == 2 {
			cx, _ = strconv.ParseFloat(parts[0], 64)
			cz, _ = strconv.ParseFloat(parts[1], 64)
		}
	}
	for _, bodyLine := range def.body {
		if err := p.parseLine(bodyLine, params, cx, cz); err != nil {
			return err
		}
	}
	return nil
}

func (p *sceneParser) doCamera(args map[string]string) error {
	if s, ok := args["pos"]; ok {
		parts := strings.SplitN(s, ",", 3)
		if len(parts) == 3 {
			x, _ := strconv.ParseFloat(parts[0], 64)
			y, _ := strconv.ParseFloat(parts[1], 64)
			z, _ := strconv.ParseFloat(parts[2], 64)
			p.cam.pos = vec3{x, y, z}
		}
	}
	if s, ok := args["yaw"]; ok {
		p.cam.yaw, _ = strconv.ParseFloat(s, 64)
	}
	if s, ok := args["pitch"]; ok {
		p.cam.pitch, _ = strconv.ParseFloat(s, 64)
	}
	p.cam.valid = true
	return nil
}

func (p *sceneParser) doQuad(args map[string]string, params map[string]float64, cx, cz float64) error {
	x0, z0, err := p.coord(args["x"], params, cx, cz)
	if err != nil {
		return err
	}
	x1, z1, err := p.coord(args["to"], params, cx, cz)
	if err != nil {
		return err
	}
	y, err := p.expr(args["y"], params)
	if err != nil {
		return err
	}
	cols, err := p.colors(args["col"], 1)
	if err != nil {
		return err
	}
	p.out = append(p.out, quad(vec3{x0, y, z0}, vec3{x1, y, z0}, vec3{x1, y, z1}, vec3{x0, y, z1}, cols[0])...)
	return nil
}

func (p *sceneParser) doFill(args map[string]string, params map[string]float64, cx, cz float64) error {
	x0, z0, err := p.coord(args["x"], params, cx, cz)
	if err != nil {
		return err
	}
	x1, z1, err := p.coord(args["to"], params, cx, cz)
	if err != nil {
		return err
	}
	y, err := p.expr(args["y"], params)
	if err != nil {
		return err
	}
	cols, err := p.colors(args["col"], 1)
	if err != nil {
		return err
	}
	cellParts := strings.SplitN(args["cell"], ",", 2)
	cellW, _ := strconv.ParseFloat(cellParts[0], 64)
	cellH := cellW
	if len(cellParts) == 2 {
		cellH, _ = strconv.ParseFloat(cellParts[1], 64)
	}
	pattern := args["pattern"]
	ix := 0
	for gx := x0; gx < x1-1e-9; gx += cellW {
		iz := 0
		for gz := z0; gz < z1-1e-9; gz += cellH {
			if pattern != "checker" || (ix+iz)%2 == 0 {
				p.out = append(p.out, quad(vec3{gx, y, gz}, vec3{gx + cellW, y, gz}, vec3{gx + cellW, y, gz + cellH}, vec3{gx, y, gz + cellH}, cols[0])...)
			}
			iz++
		}
		ix++
	}
	return nil
}

func (p *sceneParser) doBox(args map[string]string, params map[string]float64, cx, cz float64) error {
	x0, z0, err := p.coord(args["x"], params, cx, cz)
	if err != nil {
		return err
	}
	x1, z1, err := p.coord(args["to"], params, cx, cz)
	if err != nil {
		return err
	}
	yb, yt, err := p.rangeExpr(args["y"], params)
	if err != nil {
		return err
	}
	cols, err := p.colors(args["col"], 5)
	if err != nil {
		return err
	}
	p.out = append(p.out, box(x0, z0, x1, z1, yb, yt, cols[0], cols[1], cols[2], cols[3], cols[4])...)
	return nil
}

func (p *sceneParser) doRoof(args map[string]string, params map[string]float64, cx, cz float64) error {
	rcx, rcz, err := p.coord(args["cx"], params, cx, cz)
	if err != nil {
		return err
	}
	hw, err := p.expr(args["hw"], params)
	if err != nil {
		return err
	}
	hd, err := p.expr(args["hd"], params)
	if err != nil {
		return err
	}
	yb, yt, err := p.rangeExpr(args["y"], params)
	if err != nil {
		return err
	}
	cols, err := p.colors(args["col"], 4)
	if err != nil {
		return err
	}
	p.out = append(p.out, pyramidRoof(rcx, rcz, hw, hd, yb, yt, cols[0], cols[1], cols[2], cols[3])...)
	return nil
}

func (p *sceneParser) doNgon(args map[string]string, params map[string]float64, cx, cz float64) error {
	ncx, ncz, err := p.coord(args["cx"], params, cx, cz)
	if err != nil {
		return err
	}
	r, err := p.expr(args["r"], params)
	if err != nil {
		return err
	}
	y0, y1, err := p.rangeExpr(args["y"], params)
	if err != nil {
		return err
	}
	sides, _ := strconv.Atoi(args["sides"])
	if sides < 3 {
		sides = 6
	}
	cols, err := p.colors(args["col"], 1)
	if err != nil {
		return err
	}
	for i := 0; i < sides; i++ {
		a0 := float64(i) * 2 * math.Pi / float64(sides)
		a1 := float64(i+1) * 2 * math.Pi / float64(sides)
		p.out = append(p.out, quad(
			vec3{ncx + r*math.Cos(a0), y0, ncz + r*math.Sin(a0)},
			vec3{ncx + r*math.Cos(a0), y1, ncz + r*math.Sin(a0)},
			vec3{ncx + r*math.Cos(a1), y1, ncz + r*math.Sin(a1)},
			vec3{ncx + r*math.Cos(a1), y0, ncz + r*math.Sin(a1)},
			cols[0])...)
	}
	return nil
}

func (p *sceneParser) doDisk(args map[string]string, params map[string]float64, cx, cz float64) error {
	dcx, dcz, err := p.coord(args["cx"], params, cx, cz)
	if err != nil {
		return err
	}
	r, err := p.expr(args["r"], params)
	if err != nil {
		return err
	}
	y, err := p.expr(args["y"], params)
	if err != nil {
		return err
	}
	sides, _ := strconv.Atoi(args["sides"])
	if sides < 3 {
		sides = 8
	}
	cols, err := p.colors(args["col"], 1)
	if err != nil {
		return err
	}
	center := vec3{dcx, y, dcz}
	for i := 0; i < sides; i++ {
		a0 := float64(i) * 2 * math.Pi / float64(sides)
		a1 := float64(i+1) * 2 * math.Pi / float64(sides)
		p.out = append(p.out, tri{v: [3]vec3{center, {dcx + r*math.Cos(a0), y, dcz + r*math.Sin(a0)}, {dcx + r*math.Cos(a1), y, dcz + r*math.Sin(a1)}}, c: cols[0]})
	}
	return nil
}

func (p *sceneParser) expr(s string, params map[string]float64) (float64, error) {
	return evalExpr(strings.TrimSpace(s), params)
}

func evalExpr(s string, params map[string]float64) (float64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty expression")
	}
	if v, err := strconv.ParseFloat(s, 64); err == nil {
		return v, nil
	}
	isOp := func(c byte) bool { return c == '+' || c == '-' || c == '*' || c == '/' }
	for _, opset := range []string{"+-", "*/"} {
		for i := 1; i < len(s); i++ {
			c := s[i]
			if strings.ContainsRune(opset, rune(c)) && !isOp(s[i-1]) {
				lhs, err := evalExpr(s[:i], params)
				if err != nil {
					return 0, err
				}
				rhs, err := evalExpr(s[i+1:], params)
				if err != nil {
					return 0, err
				}
				switch c {
				case '+':
					return lhs + rhs, nil
				case '-':
					return lhs - rhs, nil
				case '*':
					return lhs * rhs, nil
				case '/':
					if rhs == 0 {
						return 0, fmt.Errorf("division by zero")
					}
					return lhs / rhs, nil
				}
			}
		}
	}
	if strings.HasPrefix(s, "$") {
		if params != nil {
			if v, ok := params[s[1:]]; ok {
				return v, nil
			}
		}
		return 0, fmt.Errorf("unknown param %s", s)
	}
	return 0, fmt.Errorf("cannot evaluate %q", s)
}

func (p *sceneParser) rangeExpr(s string, params map[string]float64) (float64, float64, error) {
	parts := strings.SplitN(s, "..", 2)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("expected y0..y1, got %q", s)
	}
	y0, err := evalExpr(parts[0], params)
	if err != nil {
		return 0, 0, err
	}
	y1, err := evalExpr(parts[1], params)
	if err != nil {
		return 0, 0, err
	}
	return y0, y1, nil
}

func (p *sceneParser) coord(s string, params map[string]float64, cx, cz float64) (float64, float64, error) {
	parts := strings.SplitN(s, ",", 2)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("expected x,z got %q", s)
	}
	x, err := evalExpr(parts[0], params)
	if err != nil {
		return 0, 0, err
	}
	z, err := evalExpr(parts[1], params)
	if err != nil {
		return 0, 0, err
	}
	return x + cx, z + cz, nil
}

func (p *sceneParser) colors(s string, n int) ([]col3, error) {
	names := strings.SplitN(s, ",", n+1)
	if len(names) == 1 && n > 1 {
		single := strings.TrimSpace(names[0])
		names = make([]string, n)
		for i := range names {
			names[i] = single
		}
	}
	if len(names) != n {
		return nil, fmt.Errorf("expected %d colors, got %d in %q", n, len(names), s)
	}
	out := make([]col3, n)
	for i, name := range names {
		name = strings.TrimSpace(name)
		c, ok := p.pal[name]
		if !ok {
			return nil, fmt.Errorf("unknown color %q", name)
		}
		out[i] = c
	}
	return out, nil
}

func parseHexColor(s string) (col3, error) {
	s = strings.TrimPrefix(s, "#")
	if len(s) != 6 {
		return col3{}, fmt.Errorf("expected 6-digit hex, got %q", s)
	}
	r, err1 := strconv.ParseUint(s[0:2], 16, 8)
	g, err2 := strconv.ParseUint(s[2:4], 16, 8)
	b, err3 := strconv.ParseUint(s[4:6], 16, 8)
	if err1 != nil || err2 != nil || err3 != nil {
		return col3{}, fmt.Errorf("invalid hex %q", s)
	}
	return col3{uint8(r), uint8(g), uint8(b)}, nil
}

// ── Main ──────────────────────────────────────────────────────────────────────

func main() {
	autotest := flag.String("autotest", "", "path to test script JSON (run and exit)")
	flag.Parse()

	g := newGame()

	if err := willow.Run(g.willow, willow.RunConfig{
		Title:        "Village — Willow Testbed",
		Width:        screenW,
		Height:       screenH,
		ShowFPS:      true,
		AutoTestPath: *autotest,
	}); err != nil {
		log.Fatal(err)
	}
}
