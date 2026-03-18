// EXPERIMENTAL — Multi-Channel Signed Distance Field (MSDF) font generation.
//
// This file contains a complete MSDF pipeline ported from the msdfgen reference
// implementation (Chlumsky, 2016). The pipeline is architecturally correct:
// edge coloring (switchColor with two-channel colors), per-contour distance
// computation with winding-aware contour combiner, error correction, and atlas
// packing. Mathematical tests confirm MSDF produces sharper corner boundaries
// than SDF.
//
// However, the rendered output has inconsistent quality compared to single-
// channel SDF. Specific issues:
//   - Multi-contour glyphs (0, 8, @, %, etc.) exhibit anti-aliasing artifacts
//     from channel divergence between contours.
//   - Error correction that fixes artifacts also destroys the multi-channel
//     signal at corners, negating the MSDF advantage.
//   - The MSDF shader (screenPxRange-based) produces different edge
//     characteristics than the SDF shader (fwidth-based), causing visual
//     inconsistency within the same text block.
//
// Recommendation: Use NewFontFromTTF (single-channel SDF) for production.
// This code is retained for future development. See spec/_archive/plan-msdf-quality.md.

package text

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"math"
	"sort"
	"unicode/utf8"

	"github.com/hajimehoshi/ebiten/v2"
	"golang.org/x/image/font"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
)

// ── 2D vector helpers ────────────────────────────────────────────────────────

type mv2 struct{ x, y float64 }

func (v mv2) add(u mv2) mv2       { return mv2{v.x + u.x, v.y + u.y} }
func (v mv2) sub(u mv2) mv2       { return mv2{v.x - u.x, v.y - u.y} }
func (v mv2) scale(s float64) mv2 { return mv2{v.x * s, v.y * s} }
func (v mv2) dot(u mv2) float64   { return v.x*u.x + v.y*u.y }
func (v mv2) cross(u mv2) float64 { return v.x*u.y - v.y*u.x }
func (v mv2) length() float64     { return math.Sqrt(v.x*v.x + v.y*v.y) }

func clamp01(x float64) float64 {
	if x < 0 {
		return 0
	}
	if x > 1 {
		return 1
	}
	return x
}

// ── Edge types ───────────────────────────────────────────────────────────────

const (
	msdfChanR = 1 << 0
	msdfChanG = 1 << 1
	msdfChanB = 1 << 2
)

type mEdge struct {
	pts  [4]mv2 // [0]=start, then control/end points
	kind int    // 1=line, 2=quad, 3=cubic
	ch   int    // channel bitmask
}

type mContour struct {
	edges []mEdge
}

// ── Signed distance computation ──────────────────────────────────────────────

// edgeDist holds both true and perpendicular signed distances.
type edgeDist struct {
	trueDist float64 // standard closest-point distance (for pseudo-SDF)
	perpDist float64 // perpendicular distance (for per-channel MSDF)
}

// signedDistEdge returns signed distance from p to e (true distance only).
// Kept for backward compatibility.
func signedDistEdge(e mEdge, p mv2) float64 {
	return signedDistEdgeFull(e, p).trueDist
}

// signedDistEdgeFull returns both true and perpendicular distances.
// The perpendicular distance extends the edge's influence beyond its
// endpoints along the tangent line, eliminating distance field ridges
// at vertices where edges meet.
func signedDistEdgeFull(e mEdge, p mv2) edgeDist {
	switch e.kind {
	case 1:
		return sdLineFull(p, e.pts[0], e.pts[1])
	case 2:
		if isQuadDegenerate(e.pts[0], e.pts[1], e.pts[2]) {
			return sdLineFull(p, e.pts[0], e.pts[2])
		}
		return sdQuadFull(p, e.pts[0], e.pts[1], e.pts[2])
	default:
		if isCubicDegenerate(e.pts[0], e.pts[1], e.pts[2], e.pts[3]) {
			return sdLineFull(p, e.pts[0], e.pts[3])
		}
		return sdCubicFull(p, e.pts[0], e.pts[1], e.pts[2], e.pts[3])
	}
}

// distToPerpendicularDist converts a true distance to perpendicular distance
// at an edge endpoint. This is a direct port of msdfgen's
// EdgeSegment::distanceToPerpendicularDistance.
//
// The logic: when the closest point on the edge is at an endpoint (param
// clamped to 0 or 1), check if the query point is in the "shadow" of the
// tangent line at that endpoint. If it is AND the perpendicular distance
// is smaller than the true distance, use the perpendicular distance.
//
// The shadow check (ts < 0 at start, ts > 0 at end) prevents the
// perpendicular extension from reaching through thin features to the
// other side — which was the bug in our previous unbounded implementation.
func distToPerpendicularDist(trueDist float64, p, edgePt, tangentDir mv2, atStart bool) float64 {
	dir := tangentDir
	dLen := dir.length()
	if dLen < 1e-12 {
		return trueDist
	}
	dir = dir.scale(1.0 / dLen) // normalize

	d := p.sub(edgePt)
	ts := d.dot(dir)

	// Shadow check: at start endpoint, perpendicular distance only applies
	// when ts < 0 (point is behind the edge start, in the vertex shadow).
	// At end endpoint, only when ts > 0 (point is ahead of the edge end).
	if atStart && ts >= 0 {
		return trueDist
	}
	if !atStart && ts <= 0 {
		return trueDist
	}

	perpDist := dir.cross(d)

	// Bound: only use if |perpDist| <= |trueDist|.
	if math.Abs(perpDist) <= math.Abs(trueDist) {
		return perpDist
	}
	return trueDist
}

// contourWinding returns +1 for CCW (outer) contours, -1 for CW (inner/counter)
// contours, using the shoelace formula on edge start points.
func contourWinding(c mContour) int {
	if len(c.edges) == 0 {
		return 0
	}
	total := 0.0
	for i := range c.edges {
		next := (i + 1) % len(c.edges)
		a := edgeStartPt(c.edges[i])
		b := edgeStartPt(c.edges[next])
		total += (b.x - a.x) * (b.y + a.y) // shoelace
	}
	if total > 0 {
		return 1
	}
	if total < 0 {
		return -1
	}
	return 0
}

func edgeStartPt(e mEdge) mv2 { return e.pts[0] }
func edgeEndPt(e mEdge) mv2   { return e.pts[e.kind] }

func sameSign(a, b float64) bool {
	return (a >= 0) == (b >= 0)
}

const degenerateEps = 1e-6

// isQuadDegenerate returns true if the quadratic control point is collinear
// with the endpoints (making it effectively a line segment).
func isQuadDegenerate(a, ctrl, b mv2) bool {
	ab := b.sub(a)
	len2 := ab.dot(ab)
	if len2 < degenerateEps*degenerateEps {
		return true // zero-length curve
	}
	// Distance from control point to the line a→b.
	ac := ctrl.sub(a)
	cross := math.Abs(ab.cross(ac))
	return cross/math.Sqrt(len2) < degenerateEps
}

// isCubicDegenerate returns true if all control points are collinear with
// the endpoints (making it effectively a line segment).
func isCubicDegenerate(a, c1, c2, b mv2) bool {
	ab := b.sub(a)
	len2 := ab.dot(ab)
	if len2 < degenerateEps*degenerateEps {
		return true
	}
	sqrtLen := math.Sqrt(len2)
	d1 := math.Abs(ab.cross(c1.sub(a))) / sqrtLen
	d2 := math.Abs(ab.cross(c2.sub(a))) / sqrtLen
	return d1 < degenerateEps && d2 < degenerateEps
}

func sdLine(p, a, b mv2) float64 {
	return sdLineFull(p, a, b).trueDist
}

func sdLineFull(p, a, b mv2) edgeDist {
	ab := b.sub(a)
	ap := p.sub(a)
	len2 := ab.dot(ab)
	tRaw := 0.0
	if len2 > 1e-12 {
		tRaw = ap.dot(ab) / len2
	}
	t := clamp01(tRaw)
	c := a.add(ab.scale(t))
	d := p.sub(c)
	dist := d.length()
	if ab.cross(d) < 0 {
		dist = -dist
	}

	// Convert to perpendicular distance at endpoints (msdfgen algorithm).
	pDist := dist
	if tRaw < 0 {
		pDist = distToPerpendicularDist(dist, p, a, ab, true)
	} else if tRaw > 1 {
		pDist = distToPerpendicularDist(dist, p, b, ab, false)
	}
	return edgeDist{trueDist: dist, perpDist: pDist}
}

func evalQuad(t float64, a, ctrl, b mv2) mv2 {
	mt := 1 - t
	return a.scale(mt * mt).add(ctrl.scale(2 * mt * t)).add(b.scale(t * t))
}

func tangentQuad(t float64, a, ctrl, b mv2) mv2 {
	return ctrl.sub(a).scale(2 * (1 - t)).add(b.sub(ctrl).scale(2 * t))
}

func evalCubic(t float64, a, c1, c2, b mv2) mv2 {
	mt := 1 - t
	return a.scale(mt * mt * mt).add(c1.scale(3 * mt * mt * t)).add(c2.scale(3 * mt * t * t)).add(b.scale(t * t * t))
}

func tangentCubic(t float64, a, c1, c2, b mv2) mv2 {
	mt := 1 - t
	return c1.sub(a).scale(3 * mt * mt).add(c2.sub(c1).scale(6 * mt * t)).add(b.sub(c2).scale(3 * t * t))
}

func sdQuad(p, a, ctrl, b mv2) float64 {
	return sdQuadFull(p, a, ctrl, b).trueDist
}

func sdQuadFull(p, a, ctrl, b mv2) edgeDist {
	best, bestD2 := 0.0, math.MaxFloat64
	for i := 0; i <= 8; i++ {
		t := float64(i) / 8.0
		pt := evalQuad(t, a, ctrl, b)
		if d2 := p.sub(pt).dot(p.sub(pt)); d2 < bestD2 {
			bestD2, best = d2, t
		}
	}
	bpp := a.add(b).add(ctrl.scale(-2)).scale(2)
	t := best
	for range 4 {
		bt := evalQuad(t, a, ctrl, b)
		tang := tangentQuad(t, a, ctrl, b)
		diff := p.sub(bt)
		den := tang.dot(tang) - diff.dot(bpp)
		if math.Abs(den) < 1e-10 {
			break
		}
		t = clamp01(t + diff.dot(tang)/den)
	}
	pt := evalQuad(t, a, ctrl, b)
	tang := tangentQuad(t, a, ctrl, b)
	d := p.sub(pt)
	dist := d.length()
	if tang.cross(d) < 0 {
		dist = -dist
	}

	pDist := dist
	if t <= 0 {
		pDist = distToPerpendicularDist(dist, p, a, tangentQuad(0, a, ctrl, b), true)
	} else if t >= 1 {
		pDist = distToPerpendicularDist(dist, p, b, tangentQuad(1, a, ctrl, b), false)
	}
	return edgeDist{trueDist: dist, perpDist: pDist}
}

func sdCubic(p, a, c1, c2, b mv2) float64 {
	return sdCubicFull(p, a, c1, c2, b).trueDist
}

func sdCubicFull(p, a, c1, c2, b mv2) edgeDist {
	best, bestD2 := 0.0, math.MaxFloat64
	for i := 0; i <= 12; i++ {
		t := float64(i) / 12.0
		pt := evalCubic(t, a, c1, c2, b)
		if d2 := p.sub(pt).dot(p.sub(pt)); d2 < bestD2 {
			bestD2, best = d2, t
		}
	}
	t := best
	for range 4 {
		bt := evalCubic(t, a, c1, c2, b)
		tang := tangentCubic(t, a, c1, c2, b)
		mt := 1 - t
		bpp := c1.sub(a).scale(-6 * mt).add(c2.sub(c1).scale(6 * (1 - 2*t))).add(b.sub(c2).scale(6 * t))
		diff := p.sub(bt)
		den := tang.dot(tang) - diff.dot(bpp)
		if math.Abs(den) < 1e-10 {
			break
		}
		t = clamp01(t + diff.dot(tang)/den)
	}
	pt := evalCubic(t, a, c1, c2, b)
	tang := tangentCubic(t, a, c1, c2, b)
	d := p.sub(pt)
	dist := d.length()
	if tang.cross(d) < 0 {
		dist = -dist
	}

	pDist := dist
	if t <= 0 {
		pDist = distToPerpendicularDist(dist, p, a, tangentCubic(0, a, c1, c2, b), true)
	} else if t >= 1 {
		pDist = distToPerpendicularDist(dist, p, b, tangentCubic(1, a, c1, c2, b), false)
	}
	return edgeDist{trueDist: dist, perpDist: pDist}
}

// ── Edge coloring ────────────────────────────────────────────────────────────

// colorEdgesCyclic assigns R/G/B channels in a cyclic pattern.
// Adjacent edges always get different channels — the key MSDF requirement.
func colorEdgesCyclic(contours []mContour) {
	colors := [3]int{msdfChanR, msdfChanG, msdfChanB}
	ci := 0
	for i := range contours {
		for j := range contours[i].edges {
			contours[i].edges[j].ch = colors[ci%3]
			ci++
		}
	}
}

// edgeDirection returns the tangent direction at the start (t=0) or end (t=1)
// of an edge, used for corner detection.
func edgeDirection(e mEdge, atEnd bool) mv2 {
	switch e.kind {
	case 1: // line
		d := e.pts[1].sub(e.pts[0])
		if atEnd {
			return d
		}
		return d
	case 2: // quadratic
		if atEnd {
			return tangentQuad(1, e.pts[0], e.pts[1], e.pts[2])
		}
		return tangentQuad(0, e.pts[0], e.pts[1], e.pts[2])
	case 3: // cubic
		if atEnd {
			return tangentCubic(1, e.pts[0], e.pts[1], e.pts[2], e.pts[3])
		}
		return tangentCubic(0, e.pts[0], e.pts[1], e.pts[2], e.pts[3])
	}
	return mv2{1, 0}
}

// cornerAngle returns the angle between the outgoing direction of edge a and
// the incoming direction of edge b. Sharp corners have small angles.
func cornerAngle(a, b mEdge) float64 {
	d1 := edgeDirection(a, true)  // end of a
	d2 := edgeDirection(b, false) // start of b
	len1 := d1.length()
	len2 := d2.length()
	if len1 < 1e-10 || len2 < 1e-10 {
		return math.Pi // degenerate → treat as smooth
	}
	cos := d1.dot(d2) / (len1 * len2)
	if cos > 1 {
		cos = 1
	}
	if cos < -1 {
		cos = -1
	}
	return math.Acos(cos)
}

// colorEdgesInkTrap assigns channels per-contour with corner-aware coloring.
// Kept for backward compatibility with tests. Production code uses
// colorEdgesSimple.
func colorEdgesInkTrap(contours []mContour) {
	colorEdgesSimple(contours, 3.0)
}

const msdfWhite = msdfChanR | msdfChanG | msdfChanB // 7 = all channels

// Multi-channel color constants (matching msdfgen EdgeColor).
const (
	msdfCyan    = msdfChanG | msdfChanB // 6
	msdfMagenta = msdfChanR | msdfChanB // 5
	msdfYellow  = msdfChanR | msdfChanG // 3
)

// switchColor advances *c to a new color that shares at most one channel
// with the current color and is not equal to banned. Matches msdfgen's
// switchColor exactly.
func switchColor(c *int, seed *uint64, banned int) {
	combined := *c & banned
	// If current and banned share exactly one channel, use its complement.
	if combined == msdfChanR || combined == msdfChanG || combined == msdfChanB {
		*c = combined ^ msdfWhite
		return
	}
	// If color is BLACK (0) or WHITE (7), pick from {CYAN, MAGENTA, YELLOW}.
	if *c == 0 || *c == msdfWhite {
		start := [3]int{msdfCyan, msdfMagenta, msdfYellow}
		*c = start[*seed%3]
		*seed /= 3
		return
	}
	// Bit-rotate by 1 or 2 positions within 3-bit color space.
	shift := 1 + int(*seed&1)
	*seed >>= 1
	shifted := *c << shift
	*c = (shifted | (shifted >> 3)) & msdfWhite
}

// isCornerVertex returns true if the junction between edges a and b is a
// corner, using both dot-product and cross-product tests (matching msdfgen).
func isCornerVertex(a, b mEdge, angleThreshold float64) bool {
	d1 := edgeDirection(a, true)
	d2 := edgeDirection(b, false)
	len1 := d1.length()
	len2 := d2.length()
	if len1 < 1e-10 || len2 < 1e-10 {
		return false
	}
	n1 := d1.scale(1 / len1)
	n2 := d2.scale(1 / len2)
	dot := n1.dot(n2)
	cross := n1.cross(n2)
	crossThresh := math.Sin(angleThreshold)
	return dot <= 0 || math.Abs(cross) > crossThresh
}

// splitEdgeAt subdivides a bezier edge at parameter t, returning two edges.
func splitEdgeAt(e mEdge, t float64) (mEdge, mEdge) {
	switch e.kind {
	case 1: // line
		mid := e.pts[0].add(e.pts[1].sub(e.pts[0]).scale(t))
		return mEdge{pts: [4]mv2{e.pts[0], mid}, kind: 1},
			mEdge{pts: [4]mv2{mid, e.pts[1]}, kind: 1}
	case 2: // quadratic — de Casteljau
		a, ctrl, b := e.pts[0], e.pts[1], e.pts[2]
		p01 := a.add(ctrl.sub(a).scale(t))
		p12 := ctrl.add(b.sub(ctrl).scale(t))
		p012 := p01.add(p12.sub(p01).scale(t))
		return mEdge{pts: [4]mv2{a, p01, p012}, kind: 2},
			mEdge{pts: [4]mv2{p012, p12, b}, kind: 2}
	case 3: // cubic — de Casteljau
		a, c1, c2, b := e.pts[0], e.pts[1], e.pts[2], e.pts[3]
		p01 := a.add(c1.sub(a).scale(t))
		p12 := c1.add(c2.sub(c1).scale(t))
		p23 := c2.add(b.sub(c2).scale(t))
		p012 := p01.add(p12.sub(p01).scale(t))
		p123 := p12.add(p23.sub(p12).scale(t))
		p0123 := p012.add(p123.sub(p012).scale(t))
		return mEdge{pts: [4]mv2{a, p01, p012, p0123}, kind: 3},
			mEdge{pts: [4]mv2{p0123, p123, p23, b}, kind: 3}
	}
	return e, e
}

// colorEdgesSimple implements msdfgen's edgeColoringSimple algorithm.
// It processes each contour independently, identifies corners, and assigns
// colors with the switchColor mechanism that guarantees:
// - Adjacent edges at a corner have different colors sharing ≤1 channel
// - The wrap-around constraint is enforced (last spline ≠ first spline)
// - Teardrop contours (1 corner) get edge splitting + WHITE assignment
func colorEdgesSimple(contours []mContour, angleThreshold float64) {
	var seed uint64 = 0

	for ci := range contours {
		edges := contours[ci].edges
		n := len(edges)
		if n == 0 {
			continue
		}

		// Find corners.
		var corners []int
		for i := range edges {
			next := (i + 1) % n
			if isCornerVertex(edges[i], edges[next], angleThreshold) {
				corners = append(corners, i)
			}
		}

		if len(corners) == 0 {
			// Smooth contour — all edges get one two-channel color.
			color := msdfWhite
			switchColor(&color, &seed, 0)
			for i := range edges {
				edges[i].ch = color
			}
			continue
		}

		if len(corners) == 1 {
			// Teardrop case: split edges if needed to get ≥3 edges,
			// then assign via symmetrical trichotomy.
			cornerIdx := corners[0]

			// Ensure at least 3 edges by splitting at the corner edge.
			for len(contours[ci].edges) < 3 {
				// Split the longest edge.
				bestIdx := 0
				bestLen := 0.0
				for i, e := range contours[ci].edges {
					var l float64
					switch e.kind {
					case 1:
						l = e.pts[1].sub(e.pts[0]).length()
					case 2:
						l = e.pts[2].sub(e.pts[0]).length()
					case 3:
						l = e.pts[3].sub(e.pts[0]).length()
					}
					if l > bestLen {
						bestLen = l
						bestIdx = i
					}
				}
				e1, e2 := splitEdgeAt(contours[ci].edges[bestIdx], 0.5)
				newEdges := make([]mEdge, 0, len(contours[ci].edges)+1)
				newEdges = append(newEdges, contours[ci].edges[:bestIdx]...)
				newEdges = append(newEdges, e1, e2)
				newEdges = append(newEdges, contours[ci].edges[bestIdx+1:]...)
				contours[ci].edges = newEdges
				if bestIdx <= cornerIdx {
					cornerIdx++ // corner shifted right due to insertion
				}
			}
			edges = contours[ci].edges
			n = len(edges)

			// Symmetrical trichotomy: assign colors balanced around the corner.
			// edges before corner → color A, edges around midpoint → WHITE,
			// edges after midpoint → color B
			colorA := msdfChanR
			colorB := msdfChanB
			for i := range edges {
				// Map edge position relative to corner into [0..n-1]
				pos := i - cornerIdx - 1
				if pos < 0 {
					pos += n
				}
				// Divide into thirds
				third := pos * 3 / n
				switch third {
				case 0:
					edges[i].ch = colorA
				case 1:
					edges[i].ch = msdfWhite
				default:
					edges[i].ch = colorB
				}
			}
			continue
		}

		// Multiple corners: assign colors between consecutive corners.
		// Start with WHITE so switchColor picks from {CYAN, MAGENTA, YELLOW}
		// (two-channel colors), matching msdfgen. Two-channel colors ensure
		// each edge writes to 2 of 3 RGB channels, so every pixel near the
		// edge has at least 2 channels with meaningful distance data.
		color := msdfWhite
		switchColor(&color, &seed, 0) // → picks CYAN, MAGENTA, or YELLOW
		initialColor := color

		// Walk edges starting from the edge after the first corner.
		for ci2, cornerI := range corners {
			nextCornerI := corners[(ci2+1)%len(corners)]

			// Determine banned color for switchColor.
			// At the last corner: banned = initial color (wrap-around constraint).
			banned := 0
			if ci2 == len(corners)-1 {
				banned = initialColor
			}
			switchColor(&color, &seed, banned)

			// Assign this color to all edges from cornerI+1 to nextCornerI.
			start := cornerI + 1
			end := nextCornerI + 1
			if end <= start {
				end += n
			}
			for j := start; j < end; j++ {
				edges[j%n].ch = color
			}
		}
	}
}

// msdfErrorCorrect fixes median artifacts by comparing the MSDF median against
// a reference pseudo-SDF (the closest edge distance regardless of channel).
// Where they disagree on inside/outside, all channels are clamped to the
// pseudo-SDF value, effectively reverting that pixel to single-channel SDF.
// This eliminates false-positive artifacts at thin features and sharp vertices.
func msdfErrorCorrect(img *image.NRGBA, distRange float64) {
	// Without pseudo-SDF data, fall back to the neighbor-based method.
	msdfErrorCorrectNeighbor(img)
}

// msdfErrorCorrectWithPseudo is the preferred error correction that uses
// the pseudo-SDF computed alongside the MSDF as ground truth.
func msdfErrorCorrectWithPseudo(img *image.NRGBA, pseudo []float64, w, h int) {
	b := img.Bounds()
	threshold := 0.5

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := img.NRGBAAt(b.Min.X+x, b.Min.Y+y)
			r := float64(c.R) / 255
			g := float64(c.G) / 255
			bl := float64(c.B) / 255
			med := median3Float(r, g, bl)
			psd := pseudo[y*w+x]

			medInside := med > threshold
			psdInside := psd > threshold

			if medInside != psdInside {
				// Median and pseudo-SDF disagree on inside/outside.
				// Correct to pseudo-SDF but PRESERVE the gradient by
				// shifting all channels by the same offset rather than
				// clamping to a flat value. This maintains inter-channel
				// relationships (needed for anti-aliasing) while fixing
				// the inside/outside classification.
				shift := psd - med
				newR := clamp01(r + shift)
				newG := clamp01(g + shift)
				newB := clamp01(bl + shift)
				img.SetNRGBA(b.Min.X+x, b.Min.Y+y, color.NRGBA{
					R: uint8(newR*255 + 0.5),
					G: uint8(newG*255 + 0.5),
					B: uint8(newB*255 + 0.5),
					A: 255,
				})
			}
		}
	}
}

// msdfErrorCorrectNeighbor is a fallback error correction for when pseudo-SDF
// data is not available. It checks each pixel's median against 8-neighbors.
func msdfErrorCorrectNeighbor(img *image.NRGBA) {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()

	medians := make([]float64, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := img.NRGBAAt(b.Min.X+x, b.Min.Y+y)
			r := float64(c.R) / 255
			g := float64(c.G) / 255
			bl := float64(c.B) / 255
			medians[y*w+x] = median3Float(r, g, bl)
		}
	}

	threshold := 0.5
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			med := medians[y*w+x]
			inside := med > threshold

			allDisagree := true
			count := 0
			for dy := -1; dy <= 1; dy++ {
				for dx := -1; dx <= 1; dx++ {
					if dx == 0 && dy == 0 {
						continue
					}
					nx, ny := x+dx, y+dy
					if nx < 0 || ny < 0 || nx >= w || ny >= h {
						continue
					}
					count++
					if (medians[ny*w+nx] > threshold) == inside {
						allDisagree = false
					}
				}
			}

			if !allDisagree || count == 0 {
				continue
			}

			pU8 := distFloatToU8(med)
			img.SetNRGBA(b.Min.X+x, b.Min.Y+y, color.NRGBA{
				R: pU8, G: pU8, B: pU8, A: 255,
			})
		}
	}
}

// msdfErrorCorrectBilinear simulates GPU bilinear interpolation on every 2x2
// pixel group. If the interpolated median disagrees with the interpolated
// pseudo-SDF on inside/outside, the anchor pixel is clamped. This catches
// artifacts that only appear when the GPU samples between texels.
func msdfErrorCorrectBilinear(img *image.NRGBA, pseudo []float64, w, h int) {
	b := img.Bounds()
	threshold := 0.5

	// Pre-extract channels as float64 arrays for fast neighbor access.
	rr := make([]float64, w*h)
	gg := make([]float64, w*h)
	bb := make([]float64, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := img.NRGBAAt(b.Min.X+x, b.Min.Y+y)
			idx := y*w + x
			rr[idx] = float64(c.R) / 255
			gg[idx] = float64(c.G) / 255
			bb[idx] = float64(c.B) / 255
		}
	}

	// Check every pixel as the top-left of a 2x2 bilinear group.
	// Also check as top-right, bottom-left, bottom-right by looking at all
	// four 2x2 groups that include this pixel.
	type correction struct {
		x, y int
		val  uint8
	}
	var corrections []correction

	offsets := [4][4][2]int{
		// 2x2 group where (x,y) is top-left
		{{0, 0}, {1, 0}, {0, 1}, {1, 1}},
		// where (x,y) is top-right
		{{-1, 0}, {0, 0}, {-1, 1}, {0, 1}},
		// where (x,y) is bottom-left
		{{0, -1}, {1, -1}, {0, 0}, {1, 0}},
		// where (x,y) is bottom-right
		{{-1, -1}, {0, -1}, {-1, 0}, {0, 0}},
	}

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			idx := y*w + x
			psd := pseudo[idx]

			needsCorrection := false
			for _, off := range offsets {
				// Check if all four corners are in bounds.
				valid := true
				var sr, sg, sb, sp float64
				for _, d := range off {
					nx, ny := x+d[0], y+d[1]
					if nx < 0 || ny < 0 || nx >= w || ny >= h {
						valid = false
						break
					}
					ni := ny*w + nx
					sr += rr[ni]
					sg += gg[ni]
					sb += bb[ni]
					sp += pseudo[ni]
				}
				if !valid {
					continue
				}
				// Average (bilinear interpolation at texel center).
				sr /= 4
				sg /= 4
				sb /= 4
				sp /= 4
				interpMed := median3Float(sr, sg, sb)
				if (interpMed > threshold) != (sp > threshold) {
					needsCorrection = true
					break
				}
			}

			if needsCorrection {
				corrections = append(corrections, correction{x, y, distFloatToU8(psd)})
			}
		}
	}

	// Apply corrections after scanning (avoid modifying during iteration).
	for _, c := range corrections {
		img.SetNRGBA(b.Min.X+c.x, b.Min.Y+c.y, color.NRGBA{
			R: c.val, G: c.val, B: c.val, A: 255,
		})
	}
}

// fixSignsByScanline corrects pseudo-SDF signs using horizontal ray casting.
// For each row, it counts edge crossings to determine true inside/outside
// via the nonzero winding rule, then flips signs that disagree.
// pad is the padding added around the glyph (typically int(distRange)+1).
func fixSignsByScanline(pseudo []float64, w, h, x0, y0, pad int, contours []mContour) {
	threshold := 0.5

	for py := 0; py < h; py++ {
		// Map image pixel to world coordinate (same as computeMSDFWithPseudo).
		wy := float64(py-pad+y0) + 0.5

		for px := 0; px < w; px++ {
			wx := float64(px-pad+x0) + 0.5

			winding := 0
			for ci := range contours {
				for _, e := range contours[ci].edges {
					winding += edgeCrossing(e, wx, wy)
				}
			}

			idx := py*w + px
			scanlineInside := winding != 0
			pseudoInside := pseudo[idx] > threshold

			if scanlineInside != pseudoInside {
				// Flip the pseudo-SDF across the threshold.
				pseudo[idx] = 1.0 - pseudo[idx]
			}
		}
	}
}

// edgeCrossing returns the winding contribution of edge e for a horizontal
// ray cast leftward from point (px, py). Returns +1 for upward crossing,
// -1 for downward crossing, 0 for no crossing.
func edgeCrossing(e mEdge, px, py float64) int {
	switch e.kind {
	case 1: // line
		return lineCrossing(e.pts[0], e.pts[1], px, py)
	case 2: // quadratic — approximate with 4 line segments
		count := 0
		prev := e.pts[0]
		for i := 1; i <= 4; i++ {
			t := float64(i) / 4.0
			pt := evalQuad(t, e.pts[0], e.pts[1], e.pts[2])
			count += lineCrossing(prev, pt, px, py)
			prev = pt
		}
		return count
	case 3: // cubic — approximate with 6 line segments
		count := 0
		prev := e.pts[0]
		for i := 1; i <= 6; i++ {
			t := float64(i) / 6.0
			pt := evalCubic(t, e.pts[0], e.pts[1], e.pts[2], e.pts[3])
			count += lineCrossing(prev, pt, px, py)
			prev = pt
		}
		return count
	}
	return 0
}

// lineCrossing tests if a horizontal ray going left from (px,py) crosses
// the line segment from a to b. Returns +1/-1 for upward/downward crossing.
func lineCrossing(a, b mv2, px, py float64) int {
	// Ensure a.y <= b.y for consistent half-open interval [a.y, b.y).
	if a.y > b.y {
		a, b = b, a
		// Swap reverses the crossing direction.
		if py >= a.y && py < b.y {
			// Compute x at the crossing.
			t := (py - a.y) / (b.y - a.y)
			cx := a.x + t*(b.x-a.x)
			if cx <= px {
				return -1 // downward in original direction
			}
		}
		return 0
	}
	if py >= a.y && py < b.y {
		t := (py - a.y) / (b.y - a.y)
		cx := a.x + t*(b.x-a.x)
		if cx <= px {
			return 1 // upward
		}
	}
	return 0
}

// msdfErrorCorrectEdge corrects texels near the glyph edge (pseudo-SDF close
// to 0.5) where individual channel values diverge wildly from each other.
// These divergent channels cause bilinear interpolation artifacts (glow/halo)
// when the GPU samples between this texel and its neighbor. This is the
// equivalent of msdfgen's msdfFastEdgeErrorCorrection.
func msdfErrorCorrectEdge(img *image.NRGBA, pseudo []float64, w, h int) {
	b := img.Bounds()

	// Correct texels where channels disagree significantly. Wide band
	// catches artifacts far from the edge; tight gap catches small
	// divergences that produce visible halos under bilinear sampling.
	const edgeBand = 0.45   // how close to 0.5 the pseudo-SDF must be
	const channelGap = 0.15 // max allowed gap between any channel and median

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			idx := y*w + x
			psd := pseudo[idx]

			// Only process texels near the shape edge.
			if math.Abs(psd-0.5) > edgeBand {
				continue
			}

			c := img.NRGBAAt(b.Min.X+x, b.Min.Y+y)
			r := float64(c.R) / 255
			g := float64(c.G) / 255
			bl := float64(c.B) / 255
			med := median3Float(r, g, bl)

			// Check if any channel deviates too far from the median.
			maxGap := math.Max(math.Abs(r-med), math.Max(math.Abs(g-med), math.Abs(bl-med)))
			if maxGap > channelGap {
				// Clamp all channels to median — removes divergence while
				// preserving the inside/outside classification.
				medU8 := distFloatToU8(med)
				img.SetNRGBA(b.Min.X+x, b.Min.Y+y, color.NRGBA{
					R: medU8, G: medU8, B: medU8, A: 255,
				})
			}
		}
	}
}

// glyphHasCorners returns true if any contour in the glyph has at least
// one sharp corner (where edge coloring would change channels).
func glyphHasCorners(contours []mContour, angleThreshold float64) bool {
	for _, c := range contours {
		n := len(c.edges)
		if n < 2 {
			continue
		}
		for i := range c.edges {
			next := (i + 1) % n
			if isCornerVertex(c.edges[i], c.edges[next], angleThreshold) {
				return true
			}
		}
	}
	return false
}

// msdfEqualizeToPseudo sets all channels to the pseudo-SDF value for every
// pixel. The pseudo-SDF is the true closest-edge distance, which is smooth
// and produces clean anti-aliasing for round glyphs.
func msdfEqualizeToPseudo(img *image.NRGBA, pseudo []float64, w, h int) {
	b := img.Bounds()
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := img.NRGBAAt(b.Min.X+x, b.Min.Y+y)
			if c.R == 0 && c.G == 0 && c.B == 0 && c.A == 0 {
				continue
			}
			v := distFloatToU8(pseudo[y*w+x])
			img.SetNRGBA(b.Min.X+x, b.Min.Y+y, color.NRGBA{R: v, G: v, B: v, A: 255})
		}
	}
}

// msdfErrorClampOutlierChannels clamps individual channels that are far from
// the pseudo-SDF toward the pseudo-SDF value. This handles the glow/halo
// on round glyphs with counters (like "0", "O") where one contour's channels
// produce high values near the other contour's edge.
//
// Unlike msdfErrorCorrectEdge (which clamped all channels to median based on
// channel gap), this only modifies channels that are on the WRONG SIDE of
// the pseudo-SDF — e.g., a channel reads "deep inside" when the pixel is
// near the edge. The channel is clamped toward the pseudo-SDF value,
// reducing the glow without destroying the anti-aliasing gradient.
func msdfErrorClampOutlierChannels(img *image.NRGBA, pseudo []float64, w, h int) {
	b := img.Bounds()

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			idx := y*w + x
			psd := pseudo[idx]

			// Only near edges (where glow is visible).
			if psd < 0.05 || psd > 0.95 {
				continue
			}

			c := img.NRGBAAt(b.Min.X+x, b.Min.Y+y)
			r := float64(c.R) / 255
			g := float64(c.G) / 255
			bl := float64(c.B) / 255

			// Clamp each channel toward pseudo-SDF. Channels that are much
			// further from 0.5 than the pseudo-SDF create glow artifacts
			// under GPU bilinear interpolation. Pull them toward pseudo-SDF
			// proportionally to the gap.
			clampCh := func(ch float64) uint8 {
				gap := ch - psd
				if math.Abs(gap) > 0.15 {
					// Aggressively clamp: blend 80% toward pseudo-SDF.
					clamped := psd + gap*0.2
					return uint8(clamp01(clamped)*255 + 0.5)
				}
				return uint8(ch*255 + 0.5)
			}

			img.SetNRGBA(b.Min.X+x, b.Min.Y+y, color.NRGBA{
				R: clampCh(r),
				G: clampCh(g),
				B: clampCh(bl),
				A: 255,
			})
		}
	}
}

// ── Stencil-based error correction ────────────────────────────────────────────

type stencilFlag uint8

const (
	stencilNone      stencilFlag = 0
	stencilProtected stencilFlag = 1
	stencilError     stencilFlag = 2
)

// msdfErrorCorrectStencil is the surgical error correction that protects
// corners and real edges from correction, only fixing proven interpolation
// artifacts. This replaces the nuclear msdfErrorCorrectWithPseudo.
//
// contours and pad/x0/y0 are needed to find corner vertices in image space.
func msdfErrorCorrectStencil(img *image.NRGBA, pseudo []float64, w, h, x0, y0, pad int, contours []mContour) {
	b := img.Bounds()
	threshold := 0.5

	// Pre-extract channel values.
	rr := make([]float64, w*h)
	gg := make([]float64, w*h)
	bb := make([]float64, w*h)
	medians := make([]float64, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := img.NRGBAAt(b.Min.X+x, b.Min.Y+y)
			idx := y*w + x
			rr[idx] = float64(c.R) / 255
			gg[idx] = float64(c.G) / 255
			bb[idx] = float64(c.B) / 255
			medians[idx] = median3Float(rr[idx], gg[idx], bb[idx])
		}
	}

	stencil := make([]stencilFlag, w*h)

	// Phase 1: Protect corners.
	// Find every vertex where edge colors change and mark the surrounding
	// 2x2 texels as PROTECTED.
	for _, c := range contours {
		edges := c.edges
		n := len(edges)
		for i := range edges {
			next := (i + 1) % n
			if edges[i].ch != edges[next].ch {
				// This is a color-change vertex. Get its position.
				var vx, vy float64
				switch edges[i].kind {
				case 1:
					vx, vy = edges[i].pts[1].x, edges[i].pts[1].y
				case 2:
					vx, vy = edges[i].pts[2].x, edges[i].pts[2].y
				case 3:
					vx, vy = edges[i].pts[3].x, edges[i].pts[3].y
				}
				// Convert from glyph coords to image coords.
				px := vx - float64(x0) + float64(pad)
				py := vy - float64(y0) + float64(pad)
				// Mark the 2x2 texels surrounding this point.
				for dy := 0; dy <= 1; dy++ {
					for dx := 0; dx <= 1; dx++ {
						ix := int(math.Floor(px)) + dx - 1
						iy := int(math.Floor(py)) + dy - 1
						if ix >= 0 && iy >= 0 && ix < w && iy < h {
							stencil[iy*w+ix] = stencilProtected
						}
					}
				}
			}
		}
	}

	// Phase 2: Find and correct errors.
	// Use magnitude-based correction: if median and pseudo-SDF disagree
	// on inside/outside AND the disagreement magnitude is large, it's an
	// artifact. Small disagreements near the threshold are MSDF corner
	// features that should be preserved.
	//
	// At a real corner, median ≈ pseudo-SDF ≈ 0.5 (the corner is on the
	// edge). The disagreement is tiny. At a ridge artifact, the median
	// can be 0.7 while pseudo-SDF is 0.3 — a large gap.
	//
	// For protected (corner) texels, use a stricter threshold — only
	// correct truly egregious artifacts that would be visible despite
	// being at a corner.
	const artifactThreshold = 0.1       // for non-corner texels
	const cornerArtifactThreshold = 0.3 // for corner texels (much more permissive)

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			idx := y*w + x
			med := medians[idx]
			psd := pseudo[idx]

			// Must disagree on inside/outside.
			if (med > threshold) == (psd > threshold) {
				continue
			}

			gap := math.Abs(med - psd)
			thresh := artifactThreshold
			if stencil[idx] == stencilProtected {
				thresh = cornerArtifactThreshold
			}

			if gap > thresh {
				// Correct to median (preserves inside/outside from MSDF
				// perspective, just removes channel disagreement).
				medU8 := distFloatToU8(med)
				img.SetNRGBA(b.Min.X+x, b.Min.Y+y, color.NRGBA{
					R: medU8, G: medU8, B: medU8, A: 255,
				})
			}
		}
	}
}

func median3Float(a, b, c float64) float64 {
	return math.Max(math.Min(a, b), math.Min(math.Max(a, b), c))
}

func distFloatToU8(v float64) uint8 {
	if v < 0 {
		v = 0
	}
	if v > 1 {
		v = 1
	}
	return uint8(v*255 + 0.5)
}

// ── Per-pixel MSDF computation ───────────────────────────────────────────────

func distToUint8MSDF(d, distRange float64) uint8 {
	n := 0.5 + 0.5*(d/distRange)
	if n < 0 {
		n = 0
	}
	if n > 1 {
		n = 1
	}
	return uint8(n*255 + 0.5)
}

// edgeBBox returns a bounding box for culling edge distance checks.
// Expands by distanceRange so we can skip distant edges.
func edgeBBox(e mEdge, distRange float64) (x0, y0, x1, y1 float64) {
	count := e.kind + 1 // line=2pts, quad=3pts, cubic=4pts
	x0, y0 = math.MaxFloat64, math.MaxFloat64
	x1, y1 = -math.MaxFloat64, -math.MaxFloat64
	for i := 0; i < count; i++ {
		if e.pts[i].x < x0 {
			x0 = e.pts[i].x
		}
		if e.pts[i].y < y0 {
			y0 = e.pts[i].y
		}
		if e.pts[i].x > x1 {
			x1 = e.pts[i].x
		}
		if e.pts[i].y > y1 {
			y1 = e.pts[i].y
		}
	}
	x0 -= distRange
	y0 -= distRange
	x1 += distRange
	y1 += distRange
	return
}

// msdfResult holds the MSDF image and the per-pixel pseudo-SDF used for
// error correction. The pseudo-SDF is the closest edge distance regardless
// of channel — it serves as ground truth for inside/outside classification.
type msdfResult struct {
	img      *image.NRGBA
	pseudoSD []float64 // normalized [0,1], 0.5 = edge
	w, h     int
}

// computeMSDF generates an NRGBA MSDF image for a glyph.
// x0, y0 are the top-left corner of the glyph bbox in font coordinates.
// gw, gh are the glyph dimensions. pad pixels of padding are added on all sides.
// Also returns a pseudo-SDF for error correction.
func computeMSDF(contours []mContour, x0, y0, gw, gh int, distRange float64) *image.NRGBA {
	r := computeMSDFWithPseudo(contours, x0, y0, gw, gh, distRange)
	return r.img
}

// computeMSDFOverlap handles overlapping contours by computing MSDF per
// contour independently and combining via union (max of signed distances).
// This is needed for composite glyphs where contours from separate
// components overlap (e.g. accented characters like e-acute).
func computeMSDFOverlap(contours []mContour, x0, y0, gw, gh int, distRange float64) msdfResult {
	if len(contours) <= 1 {
		return computeMSDFWithPseudo(contours, x0, y0, gw, gh, distRange)
	}

	pad := int(distRange) + 1
	w := gw + 2*pad
	h := gh + 2*pad

	// Compute per-contour MSDF and combine via union.
	// For union of SDFs: combined_dist = max(d1, d2, ...)
	// This means: if a pixel is inside ANY contour, it's inside.
	var combined msdfResult

	for i, c := range contours {
		single := []mContour{c}
		colorEdgesInkTrap(single)
		res := computeMSDFWithPseudo(single, x0, y0, gw, gh, distRange)

		if i == 0 {
			combined = res
			continue
		}

		// Union: take the max signed distance (most-inside) per channel.
		for py := 0; py < h; py++ {
			for px := 0; px < w; px++ {
				idx := py*w + px

				cOld := combined.img.NRGBAAt(px, py)
				cNew := res.img.NRGBAAt(px, py)

				// For each channel, pick the value that's more "inside"
				// (higher value = more inside in normalized distance).
				combined.img.SetNRGBA(px, py, color.NRGBA{
					R: maxU8(cOld.R, cNew.R),
					G: maxU8(cOld.G, cNew.G),
					B: maxU8(cOld.B, cNew.B),
					A: 255,
				})

				// Pseudo-SDF: take the more-inside value.
				if res.pseudoSD[idx] > combined.pseudoSD[idx] {
					combined.pseudoSD[idx] = res.pseudoSD[idx]
				}
			}
		}
	}

	return combined
}

func maxU8(a, b uint8) uint8 {
	if a > b {
		return a
	}
	return b
}

// contoursOverlap returns true if any two contour bounding boxes overlap.
// This detects composite glyphs where separate components occupy the same
// space and need per-contour MSDF computation.
func contoursOverlap(contours []mContour) bool {
	if len(contours) < 2 {
		return false
	}

	type bbox struct{ x0, y0, x1, y1 float64 }
	boxes := make([]bbox, len(contours))

	for i, c := range contours {
		bx0, by0 := math.MaxFloat64, math.MaxFloat64
		bx1, by1 := -math.MaxFloat64, -math.MaxFloat64
		for _, e := range c.edges {
			count := e.kind + 1
			for j := 0; j < count; j++ {
				if e.pts[j].x < bx0 {
					bx0 = e.pts[j].x
				}
				if e.pts[j].y < by0 {
					by0 = e.pts[j].y
				}
				if e.pts[j].x > bx1 {
					bx1 = e.pts[j].x
				}
				if e.pts[j].y > by1 {
					by1 = e.pts[j].y
				}
			}
		}
		boxes[i] = bbox{bx0, by0, bx1, by1}
	}

	// Check all pairs for overlap. Skip inner contours (counters) which are
	// fully contained within outer contours — these are normal (e.g. the
	// hole in "A"). Only flag true overlap where neither is fully contained.
	for i := 0; i < len(boxes); i++ {
		for j := i + 1; j < len(boxes); j++ {
			a, b := boxes[i], boxes[j]
			if a.x0 >= b.x1 || b.x0 >= a.x1 || a.y0 >= b.y1 || b.y0 >= a.y1 {
				continue // no overlap
			}
			// Bboxes overlap. Check if one fully contains the other
			// (inner counter, not a true overlap).
			aContainsB := a.x0 <= b.x0 && a.y0 <= b.y0 && a.x1 >= b.x1 && a.y1 >= b.y1
			bContainsA := b.x0 <= a.x0 && b.y0 <= a.y0 && b.x1 >= a.x1 && b.y1 >= a.y1
			if !aContainsB && !bContainsA {
				return true // true overlap
			}
		}
	}
	return false
}

func computeMSDFWithPseudo(contours []mContour, x0, y0, gw, gh int, distRange float64) msdfResult {
	pad := int(distRange) + 1
	w := gw + 2*pad
	h := gh + 2*pad
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	pseudoSD := make([]float64, w*h)
	if len(contours) == 0 {
		return msdfResult{img, pseudoSD, w, h}
	}

	// Precompute contour windings (shoelace formula).
	windings := make([]int, len(contours))
	for ci, c := range contours {
		windings[ci] = contourWinding(c)
	}

	inf := distRange * 1000
	for py := 0; py < h; py++ {
		wy := float64(py-pad+y0) + 0.5
		for px := 0; px < w; px++ {
			wx := float64(px-pad+x0) + 0.5
			p := mv2{wx, wy}

			// ── Per-contour distance computation (msdfgen architecture) ──
			// Each contour computes its own R, G, B distances independently.
			// Then the contour combiner merges them using winding logic.
			type contourDist struct {
				r, g, b float64 // per-channel distances
				pseudo  float64 // true distance to nearest edge in this contour
			}
			cDists := make([]contourDist, len(contours))

			for ci, c := range contours {
				minR, minG, minB := inf, inf, inf
				minTrue := inf

				for _, e := range c.edges {
					d := signedDistEdge(e, p)
					absD := math.Abs(d)

					if absD < math.Abs(minTrue) {
						minTrue = d
					}
					if e.ch&msdfChanR != 0 && absD < math.Abs(minR) {
						minR = d
					}
					if e.ch&msdfChanG != 0 && absD < math.Abs(minG) {
						minG = d
					}
					if e.ch&msdfChanB != 0 && absD < math.Abs(minB) {
						minB = d
					}
				}

				// Fallback: channels with no edges use the contour's true distance.
				if minR == inf {
					minR = minTrue
				}
				if minG == inf {
					minG = minTrue
				}
				if minB == inf {
					minB = minTrue
				}
				if minTrue == inf {
					minTrue = -inf
				}
				cDists[ci] = contourDist{minR, minG, minB, minTrue}
			}

			// ── Contour combiner (msdfgen OverlappingContourCombiner) ──
			// Combines per-contour distances using winding-number logic.
			// For each contour: if winding > 0 and distance >= 0, it's an
			// "inner" contour contributing positive (inside) distance.
			// If winding < 0 and distance <= 0, it's an "outer" contour.
			var shapeR, shapeG, shapeB float64 = -inf, -inf, -inf
			var innerR, innerG, innerB float64 = -inf, -inf, -inf
			var outerR, outerG, outerB float64 = -inf, -inf, -inf
			shapePseudo := -inf

			mergeChannel := func(dst *float64, src float64) {
				if math.Abs(src) < math.Abs(*dst) {
					*dst = src
				}
			}

			for ci, cd := range cDists {
				// Shape: merge all contours.
				mergeChannel(&shapeR, cd.r)
				mergeChannel(&shapeG, cd.g)
				mergeChannel(&shapeB, cd.b)
				mergeChannel(&shapePseudo, cd.pseudo)

				// Resolve this contour's scalar distance (median of R,G,B).
				contourScalar := median3Float(cd.r, cd.g, cd.b)

				// Inner: positive winding, positive distance (inside the contour).
				if windings[ci] > 0 && contourScalar >= 0 {
					mergeChannel(&innerR, cd.r)
					mergeChannel(&innerG, cd.g)
					mergeChannel(&innerB, cd.b)
				}
				// Outer: negative winding, negative distance (outside the contour).
				if windings[ci] < 0 && contourScalar <= 0 {
					mergeChannel(&outerR, cd.r)
					mergeChannel(&outerG, cd.g)
					mergeChannel(&outerB, cd.b)
				}
			}

			// Choose between inner, outer, and shape distances.
			finalR, finalG, finalB := shapeR, shapeG, shapeB
			if len(contours) > 1 {
				innerScalar := median3Float(innerR, innerG, innerB)
				outerScalar := median3Float(outerR, outerG, outerB)

				if innerScalar >= 0 && math.Abs(innerScalar) <= math.Abs(outerScalar) {
					finalR, finalG, finalB = innerR, innerG, innerB
				} else if outerScalar <= 0 && math.Abs(outerScalar) < math.Abs(innerScalar) {
					finalR, finalG, finalB = outerR, outerG, outerB
				}
			}

			img.SetNRGBA(px, py, color.NRGBA{
				R: distToUint8MSDF(finalR, distRange),
				G: distToUint8MSDF(finalG, distRange),
				B: distToUint8MSDF(finalB, distRange),
				A: 255,
			})

			// Pseudo-SDF for error correction.
			pNorm := 0.5 + 0.5*(shapePseudo/distRange)
			if pNorm < 0 {
				pNorm = 0
			}
			if pNorm > 1 {
				pNorm = 1
			}
			pseudoSD[py*w+px] = pNorm
		}
	}
	return msdfResult{img, pseudoSD, w, h}
}

// ── Atlas packing ────────────────────────────────────────────────────────────

type msdfGlyphBitmap struct {
	ID       rune
	Img      *image.NRGBA
	XOffset  int
	YOffset  int
	XAdvance int
	atlasX   int
	atlasY   int
}

func generateMSDFAtlas(glyphs []msdfGlyphBitmap, opts SDFGenOptions) (*image.NRGBA, []byte, error) {
	type pg struct {
		*msdfGlyphBitmap
		pw, ph int
	}
	packed := make([]pg, len(glyphs))
	for i := range glyphs {
		g := &glyphs[i]
		w, h := 0, 0
		if g.Img != nil {
			b := g.Img.Bounds()
			w, h = b.Dx(), b.Dy()
		}
		packed[i] = pg{&glyphs[i], w, h}
	}
	sort.Slice(packed, func(i, j int) bool { return packed[i].ph > packed[j].ph })

	atlasW := opts.AtlasWidth
	const glyphGap = 2 // pixels between glyphs to prevent bilinear bleed
	shelfY, shelfH, cursorX := 0, 0, 0
	for i := range packed {
		g := &packed[i]
		if cursorX+g.pw+glyphGap > atlasW {
			shelfY += shelfH + glyphGap
			shelfH, cursorX = 0, 0
		}
		g.atlasX = cursorX
		g.atlasY = shelfY
		cursorX += g.pw + glyphGap
		if g.ph > shelfH {
			shelfH = g.ph
		}
	}
	atlasH := shelfY + shelfH
	if atlasH == 0 {
		atlasH = 1
	}

	atlas := image.NewNRGBA(image.Rect(0, 0, atlasW, atlasH))

	type jsonGlyph struct {
		ID, X, Y, Width, Height, XOffset, YOffset, XAdvance int
	}
	jsonGlyphs := make([]jsonGlyph, 0, len(packed))
	for i := range packed {
		g := &packed[i]
		if g.Img != nil {
			b := g.Img.Bounds()
			for y := 0; y < b.Dy(); y++ {
				for x := 0; x < b.Dx(); x++ {
					ax, ay := g.atlasX+x, g.atlasY+y
					if ax < atlasW && ay < atlasH {
						atlas.SetNRGBA(ax, ay, g.Img.NRGBAAt(x, y))
					}
				}
			}
		}
		jsonGlyphs = append(jsonGlyphs, jsonGlyph{
			ID: int(g.ID), X: g.atlasX, Y: g.atlasY,
			Width: g.pw, Height: g.ph,
			XOffset: g.XOffset, YOffset: g.YOffset, XAdvance: g.XAdvance,
		})
	}

	metrics := sdfMetrics{
		Type:          "msdf",
		Size:          opts.Size,
		DistanceRange: opts.DistanceRange,
		LineHeight:    opts.Size * 1.2,
		Base:          opts.Size,
		Glyphs:        make([]sdfGlyphEntry, len(jsonGlyphs)),
	}
	for i, jg := range jsonGlyphs {
		metrics.Glyphs[i] = sdfGlyphEntry{
			ID: jg.ID, X: jg.X, Y: jg.Y,
			Width: jg.Width, Height: jg.Height,
			XOffset: jg.XOffset, YOffset: jg.YOffset, XAdvance: jg.XAdvance,
		}
	}
	metricsJSON, err := json.MarshalIndent(metrics, "", "  ")
	if err != nil {
		return nil, nil, err
	}
	return atlas, metricsJSON, nil
}

// ── Outline builder ──────────────────────────────────────────────────────────

func fixedPt(p fixed.Point26_6) mv2 {
	return mv2{float64(p.X) / 64.0, float64(p.Y) / 64.0}
}

func buildContours(segs sfnt.Segments) []mContour {
	var contours []mContour
	var current mContour
	var cursor mv2
	for _, seg := range segs {
		switch seg.Op {
		case sfnt.SegmentOpMoveTo:
			if len(current.edges) > 0 {
				contours = append(contours, current)
				current = mContour{}
			}
			cursor = fixedPt(seg.Args[0])
		case sfnt.SegmentOpLineTo:
			end := fixedPt(seg.Args[0])
			current.edges = append(current.edges, mEdge{pts: [4]mv2{cursor, end}, kind: 1})
			cursor = end
		case sfnt.SegmentOpQuadTo:
			ctrl, end := fixedPt(seg.Args[0]), fixedPt(seg.Args[1])
			current.edges = append(current.edges, mEdge{pts: [4]mv2{cursor, ctrl, end}, kind: 2})
			cursor = end
		case sfnt.SegmentOpCubeTo:
			c1, c2, end := fixedPt(seg.Args[0]), fixedPt(seg.Args[1]), fixedPt(seg.Args[2])
			current.edges = append(current.edges, mEdge{pts: [4]mv2{cursor, c1, c2, end}, kind: 3})
			cursor = end
		}
	}
	if len(current.edges) > 0 {
		contours = append(contours, current)
	}
	return preprocessContours(contours)
}

// preprocessContours removes degenerate edges (zero-length lines, coincident
// control points) and empty contours. This prevents numerical issues in
// distance computation and edge coloring.
func preprocessContours(contours []mContour) []mContour {
	result := contours[:0]
	for _, c := range contours {
		var clean []mEdge
		for _, e := range c.edges {
			if isEdgeZeroLength(e) {
				continue
			}
			clean = append(clean, e)
		}
		if len(clean) > 0 {
			result = append(result, mContour{edges: clean})
		}
	}
	return result
}

// isEdgeZeroLength returns true if the edge has effectively zero length.
func isEdgeZeroLength(e mEdge) bool {
	switch e.kind {
	case 1:
		d := e.pts[1].sub(e.pts[0])
		return d.dot(d) < degenerateEps*degenerateEps
	case 2:
		// Zero if start≈ctrl≈end.
		d1 := e.pts[1].sub(e.pts[0])
		d2 := e.pts[2].sub(e.pts[0])
		return d1.dot(d1) < degenerateEps*degenerateEps &&
			d2.dot(d2) < degenerateEps*degenerateEps
	case 3:
		d1 := e.pts[1].sub(e.pts[0])
		d2 := e.pts[2].sub(e.pts[0])
		d3 := e.pts[3].sub(e.pts[0])
		return d1.dot(d1) < degenerateEps*degenerateEps &&
			d2.dot(d2) < degenerateEps*degenerateEps &&
			d3.dot(d3) < degenerateEps*degenerateEps
	}
	return false
}

// ── Main entry point ─────────────────────────────────────────────────────────

// LoadDistanceFieldFontFromTTFMSDF generates an MSDF atlas from TTF/OTF data.
// It extracts glyph outlines, computes per-channel signed distances, and packs
// the results into an atlas. Distances are encoded in R, G, B channels; the
// MSDF shader reads median(R,G,B) to recover sharp edges and corners.
//
// EXPERIMENTAL — MSDF produces inconsistent rendering quality compared to SDF.
// Multi-contour glyphs (0, 8, @, etc.) exhibit anti-aliasing artifacts and
// channel divergence that degrades text appearance. The pipeline is
// architecturally complete (edge coloring, contour combiner, error correction)
// but the output quality does not match the reference msdfgen implementation.
// Use LoadDistanceFieldFontFromTTF (single-channel SDF) for production use.
func LoadDistanceFieldFontFromTTFMSDF(ttfData []byte, opts SDFGenOptions) (*distanceFieldFont, *ebiten.Image, *image.NRGBA, error) {
	opts.MSDF = true
	// MSDF benefits from a wider distance range than SDF. More range gives
	// more texel precision for channel transitions, reducing visible noise
	// at large display sizes. Godot uses 14; we default to 12.
	if opts.DistanceRange <= 0 || opts.DistanceRange == 8 {
		opts.DistanceRange = 12
	}
	opts.Defaults()

	// Parse font using sfnt to access raw glyph outlines.
	sFont, err := sfnt.Parse(ttfData)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("text: failed to parse TTF data: %w", err)
	}

	var buf sfnt.Buffer
	ppem := fixed.Int26_6(opts.Size * 64)

	// Extract font-level metrics.
	fm, err := sFont.Metrics(&buf, ppem, font.HintingNone)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("text: failed to read font metrics: %w", err)
	}
	ascent := fixedToFloat64(fm.Ascent)
	descent := fixedToFloat64(fm.Descent)
	lineHeight := fixedToFloat64(fm.Height)
	if lineHeight <= 0 {
		lineHeight = ascent + descent
	}

	// Generate MSDF for each character.
	var glyphs []msdfGlyphBitmap
	for i := 0; i < len(opts.Chars); {
		r, sz := utf8.DecodeRuneInString(opts.Chars[i:])
		i += sz

		gi, err := sFont.GlyphIndex(&buf, r)
		if err != nil || gi == 0 {
			continue
		}

		adv, err := sFont.GlyphAdvance(&buf, gi, ppem, font.HintingNone)
		if err != nil {
			continue
		}
		xAdvance := int(math.Round(fixedToFloat64(adv)))

		segs, err := sFont.LoadGlyph(&buf, gi, ppem, nil)
		if err != nil || len(segs) == 0 {
			// Whitespace or empty glyph — record advance only.
			glyphs = append(glyphs, msdfGlyphBitmap{
				ID:       r,
				Img:      nil,
				XAdvance: xAdvance,
			})
			continue
		}

		// Compute bounding box from outline segments.
		bx0, by0 := math.MaxFloat64, math.MaxFloat64
		bx1, by1 := -math.MaxFloat64, -math.MaxFloat64
		for _, seg := range segs {
			nArgs := 0
			switch seg.Op {
			case sfnt.SegmentOpMoveTo:
				nArgs = 1
			case sfnt.SegmentOpLineTo:
				nArgs = 1
			case sfnt.SegmentOpQuadTo:
				nArgs = 2
			case sfnt.SegmentOpCubeTo:
				nArgs = 3
			}
			for a := 0; a < nArgs; a++ {
				px := fixedToFloat64(seg.Args[a].X)
				py := fixedToFloat64(seg.Args[a].Y)
				if px < bx0 {
					bx0 = px
				}
				if py < by0 {
					by0 = py
				}
				if px > bx1 {
					bx1 = px
				}
				if py > by1 {
					by1 = py
				}
			}
		}

		x0 := int(math.Floor(bx0))
		y0 := int(math.Floor(by0))
		gw := int(math.Ceil(bx1)) - x0
		gh := int(math.Ceil(by1)) - y0

		if gw <= 0 || gh <= 0 {
			glyphs = append(glyphs, msdfGlyphBitmap{
				ID:       r,
				Img:      nil,
				XAdvance: xAdvance,
			})
			continue
		}

		contours := buildContours(segs)

		var res msdfResult
		if contoursOverlap(contours) {
			// Overlapping contours (composite glyphs): process per-contour
			// and combine via union. Edge coloring is done inside.
			res = computeMSDFOverlap(contours, x0, y0, gw, gh, opts.DistanceRange)
		} else {
			colorEdgesInkTrap(contours)
			res = computeMSDFWithPseudo(contours, x0, y0, gw, gh, opts.DistanceRange)
		}
		pad := int(opts.DistanceRange) + 1
		fixSignsByScanline(res.pseudoSD, res.w, res.h, x0, y0, pad, contours)
		// For cornerless glyphs (0, O, o, etc), set R=G=B=pseudo-SDF.
		// These glyphs have no sharp corners to preserve, so the MSDF
		// multi-channel signal only creates noise. Using the pseudo-SDF
		// (true closest-edge distance) makes them render identically to SDF.
		if !glyphHasCorners(contours, 3.0) {
			msdfEqualizeToPseudo(res.img, res.pseudoSD, res.w, res.h)
		}
		msdfErrorCorrectWithPseudo(res.img, res.pseudoSD, res.w, res.h)
		msdfErrorCorrectBilinear(res.img, res.pseudoSD, res.w, res.h)

		glyphs = append(glyphs, msdfGlyphBitmap{
			ID:       r,
			Img:      res.img,
			XOffset:  x0,
			YOffset:  y0 + int(math.Round(ascent)),
			XAdvance: xAdvance,
		})
	}

	// Pack into atlas and generate metrics JSON.
	atlas, metricsJSON, err := generateMSDFAtlas(glyphs, opts)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("text: MSDF atlas generation failed: %w", err)
	}

	// Override line metrics with actual font metrics.
	var m sdfMetrics
	_ = json.Unmarshal(metricsJSON, &m)
	m.LineHeight = lineHeight
	m.Base = ascent
	metricsJSON, _ = json.Marshal(m)

	sdfFont, err := loadDistanceFieldFont(metricsJSON, opts.PageIndex)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("text: failed to load generated MSDF font: %w", err)
	}

	atlasImg := ebiten.NewImage(atlas.Bounds().Dx(), atlas.Bounds().Dy())
	atlasImg.WritePixels(atlas.Pix)

	return sdfFont, atlasImg, atlas, nil
}
