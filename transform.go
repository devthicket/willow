package willow

import "math"

// identityTransform is the identity affine matrix.
var identityTransform = [6]float64{1, 0, 0, 1, 0, 0}

// computeLocalTransform computes the local affine matrix from the node's
// transform properties. Returns [a, b, c, d, tx, ty].
//
// Composition order (spec 5.5):
//
//	Translate(-PivotX, -PivotY) -> Scale -> Skew -> Rotate -> Translate(X, Y)
func computeLocalTransform(n *Node) [6]float64 {
	sx := n.scaleX
	sy := n.scaleY

	// Fast path: no rotation and no skew (the common case for static sprites).
	// Avoids Sincos and Tan entirely  -  just scale + pivot + translate.
	if n.rotation == 0 && n.skewX == 0 && n.skewY == 0 {
		return [6]float64{sx, 0, 0, sy, -n.pivotX*sx + n.x, -n.pivotY*sy + n.y}
	}

	var sin, cos float64
	if n.rotation != 0 {
		sin, cos = math.Sincos(n.rotation)
	} else {
		cos = 1
	}

	var tanSkewX, tanSkewY float64
	if n.skewX != 0 {
		tanSkewX = math.Tan(n.skewX)
	}
	if n.skewY != 0 {
		tanSkewY = math.Tan(n.skewY)
	}

	// After Scale * Translate(-pivot):
	//   a=sx, b=0, c=0, d=sy, tx=-px*sx, ty=-py*sy
	//
	// After Skew:
	a := sx
	b := tanSkewY * sx
	c := tanSkewX * sy
	d := sy

	px := n.pivotX
	py := n.pivotY
	preTx := -px*sx - tanSkewX*py*sy
	preTy := -tanSkewY*px*sx - py*sy

	// After Rotate:
	ra := cos*a - sin*b
	rb := sin*a + cos*b
	rc := cos*c - sin*d
	rd := sin*c + cos*d
	rtx := cos*preTx - sin*preTy
	rty := sin*preTx + cos*preTy

	// After Translate(X, Y):
	return [6]float64{ra, rb, rc, rd, rtx + n.x, rty + n.y}
}

// multiplyAffine multiplies two 2D affine matrices: result = parent * child.
//
//	Matrix layout: [a, b, c, d, tx, ty]
//	| a  c  tx |
//	| b  d  ty |
//	| 0  0   1 |
func multiplyAffine(p, c [6]float64) [6]float64 {
	return [6]float64{
		p[0]*c[0] + p[2]*c[1],
		p[1]*c[0] + p[3]*c[1],
		p[0]*c[2] + p[2]*c[3],
		p[1]*c[2] + p[3]*c[3],
		p[0]*c[4] + p[2]*c[5] + p[4],
		p[1]*c[4] + p[3]*c[5] + p[5],
	}
}

// invertAffine computes the inverse of a 2D affine matrix.
// Returns the identity matrix if the matrix is singular (determinant ≈ 0).
func invertAffine(m [6]float64) [6]float64 {
	det := m[0]*m[3] - m[2]*m[1]
	if det > -1e-12 && det < 1e-12 {
		return identityTransform
	}
	invDet := 1.0 / det
	a := m[3] * invDet
	b := -m[1] * invDet
	c := -m[2] * invDet
	d := m[0] * invDet
	return [6]float64{
		a, b, c, d,
		-(a*m[4] + c*m[5]),
		-(b*m[4] + d*m[5]),
	}
}

// transformPoint applies an affine matrix to a point.
func transformPoint(m [6]float64, x, y float64) (float64, float64) {
	return m[0]*x + m[2]*y + m[4], m[1]*x + m[3]*y + m[5]
}

// updateWorldTransform recomputes a node's worldTransform and worldAlpha.
// parentRecomputed indicates whether the parent's transform was recomputed this frame.
// parentAlphaChanged indicates whether the parent's alpha changed (without a full transform recompute).
func updateWorldTransform(n *Node, parentTransform [6]float64, parentAlpha float64, parentRecomputed bool, parentAlphaChanged bool) {
	if !n.visible {
		return
	}
	recompute := n.transformDirty || parentRecomputed
	alphaChanged := n.alphaDirty || parentAlphaChanged
	if recompute {
		local := computeLocalTransform(n)
		n.worldTransform = multiplyAffine(parentTransform, local)
		n.worldAlpha = parentAlpha * n.alpha
		n.transformDirty = false
		n.alphaDirty = false
	} else if alphaChanged {
		n.worldAlpha = parentAlpha * n.alpha
		n.alphaDirty = false
	}

	for _, child := range n.children {
		updateWorldTransform(child, n.worldTransform, n.worldAlpha, recompute, recompute || alphaChanged)
	}
}

// --- Transform property setters ---

// SetPosition sets the node's local X and Y and marks it dirty.
func (n *Node) SetPosition(x, y float64) {
	n.x = x
	n.y = y
	n.transformDirty = true
	invalidateAncestorCache(n)
}

// SetX sets the node's local X and marks it dirty.
func (n *Node) SetX(x float64) {
	n.x = x
	n.transformDirty = true
	invalidateAncestorCache(n)
}

// SetY sets the node's local Y and marks it dirty.
func (n *Node) SetY(y float64) {
	n.y = y
	n.transformDirty = true
	invalidateAncestorCache(n)
}

// SetScale sets the node's ScaleX and ScaleY and marks it dirty.
func (n *Node) SetScale(sx, sy float64) {
	n.scaleX = sx
	n.scaleY = sy
	n.transformDirty = true
	invalidateAncestorCache(n)
}

// SetRotation sets the node's rotation (in radians) and marks it dirty.
func (n *Node) SetRotation(r float64) {
	n.rotation = r
	n.transformDirty = true
	invalidateAncestorCache(n)
}

// SetSkew sets the node's SkewX and SkewY (in radians) and marks it dirty.
func (n *Node) SetSkew(sx, sy float64) {
	n.skewX = sx
	n.skewY = sy
	n.transformDirty = true
	invalidateAncestorCache(n)
}

// SetPivot sets the node's PivotX and PivotY and marks it dirty.
func (n *Node) SetPivot(px, py float64) {
	n.pivotX = px
	n.pivotY = py
	n.transformDirty = true
	invalidateAncestorCache(n)
}

// SetAlpha sets the node's alpha and marks it alpha-dirty.
// Unlike other transform setters, this only triggers a worldAlpha recomputation
// (a single multiply), skipping the full matrix recompute.
func (n *Node) SetAlpha(a float64) {
	n.alpha = a
	n.alphaDirty = true
	invalidateAncestorCache(n)
}

// Invalidate marks the node's transform and alpha as dirty, forcing recomputation
// on the next frame. Also invalidates the TextBlock layout and SDF cache if present.
// Useful after bulk-setting fields directly.
func (n *Node) Invalidate() {
	n.transformDirty = true
	n.alphaDirty = true
	if n.TextBlock != nil {
		n.TextBlock.layoutDirty = true
		n.TextBlock.uniformsDirty = true
	}
	invalidateAncestorCache(n)
}

// --- Coordinate conversion ---

// WorldToLocal converts a world-space point to this node's local coordinate space.
func (n *Node) WorldToLocal(wx, wy float64) (lx, ly float64) {
	inv := invertAffine(n.worldTransform)
	return transformPoint(inv, wx, wy)
}

// LocalToWorld converts a local-space point to world-space.
func (n *Node) LocalToWorld(lx, ly float64) (wx, wy float64) {
	return transformPoint(n.worldTransform, lx, ly)
}
