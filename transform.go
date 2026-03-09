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
