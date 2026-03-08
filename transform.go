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
	sx := n.ScaleX_
	sy := n.ScaleY_

	// Fast path: no rotation and no skew (the common case for static sprites).
	// Avoids Sincos and Tan entirely  -  just scale + pivot + translate.
	if n.Rotation_ == 0 && n.SkewX_ == 0 && n.SkewY_ == 0 {
		return [6]float64{sx, 0, 0, sy, -n.PivotX_*sx + n.X_, -n.PivotY_*sy + n.Y_}
	}

	var sin, cos float64
	if n.Rotation_ != 0 {
		sin, cos = math.Sincos(n.Rotation_)
	} else {
		cos = 1
	}

	var tanSkewX, tanSkewY float64
	if n.SkewX_ != 0 {
		tanSkewX = math.Tan(n.SkewX_)
	}
	if n.SkewY_ != 0 {
		tanSkewY = math.Tan(n.SkewY_)
	}

	// After Scale * Translate(-pivot):
	//   a=sx, b=0, c=0, d=sy, tx=-px*sx, ty=-py*sy
	//
	// After Skew:
	a := sx
	b := tanSkewY * sx
	c := tanSkewX * sy
	d := sy

	px := n.PivotX_
	py := n.PivotY_
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
	return [6]float64{ra, rb, rc, rd, rtx + n.X_, rty + n.Y_}
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
	if !n.Visible_ {
		return
	}
	recompute := n.TransformDirty || parentRecomputed
	alphaChanged := n.AlphaDirty || parentAlphaChanged
	if recompute {
		local := computeLocalTransform(n)
		n.WorldTransform = multiplyAffine(parentTransform, local)
		n.WorldAlpha = parentAlpha * n.Alpha_
		n.TransformDirty = false
		n.AlphaDirty = false
	} else if alphaChanged {
		n.WorldAlpha = parentAlpha * n.Alpha_
		n.AlphaDirty = false
	}

	for _, child := range n.Children_ {
		updateWorldTransform(child, n.WorldTransform, n.WorldAlpha, recompute, recompute || alphaChanged)
	}
}

// --- Transform property setters ---

// SetPosition sets the node's local X and Y and marks it dirty.
func (n *Node) SetPosition(x, y float64) {
	n.X_ = x
	n.Y_ = y
	n.TransformDirty = true
	invalidateAncestorCache(n)
}

// SetX sets the node's local X and marks it dirty.
func (n *Node) SetX(x float64) {
	n.X_ = x
	n.TransformDirty = true
	invalidateAncestorCache(n)
}

// SetY sets the node's local Y and marks it dirty.
func (n *Node) SetY(y float64) {
	n.Y_ = y
	n.TransformDirty = true
	invalidateAncestorCache(n)
}

// SetScale sets the node's ScaleX and ScaleY and marks it dirty.
func (n *Node) SetScale(sx, sy float64) {
	n.ScaleX_ = sx
	n.ScaleY_ = sy
	n.TransformDirty = true
	invalidateAncestorCache(n)
}

// SetRotation sets the node's rotation (in radians) and marks it dirty.
func (n *Node) SetRotation(r float64) {
	n.Rotation_ = r
	n.TransformDirty = true
	invalidateAncestorCache(n)
}

// SetSkew sets the node's SkewX and SkewY (in radians) and marks it dirty.
func (n *Node) SetSkew(sx, sy float64) {
	n.SkewX_ = sx
	n.SkewY_ = sy
	n.TransformDirty = true
	invalidateAncestorCache(n)
}

// SetPivot sets the node's PivotX and PivotY and marks it dirty.
func (n *Node) SetPivot(px, py float64) {
	n.PivotX_ = px
	n.PivotY_ = py
	n.TransformDirty = true
	invalidateAncestorCache(n)
}

// SetAlpha sets the node's alpha and marks it alpha-dirty.
// Unlike other transform setters, this only triggers a worldAlpha recomputation
// (a single multiply), skipping the full matrix recompute.
func (n *Node) SetAlpha(a float64) {
	n.Alpha_ = a
	n.AlphaDirty = true
	invalidateAncestorCache(n)
}

// Invalidate marks the node's transform and alpha as dirty, forcing recomputation
// on the next frame. Also invalidates the TextBlock layout and SDF cache if present.
// Useful after bulk-setting fields directly.
func (n *Node) Invalidate() {
	n.TransformDirty = true
	n.AlphaDirty = true
	if n.TextBlock != nil {
		n.TextBlock.LayoutDirty = true
		n.TextBlock.UniformsDirty = true
	}
	invalidateAncestorCache(n)
}

// --- Coordinate conversion ---

// WorldToLocal converts a world-space point to this node's local coordinate space.
func (n *Node) WorldToLocal(wx, wy float64) (lx, ly float64) {
	inv := invertAffine(n.WorldTransform)
	return transformPoint(inv, wx, wy)
}

// LocalToWorld converts a local-space point to world-space.
func (n *Node) LocalToWorld(lx, ly float64) (wx, wy float64) {
	return transformPoint(n.WorldTransform, lx, ly)
}
